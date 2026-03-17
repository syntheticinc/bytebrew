//go:build integration

package integration

import (
	"context"
	"database/sql"
	"net"
	"testing"
	"time"

	_ "github.com/glebarez/go-sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
	deliverygrpc "github.com/syntheticinc/bytebrew/bytebrew-srv/internal/delivery/grpc"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/flow_registry"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/llm"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/testutil"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/tools"
	agentservice "github.com/syntheticinc/bytebrew/bytebrew-srv/internal/service/agent"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/service/engine"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/service/eventstore"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/service/session_processor"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// StreamingHarness is a test harness for server-streaming API tests.
// Unlike ProductionHarness, it includes a SessionRegistry for server-streaming endpoints.
type StreamingHarness struct {
	grpcServer      *grpc.Server
	listener        net.Listener
	srvAddr         string
	sessionRegistry *flow_registry.SessionRegistry
	cancel          context.CancelFunc
	ctx             context.Context
}

// NewStreamingHarness creates a full in-process server with SessionRegistry for streaming API tests.
func NewStreamingHarness(t *testing.T, scenario string) *StreamingHarness {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())

	chatModel := llm.NewMockChatModel(scenario)
	snapshotRepo := testutil.NewMockSnapshotRepo()
	historyRepo := testutil.NewMockHistoryRepo()
	agentEngine := engine.New(snapshotRepo, historyRepo)

	flowsCfg, promptsCfg := testutil.TestFlowConfig()
	flowManager, err := agentservice.NewFlowManager(flowsCfg, promptsCfg)
	if err != nil {
		cancel()
		t.Fatalf("create flow manager: %v", err)
	}

	toolResolver := tools.NewDefaultToolResolver()
	agentConfig := &config.AgentConfig{
		MaxContextSize:     4000,
		MaxSteps:           10,
		ToolReturnDirectly: make(map[string]struct{}),
		Prompts:            promptsCfg,
	}

	subtaskMgr := testutil.NewMockSubtaskManager()
	taskMgr := testutil.NewMockTaskManager()

	modelSelector := llm.NewModelSelector(chatModel, "mock-model")
	agentRunStorage := testutil.NewMockAgentRunStorage()
	agentPool := agentservice.NewAgentPool(agentservice.AgentPoolConfig{
		ModelSelector:   modelSelector,
		SubtaskManager:  subtaskMgr,
		AgentRunStorage: agentRunStorage,
		AgentConfig:     agentConfig,
		MaxConcurrent:   0,
	})
	agentPoolAdapter := agentservice.NewAgentPoolAdapter(agentPool)

	toolDepsProvider := tools.NewDefaultToolDepsProvider(nil, taskMgr, subtaskMgr, agentPoolAdapter, nil, nil)
	agentPool.SetEngine(agentEngine, flowManager, toolResolver, toolDepsProvider)

	factory := infrastructure.NewEngineTurnExecutorFactory(
		agentEngine, flowManager, toolResolver, modelSelector, agentConfig,
		taskMgr, subtaskMgr, agentPoolAdapter, nil, nil, nil,
	)

	flowReg := flow_registry.NewInMemoryRegistry()
	// Create in-memory event store for tests
	eventsDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		cancel()
		t.Fatalf("open in-memory events db: %v", err)
	}
	eventsDB.SetMaxOpenConns(1)
	t.Cleanup(func() { eventsDB.Close() })

	evtStore, err := eventstore.New(eventsDB)
	if err != nil {
		cancel()
		t.Fatalf("create event store: %v", err)
	}

	sessionReg := flow_registry.NewSessionRegistry(evtStore)
	sessProcessor := session_processor.New(sessionReg, factory, evtStore)
	sessProcessor.SetAgentPoolRegistrar(agentPool)

	flowHandler, err := deliverygrpc.NewFlowHandlerWithConfig(deliverygrpc.FlowHandlerConfig{
		AgentService:        &testutil.NoopAgentService{},
		TurnExecutorFactory: factory,
		PingInterval:        60 * time.Second,
		FlowRegistry:        flowReg,
		AgentPoolProxy:      agentPool,
		AgentPoolAdapter:    agentPoolAdapter,
		SessionRegistry:     sessionReg,
		SessionProcessor:    sessProcessor,
	})
	if err != nil {
		cancel()
		t.Fatalf("create flow handler: %v", err)
	}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		cancel()
		t.Fatalf("listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterFlowServiceServer(grpcServer, flowHandler)

	go func() {
		_ = grpcServer.Serve(lis)
	}()

	return &StreamingHarness{
		grpcServer:      grpcServer,
		listener:        lis,
		srvAddr:         lis.Addr().String(),
		sessionRegistry: sessionReg,
		cancel:          cancel,
		ctx:             ctx,
	}
}

// DialClient returns a gRPC FlowServiceClient connected to the harness.
func (h *StreamingHarness) DialClient(t *testing.T) pb.FlowServiceClient {
	t.Helper()
	conn, err := grpc.NewClient(h.srvAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return pb.NewFlowServiceClient(conn)
}

// Cleanup shuts down the harness.
func (h *StreamingHarness) Cleanup() {
	h.cancel()
	h.grpcServer.GracefulStop()
}

// TC-G-03: Tool events in stream
// CreateSession → SendMessage → SubscribeSession → verify tool events arrive in stream
func TestStreamingAPI_ToolEventsInStream(t *testing.T) {
	harness := NewStreamingHarness(t, "local-read")
	defer harness.Cleanup()

	client := harness.DialClient(t)
	ctx := context.Background()

	// Create a temp file for read_file tool
	projectRoot := t.TempDir()
	writeFixture(t, projectRoot, "test.txt", "hello streaming world")

	// Create session with project root
	resp, err := client.CreateSession(ctx, &pb.CreateSessionRequest{
		ProjectKey: "test-project",
		UserId:     "test-user",
		Context: map[string]string{
			"project_root": projectRoot,
			"platform":     "linux",
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.SessionId)

	sessionID := resp.SessionId

	// Subscribe to session events (with timeout context so Recv doesn't block forever)
	recvCtx, recvCancel := context.WithTimeout(ctx, 15*time.Second)
	defer recvCancel()

	stream, err := client.SubscribeSession(recvCtx, &pb.SubscribeSessionRequest{
		SessionId: sessionID,
	})
	require.NoError(t, err)

	// Send a message (this triggers processMessage → agent turn)
	msgResp, err := client.SendMessage(ctx, &pb.SendMessageRequest{
		SessionId: sessionID,
		Content:   "Read the test file",
	})
	require.NoError(t, err)
	assert.True(t, msgResp.Accepted)

	// Collect events from stream
	var events []*pb.SessionEvent
	for {
		evt, recvErr := stream.Recv()
		if recvErr != nil {
			break
		}
		events = append(events, evt)

		// Stop after PROCESSING_STOPPED
		if evt.Type == pb.SessionEventType_SESSION_EVENT_PROCESSING_STOPPED {
			break
		}
	}
	require.NotEmpty(t, events, "should receive events from stream")

	// Verify we got PROCESSING_STARTED
	hasStarted := false
	hasStopped := false
	hasToolStart := false
	for _, evt := range events {
		switch evt.Type {
		case pb.SessionEventType_SESSION_EVENT_PROCESSING_STARTED:
			hasStarted = true
		case pb.SessionEventType_SESSION_EVENT_PROCESSING_STOPPED:
			hasStopped = true
		case pb.SessionEventType_SESSION_EVENT_TOOL_EXECUTION_START:
			hasToolStart = true
			assert.Equal(t, "read_file", evt.ToolName)
		}
	}

	assert.True(t, hasStarted, "should have PROCESSING_STARTED event")
	assert.True(t, hasStopped, "should have PROCESSING_STOPPED event")
	assert.True(t, hasToolStart, "should have TOOL_EXECUTION_START event for read_file")
}

// TC-G-09: Backward compatibility — old ExecuteFlow still works
// This is verified by existing supervisor_flow_test.go tests.
// Here we add a simple sanity check using ProductionHarness.
func TestStreamingAPI_BackwardCompat_ExecuteFlow(t *testing.T) {
	harness := NewProductionHarness(t, "echo")
	defer harness.Cleanup()

	sid, done := harness.CreateFlowAndWait(t, "Say hello")

	assert.NotEmpty(t, sid, "session ID should be assigned")

	select {
	case <-done:
		// Flow completed
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for ExecuteFlow to complete")
	}
}

// TC-G-Cancel: CancelSession via streaming API
func TestStreamingAPI_CancelSession(t *testing.T) {
	harness := NewStreamingHarness(t, "echo")
	defer harness.Cleanup()

	client := harness.DialClient(t)
	ctx := context.Background()

	// Create session
	resp, err := client.CreateSession(ctx, &pb.CreateSessionRequest{
		ProjectKey: "test-project",
		UserId:     "test-user",
	})
	require.NoError(t, err)

	sessionID := resp.SessionId

	// Cancel session
	cancelResp, err := client.CancelSession(ctx, &pb.CancelSessionRequest{
		SessionId: sessionID,
	})
	require.NoError(t, err)
	assert.True(t, cancelResp.Cancelled)

	// Verify cancelled state persists
	assert.True(t, harness.sessionRegistry.IsCancelled(sessionID))
}

// TC-G-SendMessage: SendMessage validates input
func TestStreamingAPI_SendMessage_Validation(t *testing.T) {
	harness := NewStreamingHarness(t, "echo")
	defer harness.Cleanup()

	client := harness.DialClient(t)
	ctx := context.Background()

	// Empty session_id
	resp, err := client.SendMessage(ctx, &pb.SendMessageRequest{
		SessionId: "",
		Content:   "hello",
	})
	require.NoError(t, err) // returns error in response, not gRPC error
	assert.NotEmpty(t, resp.Error)

	// Non-existent session
	resp, err = client.SendMessage(ctx, &pb.SendMessageRequest{
		SessionId: "nonexistent",
		Content:   "hello",
	})
	require.NoError(t, err)
	assert.Contains(t, resp.Error, "session not found")
}

// TC-G-AskUser: AskUser reply via SendMessage
func TestStreamingAPI_AskUserReply(t *testing.T) {
	harness := NewStreamingHarness(t, "echo")
	defer harness.Cleanup()

	client := harness.DialClient(t)
	ctx := context.Background()

	// Create session
	resp, err := client.CreateSession(ctx, &pb.CreateSessionRequest{
		ProjectKey: "test-project",
		UserId:     "test-user",
	})
	require.NoError(t, err)
	sessionID := resp.SessionId

	// Register an ask_user question directly in the registry (simulating agent side)
	replyCh := harness.sessionRegistry.RegisterAskUser(sessionID, "call-99")

	// Send reply via gRPC
	msgResp, err := client.SendMessage(ctx, &pb.SendMessageRequest{
		SessionId: sessionID,
		ReplyTo:   "call-99",
		Content:   "approved",
	})
	require.NoError(t, err)
	assert.True(t, msgResp.Accepted)

	// Verify reply arrives on the agent side
	select {
	case reply := <-replyCh:
		assert.Equal(t, "approved", reply)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ask_user reply")
	}
}

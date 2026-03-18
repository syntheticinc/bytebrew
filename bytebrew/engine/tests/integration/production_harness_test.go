//go:build integration

package integration

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	pb "github.com/syntheticinc/bytebrew/bytebrew/engine/api/proto/gen"
	deliverygrpc "github.com/syntheticinc/bytebrew/bytebrew/engine/internal/delivery/grpc"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/flow_registry"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/llm"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/testutil"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/tools"
	agentservice "github.com/syntheticinc/bytebrew/bytebrew/engine/internal/service/agent"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/service/engine"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/pkg/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ProductionHarness spins up an in-process srv with MockChatModel
// and provides helpers for integration tests.
type ProductionHarness struct {
	grpcServer   *grpc.Server
	listener     net.Listener
	srvAddr      string
	flowRegistry *flow_registry.InMemoryRegistry
	cancel       context.CancelFunc
	ctx          context.Context
}

// NewProductionHarness creates a full in-process srv with the given MockChatModel scenario.
func NewProductionHarness(t *testing.T, scenario string) *ProductionHarness {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())

	// 1. MockChatModel + in-memory repos
	chatModel := llm.NewMockChatModel(scenario)
	snapshotRepo := testutil.NewMockSnapshotRepo()
	historyRepo := testutil.NewMockHistoryRepo()
	agentEngine := engine.New(snapshotRepo, historyRepo)

	// 2. Flow config
	flowsCfg, promptsCfg := testutil.TestFlowConfig()
	flowManager, err := agentservice.NewFlowManager(flowsCfg, promptsCfg)
	if err != nil {
		cancel()
		t.Fatalf("create flow manager: %v", err)
	}

	// 3. Tool resolver + Agent config
	builtinStore := tools.NewBuiltinToolStore()
	tools.RegisterAllBuiltins(builtinStore)
	toolResolver := tools.NewAgentToolResolver(builtinStore)
	agentConfig := &config.AgentConfig{
		MaxContextSize:     4000,
		MaxSteps:           10,
		ToolReturnDirectly: make(map[string]struct{}),
		Prompts:            promptsCfg,
	}

	// 4. Mock managers
	subtaskMgr := testutil.NewMockSubtaskManager()
	taskMgr := testutil.NewMockTaskManager()

	// 5. Model selector + agent pool
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

	// 6. TurnExecutorFactory
	factory := infrastructure.NewEngineTurnExecutorFactory(
		agentEngine, flowManager, toolResolver, modelSelector, agentConfig,
		taskMgr, subtaskMgr, agentPoolAdapter, nil, nil, nil,
	)

	// 7. FlowHandler + FlowRegistry
	flowReg := flow_registry.NewInMemoryRegistry()
	flowHandler, err := deliverygrpc.NewFlowHandlerWithConfig(deliverygrpc.FlowHandlerConfig{
		AgentService:        &testutil.NoopAgentService{},
		TurnExecutorFactory: factory,
		PingInterval:        60 * time.Second,
		FlowRegistry:        flowReg,
		AgentPoolProxy:      agentPool,
		AgentPoolAdapter:    agentPoolAdapter,
	})
	if err != nil {
		cancel()
		t.Fatalf("create flow handler: %v", err)
	}

	// 8. gRPC server on random port
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

	srvAddr := lis.Addr().String()
	t.Logf("srv listening on %s", srvAddr)

	return &ProductionHarness{
		grpcServer:   grpcServer,
		listener:     lis,
		srvAddr:      srvAddr,
		flowRegistry: flowReg,
		cancel:       cancel,
		ctx:          ctx,
	}
}

// DialSrv returns a gRPC connection to the in-process srv.
func (h *ProductionHarness) DialSrv(t *testing.T) *grpc.ClientConn {
	t.Helper()

	conn, err := grpc.NewClient(h.srvAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial srv: %v", err)
	}
	return conn
}

// Cleanup shuts down the harness.
func (h *ProductionHarness) Cleanup() {
	h.cancel()
	h.grpcServer.GracefulStop()
}

// SrvAddr returns "host:port" of the srv.
func (h *ProductionHarness) SrvAddr() string { return h.srvAddr }

// FlowRegistry returns the flow registry.
func (h *ProductionHarness) FlowRegistry() *flow_registry.InMemoryRegistry { return h.flowRegistry }

// Context returns the harness context.
func (h *ProductionHarness) Context() context.Context { return h.ctx }

// CreateFlowAndWait starts an ExecuteFlow, sends a question, returns sessionID and done channel.
func (h *ProductionHarness) CreateFlowAndWait(t *testing.T, prompt string) (sessionID string, done <-chan struct{}) {
	t.Helper()

	conn := h.DialSrv(t)
	client := pb.NewFlowServiceClient(conn)

	sid := fmt.Sprintf("e2e-%s-%d", t.Name(), time.Now().UnixNano())

	stream, err := client.ExecuteFlow(context.Background())
	if err != nil {
		conn.Close()
		t.Fatalf("open ExecuteFlow stream: %v", err)
	}

	err = stream.Send(&pb.FlowRequest{
		SessionId:  sid,
		Task:       prompt,
		ProjectKey: "e2e-test",
		UserId:     "e2e-test-user",
	})
	if err != nil {
		conn.Close()
		t.Fatalf("send flow request: %v", err)
	}

	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		defer conn.Close()
		for {
			_, err := stream.Recv()
			if err != nil {
				return
			}
		}
	}()

	// Wait briefly for the flow to register
	time.Sleep(200 * time.Millisecond)

	return sid, doneCh
}

// UniqueSessionID generates a unique session ID for tests.
func UniqueSessionID(t *testing.T) string {
	return fmt.Sprintf("e2e-%s-%s", t.Name(), uuid.New().String()[:8])
}

// Ensure ProductionHarness fields are used to prevent compiler errors.
var _ = (*ProductionHarness)(nil)
var _ = (*domain.ActiveFlow)(nil)

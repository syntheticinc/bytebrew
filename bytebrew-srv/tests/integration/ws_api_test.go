//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/delivery/ws"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/flow_registry"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/llm"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/testutil"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/tools"
	agentservice "github.com/syntheticinc/bytebrew/bytebrew-srv/internal/service/agent"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/service/engine"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/service/session_processor"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/config"
)

// WsHarness is a test harness for WebSocket API tests.
// It spins up a WS server with the same agent stack as StreamingHarness.
type WsHarness struct {
	wsServer        *ws.Server
	wsURL           string
	sessionRegistry *flow_registry.SessionRegistry
	cancel          context.CancelFunc
	ctx             context.Context
}

// NewWsHarness creates a full in-process WS server for integration tests.
func NewWsHarness(t *testing.T, scenario string) *WsHarness {
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

	sessionReg := flow_registry.NewSessionRegistry()

	sessProcessor := session_processor.New(sessionReg, factory)

	wsHandler := ws.NewConnectionHandler(sessionReg, sessProcessor, &testutil.NoopAgentService{})

	wsServer, err := ws.NewServer(wsHandler)
	if err != nil {
		cancel()
		t.Fatalf("create ws server: %v", err)
	}

	go func() {
		_ = wsServer.Start(ctx)
	}()

	// Give the HTTP server a moment to start accepting connections.
	time.Sleep(50 * time.Millisecond)

	wsURL := fmt.Sprintf("ws://127.0.0.1:%d/ws", wsServer.Port())

	return &WsHarness{
		wsServer:        wsServer,
		wsURL:           wsURL,
		sessionRegistry: sessionReg,
		cancel:          cancel,
		ctx:             ctx,
	}
}

// DialWS opens a WebSocket connection to the harness.
func (h *WsHarness) DialWS(t *testing.T) *websocket.Conn {
	t.Helper()

	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}
	conn, resp, err := dialer.Dial(h.wsURL, nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("unexpected ws status: %d", resp.StatusCode)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

// Cleanup shuts down the harness.
func (h *WsHarness) Cleanup() {
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = h.wsServer.Shutdown(shutdownCtx)
	h.cancel()
}

// sendJSON sends a WsMessage as JSON over the WebSocket connection.
func sendJSON(t *testing.T, conn *websocket.Conn, msg ws.WsMessage) {
	t.Helper()
	data, err := json.Marshal(msg)
	require.NoError(t, err)
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, data))
}

// recvJSON reads a WsMessage from the WebSocket connection with a timeout.
func recvJSON(t *testing.T, conn *websocket.Conn, timeout time.Duration) ws.WsMessage {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(timeout))
	_, data, err := conn.ReadMessage()
	require.NoError(t, err, "ws read should not error")
	var msg ws.WsMessage
	require.NoError(t, json.Unmarshal(data, &msg))
	return msg
}

// createSessionViaWS sends a create_session message and returns the session ID.
func createSessionViaWS(t *testing.T, conn *websocket.Conn, projectRoot string) string {
	t.Helper()

	payload := map[string]interface{}{
		"project_key":  "test-project",
		"project_root": projectRoot,
		"platform":     "linux",
	}

	sendJSON(t, conn, ws.WsMessage{
		Type:      "create_session",
		RequestID: "cs-1",
		Payload:   payload,
	})

	resp := recvJSON(t, conn, 5*time.Second)
	require.Equal(t, "create_session_ack", resp.Type)
	require.Equal(t, "cs-1", resp.RequestID)

	sessionID, ok := resp.Payload["session_id"].(string)
	require.True(t, ok, "session_id should be a string")
	require.NotEmpty(t, sessionID)

	return sessionID
}

// TC-WS-01: Ping/pong
func TestWsAPI_PingPong(t *testing.T) {
	harness := NewWsHarness(t, "echo")
	defer harness.Cleanup()

	conn := harness.DialWS(t)

	sendJSON(t, conn, ws.WsMessage{
		Type:      "ping",
		RequestID: "r1",
	})

	resp := recvJSON(t, conn, 5*time.Second)
	assert.Equal(t, "pong", resp.Type)
	assert.Equal(t, "r1", resp.RequestID)
	assert.NotNil(t, resp.Payload["timestamp"], "pong should contain timestamp")
}

// TC-WS-02: Full message flow — create_session → subscribe → send_message → collect events
func TestWsAPI_FullMessageFlow(t *testing.T) {
	projectRoot := t.TempDir()
	writeFixture(t, projectRoot, "test.txt", "hello ws world")

	harness := NewWsHarness(t, "local-read")
	defer harness.Cleanup()

	conn := harness.DialWS(t)

	// Create session
	sessionID := createSessionViaWS(t, conn, projectRoot)

	// Subscribe
	sendJSON(t, conn, ws.WsMessage{
		Type:      "subscribe",
		RequestID: "sub-1",
		Payload:   map[string]interface{}{"session_id": sessionID},
	})

	subAck := recvJSON(t, conn, 5*time.Second)
	require.Equal(t, "subscribe_ack", subAck.Type)

	// Send message
	sendJSON(t, conn, ws.WsMessage{
		Type:      "send_message",
		RequestID: "msg-1",
		Payload: map[string]interface{}{
			"session_id": sessionID,
			"content":    "Read the test file",
		},
	})

	msgAck := recvJSON(t, conn, 5*time.Second)
	require.Equal(t, "send_message_ack", msgAck.Type)

	// Collect session events until ProcessingStopped or timeout
	var events []ws.WsMessage
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		conn.SetReadDeadline(time.Now().Add(15 * time.Second))
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var msg ws.WsMessage
		if jsonErr := json.Unmarshal(data, &msg); jsonErr != nil {
			continue
		}
		if msg.Type != "session_event" {
			continue
		}
		events = append(events, msg)

		// Check if this is ProcessingStopped
		eventPayload, ok := msg.Payload["event"].(map[string]interface{})
		if ok {
			if eventType, _ := eventPayload["type"].(string); eventType == "ProcessingStopped" {
				break
			}
		}
	}

	require.NotEmpty(t, events, "should receive session events")

	// Verify expected event types
	hasProcessingStarted := false
	hasProcessingStopped := false
	hasToolStart := false
	for _, evt := range events {
		eventPayload, ok := evt.Payload["event"].(map[string]interface{})
		if !ok {
			continue
		}
		eventType, _ := eventPayload["type"].(string)
		switch eventType {
		case "ProcessingStarted":
			hasProcessingStarted = true
		case "ProcessingStopped":
			hasProcessingStopped = true
		case "ToolExecutionStarted":
			hasToolStart = true
			assert.Equal(t, "read_file", eventPayload["tool_name"])
		}
	}

	assert.True(t, hasProcessingStarted, "should have ProcessingStarted event")
	assert.True(t, hasProcessingStopped, "should have ProcessingStopped event")
	assert.True(t, hasToolStart, "should have ToolExecutionStarted event for read_file")
}

// TC-WS-03: Cancel session
func TestWsAPI_CancelSession(t *testing.T) {
	harness := NewWsHarness(t, "echo")
	defer harness.Cleanup()

	conn := harness.DialWS(t)

	sessionID := createSessionViaWS(t, conn, t.TempDir())

	// Cancel session
	sendJSON(t, conn, ws.WsMessage{
		Type:      "cancel_session",
		RequestID: "cancel-1",
		Payload:   map[string]interface{}{"session_id": sessionID},
	})

	resp := recvJSON(t, conn, 5*time.Second)
	assert.Equal(t, "cancel_session_ack", resp.Type)
	assert.Equal(t, "cancel-1", resp.RequestID)
	assert.Equal(t, true, resp.Payload["cancelled"])

	// Verify cancelled state in registry
	assert.True(t, harness.sessionRegistry.IsCancelled(sessionID))
}

// TC-WS-04: AskUser reply
func TestWsAPI_AskUserReply(t *testing.T) {
	harness := NewWsHarness(t, "echo")
	defer harness.Cleanup()

	conn := harness.DialWS(t)

	sessionID := createSessionViaWS(t, conn, t.TempDir())

	// Register an ask_user question directly in the registry (simulating agent side)
	replyCh := harness.sessionRegistry.RegisterAskUser(sessionID, "call-42")

	// Send reply via WS
	sendJSON(t, conn, ws.WsMessage{
		Type:      "ask_user_reply",
		RequestID: "ask-1",
		Payload: map[string]interface{}{
			"session_id": sessionID,
			"call_id":    "call-42",
			"reply":      "approved",
		},
	})

	ackResp := recvJSON(t, conn, 5*time.Second)
	assert.Equal(t, "ask_user_reply_ack", ackResp.Type)

	// Verify reply arrives on the agent side
	select {
	case reply := <-replyCh:
		assert.Equal(t, "approved", reply)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for ask_user reply")
	}
}

// TC-WS-05: Subscribe with last_event_id (backfill)
func TestWsAPI_SubscribeWithBackfill(t *testing.T) {
	t.Skip("requires event store implementation with stable event IDs for reliable backfill testing")
}

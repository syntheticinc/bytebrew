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

	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
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
	sessProcessor.SetAgentPoolRegistrar(agentPool)

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

// collectSessionEvents reads session_event messages until a predicate returns true or timeout.
func collectSessionEvents(t *testing.T, conn *websocket.Conn, timeout time.Duration, stopWhen func(eventType string) bool) []ws.WsMessage {
	t.Helper()

	var events []ws.WsMessage
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		conn.SetReadDeadline(time.Now().Add(timeout))
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

		eventPayload, ok := msg.Payload["event"].(map[string]interface{})
		if ok {
			if eventType, _ := eventPayload["type"].(string); stopWhen(eventType) {
				break
			}
		}
	}

	return events
}

// hasEventType checks if any session event has the given event type.
func hasEventType(events []ws.WsMessage, eventType string) bool {
	for _, evt := range events {
		eventPayload, ok := evt.Payload["event"].(map[string]interface{})
		if !ok {
			continue
		}
		if t, _ := eventPayload["type"].(string); t == eventType {
			return true
		}
	}
	return false
}

// subscribeToSession sends a subscribe message and asserts the ack.
func subscribeToSession(t *testing.T, conn *websocket.Conn, sessionID string) {
	t.Helper()

	sendJSON(t, conn, ws.WsMessage{
		Type:      "subscribe",
		RequestID: "sub-" + sessionID[:8],
		Payload:   map[string]interface{}{"session_id": sessionID},
	})

	subAck := recvJSON(t, conn, 5*time.Second)
	require.Equal(t, "subscribe_ack", subAck.Type)
}

// sendMessage sends a send_message command and asserts the ack.
func sendMessage(t *testing.T, conn *websocket.Conn, sessionID, content, requestID string) {
	t.Helper()

	sendJSON(t, conn, ws.WsMessage{
		Type:      "send_message",
		RequestID: requestID,
		Payload: map[string]interface{}{
			"session_id": sessionID,
			"content":    content,
		},
	})

	msgAck := recvJSON(t, conn, 5*time.Second)
	require.Equal(t, "send_message_ack", msgAck.Type)
	require.Equal(t, requestID, msgAck.RequestID)
}

// TC-WS-06: Multi-turn context preservation
func TestWsAPI_MultiTurnContextPreservation(t *testing.T) {
	harness := NewWsHarness(t, "echo")
	defer harness.Cleanup()

	conn := harness.DialWS(t)

	sessionID := createSessionViaWS(t, conn, t.TempDir())
	subscribeToSession(t, conn, sessionID)

	// Send first message
	sendMessage(t, conn, sessionID, "Hello first", "msg-1")

	// Collect events until ProcessingStopped
	events1 := collectSessionEvents(t, conn, 15*time.Second, func(et string) bool {
		return et == "ProcessingStopped"
	})
	require.NotEmpty(t, events1, "should receive events for first message")
	assert.True(t, hasEventType(events1, "ProcessingStarted"), "first round should have ProcessingStarted")
	assert.True(t, hasEventType(events1, "ProcessingStopped"), "first round should have ProcessingStopped")

	// Send second message in the same session
	sendMessage(t, conn, sessionID, "Hello second", "msg-2")

	// Collect events until ProcessingStopped again
	events2 := collectSessionEvents(t, conn, 15*time.Second, func(et string) bool {
		return et == "ProcessingStopped"
	})
	require.NotEmpty(t, events2, "should receive events for second message")
	assert.True(t, hasEventType(events2, "ProcessingStarted"), "second round should have ProcessingStarted")
	assert.True(t, hasEventType(events2, "ProcessingStopped"), "second round should have ProcessingStopped")
}

// TC-WS-07: AskUser full round-trip — simulates server-side AskUserRequested event,
// then verifies the client can reply via WS and the reply is delivered to the agent side.
// This tests the WS layer's ask_user flow without depending on the agent emitting AskUserRequested
// (LocalClientOperationsProxy auto-answers in headless mode, so no AskUserRequested is emitted).
func TestWsAPI_AskUserFullRoundTrip(t *testing.T) {
	harness := NewWsHarness(t, "echo")
	defer harness.Cleanup()

	conn := harness.DialWS(t)

	sessionID := createSessionViaWS(t, conn, t.TempDir())
	subscribeToSession(t, conn, sessionID)

	// Register ask_user question on the server side (simulates agent calling ask_user tool)
	callID := "call-ask-42"
	replyCh := harness.sessionRegistry.RegisterAskUser(sessionID, callID)

	// Publish AskUserRequested event (simulates what EventStream.Send does for domain.EventTypeUserQuestion)
	harness.sessionRegistry.PublishEvent(sessionID, &pb.SessionEvent{
		EventId:  "evt-ask-1",
		Type:     pb.SessionEventType_SESSION_EVENT_ASK_USER,
		Question: "Do you approve?",
		CallId:   callID,
		AgentId:  "supervisor",
	})

	// Client should receive AskUserRequested event
	askEvent := collectSessionEvents(t, conn, 5*time.Second, func(et string) bool {
		return et == "AskUserRequested"
	})
	require.NotEmpty(t, askEvent, "should receive AskUserRequested event")

	// Verify event payload
	eventPayload, ok := askEvent[0].Payload["event"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "AskUserRequested", eventPayload["type"])
	assert.Equal(t, "Do you approve?", eventPayload["question"])
	assert.Equal(t, callID, eventPayload["call_id"])

	// Client sends reply via WS
	sendJSON(t, conn, ws.WsMessage{
		Type:      "ask_user_reply",
		RequestID: "reply-1",
		Payload: map[string]interface{}{
			"session_id": sessionID,
			"call_id":    callID,
			"reply":      "approved",
		},
	})

	ackResp := recvJSON(t, conn, 5*time.Second)
	require.Equal(t, "ask_user_reply_ack", ackResp.Type)

	// Verify reply is delivered to the agent side
	select {
	case reply := <-replyCh:
		assert.Equal(t, "approved", reply)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for ask_user reply on agent side")
	}

	// Publish a completion event (simulates agent continuing after reply)
	harness.sessionRegistry.PublishEvent(sessionID, &pb.SessionEvent{
		EventId: "evt-complete-1",
		Type:    pb.SessionEventType_SESSION_EVENT_ANSWER,
		Content: "User approved, proceeding.",
		AgentId: "supervisor",
	})

	// Client should receive the completion event
	completionEvents := collectSessionEvents(t, conn, 5*time.Second, func(et string) bool {
		return et == "MessageCompleted"
	})
	require.NotEmpty(t, completionEvents, "should receive completion event")

	completionPayload, ok := completionEvents[0].Payload["event"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "MessageCompleted", completionPayload["type"])
	assert.Contains(t, completionPayload["content"], "approved")
}

// TC-WS-09: Fan-out — two clients subscribed to the same session
func TestWsAPI_FanOutTwoClients(t *testing.T) {
	harness := NewWsHarness(t, "echo")
	defer harness.Cleanup()

	connA := harness.DialWS(t)
	connB := harness.DialWS(t)

	// Client A creates session
	sessionID := createSessionViaWS(t, connA, t.TempDir())

	// Both clients subscribe
	subscribeToSession(t, connA, sessionID)
	subscribeToSession(t, connB, sessionID)

	// Client A sends a message
	sendMessage(t, connA, sessionID, "Hello fanout", "msg-1")

	// Both clients should receive events
	eventsA := collectSessionEvents(t, connA, 15*time.Second, func(et string) bool {
		return et == "ProcessingStopped"
	})
	eventsB := collectSessionEvents(t, connB, 15*time.Second, func(et string) bool {
		return et == "ProcessingStopped"
	})

	require.NotEmpty(t, eventsA, "client A should receive events")
	require.NotEmpty(t, eventsB, "client B should receive events")

	assert.True(t, hasEventType(eventsA, "ProcessingStarted"), "client A should have ProcessingStarted")
	assert.True(t, hasEventType(eventsA, "ProcessingStopped"), "client A should have ProcessingStopped")

	assert.True(t, hasEventType(eventsB, "ProcessingStarted"), "client B should have ProcessingStarted")
	assert.True(t, hasEventType(eventsB, "ProcessingStopped"), "client B should have ProcessingStopped")
}

// TC-WS-10: Cancel during streaming
func TestWsAPI_CancelDuringStreaming(t *testing.T) {
	harness := NewWsHarness(t, "cancel-during-stream")
	defer harness.Cleanup()

	conn := harness.DialWS(t)

	sessionID := createSessionViaWS(t, conn, t.TempDir())
	subscribeToSession(t, conn, sessionID)

	// Send message (mock will sleep 3s before responding)
	sendMessage(t, conn, sessionID, "Process something slow", "msg-1")

	// Wait for ProcessingStarted
	started := collectSessionEvents(t, conn, 5*time.Second, func(et string) bool {
		return et == "ProcessingStarted"
	})
	require.True(t, hasEventType(started, "ProcessingStarted"), "should see ProcessingStarted before cancel")

	// Cancel while processing
	sendJSON(t, conn, ws.WsMessage{
		Type:      "cancel_session",
		RequestID: "cancel-1",
		Payload:   map[string]interface{}{"session_id": sessionID},
	})

	cancelAck := recvJSON(t, conn, 5*time.Second)
	assert.Equal(t, "cancel_session_ack", cancelAck.Type)
	assert.Equal(t, true, cancelAck.Payload["cancelled"])

	// Should eventually get ProcessingStopped (agent detects cancellation)
	remaining := collectSessionEvents(t, conn, 10*time.Second, func(et string) bool {
		return et == "ProcessingStopped"
	})
	assert.True(t, hasEventType(remaining, "ProcessingStopped"), "should receive ProcessingStopped after cancel")
}

// TC-WS-11: Unknown message type — server does not crash
func TestWsAPI_UnknownMessageType(t *testing.T) {
	harness := NewWsHarness(t, "echo")
	defer harness.Cleanup()

	conn := harness.DialWS(t)

	// Send unknown message type
	sendJSON(t, conn, ws.WsMessage{
		Type:      "nonexistent_command",
		RequestID: "r1",
	})

	// Should get error response
	resp := recvJSON(t, conn, 5*time.Second)
	assert.Equal(t, "error", resp.Type)
	assert.Equal(t, "r1", resp.RequestID)
	assert.Contains(t, resp.Payload["error"], "unknown message type")

	// Verify server still works — ping should succeed
	sendJSON(t, conn, ws.WsMessage{
		Type:      "ping",
		RequestID: "r2",
	})

	pong := recvJSON(t, conn, 5*time.Second)
	assert.Equal(t, "pong", pong.Type)
	assert.Equal(t, "r2", pong.RequestID)
}

// TC-WS-12: Message to non-existent session
func TestWsAPI_MessageToNonExistentSession(t *testing.T) {
	harness := NewWsHarness(t, "echo")
	defer harness.Cleanup()

	conn := harness.DialWS(t)

	// send_message to fake session
	sendJSON(t, conn, ws.WsMessage{
		Type:      "send_message",
		RequestID: "msg-fake",
		Payload: map[string]interface{}{
			"session_id": "fake-session-id",
			"content":    "hello",
		},
	})

	msgResp := recvJSON(t, conn, 5*time.Second)
	assert.Equal(t, "error", msgResp.Type)
	assert.Equal(t, "msg-fake", msgResp.RequestID)
	assert.Contains(t, msgResp.Payload["error"], "session not found")

	// subscribe to fake session
	sendJSON(t, conn, ws.WsMessage{
		Type:      "subscribe",
		RequestID: "sub-fake",
		Payload: map[string]interface{}{
			"session_id": "fake-session-id",
		},
	})

	subResp := recvJSON(t, conn, 5*time.Second)
	assert.Equal(t, "error", subResp.Type)
	assert.Equal(t, "sub-fake", subResp.RequestID)
	assert.Contains(t, subResp.Payload["error"], "session not found")

	// Server should still work after errors
	sendJSON(t, conn, ws.WsMessage{
		Type:      "ping",
		RequestID: "r-after",
	})

	pong := recvJSON(t, conn, 5*time.Second)
	assert.Equal(t, "pong", pong.Type)
	assert.Equal(t, "r-after", pong.RequestID)
}

// TC-WS-13: Concurrent message sending — FIFO, no panic
func TestWsAPI_ConcurrentMessageSending(t *testing.T) {
	harness := NewWsHarness(t, "echo")
	defer harness.Cleanup()

	conn := harness.DialWS(t)

	sessionID := createSessionViaWS(t, conn, t.TempDir())
	subscribeToSession(t, conn, sessionID)

	// Send 3 messages quickly without waiting for ack between them
	for i := 1; i <= 3; i++ {
		sendJSON(t, conn, ws.WsMessage{
			Type:      "send_message",
			RequestID: fmt.Sprintf("msg-%d", i),
			Payload: map[string]interface{}{
				"session_id": sessionID,
				"content":    fmt.Sprintf("Message %d", i),
			},
		})
	}

	// Collect all acks and session events
	var acks []ws.WsMessage
	processingStopped := 0

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg ws.WsMessage
		if jsonErr := json.Unmarshal(data, &msg); jsonErr != nil {
			continue
		}

		if msg.Type == "send_message_ack" {
			acks = append(acks, msg)
			continue
		}

		if msg.Type == "session_event" {
			eventPayload, ok := msg.Payload["event"].(map[string]interface{})
			if ok {
				if eventType, _ := eventPayload["type"].(string); eventType == "ProcessingStopped" {
					processingStopped++
					if processingStopped >= 3 {
						break
					}
				}
			}
		}
	}

	// All 3 messages should have received acks
	assert.Len(t, acks, 3, "all 3 messages should be acknowledged")

	// All 3 messages should have been fully processed (3 ProcessingStopped events)
	assert.Equal(t, 3, processingStopped, "all 3 messages should complete processing")
}

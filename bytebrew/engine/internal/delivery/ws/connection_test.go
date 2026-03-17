package ws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pb "github.com/syntheticinc/bytebrew/bytebrew/engine/api/proto/gen"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	sp "github.com/syntheticinc/bytebrew/bytebrew/engine/internal/service/session_processor"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/flow_registry"
)

// --- mocks ---

type mockAgentEnvSetter struct {
	projectRoot string
	platform    string
}

func (m *mockAgentEnvSetter) SetEnvironmentContext(projectRoot, platform string) {
	m.projectRoot = projectRoot
	m.platform = platform
}

type mockPairingProvider struct {
	data map[string]interface{}
	err  error
}

func (m *mockPairingProvider) GeneratePairingData() (map[string]interface{}, error) {
	return m.data, m.err
}

// mockTurnExecutorFactory is a no-op factory for WS tests.
// The real TurnExecutor would call LLM — we just need the processor to start without panic.
type mockTurnExecutorFactory struct{}

func (f *mockTurnExecutorFactory) CreateForSession(_ interface{ Dispose() }, _, _, _, _ string) interface {
	ExecuteTurn(interface{}, string, string, string, func(string) error, func(interface{}) error) error
} {
	return nil
}

// --- helpers ---

func activeLicense() *domain.LicenseInfo {
	return &domain.LicenseInfo{Status: domain.LicenseActive}
}

func setupTestServer(t *testing.T) (*httptest.Server, *websocket.Conn) {
	t.Helper()
	return setupTestServerWithLicense(t, activeLicense())
}

func setupTestServerWithLicense(t *testing.T, license *domain.LicenseInfo) (*httptest.Server, *websocket.Conn) {
	t.Helper()

	registry := flow_registry.NewSessionRegistry(nil)
	processor := sp.New(registry, nil, nil) // nil factory/store is OK — we don't send messages in ping/create tests
	agentSvc := &mockAgentEnvSetter{}

	handler := NewConnectionHandler(registry, processor, agentSvc, nil, license)

	server := httptest.NewServer(http.HandlerFunc(handler.ServeHTTP))
	t.Cleanup(server.Close)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	return server, conn
}

func setupTestServerWithRegistry(t *testing.T) (*httptest.Server, *websocket.Conn, *flow_registry.SessionRegistry, *ConnectionHandler) {
	t.Helper()
	return setupTestServerWithRegistryAndLicense(t, activeLicense())
}

func setupTestServerWithRegistryAndLicense(t *testing.T, license *domain.LicenseInfo) (*httptest.Server, *websocket.Conn, *flow_registry.SessionRegistry, *ConnectionHandler) {
	t.Helper()

	registry := flow_registry.NewSessionRegistry(nil)
	processor := sp.New(registry, nil, nil)
	agentSvc := &mockAgentEnvSetter{}

	handler := NewConnectionHandler(registry, processor, agentSvc, nil, license)

	server := httptest.NewServer(http.HandlerFunc(handler.ServeHTTP))
	t.Cleanup(server.Close)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	return server, conn, registry, handler
}

func sendAndReceive(t *testing.T, conn *websocket.Conn, msg WsMessage) WsMessage {
	t.Helper()

	data, err := json.Marshal(msg)
	require.NoError(t, err)
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, data))

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, respData, err := conn.ReadMessage()
	require.NoError(t, err)

	var resp WsMessage
	require.NoError(t, json.Unmarshal(respData, &resp))
	return resp
}

// --- tests ---

// TC-WS-01: Ping/pong — send ping, receive pong with timestamp.
func TestPingPong(t *testing.T) {
	_, conn := setupTestServer(t)

	resp := sendAndReceive(t, conn, WsMessage{
		Type:      "ping",
		RequestID: "req-1",
	})

	assert.Equal(t, "pong", resp.Type)
	assert.Equal(t, "req-1", resp.RequestID)
	assert.NotNil(t, resp.Payload["timestamp"])

	// Verify timestamp is a reasonable value (within last few seconds)
	ts, ok := resp.Payload["timestamp"].(float64)
	require.True(t, ok, "timestamp should be a number")
	assert.InDelta(t, float64(time.Now().UnixMilli()), ts, 5000)
}

// TC-WS-02: Create session — send create_session, receive ack with session_id.
func TestCreateSession(t *testing.T) {
	_, conn, registry, _ := setupTestServerWithRegistry(t)

	resp := sendAndReceive(t, conn, WsMessage{
		Type:      "create_session",
		RequestID: "req-2",
		Payload: map[string]interface{}{
			"project_root": "/my/project",
			"platform":     "linux",
			"project_key":  "proj-1",
		},
	})

	assert.Equal(t, "create_session_ack", resp.Type)
	assert.Equal(t, "req-2", resp.RequestID)

	sessionID, ok := resp.Payload["session_id"].(string)
	require.True(t, ok, "session_id should be a string")
	assert.NotEmpty(t, sessionID)

	// Verify session exists in registry
	assert.True(t, registry.HasSession(sessionID))

	root, platform, _, _, _, ctxOk := registry.GetSessionContext(sessionID)
	require.True(t, ctxOk)
	assert.Equal(t, "/my/project", root)
	assert.Equal(t, "linux", platform)
}

// TC-WS-03: Send message to non-existent session — error.
func TestSendMessage_SessionNotFound(t *testing.T) {
	_, conn := setupTestServer(t)

	resp := sendAndReceive(t, conn, WsMessage{
		Type:      "send_message",
		RequestID: "req-3",
		Payload: map[string]interface{}{
			"session_id": "nonexistent",
			"content":    "hello",
		},
	})

	assert.Equal(t, "error", resp.Type)
	assert.Equal(t, "req-3", resp.RequestID)
	assert.Contains(t, resp.Payload["error"], "session not found")
}

// TC-WS-03b: Send message without required fields — error.
func TestSendMessage_MissingFields(t *testing.T) {
	_, conn := setupTestServer(t)

	resp := sendAndReceive(t, conn, WsMessage{
		Type:      "send_message",
		RequestID: "req-3b",
		Payload:   map[string]interface{}{},
	})

	assert.Equal(t, "error", resp.Type)
	assert.Contains(t, resp.Payload["error"], "required")
}

// TC-WS-04: Generate pairing — send generate_pairing, receive pairing_data.
func TestGeneratePairing(t *testing.T) {
	_, conn, _, handler := setupTestServerWithRegistry(t)

	handler.SetPairingProvider(&mockPairingProvider{
		data: map[string]interface{}{
			"server_id":         "srv-1",
			"server_public_key": "abc123",
			"bridge_url":        "wss://bridge.example.com",
			"token":             "tok-xyz",
		},
	})

	resp := sendAndReceive(t, conn, WsMessage{
		Type:      "generate_pairing",
		RequestID: "req-4",
	})

	assert.Equal(t, "pairing_data", resp.Type)
	assert.Equal(t, "req-4", resp.RequestID)
	assert.Equal(t, "srv-1", resp.Payload["server_id"])
	assert.Equal(t, "abc123", resp.Payload["server_public_key"])
	assert.Equal(t, "wss://bridge.example.com", resp.Payload["bridge_url"])
	assert.Equal(t, "tok-xyz", resp.Payload["token"])
}

// TC-WS-05: Generate pairing without bridge — error.
func TestGeneratePairing_NoBridge(t *testing.T) {
	_, conn := setupTestServer(t)

	resp := sendAndReceive(t, conn, WsMessage{
		Type:      "generate_pairing",
		RequestID: "req-5",
	})

	assert.Equal(t, "error", resp.Type)
	assert.Equal(t, "req-5", resp.RequestID)
	assert.Contains(t, resp.Payload["error"], "bridge not configured")
}

// TC-WS-06: Unknown message type — error.
func TestUnknownMessageType(t *testing.T) {
	_, conn := setupTestServer(t)

	resp := sendAndReceive(t, conn, WsMessage{
		Type:      "foobar",
		RequestID: "req-6",
	})

	assert.Equal(t, "error", resp.Type)
	assert.Contains(t, resp.Payload["error"], "unknown message type")
}

// TC-WS-07: Invalid JSON — error response.
func TestInvalidJSON(t *testing.T) {
	_, conn := setupTestServer(t)

	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte("{not json")))

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, respData, err := conn.ReadMessage()
	require.NoError(t, err)

	var resp WsMessage
	require.NoError(t, json.Unmarshal(respData, &resp))

	assert.Equal(t, "error", resp.Type)
	assert.Contains(t, resp.Payload["error"], "invalid JSON")
}

// TC-WS-08: Subscribe to session and receive events.
func TestSubscribeAndReceiveEvents(t *testing.T) {
	_, conn, registry, _ := setupTestServerWithRegistry(t)

	// First create a session
	createResp := sendAndReceive(t, conn, WsMessage{
		Type:      "create_session",
		RequestID: "req-create",
		Payload: map[string]interface{}{
			"project_root": "/project",
			"platform":     "linux",
		},
	})
	sessionID := createResp.Payload["session_id"].(string)

	// Subscribe
	subResp := sendAndReceive(t, conn, WsMessage{
		Type:      "subscribe",
		RequestID: "req-sub",
		Payload: map[string]interface{}{
			"session_id": sessionID,
		},
	})
	assert.Equal(t, "subscribe_ack", subResp.Type)

	// Publish an event from the server side
	registry.PublishEvent(sessionID, &pb.SessionEvent{
		EventId: "evt-1",
		Type:    pb.SessionEventType_SESSION_EVENT_ANSWER,
		Content: "Hello from agent",
		AgentId: "supervisor",
	})

	// Read messages from WS — skip backfill_complete, find session_event
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var eventMsg WsMessage
	for {
		_, respData, err := conn.ReadMessage()
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(respData, &eventMsg))
		if eventMsg.Type != "backfill_complete" {
			break
		}
	}

	assert.Equal(t, "session_event", eventMsg.Type)
	assert.Equal(t, sessionID, eventMsg.Payload["session_id"])
	assert.Equal(t, "evt-1", eventMsg.Payload["event_id"])

	event, ok := eventMsg.Payload["event"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "MessageCompleted", event["type"])
	assert.Equal(t, "Hello from agent", event["content"])
}

// TC-WS-09: Subscribe to non-existent session — error.
func TestSubscribe_SessionNotFound(t *testing.T) {
	_, conn := setupTestServer(t)

	resp := sendAndReceive(t, conn, WsMessage{
		Type:      "subscribe",
		RequestID: "req-sub-bad",
		Payload: map[string]interface{}{
			"session_id": "nonexistent",
		},
	})

	assert.Equal(t, "error", resp.Type)
	assert.Contains(t, resp.Payload["error"], "session not found")
}

// TC-WS-10: Cancel session via WS.
func TestCancelSession(t *testing.T) {
	_, conn, registry, _ := setupTestServerWithRegistry(t)

	// Create session
	createResp := sendAndReceive(t, conn, WsMessage{
		Type:      "create_session",
		RequestID: "req-create",
		Payload:   map[string]interface{}{"project_root": "/p", "platform": "linux"},
	})
	sessionID := createResp.Payload["session_id"].(string)

	// Cancel
	cancelResp := sendAndReceive(t, conn, WsMessage{
		Type:      "cancel_session",
		RequestID: "req-cancel",
		Payload:   map[string]interface{}{"session_id": sessionID},
	})

	assert.Equal(t, "cancel_session_ack", cancelResp.Type)
	assert.Equal(t, true, cancelResp.Payload["cancelled"])

	// Verify session is cancelled in registry
	assert.True(t, registry.IsCancelled(sessionID))
}

// TC-WS-11: Cancel session without session_id — error.
func TestCancelSession_MissingSessionID(t *testing.T) {
	_, conn := setupTestServer(t)

	resp := sendAndReceive(t, conn, WsMessage{
		Type:      "cancel_session",
		RequestID: "req-cancel-bad",
		Payload:   map[string]interface{}{},
	})

	assert.Equal(t, "error", resp.Type)
	assert.Contains(t, resp.Payload["error"], "session_id required")
}

// TC-WS-12: AskUserReply forwarded to registry.
func TestAskUserReply(t *testing.T) {
	_, conn, registry, _ := setupTestServerWithRegistry(t)

	// Create session and register ask_user
	registry.CreateSession("s1", "proj", "user", "/root", "linux", "")
	replyCh := registry.RegisterAskUser("s1", "call-42")

	resp := sendAndReceive(t, conn, WsMessage{
		Type:      "ask_user_reply",
		RequestID: "req-reply",
		Payload: map[string]interface{}{
			"session_id": "s1",
			"call_id":    "call-42",
			"reply":      "yes, proceed",
		},
	})

	assert.Equal(t, "ask_user_reply_ack", resp.Type)

	// Verify the reply was delivered
	select {
	case reply := <-replyCh:
		assert.Equal(t, "yes, proceed", reply)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ask_user reply")
	}
}

// TC-WS-13: Create session with blocked license — error.
func TestCreateSession_LicenseBlocked(t *testing.T) {
	_, conn := setupTestServerWithLicense(t, domain.BlockedLicense())

	resp := sendAndReceive(t, conn, WsMessage{
		Type:      "create_session",
		RequestID: "req-blocked",
		Payload: map[string]interface{}{
			"project_root": "/my/project",
			"platform":     "linux",
		},
	})

	assert.Equal(t, "error", resp.Type)
	assert.Equal(t, "req-blocked", resp.RequestID)
	assert.Contains(t, resp.Payload["error"], "LICENSE_REQUIRED")
}

// TC-WS-14: Send message with blocked license — error.
func TestSendMessage_LicenseBlocked(t *testing.T) {
	_, conn := setupTestServerWithLicense(t, domain.BlockedLicense())

	resp := sendAndReceive(t, conn, WsMessage{
		Type:      "send_message",
		RequestID: "req-blocked-msg",
		Payload: map[string]interface{}{
			"session_id": "any-session",
			"content":    "hello",
		},
	})

	assert.Equal(t, "error", resp.Type)
	assert.Equal(t, "req-blocked-msg", resp.RequestID)
	assert.Contains(t, resp.Payload["error"], "LICENSE_REQUIRED")
}

// TC-WS-15: Create session with grace license — allowed with warning.
func TestCreateSession_LicenseGrace(t *testing.T) {
	graceLicense := &domain.LicenseInfo{Status: domain.LicenseGrace}
	_, conn, registry, _ := setupTestServerWithRegistryAndLicense(t, graceLicense)

	resp := sendAndReceive(t, conn, WsMessage{
		Type:      "create_session",
		RequestID: "req-grace",
		Payload: map[string]interface{}{
			"project_root": "/my/project",
			"platform":     "linux",
		},
	})

	assert.Equal(t, "create_session_ack", resp.Type)
	assert.Equal(t, "req-grace", resp.RequestID)

	sessionID, ok := resp.Payload["session_id"].(string)
	require.True(t, ok, "session_id should be a string")
	assert.NotEmpty(t, sessionID)
	assert.True(t, registry.HasSession(sessionID))

	// Verify warning is present
	assert.Equal(t, "license expiring soon", resp.Payload["warning"])
}

// TC-WS-16: Send message with grace license — passes license check (not blocked).
// Verifies grace license doesn't block: sends to non-existent session,
// expects "session not found" (not LICENSE_REQUIRED), proving license check passed.
func TestSendMessage_LicenseGrace(t *testing.T) {
	graceLicense := &domain.LicenseInfo{Status: domain.LicenseGrace}
	_, conn := setupTestServerWithLicense(t, graceLicense)

	resp := sendAndReceive(t, conn, WsMessage{
		Type:      "send_message",
		RequestID: "req-msg-grace",
		Payload: map[string]interface{}{
			"session_id": "nonexistent",
			"content":    "hello",
		},
	})

	// Grace license passes the check — we get "session not found", not LICENSE_REQUIRED
	assert.Equal(t, "error", resp.Type)
	assert.Contains(t, resp.Payload["error"], "session not found")
}

// TC-WS-17: Ping works regardless of license status (even blocked).
func TestPing_LicenseBlocked_StillWorks(t *testing.T) {
	_, conn := setupTestServerWithLicense(t, domain.BlockedLicense())

	resp := sendAndReceive(t, conn, WsMessage{
		Type:      "ping",
		RequestID: "req-ping-blocked",
	})

	assert.Equal(t, "pong", resp.Type)
	assert.Equal(t, "req-ping-blocked", resp.RequestID)
	assert.NotNil(t, resp.Payload["timestamp"])
}

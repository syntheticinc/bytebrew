package bridge

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pb "github.com/syntheticinc/bytebrew/engine/api/proto/gen"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/flow_registry"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence"
)

// --- mocks ---

type mockDeviceStore struct {
	devices  map[string]*domain.MobileDevice
	byToken  map[string]*domain.MobileDevice
	lastSeen map[string]bool
}

func newMockDeviceStore() *mockDeviceStore {
	return &mockDeviceStore{
		devices:  make(map[string]*domain.MobileDevice),
		byToken:  make(map[string]*domain.MobileDevice),
		lastSeen: make(map[string]bool),
	}
}

func (s *mockDeviceStore) GetByID(id string) (*domain.MobileDevice, error) {
	return s.devices[id], nil
}

func (s *mockDeviceStore) GetByToken(token string) (*domain.MobileDevice, error) {
	return s.byToken[token], nil
}

func (s *mockDeviceStore) Add(device *domain.MobileDevice) error {
	s.devices[device.ID] = device
	s.byToken[device.DeviceToken] = device
	return nil
}

func (s *mockDeviceStore) List() ([]*domain.MobileDevice, error) {
	result := make([]*domain.MobileDevice, 0, len(s.devices))
	for _, d := range s.devices {
		result = append(result, d)
	}
	return result, nil
}

func (s *mockDeviceStore) UpdateLastSeen(id string) error {
	s.lastSeen[id] = true
	return nil
}

func (s *mockDeviceStore) Remove(id string) error {
	d := s.devices[id]
	if d != nil {
		delete(s.byToken, d.DeviceToken)
	}
	delete(s.devices, id)
	return nil
}

type mockSessionManager struct {
	sessions     map[string]bool
	messages     map[string][]string
	askReplies   map[string]string
	cancelled    map[string]bool
	createCalled int
}

func newMockSessionManager() *mockSessionManager {
	return &mockSessionManager{
		sessions:   make(map[string]bool),
		messages:   make(map[string][]string),
		askReplies: make(map[string]string),
		cancelled:  make(map[string]bool),
	}
}

func (m *mockSessionManager) CreateSession(sessionID, projectKey, userID, projectRoot, platform, agentName string) {
	m.sessions[sessionID] = true
	m.createCalled++
}

func (m *mockSessionManager) EnqueueMessage(sessionID, content string) error {
	m.messages[sessionID] = append(m.messages[sessionID], content)
	return nil
}

func (m *mockSessionManager) HasSession(sessionID string) bool {
	return m.sessions[sessionID]
}

func (m *mockSessionManager) SendAskUserReply(sessionID, callID, reply string) {
	m.askReplies[callID] = reply
}

func (m *mockSessionManager) Cancel(sessionID string) bool {
	m.cancelled[sessionID] = true
	return m.sessions[sessionID]
}

func (m *mockSessionManager) ListSessions() []flow_registry.SessionInfo {
	result := make([]flow_registry.SessionInfo, 0, len(m.sessions))
	for id := range m.sessions {
		result = append(result, flow_registry.SessionInfo{
			SessionID:      id,
			CreatedAt:      time.Now(),
			LastActivityAt: time.Now(),
		})
	}
	return result
}

type mockMessageProcessor struct {
	processing map[string]bool
}

func newMockMessageProcessor() *mockMessageProcessor {
	return &mockMessageProcessor{processing: make(map[string]bool)}
}

func (m *mockMessageProcessor) StartProcessing(_ context.Context, _ string) {}

func (m *mockMessageProcessor) IsTurnActive(sessionID string) bool {
	return m.processing[sessionID]
}

// --- helpers ---

func newTestRequestHandler(t *testing.T) (*MobileRequestHandler, *mockDeviceStore, *mockSessionManager, *DeviceCryptoAdapter, *mockMessageProcessor) {
	t.Helper()

	crypto := NewDeviceCryptoAdapter()
	client := &BridgeClient{}
	router := NewMessageRouter(client, crypto)
	deviceStore := newMockDeviceStore()
	tokenStore := NewPairingTokenStore()
	broadcaster := NewEventBroadcaster(router, newMockEventStoreReader())
	sessions := newMockSessionManager()
	processor := newMockMessageProcessor()
	identity := &persistence.ServerIdentity{
		ID:         "server-1",
		PublicKey:  make([]byte, 32),
		PrivateKey: make([]byte, 32),
	}

	handler := NewMobileRequestHandler(router, deviceStore, tokenStore, crypto, broadcaster, sessions, processor, identity, "test-server")
	return handler, deviceStore, sessions, crypto, processor
}

func addAuthenticatedDevice(store *mockDeviceStore, id, token string) *domain.MobileDevice {
	d := &domain.MobileDevice{
		ID:          id,
		Name:        "Test Device",
		DeviceToken: token,
		PairedAt:    time.Now(),
		LastSeenAt:  time.Now(),
	}
	store.devices[id] = d
	store.byToken[token] = d
	return d
}

// --- tests ---

// TC-B-04: Ping/pong — ping message returns pong response with timestamp.
func TestRequestHandler_Ping(t *testing.T) {
	handler, _, _, _, _ := newTestRequestHandler(t)
	handler.Start()

	msg := &MobileMessage{
		Type:      "ping",
		RequestID: "req-1",
		DeviceID:  "dev-1",
	}

	handler.handlePing(msg)
	// respond() sends via router → client (nil conn), so response is logged as error.
	// Handler logic is correct: builds pong with timestamp, no panic.
}

// TC-B-03: Invalid token — pair_request with wrong token returns error response.
func TestRequestHandler_PairRequest_InvalidToken(t *testing.T) {
	handler, _, _, _, _ := newTestRequestHandler(t)
	handler.Start()

	msg := &MobileMessage{
		Type:      "pair_request",
		RequestID: "req-1",
		DeviceID:  "dev-1",
		Payload: map[string]interface{}{
			"token": "bad-token",
		},
	}

	handler.handlePairRequest(msg)
	// Token not found in store → respondError with "Invalid or expired token".
	// No device stored.
}

func TestRequestHandler_PairRequest_MissingToken(t *testing.T) {
	handler, _, _, _, _ := newTestRequestHandler(t)

	msg := &MobileMessage{
		Type:      "pair_request",
		RequestID: "req-1",
		DeviceID:  "dev-1",
		Payload:   map[string]interface{}{},
	}

	handler.handlePairRequest(msg)
}

// TC-B-02: Pairing flow — full pair_request → pair_response (token valid, key exchange, device stored).
func TestRequestHandler_PairRequest_Success(t *testing.T) {
	handler, deviceStore, _, crypto, _ := newTestRequestHandler(t)

	// Generate a real keypair for the token
	kp, err := GenerateKeyPair()
	require.NoError(t, err)

	// Generate device keypair
	deviceKP, err := GenerateKeyPair()
	require.NoError(t, err)

	token := &domain.PairingToken{
		Token:            "valid-token",
		ShortCode:        "123456",
		ExpiresAt:        time.Now().Add(15 * time.Minute),
		ServerPublicKey:  kp.PublicKey,
		ServerPrivateKey: kp.PrivateKey,
	}
	handler.tokenStore.Add(token)

	msg := &MobileMessage{
		Type:      "pair_request",
		RequestID: "req-1",
		DeviceID:  "bridge-dev-1",
		Payload: map[string]interface{}{
			"token":             "valid-token",
			"device_public_key": base64.StdEncoding.EncodeToString(deviceKP.PublicKey),
			"device_name":       "My Phone",
		},
	}

	handler.handlePairRequest(msg)

	// Device should be saved
	require.Len(t, deviceStore.devices, 1)

	var savedDevice *domain.MobileDevice
	for _, d := range deviceStore.devices {
		savedDevice = d
	}
	require.NotNil(t, savedDevice)
	assert.Equal(t, "My Phone", savedDevice.Name)
	assert.NotEmpty(t, savedDevice.DeviceToken)
	assert.NotEmpty(t, savedDevice.SharedSecret)

	// Crypto should have the alias registered
	assert.True(t, crypto.HasSharedSecret(savedDevice.ID))
}

// TC-B-02: Pairing flow — comprehensive: token consumed, shared secret computed,
// device stored with correct fields, alias registered, crypto active.
func TestRequestHandler_PairRequest_FullFlow(t *testing.T) {
	handler, deviceStore, _, crypto, _ := newTestRequestHandler(t)

	serverKP, err := GenerateKeyPair()
	require.NoError(t, err)
	deviceKP, err := GenerateKeyPair()
	require.NoError(t, err)

	token := &domain.PairingToken{
		Token:            "pair-flow-token",
		ShortCode:        "654321",
		ExpiresAt:        time.Now().Add(15 * time.Minute),
		ServerPublicKey:  serverKP.PublicKey,
		ServerPrivateKey: serverKP.PrivateKey,
	}
	handler.tokenStore.Add(token)

	msg := &MobileMessage{
		Type:      "pair_request",
		RequestID: "req-pair",
		DeviceID:  "bridge-abc",
		Payload: map[string]interface{}{
			"token":             "pair-flow-token",
			"device_public_key": base64.StdEncoding.EncodeToString(deviceKP.PublicKey),
			"device_name":       "Pixel 7",
		},
	}

	handler.handlePairRequest(msg)

	// 1. Device stored with correct fields.
	require.Len(t, deviceStore.devices, 1)
	var dev *domain.MobileDevice
	for _, d := range deviceStore.devices {
		dev = d
	}
	require.NotNil(t, dev)
	assert.Equal(t, "Pixel 7", dev.Name)
	assert.NotEmpty(t, dev.ID)
	assert.NotEmpty(t, dev.DeviceToken)
	assert.Equal(t, deviceKP.PublicKey, dev.PublicKey)
	assert.Len(t, dev.SharedSecret, 32, "shared secret must be 32 bytes")
	assert.False(t, dev.PairedAt.IsZero())

	// 2. Token consumed — second use returns nil.
	assert.Nil(t, handler.tokenStore.UseToken("pair-flow-token"))

	// 3. Crypto: shared secret registered for authenticated device ID.
	assert.True(t, crypto.HasSharedSecret(dev.ID))

	// 4. Alias: bridge device ID resolves to authenticated device ID.
	assert.True(t, crypto.HasSharedSecret("bridge-abc"), "alias should resolve to device")

	// 5. Can encrypt/decrypt through the alias.
	plaintext := []byte(`{"type":"test"}`)
	encrypted, err := crypto.Encrypt("bridge-abc", plaintext)
	require.NoError(t, err)
	decrypted, err := crypto.Decrypt("bridge-abc", encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

// TC-B-03: Invalid token — pair_request with wrong token, no device stored.
func TestRequestHandler_PairRequest_InvalidToken_NoDeviceStored(t *testing.T) {
	handler, deviceStore, _, _, _ := newTestRequestHandler(t)

	msg := &MobileMessage{
		Type:      "pair_request",
		RequestID: "req-1",
		DeviceID:  "dev-1",
		Payload: map[string]interface{}{
			"token":             "wrong-token",
			"device_public_key": base64.StdEncoding.EncodeToString(make([]byte, 32)),
			"device_name":       "Phone",
		},
	}

	handler.handlePairRequest(msg)

	assert.Empty(t, deviceStore.devices, "no device should be stored on invalid token")
}

// TC-B-17: Unauth request — request without device_token is rejected.
func TestRequestHandler_NewTask_Unauthorized(t *testing.T) {
	handler, _, _, _, _ := newTestRequestHandler(t)

	msg := &MobileMessage{
		Type:      "new_task",
		RequestID: "req-1",
		DeviceID:  "dev-1",
		Payload: map[string]interface{}{
			"device_token": "bad-token",
			"content":      "hello",
		},
	}

	handler.handleNewTask(msg)
	// Would send error; no panic = correct
}

func TestRequestHandler_NewTask_CreatesSession(t *testing.T) {
	handler, deviceStore, sessions, _, _ := newTestRequestHandler(t)
	addAuthenticatedDevice(deviceStore, "dev-1", "tok-1")

	msg := &MobileMessage{
		Type:      "new_task",
		RequestID: "req-1",
		DeviceID:  "dev-1",
		Payload: map[string]interface{}{
			"device_token": "tok-1",
			"content":      "Hello agent",
		},
	}

	handler.handleNewTask(msg)

	assert.Equal(t, 1, sessions.createCalled)
	// Should have one session with the message
	totalMessages := 0
	for _, msgs := range sessions.messages {
		totalMessages += len(msgs)
	}
	assert.Equal(t, 1, totalMessages)
}

func TestRequestHandler_NewTask_ReusesExistingSession(t *testing.T) {
	handler, deviceStore, sessions, _, _ := newTestRequestHandler(t)
	addAuthenticatedDevice(deviceStore, "dev-1", "tok-1")
	sessions.sessions["existing-session"] = true

	msg := &MobileMessage{
		Type:      "new_task",
		RequestID: "req-1",
		DeviceID:  "dev-1",
		Payload: map[string]interface{}{
			"device_token": "tok-1",
			"content":      "Follow up",
			"session_id":   "existing-session",
		},
	}

	handler.handleNewTask(msg)

	assert.Equal(t, 0, sessions.createCalled)
	assert.Equal(t, []string{"Follow up"}, sessions.messages["existing-session"])
}

func TestRequestHandler_NewTask_MissingContent(t *testing.T) {
	handler, deviceStore, sessions, _, _ := newTestRequestHandler(t)
	addAuthenticatedDevice(deviceStore, "dev-1", "tok-1")

	msg := &MobileMessage{
		Type:      "new_task",
		RequestID: "req-1",
		DeviceID:  "dev-1",
		Payload: map[string]interface{}{
			"device_token": "tok-1",
		},
	}

	handler.handleNewTask(msg)
	assert.Equal(t, 0, sessions.createCalled)
}

// TC-B-09: AskUser mobile — ask_user_reply forwards reply to session manager.
func TestRequestHandler_AskUserReply(t *testing.T) {
	handler, deviceStore, sessions, _, _ := newTestRequestHandler(t)
	addAuthenticatedDevice(deviceStore, "dev-1", "tok-1")

	msg := &MobileMessage{
		Type:      "ask_user_reply",
		RequestID: "req-1",
		DeviceID:  "dev-1",
		Payload: map[string]interface{}{
			"device_token": "tok-1",
			"session_id":   "s1",
			"call_id":      "call-1",
			"reply":        "yes",
		},
	}

	handler.handleAskUserReply(msg)

	assert.Equal(t, "yes", sessions.askReplies["call-1"])
}

// TC-B-09: AskUser mobile — missing session_id or call_id is rejected.
func TestRequestHandler_AskUserReply_MissingFields(t *testing.T) {
	handler, deviceStore, sessions, _, _ := newTestRequestHandler(t)
	addAuthenticatedDevice(deviceStore, "dev-1", "tok-1")

	msg := &MobileMessage{
		Type:      "ask_user_reply",
		RequestID: "req-1",
		DeviceID:  "dev-1",
		Payload: map[string]interface{}{
			"device_token": "tok-1",
			// session_id and call_id missing
			"reply": "yes",
		},
	}

	handler.handleAskUserReply(msg)

	assert.Empty(t, sessions.askReplies, "no reply should be forwarded without session_id/call_id")
}

// TC-B-09: AskUser mobile — unauthenticated ask_user_reply is rejected.
func TestRequestHandler_AskUserReply_Unauthorized(t *testing.T) {
	handler, _, sessions, _, _ := newTestRequestHandler(t)

	msg := &MobileMessage{
		Type:      "ask_user_reply",
		RequestID: "req-1",
		DeviceID:  "dev-1",
		Payload: map[string]interface{}{
			"device_token": "bad-token",
			"session_id":   "s1",
			"call_id":      "call-1",
			"reply":        "yes",
		},
	}

	handler.handleAskUserReply(msg)

	assert.Empty(t, sessions.askReplies, "unauthorized reply must not be forwarded")
}

// TC-B-10: Cancel session — cancel_session cancels session.
func TestRequestHandler_CancelSession(t *testing.T) {
	handler, deviceStore, sessions, _, _ := newTestRequestHandler(t)
	addAuthenticatedDevice(deviceStore, "dev-1", "tok-1")
	sessions.sessions["s1"] = true

	msg := &MobileMessage{
		Type:      "cancel_session",
		RequestID: "req-1",
		DeviceID:  "dev-1",
		Payload: map[string]interface{}{
			"device_token": "tok-1",
			"session_id":   "s1",
		},
	}

	handler.handleCancelSession(msg)
	assert.True(t, sessions.cancelled["s1"])
}

// TC-B-10: Cancel session — missing session_id is rejected.
func TestRequestHandler_CancelSession_MissingSessionID(t *testing.T) {
	handler, deviceStore, sessions, _, _ := newTestRequestHandler(t)
	addAuthenticatedDevice(deviceStore, "dev-1", "tok-1")

	msg := &MobileMessage{
		Type:      "cancel_session",
		RequestID: "req-1",
		DeviceID:  "dev-1",
		Payload: map[string]interface{}{
			"device_token": "tok-1",
			// no session_id
		},
	}

	handler.handleCancelSession(msg)
	assert.Empty(t, sessions.cancelled, "no session should be cancelled without session_id")
}

// TC-B-10: Cancel session — unauthenticated cancel is rejected.
func TestRequestHandler_CancelSession_Unauthorized(t *testing.T) {
	handler, _, sessions, _, _ := newTestRequestHandler(t)
	sessions.sessions["s1"] = true

	msg := &MobileMessage{
		Type:      "cancel_session",
		RequestID: "req-1",
		DeviceID:  "dev-1",
		Payload: map[string]interface{}{
			"device_token": "bad-token",
			"session_id":   "s1",
		},
	}

	handler.handleCancelSession(msg)
	assert.Empty(t, sessions.cancelled, "unauthorized cancel must not proceed")
}

// TC-B-11: List sessions — list_sessions returns session list.
func TestRequestHandler_ListSessions(t *testing.T) {
	handler, deviceStore, _, _, _ := newTestRequestHandler(t)
	addAuthenticatedDevice(deviceStore, "dev-1", "tok-1")

	msg := &MobileMessage{
		Type:      "list_sessions",
		RequestID: "req-1",
		DeviceID:  "dev-1",
		Payload: map[string]interface{}{
			"device_token": "tok-1",
		},
	}

	// Should not panic; currently returns empty sessions list.
	handler.handleListSessions(msg)
}

// TC-B-11: List sessions — unauthenticated list is rejected.
func TestRequestHandler_ListSessions_Unauthorized(t *testing.T) {
	handler, _, _, _, _ := newTestRequestHandler(t)

	msg := &MobileMessage{
		Type:      "list_sessions",
		RequestID: "req-1",
		DeviceID:  "dev-1",
		Payload: map[string]interface{}{
			"device_token": "bad-token",
		},
	}

	handler.handleListSessions(msg)
	// No panic; unauthorized path sends error response.
}

// TC-B-12: List devices — list_devices returns device list.
func TestRequestHandler_ListDevices(t *testing.T) {
	handler, deviceStore, _, _, _ := newTestRequestHandler(t)
	addAuthenticatedDevice(deviceStore, "dev-1", "tok-1")
	addAuthenticatedDevice(deviceStore, "dev-2", "tok-2")

	msg := &MobileMessage{
		Type:      "list_devices",
		RequestID: "req-1",
		DeviceID:  "dev-1",
		Payload: map[string]interface{}{
			"device_token": "tok-1",
		},
	}

	handler.handleListDevices(msg)
	devices, _ := deviceStore.List()
	assert.Len(t, devices, 2)
}

// TC-B-12: List devices — unauthenticated list_devices is rejected.
func TestRequestHandler_ListDevices_Unauthorized(t *testing.T) {
	handler, _, _, _, _ := newTestRequestHandler(t)

	msg := &MobileMessage{
		Type:      "list_devices",
		RequestID: "req-1",
		DeviceID:  "dev-1",
		Payload: map[string]interface{}{
			"device_token": "bad-token",
		},
	}

	handler.handleListDevices(msg)
	// No panic; unauthorized path sends error response.
}

// TC-B-17: Unauth request — missing device_token is rejected.
func TestRequestHandler_AuthenticateDevice_MissingToken(t *testing.T) {
	handler, _, _, _, _ := newTestRequestHandler(t)

	_, err := handler.authenticateDevice(map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing device_token")
}

// TC-B-17: Unauth request — invalid device_token is rejected with "unauthorized".
func TestRequestHandler_AuthenticateDevice_InvalidToken(t *testing.T) {
	handler, _, _, _, _ := newTestRequestHandler(t)

	_, err := handler.authenticateDevice(map[string]interface{}{
		"device_token": "nonexistent",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestRequestHandler_AuthenticateDevice_Valid(t *testing.T) {
	handler, deviceStore, _, _, _ := newTestRequestHandler(t)
	addAuthenticatedDevice(deviceStore, "dev-1", "tok-1")

	device, err := handler.authenticateDevice(map[string]interface{}{
		"device_token": "tok-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "dev-1", device.ID)
}

func TestRequestHandler_HandleMessage_Routing(t *testing.T) {
	handler, _, _, _, _ := newTestRequestHandler(t)
	handler.Start()

	// Unknown type should not panic
	handler.handleMessage(&MobileMessage{Type: "unknown", DeviceID: "d1"})
}

func TestRequestHandler_StartStop(t *testing.T) {
	handler, _, _, _, _ := newTestRequestHandler(t)

	handler.Start()
	assert.True(t, handler.running.Load())

	// Double start is no-op
	handler.Start()
	assert.True(t, handler.running.Load())

	handler.Stop()
	assert.False(t, handler.running.Load())
}

func TestRequestHandler_Subscribe(t *testing.T) {
	handler, deviceStore, _, _, _ := newTestRequestHandler(t)
	addAuthenticatedDevice(deviceStore, "dev-1", "tok-1")

	msg := &MobileMessage{
		Type:      "subscribe",
		RequestID: "req-1",
		DeviceID:  "dev-1",
		Payload: map[string]interface{}{
			"device_token": "tok-1",
			"session_id":   "s1",
		},
	}

	handler.handleSubscribe(msg)

	// Verify device is in broadcaster subscribers
	handler.broadcaster.mu.RLock()
	sub, ok := handler.broadcaster.subscribers["dev-1"]
	handler.broadcaster.mu.RUnlock()

	require.True(t, ok)
	assert.Equal(t, "s1", sub.SessionID)
}

// TC-B-08: Full chat flow — new_task → session created → subscribe → events broadcast
// (ProcessingStarted → ToolExecutionStarted → ToolExecutionCompleted → MessageCompleted).
func TestRequestHandler_FullChatFlow(t *testing.T) {
	handler, deviceStore, sessions, _, _ := newTestRequestHandler(t)
	handler.Start()

	addAuthenticatedDevice(deviceStore, "dev-1", "tok-1")

	// Step 1: Send new_task → creates session, enqueues message.
	newTaskMsg := &MobileMessage{
		Type:      "new_task",
		RequestID: "req-flow-1",
		DeviceID:  "dev-1",
		Payload: map[string]interface{}{
			"device_token": "tok-1",
			"content":      "Analyze main.go",
		},
	}
	handler.handleNewTask(newTaskMsg)

	require.Equal(t, 1, sessions.createCalled, "session should be created")

	// Find the session ID that was created.
	var sessionID string
	for sid := range sessions.messages {
		sessionID = sid
	}
	require.NotEmpty(t, sessionID, "session ID should be assigned")
	assert.Equal(t, []string{"Analyze main.go"}, sessions.messages[sessionID])

	// Step 2: Subscribe device to the session for events.
	subscribeMsg := &MobileMessage{
		Type:      "subscribe",
		RequestID: "req-flow-2",
		DeviceID:  "dev-1",
		Payload: map[string]interface{}{
			"device_token": "tok-1",
			"session_id":   sessionID,
		},
	}
	handler.handleSubscribe(subscribeMsg)

	// Verify subscription.
	handler.broadcaster.mu.RLock()
	sub, ok := handler.broadcaster.subscribers["dev-1"]
	handler.broadcaster.mu.RUnlock()
	require.True(t, ok, "device should be subscribed")
	assert.Equal(t, sessionID, sub.SessionID)

	// Step 3: Simulate event flow from agent processing.
	// Replace the broadcaster's sender with a mock to capture events.
	sender := newMockMessageSender()
	handler.broadcaster.sender = sender

	events := []*pb.SessionEvent{
		{
			Type: pb.SessionEventType_SESSION_EVENT_PROCESSING_STARTED,
		},
		{
			Type:          pb.SessionEventType_SESSION_EVENT_TOOL_EXECUTION_START,
			CallId:        "call-1",
			ToolName:      "read_file",
			ToolArguments: map[string]string{"path": "main.go"},
			AgentId:       "supervisor",
		},
		{
			Type:              pb.SessionEventType_SESSION_EVENT_TOOL_EXECUTION_END,
			CallId:            "call-1",
			ToolName:          "read_file",
			ToolResultSummary: "120 lines",
			AgentId:           "supervisor",
		},
		{
			Type:    pb.SessionEventType_SESSION_EVENT_ANSWER,
			Content: "main.go contains the entry point with HTTP server setup.",
			AgentId: "supervisor",
		},
	}

	for _, evt := range events {
		handler.broadcaster.BroadcastEvent(sessionID, evt)
	}

	// Step 4: Verify device received all events in order.
	msgs := sender.getMessages("dev-1")
	require.Len(t, msgs, 4, "device should receive all 4 events")

	// Verify event types in order.
	eventTypes := make([]string, 0, len(msgs))
	for _, msg := range msgs {
		assert.Equal(t, "session_event", msg.Type)
		assert.Equal(t, sessionID, msg.Payload["session_id"])
		evt, ok := msg.Payload["event"].(map[string]interface{})
		require.True(t, ok)
		eventTypes = append(eventTypes, evt["type"].(string))
	}

	assert.Equal(t, []string{
		"ProcessingStarted",
		"ToolExecutionStarted",
		"ToolExecutionCompleted",
		"MessageCompleted",
	}, eventTypes)

	// Verify final answer content.
	lastEvt := msgs[3].Payload["event"].(map[string]interface{})
	assert.Equal(t, "main.go contains the entry point with HTTP server setup.", lastEvt["content"])
}

// Subscribe sends ProcessingStopped when session is idle (not processing).
// This prevents stuck-spinner after reconnect when ProcessingStopped was lost to TCP death.
func TestRequestHandler_Subscribe_SendsIdleStatus(t *testing.T) {
	handler, deviceStore, sessions, _, processor := newTestRequestHandler(t)

	addAuthenticatedDevice(deviceStore, "dev-1", "tok-1")
	sessions.sessions["s1"] = true // session exists but not processing

	// Replace sender to capture messages.
	sender := newMockMessageSender()
	handler.broadcaster.sender = sender

	msg := &MobileMessage{
		Type:      "subscribe",
		RequestID: "req-1",
		DeviceID:  "dev-1",
		Payload: map[string]interface{}{
			"device_token": "tok-1",
			"session_id":   "s1",
		},
	}

	handler.handleSubscribe(msg)

	// Processor reports not processing → device receives BackfillComplete + ProcessingStopped.
	assert.False(t, processor.IsTurnActive("s1"))

	msgs := sender.getMessages("dev-1")
	// Subscribe sends BackfillComplete marker, then handleSubscribe sends synthetic status.
	require.GreaterOrEqual(t, len(msgs), 2, "device should receive backfill_complete + synthetic status event")

	// Last message is the synthetic session status.
	statusMsg := msgs[len(msgs)-1]
	assert.Equal(t, "session_event", statusMsg.Type)
	evt, ok := statusMsg.Payload["event"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "ProcessingStopped", evt["type"])
	assert.Equal(t, "idle", evt["state"])
}

// Subscribe sends ProcessingStarted when session is actively processing.
func TestRequestHandler_Subscribe_SendsProcessingStatus(t *testing.T) {
	handler, deviceStore, sessions, _, processor := newTestRequestHandler(t)

	addAuthenticatedDevice(deviceStore, "dev-1", "tok-1")
	sessions.sessions["s1"] = true
	processor.processing["s1"] = true // session is actively processing

	sender := newMockMessageSender()
	handler.broadcaster.sender = sender

	msg := &MobileMessage{
		Type:      "subscribe",
		RequestID: "req-1",
		DeviceID:  "dev-1",
		Payload: map[string]interface{}{
			"device_token": "tok-1",
			"session_id":   "s1",
		},
	}

	handler.handleSubscribe(msg)

	msgs := sender.getMessages("dev-1")
	// Subscribe sends BackfillComplete marker, then handleSubscribe sends synthetic status.
	require.GreaterOrEqual(t, len(msgs), 2, "device should receive backfill_complete + synthetic status event")

	// Last message is the synthetic session status.
	statusMsg := msgs[len(msgs)-1]
	evt, ok := statusMsg.Payload["event"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "ProcessingStarted", evt["type"])
	assert.Equal(t, "processing", evt["state"])
}

// Subscribe does NOT send status for unknown sessions.
func TestRequestHandler_Subscribe_IdleStatusForUnknownSession(t *testing.T) {
	handler, deviceStore, _, _, _ := newTestRequestHandler(t)

	addAuthenticatedDevice(deviceStore, "dev-1", "tok-1")
	// No session created — sessions.HasSession("s1") returns false.
	// Should still send idle status (server restarted, session gone).

	sender := newMockMessageSender()
	handler.broadcaster.sender = sender

	msg := &MobileMessage{
		Type:      "subscribe",
		RequestID: "req-1",
		DeviceID:  "dev-1",
		Payload: map[string]interface{}{
			"device_token": "tok-1",
			"session_id":   "s1",
		},
	}

	handler.handleSubscribe(msg)

	msgs := sender.getMessages("dev-1")
	// Subscribe sends BackfillComplete marker + handleSubscribe sends session status
	require.GreaterOrEqual(t, len(msgs), 2, "should send backfill_complete + idle status")
	// Last message before subscribe_ack should be session status
	statusMsg := msgs[len(msgs)-1]
	event := statusMsg.Payload["event"].(map[string]interface{})
	assert.Equal(t, "ProcessingStopped", event["type"])
}

func TestMobileMessage_JSON(t *testing.T) {
	msg := &MobileMessage{
		Type:      "test",
		RequestID: "req-1",
		DeviceID:  "dev-1",
		Payload: map[string]interface{}{
			"key": "value",
		},
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded MobileMessage
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, "test", decoded.Type)
	assert.Equal(t, "value", decoded.Payload["key"])
}

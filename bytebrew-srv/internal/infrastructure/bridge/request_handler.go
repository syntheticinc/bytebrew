package bridge

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/flow_registry"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/persistence"
)

// DeviceStore provides CRUD operations for paired mobile devices.
type DeviceStore interface {
	GetByID(id string) (*domain.MobileDevice, error)
	GetByToken(token string) (*domain.MobileDevice, error)
	Add(device *domain.MobileDevice) error
	List() ([]*domain.MobileDevice, error)
	UpdateLastSeen(id string) error
	Remove(id string) error
}

// SessionManager manages server-streaming sessions.
type SessionManager interface {
	CreateSession(sessionID, projectKey, userID, projectRoot, platform string)
	EnqueueMessage(sessionID, content string) error
	HasSession(sessionID string) bool
	SendAskUserReply(sessionID, callID, reply string)
	Cancel(sessionID string) bool
	ListSessions() []flow_registry.SessionInfo
}

// MessageProcessor starts background message processing for a session (consumer-side interface).
type MessageProcessor interface {
	StartProcessing(ctx context.Context, sessionID string)
}

// MobileRequestHandler routes incoming MobileMessages from the MessageRouter
// to the appropriate handler based on message type.
type MobileRequestHandler struct {
	router         *MessageRouter
	deviceStore    DeviceStore
	tokenStore     *PairingTokenStore
	crypto         *DeviceCryptoAdapter
	broadcaster    *EventBroadcaster
	sessions       SessionManager
	processor      MessageProcessor
	serverIdentity *persistence.ServerIdentity
	serverName     string

	running atomic.Bool
}

// NewMobileRequestHandler creates a new request handler with all dependencies.
func NewMobileRequestHandler(
	router *MessageRouter,
	deviceStore DeviceStore,
	tokenStore *PairingTokenStore,
	crypto *DeviceCryptoAdapter,
	broadcaster *EventBroadcaster,
	sessions SessionManager,
	processor MessageProcessor,
	serverIdentity *persistence.ServerIdentity,
	serverName string,
) *MobileRequestHandler {
	return &MobileRequestHandler{
		router:         router,
		deviceStore:    deviceStore,
		tokenStore:     tokenStore,
		crypto:         crypto,
		broadcaster:    broadcaster,
		sessions:       sessions,
		processor:      processor,
		serverIdentity: serverIdentity,
		serverName:     serverName,
	}
}

// Start registers the message handler on the router.
func (h *MobileRequestHandler) Start() {
	if !h.running.CompareAndSwap(false, true) {
		return
	}
	h.router.OnMessage(h.handleMessage)
	slog.Info("mobile request handler started")
}

// Stop marks the handler as stopped. Currently no background goroutines to cancel.
func (h *MobileRequestHandler) Stop() {
	h.running.Store(false)
	slog.Info("mobile request handler stopped")
}

func (h *MobileRequestHandler) handleMessage(msg *MobileMessage) {
	if !h.running.Load() {
		return
	}

	switch msg.Type {
	case "ping":
		h.handlePing(msg)
	case "pair_request":
		h.handlePairRequest(msg)
	case "new_task":
		h.handleNewTask(msg)
	case "subscribe":
		h.handleSubscribe(msg)
	case "ask_user_reply":
		h.handleAskUserReply(msg)
	case "cancel_session":
		h.handleCancelSession(msg)
	case "list_sessions":
		h.handleListSessions(msg)
	case "list_devices":
		h.handleListDevices(msg)
	default:
		slog.Warn("unknown mobile message type", "type", msg.Type, "device_id", msg.DeviceID)
	}
}

func (h *MobileRequestHandler) handlePing(msg *MobileMessage) {
	h.respond(msg, "pong", map[string]interface{}{
		"timestamp":   time.Now().UnixMilli(),
		"server_name": h.serverName,
		"server_id":   h.serverIdentity.ID,
	})
}

func (h *MobileRequestHandler) handlePairRequest(msg *MobileMessage) {
	tokenStr, _ := msg.Payload["token"].(string)
	if tokenStr == "" {
		h.respondError(msg, "pair_response", "missing token in pair_request")
		return
	}

	token := h.tokenStore.UseToken(tokenStr)
	if token == nil {
		h.respondError(msg, "pair_response", "Invalid or expired token")
		return
	}

	devicePubKeyB64, _ := msg.Payload["device_public_key"].(string)
	if devicePubKeyB64 == "" {
		h.respondError(msg, "pair_response", "missing device_public_key")
		return
	}

	devicePubKey, err := base64.StdEncoding.DecodeString(devicePubKeyB64)
	if err != nil {
		h.respondError(msg, "pair_response", "invalid device_public_key encoding")
		return
	}

	sharedSecret, err := ComputeSharedSecret(token.ServerPrivateKey, devicePubKey)
	if err != nil {
		slog.Error("compute shared secret failed", "error", err)
		h.respondError(msg, "pair_response", "key exchange failed")
		return
	}

	deviceName, _ := msg.Payload["device_name"].(string)
	if deviceName == "" {
		deviceName = "Mobile Device"
	}

	newDevice := &domain.MobileDevice{
		ID:           uuid.New().String(),
		Name:         deviceName,
		DeviceToken:  uuid.New().String(),
		PublicKey:    devicePubKey,
		SharedSecret: sharedSecret,
		PairedAt:     time.Now(),
		LastSeenAt:   time.Now(),
	}

	if err := h.deviceStore.Add(newDevice); err != nil {
		slog.Error("save paired device failed", "error", err)
		h.respondError(msg, "pair_response", "failed to save device")
		return
	}

	// CRITICAL: Send pair_response BEFORE RegisterAlias (plaintext first!)
	h.respond(msg, "pair_response", map[string]interface{}{
		"device_id":         newDevice.ID,
		"device_token":      newDevice.DeviceToken,
		"server_public_key": base64.StdEncoding.EncodeToString(token.ServerPublicKey),
	})

	// Now register alias and add crypto for future encrypted communication.
	h.crypto.RegisterAlias(msg.DeviceID, newDevice.ID)
	h.crypto.AddDevice(newDevice.ID, sharedSecret)

	slog.Info("device paired successfully",
		"device_id", newDevice.ID,
		"device_name", newDevice.Name,
		"bridge_device_id", msg.DeviceID,
	)
}

func (h *MobileRequestHandler) handleNewTask(msg *MobileMessage) {
	device, err := h.authenticateDevice(msg.Payload)
	if err != nil {
		h.respondError(msg, "new_task_ack", err.Error())
		return
	}

	_ = h.deviceStore.UpdateLastSeen(device.ID)

	content, _ := msg.Payload["content"].(string)
	if content == "" {
		h.respondError(msg, "new_task_ack", "missing content")
		return
	}

	sessionID, _ := msg.Payload["session_id"].(string)
	if sessionID == "" {
		sessionID = uuid.New().String()
	}
	if !h.sessions.HasSession(sessionID) {
		projectRoot, _ := msg.Payload["project_root"].(string)
		platform, _ := msg.Payload["platform"].(string)
		h.sessions.CreateSession(sessionID, "", device.ID, projectRoot, platform)
	}

	if err := h.sessions.EnqueueMessage(sessionID, content); err != nil {
		slog.Error("enqueue message failed", "session_id", sessionID, "error", err)
		h.respondError(msg, "new_task_ack", "failed to enqueue message")
		return
	}

	// Start message processing loop (idempotent — no-op if already running)
	h.processor.StartProcessing(context.Background(), sessionID)

	h.respond(msg, "new_task_ack", map[string]interface{}{
		"session_id": sessionID,
	})
}

func (h *MobileRequestHandler) handleSubscribe(msg *MobileMessage) {
	device, err := h.authenticateDevice(msg.Payload)
	if err != nil {
		h.respondError(msg, "subscribe_ack", err.Error())
		return
	}

	_ = h.deviceStore.UpdateLastSeen(device.ID)

	sessionID, _ := msg.Payload["session_id"].(string)
	if sessionID == "" {
		h.respondError(msg, "subscribe_ack", "missing session_id")
		return
	}

	lastEventID, _ := msg.Payload["last_event_id"].(string)
	h.broadcaster.Subscribe(device.ID, sessionID, lastEventID)

	h.respond(msg, "subscribe_ack", map[string]interface{}{
		"session_id": sessionID,
	})
}

func (h *MobileRequestHandler) handleAskUserReply(msg *MobileMessage) {
	device, err := h.authenticateDevice(msg.Payload)
	if err != nil {
		h.respondError(msg, "error", err.Error())
		return
	}

	_ = h.deviceStore.UpdateLastSeen(device.ID)

	sessionID, _ := msg.Payload["session_id"].(string)
	callID, _ := msg.Payload["call_id"].(string)
	reply, _ := msg.Payload["reply"].(string)

	if sessionID == "" || callID == "" {
		h.respondError(msg, "error", "missing session_id or call_id")
		return
	}

	h.sessions.SendAskUserReply(sessionID, callID, reply)
}

func (h *MobileRequestHandler) handleCancelSession(msg *MobileMessage) {
	_, err := h.authenticateDevice(msg.Payload)
	if err != nil {
		h.respondError(msg, "error", err.Error())
		return
	}

	sessionID, _ := msg.Payload["session_id"].(string)
	if sessionID == "" {
		h.respondError(msg, "error", "missing session_id")
		return
	}

	cancelled := h.sessions.Cancel(sessionID)
	h.respond(msg, "cancel_ack", map[string]interface{}{
		"session_id": sessionID,
		"cancelled":  cancelled,
	})
}

func (h *MobileRequestHandler) handleListSessions(msg *MobileMessage) {
	_, err := h.authenticateDevice(msg.Payload)
	if err != nil {
		h.respondError(msg, "error", err.Error())
		return
	}

	sessions := h.sessions.ListSessions()

	result := make([]map[string]interface{}, 0, len(sessions))
	for _, s := range sessions {
		status := "idle"
		if s.HasAskUser {
			status = "needs_attention"
		} else if s.IsCancelled {
			status = "idle"
		} else if time.Since(s.LastActivityAt) < 30*time.Second {
			status = "active"
		}

		projectName := s.ProjectKey
		if s.ProjectRoot != "" {
			projectName = filepath.Base(s.ProjectRoot)
		}

		result = append(result, map[string]interface{}{
			"session_id":       s.SessionID,
			"project_name":     projectName,
			"project_key":      s.ProjectKey,
			"project_root":     s.ProjectRoot,
			"platform":         s.Platform,
			"status":           status,
			"current_task":     "",
			"has_ask_user":     s.HasAskUser,
			"started_at":       s.CreatedAt.Format(time.RFC3339),
			"last_activity_at": s.LastActivityAt.Format(time.RFC3339),
		})
	}

	h.respond(msg, "sessions_list", map[string]interface{}{
		"sessions":    result,
		"server_name": h.serverName,
		"server_id":   h.serverIdentity.ID,
	})
}

func (h *MobileRequestHandler) handleListDevices(msg *MobileMessage) {
	_, err := h.authenticateDevice(msg.Payload)
	if err != nil {
		h.respondError(msg, "error", err.Error())
		return
	}

	devices, err := h.deviceStore.List()
	if err != nil {
		slog.Error("list devices failed", "error", err)
		h.respondError(msg, "error", "failed to list devices")
		return
	}

	result := make([]map[string]interface{}, 0, len(devices))
	for _, d := range devices {
		result = append(result, map[string]interface{}{
			"device_id":    d.ID,
			"device_name":  d.Name,
			"paired_at":    d.PairedAt.Format(time.RFC3339),
			"last_seen_at": d.LastSeenAt.Format(time.RFC3339),
		})
	}

	h.respond(msg, "devices_list", map[string]interface{}{
		"devices": result,
	})
}

// authenticateDevice validates the device_token from the payload and returns
// the corresponding MobileDevice or an error.
func (h *MobileRequestHandler) authenticateDevice(payload map[string]interface{}) (*domain.MobileDevice, error) {
	token, _ := payload["device_token"].(string)
	if token == "" {
		return nil, fmt.Errorf("missing device_token")
	}

	device, err := h.deviceStore.GetByToken(token)
	if err != nil {
		return nil, fmt.Errorf("unauthorized")
	}
	if device == nil {
		return nil, fmt.Errorf("unauthorized")
	}

	return device, nil
}

// respond sends a typed response back to the originating device.
func (h *MobileRequestHandler) respond(msg *MobileMessage, responseType string, payload map[string]interface{}) {
	response := &MobileMessage{
		Type:      responseType,
		RequestID: msg.RequestID,
		DeviceID:  msg.DeviceID,
		Payload:   payload,
	}
	if err := h.router.SendMessage(msg.DeviceID, response); err != nil {
		slog.Error("failed to send response", "error", err, "type", responseType, "device_id", msg.DeviceID)
	}
}

// respondError sends an error response with the given message.
func (h *MobileRequestHandler) respondError(msg *MobileMessage, responseType string, errMsg string) {
	h.respond(msg, responseType, map[string]interface{}{
		"error": errMsg,
	})
}

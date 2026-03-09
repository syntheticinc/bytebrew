package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
	sp "github.com/syntheticinc/bytebrew/bytebrew-srv/internal/service/session_processor"
)

// SessionRegistry provides session management for WS clients (consumer-side interface).
type SessionRegistry interface {
	CreateSession(sessionID, projectKey, userID, projectRoot, platform string)
	Subscribe(sessionID string) (ch <-chan *pb.SessionEvent, cleanup func())
	ReplayEvents(sessionID, lastEventID string) []*pb.SessionEvent
	EnqueueMessage(sessionID, content string) error
	DrainMessages(sessionID string)
	SendAskUserReply(sessionID, callID, reply string)
	Cancel(sessionID string) bool
	HasSession(sessionID string) bool
}

// AgentEnvironmentSetter sets environment context for the agent (consumer-side interface).
type AgentEnvironmentSetter interface {
	SetEnvironmentContext(projectRoot, platform string)
}

// PairingDataProvider generates pairing data for mobile device pairing (consumer-side interface).
type PairingDataProvider interface {
	GeneratePairingData() (map[string]interface{}, error)
}

// WsMessage is the wire format for client <-> server communication.
type WsMessage struct {
	Type      string                 `json:"type"`
	RequestID string                 `json:"request_id,omitempty"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
}

// ConnectionHandler handles WS upgrade and message dispatch.
type ConnectionHandler struct {
	upgrader         websocket.Upgrader
	sessionRegistry  SessionRegistry
	sessionProcessor *sp.Processor
	agentService     AgentEnvironmentSetter
	pairingProvider  PairingDataProvider // optional, nil when bridge not configured
}

// NewConnectionHandler creates a new WebSocket connection handler.
func NewConnectionHandler(
	registry SessionRegistry,
	processor *sp.Processor,
	agentService AgentEnvironmentSetter,
) *ConnectionHandler {
	return &ConnectionHandler{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Only localhost connections (already guaranteed by bind to 127.0.0.1)
				return true
			},
		},
		sessionRegistry:  registry,
		sessionProcessor: processor,
		agentService:     agentService,
	}
}

// SetPairingProvider sets the pairing data provider (called after bridge is initialized).
func (h *ConnectionHandler) SetPairingProvider(p PairingDataProvider) {
	h.pairingProvider = p
}

// connState tracks per-connection state (active subscriptions, context).
type connState struct {
	ctx    context.Context
	cancel context.CancelFunc

	mu              sync.Mutex
	activeCleanups  []func() // subscription cleanups to call on disconnect
}

func (cs *connState) addCleanup(cleanup func()) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.activeCleanups = append(cs.activeCleanups, cleanup)
}

func (cs *connState) cleanupAll() {
	cs.mu.Lock()
	cleanups := cs.activeCleanups
	cs.activeCleanups = nil
	cs.mu.Unlock()
	for _, fn := range cleanups {
		fn()
	}
}

// ServeHTTP upgrades the connection to WebSocket and runs the read loop.
func (h *ConnectionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("[WS] upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	slog.InfoContext(r.Context(), "[WS] client connected", "remote", r.RemoteAddr)

	ctx, cancel := context.WithCancel(context.Background())
	state := &connState{ctx: ctx, cancel: cancel}
	defer func() {
		cancel()
		state.cleanupAll()
	}()

	writer := &wsWriter{conn: conn}

	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				slog.Error("[WS] read error", "error", err)
			}
			return
		}

		var msg WsMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			writer.sendError("", "invalid JSON")
			continue
		}

		h.handleMessage(writer, msg, state)
	}
}

// wsWriter provides thread-safe writing to a WebSocket connection.
type wsWriter struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (w *wsWriter) send(msg *WsMessage) {
	w.mu.Lock()
	defer w.mu.Unlock()

	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("[WS] marshal error", "error", err)
		return
	}
	if err := w.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		slog.Error("[WS] write error", "error", err)
	}
}

func (w *wsWriter) sendError(requestID, errMsg string) {
	w.send(&WsMessage{
		Type:      "error",
		RequestID: requestID,
		Payload:   map[string]interface{}{"error": errMsg},
	})
}

func (h *ConnectionHandler) handleMessage(writer *wsWriter, msg WsMessage, state *connState) {
	switch msg.Type {
	case "ping":
		writer.send(&WsMessage{
			Type:      "pong",
			RequestID: msg.RequestID,
			Payload:   map[string]interface{}{"timestamp": time.Now().UnixMilli()},
		})

	case "create_session":
		h.handleCreateSession(writer, &msg)

	case "send_message":
		h.handleSendMessage(writer, &msg, state)

	case "subscribe":
		h.handleSubscribe(writer, &msg, state)

	case "ask_user_reply":
		h.handleAskUserReply(writer, &msg)

	case "cancel_session":
		h.handleCancelSession(writer, &msg)

	case "generate_pairing":
		h.handleGeneratePairing(writer, &msg)

	default:
		writer.sendError(msg.RequestID, "unknown message type: "+msg.Type)
	}
}

func (h *ConnectionHandler) handleCreateSession(writer *wsWriter, msg *WsMessage) {
	projectRoot, _ := msg.Payload["project_root"].(string)
	platform, _ := msg.Payload["platform"].(string)
	projectKey, _ := msg.Payload["project_key"].(string)

	if projectRoot != "" || platform != "" {
		h.agentService.SetEnvironmentContext(projectRoot, platform)
	}

	sessionID := uuid.New().String()
	h.sessionRegistry.CreateSession(sessionID, projectKey, "", projectRoot, platform)

	slog.Info("[WS] session created", "session_id", sessionID, "project_root", projectRoot)

	writer.send(&WsMessage{
		Type:      "create_session_ack",
		RequestID: msg.RequestID,
		Payload:   map[string]interface{}{"session_id": sessionID},
	})
}

func (h *ConnectionHandler) handleSendMessage(writer *wsWriter, msg *WsMessage, state *connState) {
	sessionID, _ := msg.Payload["session_id"].(string)
	content, _ := msg.Payload["content"].(string)

	if sessionID == "" || content == "" {
		writer.sendError(msg.RequestID, "session_id and content required")
		return
	}

	if !h.sessionRegistry.HasSession(sessionID) {
		writer.sendError(msg.RequestID, "session not found")
		return
	}

	if err := h.sessionRegistry.EnqueueMessage(sessionID, content); err != nil {
		writer.sendError(msg.RequestID, err.Error())
		return
	}

	// Start processing (idempotent) — uses connection context so goroutine stops on disconnect
	h.sessionProcessor.StartProcessing(state.ctx, sessionID)

	writer.send(&WsMessage{
		Type:      "send_message_ack",
		RequestID: msg.RequestID,
		Payload:   map[string]interface{}{"session_id": sessionID},
	})
}

// handleSubscribe subscribes to session events in a background goroutine.
// The read loop continues so the client can send ask_user_reply, cancel_session, etc.
// while subscribed.
func (h *ConnectionHandler) handleSubscribe(writer *wsWriter, msg *WsMessage, state *connState) {
	sessionID, _ := msg.Payload["session_id"].(string)
	lastEventID, _ := msg.Payload["last_event_id"].(string)

	if sessionID == "" {
		writer.sendError(msg.RequestID, "session_id required")
		return
	}

	if !h.sessionRegistry.HasSession(sessionID) {
		writer.sendError(msg.RequestID, "session not found")
		return
	}

	// Clean up previous subscriptions (prevents goroutine leak on reconnect/re-subscribe)
	state.cleanupAll()

	// Send ack first
	writer.send(&WsMessage{
		Type:      "subscribe_ack",
		RequestID: msg.RequestID,
		Payload:   map[string]interface{}{"session_id": sessionID},
	})

	// Replay missed events
	if lastEventID != "" {
		missed := h.sessionRegistry.ReplayEvents(sessionID, lastEventID)
		for _, event := range missed {
			h.sendSessionEvent(writer, sessionID, event)
		}
	}

	// Subscribe to live events
	eventCh, cleanup := h.sessionRegistry.Subscribe(sessionID)
	state.addCleanup(cleanup)

	// Start processing (idempotent -- may already be running)
	h.sessionProcessor.StartProcessing(state.ctx, sessionID)

	// Write events in background goroutine so read loop continues.
	// Goroutine exits when eventCh is closed (cleanup) or connection context is cancelled.
	go func() {
		for {
			select {
			case event, ok := <-eventCh:
				if !ok {
					return
				}
				h.sendSessionEvent(writer, sessionID, event)
			case <-state.ctx.Done():
				return
			}
		}
	}()
}

func (h *ConnectionHandler) sendSessionEvent(writer *wsWriter, sessionID string, event *pb.SessionEvent) {
	serialized := serializeEvent(event)
	if serialized == nil {
		return
	}

	writer.send(&WsMessage{
		Type: "session_event",
		Payload: map[string]interface{}{
			"session_id": sessionID,
			"event_id":   event.GetEventId(),
			"event":      serialized,
		},
	})
}

func (h *ConnectionHandler) handleAskUserReply(writer *wsWriter, msg *WsMessage) {
	sessionID, _ := msg.Payload["session_id"].(string)
	callID, _ := msg.Payload["call_id"].(string)
	reply, _ := msg.Payload["reply"].(string)

	if sessionID == "" || callID == "" {
		writer.sendError(msg.RequestID, "session_id and call_id required")
		return
	}

	h.sessionRegistry.SendAskUserReply(sessionID, callID, reply)

	writer.send(&WsMessage{
		Type:      "ask_user_reply_ack",
		RequestID: msg.RequestID,
		Payload:   map[string]interface{}{"session_id": sessionID},
	})
}

func (h *ConnectionHandler) handleCancelSession(writer *wsWriter, msg *WsMessage) {
	sessionID, _ := msg.Payload["session_id"].(string)
	if sessionID == "" {
		writer.sendError(msg.RequestID, "session_id required")
		return
	}

	cancelled := h.sessionRegistry.Cancel(sessionID)
	if cancelled {
		h.sessionRegistry.DrainMessages(sessionID)
	}

	writer.send(&WsMessage{
		Type:      "cancel_session_ack",
		RequestID: msg.RequestID,
		Payload:   map[string]interface{}{"session_id": sessionID, "cancelled": cancelled},
	})
}

func (h *ConnectionHandler) handleGeneratePairing(writer *wsWriter, msg *WsMessage) {
	if h.pairingProvider == nil {
		writer.sendError(msg.RequestID, "bridge not configured")
		return
	}

	data, err := h.pairingProvider.GeneratePairingData()
	if err != nil {
		writer.sendError(msg.RequestID, err.Error())
		return
	}

	writer.send(&WsMessage{
		Type:      "pairing_data",
		RequestID: msg.RequestID,
		Payload:   data,
	})
}

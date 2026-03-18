package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// ChatService handles agent chat sessions via SSE.
type ChatService interface {
	// Chat starts a chat session and streams events.
	// Returns a channel of SSE events and an error.
	Chat(agentName, message, userID, sessionID string) (<-chan SSEEvent, error)
}

// ChatHandler serves POST /api/v1/agents/{name}/chat with SSE streaming.
type ChatHandler struct {
	service ChatService
}

// NewChatHandler creates a new ChatHandler.
func NewChatHandler(service ChatService) *ChatHandler {
	return &ChatHandler{service: service}
}

type chatRequest struct {
	Message   string `json:"message"`
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
}

// Chat handles SSE streaming chat.
func (h *ChatHandler) Chat(w http.ResponseWriter, r *http.Request) {
	agentName := chi.URLParam(r, "name")
	if agentName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "agent name required"})
		return
	}

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message required"})
		return
	}

	events, err := h.service.Chat(agentName, req.Message, req.UserID, req.SessionID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	sse, sseErr := NewSSEWriter(w)
	if sseErr != nil {
		return
	}
	stopHB := sse.StartHeartbeat(15 * time.Second)
	defer stopHB()

	for event := range events {
		sse.WriteEvent(event.Type, event.Data)
	}
}

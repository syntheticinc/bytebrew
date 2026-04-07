package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// AssistantService handles builder assistant messages.
type AssistantService interface {
	HandleMessage(ctx context.Context, sessionID, message string,
		hasSchemas bool, eventStream domain.AgentEventStream) (string, error)
}

// SchemaCounter checks whether any schemas exist.
type SchemaCounter interface {
	HasSchemas(ctx context.Context) (bool, error)
}

// AssistantHandler handles POST /api/v1/admin/assistant/chat.
type AssistantHandler struct {
	assistant AssistantService
	schemas   SchemaCounter
}

// NewAssistantHandler creates a new AssistantHandler.
func NewAssistantHandler(assistant AssistantService, schemas SchemaCounter) *AssistantHandler {
	return &AssistantHandler{
		assistant: assistant,
		schemas:   schemas,
	}
}

type assistantChatRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id"`
}

type assistantChatResponse struct {
	Response  string `json:"response"`
	SessionID string `json:"session_id"`
}

// Chat handles POST /api/v1/admin/assistant/chat
func (h *AssistantHandler) Chat(w http.ResponseWriter, r *http.Request) {
	var req assistantChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Message == "" {
		writeJSONError(w, http.StatusBadRequest, "message is required")
		return
	}

	if req.SessionID == "" {
		req.SessionID = "assistant-default"
	}

	hasSchemas := false
	if h.schemas != nil {
		hs, err := h.schemas.HasSchemas(r.Context())
		if err == nil {
			hasSchemas = hs
		}
	}

	response, err := h.assistant.HandleMessage(r.Context(), req.SessionID, req.Message, hasSchemas, nil)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, assistantChatResponse{
		Response:  response,
		SessionID: req.SessionID,
	})
}

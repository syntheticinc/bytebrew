package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// SessionResponder delivers a reply to a pending ask_user question.
type SessionResponder interface {
	HasSession(sessionID string) bool
	SendAskUserReply(sessionID, callID, reply string)
}

// RespondHandler handles POST /api/v1/sessions/{id}/respond.
type RespondHandler struct {
	responder SessionResponder
}

// NewRespondHandler creates a RespondHandler.
func NewRespondHandler(responder SessionResponder) *RespondHandler {
	return &RespondHandler{responder: responder}
}

type respondRequest struct {
	CallID  string   `json:"call_id"`
	Answers []string `json:"answers"`
}

// Respond handles POST /api/v1/sessions/{id}/respond.
// It delivers the user's answer to a pending ask_user question.
func (h *RespondHandler) Respond(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		writeJSONError(w, http.StatusBadRequest, "session id is required")
		return
	}

	var req respondRequest
	if err := readJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}

	if req.CallID == "" {
		writeJSONError(w, http.StatusBadRequest, "call_id is required")
		return
	}

	if !h.responder.HasSession(sessionID) {
		writeJSONError(w, http.StatusNotFound, fmt.Sprintf("session not found: %s", sessionID))
		return
	}

	// Convert answers to JSON string expected by SendAskUserReply.
	// The ask_user tool expects a JSON array of QuestionAnswer objects,
	// but for simple cases a JSON array of strings also works.
	answersJSON, err := json.Marshal(req.Answers)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("marshal answers: %s", err.Error()))
		return
	}

	h.responder.SendAskUserReply(sessionID, req.CallID, string(answersJSON))

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

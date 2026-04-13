package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// DispatchQueryer provides read access to dispatched task packets.
type DispatchQueryer interface {
	GetTask(taskID string) (*domain.TaskPacket, bool)
	ListTasksBySession(sessionID string) []*domain.TaskPacket
}

// DispatchHandler serves dispatch task query endpoints.
type DispatchHandler struct {
	queryer DispatchQueryer
}

// NewDispatchHandler creates a new DispatchHandler.
func NewDispatchHandler(queryer DispatchQueryer) *DispatchHandler {
	return &DispatchHandler{queryer: queryer}
}

// TaskPacketResponse is the JSON representation of a dispatched task.
type TaskPacketResponse struct {
	ID          string `json:"id"`
	AgentName   string `json:"agent_name"`
	Task        string `json:"task"`
	SessionID   string `json:"session_id"`
	State       string `json:"state"`
	Result      string `json:"result,omitempty"`
	Error       string `json:"error,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// Get handles GET /api/v1/dispatch/tasks/{taskId}.
func (h *DispatchHandler) Get(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskId")
	if taskID == "" {
		http.Error(w, `{"error":"task id required"}`, http.StatusBadRequest)
		return
	}

	packet, ok := h.queryer.GetTask(taskID)
	if !ok {
		http.Error(w, `{"error":"task not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toTaskPacketResponse(packet))
}

// ListBySession handles GET /api/v1/sessions/{sessionId}/dispatch-tasks.
func (h *DispatchHandler) ListBySession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		http.Error(w, `{"error":"session id required"}`, http.StatusBadRequest)
		return
	}

	packets := h.queryer.ListTasksBySession(sessionID)

	responses := make([]TaskPacketResponse, 0, len(packets))
	for _, p := range packets {
		responses = append(responses, toTaskPacketResponse(p))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(responses)
}

func toTaskPacketResponse(p *domain.TaskPacket) TaskPacketResponse {
	updatedAt := p.CreatedAt
	if !p.FinishedAt.IsZero() {
		updatedAt = p.FinishedAt
	} else if !p.StartedAt.IsZero() {
		updatedAt = p.StartedAt
	}

	return TaskPacketResponse{
		ID:        p.ID,
		AgentName: p.ChildAgent,
		Task:      p.Input,
		SessionID: p.SessionID,
		State:     string(p.Status),
		Result:    p.Result,
		Error:     p.Error,
		CreatedAt: p.CreatedAt.Format(time.RFC3339),
		UpdatedAt: updatedAt.Format(time.RFC3339),
	}
}

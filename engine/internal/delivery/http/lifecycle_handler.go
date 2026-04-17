package http

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// LifecycleStatus holds the lifecycle status response for an agent.
type LifecycleStatus struct {
	Mode          string `json:"mode"`
	State         string `json:"state"`
	TasksHandled  int    `json:"tasks_handled"`
	ContextTokens int    `json:"context_tokens"`
	MaxContext    int    `json:"max_context"`
}

// LifecycleProvider retrieves lifecycle status for an agent.
type LifecycleProvider interface {
	GetLifecycleStatus(ctx context.Context, agentName, sessionID string) (*LifecycleStatus, error)
}

// LifecycleHandler serves /api/v1/agents/{name}/lifecycle endpoints.
type LifecycleHandler struct {
	provider LifecycleProvider
}

// NewLifecycleHandler creates a new LifecycleHandler.
func NewLifecycleHandler(provider LifecycleProvider) *LifecycleHandler {
	return &LifecycleHandler{provider: provider}
}

// Status handles GET /api/v1/agents/{name}/lifecycle.
func (h *LifecycleHandler) Status(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "agent name is required")
		return
	}

	sessionID := r.URL.Query().Get("session_id")

	status, err := h.provider.GetLifecycleStatus(r.Context(), name, sessionID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if status == nil {
		writeJSONError(w, http.StatusNotFound, "no lifecycle instance for agent: "+name)
		return
	}

	writeJSON(w, http.StatusOK, status)
}

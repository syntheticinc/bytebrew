package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
)

// BuilderAssistantRestorer defines what is needed to restore builder-assistant.
type BuilderAssistantRestorer interface {
	RestoreBuilderAssistant(ctx context.Context) error
}

// BuilderAssistantHandler serves /api/v1/admin/builder-assistant endpoints.
type BuilderAssistantHandler struct {
	restorer BuilderAssistantRestorer
}

// NewBuilderAssistantHandler creates a new BuilderAssistantHandler.
func NewBuilderAssistantHandler(restorer BuilderAssistantRestorer) *BuilderAssistantHandler {
	return &BuilderAssistantHandler{restorer: restorer}
}

// Restore handles POST /api/v1/admin/builder-assistant/restore.
// Idempotently resets builder-assistant to factory defaults.
func (h *BuilderAssistantHandler) Restore(w http.ResponseWriter, r *http.Request) {
	if err := h.restorer.RestoreBuilderAssistant(r.Context()); err != nil {
		slog.ErrorContext(r.Context(), "failed to restore builder-assistant", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to restore builder-assistant"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "builder-assistant restored to factory defaults"})
}

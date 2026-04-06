package http

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// MemoryLister lists memories for a schema (AC-MEM-03).
type MemoryLister interface {
	Execute(ctx context.Context, schemaID string) ([]*domain.Memory, error)
}

// MemoryClearer clears memories for a schema (AC-MEM-03).
type MemoryClearer interface {
	ClearAll(ctx context.Context, schemaID string) (int64, error)
	DeleteOne(ctx context.Context, id string) error
}

// MemoryHandler handles memory-related HTTP endpoints.
type MemoryHandler struct {
	lister  MemoryLister
	clearer MemoryClearer
}

// NewMemoryHandler creates a new MemoryHandler.
func NewMemoryHandler(lister MemoryLister, clearer MemoryClearer) *MemoryHandler {
	return &MemoryHandler{
		lister:  lister,
		clearer: clearer,
	}
}

// memoryResponse represents a single memory entry in the API response.
type memoryResponse struct {
	ID        string            `json:"id"`
	SchemaID  string            `json:"schema_id"`
	UserID    string            `json:"user_id"`
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

// ListMemories handles GET /api/v1/schemas/{id}/memory
func (h *MemoryHandler) ListMemories(w http.ResponseWriter, r *http.Request) {
	schemaID := chi.URLParam(r, "id")
	if schemaID == "" {
		writeJSONError(w, http.StatusBadRequest, "schema id required")
		return
	}

	memories, err := h.lister.Execute(r.Context(), schemaID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	resp := make([]memoryResponse, 0, len(memories))
	for _, m := range memories {
		resp = append(resp, memoryResponse{
			ID:        m.ID,
			SchemaID:  m.SchemaID,
			UserID:    m.UserID,
			Content:   m.Content,
			Metadata:  m.Metadata,
			CreatedAt: m.CreatedAt,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// ClearMemories handles DELETE /api/v1/schemas/{id}/memory
func (h *MemoryHandler) ClearMemories(w http.ResponseWriter, r *http.Request) {
	schemaID := chi.URLParam(r, "id")
	if schemaID == "" {
		writeJSONError(w, http.StatusBadRequest, "schema id required")
		return
	}

	deleted, err := h.clearer.ClearAll(r.Context(), schemaID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"deleted": deleted,
	})
}

// DeleteMemory handles DELETE /api/v1/schemas/{id}/memory/{entry_id}
func (h *MemoryHandler) DeleteMemory(w http.ResponseWriter, r *http.Request) {
	entryID := chi.URLParam(r, "entry_id")
	if entryID == "" {
		writeJSONError(w, http.StatusBadRequest, "memory entry id required")
		return
	}

	if err := h.clearer.DeleteOne(r.Context(), entryID); err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

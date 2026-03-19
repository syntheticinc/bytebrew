package http

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// KnowledgeStats provides knowledge base statistics for an agent.
type KnowledgeStats interface {
	GetStats(ctx context.Context, agentName string) (docCount int, chunkCount int, lastIndexed *time.Time, err error)
}

// KnowledgeReindexer triggers re-indexing for an agent's knowledge base.
type KnowledgeReindexer interface {
	Reindex(ctx context.Context, agentName string) error
}

// KnowledgeHandler serves /api/v1/agents/{name}/knowledge endpoints.
type KnowledgeHandler struct {
	stats     KnowledgeStats
	reindexer KnowledgeReindexer
}

// NewKnowledgeHandler creates a KnowledgeHandler.
func NewKnowledgeHandler(stats KnowledgeStats, reindexer KnowledgeReindexer) *KnowledgeHandler {
	return &KnowledgeHandler{
		stats:     stats,
		reindexer: reindexer,
	}
}

// knowledgeStatusResponse is the JSON response for GET .../knowledge/status.
type knowledgeStatusResponse struct {
	Agent       string     `json:"agent"`
	Documents   int        `json:"documents"`
	Chunks      int        `json:"chunks"`
	LastIndexed *time.Time `json:"last_indexed,omitempty"`
}

// Status handles GET /api/v1/agents/{name}/knowledge/status.
func (h *KnowledgeHandler) Status(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "agent name is required")
		return
	}

	docCount, chunkCount, lastIndexed, err := h.stats.GetStats(r.Context(), name)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, knowledgeStatusResponse{
		Agent:       name,
		Documents:   docCount,
		Chunks:      chunkCount,
		LastIndexed: lastIndexed,
	})
}

// Reindex handles POST /api/v1/agents/{name}/knowledge/reindex.
func (h *KnowledgeHandler) Reindex(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "agent name is required")
		return
	}

	if h.reindexer == nil {
		writeJSONError(w, http.StatusNotImplemented, "knowledge reindexing not available")
		return
	}

	// Launch async reindex
	go func() {
		ctx := context.Background()
		if err := h.reindexer.Reindex(ctx, name); err != nil {
			// Logged inside reindexer; nothing more to do here.
			_ = err
		}
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"status": "indexing_started",
	})
}

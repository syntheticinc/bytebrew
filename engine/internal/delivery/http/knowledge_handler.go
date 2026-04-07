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

// KnowledgeFileLister lists knowledge files for an agent (AC-KB-LIST-01..05).
type KnowledgeFileLister interface {
	ListFiles(ctx context.Context, agentName string) ([]KnowledgeFileResponse, error)
	DeleteFile(ctx context.Context, agentName, fileID string) error
	ReindexFile(ctx context.Context, agentName, fileID string) error
}

// KnowledgeFileResponse represents a knowledge file in the API response (AC-KB-LIST-02).
type KnowledgeFileResponse struct {
	ID         string `json:"id"`
	FileName   string `json:"file_name"`
	FileType   string `json:"file_type"`
	FileSize   int64  `json:"file_size"`
	Status     string `json:"status"` // uploading, indexing, ready, error
	StatusMsg  string `json:"status_message,omitempty"`
	ChunkCount int    `json:"chunk_count"`
	CreatedAt  string `json:"created_at"`
	IndexedAt  string `json:"indexed_at,omitempty"`
}

// KnowledgeHandler serves /api/v1/agents/{name}/knowledge endpoints.
type KnowledgeHandler struct {
	stats      KnowledgeStats
	reindexer  KnowledgeReindexer
	fileLister KnowledgeFileLister
}

// NewKnowledgeHandler creates a KnowledgeHandler.
func NewKnowledgeHandler(stats KnowledgeStats, reindexer KnowledgeReindexer) *KnowledgeHandler {
	return &KnowledgeHandler{
		stats:     stats,
		reindexer: reindexer,
	}
}

// SetFileLister sets the file lister (optional, may not be wired in all deployments).
func (h *KnowledgeHandler) SetFileLister(lister KnowledgeFileLister) {
	h.fileLister = lister
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

// ListFiles handles GET /api/v1/agents/{name}/knowledge (AC-KB-LIST-01..02).
func (h *KnowledgeHandler) ListFiles(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "agent name is required")
		return
	}

	if h.fileLister == nil {
		writeJSONError(w, http.StatusNotImplemented, "file listing not available")
		return
	}

	files, err := h.fileLister.ListFiles(r.Context(), name)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if files == nil {
		files = []KnowledgeFileResponse{}
	}

	writeJSON(w, http.StatusOK, files)
}

// DeleteFile handles DELETE /api/v1/agents/{name}/knowledge/{file_id} (AC-KB-LIST-04).
func (h *KnowledgeHandler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	fileID := chi.URLParam(r, "file_id")
	if name == "" || fileID == "" {
		writeJSONError(w, http.StatusBadRequest, "agent name and file_id are required")
		return
	}

	if h.fileLister == nil {
		writeJSONError(w, http.StatusNotImplemented, "file management not available")
		return
	}

	if err := h.fileLister.DeleteFile(r.Context(), name, fileID); err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ReindexFile handles POST /api/v1/agents/{name}/knowledge/{file_id}/reindex (AC-KB-LIST-05).
func (h *KnowledgeHandler) ReindexFile(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	fileID := chi.URLParam(r, "file_id")
	if name == "" || fileID == "" {
		writeJSONError(w, http.StatusBadRequest, "agent name and file_id are required")
		return
	}

	if h.fileLister == nil {
		writeJSONError(w, http.StatusNotImplemented, "file management not available")
		return
	}

	if err := h.fileLister.ReindexFile(r.Context(), name, fileID); err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "reindex_started"})
}

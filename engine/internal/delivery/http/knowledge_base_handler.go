package http

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// KnowledgeBaseInfo is the API response for a knowledge base.
type KnowledgeBaseInfo struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Description      string   `json:"description,omitempty"`
	EmbeddingModelID string   `json:"embedding_model_id,omitempty"`
	FileCount        int      `json:"file_count"`
	LinkedAgents     []string `json:"linked_agents"` // agent names
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

// CreateKBRequest is the request body for creating a knowledge base.
type CreateKBRequest struct {
	Name             string `json:"name"`
	Description      string `json:"description,omitempty"`
	EmbeddingModelID string `json:"embedding_model_id"`
}

// UpdateKBRequest is the request body for updating a knowledge base.
type UpdateKBRequest struct {
	Name             string `json:"name"`
	Description      string `json:"description,omitempty"`
	EmbeddingModelID string `json:"embedding_model_id"`
}

// KBStore provides CRUD for knowledge bases.
type KBStore interface {
	Create(ctx context.Context, name, description, embeddingModelID, tenantID string) (*KnowledgeBaseInfo, error)
	Update(ctx context.Context, id, name, description, embeddingModelID string) (*KnowledgeBaseInfo, error)
	GetByID(ctx context.Context, id string) (*KnowledgeBaseInfo, error)
	List(ctx context.Context) ([]KnowledgeBaseInfo, error)
	Delete(ctx context.Context, id string) error
	LinkAgent(ctx context.Context, kbID, agentName string) error
	UnlinkAgent(ctx context.Context, kbID, agentName string) error
}

// KBFileManager provides file operations on a knowledge base.
type KBFileManager interface {
	ListFiles(ctx context.Context, kbID string) ([]KnowledgeFileResponse, error)
	UploadFile(ctx context.Context, tenantID, kbID, embeddingModelID, fileName, fileType string, fileSize int64, fileHash string, content []byte) (*KnowledgeFileResponse, error)
	DeleteFile(ctx context.Context, kbID, fileID string) error
	ReindexFile(ctx context.Context, kbID, embeddingModelID, fileID string) error
	DeleteAllFiles(ctx context.Context, kbID string) error
}

// KnowledgeBaseHandler serves /api/v1/knowledge-bases endpoints.
type KnowledgeBaseHandler struct {
	store       KBStore
	fileManager KBFileManager
}

// NewKnowledgeBaseHandler creates a new handler.
func NewKnowledgeBaseHandler(store KBStore, fileManager KBFileManager) *KnowledgeBaseHandler {
	return &KnowledgeBaseHandler{store: store, fileManager: fileManager}
}

// List handles GET /api/v1/knowledge-bases.
func (h *KnowledgeBaseHandler) List(w http.ResponseWriter, r *http.Request) {
	kbs, err := h.store.List(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if kbs == nil {
		kbs = []KnowledgeBaseInfo{}
	}
	writeJSON(w, http.StatusOK, kbs)
}

// Get handles GET /api/v1/knowledge-bases/{id}.
func (h *KnowledgeBaseHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	kb, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if kb == nil {
		writeJSONError(w, http.StatusNotFound, "knowledge base not found")
		return
	}
	writeJSON(w, http.StatusOK, kb)
}

// Create handles POST /api/v1/knowledge-bases.
func (h *KnowledgeBaseHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateKBRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}

	tenantID := domain.TenantIDFromContext(r.Context())
	if tenantID == "" {
		tenantID = domain.CETenantID
	}

	kb, err := h.store.Create(r.Context(), req.Name, req.Description, req.EmbeddingModelID, tenantID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, kb)
}

// Update handles PUT /api/v1/knowledge-bases/{id}.
func (h *KnowledgeBaseHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req UpdateKBRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}

	kb, err := h.store.Update(r.Context(), id, req.Name, req.Description, req.EmbeddingModelID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if kb == nil {
		writeJSONError(w, http.StatusNotFound, "knowledge base not found")
		return
	}
	writeJSON(w, http.StatusOK, kb)
}

// Delete handles DELETE /api/v1/knowledge-bases/{id}.
func (h *KnowledgeBaseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Delete all files first
	if h.fileManager != nil {
		if err := h.fileManager.DeleteAllFiles(r.Context(), id); err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// LinkAgent handles POST /api/v1/knowledge-bases/{id}/agents/{agent_name}.
func (h *KnowledgeBaseHandler) LinkAgent(w http.ResponseWriter, r *http.Request) {
	kbID := chi.URLParam(r, "id")
	agentName := chi.URLParam(r, "agent_name")
	if kbID == "" || agentName == "" {
		writeJSONError(w, http.StatusBadRequest, "kb id and agent_name are required")
		return
	}
	if err := h.store.LinkAgent(r.Context(), kbID, agentName); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "linked"})
}

// UnlinkAgent handles DELETE /api/v1/knowledge-bases/{id}/agents/{agent_name}.
func (h *KnowledgeBaseHandler) UnlinkAgent(w http.ResponseWriter, r *http.Request) {
	kbID := chi.URLParam(r, "id")
	agentName := chi.URLParam(r, "agent_name")
	if kbID == "" || agentName == "" {
		writeJSONError(w, http.StatusBadRequest, "kb id and agent_name are required")
		return
	}
	if err := h.store.UnlinkAgent(r.Context(), kbID, agentName); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "unlinked"})
}

// ListFiles handles GET /api/v1/knowledge-bases/{id}/files.
func (h *KnowledgeBaseHandler) ListFiles(w http.ResponseWriter, r *http.Request) {
	kbID := chi.URLParam(r, "id")
	if h.fileManager == nil {
		writeJSONError(w, http.StatusNotImplemented, "Knowledge indexing requires an embedding model. Configure one in Models → select type Embeddings.")
		return
	}
	files, err := h.fileManager.ListFiles(r.Context(), kbID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if files == nil {
		files = []KnowledgeFileResponse{}
	}
	writeJSON(w, http.StatusOK, files)
}

// UploadFile handles POST /api/v1/knowledge-bases/{id}/files.
func (h *KnowledgeBaseHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	kbID := chi.URLParam(r, "id")
	if h.fileManager == nil {
		writeJSONError(w, http.StatusNotImplemented, "Knowledge indexing requires an embedding model. Configure one in Models → select type Embeddings.")
		return
	}

	// Resolve KB to get embedding model ID
	kb, err := h.store.GetByID(r.Context(), kbID)
	if err != nil || kb == nil {
		writeJSONError(w, http.StatusNotFound, "knowledge base not found")
		return
	}
	if kb.EmbeddingModelID == "" {
		writeJSONError(w, http.StatusBadRequest, "no embedding model configured for this knowledge base")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize+1024)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeJSONError(w, http.StatusBadRequest, "file too large or invalid multipart form (max 50MB)")
		return
	}
	defer func() { _ = r.MultipartForm.RemoveAll() }()

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "file field is required")
		return
	}
	defer func() { _ = file.Close() }()

	originalName := sanitizeUploadFilename(header.Filename)
	if originalName == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid filename")
		return
	}

	ext := strings.ToLower(filepath.Ext(originalName))
	allowedMIME, ok := allowedMIMETypes[ext]
	if !ok {
		writeJSONError(w, http.StatusBadRequest,
			fmt.Sprintf("unsupported file type %q, supported: txt, md, csv, pdf, docx", ext))
		return
	}

	if !validateMIME(header, allowedMIME) {
		writeJSONError(w, http.StatusBadRequest,
			fmt.Sprintf("file content type does not match extension %q", ext))
		return
	}

	content, err := io.ReadAll(file)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to read file")
		return
	}
	if len(content) == 0 {
		writeJSONError(w, http.StatusBadRequest, "empty file")
		return
	}

	fileHash := fmt.Sprintf("%x", sha256.Sum256(content))
	fileType := strings.TrimPrefix(ext, ".")

	tenantID := domain.TenantIDFromContext(r.Context())
	if tenantID == "" {
		tenantID = domain.CETenantID
	}

	resp, err := h.fileManager.UploadFile(r.Context(), tenantID, kbID, kb.EmbeddingModelID, originalName, fileType, int64(len(content)), fileHash, content)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// DeleteFile handles DELETE /api/v1/knowledge-bases/{id}/files/{file_id}.
func (h *KnowledgeBaseHandler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	kbID := chi.URLParam(r, "id")
	fileID := chi.URLParam(r, "file_id")
	if h.fileManager == nil {
		writeJSONError(w, http.StatusNotImplemented, "Knowledge indexing requires an embedding model. Configure one in Models → select type Embeddings.")
		return
	}
	if err := h.fileManager.DeleteFile(r.Context(), kbID, fileID); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ReindexFile handles POST /api/v1/knowledge-bases/{id}/files/{file_id}/reindex.
func (h *KnowledgeBaseHandler) ReindexFile(w http.ResponseWriter, r *http.Request) {
	kbID := chi.URLParam(r, "id")
	fileID := chi.URLParam(r, "file_id")
	if h.fileManager == nil {
		writeJSONError(w, http.StatusNotImplemented, "Knowledge indexing requires an embedding model. Configure one in Models → select type Embeddings.")
		return
	}

	kb, err := h.store.GetByID(r.Context(), kbID)
	if err != nil || kb == nil {
		writeJSONError(w, http.StatusNotFound, "knowledge base not found")
		return
	}

	if err := h.fileManager.ReindexFile(r.Context(), kbID, kb.EmbeddingModelID, fileID); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "reindex_started"})
}


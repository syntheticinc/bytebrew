package knowledge

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/indexing"
	infknowledge "github.com/syntheticinc/bytebrew/engine/internal/infrastructure/knowledge"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
)

// tenantFromCtx extracts tenant_id from context, falling back to "default" for CE mode.
func tenantFromCtx(ctx context.Context) string {
	tid := domain.TenantIDFromContext(ctx)
	if tid == "" {
		return "default"
	}
	return tid
}

// DocumentRepository persists knowledge documents and chunks.
type DocumentRepository interface {
	SaveDocument(ctx context.Context, doc *models.KnowledgeDocument) error
	SaveChunks(ctx context.Context, chunks []models.KnowledgeChunk) error
	DeleteChunksByDocument(ctx context.Context, documentID string) error
	DeleteDocument(ctx context.Context, id string) error
	GetDocumentByID(ctx context.Context, id string) (*models.KnowledgeDocument, error)
	ListDocumentsByAgent(ctx context.Context, agentName string) ([]models.KnowledgeDocument, error)
}

// EmbeddingProvider generates vector embeddings for text.
type EmbeddingProvider interface {
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}

// EmbeddingModelInfo holds embedding model details resolved from DB.
type EmbeddingModelInfo struct {
	BaseURL      string
	APIKey       string
	ModelName    string
	EmbeddingDim int
}

// EmbeddingModelResolver resolves the embedding model for an agent's knowledge capability.
type EmbeddingModelResolver interface {
	ResolveEmbeddingModel(ctx context.Context, agentName string) (*EmbeddingModelInfo, error)
}

// FileResponse is the API response for a knowledge file.
type FileResponse struct {
	ID         string `json:"id"`
	FileName   string `json:"file_name"`
	FileType   string `json:"file_type"`
	FileSize   int64  `json:"file_size"`
	Status     string `json:"status"`
	StatusMsg  string `json:"status_message,omitempty"`
	ChunkCount int    `json:"chunk_count"`
	CreatedAt  string `json:"created_at"`
	IndexedAt  string `json:"indexed_at,omitempty"`
}

// UploadService handles file uploads, storage, and async indexing.
type UploadService struct {
	repo              DocumentRepository
	embeddingResolver EmbeddingModelResolver // resolves embedding model from capability config (may be nil)
	dataDir           string
}

// NewUploadService creates a new knowledge upload service.
func NewUploadService(repo DocumentRepository, dataDir string) *UploadService {
	return &UploadService{
		repo:    repo,
		dataDir: dataDir,
	}
}

// SetEmbeddingResolver sets the resolver for capability-based embedding models.
func (s *UploadService) SetEmbeddingResolver(resolver EmbeddingModelResolver) {
	s.embeddingResolver = resolver
}

// UploadFile stores a file on disk, creates a DB record, and triggers async indexing.
func (s *UploadService) UploadFile(ctx context.Context, tenantID, agentName, fileName, fileType string, fileSize int64, fileHash string, content []byte) (*FileResponse, error) {
	// Create storage directory: data/knowledge/{tenant_id}/{agent_name}/
	dir := filepath.Join(s.dataDir, "knowledge", tenantID, agentName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create knowledge directory: %w", err)
	}

	// Generate unique filename to prevent collisions
	docID := uuid.New().String()
	storedName := docID + "_" + fileName
	filePath := filepath.Join(dir, storedName)

	// Write file to disk
	if err := os.WriteFile(filePath, content, 0o644); err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}

	// Create DB record
	doc := &models.KnowledgeDocument{
		ID:        docID,
		TenantID:  tenantID,
		AgentName: agentName,
		FilePath:  filePath,
		FileName:  fileName,
		FileType:  fileType,
		FileSize:  fileSize,
		FileHash:  fileHash,
		Status:    "indexing",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.SaveDocument(ctx, doc); err != nil {
		_ = os.Remove(filePath) // cleanup on failure
		return nil, fmt.Errorf("save document record: %w", err)
	}

	// Trigger async indexing
	go s.indexFileAsync(docID, tenantID, agentName, fileName, string(content))

	return &FileResponse{
		ID:        docID,
		FileName:  fileName,
		FileType:  fileType,
		FileSize:  fileSize,
		Status:    "indexing",
		CreatedAt: doc.CreatedAt.Format(time.RFC3339),
	}, nil
}

// resolveEmbeddingProvider picks the embedding provider for this agent from capability config.
func (s *UploadService) resolveEmbeddingProvider(ctx context.Context, agentName string) (EmbeddingProvider, error) {
	if s.embeddingResolver != nil {
		info, err := s.embeddingResolver.ResolveEmbeddingModel(ctx, agentName)
		if err == nil && info != nil {
			slog.InfoContext(ctx, "[KnowledgeUpload] using configured embedding model",
				"agent", agentName, "model", info.ModelName, "dim", info.EmbeddingDim)
			return indexing.NewOpenAIEmbeddingsClient(info.BaseURL, info.APIKey, info.ModelName, info.EmbeddingDim), nil
		}
	}
	return nil, fmt.Errorf("no embedding model configured for agent %q: add an embedding model in Settings > Models and select it in the Knowledge capability config", agentName)
}

// indexFileAsync chunks, embeds, and stores vector data for an uploaded file.
func (s *UploadService) indexFileAsync(docID, tenantID, agentName, fileName, content string) {
	ctx := context.Background()

	// Extract text from binary formats (PDF, DOCX) before chunking.
	fileType := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), ".")
	text, extractErr := infknowledge.ExtractText([]byte(content), fileType)
	if extractErr != nil {
		slog.ErrorContext(ctx, "[KnowledgeUpload] text extraction failed",
			"doc_id", docID, "file", fileName, "error", extractErr)
		s.updateDocStatus(ctx, docID, "error", fmt.Sprintf("text extraction failed: %v", extractErr), 0)
		return
	}

	// Chunk the extracted text
	chunker := infknowledge.ChunkerForFile(fileName)
	chunks := chunker.Chunk(text)

	if len(chunks) == 0 {
		s.updateDocStatus(ctx, docID, "ready", "", 0)
		return
	}

	// Resolve embedding provider (capability model or Ollama fallback)
	embedder, err := s.resolveEmbeddingProvider(ctx, agentName)
	if err != nil {
		slog.ErrorContext(ctx, "[KnowledgeUpload] no embedding provider available",
			"doc_id", docID, "agent", agentName, "error", err)
		s.updateDocStatus(ctx, docID, "error", err.Error(), 0)
		return
	}

	// Embed
	texts := make([]string, len(chunks))
	for i, c := range chunks {
		texts[i] = c.Content
	}

	embeddings, err := embedder.EmbedBatch(ctx, texts)
	if err != nil {
		slog.ErrorContext(ctx, "[KnowledgeUpload] embedding failed",
			"doc_id", docID, "agent", agentName, "error", err)
		s.updateDocStatus(ctx, docID, "error", fmt.Sprintf("embedding failed: %v", err), 0)
		return
	}

	// Save chunks
	chunkModels := make([]models.KnowledgeChunk, 0, len(chunks))
	for i, c := range chunks {
		if i >= len(embeddings) || embeddings[i] == nil {
			continue
		}
		chunkModels = append(chunkModels, models.KnowledgeChunk{
			ID:         uuid.New().String(),
			DocumentID: docID,
			TenantID:   tenantID,
			AgentName:  agentName,
			Content:    c.Content,
			ChunkOrder: c.Order,
			Embedding:  pgvector.NewVector(embeddings[i]),
		})
	}

	// BUG-011: If chunking produced content but all embeddings are nil/empty,
	// the embedding provider is likely unavailable. Mark as error, not ready.
	if len(chunkModels) == 0 && len(chunks) > 0 {
		slog.ErrorContext(ctx, "[KnowledgeUpload] no embeddings generated for any chunk",
			"doc_id", docID, "agent", agentName, "chunks_input", len(chunks))
		s.updateDocStatus(ctx, docID, "error",
			"no embeddings generated (embedding provider may be unavailable)", 0)
		return
	}

	if len(chunkModels) > 0 {
		if err := s.repo.SaveChunks(ctx, chunkModels); err != nil {
			slog.ErrorContext(ctx, "[KnowledgeUpload] save chunks failed",
				"doc_id", docID, "error", err)
			s.updateDocStatus(ctx, docID, "error", fmt.Sprintf("save chunks failed: %v", err), 0)
			return
		}
	}

	s.updateDocStatus(ctx, docID, "ready", "", len(chunkModels))
	slog.InfoContext(ctx, "[KnowledgeUpload] indexing complete",
		"doc_id", docID, "agent", agentName, "chunks", len(chunkModels))
}

// updateDocStatus updates a document's status, status message, and chunk count.
func (s *UploadService) updateDocStatus(ctx context.Context, docID, status, statusMsg string, chunkCount int) {
	doc, err := s.repo.GetDocumentByID(ctx, docID)
	if err != nil || doc == nil {
		slog.ErrorContext(ctx, "[KnowledgeUpload] failed to find doc for status update",
			"doc_id", docID, "error", err)
		return
	}
	doc.Status = status
	doc.StatusMsg = statusMsg
	doc.ChunkCount = chunkCount
	doc.UpdatedAt = time.Now()
	if status == "ready" {
		doc.IndexedAt = time.Now()
	}
	if err := s.repo.SaveDocument(ctx, doc); err != nil {
		slog.ErrorContext(ctx, "[KnowledgeUpload] failed to update doc status",
			"doc_id", docID, "error", err)
	}
}

// ListFiles returns knowledge files for an agent (tenant-scoped).
func (s *UploadService) ListFiles(ctx context.Context, agentName string) ([]FileResponse, error) {
	docs, err := s.repo.ListDocumentsByAgent(ctx, agentName)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}

	files := make([]FileResponse, 0, len(docs))
	for _, doc := range docs {
		f := FileResponse{
			ID:         doc.ID,
			FileName:   doc.FileName,
			FileType:   doc.FileType,
			FileSize:   doc.FileSize,
			Status:     doc.Status,
			StatusMsg:  doc.StatusMsg,
			ChunkCount: doc.ChunkCount,
			CreatedAt:  doc.CreatedAt.Format(time.RFC3339),
		}
		if !doc.IndexedAt.IsZero() {
			f.IndexedAt = doc.IndexedAt.Format(time.RFC3339)
		}
		files = append(files, f)
	}
	return files, nil
}

// DeleteFile removes a file, its chunks, and the physical file.
// Verifies ownership (agent_name + tenant_id) before deletion to prevent cross-tenant access.
func (s *UploadService) DeleteFile(ctx context.Context, agentName, fileID string) error {
	doc, err := s.repo.GetDocumentByID(ctx, fileID)
	if err != nil {
		return fmt.Errorf("get document: %w", err)
	}
	if doc == nil || doc.AgentName != agentName {
		return fmt.Errorf("file not found")
	}
	// SCC-02: verify tenant ownership
	tenantID := tenantFromCtx(ctx)
	if doc.TenantID != tenantID {
		return fmt.Errorf("file not found")
	}

	// Delete chunks
	if err := s.repo.DeleteChunksByDocument(ctx, fileID); err != nil {
		return fmt.Errorf("delete chunks: %w", err)
	}

	// Delete physical file
	if doc.FilePath != "" {
		_ = os.Remove(doc.FilePath)
	}

	// Delete document record
	if err := s.repo.DeleteDocument(ctx, fileID); err != nil {
		return fmt.Errorf("delete document: %w", err)
	}

	return nil
}

// ReindexFile re-indexes a single file by deleting old chunks and re-chunking.
// Verifies ownership before re-indexing.
func (s *UploadService) ReindexFile(ctx context.Context, agentName, fileID string) error {
	doc, err := s.repo.GetDocumentByID(ctx, fileID)
	if err != nil || doc == nil || doc.AgentName != agentName {
		return fmt.Errorf("file not found")
	}
	// SCC-02: verify tenant ownership
	tenantID := tenantFromCtx(ctx)
	if doc.TenantID != tenantID {
		return fmt.Errorf("file not found")
	}

	// Read content from disk
	content, err := os.ReadFile(doc.FilePath)
	if err != nil {
		s.updateDocStatus(ctx, fileID, "error", fmt.Sprintf("read file failed: %v", err), 0)
		return fmt.Errorf("read file: %w", err)
	}

	// Delete old chunks
	if err := s.repo.DeleteChunksByDocument(ctx, fileID); err != nil {
		return fmt.Errorf("delete old chunks: %w", err)
	}

	s.updateDocStatus(ctx, fileID, "indexing", "", 0)

	// Async re-index
	go s.indexFileAsync(fileID, doc.TenantID, agentName, doc.FileName, string(content))

	return nil
}

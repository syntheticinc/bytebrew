package config_repo

import (
	"context"
	"fmt"
	"time"

	"github.com/pgvector/pgvector-go"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// GORMKnowledgeRepository provides CRUD and vector search for knowledge documents and chunks.
type GORMKnowledgeRepository struct {
	db *gorm.DB
}

// NewGORMKnowledgeRepository creates a new GORMKnowledgeRepository.
func NewGORMKnowledgeRepository(db *gorm.DB) *GORMKnowledgeRepository {
	return &GORMKnowledgeRepository{db: db}
}

// GetDocumentByPath returns a document by agent name and file path, or nil if not found.
func (r *GORMKnowledgeRepository) GetDocumentByPath(ctx context.Context, agentName, filePath string) (*models.KnowledgeDocument, error) {
	var doc models.KnowledgeDocument
	err := r.db.WithContext(ctx).
		Where("agent_name = ? AND file_path = ?", agentName, filePath).
		First(&doc).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get document by path: %w", err)
	}
	return &doc, nil
}

// SaveDocument creates or updates a knowledge document.
func (r *GORMKnowledgeRepository) SaveDocument(ctx context.Context, doc *models.KnowledgeDocument) error {
	if err := r.db.WithContext(ctx).Save(doc).Error; err != nil {
		return fmt.Errorf("save document: %w", err)
	}
	return nil
}

// DeleteDocumentsByAgent removes all documents for a given agent.
func (r *GORMKnowledgeRepository) DeleteDocumentsByAgent(ctx context.Context, agentName string) error {
	if err := r.db.WithContext(ctx).Where("agent_name = ?", agentName).Delete(&models.KnowledgeDocument{}).Error; err != nil {
		return fmt.Errorf("delete documents by agent: %w", err)
	}
	return nil
}

// ListDocumentsByAgent returns all documents belonging to a given agent.
func (r *GORMKnowledgeRepository) ListDocumentsByAgent(ctx context.Context, agentName string) ([]models.KnowledgeDocument, error) {
	var docs []models.KnowledgeDocument
	if err := r.db.WithContext(ctx).Where("agent_name = ?", agentName).Find(&docs).Error; err != nil {
		return nil, fmt.Errorf("list documents by agent: %w", err)
	}
	return docs, nil
}

// SaveChunks inserts a batch of knowledge chunks.
func (r *GORMKnowledgeRepository) SaveChunks(ctx context.Context, chunks []models.KnowledgeChunk) error {
	if len(chunks) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).Create(&chunks).Error; err != nil {
		return fmt.Errorf("save chunks: %w", err)
	}
	return nil
}

// DeleteChunksByDocument removes all chunks belonging to a document.
func (r *GORMKnowledgeRepository) DeleteChunksByDocument(ctx context.Context, documentID string) error {
	if err := r.db.WithContext(ctx).Where("document_id = ?", documentID).Delete(&models.KnowledgeChunk{}).Error; err != nil {
		return fmt.Errorf("delete chunks by document: %w", err)
	}
	return nil
}

// DeleteChunksByAgent removes all chunks belonging to an agent.
func (r *GORMKnowledgeRepository) DeleteChunksByAgent(ctx context.Context, agentName string) error {
	if err := r.db.WithContext(ctx).Where("agent_name = ?", agentName).Delete(&models.KnowledgeChunk{}).Error; err != nil {
		return fmt.Errorf("delete chunks by agent: %w", err)
	}
	return nil
}

// SearchSimilar finds the most similar chunks by cosine distance using pgvector.
func (r *GORMKnowledgeRepository) SearchSimilar(ctx context.Context, agentName string, embedding pgvector.Vector, limit int) ([]models.KnowledgeChunk, error) {
	var chunks []models.KnowledgeChunk
	err := r.db.WithContext(ctx).
		Raw("SELECT * FROM knowledge_chunks WHERE agent_name = ? ORDER BY embedding <=> ? LIMIT ?", agentName, embedding, limit).
		Scan(&chunks).Error
	if err != nil {
		return nil, fmt.Errorf("search similar: %w", err)
	}
	return chunks, nil
}

// GetStats returns document count, chunk count, and last indexed time for an agent.
func (r *GORMKnowledgeRepository) GetStats(ctx context.Context, agentName string) (docCount int, chunkCount int, lastIndexed *time.Time, err error) {
	var dc int64
	if err := r.db.WithContext(ctx).Model(&models.KnowledgeDocument{}).Where("agent_name = ?", agentName).Count(&dc).Error; err != nil {
		return 0, 0, nil, fmt.Errorf("count documents: %w", err)
	}

	var cc int64
	if err := r.db.WithContext(ctx).Model(&models.KnowledgeChunk{}).Where("agent_name = ?", agentName).Count(&cc).Error; err != nil {
		return 0, 0, nil, fmt.Errorf("count chunks: %w", err)
	}

	var doc models.KnowledgeDocument
	result := r.db.WithContext(ctx).
		Where("agent_name = ?", agentName).
		Order("indexed_at DESC").
		First(&doc)
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return 0, 0, nil, fmt.Errorf("get last indexed: %w", result.Error)
	}

	var li *time.Time
	if result.Error == nil {
		li = &doc.IndexedAt
	}

	return int(dc), int(cc), li, nil
}

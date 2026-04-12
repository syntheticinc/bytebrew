package models

import (
	"time"

	"github.com/pgvector/pgvector-go"
)

// KnowledgeDocument represents an indexed document in the knowledge base.
type KnowledgeDocument struct {
	ID         string    `gorm:"primaryKey;type:varchar(36)"`
	TenantID   string    `gorm:"type:varchar(36);not null;default:'default';index:idx_knowledge_docs_tenant_agent"` // tenant isolation (WP-3)
	AgentName  string    `gorm:"type:varchar(255);not null;index:idx_knowledge_docs_tenant_agent"`
	FilePath   string    `gorm:"type:text;not null"`
	FileName   string    `gorm:"type:varchar(500);not null"`
	FileType   string    `gorm:"type:varchar(20);not null;default:txt"` // pdf, docx, doc, txt, md, csv (AC-KB-FMT-01..05)
	FileSize   int64     `gorm:"not null;default:0"`                    // bytes (AC-KB-LIST-02)
	FileHash   string    `gorm:"type:varchar(64);not null"`             // SHA256
	Status     string    `gorm:"type:varchar(20);not null;default:uploading"` // uploading, indexing, ready, error (AC-KB-LIST-03)
	StatusMsg  string    `gorm:"type:text"`                             // error message if status=error
	ChunkCount int
	IndexedAt  time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (KnowledgeDocument) TableName() string { return "knowledge_documents" }

// KnowledgeChunk represents a single chunk of a document with its embedding.
type KnowledgeChunk struct {
	ID         string          `gorm:"primaryKey;type:varchar(36)"`
	DocumentID string          `gorm:"type:varchar(36);not null;index"`
	TenantID   string          `gorm:"type:varchar(36);not null;default:'default';index:idx_knowledge_chunks_tenant_agent"` // denormalized for fast WHERE (WP-3)
	AgentName  string          `gorm:"type:varchar(255);not null;index:idx_knowledge_chunks_tenant_agent"` // denormalized for fast filtering
	Content    string          `gorm:"type:text;not null"`
	ChunkOrder int
	Embedding  pgvector.Vector `gorm:"type:vector(768)"`
	CreatedAt  time.Time

	Document KnowledgeDocument `gorm:"foreignKey:DocumentID"`
}

func (KnowledgeChunk) TableName() string { return "knowledge_chunks" }

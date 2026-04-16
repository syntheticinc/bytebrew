package models

import (
	"path/filepath"
	"time"

	"github.com/pgvector/pgvector-go"
)

// KnowledgeBase is a standalone knowledge collection linked to agents via many-to-many.
// Analogous to LLMProviderModel (Models): a global entity that agents reference.
type KnowledgeBase struct {
	ID               string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID         string    `gorm:"type:varchar(36);not null;default:'default';index:idx_kb_tenant"`
	Name             string    `gorm:"type:varchar(255);not null"`
	Description      string    `gorm:"type:text"`
	EmbeddingModelID *string   `gorm:"type:uuid"` // FK to models table (type=embedding)
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (KnowledgeBase) TableName() string { return "knowledge_bases" }

// KnowledgeBaseAgent is the join table for many-to-many between KnowledgeBase and Agent.
// Uses agent_id uuid (converted from agent_name in migration 029).
type KnowledgeBaseAgent struct {
	KnowledgeBaseID string `gorm:"primaryKey;type:uuid;not null;index:idx_kba_kb"`
	AgentID         string `gorm:"primaryKey;type:uuid;not null;index:idx_kba_agent"`
}

func (KnowledgeBaseAgent) TableName() string { return "knowledge_base_agents" }

// KnowledgeDocument represents an indexed document in a knowledge base.
// file_name dropped in migration 029 — derivable as filepath.Base(FilePath).
// agent_name dropped in migration 029 — pre-KB migration legacy.
type KnowledgeDocument struct {
	ID              string    `gorm:"primaryKey;type:varchar(36)"`
	KnowledgeBaseID string    `gorm:"type:varchar(36);index:idx_knowledge_docs_kb"`
	TenantID        string    `gorm:"type:varchar(36);not null;default:'default';index:idx_knowledge_docs_tenant"`
	FilePath        string    `gorm:"type:text;not null"`
	FileType        string    `gorm:"type:varchar(20);not null;default:txt"` // pdf, docx, doc, txt, md, csv
	FileSize        int64     `gorm:"not null;default:0"`
	FileHash        string    `gorm:"type:varchar(64);not null"`
	Status          string    `gorm:"type:varchar(20);not null;default:uploading"` // uploading, indexing, ready, error
	StatusMsg       string    `gorm:"type:text"`
	ChunkCount      int
	IndexedAt       time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (KnowledgeDocument) TableName() string { return "knowledge_documents" }

// FileName returns the base name of the file path (derived, not stored).
func (d *KnowledgeDocument) FileName() string {
	return filepath.Base(d.FilePath)
}

// KnowledgeChunk represents a single chunk of a document with its embedding.
// agent_name and knowledge_base_id dropped in migration 029 — derive via document->KB joins.
type KnowledgeChunk struct {
	ID         string          `gorm:"primaryKey;type:varchar(36)"`
	DocumentID string          `gorm:"type:varchar(36);not null;index"`
	TenantID   string          `gorm:"type:varchar(36);not null;default:'default';index:idx_knowledge_chunks_tenant"`
	Content    string          `gorm:"type:text;not null"`
	ChunkOrder int
	Embedding  pgvector.Vector `gorm:"type:vector"` // variable dimension
	CreatedAt  time.Time

	Document KnowledgeDocument `gorm:"foreignKey:DocumentID"`
}

func (KnowledgeChunk) TableName() string { return "knowledge_chunks" }

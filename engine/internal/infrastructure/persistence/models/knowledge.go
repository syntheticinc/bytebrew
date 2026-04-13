package models

import (
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
// Uses agent_name (not UUID) because agents are identified by name throughout the system.
type KnowledgeBaseAgent struct {
	KnowledgeBaseID string `gorm:"primaryKey;type:uuid;not null;index:idx_kba_kb"`
	AgentName       string `gorm:"primaryKey;type:varchar(255);not null;index:idx_kba_agent"`
}

func (KnowledgeBaseAgent) TableName() string { return "knowledge_base_agents" }

// KnowledgeDocument represents an indexed document in a knowledge base.
type KnowledgeDocument struct {
	ID              string    `gorm:"primaryKey;type:varchar(36)"`
	KnowledgeBaseID string    `gorm:"type:varchar(36);index:idx_knowledge_docs_kb"` // FK to knowledge_bases (empty for legacy agent-scoped docs)
	TenantID        string    `gorm:"type:varchar(36);not null;default:'default';index:idx_knowledge_docs_tenant"`
	AgentName       string    `gorm:"type:varchar(255);index"` // legacy, kept for migration; new docs use KnowledgeBaseID
	FilePath        string    `gorm:"type:text;not null"`
	FileName        string    `gorm:"type:varchar(500);not null"`
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

// KnowledgeChunk represents a single chunk of a document with its embedding.
type KnowledgeChunk struct {
	ID              string          `gorm:"primaryKey;type:varchar(36)"`
	DocumentID      string          `gorm:"type:varchar(36);not null;index"`
	KnowledgeBaseID string          `gorm:"type:varchar(36);index:idx_knowledge_chunks_kb"` // denormalized for fast search (empty for legacy)
	TenantID        string          `gorm:"type:varchar(36);not null;default:'default';index:idx_knowledge_chunks_tenant"`
	AgentName       string          `gorm:"type:varchar(255);index"` // legacy, kept for migration
	Content         string          `gorm:"type:text;not null"`
	ChunkOrder      int
	Embedding       pgvector.Vector `gorm:"type:vector"` // variable dimension
	CreatedAt       time.Time

	Document KnowledgeDocument `gorm:"foreignKey:DocumentID"`
}

func (KnowledgeChunk) TableName() string { return "knowledge_chunks" }

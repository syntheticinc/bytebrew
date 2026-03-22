package models

import (
	"time"

	"github.com/pgvector/pgvector-go"
)

// KnowledgeDocument represents an indexed document in the knowledge base.
type KnowledgeDocument struct {
	ID         string    `gorm:"primaryKey;type:varchar(36)"`
	AgentName  string    `gorm:"type:varchar(255);not null;index"`
	FilePath   string    `gorm:"type:text;not null"`
	FileName   string    `gorm:"type:varchar(500);not null"`
	FileHash   string    `gorm:"type:varchar(64);not null"` // SHA256
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
	AgentName  string          `gorm:"type:varchar(255);not null;index"` // denormalized for fast filtering
	Content    string          `gorm:"type:text;not null"`
	ChunkOrder int
	Embedding  pgvector.Vector `gorm:"type:vector(768)"`
	CreatedAt  time.Time

	Document KnowledgeDocument `gorm:"foreignKey:DocumentID"`
}

func (KnowledgeChunk) TableName() string { return "knowledge_chunks" }

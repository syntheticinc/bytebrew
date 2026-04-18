package models

import "time"

// AgentContextSnapshotModel maps to the "agent_context_snapshots" table.
// Stores agent context snapshots for session resume.
//
// Uniqueness: one ACTIVE snapshot per (session_id, agent_id). The uniqueness
// constraint is expressed as a partial unique index (WHERE status='active')
// — applied via raw SQL in migrate.go because GORM tags cannot express the
// WHERE clause. Compacted/expired rows accumulate as history.
type AgentContextSnapshotModel struct {
	ID            string    `gorm:"primaryKey;type:uuid"`
	TenantID      string    `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000001'" json:"tenant_id"`
	SessionID     string    `gorm:"type:uuid;not null"`
	AgentID       string    `gorm:"type:uuid;not null"`
	SchemaVersion int       `gorm:"not null;default:1"`
	ContextData   []byte    `gorm:"type:bytea;not null"`
	StepNumber    int       `gorm:"not null;default:0"`
	TokenCount    int       `gorm:"not null;default:0"`
	Status        string    `gorm:"type:varchar(20);not null;default:active"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
}

func (AgentContextSnapshotModel) TableName() string { return "agent_context_snapshots" }

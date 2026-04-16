package models

import "time"

// AgentContextSnapshotModel maps to the "agent_context_snapshots" table.
// Stores agent context snapshots for session resume.
type AgentContextSnapshotModel struct {
	ID            string    `gorm:"primaryKey;type:varchar(36)"`
	SessionID     string    `gorm:"type:varchar(36);not null;uniqueIndex:idx_rt_ctx_session_agent"`
	AgentID       string    `gorm:"type:uuid;not null;uniqueIndex:idx_rt_ctx_session_agent"`
	SchemaVersion int       `gorm:"not null;default:1"`
	ContextData   []byte    `gorm:"type:bytea;not null"`
	StepNumber    int       `gorm:"not null;default:0"`
	TokenCount    int       `gorm:"not null;default:0"`
	Status        string    `gorm:"type:varchar(20);not null;default:active"`
	TenantID      string    `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000001'" json:"tenant_id"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
}

func (AgentContextSnapshotModel) TableName() string { return "agent_context_snapshots" }

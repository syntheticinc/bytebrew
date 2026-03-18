package models

import "time"

// RuntimeAgentContextModel maps to the "runtime_agent_contexts" table.
// Stores agent context snapshots for session resume.
type RuntimeAgentContextModel struct {
	ID            string    `gorm:"primaryKey;type:varchar(36)"`
	SessionID     string    `gorm:"type:varchar(36);not null;index:idx_rt_ctx_session_agent"`
	AgentID       string    `gorm:"type:varchar(100);not null;uniqueIndex:idx_rt_ctx_agent_unique"`
	FlowType      string    `gorm:"type:varchar(50);not null"`
	SchemaVersion int       `gorm:"not null;default:1"`
	ContextData   []byte    `gorm:"type:bytea;not null"`
	StepNumber    int       `gorm:"not null;default:0"`
	TokenCount    int       `gorm:"not null;default:0"`
	Status        string    `gorm:"type:varchar(20);not null;default:active"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
}

func (RuntimeAgentContextModel) TableName() string { return "runtime_agent_contexts" }

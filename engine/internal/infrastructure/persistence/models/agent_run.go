package models

import "time"

// AgentRunModel maps to the "agent_runs" table.
// Stores domain.AgentRun data for Code Agent execution tracking.
type AgentRunModel struct {
	ID          string     `gorm:"primaryKey;type:varchar(36)"`
	SubtaskID   string     `gorm:"type:varchar(36);not null"`
	SessionID   string     `gorm:"type:varchar(36);not null;index"`
	FlowType    string     `gorm:"type:varchar(50);not null;default:coder"`
	Status      string     `gorm:"type:varchar(20);not null;index:idx_agent_runs_session_status"`
	Result      string     `gorm:"type:text"`
	Error       string     `gorm:"type:text"`
	TenantID    string     `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000001'" json:"tenant_id"`
	StartedAt   time.Time  `gorm:"not null"`
	CompletedAt *time.Time
}

func (AgentRunModel) TableName() string { return "agent_runs" }

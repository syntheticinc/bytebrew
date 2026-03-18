package models

import "time"

// RuntimeAgentRunModel maps to the "runtime_agent_runs" table.
// Stores domain.AgentRun data for Code Agent execution tracking.
type RuntimeAgentRunModel struct {
	ID          string     `gorm:"primaryKey;type:varchar(36)"`
	SubtaskID   string     `gorm:"type:varchar(36);not null"`
	SessionID   string     `gorm:"type:varchar(36);not null;index"`
	FlowType    string     `gorm:"type:varchar(50);not null;default:coder"`
	Status      string     `gorm:"type:varchar(20);not null;index:idx_agent_runs_session_status"`
	Result      string     `gorm:"type:text"`
	Error       string     `gorm:"type:text"`
	StartedAt   time.Time  `gorm:"not null"`
	CompletedAt *time.Time
}

func (RuntimeAgentRunModel) TableName() string { return "runtime_agent_runs" }

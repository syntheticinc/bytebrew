package models

import "time"

// AgentRelationModel maps to the "agent_relations" table.
//
// V2 has a single implicit relationship type — DELEGATION — expressed via the
// agent-first runtime (orchestrator delegates to target via a tool call).
// Hence no `type` column. See docs/architecture/agent-first-runtime.md §3.1
// and the V2 cleanup checklist (Group A.1).
type AgentRelationModel struct {
	ID              string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	SchemaID        string    `gorm:"type:uuid;not null;index"`
	SourceAgentName string    `gorm:"type:varchar(255);not null"`
	TargetAgentName string    `gorm:"type:varchar(255);not null"`
	Config          string    `gorm:"type:text"` // JSON, optional routing hints (priority, conditions)
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`

	Schema SchemaModel `gorm:"foreignKey:SchemaID"`
}

func (AgentRelationModel) TableName() string { return "agent_relations" }

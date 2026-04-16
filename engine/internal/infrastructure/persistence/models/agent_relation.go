package models

import "time"

// AgentRelationModel maps to the "agent_relations" table.
//
// V2 has a single implicit relationship type — DELEGATION — expressed via the
// agent-first runtime (orchestrator delegates to target via a tool call).
// Hence no `type` column. See docs/architecture/agent-first-runtime.md §3.1
// and the V2 cleanup checklist (Group A.1).
//
// Q.5: source/target are now uuid FKs to agents.id (was agent_name varchar).
type AgentRelationModel struct {
	ID            string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	SchemaID      string    `gorm:"type:uuid;not null;index"`
	SourceAgentID string    `gorm:"type:uuid;not null;uniqueIndex:idx_agent_relations_pair"`
	TargetAgentID string    `gorm:"type:uuid;not null;uniqueIndex:idx_agent_relations_pair"`
	Config        string    `gorm:"type:jsonb"` // JSON, optional routing hints (priority, conditions)
	TenantID      string    `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000001'" json:"tenant_id"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`

	Schema      SchemaModel `gorm:"foreignKey:SchemaID"`
	SourceAgent AgentModel  `gorm:"foreignKey:SourceAgentID"`
	TargetAgent AgentModel  `gorm:"foreignKey:TargetAgentID"`
}

func (AgentRelationModel) TableName() string { return "agent_relations" }

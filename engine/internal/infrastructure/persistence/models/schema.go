package models

import "time"

// SchemaModel maps to the "schemas" table.
//
// V2: schema membership is derived from `agent_relations` (see
// docs/architecture/agent-first-runtime.md §2.1) — there is no
// `schema_agents` join table.
//
// ChatEnabled replaces the removed triggers table: when true, the schema
// exposes POST /api/v1/schemas/{id}/chat against schemas.entry_agent_id.
type SchemaModel struct {
	ID               string     `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID         string     `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000001';uniqueIndex:idx_schemas_tenant_name,priority:1;index" json:"tenant_id"`
	Name             string     `gorm:"not null;type:varchar(255);uniqueIndex:idx_schemas_tenant_name,priority:2"`
	Description      string     `gorm:"type:text"`
	EntryAgentID     *string    `gorm:"type:uuid;index" json:"entry_agent_id"`
	ChatEnabled      bool       `gorm:"column:chat_enabled;not null;default:false" json:"chat_enabled"`
	ChatLastFiredAt  *time.Time `gorm:"column:chat_last_fired_at" json:"chat_last_fired_at"`
	IsSystem         bool       `gorm:"column:is_system;not null;default:false"`
	CreatedAt        time.Time  `gorm:"autoCreateTime"`
	UpdatedAt        time.Time  `gorm:"autoUpdateTime"`
}

func (SchemaModel) TableName() string { return "schemas" }

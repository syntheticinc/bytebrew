package models

import "time"

// SchemaModel maps to the "schemas" table.
//
// V2: schema membership is derived from `agent_relations` (see
// docs/architecture/agent-first-runtime.md §2.1) — there is no
// `schema_agents` join table.
type SchemaModel struct {
	ID          string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Name        string    `gorm:"uniqueIndex;not null;type:varchar(255)"`
	Description string    `gorm:"type:text"`
	IsSystem    bool      `gorm:"column:is_system;not null;default:false"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (SchemaModel) TableName() string { return "schemas" }

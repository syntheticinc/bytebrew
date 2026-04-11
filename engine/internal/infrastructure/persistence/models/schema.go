package models

import "time"

// SchemaModel maps to the "schemas" table.
type SchemaModel struct {
	ID          string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Name        string    `gorm:"uniqueIndex;not null;type:varchar(255)"`
	Description string    `gorm:"type:text"`
	IsSystem    bool      `gorm:"column:is_system;not null;default:false"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (SchemaModel) TableName() string { return "schemas" }

// SchemaAgentModel maps to the "schema_agents" join table (many-to-many: schema <-> agent).
type SchemaAgentModel struct {
	ID       string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	SchemaID string `gorm:"type:uuid;not null;uniqueIndex:idx_schema_agent"`
	AgentID  string `gorm:"type:uuid;not null;uniqueIndex:idx_schema_agent"`
	Position int    `gorm:"not null;default:0"` // ordering on canvas
}

func (SchemaAgentModel) TableName() string { return "schema_agents" }

package models

import "time"

// SessionModel maps to the "sessions" table.
//
// Q.5: dropped agent_name and agent_id — a session belongs to a schema
// (schema_id), not a single agent. The entry agent is resolved via
// schemas.entry_agent_id at dispatch time. schema_id is now NOT NULL.
type SessionModel struct {
	ID          string     `gorm:"primaryKey;type:uuid"`
	Title       string     `gorm:"type:varchar(500)"`
	UserID      *string    `gorm:"type:uuid;index"`
	Status      string     `gorm:"type:varchar(20);not null;default:active;index"`
	TenantID    string     `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000001'" json:"tenant_id"`
	SchemaID    string     `gorm:"type:uuid;not null" json:"schema_id"`
	CreatedAt   time.Time  `gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime"`
	CompletedAt *time.Time

	Tasks []TaskModel `gorm:"foreignKey:SessionID"`
}

func (SessionModel) TableName() string { return "sessions" }

package models

import "time"

// MemoryModel maps to the "memories" table.
// Stores cross-session memory entries scoped to a schema.
type MemoryModel struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	SchemaID  string    `gorm:"type:uuid;not null;index:idx_memories_schema_user"`
	UserID    string    `gorm:"type:uuid;not null;index:idx_memories_schema_user"`
	Content   string    `gorm:"type:text;not null"`
	Metadata  string    `gorm:"type:jsonb"` // JSON
	TenantID  string    `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000001'" json:"tenant_id"`
	CreatedAt time.Time `gorm:"autoCreateTime;index:idx_memories_created"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (MemoryModel) TableName() string { return "memories" }

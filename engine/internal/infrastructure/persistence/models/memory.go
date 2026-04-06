package models

import "time"

// MemoryModel maps to the "memories" table.
// Stores cross-session memory entries scoped to a schema.
type MemoryModel struct {
	ID        uint      `gorm:"primaryKey"`
	SchemaID  uint      `gorm:"not null;index:idx_memories_schema_user"`
	UserID    string    `gorm:"type:varchar(255);not null;index:idx_memories_schema_user"`
	Content   string    `gorm:"type:text;not null"`
	Metadata  string    `gorm:"type:text"` // JSON
	CreatedAt time.Time `gorm:"autoCreateTime;index:idx_memories_created"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (MemoryModel) TableName() string { return "memories" }

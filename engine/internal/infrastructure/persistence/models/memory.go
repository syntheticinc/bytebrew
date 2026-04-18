package models

import "time"

// MemoryModel maps to the "memories" table.
// Stores cross-session memory entries scoped to (tenant, schema, user_sub).
// user_sub is the opaque JWT sub claim — end-users are external, no FK.
type MemoryModel struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID  string    `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000001';index:idx_memories_isolation,priority:1" json:"tenant_id"`
	SchemaID  string    `gorm:"type:uuid;not null;index:idx_memories_isolation,priority:2"`
	UserSub   string    `gorm:"column:user_sub;type:varchar(255);not null;index:idx_memories_isolation,priority:3" json:"user_sub"`
	Content   string    `gorm:"type:text;not null"`
	Metadata  string    `gorm:"type:jsonb"` // JSON
	CreatedAt time.Time `gorm:"autoCreateTime;index:idx_memories_created"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (MemoryModel) TableName() string { return "memories" }

package models

import "time"

// UserModel maps to the "users" table.
// Lazy-created on first JWT/token seen. No password storage — auth is external.
type UserModel struct {
	ID          string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID    string    `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000001';index:idx_users_tenant_id;uniqueIndex:idx_users_tenant_external"`
	ExternalID  string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_users_tenant_external"`
	Email       *string   `gorm:"type:varchar(255)"`
	DisplayName *string   `gorm:"type:varchar(255)"`
	Disabled    bool      `gorm:"not null;default:false"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (UserModel) TableName() string { return "users" }

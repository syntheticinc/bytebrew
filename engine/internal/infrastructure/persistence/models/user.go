package models

import "time"

// UserModel maps to the "users" table.
// System/admin users only — end-users are external (identified by user_sub on sessions/memories).
// Schema is managed by Liquibase — GORM tags here are for field mapping only, NOT for AutoMigrate.
type UserModel struct {
	ID           string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID     string    `gorm:"column:tenant_id;type:uuid;not null;default:'00000000-0000-0000-0000-000000000001'"`
	Username     string    `gorm:"column:username;type:varchar(255);not null"`
	PasswordHash string    `gorm:"column:password_hash;type:varchar(255);not null"`
	Role         string    `gorm:"column:role;type:varchar(20);not null;default:'admin'"`
	Disabled     bool      `gorm:"column:disabled;not null;default:false"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

func (UserModel) TableName() string { return "users" }

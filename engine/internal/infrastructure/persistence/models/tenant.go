package models

import "time"

// TenantModel maps to the "tenants" table.
type TenantModel struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantUID string    `gorm:"uniqueIndex;type:varchar(255);not null"` // external UUID
	Email     string    `gorm:"uniqueIndex;type:varchar(255);not null"`
	PlanType  string    `gorm:"type:varchar(50);not null;default:free"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (TenantModel) TableName() string { return "tenants" }

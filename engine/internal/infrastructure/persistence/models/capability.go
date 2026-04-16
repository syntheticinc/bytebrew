package models

import "time"

// CapabilityModel maps to the "capabilities" table.
type CapabilityModel struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	AgentID   string    `gorm:"type:uuid;not null;index"`
	Type      string    `gorm:"type:varchar(50);not null"`
	Config    string    `gorm:"type:text"` // JSON
	Enabled   bool      `gorm:"not null;default:true"`
	TenantID  string    `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000001'" json:"tenant_id"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`

	Agent AgentModel `gorm:"foreignKey:AgentID"`
}

func (CapabilityModel) TableName() string { return "capabilities" }

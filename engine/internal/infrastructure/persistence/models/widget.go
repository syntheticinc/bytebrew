package models

import "time"

// WidgetModel maps to the "widgets" table.
type WidgetModel struct {
	ID              string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID        string    `gorm:"type:varchar(255);index"`
	Name            string    `gorm:"type:varchar(255);not null"`
	SchemaID        string    `gorm:"type:uuid;not null;index"`
	PrimaryColor    string    `gorm:"type:varchar(20);not null;default:#6366f1"`
	Position        string    `gorm:"type:varchar(20);not null;default:bottom-right"`
	Size            string    `gorm:"type:varchar(20);not null;default:standard"`
	WelcomeMessage  string    `gorm:"type:text"`
	Placeholder     string    `gorm:"type:varchar(255)"`
	AvatarURL       string    `gorm:"type:varchar(500)"`
	DomainWhitelist string    `gorm:"type:text"` // comma-separated
	CustomHeaders   string    `gorm:"type:text"` // JSON map[string]string
	Enabled         bool      `gorm:"not null;default:true"`
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`

	Schema SchemaModel `gorm:"foreignKey:SchemaID"`
}

func (WidgetModel) TableName() string { return "widgets" }

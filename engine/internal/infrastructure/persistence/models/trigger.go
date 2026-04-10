package models

import "time"

// TriggerModel maps to the "triggers" table.
type TriggerModel struct {
	ID                uint       `gorm:"primaryKey"`
	Type              string     `gorm:"type:varchar(10);not null;index"`
	Title             string     `gorm:"type:varchar(255);not null"`
	AgentID           *uint      `gorm:"index"`
	SchemaID          *uint      `gorm:"index;constraint:OnDelete:SET NULL"`
	Schedule          string     `gorm:"type:varchar(100)"`
	WebhookPath       string     `gorm:"type:varchar(500)"`
	Description       string     `gorm:"type:text"`
	Enabled           bool       `gorm:"not null;default:true"`
	OnCompleteURL     string     `gorm:"column:on_complete_url;type:varchar(2000)"`
	OnCompleteHeaders string     `gorm:"column:on_complete_headers;type:text"`
	LastFiredAt       *time.Time
	CreatedAt         time.Time  `gorm:"autoCreateTime"`
	UpdatedAt         time.Time  `gorm:"autoUpdateTime"`

	Agent  AgentModel  `gorm:"foreignKey:AgentID"`
	Schema SchemaModel `gorm:"foreignKey:SchemaID"`
}

func (TriggerModel) TableName() string { return "triggers" }

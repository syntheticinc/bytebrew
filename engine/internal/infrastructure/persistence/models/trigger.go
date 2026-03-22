package models

import "time"

// TriggerModel maps to the "triggers" table.
type TriggerModel struct {
	ID          uint       `gorm:"primaryKey"`
	Type        string     `gorm:"type:varchar(10);not null;index"`
	Title       string     `gorm:"type:varchar(255);not null"`
	AgentID     uint       `gorm:"not null;index"`
	Schedule    string     `gorm:"type:varchar(100)"`
	WebhookPath string     `gorm:"type:varchar(500);uniqueIndex"`
	Description string     `gorm:"type:text"`
	Enabled     bool       `gorm:"not null;default:true"`
	LastFiredAt *time.Time
	CreatedAt   time.Time  `gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime"`

	Agent AgentModel `gorm:"foreignKey:AgentID"`
}

func (TriggerModel) TableName() string { return "triggers" }

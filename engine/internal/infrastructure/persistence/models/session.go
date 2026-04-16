package models

import "time"

// SessionModel maps to the "sessions" table.
type SessionModel struct {
	ID          string     `gorm:"primaryKey;type:varchar(36)"`
	Title       string     `gorm:"type:varchar(500)"`
	AgentName   string     `gorm:"type:varchar(255);not null"`
	AgentID     *string    `gorm:"type:uuid;index"`
	UserID      *string    `gorm:"type:uuid;index"`
	Status      string     `gorm:"type:varchar(20);not null;default:active;index"`
	CreatedAt   time.Time  `gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime"`
	CompletedAt *time.Time

	Agent *AgentModel `gorm:"foreignKey:AgentID"`
	Tasks []TaskModel `gorm:"foreignKey:SessionID"`
}

func (SessionModel) TableName() string { return "sessions" }

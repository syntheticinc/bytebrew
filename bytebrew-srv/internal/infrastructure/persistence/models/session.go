package models

import "time"

// SessionModel maps to the "sessions" table.
type SessionModel struct {
	ID          string     `gorm:"primaryKey;type:varchar(36)"`
	AgentName   string     `gorm:"type:varchar(255);not null"`
	AgentID     *uint      `gorm:"index"`
	UserID      string     `gorm:"type:varchar(255);index"`
	Status      string     `gorm:"type:varchar(20);not null;default:active;index"`
	CreatedAt   time.Time  `gorm:"autoCreateTime"`
	CompletedAt *time.Time

	Agent  *AgentModel         `gorm:"foreignKey:AgentID"`
	Events []SessionEventModel `gorm:"foreignKey:SessionID"`
	Tasks  []TaskModel         `gorm:"foreignKey:SessionID"`
}

func (SessionModel) TableName() string { return "sessions" }

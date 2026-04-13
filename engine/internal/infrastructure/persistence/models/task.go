package models

import (
	"time"

	"github.com/google/uuid"
)

// TaskModel maps to the "tasks" table.
type TaskModel struct {
	ID                 uuid.UUID  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Title              string     `gorm:"type:varchar(500);not null"`
	Description        string     `gorm:"type:text"`
	AcceptanceCriteria string     `gorm:"type:text"` // JSON array
	AgentName          string     `gorm:"type:varchar(255);not null;index"`
	Source             string     `gorm:"type:varchar(20);not null;index"`
	SourceID           string     `gorm:"type:varchar(255);index:idx_source_composite"`
	UserID             string     `gorm:"type:varchar(255);index"`
	SessionID          *string    `gorm:"type:varchar(36);index"`
	ParentTaskID       *uuid.UUID `gorm:"type:uuid;index"`
	Depth              int        `gorm:"not null;default:0"`
	Status             string     `gorm:"type:varchar(20);not null;default:pending;index"`
	Mode               string     `gorm:"type:varchar(20);not null;default:interactive"`
	Priority           int        `gorm:"not null;default:0"`
	AssignedAgentID    string     `gorm:"type:varchar(100);index"`
	BlockedBy          string     `gorm:"type:text"` // JSON array of task UUIDs
	Result             string     `gorm:"type:text"`
	Error              string     `gorm:"type:text"`
	CreatedAt          time.Time  `gorm:"autoCreateTime;index"`
	UpdatedAt          time.Time  `gorm:"autoUpdateTime"`
	ApprovedAt         *time.Time
	StartedAt          *time.Time
	CompletedAt        *time.Time

	// Session association disabled — tasks can exist without sessions (webhook, cron, API)
	ParentTask *TaskModel  `gorm:"foreignKey:ParentTaskID"`
	SubTasks   []TaskModel `gorm:"foreignKey:ParentTaskID"`
}

func (TaskModel) TableName() string { return "tasks" }

package models

import (
	"time"

	"github.com/google/uuid"
)

// TaskModel maps to the "tasks" table.
//
// Q.5: dropped agent_name (derived from session's schema), source/source_id
// (not in target), assigned_agent_id (not in target), depth (computed from
// parent_task_id chain). See target-schema.dbml Table tasks.
type TaskModel struct {
	ID                 uuid.UUID  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Title              string     `gorm:"type:varchar(500);not null"`
	Description        string     `gorm:"type:text"`
	AcceptanceCriteria string     `gorm:"type:text"` // JSON array
	UserID             string     `gorm:"type:varchar(255);index"`
	SessionID          *string    `gorm:"type:varchar(36);index"`
	ParentTaskID       *uuid.UUID `gorm:"type:uuid;index"`
	Status             string     `gorm:"type:varchar(20);not null;default:pending;index"`
	Mode               string     `gorm:"type:varchar(20);not null;default:interactive"`
	Priority           int        `gorm:"not null;default:0"`
	BlockedBy          string     `gorm:"type:text"` // JSON array of task UUIDs
	Result             string     `gorm:"type:text"`
	Error              string     `gorm:"type:text"`
	TenantID           string     `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000001'" json:"tenant_id"`
	CreatedAt          time.Time  `gorm:"autoCreateTime;index"`
	UpdatedAt          time.Time  `gorm:"autoUpdateTime"`
	ApprovedAt         *time.Time
	StartedAt          *time.Time
	CompletedAt        *time.Time

	ParentTask *TaskModel  `gorm:"foreignKey:ParentTaskID"`
	SubTasks   []TaskModel `gorm:"foreignKey:ParentTaskID"`
}

func (TaskModel) TableName() string { return "tasks" }

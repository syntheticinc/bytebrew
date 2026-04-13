package models

import "time"

// RuntimeTaskModel maps to the "runtime_tasks" table.
// Stores domain.Task data (legacy work-task with AcceptanceCriteria).
// Not to be confused with TaskModel (admin dashboard tasks).
type RuntimeTaskModel struct {
	ID                 string     `gorm:"primaryKey;type:varchar(36)"`
	SessionID          string     `gorm:"type:varchar(36);not null;index"`
	Title              string     `gorm:"type:varchar(500);not null"`
	Description        string     `gorm:"type:text"`
	AcceptanceCriteria string     `gorm:"type:text"` // JSON array
	Status             string     `gorm:"type:varchar(20);not null;default:draft;index"`
	Priority           int        `gorm:"not null;default:0"`
	CreatedAt          time.Time  `gorm:"autoCreateTime"`
	UpdatedAt          time.Time  `gorm:"autoUpdateTime"`
	ApprovedAt         *time.Time
	CompletedAt        *time.Time
}

func (RuntimeTaskModel) TableName() string { return "runtime_tasks" }

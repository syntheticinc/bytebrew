package models

import "time"

// RuntimeSubtaskModel maps to the "runtime_subtasks" table.
// Stores domain.Subtask data for Code Agent work items.
type RuntimeSubtaskModel struct {
	ID              string     `gorm:"primaryKey;type:varchar(36)"`
	SessionID       string     `gorm:"type:varchar(36);not null;index"`
	TaskID          string     `gorm:"type:varchar(36);not null;index"`
	Title           string     `gorm:"type:varchar(500);not null"`
	Description     string     `gorm:"type:text"`
	Status          string     `gorm:"type:varchar(20);not null;default:pending;index"`
	AssignedAgentID string     `gorm:"type:varchar(100);index"`
	BlockedBy       string     `gorm:"type:text"` // JSON array of subtask IDs
	FilesInvolved   string     `gorm:"type:text"` // JSON array of file paths
	Result          string     `gorm:"type:text"`
	Context         string     `gorm:"type:text"` // JSON map[string]string
	CreatedAt       time.Time  `gorm:"autoCreateTime"`
	UpdatedAt       time.Time  `gorm:"autoUpdateTime"`
	CompletedAt     *time.Time
}

func (RuntimeSubtaskModel) TableName() string { return "runtime_subtasks" }

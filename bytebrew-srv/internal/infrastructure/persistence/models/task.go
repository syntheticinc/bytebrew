package models

import "time"

// TaskModel maps to the "tasks" table.
type TaskModel struct {
	ID           uint       `gorm:"primaryKey"`
	Title        string     `gorm:"type:varchar(500);not null"`
	Description  string     `gorm:"type:text"`
	AgentName    string     `gorm:"type:varchar(255);not null;index"`
	Source       string     `gorm:"type:varchar(20);not null;index"`
	SourceID     string     `gorm:"type:varchar(255);index:idx_source_composite"`
	UserID       string     `gorm:"type:varchar(255);index"`
	SessionID    string     `gorm:"type:varchar(36);index"`
	ParentTaskID *uint      `gorm:"index"`
	Depth        int        `gorm:"not null;default:0"`
	Status       string     `gorm:"type:varchar(20);not null;default:pending;index"`
	Mode         string     `gorm:"type:varchar(20);not null;default:interactive"`
	Result       string     `gorm:"type:text"`
	Error        string     `gorm:"type:text"`
	CreatedAt    time.Time  `gorm:"autoCreateTime;index"`
	StartedAt    *time.Time
	CompletedAt  *time.Time

	Session    *SessionModel `gorm:"foreignKey:SessionID"`
	ParentTask *TaskModel    `gorm:"foreignKey:ParentTaskID"`
	SubTasks   []TaskModel   `gorm:"foreignKey:ParentTaskID"`
}

func (TaskModel) TableName() string { return "tasks" }

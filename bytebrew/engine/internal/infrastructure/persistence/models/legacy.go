package models

// Legacy models used by SQLite-based session persistence (engine.db, work.db).
// These coexist with new PostgreSQL models for config storage.
// Session data (messages, context, tasks) stays in SQLite (per-server, ephemeral).

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Message stores chat messages in SQLite session storage.
type Message struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	SessionID   uuid.UUID      `gorm:"type:uuid;not null;index"`
	MessageType string         `gorm:"type:varchar(50);not null"`
	Sender      string         `gorm:"type:varchar(255)"`
	AgentID     *string        `gorm:"type:varchar(100);index:idx_msg_session_agent"`
	Content     string         `gorm:"type:text;not null"`
	Metadata    datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt   time.Time      `gorm:"not null;default:now();index:idx_session_created"`
	UpdatedAt   time.Time      `gorm:"not null;default:now()"`
}

func (Message) TableName() string { return "message" }

// Task is the pre-pivot Task model for SQLite work storage.
// Not to be confused with TaskModel (PostgreSQL config storage).
type Task struct {
	ID           uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Code         string     `gorm:"type:varchar(50);uniqueIndex;not null"`
	Title        string     `gorm:"type:varchar(500);not null"`
	Description  string     `gorm:"type:text"`
	TaskType     string     `gorm:"type:varchar(50);not null;index"`
	ParentTaskID *uuid.UUID `gorm:"type:uuid;index"`
	SessionID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	Status       string     `gorm:"type:varchar(50);not null;index"`
	CreatedAt    time.Time  `gorm:"not null;default:now()"`
	UpdatedAt    time.Time  `gorm:"not null;default:now()"`

	ParentTask *Task  `gorm:"foreignKey:ParentTaskID"`
	SubTasks   []Task `gorm:"foreignKey:ParentTaskID"`
}

func (Task) TableName() string { return "task" }

// AgentContextSnapshot stores agent context for session resume.
type AgentContextSnapshot struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	SessionID     uuid.UUID `gorm:"type:uuid;not null;index:idx_snap_session_agent"`
	AgentID       string    `gorm:"type:varchar(100);not null;uniqueIndex:idx_snap_agent_unique"`
	FlowType      string    `gorm:"type:varchar(50);not null"`
	SchemaVersion int       `gorm:"not null;default:1"`
	ContextData   []byte    `gorm:"type:blob;not null"`
	StepNumber    int       `gorm:"not null;default:0"`
	TokenCount    int       `gorm:"not null;default:0"`
	Status        string    `gorm:"type:varchar(20);not null;default:'active'"`
	CreatedAt     time.Time `gorm:"not null;default:now()"`
	UpdatedAt     time.Time `gorm:"not null;default:now()"`
}

func (AgentContextSnapshot) TableName() string { return "agent_context_snapshot" }

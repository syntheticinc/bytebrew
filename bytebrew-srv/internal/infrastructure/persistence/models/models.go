package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// ProjectFile represents file metadata with AI-generated description
type ProjectFile struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ProjectID   uuid.UUID `gorm:"type:uuid;not null;index"`
	FilePath    string    `gorm:"type:varchar(500);not null"`
	Description string    `gorm:"type:text;not null"`
	Language    string    `gorm:"type:varchar(50);not null;index"`
	SizeBytes   int64     `gorm:"not null"`
	CreatedAt   time.Time `gorm:"not null;default:now()"`
	UpdatedAt   time.Time `gorm:"not null;default:now()"`

	// Relationships
	Project Project `gorm:"foreignKey:ProjectID"`
}

func (ProjectFile) TableName() string {
	return "project_file"
}

// AgentType represents predefined agent types
type AgentType struct {
	ID           uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Code         string         `gorm:"type:varchar(100);uniqueIndex;not null"`
	Name         string         `gorm:"type:varchar(255);not null"`
	Description  string         `gorm:"type:text"`
	SystemPrompt string         `gorm:"type:text;not null"`
	Tools        datatypes.JSON `gorm:"type:jsonb"`

	// Relationships
	WorkflowSteps []WorkflowStep `gorm:"foreignKey:AgentTypeID"`
}

func (AgentType) TableName() string {
	return "agent_type"
}

// Memory represents Mem0 persistence layer
type Memory struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Content   string         `gorm:"type:text;not null"`
	Level     string         `gorm:"type:varchar(50);not null;index"`
	Metadata  datatypes.JSON `gorm:"type:jsonb"`
	Embedding []float32      `gorm:"type:vector(768)"`
	CreatedAt time.Time      `gorm:"not null;default:now();index"`
	UpdatedAt time.Time      `gorm:"not null;default:now()"`
}

func (Memory) TableName() string {
	return "memory"
}

// ChatSession represents session aggregate root
type ChatSession struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	ProjectID *uuid.UUID `gorm:"type:uuid;index"`
	CreatedAt time.Time  `gorm:"not null;default:now()"`
	UpdatedAt time.Time  `gorm:"not null;default:now()"`

	// Relationships
	User     User      `gorm:"foreignKey:UserID"`
	Project  *Project  `gorm:"foreignKey:ProjectID"`
	Messages []Message `gorm:"foreignKey:SessionID"`
	Tasks    []Task    `gorm:"foreignKey:SessionID"`
}

func (ChatSession) TableName() string {
	return "chat_session"
}

// Message represents a message in a session
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

	// Relationships
	Session ChatSession `gorm:"foreignKey:SessionID"`
}

func (Message) TableName() string {
	return "message"
}

// Task represents a universal task with hierarchy
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

	// Relationships
	ParentTask        *Task              `gorm:"foreignKey:ParentTaskID"`
	SubTasks          []Task             `gorm:"foreignKey:ParentTaskID"`
	Session           ChatSession        `gorm:"foreignKey:SessionID"`
	Artifacts         []TaskArtifact     `gorm:"foreignKey:TaskID"`
	WorkflowExecution *WorkflowExecution `gorm:"foreignKey:TaskID"`
}

func (Task) TableName() string {
	return "task"
}

// TaskArtifact represents structured results from agents
type TaskArtifact struct {
	ID             uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TaskID         uuid.UUID      `gorm:"type:uuid;not null;index"`
	ArtifactType   string         `gorm:"type:varchar(100);not null;index:idx_task_artifact_type"`
	Content        datatypes.JSON `gorm:"type:jsonb;not null"`
	FilePath       string         `gorm:"type:varchar(500)"`
	CreatedByAgent string         `gorm:"type:varchar(100);not null"`
	CreatedAt      time.Time      `gorm:"not null;default:now();index:idx_task_created"`
	UpdatedAt      time.Time      `gorm:"not null;default:now()"`

	// Relationships
	Task Task `gorm:"foreignKey:TaskID"`
}

func (TaskArtifact) TableName() string {
	return "task_artifact"
}

// WorkflowExecution represents execution instance of workflow
type WorkflowExecution struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TaskID       uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	WorkflowType string    `gorm:"type:varchar(100);not null;index"`
	Status       string    `gorm:"type:varchar(50);not null;default:'pending'"`
	CreatedAt    time.Time `gorm:"not null;default:now()"`
	UpdatedAt    time.Time `gorm:"not null;default:now()"`

	// Relationships
	Task  Task           `gorm:"foreignKey:TaskID"`
	Steps []WorkflowStep `gorm:"foreignKey:WorkflowID"`
}

func (WorkflowExecution) TableName() string {
	return "workflow_execution"
}

// WorkflowStep represents individual step in workflow execution
type WorkflowStep struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	WorkflowID  uuid.UUID `gorm:"type:uuid;not null;index"`
	StepNumber  int       `gorm:"not null;uniqueIndex:idx_workflow_step_number"`
	AgentTypeID uuid.UUID `gorm:"type:uuid;not null"`
	Status      string    `gorm:"type:varchar(50);not null;default:'pending';index:idx_workflow_status"`
	CreatedAt   time.Time `gorm:"not null;default:now()"`
	UpdatedAt   time.Time `gorm:"not null;default:now()"`

	// Relationships
	Workflow  WorkflowExecution `gorm:"foreignKey:WorkflowID"`
	AgentType AgentType         `gorm:"foreignKey:AgentTypeID"`
}

func (WorkflowStep) TableName() string {
	return "workflow_step"
}

// AgentContextSnapshot represents a serialized context snapshot for crash recovery and suspend/resume
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

func (AgentContextSnapshot) TableName() string {
	return "agent_context_snapshot"
}

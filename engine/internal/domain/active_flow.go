package domain

import (
	"fmt"
	"time"
)

// ActiveFlow represents an active flow session
type ActiveFlow struct {
	SessionID   string
	ProjectKey  string
	UserID      string
	Task        string
	Status      FlowStatus
	StartedAt   time.Time
	ProjectRoot string // Absolute path to the project directory
	Platform    string // "windows", "linux", "darwin"
}

// FlowStatus represents the status of a flow
type FlowStatus string

const (
	FlowStatusRunning   FlowStatus = "running"
	FlowStatusCompleted FlowStatus = "completed"
	FlowStatusFailed    FlowStatus = "failed"
)

// NewActiveFlow creates a new ActiveFlow with validation
func NewActiveFlow(sessionID, projectKey, userID, task string) (*ActiveFlow, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	if projectKey == "" {
		return nil, fmt.Errorf("project_key is required")
	}
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	return &ActiveFlow{
		SessionID:  sessionID,
		ProjectKey: projectKey,
		UserID:     userID,
		Task:       task,
		Status:     FlowStatusRunning,
		StartedAt:  time.Now(),
	}, nil
}

// MarkComplete marks the flow as completed
func (af *ActiveFlow) MarkComplete() {
	af.Status = FlowStatusCompleted
}

// MarkFailed marks the flow as failed
func (af *ActiveFlow) MarkFailed() {
	af.Status = FlowStatusFailed
}

// IsRunning returns true if flow is running
func (af *ActiveFlow) IsRunning() bool {
	return af.Status == FlowStatusRunning
}

// IsComplete returns true if flow is completed
func (af *ActiveFlow) IsComplete() bool {
	return af.Status == FlowStatusCompleted
}

// IsFailed returns true if flow is failed
func (af *ActiveFlow) IsFailed() bool {
	return af.Status == FlowStatusFailed
}

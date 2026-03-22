package domain

import (
	"fmt"
	"time"
)

// AgentRunStatus represents the lifecycle stage of an agent run
type AgentRunStatus string

const (
	AgentRunRunning   AgentRunStatus = "running"
	AgentRunCompleted AgentRunStatus = "completed"
	AgentRunFailed    AgentRunStatus = "failed"
	AgentRunStopped   AgentRunStatus = "stopped"
)

// AgentRun represents the execution of a code agent working on a subtask
type AgentRun struct {
	ID          string
	SubtaskID   string
	SessionID   string
	FlowType    FlowType
	Status      AgentRunStatus
	Result      string
	Error       string
	StartedAt   time.Time
	CompletedAt *time.Time
}

// NewAgentRun creates a new AgentRun with validation
func NewAgentRun(id, subtaskID, sessionID string, flowType FlowType) (*AgentRun, error) {
	run := &AgentRun{
		ID:        id,
		SubtaskID: subtaskID,
		SessionID: sessionID,
		FlowType:  flowType,
		Status:    AgentRunRunning,
		StartedAt: time.Now(),
	}

	if err := run.Validate(); err != nil {
		return nil, err
	}

	return run, nil
}

// Validate validates the AgentRun
func (r *AgentRun) Validate() error {
	if r.ID == "" {
		return fmt.Errorf("id is required")
	}
	// SubtaskID optional — HTTP chat path may spawn agents without task/subtask context
	if r.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}

	switch r.Status {
	case AgentRunRunning, AgentRunCompleted, AgentRunFailed, AgentRunStopped:
		// Valid
	default:
		return fmt.Errorf("invalid agent run status: %s", r.Status)
	}

	return nil
}

// Complete transitions to completed status
func (r *AgentRun) Complete(result string) {
	r.Status = AgentRunCompleted
	r.Result = result
	now := time.Now()
	r.CompletedAt = &now
}

// Fail transitions to failed status
func (r *AgentRun) Fail(reason string) {
	r.Status = AgentRunFailed
	r.Error = reason
	now := time.Now()
	r.CompletedAt = &now
}

// Stop transitions to stopped status
func (r *AgentRun) Stop() {
	r.Status = AgentRunStopped
	now := time.Now()
	r.CompletedAt = &now
}

// IsTerminal returns true if the agent run is in a terminal state
func (r *AgentRun) IsTerminal() bool {
	return r.Status == AgentRunCompleted ||
		r.Status == AgentRunFailed ||
		r.Status == AgentRunStopped
}

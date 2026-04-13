package lifecycle

import (
	"fmt"
	"time"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// DispatchStatus represents the status of a dispatched task between agents.
type DispatchStatus string

const (
	DispatchPending   DispatchStatus = "pending"
	DispatchRunning   DispatchStatus = "running"
	DispatchCompleted DispatchStatus = "completed"
	DispatchFailed    DispatchStatus = "failed"
	DispatchTimeout   DispatchStatus = "timeout"
)

// Event type constants for task dispatch lifecycle.
// String values are preserved for SSE client compatibility.
const (
	EventTypeTaskDispatched domain.AgentEventType = "task.dispatched"
	EventTypeTaskCompleted  domain.AgentEventType = "task.completed"
	EventTypeTaskFailed     domain.AgentEventType = "task.failed"
	EventTypeTaskTimeout    domain.AgentEventType = "task.timeout"
)

// DispatchRecord represents a task dispatched from a parent agent to a child agent.
type DispatchRecord struct {
	ID          string
	ParentAgent string
	ChildAgent  string
	SessionID   string
	Input       string
	Status      DispatchStatus
	Result      string
	Error       string
	Timeout     time.Duration // 0 = no timeout
	CreatedAt   time.Time
	StartedAt   time.Time
	FinishedAt  time.Time
}

// NewDispatchRecord creates a new DispatchRecord with validation.
func NewDispatchRecord(id, parentAgent, childAgent, sessionID, input string, timeout time.Duration) (*DispatchRecord, error) {
	dr := &DispatchRecord{
		ID:          id,
		ParentAgent: parentAgent,
		ChildAgent:  childAgent,
		SessionID:   sessionID,
		Input:       input,
		Status:      DispatchPending,
		Timeout:     timeout,
		CreatedAt:   time.Now(),
	}
	if err := dr.Validate(); err != nil {
		return nil, err
	}
	return dr, nil
}

// Validate validates the DispatchRecord.
func (dr *DispatchRecord) Validate() error {
	if dr.ID == "" {
		return fmt.Errorf("dispatch record id is required")
	}
	if dr.ParentAgent == "" {
		return fmt.Errorf("dispatch record parent_agent is required")
	}
	if dr.ChildAgent == "" {
		return fmt.Errorf("dispatch record child_agent is required")
	}
	if dr.Input == "" {
		return fmt.Errorf("dispatch record input is required")
	}
	return nil
}

// Start marks the record as running.
func (dr *DispatchRecord) Start() error {
	if dr.Status != DispatchPending {
		return fmt.Errorf("dispatch must be pending to start, current: %s", dr.Status)
	}
	dr.Status = DispatchRunning
	dr.StartedAt = time.Now()
	return nil
}

// Complete marks the record as completed with a result.
func (dr *DispatchRecord) Complete(result string) error {
	if dr.Status != DispatchRunning {
		return fmt.Errorf("dispatch must be running to complete, current: %s", dr.Status)
	}
	dr.Status = DispatchCompleted
	dr.Result = result
	dr.FinishedAt = time.Now()
	return nil
}

// Fail marks the record as failed with an error.
func (dr *DispatchRecord) Fail(errMsg string) error {
	if dr.Status != DispatchRunning && dr.Status != DispatchPending {
		return fmt.Errorf("dispatch must be pending or running to fail, current: %s", dr.Status)
	}
	dr.Status = DispatchFailed
	dr.Error = errMsg
	dr.FinishedAt = time.Now()
	return nil
}

// MarkTimeout marks the record as timed out.
func (dr *DispatchRecord) MarkTimeout() error {
	if dr.Status != DispatchRunning {
		return fmt.Errorf("dispatch must be running to timeout, current: %s", dr.Status)
	}
	dr.Status = DispatchTimeout
	dr.Error = "task timed out"
	dr.FinishedAt = time.Now()
	return nil
}

// IsTerminal returns true if the record is in a terminal state.
func (dr *DispatchRecord) IsTerminal() bool {
	switch dr.Status {
	case DispatchCompleted, DispatchFailed, DispatchTimeout:
		return true
	}
	return false
}

// IsExpired returns true if the record has exceeded its timeout.
func (dr *DispatchRecord) IsExpired() bool {
	if dr.Timeout <= 0 {
		return false
	}
	if dr.StartedAt.IsZero() {
		return false
	}
	return time.Since(dr.StartedAt) > dr.Timeout
}

// NewTaskDispatchedEvent creates a task.dispatched event.
func NewTaskDispatchedEvent(taskID, parentAgent, childAgent string) *domain.AgentEvent {
	return &domain.AgentEvent{
		Type:          EventTypeTaskDispatched,
		SchemaVersion: domain.EventSchemaVersion,
		Timestamp:     time.Now(),
		AgentID:       parentAgent,
		Metadata: map[string]interface{}{
			"task_id":      taskID,
			"parent_agent": parentAgent,
			"child_agent":  childAgent,
		},
	}
}

// NewTaskCompletedEvent creates a task.completed event.
func NewTaskCompletedEvent(taskID, childAgent string) *domain.AgentEvent {
	return &domain.AgentEvent{
		Type:          EventTypeTaskCompleted,
		SchemaVersion: domain.EventSchemaVersion,
		Timestamp:     time.Now(),
		AgentID:       childAgent,
		Metadata: map[string]interface{}{
			"task_id":     taskID,
			"child_agent": childAgent,
		},
	}
}

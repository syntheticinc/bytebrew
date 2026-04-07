package domain

import (
	"fmt"
	"time"
)

// TaskPacketStatus represents the status of a task dispatched between agents.
type TaskPacketStatus string

const (
	TaskPacketPending   TaskPacketStatus = "pending"
	TaskPacketRunning   TaskPacketStatus = "running"
	TaskPacketCompleted TaskPacketStatus = "completed"
	TaskPacketFailed    TaskPacketStatus = "failed"
	TaskPacketTimeout   TaskPacketStatus = "timeout"
)

// TaskPacket represents a task dispatched from a parent agent to a child agent.
type TaskPacket struct {
	ID          string
	ParentAgent string
	ChildAgent  string
	Input       string
	Status      TaskPacketStatus
	Result      string
	Error       string
	Timeout     time.Duration // 0 = no timeout
	CreatedAt   time.Time
	StartedAt   time.Time
	FinishedAt  time.Time
}

// NewTaskPacket creates a new TaskPacket with validation.
func NewTaskPacket(id, parentAgent, childAgent, input string, timeout time.Duration) (*TaskPacket, error) {
	tp := &TaskPacket{
		ID:          id,
		ParentAgent: parentAgent,
		ChildAgent:  childAgent,
		Input:       input,
		Status:      TaskPacketPending,
		Timeout:     timeout,
		CreatedAt:   time.Now(),
	}
	if err := tp.Validate(); err != nil {
		return nil, err
	}
	return tp, nil
}

// Validate validates the TaskPacket.
func (tp *TaskPacket) Validate() error {
	if tp.ID == "" {
		return fmt.Errorf("task packet id is required")
	}
	if tp.ParentAgent == "" {
		return fmt.Errorf("task packet parent_agent is required")
	}
	if tp.ChildAgent == "" {
		return fmt.Errorf("task packet child_agent is required")
	}
	if tp.Input == "" {
		return fmt.Errorf("task packet input is required")
	}
	return nil
}

// Start marks the task as running.
func (tp *TaskPacket) Start() error {
	if tp.Status != TaskPacketPending {
		return fmt.Errorf("task must be pending to start, current: %s", tp.Status)
	}
	tp.Status = TaskPacketRunning
	tp.StartedAt = time.Now()
	return nil
}

// Complete marks the task as completed with a result.
func (tp *TaskPacket) Complete(result string) error {
	if tp.Status != TaskPacketRunning {
		return fmt.Errorf("task must be running to complete, current: %s", tp.Status)
	}
	tp.Status = TaskPacketCompleted
	tp.Result = result
	tp.FinishedAt = time.Now()
	return nil
}

// Fail marks the task as failed with an error.
func (tp *TaskPacket) Fail(errMsg string) error {
	if tp.Status != TaskPacketRunning && tp.Status != TaskPacketPending {
		return fmt.Errorf("task must be pending or running to fail, current: %s", tp.Status)
	}
	tp.Status = TaskPacketFailed
	tp.Error = errMsg
	tp.FinishedAt = time.Now()
	return nil
}

// MarkTimeout marks the task as timed out.
func (tp *TaskPacket) MarkTimeout() error {
	if tp.Status != TaskPacketRunning {
		return fmt.Errorf("task must be running to timeout, current: %s", tp.Status)
	}
	tp.Status = TaskPacketTimeout
	tp.Error = "task timed out"
	tp.FinishedAt = time.Now()
	return nil
}

// IsTerminal returns true if the task is in a terminal state.
func (tp *TaskPacket) IsTerminal() bool {
	switch tp.Status {
	case TaskPacketCompleted, TaskPacketFailed, TaskPacketTimeout:
		return true
	}
	return false
}

// IsExpired returns true if the task has exceeded its timeout.
func (tp *TaskPacket) IsExpired() bool {
	if tp.Timeout <= 0 {
		return false
	}
	if tp.StartedAt.IsZero() {
		return false
	}
	return time.Since(tp.StartedAt) > tp.Timeout
}

// --- Task dispatch event types ---

const (
	EventTypeTaskDispatched AgentEventType = "task.dispatched"
	EventTypeTaskCompleted  AgentEventType = "task.completed"
	EventTypeTaskFailed     AgentEventType = "task.failed"
	EventTypeTaskTimeout    AgentEventType = "task.timeout"
)

// NewTaskDispatchedEvent creates a task.dispatched event.
func NewTaskDispatchedEvent(taskID, parentAgent, childAgent string) *AgentEvent {
	return &AgentEvent{
		Type:          EventTypeTaskDispatched,
		SchemaVersion: EventSchemaVersion,
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
func NewTaskCompletedEvent(taskID, childAgent string) *AgentEvent {
	return &AgentEvent{
		Type:          EventTypeTaskCompleted,
		SchemaVersion: EventSchemaVersion,
		Timestamp:     time.Now(),
		AgentID:       childAgent,
		Metadata: map[string]interface{}{
			"task_id":     taskID,
			"child_agent": childAgent,
		},
	}
}

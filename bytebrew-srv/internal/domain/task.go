package domain

import (
	"fmt"
	"time"
)

// TaskStatus represents the lifecycle stage of a task
type TaskStatus string

const (
	TaskStatusDraft      TaskStatus = "draft"       // Waiting for user approval
	TaskStatusApproved   TaskStatus = "approved"    // Approved, can create subtasks
	TaskStatusInProgress TaskStatus = "in_progress" // Has active subtasks
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

// Task represents a high-level work item that requires user approval and may spawn subtasks
type Task struct {
	ID                 string
	SessionID          string
	Title              string
	Description        string
	AcceptanceCriteria []string
	Status             TaskStatus
	Priority           int // 0 = normal, 1 = high, 2 = critical
	CreatedAt          time.Time
	UpdatedAt          time.Time
	ApprovedAt         *time.Time
	CompletedAt        *time.Time
}

// NewTask creates a new Task with validation
func NewTask(id, sessionID, title, description string, criteria []string) (*Task, error) {
	now := time.Now()
	task := &Task{
		ID:                 id,
		SessionID:          sessionID,
		Title:              title,
		Description:        description,
		AcceptanceCriteria: criteria,
		Status:             TaskStatusDraft,
		Priority:           0, // default = normal
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := task.Validate(); err != nil {
		return nil, err
	}

	return task, nil
}

// Validate validates the Task
func (t *Task) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("id is required")
	}
	if t.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	if t.Title == "" {
		return fmt.Errorf("title is required")
	}

	switch t.Status {
	case TaskStatusDraft, TaskStatusApproved, TaskStatusInProgress,
		TaskStatusCompleted, TaskStatusFailed, TaskStatusCancelled:
		// Valid
	default:
		return fmt.Errorf("invalid task status: %s", t.Status)
	}

	if t.Priority < 0 || t.Priority > 2 {
		return fmt.Errorf("invalid priority: %d (must be 0-2)", t.Priority)
	}

	return nil
}

// Approve transitions from draft to approved
func (t *Task) Approve() error {
	if t.Status != TaskStatusDraft {
		return fmt.Errorf("task must be in draft to approve, current: %s", t.Status)
	}
	now := time.Now()
	t.Status = TaskStatusApproved
	t.ApprovedAt = &now
	t.UpdatedAt = now
	return nil
}

// Start transitions from approved to in_progress
func (t *Task) Start() error {
	if t.Status != TaskStatusApproved {
		return fmt.Errorf("task must be approved to start, current: %s", t.Status)
	}
	t.Status = TaskStatusInProgress
	t.UpdatedAt = time.Now()
	return nil
}

// Complete transitions from in_progress to completed
func (t *Task) Complete() error {
	if t.Status != TaskStatusInProgress {
		return fmt.Errorf("task must be in_progress to complete, current: %s", t.Status)
	}
	now := time.Now()
	t.Status = TaskStatusCompleted
	t.CompletedAt = &now
	t.UpdatedAt = now
	return nil
}

// Fail transitions from in_progress to failed
func (t *Task) Fail() error {
	if t.Status != TaskStatusInProgress {
		return fmt.Errorf("task must be in_progress to fail, current: %s", t.Status)
	}
	t.Status = TaskStatusFailed
	t.UpdatedAt = time.Now()
	return nil
}

// Cancel transitions from any status (except completed) to cancelled
func (t *Task) Cancel() error {
	if t.Status == TaskStatusCompleted {
		return fmt.Errorf("cannot cancel completed task")
	}
	t.Status = TaskStatusCancelled
	t.UpdatedAt = time.Now()
	return nil
}

// SetPriority sets the task priority with validation
func (t *Task) SetPriority(priority int) error {
	if priority < 0 || priority > 2 {
		return fmt.Errorf("invalid priority: %d (must be 0-2)", priority)
	}
	t.Priority = priority
	t.UpdatedAt = time.Now()
	return nil
}

// IsTerminal returns true if the task is in a terminal state
func (t *Task) IsTerminal() bool {
	return t.Status == TaskStatusCompleted ||
		t.Status == TaskStatusFailed ||
		t.Status == TaskStatusCancelled
}

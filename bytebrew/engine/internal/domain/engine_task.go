package domain

import (
	"fmt"
	"time"
)

// TaskSource identifies how an EngineTask was created.
type TaskSource string

const (
	TaskSourceAgent     TaskSource = "agent"
	TaskSourceCron      TaskSource = "cron"
	TaskSourceWebhook   TaskSource = "webhook"
	TaskSourceAPI       TaskSource = "api"
	TaskSourceDashboard TaskSource = "dashboard"
)

// EngineTaskStatus represents the lifecycle stage of an EngineTask.
type EngineTaskStatus string

const (
	EngineTaskStatusPending    EngineTaskStatus = "pending"
	EngineTaskStatusInProgress EngineTaskStatus = "in_progress"
	EngineTaskStatusCompleted  EngineTaskStatus = "completed"
	EngineTaskStatusFailed     EngineTaskStatus = "failed"
	EngineTaskStatusNeedsInput EngineTaskStatus = "needs_input"
	EngineTaskStatusEscalated  EngineTaskStatus = "escalated"
	EngineTaskStatusCancelled  EngineTaskStatus = "cancelled"
)

// TaskMode determines how an EngineTask executes.
type TaskMode string

const (
	TaskModeInteractive TaskMode = "interactive"
	TaskModeBackground  TaskMode = "background"
)

// EngineTask is the universal unit of work in ByteBrew Engine.
// Created by agents, cron triggers, webhooks, API, or dashboard.
type EngineTask struct {
	ID           uint
	Title        string
	Description  string
	AgentName    string
	Source       TaskSource
	SourceID     string
	UserID       string
	SessionID    string
	ParentTaskID *uint
	Depth        int
	Status       EngineTaskStatus
	Mode         TaskMode
	Result       string
	Error        string
	CreatedAt    time.Time
	StartedAt    *time.Time
	CompletedAt  *time.Time
}

// IsTopLevel returns true if the task has no parent.
func (t *EngineTask) IsTopLevel() bool {
	return t.ParentTaskID == nil
}

// IsTerminal returns true if the task is in a terminal state.
func (t *EngineTask) IsTerminal() bool {
	return t.Status == EngineTaskStatusCompleted ||
		t.Status == EngineTaskStatusFailed ||
		t.Status == EngineTaskStatusCancelled
}

// engineTaskValidTransitions defines the state machine for EngineTask status.
var engineTaskValidTransitions = map[EngineTaskStatus][]EngineTaskStatus{
	EngineTaskStatusPending:    {EngineTaskStatusInProgress, EngineTaskStatusCancelled},
	EngineTaskStatusInProgress: {EngineTaskStatusCompleted, EngineTaskStatusFailed, EngineTaskStatusNeedsInput, EngineTaskStatusEscalated, EngineTaskStatusCancelled},
	EngineTaskStatusNeedsInput: {EngineTaskStatusInProgress, EngineTaskStatusCancelled},
	EngineTaskStatusEscalated:  {EngineTaskStatusInProgress, EngineTaskStatusCancelled},
	EngineTaskStatusCompleted:  {},
	EngineTaskStatusFailed:     {},
	EngineTaskStatusCancelled:  {},
}

// CanTransitionTo checks whether a transition to the target status is allowed.
func (t *EngineTask) CanTransitionTo(target EngineTaskStatus) bool {
	allowed, ok := engineTaskValidTransitions[t.Status]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == target {
			return true
		}
	}
	return false
}

// Transition attempts to change the task status with validation and timestamp updates.
func (t *EngineTask) Transition(target EngineTaskStatus) error {
	if !t.CanTransitionTo(target) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, t.Status, target)
	}

	now := time.Now()
	t.Status = target

	switch target {
	case EngineTaskStatusInProgress:
		if t.StartedAt == nil {
			t.StartedAt = &now
		}
	case EngineTaskStatusCompleted, EngineTaskStatusFailed, EngineTaskStatusCancelled:
		t.CompletedAt = &now
	}

	return nil
}

// Domain errors for EngineTask.
var (
	ErrInvalidTransition = fmt.Errorf("invalid status transition")
	ErrEngineTaskNotFound = fmt.Errorf("engine task not found")
)

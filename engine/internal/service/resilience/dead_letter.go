package resilience

import (
	"log/slog"
	"sync"
	"time"
)

// DeadLetterConfig holds configuration for the dead letter queue.
type DeadLetterConfig struct {
	TaskTimeout time.Duration // default 5 minutes
}

// DefaultDeadLetterConfig returns default dead letter configuration.
func DefaultDeadLetterConfig() DeadLetterConfig {
	return DeadLetterConfig{
		TaskTimeout: 5 * time.Minute,
	}
}

// TaskStatus represents the status of a tracked task.
type TaskStatus string

const (
	TaskStatusRunning  TaskStatus = "running"
	TaskStatusComplete TaskStatus = "completed"
	TaskStatusFailed   TaskStatus = "failed"
	TaskStatusTimeout  TaskStatus = "timeout" // AC-RESIL-07
)

// TrackedTask represents a task being monitored for timeout.
type TrackedTask struct {
	TaskID    string
	AgentID   string
	AgentName string
	StartedAt time.Time
	Status    TaskStatus
	Timeout   time.Duration

	// Populated when the task is moved to dead-letter (Status == TaskStatusTimeout).
	Reason    string    // human-readable reason, e.g. "task_timeout"
	MovedAt   time.Time // when the task was moved to dead-letter
	LastError string    // last error message observed (optional)
}

// TimeoutCallback is called when a task times out (AC-RESIL-07).
type TimeoutCallback func(task TrackedTask, elapsed time.Duration)

// DeadLetterQueue tracks tasks and detects timeouts.
// AC-RESIL-07: task timeout → status=timeout, parent gets event
// AC-RESIL-08: dead letter tasks visible in Inspect
type DeadLetterQueue struct {
	mu       sync.RWMutex
	tasks    map[string]*TrackedTask
	config   DeadLetterConfig
	callback TimeoutCallback
}

// NewDeadLetterQueue creates a new dead letter queue.
func NewDeadLetterQueue(config DeadLetterConfig, callback TimeoutCallback) *DeadLetterQueue {
	return &DeadLetterQueue{
		tasks:    make(map[string]*TrackedTask),
		config:   config,
		callback: callback,
	}
}

// Track begins tracking a task for timeout.
func (q *DeadLetterQueue) Track(taskID, agentID string) {
	q.TrackWithName(taskID, agentID, "")
}

// TrackWithName begins tracking a task and records a human-readable agent name.
// Callers that know the agent's display name should prefer this over Track.
func (q *DeadLetterQueue) TrackWithName(taskID, agentID, agentName string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.tasks[taskID] = &TrackedTask{
		TaskID:    taskID,
		AgentID:   agentID,
		AgentName: agentName,
		StartedAt: time.Now(),
		Status:    TaskStatusRunning,
		Timeout:   q.config.TaskTimeout,
	}
}

// TrackWithTimeout begins tracking with a custom timeout.
func (q *DeadLetterQueue) TrackWithTimeout(taskID, agentID string, timeout time.Duration) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.tasks[taskID] = &TrackedTask{
		TaskID:    taskID,
		AgentID:   agentID,
		StartedAt: time.Now(),
		Status:    TaskStatusRunning,
		Timeout:   timeout,
	}
}

// RecordError stores a last-error message for a tracked task so the dead
// letter UI can surface why a task timed out or failed.
func (q *DeadLetterQueue) RecordError(taskID, message string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if task, ok := q.tasks[taskID]; ok {
		task.LastError = message
	}
}

// Complete marks a task as completed.
func (q *DeadLetterQueue) Complete(taskID string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if task, ok := q.tasks[taskID]; ok {
		task.Status = TaskStatusComplete
	}
}

// Fail marks a task as failed.
func (q *DeadLetterQueue) Fail(taskID string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if task, ok := q.tasks[taskID]; ok {
		task.Status = TaskStatusFailed
	}
}

// CheckTimeouts checks all running tasks for timeout.
// Returns tasks that timed out.
func (q *DeadLetterQueue) CheckTimeouts() []TrackedTask {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now()
	var timedOut []TrackedTask

	for _, task := range q.tasks {
		if task.Status != TaskStatusRunning {
			continue
		}
		elapsed := now.Sub(task.StartedAt)
		if elapsed > task.Timeout {
			task.Status = TaskStatusTimeout
			task.MovedAt = now
			if task.Reason == "" {
				task.Reason = "task_timeout"
			}
			timedOut = append(timedOut, *task)

			slog.Warn("[DeadLetter] task timeout",
				"task_id", task.TaskID, "agent_id", task.AgentID,
				"elapsed_ms", elapsed.Milliseconds())

			if q.callback != nil {
				q.callback(*task, elapsed)
			}
		}
	}
	return timedOut
}

// DeadLetters returns all tasks in timeout status (AC-RESIL-08).
func (q *DeadLetterQueue) DeadLetters() []TrackedTask {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var dead []TrackedTask
	for _, task := range q.tasks {
		if task.Status == TaskStatusTimeout {
			dead = append(dead, *task)
		}
	}
	return dead
}

// Remove removes a task from tracking.
func (q *DeadLetterQueue) Remove(taskID string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.tasks, taskID)
}

// RunningCount returns the number of currently running tasks.
func (q *DeadLetterQueue) RunningCount() int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	count := 0
	for _, task := range q.tasks {
		if task.Status == TaskStatusRunning {
			count++
		}
	}
	return count
}

package lifecycle

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// Dispatcher manages task dispatch between parent and child agents.
type Dispatcher struct {
	mu      sync.RWMutex
	tasks   map[string]*DispatchRecord // taskID -> record
	manager *Manager
}

// NewDispatcher creates a new task Dispatcher.
func NewDispatcher(manager *Manager) *Dispatcher {
	return &Dispatcher{
		tasks:   make(map[string]*DispatchRecord),
		manager: manager,
	}
}

// Dispatch creates a task, assigns it to a child agent, and executes it.
// Returns the result when the child completes (blocking).
func (d *Dispatcher) Dispatch(ctx context.Context, taskID, parentAgent, childAgent, sessionID, input string,
	childMode domain.LifecycleMode, maxContext int, timeout time.Duration,
	eventStream domain.AgentEventStream) (*DispatchRecord, error) {

	record, err := NewDispatchRecord(taskID, parentAgent, childAgent, sessionID, input, timeout)
	if err != nil {
		return nil, fmt.Errorf("create dispatch record: %w", err)
	}

	d.mu.Lock()
	d.tasks[taskID] = record
	d.mu.Unlock()

	// Emit task.dispatched event
	if eventStream != nil {
		eventStream.Send(NewTaskDispatchedEvent(taskID, parentAgent, childAgent))
	}

	slog.InfoContext(ctx, "lifecycle: dispatching task", "task_id", taskID, "parent", parentAgent, "child", childAgent)

	// Start the task
	if err := record.Start(); err != nil {
		return record, fmt.Errorf("start task: %w", err)
	}

	// Apply timeout if configured
	execCtx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// Execute the child agent via the lifecycle manager
	output, err := d.manager.ExecuteTask(execCtx, childAgent, taskID, input, childMode, maxContext, eventStream)
	if err != nil {
		// Check if it's a timeout
		if execCtx.Err() == context.DeadlineExceeded {
			if markErr := record.MarkTimeout(); markErr != nil {
				slog.ErrorContext(ctx, "lifecycle: failed to mark timeout", "error", markErr)
			}
			if eventStream != nil {
				eventStream.Send(&domain.AgentEvent{
					Type:          EventTypeTaskTimeout,
					SchemaVersion: domain.EventSchemaVersion,
					Timestamp:     time.Now(),
					AgentID:       childAgent,
					Metadata: map[string]interface{}{
						"task_id":     taskID,
						"child_agent": childAgent,
					},
				})
			}
			return record, fmt.Errorf("task %q timed out", taskID)
		}

		if failErr := record.Fail(err.Error()); failErr != nil {
			slog.ErrorContext(ctx, "lifecycle: failed to mark failure", "error", failErr)
		}
		if eventStream != nil {
			eventStream.Send(&domain.AgentEvent{
				Type:          EventTypeTaskFailed,
				SchemaVersion: domain.EventSchemaVersion,
				Timestamp:     time.Now(),
				AgentID:       childAgent,
				Content:       err.Error(),
				Metadata: map[string]interface{}{
					"task_id":     taskID,
					"child_agent": childAgent,
				},
			})
		}
		return record, fmt.Errorf("task %q failed: %w", taskID, err)
	}

	// Complete the task
	if err := record.Complete(output); err != nil {
		return record, fmt.Errorf("complete task: %w", err)
	}

	// Emit task.completed event
	if eventStream != nil {
		eventStream.Send(NewTaskCompletedEvent(taskID, childAgent))
	}

	slog.InfoContext(ctx, "lifecycle: task completed", "task_id", taskID, "child", childAgent)

	return record, nil
}

// GetTask returns a dispatch record by ID.
func (d *Dispatcher) GetTask(taskID string) (*DispatchRecord, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	dr, ok := d.tasks[taskID]
	return dr, ok
}

// ListTasks returns all dispatch records for a parent agent.
func (d *Dispatcher) ListTasks(parentAgent string) []*DispatchRecord {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var result []*DispatchRecord
	for _, dr := range d.tasks {
		if dr.ParentAgent == parentAgent {
			result = append(result, dr)
		}
	}
	return result
}

// ListTasksBySession returns all dispatch records for a given session.
func (d *Dispatcher) ListTasksBySession(sessionID string) []*DispatchRecord {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var result []*DispatchRecord
	for _, dr := range d.tasks {
		if dr.SessionID == sessionID {
			result = append(result, dr)
		}
	}
	return result
}

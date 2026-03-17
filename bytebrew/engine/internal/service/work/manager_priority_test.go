package work

import (
	"context"
	"testing"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
)

func TestManager_SetTaskPriority(t *testing.T) {
	ctx := context.Background()
	taskStore := &mockTaskStorage{tasks: []*domain.Task{}}
	subtaskStore := &mockSubtaskStorage{}
	manager := New(taskStore, subtaskStore)

	// Create task
	task, err := manager.CreateTask(ctx, "session-1", "Test Task", "Description", nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Set priority to high
	if err := manager.SetTaskPriority(ctx, task.ID, 1); err != nil {
		t.Fatalf("SetTaskPriority failed: %v", err)
	}

	// Verify
	retrieved, err := manager.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if retrieved.Priority != 1 {
		t.Errorf("Priority = %d, want 1", retrieved.Priority)
	}

	// Test invalid priority
	if err := manager.SetTaskPriority(ctx, task.ID, 5); err == nil {
		t.Error("Expected error for invalid priority, got nil")
	}

	// Test non-existent task
	if err := manager.SetTaskPriority(ctx, "non-existent", 1); err == nil {
		t.Error("Expected error for non-existent task, got nil")
	}
}

func TestManager_GetNextTask(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		tasks         []*domain.Task
		wantTaskID    string
		wantNil       bool
	}{
		{
			name: "in_progress task takes precedence",
			tasks: []*domain.Task{
				createApprovedTask("task-1", 0),
				createInProgressTask("task-2", 0),
				createApprovedTask("task-3", 2), // high priority
			},
			wantTaskID: "task-2",
		},
		{
			name: "highest priority approved task",
			tasks: []*domain.Task{
				createApprovedTask("task-1", 0),
				createApprovedTask("task-2", 1),
				createApprovedTask("task-3", 2),
			},
			wantTaskID: "task-3", // priority 2 = critical
		},
		{
			name: "same priority - earliest created",
			tasks: []*domain.Task{
				createApprovedTaskWithTime("task-1", 1, time.Now().Add(-2*time.Hour)),
				createApprovedTaskWithTime("task-2", 1, time.Now().Add(-1*time.Hour)),
			},
			wantTaskID: "task-1",
		},
		{
			name: "no approved or in_progress tasks",
			tasks: []*domain.Task{
				createDraftTask("task-1", 2),
				createCompletedTask("task-2", 1),
			},
			wantNil: true,
		},
		{
			name:    "no tasks",
			tasks:   []*domain.Task{},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskStore := &mockTaskStorage{tasks: tt.tasks}
			subtaskStore := &mockSubtaskStorage{}
			manager := New(taskStore, subtaskStore)

			nextTask, err := manager.GetNextTask(ctx, "session-1")
			if err != nil {
				t.Fatalf("GetNextTask failed: %v", err)
			}

			if tt.wantNil {
				if nextTask != nil {
					t.Errorf("Expected nil, got task %s", nextTask.ID)
				}
			} else {
				if nextTask == nil {
					t.Fatal("Expected task, got nil")
				}
				if nextTask.ID != tt.wantTaskID {
					t.Errorf("Got task %s, want %s", nextTask.ID, tt.wantTaskID)
				}
			}
		})
	}
}

// Helper functions to create tasks with different states

func createDraftTask(id string, priority int) *domain.Task {
	return &domain.Task{
		ID:        id,
		SessionID: "session-1",
		Title:     "Task " + id,
		Status:    domain.TaskStatusDraft,
		Priority:  priority,
		CreatedAt: time.Now(),
	}
}

func createApprovedTask(id string, priority int) *domain.Task {
	return createApprovedTaskWithTime(id, priority, time.Now())
}

func createApprovedTaskWithTime(id string, priority int, createdAt time.Time) *domain.Task {
	return &domain.Task{
		ID:        id,
		SessionID: "session-1",
		Title:     "Task " + id,
		Status:    domain.TaskStatusApproved,
		Priority:  priority,
		CreatedAt: createdAt,
	}
}

func createInProgressTask(id string, priority int) *domain.Task {
	return &domain.Task{
		ID:        id,
		SessionID: "session-1",
		Title:     "Task " + id,
		Status:    domain.TaskStatusInProgress,
		Priority:  priority,
		CreatedAt: time.Now(),
	}
}

func createCompletedTask(id string, priority int) *domain.Task {
	return &domain.Task{
		ID:        id,
		SessionID: "session-1",
		Title:     "Task " + id,
		Status:    domain.TaskStatusCompleted,
		Priority:  priority,
		CreatedAt: time.Now(),
	}
}

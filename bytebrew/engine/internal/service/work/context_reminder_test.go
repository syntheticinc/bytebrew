package work

import (
	"context"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
)

// mockTaskStorage implements TaskStorage for testing
type mockTaskStorage struct {
	tasks []*domain.Task
}

func (m *mockTaskStorage) Save(_ context.Context, task *domain.Task) error {
	m.tasks = append(m.tasks, task)
	return nil
}
func (m *mockTaskStorage) Update(_ context.Context, task *domain.Task) error {
	for i, t := range m.tasks {
		if t.ID == task.ID {
			m.tasks[i] = task
			return nil
		}
	}
	return nil
}
func (m *mockTaskStorage) GetByID(_ context.Context, id string) (*domain.Task, error) {
	for _, t := range m.tasks {
		if t.ID == id {
			return t, nil
		}
	}
	return nil, nil
}
func (m *mockTaskStorage) GetBySessionID(_ context.Context, _ string) ([]*domain.Task, error) {
	return m.tasks, nil
}
func (m *mockTaskStorage) GetByStatus(_ context.Context, _ string, status domain.TaskStatus) ([]*domain.Task, error) {
	var result []*domain.Task
	for _, t := range m.tasks {
		if t.Status == status {
			result = append(result, t)
		}
	}
	return result, nil
}
func (m *mockTaskStorage) GetBySessionIDOrdered(_ context.Context, _ string) ([]*domain.Task, error) {
	sorted := make([]*domain.Task, len(m.tasks))
	copy(sorted, m.tasks)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Priority != sorted[j].Priority {
			return sorted[i].Priority > sorted[j].Priority // DESC
		}
		return sorted[i].CreatedAt.Before(sorted[j].CreatedAt) // ASC
	})
	return sorted, nil
}

// mockSubtaskStorage implements SubtaskStorage for testing
type mockSubtaskStorage struct{}

func (m *mockSubtaskStorage) Save(_ context.Context, _ *domain.Subtask) error     { return nil }
func (m *mockSubtaskStorage) Update(_ context.Context, _ *domain.Subtask) error   { return nil }
func (m *mockSubtaskStorage) GetByID(_ context.Context, _ string) (*domain.Subtask, error) { return nil, nil }
func (m *mockSubtaskStorage) GetByTaskID(_ context.Context, _ string) ([]*domain.Subtask, error) { return nil, nil }
func (m *mockSubtaskStorage) GetBySessionID(_ context.Context, _ string) ([]*domain.Subtask, error) { return nil, nil }
func (m *mockSubtaskStorage) GetReadySubtasks(_ context.Context, _ string) ([]*domain.Subtask, error) { return nil, nil }
func (m *mockSubtaskStorage) GetByAgentID(_ context.Context, _ string) (*domain.Subtask, error) { return nil, nil }

func TestContextReminder_DraftTask_RecentShowsAwaiting(t *testing.T) {
	taskStore := &mockTaskStorage{
		tasks: []*domain.Task{
			{
				ID:        "t1",
				SessionID: "s1",
				Title:     "Test task",
				Status:    domain.TaskStatusDraft,
				CreatedAt: time.Now().Add(-5 * time.Minute), // 5 minutes ago
			},
		},
	}
	mgr := New(taskStore, &mockSubtaskStorage{})
	reminder := NewWorkContextReminder(mgr)

	content, priority, ok := reminder.GetContextReminder(context.Background(), "s1")
	if !ok {
		t.Fatal("expected context reminder")
	}
	if priority != 90 {
		t.Errorf("expected priority 90, got %d", priority)
	}
	if !strings.Contains(content, "Awaiting user approval") {
		t.Errorf("expected 'Awaiting user approval', got: %s", content)
	}
	if strings.Contains(content, "STALE") {
		t.Errorf("should not be STALE after 5 minutes, got: %s", content)
	}
}

func TestContextReminder_DraftTask_OldShowsStale(t *testing.T) {
	taskStore := &mockTaskStorage{
		tasks: []*domain.Task{
			{
				ID:        "t1",
				SessionID: "s1",
				Title:     "Old task",
				Status:    domain.TaskStatusDraft,
				CreatedAt: time.Now().Add(-45 * time.Minute), // 45 minutes ago
			},
		},
	}
	mgr := New(taskStore, &mockSubtaskStorage{})
	reminder := NewWorkContextReminder(mgr)

	content, _, ok := reminder.GetContextReminder(context.Background(), "s1")
	if !ok {
		t.Fatal("expected context reminder")
	}
	if !strings.Contains(content, "STALE") {
		t.Errorf("expected 'STALE' for old draft task, got: %s", content)
	}
}

func TestContextReminder_NonDraftTask_NoStaleMarking(t *testing.T) {
	taskStore := &mockTaskStorage{
		tasks: []*domain.Task{
			{
				ID:        "t1",
				SessionID: "s1",
				Title:     "Active task",
				Status:    domain.TaskStatusInProgress,
				CreatedAt: time.Now().Add(-60 * time.Minute),
			},
		},
	}
	mgr := New(taskStore, &mockSubtaskStorage{})
	reminder := NewWorkContextReminder(mgr)

	content, _, ok := reminder.GetContextReminder(context.Background(), "s1")
	if !ok {
		t.Fatal("expected context reminder")
	}
	if strings.Contains(content, "STALE") {
		t.Errorf("in_progress task should not be marked STALE, got: %s", content)
	}
	if strings.Contains(content, "Awaiting") {
		t.Errorf("in_progress task should not show 'Awaiting', got: %s", content)
	}
}

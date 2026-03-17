package task

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
)

type mockCancelRepo struct {
	tasks    map[uint]*domain.EngineTask
	subTasks map[uint][]domain.EngineTask // parentID -> sub-tasks
	statuses map[uint]domain.EngineTaskStatus
}

func newMockCancelRepo() *mockCancelRepo {
	return &mockCancelRepo{
		tasks:    make(map[uint]*domain.EngineTask),
		subTasks: make(map[uint][]domain.EngineTask),
		statuses: make(map[uint]domain.EngineTaskStatus),
	}
}

func (m *mockCancelRepo) GetByID(_ context.Context, id uint) (*domain.EngineTask, error) {
	task, ok := m.tasks[id]
	if !ok {
		return nil, fmt.Errorf("task %d not found", id)
	}
	// Return status from statuses map if it was updated.
	if s, updated := m.statuses[id]; updated {
		task.Status = s
	}
	return task, nil
}

func (m *mockCancelRepo) UpdateStatus(_ context.Context, id uint, status domain.EngineTaskStatus) error {
	if _, ok := m.tasks[id]; !ok {
		return fmt.Errorf("task %d not found", id)
	}
	m.statuses[id] = status
	return nil
}

func (m *mockCancelRepo) GetSubTasks(_ context.Context, parentID uint) ([]domain.EngineTask, error) {
	return m.subTasks[parentID], nil
}

func TestTaskCanceller_CancelTopLevel(t *testing.T) {
	repo := newMockCancelRepo()
	repo.tasks[1] = &domain.EngineTask{ID: 1, Status: domain.EngineTaskStatusInProgress}

	c := NewTaskCanceller(repo)
	err := c.Cancel(context.Background(), 1)

	require.NoError(t, err)
	assert.Equal(t, domain.EngineTaskStatusCancelled, repo.statuses[1])
}

func TestTaskCanceller_CancelWithSubTasks(t *testing.T) {
	repo := newMockCancelRepo()
	repo.tasks[1] = &domain.EngineTask{ID: 1, Status: domain.EngineTaskStatusInProgress}
	repo.tasks[2] = &domain.EngineTask{ID: 2, Status: domain.EngineTaskStatusPending}
	repo.tasks[3] = &domain.EngineTask{ID: 3, Status: domain.EngineTaskStatusInProgress}
	repo.subTasks[1] = []domain.EngineTask{
		{ID: 2, Status: domain.EngineTaskStatusPending},
		{ID: 3, Status: domain.EngineTaskStatusInProgress},
	}

	c := NewTaskCanceller(repo)
	err := c.Cancel(context.Background(), 1)

	require.NoError(t, err)
	assert.Equal(t, domain.EngineTaskStatusCancelled, repo.statuses[1])
	assert.Equal(t, domain.EngineTaskStatusCancelled, repo.statuses[2])
	assert.Equal(t, domain.EngineTaskStatusCancelled, repo.statuses[3])
}

func TestTaskCanceller_CancelTerminalTask_Noop(t *testing.T) {
	repo := newMockCancelRepo()
	repo.tasks[1] = &domain.EngineTask{ID: 1, Status: domain.EngineTaskStatusCompleted}

	c := NewTaskCanceller(repo)
	err := c.Cancel(context.Background(), 1)

	require.NoError(t, err)
	// No status update should have been made.
	_, updated := repo.statuses[1]
	assert.False(t, updated)
}

func TestTaskCanceller_CancelDeepHierarchy(t *testing.T) {
	repo := newMockCancelRepo()
	repo.tasks[1] = &domain.EngineTask{ID: 1, Status: domain.EngineTaskStatusInProgress}
	repo.tasks[2] = &domain.EngineTask{ID: 2, Status: domain.EngineTaskStatusInProgress}
	repo.tasks[3] = &domain.EngineTask{ID: 3, Status: domain.EngineTaskStatusPending}
	repo.subTasks[1] = []domain.EngineTask{{ID: 2, Status: domain.EngineTaskStatusInProgress}}
	repo.subTasks[2] = []domain.EngineTask{{ID: 3, Status: domain.EngineTaskStatusPending}}

	c := NewTaskCanceller(repo)
	err := c.Cancel(context.Background(), 1)

	require.NoError(t, err)
	assert.Equal(t, domain.EngineTaskStatusCancelled, repo.statuses[1])
	assert.Equal(t, domain.EngineTaskStatusCancelled, repo.statuses[2])
	assert.Equal(t, domain.EngineTaskStatusCancelled, repo.statuses[3])
}

func TestTaskCanceller_CancelSkipsTerminalSubTasks(t *testing.T) {
	repo := newMockCancelRepo()
	repo.tasks[1] = &domain.EngineTask{ID: 1, Status: domain.EngineTaskStatusInProgress}
	repo.tasks[2] = &domain.EngineTask{ID: 2, Status: domain.EngineTaskStatusCompleted}
	repo.tasks[3] = &domain.EngineTask{ID: 3, Status: domain.EngineTaskStatusPending}
	repo.subTasks[1] = []domain.EngineTask{
		{ID: 2, Status: domain.EngineTaskStatusCompleted},
		{ID: 3, Status: domain.EngineTaskStatusPending},
	}

	c := NewTaskCanceller(repo)
	err := c.Cancel(context.Background(), 1)

	require.NoError(t, err)
	assert.Equal(t, domain.EngineTaskStatusCancelled, repo.statuses[1])
	// Completed sub-task should not be touched.
	_, task2Updated := repo.statuses[2]
	assert.False(t, task2Updated)
	// Pending sub-task should be cancelled.
	assert.Equal(t, domain.EngineTaskStatusCancelled, repo.statuses[3])
}

func TestTaskCanceller_NotFound(t *testing.T) {
	repo := newMockCancelRepo()
	c := NewTaskCanceller(repo)

	err := c.Cancel(context.Background(), 999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get task 999")
}

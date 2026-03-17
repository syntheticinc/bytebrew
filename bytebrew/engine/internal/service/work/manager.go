package work

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/google/uuid"
)

// TaskStorage defines persistence operations for tasks (consumer-side)
type TaskStorage interface {
	Save(ctx context.Context, task *domain.Task) error
	Update(ctx context.Context, task *domain.Task) error
	GetByID(ctx context.Context, id string) (*domain.Task, error)
	GetBySessionID(ctx context.Context, sessionID string) ([]*domain.Task, error)
	GetBySessionIDOrdered(ctx context.Context, sessionID string) ([]*domain.Task, error)
	GetByStatus(ctx context.Context, sessionID string, status domain.TaskStatus) ([]*domain.Task, error)
}

// SubtaskStorage defines persistence operations for subtasks (consumer-side)
type SubtaskStorage interface {
	Save(ctx context.Context, subtask *domain.Subtask) error
	Update(ctx context.Context, subtask *domain.Subtask) error
	GetByID(ctx context.Context, id string) (*domain.Subtask, error)
	GetByTaskID(ctx context.Context, taskID string) ([]*domain.Subtask, error)
	GetBySessionID(ctx context.Context, sessionID string) ([]*domain.Subtask, error)
	GetReadySubtasks(ctx context.Context, taskID string) ([]*domain.Subtask, error)
	GetByAgentID(ctx context.Context, agentID string) (*domain.Subtask, error)
}

// Manager handles tasks and subtasks CRUD with business logic
type Manager struct {
	tasks    TaskStorage
	subtasks SubtaskStorage
}

// New creates a new WorkManager
func New(tasks TaskStorage, subtasks SubtaskStorage) *Manager {
	return &Manager{
		tasks:    tasks,
		subtasks: subtasks,
	}
}

// --- Task methods ---

// CreateTask creates a new task in draft status
func (m *Manager) CreateTask(ctx context.Context, sessionID, title, description string, criteria []string) (*domain.Task, error) {
	id := uuid.New().String()[:8]

	task, err := domain.NewTask(id, sessionID, title, description, criteria)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	if err := m.tasks.Save(ctx, task); err != nil {
		return nil, fmt.Errorf("save task: %w", err)
	}

	slog.InfoContext(ctx, "task created", "task_id", id, "title", title)
	return task, nil
}

// ApproveTask transitions task from draft to approved
func (m *Manager) ApproveTask(ctx context.Context, taskID string) error {
	task, err := m.tasks.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if err := task.Approve(); err != nil {
		return fmt.Errorf("approve task: %w", err)
	}

	if err := m.tasks.Update(ctx, task); err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	slog.InfoContext(ctx, "task approved", "task_id", taskID)
	return nil
}

// GetTask retrieves a task by ID
func (m *Manager) GetTask(ctx context.Context, taskID string) (*domain.Task, error) {
	return m.tasks.GetByID(ctx, taskID)
}

// GetTasks retrieves all tasks for a session
func (m *Manager) GetTasks(ctx context.Context, sessionID string) ([]*domain.Task, error) {
	return m.tasks.GetBySessionID(ctx, sessionID)
}

// CompleteTask verifies all subtasks are done and marks task completed
func (m *Manager) CompleteTask(ctx context.Context, taskID string) error {
	task, err := m.tasks.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Verify all subtasks are in terminal state
	subtasks, err := m.subtasks.GetByTaskID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get subtasks: %w", err)
	}

	for _, subtask := range subtasks {
		if !subtask.IsTerminal() {
			return fmt.Errorf("cannot complete task: subtask %s is still %s", subtask.ID, subtask.Status)
		}
	}

	if err := task.Complete(); err != nil {
		return fmt.Errorf("complete task: %w", err)
	}

	if err := m.tasks.Update(ctx, task); err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	slog.InfoContext(ctx, "task completed", "task_id", taskID)
	return nil
}

// FailTask marks task as failed
func (m *Manager) FailTask(ctx context.Context, taskID, reason string) error {
	task, err := m.tasks.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if err := task.Fail(); err != nil {
		return fmt.Errorf("fail task: %w", err)
	}

	if err := m.tasks.Update(ctx, task); err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	slog.InfoContext(ctx, "task failed", "task_id", taskID, "reason", reason)
	return nil
}

// CancelTask marks task as cancelled
func (m *Manager) CancelTask(ctx context.Context, taskID, reason string) error {
	task, err := m.tasks.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if err := task.Cancel(); err != nil {
		return fmt.Errorf("cancel task: %w", err)
	}

	if err := m.tasks.Update(ctx, task); err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	slog.InfoContext(ctx, "task cancelled", "task_id", taskID, "reason", reason)
	return nil
}

// StartTask transitions task from approved to in_progress
func (m *Manager) StartTask(ctx context.Context, taskID string) error {
	task, err := m.tasks.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if err := task.Start(); err != nil {
		return fmt.Errorf("start task: %w", err)
	}

	if err := m.tasks.Update(ctx, task); err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	slog.InfoContext(ctx, "task started", "task_id", taskID)
	return nil
}

// SetTaskPriority sets the priority of a task (0 = normal, 1 = high, 2 = critical)
func (m *Manager) SetTaskPriority(ctx context.Context, taskID string, priority int) error {
	task, err := m.tasks.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if err := task.SetPriority(priority); err != nil {
		return fmt.Errorf("set task priority: %w", err)
	}

	if err := m.tasks.Update(ctx, task); err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	slog.InfoContext(ctx, "task priority set", "task_id", taskID, "priority", priority)
	return nil
}

// GetNextTask retrieves the highest-priority task ready for work
// Priority: in_progress tasks first, then approved by priority DESC, created_at ASC
func (m *Manager) GetNextTask(ctx context.Context, sessionID string) (*domain.Task, error) {
	// First check for in_progress task (resume current work)
	inProgress, err := m.tasks.GetByStatus(ctx, sessionID, domain.TaskStatusInProgress)
	if err != nil {
		return nil, fmt.Errorf("get in_progress tasks: %w", err)
	}
	if len(inProgress) > 0 {
		slog.DebugContext(ctx, "found in_progress task", "task_id", inProgress[0].ID)
		return inProgress[0], nil
	}

	// Get approved tasks ordered by priority DESC, created_at ASC (use DB sorting)
	approved, err := m.tasks.GetByStatus(ctx, sessionID, domain.TaskStatusApproved)
	if err != nil {
		return nil, fmt.Errorf("get approved tasks: %w", err)
	}
	if len(approved) == 0 {
		return nil, nil
	}

	// Use GetBySessionIDOrdered to get tasks with proper SQL sorting
	// Filter to approved status in-memory (DB method returns all statuses)
	allTasks, err := m.tasks.GetBySessionIDOrdered(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get tasks ordered: %w", err)
	}

	// Find first approved task (already sorted by priority DESC, created_at ASC)
	for _, task := range allTasks {
		if task.Status == domain.TaskStatusApproved {
			slog.DebugContext(ctx, "found next approved task", "task_id", task.ID, "priority", task.Priority)
			return task, nil
		}
	}

	return nil, nil
}

// --- Subtask methods ---

// CreateSubtask creates a subtask for a task
func (m *Manager) CreateSubtask(ctx context.Context, sessionID, taskID, title, description string, blockedBy, files []string) (*domain.Subtask, error) {
	id := uuid.New().String()[:8]

	subtask, err := domain.NewTaskSubtask(id, sessionID, taskID, title, description, blockedBy, files)
	if err != nil {
		return nil, fmt.Errorf("create subtask: %w", err)
	}

	if err := m.subtasks.Save(ctx, subtask); err != nil {
		return nil, fmt.Errorf("save subtask: %w", err)
	}

	slog.InfoContext(ctx, "subtask created", "subtask_id", id, "task_id", taskID, "title", title)
	return subtask, nil
}

// GetSubtask retrieves a subtask by ID
func (m *Manager) GetSubtask(ctx context.Context, subtaskID string) (*domain.Subtask, error) {
	return m.subtasks.GetByID(ctx, subtaskID)
}

// GetSubtasksByTask retrieves all subtasks for a task
func (m *Manager) GetSubtasksByTask(ctx context.Context, taskID string) ([]*domain.Subtask, error) {
	return m.subtasks.GetByTaskID(ctx, taskID)
}

// GetReadySubtasks retrieves subtasks ready to execute (pending, no blockers)
func (m *Manager) GetReadySubtasks(ctx context.Context, taskID string) ([]*domain.Subtask, error) {
	return m.subtasks.GetReadySubtasks(ctx, taskID)
}

// AssignSubtaskToAgent assigns a subtask to a Code Agent
func (m *Manager) AssignSubtaskToAgent(ctx context.Context, subtaskID, agentID string) error {
	subtask, err := m.subtasks.GetByID(ctx, subtaskID)
	if err != nil {
		return fmt.Errorf("get subtask: %w", err)
	}
	if subtask == nil {
		return fmt.Errorf("subtask not found: %s", subtaskID)
	}

	if err := subtask.Start(); err != nil {
		return fmt.Errorf("start subtask: %w", err)
	}
	subtask.AssignToAgent(agentID)

	if err := m.subtasks.Update(ctx, subtask); err != nil {
		return fmt.Errorf("update subtask: %w", err)
	}

	slog.InfoContext(ctx, "subtask assigned to agent", "subtask_id", subtaskID, "agent_id", agentID)
	return nil
}

// CompleteSubtask marks a subtask as completed
func (m *Manager) CompleteSubtask(ctx context.Context, subtaskID, result string) error {
	subtask, err := m.subtasks.GetByID(ctx, subtaskID)
	if err != nil {
		return fmt.Errorf("get subtask: %w", err)
	}
	if subtask == nil {
		return fmt.Errorf("subtask not found: %s", subtaskID)
	}

	if err := subtask.Complete(result); err != nil {
		return fmt.Errorf("complete subtask: %w", err)
	}

	if err := m.subtasks.Update(ctx, subtask); err != nil {
		return fmt.Errorf("update subtask: %w", err)
	}

	slog.InfoContext(ctx, "subtask completed", "subtask_id", subtaskID)
	return nil
}

// FailSubtask marks a subtask as failed
func (m *Manager) FailSubtask(ctx context.Context, subtaskID, reason string) error {
	subtask, err := m.subtasks.GetByID(ctx, subtaskID)
	if err != nil {
		return fmt.Errorf("get subtask: %w", err)
	}
	if subtask == nil {
		return fmt.Errorf("subtask not found: %s", subtaskID)
	}

	if err := subtask.Fail(reason); err != nil {
		return fmt.Errorf("fail subtask: %w", err)
	}

	if err := m.subtasks.Update(ctx, subtask); err != nil {
		return fmt.Errorf("update subtask: %w", err)
	}

	slog.InfoContext(ctx, "subtask failed", "subtask_id", subtaskID, "reason", reason)
	return nil
}

// GetSubtaskByAgentID retrieves the active subtask assigned to an agent
func (m *Manager) GetSubtaskByAgentID(ctx context.Context, agentID string) (*domain.Subtask, error) {
	return m.subtasks.GetByAgentID(ctx, agentID)
}

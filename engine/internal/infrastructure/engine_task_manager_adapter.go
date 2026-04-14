package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/configrepo"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/tools"
)

// MaxTaskDepth caps how deep a subtask tree can go.
// Reasonable workflows rarely exceed 5-6 levels; 10 leaves room for unusual cases
// while preventing runaway nesting (which would bloat the reminder context).
const MaxTaskDepth = 10

// EngineTaskManagerAdapter adapts GORMTaskRepository to both:
//   - tools.EngineTaskManager (for agent tools)
//   - agent.SubtaskManager (for AgentPool spawn/complete/fail operations)
//
// This is the single implementation of the unified task manager interface used
// across the codebase after the System 1 → System 2 unification.
//
// All IDs in the public API are uuid.UUID. Conversion from external string IDs
// (agent JSON, HTTP path params) happens at the tool/HTTP boundary, not here.
type EngineTaskManagerAdapter struct {
	repo           *configrepo.GORMTaskRepository
	completionHook *TaskCompletionHook // optional — fires on complete/fail/cancel transitions
}

// NewEngineTaskManagerAdapter creates a new adapter over the given task repository.
func NewEngineTaskManagerAdapter(repo *configrepo.GORMTaskRepository) *EngineTaskManagerAdapter {
	return &EngineTaskManagerAdapter{repo: repo}
}

// SetCompletionHook attaches an optional webhook hook that fires on terminal transitions.
// Safe to call at startup after the adapter is constructed.
func (a *EngineTaskManagerAdapter) SetCompletionHook(hook *TaskCompletionHook) {
	a.completionHook = hook
}

// notifyTerminal is called after a task moves to a terminal state.
// Wrapped in a nil-check so tests and unit scenarios can run without a hook.
func (a *EngineTaskManagerAdapter) notifyTerminal(ctx context.Context, id uuid.UUID) {
	if a.completionHook != nil {
		a.completionHook.OnCompleted(ctx, id)
	}
}

// validateBlockers ensures every id in blockers references an existing task.
// Empty list → OK.
func (a *EngineTaskManagerAdapter) validateBlockers(ctx context.Context, blockers []uuid.UUID) error {
	for _, id := range blockers {
		if id == uuid.Nil {
			return fmt.Errorf("blocked_by contains empty task id")
		}
		if _, err := a.repo.GetByID(ctx, id); err != nil {
			if errors.Is(err, domain.ErrEngineTaskNotFound) {
				return fmt.Errorf("blocked_by references unknown task: %s", id)
			}
			return fmt.Errorf("validate blocker %s: %w", id, err)
		}
	}
	return nil
}

// validateParent fetches the parent, rejects cycles, and returns parent.Depth.
// Called before creating a subtask. Also enforces MaxTaskDepth.
func (a *EngineTaskManagerAdapter) validateParent(ctx context.Context, parentID uuid.UUID) (int, error) {
	if parentID == uuid.Nil {
		return 0, fmt.Errorf("parent_task_id is required for subtask")
	}
	parent, err := a.repo.GetByID(ctx, parentID)
	if err != nil {
		if errors.Is(err, domain.ErrEngineTaskNotFound) {
			return 0, fmt.Errorf("parent task not found: %s", parentID)
		}
		return 0, fmt.Errorf("get parent %s: %w", parentID, err)
	}
	if parent.IsTerminal() {
		return 0, fmt.Errorf("cannot add subtask to terminal task %s (status=%s)", parentID, parent.Status)
	}
	newDepth := parent.Depth + 1
	if newDepth >= MaxTaskDepth {
		return 0, fmt.Errorf("subtask depth %d exceeds maximum %d", newDepth, MaxTaskDepth)
	}
	// Cycle defence: walk up the parent chain, ensuring we don't revisit a node.
	// (Impossible via API today because reparenting is not supported, but protects
	// against direct DB tampering.)
	visited := map[uuid.UUID]bool{parent.ID: true}
	current := parent
	for current.ParentTaskID != nil {
		if visited[*current.ParentTaskID] {
			return 0, fmt.Errorf("parent_task_id cycle detected at %s", *current.ParentTaskID)
		}
		visited[*current.ParentTaskID] = true
		next, err := a.repo.GetByID(ctx, *current.ParentTaskID)
		if err != nil {
			// Broken link — treat as a cycle risk and reject.
			return 0, fmt.Errorf("walk parent chain at %s: %w", *current.ParentTaskID, err)
		}
		current = next
	}
	return newDepth, nil
}

// --- tools.EngineTaskManager ---

func (a *EngineTaskManagerAdapter) CreateTask(ctx context.Context, params tools.CreateEngineTaskParams) (uuid.UUID, error) {
	if params.Title == "" {
		return uuid.Nil, fmt.Errorf("title is required")
	}
	if err := a.validateBlockers(ctx, params.BlockedBy); err != nil {
		return uuid.Nil, err
	}
	status := domain.EngineTaskStatusPending
	if params.RequireApproval {
		status = domain.EngineTaskStatusDraft
	}
	mode := params.Mode
	if mode == "" {
		mode = domain.TaskModeInteractive
	}
	task := &domain.EngineTask{
		ID:                 uuid.New(),
		Title:              params.Title,
		Description:        params.Description,
		AcceptanceCriteria: params.AcceptanceCriteria,
		AgentName:          params.AgentName,
		SessionID:          params.SessionID,
		Source:             domain.TaskSource(params.Source),
		SourceID:           params.SourceID,
		UserID:             params.UserID,
		Priority:           params.Priority,
		BlockedBy:          params.BlockedBy,
		Status:             status,
		Mode:               mode,
		Depth:              0,
	}
	if err := a.repo.Create(ctx, task); err != nil {
		return uuid.Nil, err
	}
	return task.ID, nil
}

// AttachSession records which session is executing the task. Used by the
// autonomous TaskExecutor so admin UI / Inspect UI can trace cron-run events
// back to the task that originated them.
func (a *EngineTaskManagerAdapter) AttachSession(ctx context.Context, taskID uuid.UUID, sessionID string) error {
	task, err := a.repo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task.SessionID == sessionID {
		return nil // already set
	}
	task.SessionID = sessionID
	return a.repo.Update(ctx, task)
}

func (a *EngineTaskManagerAdapter) UpdateTask(ctx context.Context, id uuid.UUID, title, description string) error {
	task, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if title != "" {
		task.Title = title
	}
	if description != "" {
		task.Description = description
	}
	return a.repo.Update(ctx, task)
}

func (a *EngineTaskManagerAdapter) GetTask(ctx context.Context, id uuid.UUID) (*domain.EngineTask, error) {
	return a.repo.GetByID(ctx, id)
}

func (a *EngineTaskManagerAdapter) SetTaskStatus(ctx context.Context, id uuid.UUID, status string, result string) error {
	if err := a.repo.UpdateStatus(ctx, id, domain.EngineTaskStatus(status), result); err != nil {
		return err
	}
	if isTerminalStatus(status) {
		a.notifyTerminal(ctx, id)
	}
	return nil
}

// isTerminalStatus returns true if the string matches a terminal EngineTaskStatus.
func isTerminalStatus(status string) bool {
	switch domain.EngineTaskStatus(status) {
	case domain.EngineTaskStatusCompleted,
		domain.EngineTaskStatusFailed,
		domain.EngineTaskStatusCancelled:
		return true
	}
	return false
}

func (a *EngineTaskManagerAdapter) ListTasks(ctx context.Context, sessionID string) ([]tools.EngineTaskSummary, error) {
	tasks, err := a.repo.List(ctx, configrepo.TaskFilter{SessionID: &sessionID})
	if err != nil {
		return nil, err
	}
	return toTaskSummaries(tasks), nil
}

func (a *EngineTaskManagerAdapter) CreateSubTask(ctx context.Context, parentID uuid.UUID, params tools.CreateEngineTaskParams) (uuid.UUID, error) {
	if params.Title == "" {
		return uuid.Nil, fmt.Errorf("title is required")
	}
	if parentID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("parent task not found: %s", parentID)
	}
	depth, err := a.validateParent(ctx, parentID)
	if err != nil {
		return uuid.Nil, err
	}
	if err := a.validateBlockers(ctx, params.BlockedBy); err != nil {
		return uuid.Nil, err
	}
	status := domain.EngineTaskStatusPending
	if params.RequireApproval {
		status = domain.EngineTaskStatusDraft
	}
	subMode := params.Mode
	if subMode == "" {
		subMode = domain.TaskModeInteractive
	}
	task := &domain.EngineTask{
		ID:                 uuid.New(),
		Title:              params.Title,
		Description:        params.Description,
		AcceptanceCriteria: params.AcceptanceCriteria,
		AgentName:          params.AgentName,
		SessionID:          params.SessionID,
		Source:             domain.TaskSource(params.Source),
		SourceID:           params.SourceID,
		UserID:             params.UserID,
		ParentTaskID:       &parentID,
		Depth:              depth,
		Priority:           params.Priority,
		BlockedBy:          params.BlockedBy,
		Status:             status,
		Mode:               subMode,
	}
	if err := a.repo.Create(ctx, task); err != nil {
		return uuid.Nil, err
	}
	return task.ID, nil
}

func (a *EngineTaskManagerAdapter) ListSubtasks(ctx context.Context, parentID uuid.UUID) ([]tools.EngineTaskSummary, error) {
	tasks, err := a.repo.GetSubTasks(ctx, parentID)
	if err != nil {
		return nil, err
	}
	return toTaskSummaries(tasks), nil
}

func (a *EngineTaskManagerAdapter) ListReadySubtasks(ctx context.Context, parentID uuid.UUID) ([]tools.EngineTaskSummary, error) {
	tasks, err := a.repo.GetReadySubtasks(ctx, parentID)
	if err != nil {
		return nil, err
	}
	return toTaskSummaries(tasks), nil
}

func (a *EngineTaskManagerAdapter) ApproveTask(ctx context.Context, id uuid.UUID) error {
	return a.repo.UpdateStatus(ctx, id, domain.EngineTaskStatusApproved, "")
}

func (a *EngineTaskManagerAdapter) StartTask(ctx context.Context, id uuid.UUID) error {
	return a.repo.UpdateStatus(ctx, id, domain.EngineTaskStatusInProgress, "")
}

func (a *EngineTaskManagerAdapter) CompleteTask(ctx context.Context, id uuid.UUID, result string) error {
	if err := a.repo.UpdateStatus(ctx, id, domain.EngineTaskStatusCompleted, result); err != nil {
		return err
	}
	a.notifyTerminal(ctx, id)
	return nil
}

func (a *EngineTaskManagerAdapter) FailTask(ctx context.Context, id uuid.UUID, reason string) error {
	task, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := task.Fail(reason); err != nil {
		return err
	}
	if err := a.repo.Update(ctx, task); err != nil {
		return err
	}
	a.notifyTerminal(ctx, id)
	return nil
}

func (a *EngineTaskManagerAdapter) CancelTask(ctx context.Context, id uuid.UUID, reason string) error {
	if err := a.repo.Cancel(ctx, id, reason); err != nil {
		return err
	}
	a.notifyTerminal(ctx, id)
	return nil
}

func (a *EngineTaskManagerAdapter) SetTaskPriority(ctx context.Context, id uuid.UUID, priority int) error {
	task, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := task.SetPriority(priority); err != nil {
		return err
	}
	return a.repo.Update(ctx, task)
}

func (a *EngineTaskManagerAdapter) GetNextTask(ctx context.Context, sessionID string) (*domain.EngineTask, error) {
	// 1. In-progress tasks have priority (resume existing work).
	inProgress, err := a.repo.GetByStatus(ctx, sessionID, domain.EngineTaskStatusInProgress)
	if err != nil {
		return nil, err
	}
	if len(inProgress) > 0 {
		return &inProgress[0], nil
	}

	// 2. Approved tasks (passed user approval).
	approved, err := a.repo.GetByStatus(ctx, sessionID, domain.EngineTaskStatusApproved)
	if err != nil {
		return nil, err
	}
	if len(approved) > 0 {
		return &approved[0], nil
	}

	// 3. Pending tasks (auto-approved).
	pending, err := a.repo.GetByStatus(ctx, sessionID, domain.EngineTaskStatusPending)
	if err != nil {
		return nil, err
	}
	if len(pending) > 0 {
		return &pending[0], nil
	}

	return nil, nil
}

func (a *EngineTaskManagerAdapter) AssignTaskToAgent(ctx context.Context, id uuid.UUID, agentID string) error {
	task, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	task.AssignToAgent(agentID)
	// Auto-transition to in_progress if not already running.
	if task.Status == domain.EngineTaskStatusApproved || task.Status == domain.EngineTaskStatusPending {
		if err := task.Start(); err != nil {
			return err
		}
	}
	return a.repo.Update(ctx, task)
}

func (a *EngineTaskManagerAdapter) GetTaskByAgentID(ctx context.Context, agentID string) (*domain.EngineTask, error) {
	return a.repo.GetByAgentID(ctx, agentID)
}

// --- task.ReminderSource (for context reminder) ---

// ListTasksDomain returns all tasks for a session as domain entities.
func (a *EngineTaskManagerAdapter) ListTasksDomain(ctx context.Context, sessionID string) ([]domain.EngineTask, error) {
	return a.repo.GetBySession(ctx, sessionID)
}

// ListSubtasksDomain returns all subtasks for a parent as domain entities.
func (a *EngineTaskManagerAdapter) ListSubtasksDomain(ctx context.Context, parentID uuid.UUID) ([]domain.EngineTask, error) {
	return a.repo.GetSubTasks(ctx, parentID)
}

func toTaskSummaries(tasks []domain.EngineTask) []tools.EngineTaskSummary {
	result := make([]tools.EngineTaskSummary, 0, len(tasks))
	for _, t := range tasks {
		var parentID *string
		if t.ParentTaskID != nil {
			s := t.ParentTaskID.String()
			parentID = &s
		}
		result = append(result, tools.EngineTaskSummary{
			ID:              t.ID.String(),
			Title:           t.Title,
			Status:          string(t.Status),
			AgentName:       t.AgentName,
			ParentID:        parentID,
			Priority:        t.Priority,
			AssignedAgentID: t.AssignedAgentID,
		})
	}
	return result
}

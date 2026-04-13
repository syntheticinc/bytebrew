package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// testTaskManager implements TaskManager for testing (renamed to avoid conflict)
type testTaskManager struct {
	tasks    map[string]*domain.Task
	bySession map[string][]*domain.Task
	createFn func(ctx context.Context, sessionID, title, description string, criteria []string) (*domain.Task, error)
}

func newTestTaskManager() *testTaskManager {
	return &testTaskManager{
		tasks:     make(map[string]*domain.Task),
		bySession: make(map[string][]*domain.Task),
	}
}

func (m *testTaskManager) CreateTask(ctx context.Context, sessionID, title, description string, criteria []string) (*domain.Task, error) {
	if m.createFn != nil {
		return m.createFn(ctx, sessionID, title, description, criteria)
	}
	task := &domain.Task{
		ID:                 "task-001",
		SessionID:          sessionID,
		Title:              title,
		Description:        description,
		Status:             domain.TaskStatusDraft,
		AcceptanceCriteria: criteria,
	}
	m.tasks[task.ID] = task
	m.bySession[sessionID] = append(m.bySession[sessionID], task)
	return task, nil
}

func (m *testTaskManager) ApproveTask(_ context.Context, taskID string) error {
	task, ok := m.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}
	task.Status = domain.TaskStatusApproved
	return nil
}

func (m *testTaskManager) StartTask(_ context.Context, taskID string) error {
	task, ok := m.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}
	task.Status = domain.TaskStatusInProgress
	return nil
}

func (m *testTaskManager) GetTask(_ context.Context, taskID string) (*domain.Task, error) {
	task, ok := m.tasks[taskID]
	if !ok {
		return nil, nil
	}
	return task, nil
}

func (m *testTaskManager) GetTasks(_ context.Context, sessionID string) ([]*domain.Task, error) {
	return m.bySession[sessionID], nil
}

func (m *testTaskManager) CompleteTask(_ context.Context, taskID string) error {
	task, ok := m.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}
	task.Status = domain.TaskStatusCompleted
	return nil
}

func (m *testTaskManager) FailTask(_ context.Context, taskID, reason string) error {
	task, ok := m.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}
	task.Status = domain.TaskStatusFailed
	return nil
}

func (m *testTaskManager) CancelTask(_ context.Context, taskID, reason string) error {
	task, ok := m.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}
	if task.Status == domain.TaskStatusCompleted {
		return fmt.Errorf("cannot cancel completed task")
	}
	task.Status = domain.TaskStatusCancelled
	return nil
}

func (m *testTaskManager) SetTaskPriority(_ context.Context, taskID string, priority int) error {
	task, ok := m.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}
	if priority < 0 || priority > 2 {
		return fmt.Errorf("invalid priority: %d (must be 0-2)", priority)
	}
	task.Priority = priority
	return nil
}

func (m *testTaskManager) GetNextTask(_ context.Context, sessionID string) (*domain.Task, error) {
	tasks := m.bySession[sessionID]
	if len(tasks) == 0 {
		return nil, nil
	}
	// Return first in_progress, or first approved (ordered by priority DESC)
	for _, t := range tasks {
		if t.Status == domain.TaskStatusInProgress {
			return t, nil
		}
	}
	var best *domain.Task
	for _, t := range tasks {
		if t.Status == domain.TaskStatusApproved {
			if best == nil || t.Priority > best.Priority {
				best = t
			}
		}
	}
	return best, nil
}

func makeTaskArgs(t *testing.T, args manageTasksArgs) string {
	t.Helper()
	b, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	return string(b)
}

func TestManageTasks_Create(t *testing.T) {
	mgr := newTestTaskManager()
	tool := NewManageTasksTool(mgr, newMockUserAsker("approved"), "sess-1")

	args := makeTaskArgs(t, manageTasksArgs{
		Action:             "create",
		Title:              "Implement health check",
		Description:        "Add /health endpoint to the API",
		AcceptanceCriteria: []string{"Endpoint returns 200", "Response includes uptime"},
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "created and approved by user") {
		t.Errorf("expected 'created and approved by user' in result, got: %s", result)
	}
	if !strings.Contains(result, "task-001") {
		t.Errorf("expected task ID in result, got: %s", result)
	}

	// Verify task status is approved
	task := mgr.tasks["task-001"]
	if task == nil {
		t.Fatal("task was not created")
	}
	if task.Status != domain.TaskStatusApproved {
		t.Errorf("expected task status approved, got: %s", task.Status)
	}
}

func TestManageTasks_Create_RejectsJSONDescription(t *testing.T) {
	mgr := newTestTaskManager()
	tool := NewManageTasksTool(mgr, newMockUserAsker(), "sess-1")

	args := makeTaskArgs(t, manageTasksArgs{
		Action:      "create",
		Title:       "Test Task",
		Description: `[{"key": "value", "plan": "This is JSON"}]`,
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "[ERROR]") {
		t.Errorf("expected error for JSON description, got: %s", result)
	}
	if !strings.Contains(result, "plain text") {
		t.Errorf("expected 'plain text' in error message, got: %s", result)
	}
	if !strings.Contains(result, "not JSON") {
		t.Errorf("expected 'not JSON' in error message, got: %s", result)
	}

	// Verify CreateTask was NOT called
	if len(mgr.tasks) != 0 {
		t.Errorf("expected no tasks to be created, but found %d", len(mgr.tasks))
	}
}

func TestManageTasks_Create_AllowsPlainTextDescription(t *testing.T) {
	mgr := newTestTaskManager()
	tool := NewManageTasksTool(mgr, newMockUserAsker("approved"), "sess-1")

	plainDescription := `This is a normal plain text description.

## Section
- bullet point
- another bullet

Some more text with code: func main() { return }`

	args := makeTaskArgs(t, manageTasksArgs{
		Action:      "create",
		Title:       "Test Task",
		Description: plainDescription,
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(result, "[ERROR]") {
		t.Errorf("expected success for plain text description, got error: %s", result)
	}
	if !strings.Contains(result, "created and approved by user") {
		t.Errorf("expected 'created and approved by user', got: %s", result)
	}

	// Verify CreateTask WAS called
	if len(mgr.tasks) != 1 {
		t.Errorf("expected 1 task to be created, but found %d", len(mgr.tasks))
	}
	if mgr.tasks["task-001"].Description != plainDescription {
		t.Errorf("expected description to match, got: %q", mgr.tasks["task-001"].Description)
	}
}

func TestManageTasks_Create_MissingTitle(t *testing.T) {
	mgr := newTestTaskManager()
	tool := NewManageTasksTool(mgr, newMockUserAsker(), "sess-1")

	args := makeTaskArgs(t, manageTasksArgs{
		Action:      "create",
		Description: "Some description",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "[ERROR]") || !strings.Contains(result, "title") {
		t.Errorf("expected error about title, got: %s", result)
	}
}

func TestManageTasks_Approve(t *testing.T) {
	mgr := newTestTaskManager()
	mgr.tasks["task-1"] = &domain.Task{
		ID:     "task-1",
		Status: domain.TaskStatusDraft,
	}

	tool := NewManageTasksTool(mgr, newMockUserAsker(), "sess-1")
	args := makeTaskArgs(t, manageTasksArgs{
		Action: "approve",
		TaskID: "task-1",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "approved") {
		t.Errorf("expected 'approved' in result, got: %s", result)
	}

	// Verify state changed
	if mgr.tasks["task-1"].Status != domain.TaskStatusApproved {
		t.Errorf("expected task status approved, got: %s", mgr.tasks["task-1"].Status)
	}
}

func TestManageTasks_Start(t *testing.T) {
	mgr := newTestTaskManager()
	mgr.tasks["task-1"] = &domain.Task{
		ID:     "task-1",
		Status: domain.TaskStatusApproved,
	}

	tool := NewManageTasksTool(mgr, newMockUserAsker(), "sess-1")
	args := makeTaskArgs(t, manageTasksArgs{
		Action: "start",
		TaskID: "task-1",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "started") {
		t.Errorf("expected 'started' in result, got: %s", result)
	}

	if mgr.tasks["task-1"].Status != domain.TaskStatusInProgress {
		t.Errorf("expected task status in_progress, got: %s", mgr.tasks["task-1"].Status)
	}
}

func TestManageTasks_List(t *testing.T) {
	mgr := newTestTaskManager()
	mgr.bySession["sess-1"] = []*domain.Task{
		{ID: "task-1", Title: "First task", Status: domain.TaskStatusDraft},
		{ID: "task-2", Title: "Second task", Status: domain.TaskStatusInProgress},
	}

	tool := NewManageTasksTool(mgr, newMockUserAsker(), "sess-1")
	args := makeTaskArgs(t, manageTasksArgs{
		Action: "list",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "task-1") {
		t.Errorf("expected task-1 in list, got: %s", result)
	}
	if !strings.Contains(result, "task-2") {
		t.Errorf("expected task-2 in list, got: %s", result)
	}
	if !strings.Contains(result, "First task") {
		t.Errorf("expected task title in list, got: %s", result)
	}
}

func TestManageTasks_List_Empty(t *testing.T) {
	mgr := newTestTaskManager()
	tool := NewManageTasksTool(mgr, newMockUserAsker(), "sess-1")

	args := makeTaskArgs(t, manageTasksArgs{
		Action: "list",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "No tasks") {
		t.Errorf("expected 'No tasks' message, got: %s", result)
	}
}

func TestManageTasks_Get(t *testing.T) {
	mgr := newTestTaskManager()
	mgr.tasks["task-1"] = &domain.Task{
		ID:                 "task-1",
		Title:              "Test task",
		Description:        "Detailed description",
		Status:             domain.TaskStatusInProgress,
		AcceptanceCriteria: []string{"Criterion 1", "Criterion 2"},
	}

	tool := NewManageTasksTool(mgr, newMockUserAsker(), "sess-1")
	args := makeTaskArgs(t, manageTasksArgs{
		Action: "get",
		TaskID: "task-1",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "Test task") {
		t.Errorf("expected title in result, got: %s", result)
	}
	if !strings.Contains(result, "Detailed description") {
		t.Errorf("expected description in result, got: %s", result)
	}
}

func TestManageTasks_Get_NotFound(t *testing.T) {
	mgr := newTestTaskManager()
	tool := NewManageTasksTool(mgr, newMockUserAsker(), "sess-1")

	args := makeTaskArgs(t, manageTasksArgs{
		Action: "get",
		TaskID: "nonexistent",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "[ERROR]") || !strings.Contains(result, "not found") {
		t.Errorf("expected not found error, got: %s", result)
	}
}

func TestManageTasks_Complete(t *testing.T) {
	mgr := newTestTaskManager()
	mgr.tasks["task-1"] = &domain.Task{
		ID:     "task-1",
		Status: domain.TaskStatusInProgress,
	}

	tool := NewManageTasksTool(mgr, newMockUserAsker(), "sess-1")
	args := makeTaskArgs(t, manageTasksArgs{
		Action: "complete",
		TaskID: "task-1",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "completed") {
		t.Errorf("expected 'completed' in result, got: %s", result)
	}

	if mgr.tasks["task-1"].Status != domain.TaskStatusCompleted {
		t.Errorf("expected task status completed, got: %s", mgr.tasks["task-1"].Status)
	}
}

func TestManageTasks_Fail(t *testing.T) {
	mgr := newTestTaskManager()
	mgr.tasks["task-1"] = &domain.Task{
		ID:     "task-1",
		Status: domain.TaskStatusInProgress,
	}

	tool := NewManageTasksTool(mgr, newMockUserAsker(), "sess-1")
	args := makeTaskArgs(t, manageTasksArgs{
		Action: "fail",
		TaskID: "task-1",
		Reason: "requirements changed",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "failed") {
		t.Errorf("expected 'failed' in result, got: %s", result)
	}
	if !strings.Contains(result, "requirements changed") {
		t.Errorf("expected reason in result, got: %s", result)
	}

	if mgr.tasks["task-1"].Status != domain.TaskStatusFailed {
		t.Errorf("expected task status failed, got: %s", mgr.tasks["task-1"].Status)
	}
}

func TestManageTasks_Fail_DefaultReason(t *testing.T) {
	mgr := newTestTaskManager()
	mgr.tasks["task-1"] = &domain.Task{
		ID:     "task-1",
		Status: domain.TaskStatusInProgress,
	}

	tool := NewManageTasksTool(mgr, newMockUserAsker(), "sess-1")
	args := makeTaskArgs(t, manageTasksArgs{
		Action: "fail",
		TaskID: "task-1",
		// No Reason — should default to "no reason specified"
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "no reason specified") {
		t.Errorf("expected default reason, got: %s", result)
	}
}

func TestManageTasks_InvalidAction(t *testing.T) {
	mgr := newTestTaskManager()
	tool := NewManageTasksTool(mgr, newMockUserAsker(), "sess-1")

	args := makeTaskArgs(t, manageTasksArgs{
		Action: "invalid_action",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "[ERROR]") || !strings.Contains(result, "Unknown action") {
		t.Errorf("expected unknown action error, got: %s", result)
	}
}

func TestManageTasks_InvalidJSON(t *testing.T) {
	mgr := newTestTaskManager()
	tool := NewManageTasksTool(mgr, newMockUserAsker(), "sess-1")

	result, err := tool.InvokableRun(context.Background(), "not valid json{")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "[ERROR]") || !strings.Contains(result, "Invalid JSON") {
		t.Errorf("expected invalid JSON error, got: %s", result)
	}
}

func TestManageTasks_Info(t *testing.T) {
	mgr := newTestTaskManager()
	tool := NewManageTasksTool(mgr, newMockUserAsker(), "sess-1")

	info, err := tool.Info(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.Name != "manage_tasks" {
		t.Errorf("expected tool name 'manage_tasks', got: %s", info.Name)
	}
	if info.Desc == "" {
		t.Error("expected non-empty description")
	}
}

func TestManageTasks_Cancel(t *testing.T) {
	mgr := newTestTaskManager()
	mgr.tasks["task-1"] = &domain.Task{
		ID:     "task-1",
		Status: domain.TaskStatusDraft,
	}

	tool := NewManageTasksTool(mgr, newMockUserAsker(), "sess-1")
	args := makeTaskArgs(t, manageTasksArgs{
		Action: "cancel",
		TaskID: "task-1",
		Reason: "user changed mind",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "cancelled") {
		t.Errorf("expected 'cancelled' in result, got: %s", result)
	}
	if !strings.Contains(result, "user changed mind") {
		t.Errorf("expected reason in result, got: %s", result)
	}
	if mgr.tasks["task-1"].Status != domain.TaskStatusCancelled {
		t.Errorf("expected status cancelled, got: %s", mgr.tasks["task-1"].Status)
	}
}

func TestManageTasks_Cancel_DefaultReason(t *testing.T) {
	mgr := newTestTaskManager()
	mgr.tasks["task-1"] = &domain.Task{
		ID:     "task-1",
		Status: domain.TaskStatusDraft,
	}

	tool := NewManageTasksTool(mgr, newMockUserAsker(), "sess-1")
	args := makeTaskArgs(t, manageTasksArgs{
		Action: "cancel",
		TaskID: "task-1",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "cancelled by user") {
		t.Errorf("expected default reason, got: %s", result)
	}
}

func TestManageTasks_Cancel_MissingTaskID(t *testing.T) {
	mgr := newTestTaskManager()
	tool := NewManageTasksTool(mgr, newMockUserAsker(), "sess-1")

	args := makeTaskArgs(t, manageTasksArgs{
		Action: "cancel",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "[ERROR]") || !strings.Contains(result, "task_id") {
		t.Errorf("expected error about task_id, got: %s", result)
	}
}

func TestManageTasks_Cancel_NotFound(t *testing.T) {
	mgr := newTestTaskManager()
	tool := NewManageTasksTool(mgr, newMockUserAsker(), "sess-1")

	args := makeTaskArgs(t, manageTasksArgs{
		Action: "cancel",
		TaskID: "nonexistent",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "[ERROR]") {
		t.Errorf("expected error, got: %s", result)
	}
}

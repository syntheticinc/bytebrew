package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
)

// mockSubtaskManager implements SubtaskManager for testing
type mockSubtaskManager struct {
	subtasks map[string]*domain.Subtask
	byTask   map[string][]*domain.Subtask
	ready    map[string][]*domain.Subtask
	createFn func(ctx context.Context, sessionID, taskID, title, description string, blockedBy, files []string) (*domain.Subtask, error)
}

func newMockSubtaskManager() *mockSubtaskManager {
	return &mockSubtaskManager{
		subtasks: make(map[string]*domain.Subtask),
		byTask:   make(map[string][]*domain.Subtask),
		ready:    make(map[string][]*domain.Subtask),
	}
}

func (m *mockSubtaskManager) CreateSubtask(ctx context.Context, sessionID, taskID, title, description string, blockedBy, files []string) (*domain.Subtask, error) {
	if m.createFn != nil {
		return m.createFn(ctx, sessionID, taskID, title, description, blockedBy, files)
	}
	st := &domain.Subtask{
		ID:            "sub-001",
		SessionID:     sessionID,
		TaskID:        taskID,
		Title:         title,
		Description:   description,
		Status:        domain.SubtaskStatusPending,
		BlockedBy:     blockedBy,
		FilesInvolved: files,
	}
	m.subtasks[st.ID] = st
	m.byTask[taskID] = append(m.byTask[taskID], st)
	return st, nil
}

func (m *mockSubtaskManager) GetSubtask(_ context.Context, subtaskID string) (*domain.Subtask, error) {
	st, ok := m.subtasks[subtaskID]
	if !ok {
		return nil, nil
	}
	return st, nil
}

func (m *mockSubtaskManager) GetSubtasksByTask(_ context.Context, taskID string) ([]*domain.Subtask, error) {
	return m.byTask[taskID], nil
}

func (m *mockSubtaskManager) GetReadySubtasks(_ context.Context, taskID string) ([]*domain.Subtask, error) {
	return m.ready[taskID], nil
}

func (m *mockSubtaskManager) CompleteSubtask(_ context.Context, subtaskID, result string) error {
	st, ok := m.subtasks[subtaskID]
	if !ok {
		return fmt.Errorf("subtask not found: %s", subtaskID)
	}
	st.Status = domain.SubtaskStatusCompleted
	st.Result = result
	return nil
}

func (m *mockSubtaskManager) FailSubtask(_ context.Context, subtaskID, reason string) error {
	st, ok := m.subtasks[subtaskID]
	if !ok {
		return fmt.Errorf("subtask not found: %s", subtaskID)
	}
	st.Status = domain.SubtaskStatusFailed
	st.Result = reason
	return nil
}

func makeSubtaskArgs(t *testing.T, args manageSubtasksArgs) string {
	t.Helper()
	b, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	return string(b)
}

func TestManageSubtasks_Create(t *testing.T) {
	mgr := newMockSubtaskManager()
	tool := NewManageSubtasksTool(mgr, "sess-1")

	args := makeSubtaskArgs(t, manageSubtasksArgs{
		Action:      "create",
		TaskID:      "task-1",
		Title:       "Implement feature X",
		Description: "Implement feature X in main.go: add NewFeatureX() function that returns string. Follow existing patterns in handler.go. Acceptance: builds, tests pass.",
		BlockedBy:   []string{"sub-000"},
		Files:       []string{"main.go"},
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "Subtask created") {
		t.Errorf("expected 'Subtask created' in result, got: %s", result)
	}
	if !strings.Contains(result, "sub-001") {
		t.Errorf("expected subtask ID in result, got: %s", result)
	}
	if !strings.Contains(result, "Implement feature X") {
		t.Errorf("expected title in result, got: %s", result)
	}
}

func TestManageSubtasks_Create_MissingTaskID(t *testing.T) {
	mgr := newMockSubtaskManager()
	tool := NewManageSubtasksTool(mgr, "sess-1")

	args := makeSubtaskArgs(t, manageSubtasksArgs{
		Action: "create",
		Title:  "Something",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "[ERROR]") || !strings.Contains(result, "task_id") {
		t.Errorf("expected error about task_id, got: %s", result)
	}
}

func TestManageSubtasks_Create_MissingTitle(t *testing.T) {
	mgr := newMockSubtaskManager()
	tool := NewManageSubtasksTool(mgr, "sess-1")

	args := makeSubtaskArgs(t, manageSubtasksArgs{
		Action: "create",
		TaskID: "task-1",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "[ERROR]") || !strings.Contains(result, "title") {
		t.Errorf("expected error about title, got: %s", result)
	}
}

func TestManageSubtasks_Create_RejectsJSONDescription(t *testing.T) {
	mgr := newMockSubtaskManager()
	tool := NewManageSubtasksTool(mgr, "sess-1")

	args := makeSubtaskArgs(t, manageSubtasksArgs{
		Action:      "create",
		TaskID:      "task-1",
		Title:       "Test Subtask",
		Description: `{"step": "implement", "details": "JSON object instead of text"}`,
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

	// Verify CreateSubtask was NOT called
	if len(mgr.subtasks) != 0 {
		t.Errorf("expected no subtasks to be created, but found %d", len(mgr.subtasks))
	}
}

func TestManageSubtasks_Create_AllowsPlainTextDescription(t *testing.T) {
	mgr := newMockSubtaskManager()
	tool := NewManageSubtasksTool(mgr, "sess-1")

	plainDescription := `Implement authentication middleware in internal/middleware/auth.go.

What: Create AuthMiddleware(next http.Handler) http.Handler function.
Where: internal/middleware/auth.go (new file)
How: Follow pattern in internal/middleware/logging.go.
Acceptance: go build compiles, unit test with mock token passes.`

	args := makeSubtaskArgs(t, manageSubtasksArgs{
		Action:      "create",
		TaskID:      "task-1",
		Title:       "Auth middleware",
		Description: plainDescription,
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(result, "[ERROR]") {
		t.Errorf("expected success for plain text description, got error: %s", result)
	}
	if !strings.Contains(result, "Subtask created") {
		t.Errorf("expected 'Subtask created', got: %s", result)
	}

	// Verify CreateSubtask WAS called
	if len(mgr.subtasks) != 1 {
		t.Errorf("expected 1 subtask to be created, but found %d", len(mgr.subtasks))
	}
	if mgr.subtasks["sub-001"].Description != plainDescription {
		t.Errorf("expected description to match, got: %q", mgr.subtasks["sub-001"].Description)
	}
}

func TestManageSubtasks_Create_EmptyDescription(t *testing.T) {
	mgr := newMockSubtaskManager()
	tool := NewManageSubtasksTool(mgr, "sess-1")

	args := makeSubtaskArgs(t, manageSubtasksArgs{
		Action: "create",
		TaskID: "task-1",
		Title:  "Some task",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "[ERROR]") || !strings.Contains(result, "description is required") {
		t.Errorf("expected error about empty description, got: %s", result)
	}
}

func TestManageSubtasks_Create_ShortDescription(t *testing.T) {
	mgr := newMockSubtaskManager()
	tool := NewManageSubtasksTool(mgr, "sess-1")

	args := makeSubtaskArgs(t, manageSubtasksArgs{
		Action:      "create",
		TaskID:      "task-1",
		Title:       "Some task",
		Description: "Too short",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "[ERROR]") || !strings.Contains(result, "too short") {
		t.Errorf("expected error about short description, got: %s", result)
	}
}

func TestManageSubtasks_Create_DescriptionRepeatsTitle(t *testing.T) {
	mgr := newMockSubtaskManager()
	tool := NewManageSubtasksTool(mgr, "sess-1")

	// Title long enough to pass the 100-char check when used as description
	longTitle := "Implement the complete authentication middleware with JWT validation and token refresh for all API endpoints"
	args := makeSubtaskArgs(t, manageSubtasksArgs{
		Action:      "create",
		TaskID:      "task-1",
		Title:       longTitle,
		Description: longTitle,
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "[ERROR]") || !strings.Contains(result, "repeats the title") {
		t.Errorf("expected error about description repeating title, got: %s", result)
	}
}

func TestManageSubtasks_Create_MissingAcceptanceCriteria(t *testing.T) {
	mgr := newMockSubtaskManager()
	tool := NewManageSubtasksTool(mgr, "sess-1")

	args := makeSubtaskArgs(t, manageSubtasksArgs{
		Action:      "create",
		TaskID:      "task-1",
		Title:       "Add user repository",
		Description: "Create internal/repository/user/repo.go with UserRepository struct. Add Create(ctx, user) error and GetByID(ctx, id) (User, error) methods. Follow pattern in internal/repository/task/repo.go.",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "[ERROR]") || !strings.Contains(result, "acceptance criteria") {
		t.Errorf("expected error about missing acceptance criteria, got: %s", result)
	}
}

func TestManageSubtasks_List(t *testing.T) {
	mgr := newMockSubtaskManager()
	mgr.byTask["task-1"] = []*domain.Subtask{
		{ID: "sub-1", Title: "First subtask", Status: domain.SubtaskStatusPending},
		{ID: "sub-2", Title: "Second subtask", Status: domain.SubtaskStatusInProgress, AssignedAgentID: "code-agent-abc"},
	}

	tool := NewManageSubtasksTool(mgr, "sess-1")
	args := makeSubtaskArgs(t, manageSubtasksArgs{
		Action: "list",
		TaskID: "task-1",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "sub-1") {
		t.Errorf("expected sub-1 in list, got: %s", result)
	}
	if !strings.Contains(result, "sub-2") {
		t.Errorf("expected sub-2 in list, got: %s", result)
	}
	if !strings.Contains(result, "code-agent-abc") {
		t.Errorf("expected agent ID in list, got: %s", result)
	}
}

func TestManageSubtasks_List_Empty(t *testing.T) {
	mgr := newMockSubtaskManager()
	tool := NewManageSubtasksTool(mgr, "sess-1")

	args := makeSubtaskArgs(t, manageSubtasksArgs{
		Action: "list",
		TaskID: "task-1",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "No subtasks") {
		t.Errorf("expected 'No subtasks' message, got: %s", result)
	}
}

func TestManageSubtasks_Get(t *testing.T) {
	mgr := newMockSubtaskManager()
	mgr.subtasks["sub-1"] = &domain.Subtask{
		ID:          "sub-1",
		Title:       "Test subtask",
		Description: "Detailed description",
		Status:      domain.SubtaskStatusInProgress,
		BlockedBy:   []string{"sub-0"},
	}

	tool := NewManageSubtasksTool(mgr, "sess-1")
	args := makeSubtaskArgs(t, manageSubtasksArgs{
		Action:    "get",
		SubtaskID: "sub-1",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "Test subtask") {
		t.Errorf("expected title in result, got: %s", result)
	}
	if !strings.Contains(result, "Detailed description") {
		t.Errorf("expected description in result, got: %s", result)
	}
}

func TestManageSubtasks_Get_NotFound(t *testing.T) {
	mgr := newMockSubtaskManager()
	tool := NewManageSubtasksTool(mgr, "sess-1")

	args := makeSubtaskArgs(t, manageSubtasksArgs{
		Action:    "get",
		SubtaskID: "nonexistent",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "[ERROR]") || !strings.Contains(result, "not found") {
		t.Errorf("expected not found error, got: %s", result)
	}
}

func TestManageSubtasks_GetReady(t *testing.T) {
	mgr := newMockSubtaskManager()
	mgr.ready["task-1"] = []*domain.Subtask{
		{ID: "sub-1", Title: "Ready subtask"},
	}

	tool := NewManageSubtasksTool(mgr, "sess-1")
	args := makeSubtaskArgs(t, manageSubtasksArgs{
		Action: "get_ready",
		TaskID: "task-1",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "Ready subtasks") {
		t.Errorf("expected 'Ready subtasks' header, got: %s", result)
	}
	if !strings.Contains(result, "sub-1") {
		t.Errorf("expected sub-1 in ready list, got: %s", result)
	}
}

func TestManageSubtasks_GetReady_None(t *testing.T) {
	mgr := newMockSubtaskManager()
	tool := NewManageSubtasksTool(mgr, "sess-1")

	args := makeSubtaskArgs(t, manageSubtasksArgs{
		Action: "get_ready",
		TaskID: "task-1",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "No ready subtasks") {
		t.Errorf("expected 'No ready subtasks' message, got: %s", result)
	}
}

func TestManageSubtasks_Complete(t *testing.T) {
	mgr := newMockSubtaskManager()
	mgr.subtasks["sub-1"] = &domain.Subtask{
		ID:     "sub-1",
		Status: domain.SubtaskStatusInProgress,
	}

	tool := NewManageSubtasksTool(mgr, "sess-1")
	args := makeSubtaskArgs(t, manageSubtasksArgs{
		Action:    "complete",
		SubtaskID: "sub-1",
		Result:    "All tests pass",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "completed") {
		t.Errorf("expected 'completed' in result, got: %s", result)
	}

	// Verify state changed
	if mgr.subtasks["sub-1"].Status != domain.SubtaskStatusCompleted {
		t.Errorf("expected subtask status completed, got: %s", mgr.subtasks["sub-1"].Status)
	}
}

func TestManageSubtasks_Complete_DefaultResult(t *testing.T) {
	mgr := newMockSubtaskManager()
	mgr.subtasks["sub-1"] = &domain.Subtask{
		ID:     "sub-1",
		Status: domain.SubtaskStatusInProgress,
	}

	tool := NewManageSubtasksTool(mgr, "sess-1")
	args := makeSubtaskArgs(t, manageSubtasksArgs{
		Action:    "complete",
		SubtaskID: "sub-1",
		// No Result — should default to "completed"
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "completed") {
		t.Errorf("expected 'completed' in result, got: %s", result)
	}
	if mgr.subtasks["sub-1"].Result != "completed" {
		t.Errorf("expected default result 'completed', got: %s", mgr.subtasks["sub-1"].Result)
	}
}

func TestManageSubtasks_Fail(t *testing.T) {
	mgr := newMockSubtaskManager()
	mgr.subtasks["sub-1"] = &domain.Subtask{
		ID:     "sub-1",
		Status: domain.SubtaskStatusInProgress,
	}

	tool := NewManageSubtasksTool(mgr, "sess-1")
	args := makeSubtaskArgs(t, manageSubtasksArgs{
		Action:    "fail",
		SubtaskID: "sub-1",
		Reason:    "compilation error",
	})

	result, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "failed") {
		t.Errorf("expected 'failed' in result, got: %s", result)
	}
	if !strings.Contains(result, "compilation error") {
		t.Errorf("expected reason in result, got: %s", result)
	}

	if mgr.subtasks["sub-1"].Status != domain.SubtaskStatusFailed {
		t.Errorf("expected subtask status failed, got: %s", mgr.subtasks["sub-1"].Status)
	}
}

func TestManageSubtasks_Fail_DefaultReason(t *testing.T) {
	mgr := newMockSubtaskManager()
	mgr.subtasks["sub-1"] = &domain.Subtask{
		ID:     "sub-1",
		Status: domain.SubtaskStatusInProgress,
	}

	tool := NewManageSubtasksTool(mgr, "sess-1")
	args := makeSubtaskArgs(t, manageSubtasksArgs{
		Action:    "fail",
		SubtaskID: "sub-1",
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

func TestManageSubtasks_InvalidAction(t *testing.T) {
	mgr := newMockSubtaskManager()
	tool := NewManageSubtasksTool(mgr, "sess-1")

	args := makeSubtaskArgs(t, manageSubtasksArgs{
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

func TestManageSubtasks_InvalidJSON(t *testing.T) {
	mgr := newMockSubtaskManager()
	tool := NewManageSubtasksTool(mgr, "sess-1")

	result, err := tool.InvokableRun(context.Background(), "not valid json{")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "[ERROR]") || !strings.Contains(result, "Invalid JSON") {
		t.Errorf("expected invalid JSON error, got: %s", result)
	}
}

func TestManageSubtasks_Info(t *testing.T) {
	mgr := newMockSubtaskManager()
	tool := NewManageSubtasksTool(mgr, "sess-1")

	info, err := tool.Info(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.Name != "manage_subtasks" {
		t.Errorf("expected tool name 'manage_subtasks', got: %s", info.Name)
	}
	if info.Desc == "" {
		t.Error("expected non-empty description")
	}
}

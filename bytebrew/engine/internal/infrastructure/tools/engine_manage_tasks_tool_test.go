package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEngineTaskManager implements EngineTaskManager for testing.
type mockEngineTaskManager struct {
	nextID      uint
	tasks       map[uint]CreateEngineTaskParams
	statusCalls []setStatusCall
	updateCalls []updateCall
	listResult  []EngineTaskSummary
	listErr     error
	createErr   error
}

type setStatusCall struct {
	ID     uint
	Status string
	Result string
}

type updateCall struct {
	ID          uint
	Title       string
	Description string
}

func newMockEngineTaskManager() *mockEngineTaskManager {
	return &mockEngineTaskManager{
		nextID: 1,
		tasks:  make(map[uint]CreateEngineTaskParams),
	}
}

func (m *mockEngineTaskManager) CreateTask(_ context.Context, params CreateEngineTaskParams) (uint, error) {
	if m.createErr != nil {
		return 0, m.createErr
	}
	id := m.nextID
	m.nextID++
	m.tasks[id] = params
	return id, nil
}

func (m *mockEngineTaskManager) UpdateTask(_ context.Context, id uint, title, description string) error {
	m.updateCalls = append(m.updateCalls, updateCall{ID: id, Title: title, Description: description})
	return nil
}

func (m *mockEngineTaskManager) SetTaskStatus(_ context.Context, id uint, status string, result string) error {
	m.statusCalls = append(m.statusCalls, setStatusCall{ID: id, Status: status, Result: result})
	return nil
}

func (m *mockEngineTaskManager) ListTasks(_ context.Context, _ string) ([]EngineTaskSummary, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listResult, nil
}

func (m *mockEngineTaskManager) CreateSubTask(_ context.Context, parentID uint, params CreateEngineTaskParams) (uint, error) {
	if m.createErr != nil {
		return 0, m.createErr
	}
	id := m.nextID
	m.nextID++
	m.tasks[id] = params
	return id, nil
}

func TestEngineManageTasksTool_Create_Single(t *testing.T) {
	mgr := newMockEngineTaskManager()
	tl := NewEngineManageTasksTool(mgr, "session-1")

	args, _ := json.Marshal(engineManageTasksArgs{
		Action: "create",
		Tasks:  []engineManageTaskCreate{{Title: "Fix bug", Description: "Fix the login bug"}},
	})
	result, err := tl.InvokableRun(context.Background(), string(args))

	require.NoError(t, err)
	assert.Contains(t, result, "Task created (ID: 1)")
	assert.Equal(t, 1, len(mgr.tasks))
	assert.Equal(t, "Fix bug", mgr.tasks[1].Title)
	assert.Equal(t, "agent", mgr.tasks[1].Source)
	assert.Equal(t, "session-1", mgr.tasks[1].SessionID)
}

func TestEngineManageTasksTool_Create_Multiple(t *testing.T) {
	mgr := newMockEngineTaskManager()
	tl := NewEngineManageTasksTool(mgr, "session-1")

	args, _ := json.Marshal(engineManageTasksArgs{
		Action: "create",
		Tasks: []engineManageTaskCreate{
			{Title: "Task A"},
			{Title: "Task B", Description: "Desc B"},
		},
	})
	result, err := tl.InvokableRun(context.Background(), string(args))

	require.NoError(t, err)
	assert.Contains(t, result, "2 tasks created")
	assert.Contains(t, result, "Task A")
	assert.Contains(t, result, "Task B")
	assert.Equal(t, 2, len(mgr.tasks))
}

func TestEngineManageTasksTool_Create_EmptyTasks(t *testing.T) {
	mgr := newMockEngineTaskManager()
	tl := NewEngineManageTasksTool(mgr, "session-1")

	args, _ := json.Marshal(engineManageTasksArgs{Action: "create", Tasks: nil})
	result, err := tl.InvokableRun(context.Background(), string(args))

	require.NoError(t, err)
	assert.Contains(t, result, "[ERROR]")
	assert.Contains(t, result, "tasks array is required")
}

func TestEngineManageTasksTool_Create_MissingTitle(t *testing.T) {
	mgr := newMockEngineTaskManager()
	tl := NewEngineManageTasksTool(mgr, "session-1")

	args, _ := json.Marshal(engineManageTasksArgs{
		Action: "create",
		Tasks:  []engineManageTaskCreate{{Title: "", Description: "no title"}},
	})
	result, err := tl.InvokableRun(context.Background(), string(args))

	require.NoError(t, err)
	assert.Contains(t, result, "[ERROR]")
	assert.Contains(t, result, "must have a title")
}

func TestEngineManageTasksTool_Update(t *testing.T) {
	mgr := newMockEngineTaskManager()
	tl := NewEngineManageTasksTool(mgr, "session-1")

	args, _ := json.Marshal(engineManageTasksArgs{
		Action:      "update",
		TaskID:      5,
		Title:       "New Title",
		Description: "New Desc",
	})
	result, err := tl.InvokableRun(context.Background(), string(args))

	require.NoError(t, err)
	assert.Contains(t, result, "Task 5 updated")
	require.Len(t, mgr.updateCalls, 1)
	assert.Equal(t, uint(5), mgr.updateCalls[0].ID)
	assert.Equal(t, "New Title", mgr.updateCalls[0].Title)
}

func TestEngineManageTasksTool_Update_NoID(t *testing.T) {
	mgr := newMockEngineTaskManager()
	tl := NewEngineManageTasksTool(mgr, "session-1")

	args, _ := json.Marshal(engineManageTasksArgs{Action: "update", Title: "Something"})
	result, err := tl.InvokableRun(context.Background(), string(args))

	require.NoError(t, err)
	assert.Contains(t, result, "[ERROR]")
	assert.Contains(t, result, "task_id is required")
}

func TestEngineManageTasksTool_Update_NoFields(t *testing.T) {
	mgr := newMockEngineTaskManager()
	tl := NewEngineManageTasksTool(mgr, "session-1")

	args, _ := json.Marshal(engineManageTasksArgs{Action: "update", TaskID: 1})
	result, err := tl.InvokableRun(context.Background(), string(args))

	require.NoError(t, err)
	assert.Contains(t, result, "[ERROR]")
	assert.Contains(t, result, "at least one of title or description")
}

func TestEngineManageTasksTool_SetStatus(t *testing.T) {
	mgr := newMockEngineTaskManager()
	tl := NewEngineManageTasksTool(mgr, "session-1")

	args, _ := json.Marshal(engineManageTasksArgs{
		Action: "set_status",
		TaskID: 3,
		Status: "completed",
		Result: "All done",
	})
	result, err := tl.InvokableRun(context.Background(), string(args))

	require.NoError(t, err)
	assert.Contains(t, result, "Task 3 status set to")
	require.Len(t, mgr.statusCalls, 1)
	assert.Equal(t, "completed", mgr.statusCalls[0].Status)
	assert.Equal(t, "All done", mgr.statusCalls[0].Result)
}

func TestEngineManageTasksTool_SetStatus_NoStatus(t *testing.T) {
	mgr := newMockEngineTaskManager()
	tl := NewEngineManageTasksTool(mgr, "session-1")

	args, _ := json.Marshal(engineManageTasksArgs{Action: "set_status", TaskID: 1})
	result, err := tl.InvokableRun(context.Background(), string(args))

	require.NoError(t, err)
	assert.Contains(t, result, "[ERROR]")
	assert.Contains(t, result, "status is required")
}

func TestEngineManageTasksTool_List(t *testing.T) {
	parentID := uint(1)
	mgr := newMockEngineTaskManager()
	mgr.listResult = []EngineTaskSummary{
		{ID: 1, Title: "Task 1", Status: "pending", AgentName: "supervisor"},
		{ID: 2, Title: "Task 2", Status: "completed", AgentName: "coder", ParentID: &parentID},
	}
	tl := NewEngineManageTasksTool(mgr, "session-1")

	args, _ := json.Marshal(engineManageTasksArgs{Action: "list"})
	result, err := tl.InvokableRun(context.Background(), string(args))

	require.NoError(t, err)
	assert.Contains(t, result, "Tasks (2)")
	assert.Contains(t, result, "Task 1")
	assert.Contains(t, result, "Task 2")
	assert.Contains(t, result, "parent: 1")
}

func TestEngineManageTasksTool_List_Empty(t *testing.T) {
	mgr := newMockEngineTaskManager()
	mgr.listResult = nil
	tl := NewEngineManageTasksTool(mgr, "session-1")

	args, _ := json.Marshal(engineManageTasksArgs{Action: "list"})
	result, err := tl.InvokableRun(context.Background(), string(args))

	require.NoError(t, err)
	assert.Contains(t, result, "No tasks found")
}

func TestEngineManageTasksTool_CreateSubtask(t *testing.T) {
	mgr := newMockEngineTaskManager()
	tl := NewEngineManageTasksTool(mgr, "session-1")

	args, _ := json.Marshal(engineManageTasksArgs{
		Action:       "create_subtask",
		ParentTaskID: 10,
		Title:        "Sub task",
		Description:  "Sub desc",
	})
	result, err := tl.InvokableRun(context.Background(), string(args))

	require.NoError(t, err)
	assert.Contains(t, result, "Sub-task created (ID: 1, parent: 10)")
}

func TestEngineManageTasksTool_CreateSubtask_NoParent(t *testing.T) {
	mgr := newMockEngineTaskManager()
	tl := NewEngineManageTasksTool(mgr, "session-1")

	args, _ := json.Marshal(engineManageTasksArgs{Action: "create_subtask", Title: "Sub"})
	result, err := tl.InvokableRun(context.Background(), string(args))

	require.NoError(t, err)
	assert.Contains(t, result, "[ERROR]")
	assert.Contains(t, result, "parent_task_id is required")
}

func TestEngineManageTasksTool_CreateSubtask_NoTitle(t *testing.T) {
	mgr := newMockEngineTaskManager()
	tl := NewEngineManageTasksTool(mgr, "session-1")

	args, _ := json.Marshal(engineManageTasksArgs{Action: "create_subtask", ParentTaskID: 1})
	result, err := tl.InvokableRun(context.Background(), string(args))

	require.NoError(t, err)
	assert.Contains(t, result, "[ERROR]")
	assert.Contains(t, result, "title is required")
}

func TestEngineManageTasksTool_UnknownAction(t *testing.T) {
	mgr := newMockEngineTaskManager()
	tl := NewEngineManageTasksTool(mgr, "session-1")

	args, _ := json.Marshal(engineManageTasksArgs{Action: "destroy"})
	result, err := tl.InvokableRun(context.Background(), string(args))

	require.NoError(t, err)
	assert.Contains(t, result, "[ERROR]")
	assert.Contains(t, result, "Unknown action")
}

func TestEngineManageTasksTool_InvalidJSON(t *testing.T) {
	mgr := newMockEngineTaskManager()
	tl := NewEngineManageTasksTool(mgr, "session-1")

	result, err := tl.InvokableRun(context.Background(), "not json at all")

	require.NoError(t, err)
	assert.Contains(t, result, "[ERROR]")
	assert.Contains(t, result, "Invalid JSON")
}

func TestEngineManageTasksTool_CreateError(t *testing.T) {
	mgr := newMockEngineTaskManager()
	mgr.createErr = fmt.Errorf("db connection lost")
	tl := NewEngineManageTasksTool(mgr, "session-1")

	args, _ := json.Marshal(engineManageTasksArgs{
		Action: "create",
		Tasks:  []engineManageTaskCreate{{Title: "Fail"}},
	})
	result, err := tl.InvokableRun(context.Background(), string(args))

	require.NoError(t, err)
	assert.Contains(t, result, "[ERROR]")
	assert.Contains(t, result, "db connection lost")
}

func TestEngineManageTasksTool_ListError(t *testing.T) {
	mgr := newMockEngineTaskManager()
	mgr.listErr = fmt.Errorf("query failed")
	tl := NewEngineManageTasksTool(mgr, "session-1")

	args, _ := json.Marshal(engineManageTasksArgs{Action: "list"})
	result, err := tl.InvokableRun(context.Background(), string(args))

	require.NoError(t, err)
	assert.Contains(t, result, "[ERROR]")
	assert.Contains(t, result, "query failed")
}

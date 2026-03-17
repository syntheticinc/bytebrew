package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTaskManager is a mock TaskManager that records calls
type mockTaskManager struct {
	tasks       map[string]*domain.Task
	sessionID   string
	createCount int
}

func newMockTaskManager() *mockTaskManager {
	return &mockTaskManager{
		tasks: make(map[string]*domain.Task),
	}
}

func (m *mockTaskManager) CreateTask(_ context.Context, sessionID, title, description string, criteria []string) (*domain.Task, error) {
	m.createCount++
	m.sessionID = sessionID
	task, err := domain.NewTask("task-1", sessionID, title, description, criteria)
	if err != nil {
		return nil, err
	}
	m.tasks[task.ID] = task
	return task, nil
}

func (m *mockTaskManager) ApproveTask(_ context.Context, taskID string) error {
	task := m.tasks[taskID]
	if task == nil {
		return nil
	}
	return task.Approve()
}

func (m *mockTaskManager) StartTask(_ context.Context, taskID string) error {
	task := m.tasks[taskID]
	if task == nil {
		return nil
	}
	return task.Start()
}

func (m *mockTaskManager) GetTask(_ context.Context, taskID string) (*domain.Task, error) {
	return m.tasks[taskID], nil
}

func (m *mockTaskManager) GetTasks(_ context.Context, _ string) ([]*domain.Task, error) {
	var result []*domain.Task
	for _, t := range m.tasks {
		result = append(result, t)
	}
	return result, nil
}

func (m *mockTaskManager) CompleteTask(_ context.Context, taskID string) error {
	task := m.tasks[taskID]
	if task == nil {
		return nil
	}
	return task.Complete()
}

func (m *mockTaskManager) CancelTask(_ context.Context, taskID, _ string) error {
	task := m.tasks[taskID]
	if task == nil {
		return nil
	}
	return task.Cancel()
}

func (m *mockTaskManager) FailTask(_ context.Context, taskID, _ string) error {
	task := m.tasks[taskID]
	if task == nil {
		return nil
	}
	return task.Fail()
}

func (m *mockTaskManager) SetTaskPriority(_ context.Context, taskID string, priority int) error {
	task := m.tasks[taskID]
	if task == nil {
		return nil
	}
	return task.SetPriority(priority)
}

func (m *mockTaskManager) GetNextTask(_ context.Context, sessionID string) (*domain.Task, error) {
	for _, task := range m.tasks {
		if task.Status == domain.TaskStatusApproved || task.Status == domain.TaskStatusInProgress {
			return task, nil
		}
	}
	return nil, nil
}

// mockUserAsker simulates user responses for ask_user tool
type mockUserAsker struct {
	responses      []string // queue of responses to return
	questionsAsked []string // record of questionsJSON received
	callIndex      int
}

func newMockUserAsker(responses ...string) *mockUserAsker {
	return &mockUserAsker{responses: responses}
}

func (m *mockUserAsker) AskUserQuestionnaire(_ context.Context, _, questionsJSON string) (string, error) {
	m.questionsAsked = append(m.questionsAsked, questionsJSON)

	// Determine the raw answer to wrap
	rawAnswer := "approved"
	if m.callIndex < len(m.responses) {
		rawAnswer = m.responses[m.callIndex]
		m.callIndex++
	}

	// Parse questions to get text for building QuestionAnswer response
	var questions []Question
	if err := json.Unmarshal([]byte(questionsJSON), &questions); err != nil {
		// fallback: return raw answer wrapped in JSON array
		answers := []QuestionAnswer{{Question: "unknown", Answer: rawAnswer}}
		data, _ := json.Marshal(answers)
		return string(data), nil
	}

	// Build QuestionAnswer array matching questions
	var answers []QuestionAnswer
	for _, q := range questions {
		answers = append(answers, QuestionAnswer{Question: q.Text, Answer: rawAnswer})
	}
	data, _ := json.Marshal(answers)
	return string(data), nil
}

// --- ask_user tool ---

func TestAskUser_PassesQuestionsToProxy(t *testing.T) {
	asker := newMockUserAsker("approved")
	askTool := NewAskUserTool(asker, "session-1")
	ctx := context.Background()

	args := `{"questions":[{"text":"# Task: Health Check\n\nDo you approve?","options":[{"label":"approved"},{"label":"rejected"}],"default":"approved"}]}`

	result, err := askTool.InvokableRun(ctx, args)
	require.NoError(t, err)

	assert.Contains(t, result, "approved")
	require.Len(t, asker.questionsAsked, 1)
	assert.Contains(t, asker.questionsAsked[0], "Health Check")
}

func TestAskUser_ReturnsUserFeedback(t *testing.T) {
	asker := newMockUserAsker("change step 3 to use PostgreSQL")
	askTool := NewAskUserTool(asker, "session-1")
	ctx := context.Background()

	result, err := askTool.InvokableRun(ctx, `{"questions":[{"text":"Do you approve?"}]}`)
	require.NoError(t, err)

	assert.Contains(t, result, "change step 3 to use PostgreSQL")
}

func TestAskUser_Workflow_EmptyQuestionsReturnsError(t *testing.T) {
	asker := newMockUserAsker()
	askTool := NewAskUserTool(asker, "session-1")
	ctx := context.Background()

	result, err := askTool.InvokableRun(ctx, `{"questions":[]}`)
	require.NoError(t, err)
	assert.Contains(t, result, "[ERROR]")
	assert.Empty(t, asker.questionsAsked, "should not call proxy on empty questions")
}

func TestAskUser_QuestionWithDefault(t *testing.T) {
	asker := newMockUserAsker("approved")
	askTool := NewAskUserTool(asker, "session-1")
	ctx := context.Background()

	args := `{"questions":[{"text":"Do you approve?","default":"approved"}]}`
	result, err := askTool.InvokableRun(ctx, args)
	require.NoError(t, err)
	assert.Contains(t, result, "approved")

	// Verify proxy received the default in questions JSON
	require.Len(t, asker.questionsAsked, 1)
	var questions []Question
	require.NoError(t, json.Unmarshal([]byte(asker.questionsAsked[0]), &questions))
	require.Len(t, questions, 1)
	assert.Equal(t, "approved", questions[0].Default)
}

func TestAskUser_QuestionWithoutDefault(t *testing.T) {
	asker := newMockUserAsker("yes")
	askTool := NewAskUserTool(asker, "session-1")
	ctx := context.Background()

	args := `{"questions":[{"text":"What do you think?"}]}`
	result, err := askTool.InvokableRun(ctx, args)
	require.NoError(t, err)
	assert.Contains(t, result, "yes")

	// Verify proxy received question without default
	require.Len(t, asker.questionsAsked, 1)
	var questions []Question
	require.NoError(t, json.Unmarshal([]byte(asker.questionsAsked[0]), &questions))
	require.Len(t, questions, 1)
	assert.Equal(t, "", questions[0].Default)
}

// --- Full supervisor workflow: create (auto-asks user) ---

func TestSupervisorWorkflow_CreateAutoApproves(t *testing.T) {
	ctx := context.Background()
	manager := newMockTaskManager()
	asker := newMockUserAsker("approved")

	tasksTool := NewManageTasksTool(manager, asker, "session-1")

	// Create task (internally calls ask_user and auto-approves)
	createArgs := manageTasksArgs{
		Action:             "create",
		Title:              "As a developer, I want a health check endpoint",
		Description:        "Add GET /health returning 200 OK",
		AcceptanceCriteria: []string{"GET /health returns 200"},
	}
	createJSON, _ := json.Marshal(createArgs)
	createResult, err := tasksTool.InvokableRun(ctx, string(createJSON))
	require.NoError(t, err)

	// Should return "created and approved"
	assert.Contains(t, createResult, "created and approved by user")
	assert.Contains(t, createResult, "task-1")

	// Verify task was created and approved
	task := manager.tasks["task-1"]
	require.NotNil(t, task)
	assert.Equal(t, domain.TaskStatusApproved, task.Status)

	// Verify ask_user was called with task content
	require.Len(t, asker.questionsAsked, 1)
	assert.Contains(t, asker.questionsAsked[0], "# Task:")
	assert.Contains(t, asker.questionsAsked[0], "health check endpoint")
	assert.Contains(t, asker.questionsAsked[0], "Do you approve this Task?")
}

func TestSupervisorWorkflow_CreateWithFeedback_Revise(t *testing.T) {
	ctx := context.Background()
	manager := &multiCreateTaskManager{tasks: make(map[string]*domain.Task)}

	// Round 1: User gives feedback (not approved)
	asker1 := newMockUserAsker("change step 3 to use PostgreSQL")
	tasksTool1 := NewManageTasksTool(manager, asker1, "session-1")

	createArgs1, _ := json.Marshal(manageTasksArgs{
		Action:             "create",
		Title:              "Add database layer",
		Description:        "Add SQLite for persistence",
		AcceptanceCriteria: []string{"Data persists"},
	})
	createResult1, err := tasksTool1.InvokableRun(ctx, string(createArgs1))
	require.NoError(t, err)

	// Should return "NOT approved" with feedback
	assert.Contains(t, createResult1, "NOT approved")
	assert.Contains(t, createResult1, "change step 3 to use PostgreSQL")

	// Round 2: User approves revised task
	asker2 := newMockUserAsker("approved")
	tasksTool2 := NewManageTasksTool(manager, asker2, "session-1")

	createArgs2, _ := json.Marshal(manageTasksArgs{
		Action:             "create",
		Title:              "Add database layer (PostgreSQL)",
		Description:        "Add PostgreSQL for persistence",
		AcceptanceCriteria: []string{"Data persists in PostgreSQL"},
	})
	createResult2, err := tasksTool2.InvokableRun(ctx, string(createArgs2))
	require.NoError(t, err)

	// Should return "created and approved"
	assert.Contains(t, createResult2, "created and approved by user")

	// Verify: two tasks created
	assert.Equal(t, 2, manager.createCount)
	// First task should be cancelled (user feedback)
	task1 := manager.tasks["task-x"]
	assert.Equal(t, domain.TaskStatusCancelled, task1.Status)
	// Second task should be approved
	task2 := manager.tasks["task-xx"]
	assert.Equal(t, domain.TaskStatusApproved, task2.Status)
}

func TestSupervisorWorkflow_QuestionContainsFullContent(t *testing.T) {
	ctx := context.Background()
	manager := newMockTaskManager()
	asker := newMockUserAsker("approved")

	tasksTool := NewManageTasksTool(manager, asker, "session-1")

	// Create task with rich description
	longDescription := `Business Context: Our monitoring system needs to verify the server is alive.
Currently there's no health check endpoint.

Current State: Server is defined in cmd/server/main.go.
Routes are registered in internal/delivery/grpc/router.go.
No health endpoint exists.

Technical Plan:
1. Add HealthHandler in internal/delivery/grpc/health_handler.go
2. Register route in router.go
3. Handler returns {"status": "ok", "version": "..."} with 200

Dependencies: None — new endpoint, doesn't affect existing routes.
Out of Scope: Readiness checks, dependency health.`

	createArgs, _ := json.Marshal(manageTasksArgs{
		Action:      "create",
		Title:       "As a developer, I want a /health endpoint, so that monitoring can verify the server is running",
		Description: longDescription,
		AcceptanceCriteria: []string{
			"Given server is running, when GET /health is called, then response is 200 with status=ok",
			"Given server is running, when GET /health is called, then response includes version field",
		},
	})
	createResult, err := tasksTool.InvokableRun(ctx, string(createArgs))
	require.NoError(t, err)

	// Should return "created and approved"
	assert.Contains(t, createResult, "created and approved by user")

	// Verify asker received the FULL content via questionnaire JSON
	require.Len(t, asker.questionsAsked, 1)
	var questions []Question
	require.NoError(t, json.Unmarshal([]byte(asker.questionsAsked[0]), &questions))
	require.Len(t, questions, 1)

	questionText := questions[0].Text
	assert.Contains(t, questionText, "# Task:")
	assert.Contains(t, questionText, "monitoring system")
	assert.Contains(t, questionText, "Technical Plan")
	assert.Contains(t, questionText, "HealthHandler")
	assert.Contains(t, questionText, "GET /health")
	assert.Contains(t, questionText, "version field")
	assert.Contains(t, questionText, "Do you approve this Task?")

	// Question text should start with markdown heading (task content, not raw JSON)
	assert.True(t, strings.HasPrefix(questionText, "# Task:"), "question text should start with markdown heading")
}

// --- Helpers ---

// multiCreateTaskManager generates unique IDs for each create call
type multiCreateTaskManager struct {
	tasks       map[string]*domain.Task
	createCount int
}

func (m *multiCreateTaskManager) CreateTask(_ context.Context, sessionID, title, description string, criteria []string) (*domain.Task, error) {
	m.createCount++
	id := "task-" + strings.Repeat("x", m.createCount) // unique per call
	task, err := domain.NewTask(id, sessionID, title, description, criteria)
	if err != nil {
		return nil, err
	}
	m.tasks[task.ID] = task
	return task, nil
}

func (m *multiCreateTaskManager) ApproveTask(_ context.Context, taskID string) error {
	if t, ok := m.tasks[taskID]; ok {
		return t.Approve()
	}
	return nil
}

func (m *multiCreateTaskManager) StartTask(_ context.Context, taskID string) error {
	if t, ok := m.tasks[taskID]; ok {
		return t.Start()
	}
	return nil
}

func (m *multiCreateTaskManager) GetTask(_ context.Context, taskID string) (*domain.Task, error) {
	return m.tasks[taskID], nil
}

func (m *multiCreateTaskManager) GetTasks(_ context.Context, _ string) ([]*domain.Task, error) {
	var result []*domain.Task
	for _, t := range m.tasks {
		result = append(result, t)
	}
	return result, nil
}

func (m *multiCreateTaskManager) CompleteTask(_ context.Context, taskID string) error {
	if t, ok := m.tasks[taskID]; ok {
		return t.Complete()
	}
	return nil
}

func (m *multiCreateTaskManager) FailTask(_ context.Context, taskID, _ string) error {
	if t, ok := m.tasks[taskID]; ok {
		return t.Fail()
	}
	return nil
}

func (m *multiCreateTaskManager) CancelTask(_ context.Context, taskID, _ string) error {
	if t, ok := m.tasks[taskID]; ok {
		return t.Cancel()
	}
	return nil
}

func (m *multiCreateTaskManager) SetTaskPriority(_ context.Context, taskID string, priority int) error {
	if t, ok := m.tasks[taskID]; ok {
		return t.SetPriority(priority)
	}
	return nil
}

func (m *multiCreateTaskManager) GetNextTask(_ context.Context, sessionID string) (*domain.Task, error) {
	for _, task := range m.tasks {
		if task.Status == domain.TaskStatusApproved || task.Status == domain.TaskStatusInProgress {
			return task, nil
		}
	}
	return nil, nil
}

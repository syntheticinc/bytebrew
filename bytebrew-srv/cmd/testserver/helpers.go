package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/config"
)

// noopAgentService implements AgentService interface for FlowHandler
type noopAgentService struct{}

func (n *noopAgentService) SetEnvironmentContext(projectRoot, platform string) {}
func (n *noopAgentService) SetTestingStrategy(yamlContent string)              {}

// mockSnapshotRepo implements SnapshotRepository interface
type mockSnapshotRepo struct {
	snapshots map[string]*domain.AgentContextSnapshot
}

func newMockSnapshotRepo() *mockSnapshotRepo {
	return &mockSnapshotRepo{
		snapshots: make(map[string]*domain.AgentContextSnapshot),
	}
}

func (m *mockSnapshotRepo) Save(ctx context.Context, snapshot *domain.AgentContextSnapshot) error {
	m.snapshots[snapshot.AgentID] = snapshot
	return nil
}

func (m *mockSnapshotRepo) Load(ctx context.Context, sessionID, agentID string) (*domain.AgentContextSnapshot, error) {
	snap, exists := m.snapshots[agentID]
	if !exists {
		return nil, nil
	}
	return snap, nil
}

func (m *mockSnapshotRepo) Delete(ctx context.Context, sessionID, agentID string) error {
	delete(m.snapshots, agentID)
	return nil
}

func (m *mockSnapshotRepo) FindActive(ctx context.Context) ([]*domain.AgentContextSnapshot, error) {
	return nil, nil
}

// mockHistoryRepo implements MessageRepository interface
type mockHistoryRepo struct {
	messages []*domain.Message
}

func newMockHistoryRepo() *mockHistoryRepo {
	return &mockHistoryRepo{
		messages: make([]*domain.Message, 0),
	}
}

func (m *mockHistoryRepo) Create(ctx context.Context, message *domain.Message) error {
	m.messages = append(m.messages, message)
	return nil
}

// mockSubtaskManager implements tools.SubtaskManager interface for testing
type mockSubtaskManager struct {
	subtasks map[string]*domain.Subtask
}

func newMockSubtaskManager() *mockSubtaskManager {
	return &mockSubtaskManager{
		subtasks: make(map[string]*domain.Subtask),
	}
}

func (m *mockSubtaskManager) CreateSubtask(ctx context.Context, sessionID, taskID, title, description string, blockedBy, files []string) (*domain.Subtask, error) {
	id := fmt.Sprintf("subtask-%d", len(m.subtasks)+1)
	subtask := &domain.Subtask{
		ID:            id,
		SessionID:     sessionID,
		TaskID:        taskID,
		Title:         title,
		Description:   description,
		Status:        domain.SubtaskStatusPending,
		BlockedBy:     blockedBy,
		FilesInvolved: files,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	m.subtasks[id] = subtask
	return subtask, nil
}

func (m *mockSubtaskManager) GetSubtask(ctx context.Context, subtaskID string) (*domain.Subtask, error) {
	subtask, exists := m.subtasks[subtaskID]
	if !exists {
		return nil, nil
	}
	return subtask, nil
}

func (m *mockSubtaskManager) GetSubtasksByTask(ctx context.Context, taskID string) ([]*domain.Subtask, error) {
	var result []*domain.Subtask
	for _, s := range m.subtasks {
		if s.TaskID == taskID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockSubtaskManager) GetReadySubtasks(ctx context.Context, taskID string) ([]*domain.Subtask, error) {
	var result []*domain.Subtask
	for _, s := range m.subtasks {
		if s.TaskID == taskID && s.Status == domain.SubtaskStatusPending {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockSubtaskManager) CompleteSubtask(ctx context.Context, subtaskID, resultText string) error {
	subtask, exists := m.subtasks[subtaskID]
	if !exists {
		return fmt.Errorf("subtask not found: %s", subtaskID)
	}
	subtask.Status = domain.SubtaskStatusCompleted
	subtask.Result = resultText
	now := time.Now()
	subtask.CompletedAt = &now
	subtask.UpdatedAt = now
	return nil
}

func (m *mockSubtaskManager) FailSubtask(ctx context.Context, subtaskID, reason string) error {
	subtask, exists := m.subtasks[subtaskID]
	if !exists {
		return fmt.Errorf("subtask not found: %s", subtaskID)
	}
	subtask.Status = domain.SubtaskStatusFailed
	subtask.Result = reason
	subtask.UpdatedAt = time.Now()
	return nil
}

func (m *mockSubtaskManager) AssignSubtaskToAgent(ctx context.Context, subtaskID, agentID string) error {
	subtask, exists := m.subtasks[subtaskID]
	if !exists {
		return fmt.Errorf("subtask not found: %s", subtaskID)
	}
	subtask.AssignedAgentID = agentID
	subtask.UpdatedAt = time.Now()
	return nil
}

// mockTaskManager implements tools.TaskManager interface for testing
type mockTaskManager struct {
	tasks  map[string]*domain.Task
	nextID int
}

func newMockTaskManager() *mockTaskManager {
	return &mockTaskManager{
		tasks: make(map[string]*domain.Task),
	}
}

func (m *mockTaskManager) CreateTask(ctx context.Context, sessionID, title, description string, criteria []string) (*domain.Task, error) {
	m.nextID++
	id := fmt.Sprintf("task-%d", m.nextID)
	task, err := domain.NewTask(id, sessionID, title, description, criteria)
	if err != nil {
		return nil, err
	}
	m.tasks[id] = task
	return task, nil
}

func (m *mockTaskManager) ApproveTask(ctx context.Context, taskID string) error {
	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}
	return task.Approve()
}

func (m *mockTaskManager) StartTask(ctx context.Context, taskID string) error {
	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}
	return task.Start()
}

func (m *mockTaskManager) GetTask(ctx context.Context, taskID string) (*domain.Task, error) {
	task, exists := m.tasks[taskID]
	if !exists {
		return nil, nil
	}
	return task, nil
}

func (m *mockTaskManager) GetTasks(ctx context.Context, sessionID string) ([]*domain.Task, error) {
	var result []*domain.Task
	for _, t := range m.tasks {
		if t.SessionID == sessionID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *mockTaskManager) CompleteTask(ctx context.Context, taskID string) error {
	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}
	return task.Complete()
}

func (m *mockTaskManager) FailTask(ctx context.Context, taskID, reason string) error {
	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}
	return task.Fail()
}

func (m *mockTaskManager) CancelTask(ctx context.Context, taskID, reason string) error {
	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}
	return task.Cancel()
}

func (m *mockTaskManager) SetTaskPriority(ctx context.Context, taskID string, priority int) error {
	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}
	return task.SetPriority(priority)
}

func (m *mockTaskManager) GetNextTask(ctx context.Context, sessionID string) (*domain.Task, error) {
	for _, t := range m.tasks {
		if t.SessionID == sessionID && (t.Status == domain.TaskStatusApproved || t.Status == domain.TaskStatusInProgress) {
			return t, nil
		}
	}
	return nil, nil
}

// mockAgentRunStorage implements AgentRunStorage interface for testing
type mockAgentRunStorage struct {
	runs map[string]*domain.AgentRun
	mu   sync.Mutex
}

func newMockAgentRunStorage() *mockAgentRunStorage {
	return &mockAgentRunStorage{runs: make(map[string]*domain.AgentRun)}
}

func (m *mockAgentRunStorage) Save(ctx context.Context, run *domain.AgentRun) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runs[run.ID] = run
	return nil
}

func (m *mockAgentRunStorage) Update(ctx context.Context, run *domain.AgentRun) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runs[run.ID] = run
	return nil
}

func (m *mockAgentRunStorage) GetByID(ctx context.Context, id string) (*domain.AgentRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.runs[id], nil
}

func (m *mockAgentRunStorage) GetRunningBySession(ctx context.Context, sessionID string) ([]*domain.AgentRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*domain.AgentRun
	for _, r := range m.runs {
		if r.SessionID == sessionID && r.Status == domain.AgentRunRunning {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *mockAgentRunStorage) CountRunningBySession(ctx context.Context, sessionID string) (int, error) {
	runs, _ := m.GetRunningBySession(ctx, sessionID)
	return len(runs), nil
}

func (m *mockAgentRunStorage) CleanupOrphanedRuns(ctx context.Context) (int64, error) {
	return 0, nil
}

// testFlowConfig returns programmatic FlowsConfig and PromptsConfig for testing
func testFlowConfig() (*config.FlowsConfig, *config.PromptsConfig) {
	flowsCfg := &config.FlowsConfig{
		Flows: map[string]config.FlowDefinition{
			"supervisor": {
				Name:            "supervisor-flow",
				SystemPromptRef: "supervisor_prompt",
				Tools: []string{
					"manage_subtasks", "manage_tasks",
					"read_file", "write_file", "edit_file",
					"search_code", "get_project_tree", "smart_search", "grep_search", "glob",
					"execute_command", "ask_user",
					"spawn_code_agent",
					"lsp",
				},
				MaxSteps:        10,
				MaxContextSize:  4000,
				Lifecycle: config.LifecycleConfig{
					SuspendOn: []string{},
					ReportTo:  "user",
				},
			},
			"coder": {
				Name:            "coder-flow",
				SystemPromptRef: "code_agent_prompt",
				Tools: []string{
					"read_file", "write_file", "edit_file",
					"search_code", "get_project_tree",
					"execute_command",
				},
				MaxSteps:        10,
				MaxContextSize:  4000,
				Lifecycle: config.LifecycleConfig{
					SuspendOn: []string{},
					ReportTo:  "parent_agent",
				},
			},
		},
	}

	promptsCfg := &config.PromptsConfig{
		SupervisorPrompt: "You are a test supervisor. Follow instructions exactly.",
		SystemPrompt:     "You are a helpful assistant.",
		CodeAgentPrompt:  "You are a code agent. Complete the assigned subtask.",
	}

	return flowsCfg, promptsCfg
}

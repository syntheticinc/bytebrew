package testutil

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/config"
)

// NoopAgentService implements AgentService interface for FlowHandler
type NoopAgentService struct{}

func (n *NoopAgentService) SetEnvironmentContext(projectRoot, platform string) {}
func (n *NoopAgentService) SetTestingStrategy(yamlContent string)              {}

// MockSnapshotRepo implements SnapshotRepository interface
type MockSnapshotRepo struct {
	Snapshots map[string]*domain.AgentContextSnapshot
}

func NewMockSnapshotRepo() *MockSnapshotRepo {
	return &MockSnapshotRepo{
		Snapshots: make(map[string]*domain.AgentContextSnapshot),
	}
}

func (m *MockSnapshotRepo) Save(ctx context.Context, snapshot *domain.AgentContextSnapshot) error {
	m.Snapshots[snapshot.AgentID] = snapshot
	return nil
}

func (m *MockSnapshotRepo) Load(ctx context.Context, sessionID, agentID string) (*domain.AgentContextSnapshot, error) {
	snap, exists := m.Snapshots[agentID]
	if !exists {
		return nil, nil
	}
	return snap, nil
}

func (m *MockSnapshotRepo) Delete(ctx context.Context, sessionID, agentID string) error {
	delete(m.Snapshots, agentID)
	return nil
}

func (m *MockSnapshotRepo) FindActive(ctx context.Context) ([]*domain.AgentContextSnapshot, error) {
	return nil, nil
}

// MockHistoryRepo implements MessageRepository interface
type MockHistoryRepo struct {
	Messages []*domain.Message
}

func NewMockHistoryRepo() *MockHistoryRepo {
	return &MockHistoryRepo{
		Messages: make([]*domain.Message, 0),
	}
}

func (m *MockHistoryRepo) Create(ctx context.Context, message *domain.Message) error {
	m.Messages = append(m.Messages, message)
	return nil
}

// MockSubtaskManager implements tools.SubtaskManager interface for testing
type MockSubtaskManager struct {
	Subtasks map[string]*domain.Subtask
}

func NewMockSubtaskManager() *MockSubtaskManager {
	return &MockSubtaskManager{
		Subtasks: make(map[string]*domain.Subtask),
	}
}

func (m *MockSubtaskManager) CreateSubtask(ctx context.Context, sessionID, taskID, title, description string, blockedBy, files []string) (*domain.Subtask, error) {
	id := fmt.Sprintf("subtask-%d", len(m.Subtasks)+1)
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
	m.Subtasks[id] = subtask
	return subtask, nil
}

func (m *MockSubtaskManager) GetSubtask(ctx context.Context, subtaskID string) (*domain.Subtask, error) {
	subtask, exists := m.Subtasks[subtaskID]
	if !exists {
		return nil, nil
	}
	return subtask, nil
}

func (m *MockSubtaskManager) GetSubtasksByTask(ctx context.Context, taskID string) ([]*domain.Subtask, error) {
	var result []*domain.Subtask
	for _, s := range m.Subtasks {
		if s.TaskID == taskID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *MockSubtaskManager) GetReadySubtasks(ctx context.Context, taskID string) ([]*domain.Subtask, error) {
	var result []*domain.Subtask
	for _, s := range m.Subtasks {
		if s.TaskID == taskID && s.Status == domain.SubtaskStatusPending {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *MockSubtaskManager) CompleteSubtask(ctx context.Context, subtaskID, resultText string) error {
	subtask, exists := m.Subtasks[subtaskID]
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

func (m *MockSubtaskManager) FailSubtask(ctx context.Context, subtaskID, reason string) error {
	subtask, exists := m.Subtasks[subtaskID]
	if !exists {
		return fmt.Errorf("subtask not found: %s", subtaskID)
	}
	subtask.Status = domain.SubtaskStatusFailed
	subtask.Result = reason
	subtask.UpdatedAt = time.Now()
	return nil
}

func (m *MockSubtaskManager) AssignSubtaskToAgent(ctx context.Context, subtaskID, agentID string) error {
	subtask, exists := m.Subtasks[subtaskID]
	if !exists {
		return fmt.Errorf("subtask not found: %s", subtaskID)
	}
	subtask.AssignedAgentID = agentID
	subtask.UpdatedAt = time.Now()
	return nil
}

// MockTaskManager implements tools.TaskManager interface for testing
type MockTaskManager struct {
	Tasks  map[string]*domain.Task
	nextID int
}

func NewMockTaskManager() *MockTaskManager {
	return &MockTaskManager{
		Tasks: make(map[string]*domain.Task),
	}
}

func (m *MockTaskManager) CreateTask(ctx context.Context, sessionID, title, description string, criteria []string) (*domain.Task, error) {
	m.nextID++
	id := fmt.Sprintf("task-%d", m.nextID)
	task, err := domain.NewTask(id, sessionID, title, description, criteria)
	if err != nil {
		return nil, err
	}
	m.Tasks[id] = task
	return task, nil
}

func (m *MockTaskManager) ApproveTask(ctx context.Context, taskID string) error {
	task, exists := m.Tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}
	return task.Approve()
}

func (m *MockTaskManager) StartTask(ctx context.Context, taskID string) error {
	task, exists := m.Tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}
	return task.Start()
}

func (m *MockTaskManager) GetTask(ctx context.Context, taskID string) (*domain.Task, error) {
	task, exists := m.Tasks[taskID]
	if !exists {
		return nil, nil
	}
	return task, nil
}

func (m *MockTaskManager) GetTasks(ctx context.Context, sessionID string) ([]*domain.Task, error) {
	var result []*domain.Task
	for _, t := range m.Tasks {
		if t.SessionID == sessionID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *MockTaskManager) CompleteTask(ctx context.Context, taskID string) error {
	task, exists := m.Tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}
	return task.Complete()
}

func (m *MockTaskManager) FailTask(ctx context.Context, taskID, reason string) error {
	task, exists := m.Tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}
	return task.Fail()
}

func (m *MockTaskManager) CancelTask(ctx context.Context, taskID, reason string) error {
	task, exists := m.Tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}
	return task.Cancel()
}

func (m *MockTaskManager) SetTaskPriority(ctx context.Context, taskID string, priority int) error {
	task, exists := m.Tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}
	return task.SetPriority(priority)
}

func (m *MockTaskManager) GetNextTask(ctx context.Context, sessionID string) (*domain.Task, error) {
	for _, t := range m.Tasks {
		if t.SessionID == sessionID && (t.Status == domain.TaskStatusApproved || t.Status == domain.TaskStatusInProgress) {
			return t, nil
		}
	}
	return nil, nil
}

// MockAgentRunStorage implements AgentRunStorage interface for testing
type MockAgentRunStorage struct {
	Runs map[string]*domain.AgentRun
	mu   sync.Mutex
}

func NewMockAgentRunStorage() *MockAgentRunStorage {
	return &MockAgentRunStorage{Runs: make(map[string]*domain.AgentRun)}
}

func (m *MockAgentRunStorage) Save(ctx context.Context, run *domain.AgentRun) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Runs[run.ID] = run
	return nil
}

func (m *MockAgentRunStorage) Update(ctx context.Context, run *domain.AgentRun) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Runs[run.ID] = run
	return nil
}

func (m *MockAgentRunStorage) GetByID(ctx context.Context, id string) (*domain.AgentRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Runs[id], nil
}

func (m *MockAgentRunStorage) GetRunningBySession(ctx context.Context, sessionID string) ([]*domain.AgentRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*domain.AgentRun
	for _, r := range m.Runs {
		if r.SessionID == sessionID && r.Status == domain.AgentRunRunning {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *MockAgentRunStorage) CountRunningBySession(ctx context.Context, sessionID string) (int, error) {
	runs, _ := m.GetRunningBySession(ctx, sessionID)
	return len(runs), nil
}

func (m *MockAgentRunStorage) CleanupOrphanedRuns(ctx context.Context) (int64, error) {
	return 0, nil
}

// TestFlowConfig returns programmatic FlowsConfig and PromptsConfig for testing
func TestFlowConfig() (*config.FlowsConfig, *config.PromptsConfig) {
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
				MaxSteps:       10,
				MaxContextSize: 4000,
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
				MaxSteps:       10,
				MaxContextSize: 4000,
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

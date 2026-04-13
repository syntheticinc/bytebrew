package testutil

import (
	"context"
	"sync"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/pkg/config"
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
					"read_file", "write_file", "edit_file",
					"search_code", "get_project_tree", "smart_search", "grep_search", "glob",
					"execute_command", "ask_user",
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

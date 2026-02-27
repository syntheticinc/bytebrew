package tools

import (
	"context"
	"testing"

	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing

type mockProxyForResolver struct{}

func (m *mockProxyForResolver) ReadFile(ctx context.Context, sessionID, filePath string, startLine, endLine int32) (string, error) {
	return "mock content", nil
}

func (m *mockProxyForResolver) WriteFile(ctx context.Context, sessionID, filePath, content string) (string, error) {
	return "mock written", nil
}

func (m *mockProxyForResolver) EditFile(ctx context.Context, sessionID, filePath, oldString, newString string, replaceAll bool) (string, error) {
	return "mock edited", nil
}

func (m *mockProxyForResolver) SearchCode(ctx context.Context, sessionID, query, projectKey string, limit int32, minScore float32) ([]byte, error) {
	return []byte("mock search"), nil
}

func (m *mockProxyForResolver) GetProjectTree(ctx context.Context, sessionID, projectKey, path string, maxDepth int) (string, error) {
	return "mock tree", nil
}

func (m *mockProxyForResolver) GrepSearch(ctx context.Context, sessionID, pattern string, limit int32, fileTypes []string, ignoreCase bool) (string, error) {
	return "mock grep", nil
}

func (m *mockProxyForResolver) GlobSearch(ctx context.Context, sessionID, pattern string, limit int32) (string, error) {
	return "mock glob", nil
}

func (m *mockProxyForResolver) SymbolSearch(ctx context.Context, sessionID, symbolName string, limit int32, symbolTypes []string) (string, error) {
	return "mock symbol", nil
}

func (m *mockProxyForResolver) ExecuteSubQueries(ctx context.Context, sessionID string, subQueries []*pb.SubQuery) ([]*pb.SubResult, error) {
	return nil, nil
}

func (m *mockProxyForResolver) ExecuteCommand(ctx context.Context, sessionID, command, cwd string, timeout int32) (string, error) {
	return "mock output", nil
}

func (m *mockProxyForResolver) AskUserQuestionnaire(ctx context.Context, sessionID, questionsJSON string) (string, error) {
	return `[{"question":"mock","answer":"mock answer"}]`, nil
}

func (m *mockProxyForResolver) LspRequest(ctx context.Context, sessionID, symbolName, operation string) (string, error) {
	return "", nil
}

func (m *mockProxyForResolver) ExecuteCommandFull(ctx context.Context, sessionID string, arguments map[string]string) (string, error) {
	return "mock output", nil
}

type mockTaskManagerForResolver struct{}

func (m *mockTaskManagerForResolver) CreateTask(ctx context.Context, sessionID, title, description string, criteria []string) (*domain.Task, error) {
	return &domain.Task{ID: "task-1", Title: title}, nil
}

func (m *mockTaskManagerForResolver) ApproveTask(ctx context.Context, taskID string) error {
	return nil
}

func (m *mockTaskManagerForResolver) StartTask(ctx context.Context, taskID string) error {
	return nil
}

func (m *mockTaskManagerForResolver) GetTask(ctx context.Context, taskID string) (*domain.Task, error) {
	return &domain.Task{ID: taskID}, nil
}

func (m *mockTaskManagerForResolver) GetTasks(ctx context.Context, sessionID string) ([]*domain.Task, error) {
	return []*domain.Task{{ID: "task-1"}}, nil
}

func (m *mockTaskManagerForResolver) CompleteTask(ctx context.Context, taskID string) error {
	return nil
}

func (m *mockTaskManagerForResolver) FailTask(ctx context.Context, taskID, reason string) error {
	return nil
}

func (m *mockTaskManagerForResolver) CancelTask(ctx context.Context, taskID, reason string) error {
	return nil
}

func (m *mockTaskManagerForResolver) SetTaskPriority(ctx context.Context, taskID string, priority int) error {
	return nil
}

func (m *mockTaskManagerForResolver) GetNextTask(ctx context.Context, sessionID string) (*domain.Task, error) {
	return &domain.Task{ID: "task-1"}, nil
}

type mockSubtaskManagerForResolver struct{}

func (m *mockSubtaskManagerForResolver) CreateSubtask(ctx context.Context, sessionID, taskID, title, description string, blockedBy, files []string) (*domain.Subtask, error) {
	return &domain.Subtask{ID: "subtask-1", Title: title}, nil
}

func (m *mockSubtaskManagerForResolver) GetSubtask(ctx context.Context, subtaskID string) (*domain.Subtask, error) {
	return &domain.Subtask{ID: subtaskID}, nil
}

func (m *mockSubtaskManagerForResolver) GetSubtasksByTask(ctx context.Context, taskID string) ([]*domain.Subtask, error) {
	return []*domain.Subtask{{ID: "subtask-1"}}, nil
}

func (m *mockSubtaskManagerForResolver) GetReadySubtasks(ctx context.Context, taskID string) ([]*domain.Subtask, error) {
	return []*domain.Subtask{{ID: "subtask-1"}}, nil
}

func (m *mockSubtaskManagerForResolver) CompleteSubtask(ctx context.Context, subtaskID, result string) error {
	return nil
}

func (m *mockSubtaskManagerForResolver) FailSubtask(ctx context.Context, subtaskID, reason string) error {
	return nil
}

type mockAgentPoolForResolver struct{}

func (m *mockAgentPoolForResolver) Spawn(ctx context.Context, sessionID, projectKey, subtaskID string, blocking bool) (string, error) {
	return "agent-1", nil
}

func (m *mockAgentPoolForResolver) WaitForAllSessionAgents(ctx context.Context, sessionID string) (WaitResult, error) {
	return WaitResult{AllDone: true}, nil
}

func (m *mockAgentPoolForResolver) HasBlockingWait(sessionID string) bool {
	return false
}

func (m *mockAgentPoolForResolver) NotifyUserMessage(sessionID, message string) {}

func (m *mockAgentPoolForResolver) GetStatusInfo(agentID string) (*AgentInfo, bool) {
	return &AgentInfo{ID: agentID, Status: "running"}, true
}

func (m *mockAgentPoolForResolver) GetAllAgentInfos() []AgentInfo {
	return []AgentInfo{{ID: "agent-1", Status: "running"}}
}

func (m *mockAgentPoolForResolver) StopAgent(agentID string) error {
	return nil
}

func (m *mockAgentPoolForResolver) RestartAgent(ctx context.Context, agentID string, blocking bool) (string, error) {
	return "agent-2", nil
}

func (m *mockAgentPoolForResolver) SpawnWithDescription(ctx context.Context, sessionID, projectKey string, flowType domain.FlowType, description string, blocking bool) (string, error) {
	return "agent-3", nil
}

type mockWebSearchToolForResolver struct{}

func (m *mockWebSearchToolForResolver) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: "web_search"}, nil
}

func (m *mockWebSearchToolForResolver) InvokableRun(ctx context.Context, args string, opts ...tool.Option) (string, error) {
	return "mock web search results", nil
}

func TestDefaultToolResolver_ResolveKnownTools(t *testing.T) {
	resolver := NewDefaultToolResolver()
	require.NotNil(t, resolver)

	deps := ToolDependencies{
		SessionID:  "session-1",
		ProjectKey: "project-1",
		Proxy:      &mockProxyForResolver{},
	}

	ctx := context.Background()
	tools, err := resolver.Resolve(ctx, []string{"read_file", "search_code"}, deps)
	require.NoError(t, err)
	assert.Len(t, tools, 2)

	// Verify tool info
	for _, tl := range tools {
		info, err := tl.Info(ctx)
		require.NoError(t, err)
		assert.Contains(t, []string{"read_file", "search_code"}, info.Name)
	}
}

func TestDefaultToolResolver_ResolveUnknown(t *testing.T) {
	resolver := NewDefaultToolResolver()

	deps := ToolDependencies{
		SessionID: "session-1",
		Proxy:     &mockProxyForResolver{},
	}

	ctx := context.Background()
	tools, err := resolver.Resolve(ctx, []string{"unknown_tool"}, deps)
	assert.Error(t, err)
	assert.Nil(t, tools)
	assert.Contains(t, err.Error(), "unknown tool")
}

func TestDefaultToolResolver_ResolveOptionalNil(t *testing.T) {
	resolver := NewDefaultToolResolver()

	deps := ToolDependencies{
		SessionID:     "session-1",
		Proxy:         &mockProxyForResolver{},
		WebSearchTool: nil, // optional, not configured
	}

	ctx := context.Background()
	tools, err := resolver.Resolve(ctx, []string{"read_file", "web_search"}, deps)
	require.NoError(t, err)
	assert.Len(t, tools, 1) // web_search skipped (nil), read_file resolved

	info, err := tools[0].Info(ctx)
	require.NoError(t, err)
	assert.Equal(t, "read_file", info.Name)
}

func TestDefaultToolResolver_ResolveEmpty(t *testing.T) {
	resolver := NewDefaultToolResolver()

	deps := ToolDependencies{
		SessionID: "session-1",
		Proxy:     &mockProxyForResolver{},
	}

	ctx := context.Background()
	tools, err := resolver.Resolve(ctx, []string{}, deps)
	require.NoError(t, err)
	assert.Len(t, tools, 0)
}

func TestDefaultToolResolver_ResolveAllKnownTools(t *testing.T) {
	resolver := NewDefaultToolResolver()

	deps := ToolDependencies{
		SessionID:      "session-1",
		ProjectKey:     "project-1",
		Proxy:          &mockProxyForResolver{},
		TaskManager:    &mockTaskManagerForResolver{},
		SubtaskManager: &mockSubtaskManagerForResolver{},
		AgentPool:      &mockAgentPoolForResolver{},
		WebSearchTool:  &mockWebSearchToolForResolver{},
		WebFetchTool:   &mockWebSearchToolForResolver{}, // reuse for testing
	}

	allToolNames := []string{
		"read_file",
		"write_file",
		"edit_file",
		"search_code",
		"grep_search",
		"glob",
		"smart_search",
		"get_project_tree",
		"execute_command",
		"web_search",
		"web_fetch",
		"manage_tasks",
		"manage_subtasks",
		"spawn_code_agent",
		"ask_user",
	}

	ctx := context.Background()
	tools, err := resolver.Resolve(ctx, allToolNames, deps)
	require.NoError(t, err)
	assert.Len(t, tools, len(allToolNames))
}

func TestDefaultToolResolver_ProxyRequired(t *testing.T) {
	resolver := NewDefaultToolResolver()

	deps := ToolDependencies{
		SessionID: "session-1",
		Proxy:     nil, // no proxy
	}

	ctx := context.Background()
	tools, err := resolver.Resolve(ctx, []string{"read_file", "write_file"}, deps)
	require.NoError(t, err)
	assert.Len(t, tools, 0) // both skipped (proxy required)
}

func TestDefaultToolResolver_OptionalManagers(t *testing.T) {
	resolver := NewDefaultToolResolver()

	deps := ToolDependencies{
		SessionID:      "session-1",
		Proxy:          &mockProxyForResolver{},
		TaskManager:    nil, // not configured
		SubtaskManager: nil, // not configured
		AgentPool:      nil, // not configured
	}

	ctx := context.Background()
	tools, err := resolver.Resolve(ctx, []string{
		"read_file",          // should work
		"manage_tasks",       // skipped (no manager)
		"manage_subtasks",    // skipped (no manager)
		"spawn_code_agent",   // skipped (no pool)
	}, deps)
	require.NoError(t, err)
	assert.Len(t, tools, 1) // only read_file resolved
}

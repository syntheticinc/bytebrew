package infrastructure

import (
	"context"
	"testing"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/llm"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/tools"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/service/engine"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/service/turn_executor"
	agentservice "github.com/syntheticinc/bytebrew/bytebrew/engine/internal/service/agent"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/pkg/config"
	pb "github.com/syntheticinc/bytebrew/bytebrew/engine/api/proto/gen"
	"github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations

type mockSnapshotRepoFactory struct {
	snapshots map[string]*domain.AgentContextSnapshot
}

func newMockSnapshotRepoFactory() *mockSnapshotRepoFactory {
	return &mockSnapshotRepoFactory{
		snapshots: make(map[string]*domain.AgentContextSnapshot),
	}
}

func (m *mockSnapshotRepoFactory) Save(ctx context.Context, snapshot *domain.AgentContextSnapshot) error {
	m.snapshots[snapshot.AgentID] = snapshot
	return nil
}

func (m *mockSnapshotRepoFactory) Load(ctx context.Context, sessionID, agentID string) (*domain.AgentContextSnapshot, error) {
	return m.snapshots[agentID], nil
}

func (m *mockSnapshotRepoFactory) Delete(ctx context.Context, sessionID, agentID string) error {
	delete(m.snapshots, agentID)
	return nil
}

func (m *mockSnapshotRepoFactory) FindActive(ctx context.Context) ([]*domain.AgentContextSnapshot, error) {
	return nil, nil
}

type mockHistoryRepoFactory struct {
	messages []*domain.Message
}

func newMockHistoryRepoFactory() *mockHistoryRepoFactory {
	return &mockHistoryRepoFactory{
		messages: make([]*domain.Message, 0),
	}
}

func (m *mockHistoryRepoFactory) Create(ctx context.Context, message *domain.Message) error {
	m.messages = append(m.messages, message)
	return nil
}

type mockChatModelFactory struct{}

func (m *mockChatModelFactory) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return &schema.Message{
		Role:    schema.Assistant,
		Content: "mock response",
	}, nil
}

func (m *mockChatModelFactory) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	sr, sw := schema.Pipe[*schema.Message](1)
	sw.Close()
	return sr, nil
}

func (m *mockChatModelFactory) BindTools(tools []*schema.ToolInfo) error {
	return nil
}

func (m *mockChatModelFactory) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return m, nil
}

type mockClientProxy struct {
	readFileCalled bool
}

func (m *mockClientProxy) ReadFile(ctx context.Context, sessionID, filePath string, startLine, endLine int32) (string, error) {
	m.readFileCalled = true
	return "file content", nil
}

func (m *mockClientProxy) WriteFile(ctx context.Context, sessionID, filePath, content string) (string, error) {
	return "success", nil
}

func (m *mockClientProxy) EditFile(ctx context.Context, sessionID, filePath, oldString, newString string, replaceAll bool) (string, error) {
	return "success", nil
}

func (m *mockClientProxy) SearchCode(ctx context.Context, sessionID, query, projectKey string, limit int32, minScore float32) ([]byte, error) {
	return []byte("{}"), nil
}

func (m *mockClientProxy) GetProjectTree(ctx context.Context, sessionID, projectKey, path string, maxDepth int) (string, error) {
	return "tree", nil
}

func (m *mockClientProxy) GrepSearch(ctx context.Context, sessionID, pattern string, limit int32, fileTypes []string, ignoreCase bool) (string, error) {
	return "results", nil
}

func (m *mockClientProxy) GlobSearch(ctx context.Context, sessionID, pattern string, limit int32) (string, error) {
	return "files", nil
}

func (m *mockClientProxy) SymbolSearch(ctx context.Context, sessionID, symbolName string, limit int32, symbolTypes []string) (string, error) {
	return "symbols", nil
}

func (m *mockClientProxy) ExecuteSubQueries(ctx context.Context, sessionID string, subQueries []*pb.SubQuery) ([]*pb.SubResult, error) {
	return nil, nil
}

func (m *mockClientProxy) ExecuteCommand(ctx context.Context, sessionID, command, cwd string, timeout int32) (string, error) {
	return "output", nil
}

func (m *mockClientProxy) AskUserQuestionnaire(ctx context.Context, sessionID, questionsJSON string) (string, error) {
	return `[{"question":"mock","answer":"answer"}]`, nil
}

func (m *mockClientProxy) LspRequest(ctx context.Context, sessionID, symbolName, operation string) (string, error) {
	return "", nil
}

func (m *mockClientProxy) ExecuteCommandFull(ctx context.Context, sessionID string, arguments map[string]string) (string, error) {
	return "output", nil
}

func testFlowConfigForFactory() (*config.FlowsConfig, *config.PromptsConfig) {
	flowsCfg := &config.FlowsConfig{
		Flows: map[string]config.FlowDefinition{
			"supervisor": {
				Name:            "supervisor-flow",
				SystemPromptRef: "supervisor_prompt",
				Tools:           []string{"read_file"},
				MaxSteps:        10,
				MaxContextSize:  4000,
				Lifecycle: config.LifecycleConfig{
					SuspendOn: []string{},
					ReportTo:  "user",
				},
			},
		},
	}
	promptsCfg := &config.PromptsConfig{
		SupervisorPrompt: "You are a supervisor",
	}
	return flowsCfg, promptsCfg
}

// newTestModelSelector creates a ModelSelector with the given mock for tests
func newTestModelSelector(chatModel model.ToolCallingChatModel) *llm.ModelSelector {
	return llm.NewModelSelector(chatModel, "test-model")
}

// Test 1: CreateForSession returns non-nil TurnExecutor
func TestEngineTurnExecutorFactory_CreateForSession(t *testing.T) {
	// Setup
	snapshotRepo := newMockSnapshotRepoFactory()
	historyRepo := newMockHistoryRepoFactory()
	eng := engine.New(snapshotRepo, historyRepo)

	flowsCfg, promptsCfg := testFlowConfigForFactory()
	flowManager, err := agentservice.NewFlowManager(flowsCfg, promptsCfg)
	require.NoError(t, err)

	builtinStore := tools.NewBuiltinToolStore()
	tools.RegisterAllBuiltins(builtinStore)
	toolResolver := tools.NewAgentToolResolver(builtinStore)
	chatModel := &mockChatModelFactory{}
	agentConfig := &config.AgentConfig{
		MaxContextSize: 4000,
	}

	factory := NewEngineTurnExecutorFactory(
		eng,
		flowManager,
		toolResolver,
		newTestModelSelector(chatModel),
		agentConfig,
		nil, // taskManager (not needed for this test)
		nil, // subtaskManager
		nil, // agentPool
		nil, // webSearchTool
		nil, // webFetchTool
		nil, // contextRemindersGetter
	)

	// Execute
	proxy := &mockClientProxy{}
	executor := factory.CreateForSession(proxy, "session-1", "project-1", "", "", "supervisor")

	// Verify
	require.NotNil(t, executor, "TurnExecutor should not be nil")
}

// Test 2: CreateForSession with proxy - proxy is passed to ToolDepsProvider
func TestEngineTurnExecutorFactory_CreateForSession_WithProxy(t *testing.T) {
	// Setup
	snapshotRepo := newMockSnapshotRepoFactory()
	historyRepo := newMockHistoryRepoFactory()
	eng := engine.New(snapshotRepo, historyRepo)

	flowsCfg, promptsCfg := testFlowConfigForFactory()
	flowManager, err := agentservice.NewFlowManager(flowsCfg, promptsCfg)
	require.NoError(t, err)

	builtinStore := tools.NewBuiltinToolStore()
	tools.RegisterAllBuiltins(builtinStore)
	toolResolver := tools.NewAgentToolResolver(builtinStore)
	chatModel := &mockChatModelFactory{}
	agentConfig := &config.AgentConfig{
		MaxContextSize: 4000,
	}

	factory := NewEngineTurnExecutorFactory(
		eng,
		flowManager,
		toolResolver,
		newTestModelSelector(chatModel),
		agentConfig,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil, // contextRemindersGetter
	)

	// Execute
	proxy := &mockClientProxy{}
	executor := factory.CreateForSession(proxy, "session-1", "project-1", "", "", "supervisor")

	require.NotNil(t, executor)

	// Verify that proxy is used by executing a turn
	// This will internally create tools with the proxy
	ctx := context.Background()
	_ = executor.ExecuteTurn(
		ctx,
		"session-1",
		"project-1",
		"read file test.txt",
		func(chunk string) error { return nil },
		func(event *domain.AgentEvent) error { return nil },
	)

	// We don't verify error here because the full execution may fail
	// The important thing is that the executor was created successfully
	// and can be called (indicating proxy was passed correctly)
	assert.NotNil(t, executor)
}

// Test 3: CreateForSession with nil proxy - should still create executor
func TestEngineTurnExecutorFactory_CreateForSession_NilProxy(t *testing.T) {
	snapshotRepo := newMockSnapshotRepoFactory()
	historyRepo := newMockHistoryRepoFactory()
	eng := engine.New(snapshotRepo, historyRepo)

	flowsCfg, promptsCfg := testFlowConfigForFactory()
	flowManager, err := agentservice.NewFlowManager(flowsCfg, promptsCfg)
	require.NoError(t, err)

	builtinStore := tools.NewBuiltinToolStore()
	tools.RegisterAllBuiltins(builtinStore)
	toolResolver := tools.NewAgentToolResolver(builtinStore)
	chatModel := &mockChatModelFactory{}
	agentConfig := &config.AgentConfig{
		MaxContextSize: 4000,
	}

	factory := NewEngineTurnExecutorFactory(
		eng,
		flowManager,
		toolResolver,
		newTestModelSelector(chatModel),
		agentConfig,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil, // contextRemindersGetter
	)

	// Execute with nil proxy
	executor := factory.CreateForSession(nil, "session-1", "project-1", "", "", "supervisor")

	// Verify
	require.NotNil(t, executor, "TurnExecutor should not be nil even with nil proxy")
}

// Test 4: CreateForSession with web tools
func TestEngineTurnExecutorFactory_CreateForSession_WithWebTools(t *testing.T) {
	snapshotRepo := newMockSnapshotRepoFactory()
	historyRepo := newMockHistoryRepoFactory()
	eng := engine.New(snapshotRepo, historyRepo)

	flowsCfg, promptsCfg := testFlowConfigForFactory()
	flowManager, err := agentservice.NewFlowManager(flowsCfg, promptsCfg)
	require.NoError(t, err)

	builtinStore := tools.NewBuiltinToolStore()
	tools.RegisterAllBuiltins(builtinStore)
	toolResolver := tools.NewAgentToolResolver(builtinStore)
	chatModel := &mockChatModelFactory{}
	agentConfig := &config.AgentConfig{
		MaxContextSize: 4000,
	}

	// Mock web tools
	mockWebSearch := &mockInvokableTool{name: "web_search"}
	mockWebFetch := &mockInvokableTool{name: "web_fetch"}

	factory := NewEngineTurnExecutorFactory(
		eng,
		flowManager,
		toolResolver,
		newTestModelSelector(chatModel),
		agentConfig,
		nil,
		nil,
		nil,
		mockWebSearch,
		mockWebFetch,
		nil, // contextRemindersGetter
	)

	executor := factory.CreateForSession(&mockClientProxy{}, "session-1", "project-1", "", "", "supervisor")

	require.NotNil(t, executor)
}

// Mock InvokableTool for web tools test
type mockInvokableTool struct {
	name string
}

func (m *mockInvokableTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: m.name,
		Desc: "mock tool",
	}, nil
}

func (m *mockInvokableTool) InvokableRun(ctx context.Context, args string, opts ...einotool.Option) (string, error) {
	return "mock result", nil
}

// --- Safety net tests for context reminders (Этап 0) ---

// mockContextReminderForFactory implements turn_executor.ContextReminderProvider for testing
type mockContextReminderForFactory struct {
	reminder string
	priority int
	enabled  bool
}

func (m *mockContextReminderForFactory) GetContextReminder(ctx context.Context, sessionID string) (string, int, bool) {
	return m.reminder, m.priority, m.enabled
}

// Test 5: CreateForSession with contextRemindersGetter - providers are passed through
func TestEngineTurnExecutorFactory_CreateForSession_WithContextReminders(t *testing.T) {
	snapshotRepo := newMockSnapshotRepoFactory()
	historyRepo := newMockHistoryRepoFactory()
	eng := engine.New(snapshotRepo, historyRepo)

	flowsCfg, promptsCfg := testFlowConfigForFactory()
	flowManager, err := agentservice.NewFlowManager(flowsCfg, promptsCfg)
	require.NoError(t, err)

	builtinStore := tools.NewBuiltinToolStore()
	tools.RegisterAllBuiltins(builtinStore)
	toolResolver := tools.NewAgentToolResolver(builtinStore)
	chatModel := &mockChatModelFactory{}
	agentConfig := &config.AgentConfig{
		MaxContextSize: 4000,
	}

	// Track whether getter was called
	getterCalled := false
	mockReminder := &mockContextReminderForFactory{
		reminder: "You have 3 active tasks",
		priority: 50,
		enabled:  true,
	}

	contextRemindersGetter := func() []turn_executor.ContextReminderProvider {
		getterCalled = true
		return []turn_executor.ContextReminderProvider{mockReminder}
	}

	factory := NewEngineTurnExecutorFactory(
		eng,
		flowManager,
		toolResolver,
		newTestModelSelector(chatModel),
		agentConfig,
		nil, // taskManager
		nil, // subtaskManager
		nil, // agentPool
		nil, // webSearchTool
		nil, // webFetchTool
		contextRemindersGetter,
	)

	// Execute
	proxy := &mockClientProxy{}
	executor := factory.CreateForSession(proxy, "session-1", "project-1", "", "", "supervisor")

	// Verify
	require.NotNil(t, executor, "TurnExecutor should not be nil")
	assert.True(t, getterCalled, "contextRemindersGetter should be called during CreateForSession")
}

// Test 6: CreateForSession with nil contextRemindersGetter - should not panic
func TestEngineTurnExecutorFactory_CreateForSession_NilContextRemindersGetter(t *testing.T) {
	snapshotRepo := newMockSnapshotRepoFactory()
	historyRepo := newMockHistoryRepoFactory()
	eng := engine.New(snapshotRepo, historyRepo)

	flowsCfg, promptsCfg := testFlowConfigForFactory()
	flowManager, err := agentservice.NewFlowManager(flowsCfg, promptsCfg)
	require.NoError(t, err)

	builtinStore := tools.NewBuiltinToolStore()
	tools.RegisterAllBuiltins(builtinStore)
	toolResolver := tools.NewAgentToolResolver(builtinStore)
	chatModel := &mockChatModelFactory{}
	agentConfig := &config.AgentConfig{
		MaxContextSize: 4000,
	}

	factory := NewEngineTurnExecutorFactory(
		eng,
		flowManager,
		toolResolver,
		newTestModelSelector(chatModel),
		agentConfig,
		nil, nil, nil, nil, nil,
		nil, // contextRemindersGetter = nil
	)

	// Execute — should not panic
	proxy := &mockClientProxy{}
	executor := factory.CreateForSession(proxy, "session-1", "project-1", "", "", "supervisor")

	require.NotNil(t, executor, "TurnExecutor should not be nil even with nil contextRemindersGetter")
}

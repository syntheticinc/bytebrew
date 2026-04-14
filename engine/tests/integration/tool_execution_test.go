//go:build integration

package integration

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/turnexecutorfactory"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/llm"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/testutil"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/tools"
	agentservice "github.com/syntheticinc/bytebrew/engine/internal/service/agent"
	"github.com/syntheticinc/bytebrew/engine/internal/service/engine"
	"github.com/syntheticinc/bytebrew/engine/internal/service/orchestrator"
	"github.com/syntheticinc/bytebrew/engine/pkg/config"
)

// eventCollector collects AgentEvents emitted during a turn.
type eventCollector struct {
	mu     sync.Mutex
	events []*domain.AgentEvent
	chunks []string
}

func newEventCollector() *eventCollector {
	return &eventCollector{}
}

func (c *eventCollector) chunkCallback(chunk string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.chunks = append(c.chunks, chunk)
	return nil
}

func (c *eventCollector) eventCallback(event *domain.AgentEvent) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, event)
	return nil
}

func (c *eventCollector) getEvents() []*domain.AgentEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make([]*domain.AgentEvent, len(c.events))
	copy(cp, c.events)
	return cp
}

func (c *eventCollector) getChunks() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make([]string, len(c.chunks))
	copy(cp, c.chunks)
	return cp
}

// finalAnswer returns the combined answer content from:
// 1. Answer chunks (chunkCallback) — primary in streaming mode
// 2. EventTypeAnswer events (fallback)
// In streaming mode, the agent sends text via chunkCallback; eventCallback
// receives tool_call/tool_result events. The EngineAdapter sends a final
// IsComplete=true answer event which may have empty content.
func (c *eventCollector) finalAnswer() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	// First: try chunks (streaming mode primary path)
	if len(c.chunks) > 0 {
		var sb string
		for _, ch := range c.chunks {
			sb += ch
		}
		if sb != "" {
			return sb
		}
	}

	// Fallback: look for answer events with content
	for i := len(c.events) - 1; i >= 0; i-- {
		if c.events[i].Type == domain.EventTypeAnswer && c.events[i].Content != "" {
			return c.events[i].Content
		}
	}

	// Last resort: answer_chunk events
	var answer string
	for _, e := range c.events {
		if e.Type == domain.EventTypeAnswerChunk {
			answer += e.Content
		}
	}
	return answer
}

// toolCallNames returns the names of all tool calls in order.
// For EventTypeToolCall, the tool name is stored in Content field.
func (c *eventCollector) toolCallNames() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	var names []string
	for _, e := range c.events {
		if e.Type == domain.EventTypeToolCall {
			names = append(names, e.Content)
		}
	}
	return names
}

// createTurnExecutor builds a TurnExecutor wired to MockChatModel + LocalClientOperationsProxy.
func createTurnExecutor(t *testing.T, scenario, projectRoot string) *turnExecutorSetup {
	t.Helper()

	chatModel := llm.NewMockChatModel(scenario)
	snapshotRepo := testutil.NewMockSnapshotRepo()
	historyRepo := testutil.NewMockHistoryRepo()
	agentEngine := engine.New(snapshotRepo, historyRepo)

	flowsCfg, promptsCfg := testutil.TestFlowConfig()
	flowManager, err := agentservice.NewFlowManager(flowsCfg, promptsCfg)
	require.NoError(t, err, "create flow manager")

	builtinStore := tools.NewBuiltinToolStore()
	tools.RegisterAllBuiltins(builtinStore)
	toolResolver := tools.NewAgentToolResolver(builtinStore)
	agentConfig := &config.AgentConfig{
		MaxContextSize:     4000,
		MaxSteps:           10,
		ToolReturnDirectly: make(map[string]struct{}),
		Prompts:            promptsCfg,
	}

	subtaskMgr := testutil.NewMockSubtaskManager()
	taskMgr := testutil.NewMockTaskManager()

	modelSelector := llm.NewModelSelector(chatModel, "mock-model")
	agentRunStorage := testutil.NewMockAgentRunStorage()
	agentPool := agentservice.NewAgentPool(agentservice.AgentPoolConfig{
		ModelSelector:   modelSelector,
		SubtaskManager:  subtaskMgr,
		AgentRunStorage: agentRunStorage,
		AgentConfig:     agentConfig,
		MaxConcurrent:   0,
	})
	agentPoolAdapter := agentservice.NewAgentPoolAdapter(agentPool)

	toolDepsProvider := tools.NewDefaultToolDepsProvider(nil, taskMgr, subtaskMgr, agentPoolAdapter, nil, nil)
	agentPool.SetEngine(agentEngine, flowManager, toolResolver, toolDepsProvider, nil, nil)

	proxy := tools.NewLocalClientOperationsProxy(projectRoot)

	factory := turnexecutorfactory.New(
		agentEngine, flowManager, toolResolver, modelSelector, agentConfig,
		taskMgr, subtaskMgr, agentPoolAdapter, nil, nil, nil,
	)

	executor := factory.CreateForSession(proxy, "test-session", "test-project", projectRoot, "linux", "supervisor")
	require.NotNil(t, executor, "TurnExecutor must not be nil")

	return &turnExecutorSetup{
		executor:   executor,
		proxy:      proxy,
		subtaskMgr: subtaskMgr,
		agentPool:  agentPool,
	}
}

type turnExecutorSetup struct {
	executor   orchestrator.TurnExecutor
	proxy      *tools.LocalClientOperationsProxy
	subtaskMgr *testutil.MockSubtaskManager
	agentPool  *agentservice.AgentPool
}

// writeFixture creates a file in the project root with the given content.
func writeFixture(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
}

// TC-A-01: read_file -> answer
func TestToolExecution_ReadFile(t *testing.T) {
	projectRoot := t.TempDir()
	writeFixture(t, projectRoot, "test.txt", "hello world\nsecond line")

	setup := createTurnExecutor(t, "local-read", projectRoot)
	defer setup.proxy.Dispose()

	collector := newEventCollector()
	ctx := context.Background()

	err := setup.executor.ExecuteTurn(ctx, "test-session", "test-project", "Read the test file",
		collector.chunkCallback, collector.eventCallback)
	require.NoError(t, err)

	// Verify tool was called
	toolNames := collector.toolCallNames()
	assert.Contains(t, toolNames, "read_file", "read_file should be called")

	// Verify final answer contains file content
	answer := collector.finalAnswer()
	assert.Contains(t, answer, "hello world", "answer should contain file content")
}

// TC-A-02: write_file -> answer
func TestToolExecution_WriteFile(t *testing.T) {
	projectRoot := t.TempDir()

	setup := createTurnExecutor(t, "local-write", projectRoot)
	defer setup.proxy.Dispose()

	collector := newEventCollector()
	ctx := context.Background()

	err := setup.executor.ExecuteTurn(ctx, "test-session", "test-project", "Write a file",
		collector.chunkCallback, collector.eventCallback)
	require.NoError(t, err)

	// Verify tool was called
	toolNames := collector.toolCallNames()
	assert.Contains(t, toolNames, "write_file", "write_file should be called")

	// Verify file was created on disk
	content, err := os.ReadFile(filepath.Join(projectRoot, "output.txt"))
	require.NoError(t, err, "output.txt should exist")
	assert.Equal(t, "hello from agent", string(content))

	// Verify answer confirms success
	answer := collector.finalAnswer()
	assert.Contains(t, answer, "WRITE_DONE", "answer should confirm write")
}

// TC-A-03: edit_file -> answer
func TestToolExecution_EditFile(t *testing.T) {
	projectRoot := t.TempDir()
	writeFixture(t, projectRoot, "app.txt", "line1\nold_value\nline3")

	setup := createTurnExecutor(t, "local-edit", projectRoot)
	defer setup.proxy.Dispose()

	collector := newEventCollector()
	ctx := context.Background()

	err := setup.executor.ExecuteTurn(ctx, "test-session", "test-project", "Edit the file",
		collector.chunkCallback, collector.eventCallback)
	require.NoError(t, err)

	// Verify tool was called
	toolNames := collector.toolCallNames()
	assert.Contains(t, toolNames, "edit_file", "edit_file should be called")

	// Verify file was modified on disk
	content, err := os.ReadFile(filepath.Join(projectRoot, "app.txt"))
	require.NoError(t, err, "app.txt should exist")
	assert.Contains(t, string(content), "new_value", "file should contain new_value")
	assert.NotContains(t, string(content), "old_value", "file should not contain old_value")

	// Verify answer confirms success
	answer := collector.finalAnswer()
	assert.Contains(t, answer, "EDIT_DONE", "answer should confirm edit")
}

// TC-A-04: execute_command -> answer
func TestToolExecution_ExecuteCommand(t *testing.T) {
	projectRoot := t.TempDir()

	setup := createTurnExecutor(t, "local-exec", projectRoot)
	defer setup.proxy.Dispose()

	collector := newEventCollector()
	ctx := context.Background()

	err := setup.executor.ExecuteTurn(ctx, "test-session", "test-project", "Run a command",
		collector.chunkCallback, collector.eventCallback)
	require.NoError(t, err)

	// Verify tool was called
	toolNames := collector.toolCallNames()
	assert.Contains(t, toolNames, "execute_command", "execute_command should be called")

	// Verify answer contains command output
	answer := collector.finalAnswer()
	assert.Contains(t, answer, "EXEC_RESULT", "answer should contain exec result")
	assert.Contains(t, answer, "hello_from_test", "answer should contain command output")
}

// TC-A-05: multi-tool chain (read a.txt -> read b.txt -> answer with both)
func TestToolExecution_MultiToolChain(t *testing.T) {
	projectRoot := t.TempDir()
	writeFixture(t, projectRoot, "a.txt", "content_of_a")
	writeFixture(t, projectRoot, "b.txt", "content_of_b")

	setup := createTurnExecutor(t, "local-multi-tool", projectRoot)
	defer setup.proxy.Dispose()

	collector := newEventCollector()
	ctx := context.Background()

	err := setup.executor.ExecuteTurn(ctx, "test-session", "test-project", "Read both files",
		collector.chunkCallback, collector.eventCallback)
	require.NoError(t, err)

	// Verify both read_file calls were made
	toolNames := collector.toolCallNames()
	readCount := 0
	for _, name := range toolNames {
		if name == "read_file" {
			readCount++
		}
	}
	assert.Equal(t, 2, readCount, "should have 2 read_file calls")

	// Verify answer contains both file contents
	answer := collector.finalAnswer()
	assert.Contains(t, answer, "MULTI_READ", "answer should be multi-read result")
	assert.Contains(t, answer, "content_of_a", "answer should contain a.txt content")
	assert.Contains(t, answer, "content_of_b", "answer should contain b.txt content")
}

// TC-A-06: tool error recovery (read nonexistent file -> LLM recovers)
func TestToolExecution_ErrorRecovery(t *testing.T) {
	projectRoot := t.TempDir()
	// Intentionally do NOT create nonexistent.txt

	setup := createTurnExecutor(t, "local-read-error", projectRoot)
	defer setup.proxy.Dispose()

	collector := newEventCollector()
	ctx := context.Background()

	err := setup.executor.ExecuteTurn(ctx, "test-session", "test-project", "Read the file",
		collector.chunkCallback, collector.eventCallback)
	require.NoError(t, err)

	// Verify read_file was called (even though it fails)
	toolNames := collector.toolCallNames()
	assert.Contains(t, toolNames, "read_file", "read_file should be called")

	// Verify LLM received the error and recovered
	answer := collector.finalAnswer()
	assert.Contains(t, answer, "RECOVERED", "LLM should recover from file-not-found error")
}

// TC-A-09: glob + grep search
func TestToolExecution_GlobAndGrep(t *testing.T) {
	projectRoot := t.TempDir()
	writeFixture(t, projectRoot, "src/main.go", "package main\n\nfunc hello() {}\n")
	writeFixture(t, projectRoot, "src/util.go", "package main\n\nfunc helperHello() {}\n")
	writeFixture(t, projectRoot, "readme.md", "# Hello World\n")

	setup := createTurnExecutor(t, "local-glob-grep", projectRoot)
	defer setup.proxy.Dispose()

	collector := newEventCollector()
	ctx := context.Background()

	err := setup.executor.ExecuteTurn(ctx, "test-session", "test-project", "Search for files and patterns",
		collector.chunkCallback, collector.eventCallback)
	require.NoError(t, err)

	// Verify both glob and grep_search were called
	toolNames := collector.toolCallNames()
	assert.Contains(t, toolNames, "glob", "glob should be called")
	assert.Contains(t, toolNames, "grep_search", "grep_search should be called")

	// Verify answer indicates search completion
	answer := collector.finalAnswer()
	assert.Contains(t, answer, "SEARCH_DONE", "answer should indicate search completion")
}

// TC-A-07: AskUser flow — LLM calls ask_user → headless auto-answer → LLM receives response
func TestToolExecution_AskUser(t *testing.T) {
	projectRoot := t.TempDir()

	setup := createTurnExecutor(t, "ask-user", projectRoot)
	defer setup.proxy.Dispose()

	collector := newEventCollector()
	ctx := context.Background()

	err := setup.executor.ExecuteTurn(ctx, "test-session", "test-project", "Ask the user something",
		collector.chunkCallback, collector.eventCallback)
	require.NoError(t, err)

	// Verify ask_user tool was called
	toolNames := collector.toolCallNames()
	assert.Contains(t, toolNames, "ask_user", "ask_user should be called")

	// Verify final answer contains user response
	// In headless mode, LocalClientOperationsProxy auto-selects the first option
	answer := collector.finalAnswer()
	assert.NotEmpty(t, answer, "should have an answer after ask_user")
	assert.Contains(t, answer, "User said:", "answer should contain user response")
}

// TC-A-10: Cancel during execution — context cancellation stops processing
func TestToolExecution_CancelDuringExecution(t *testing.T) {
	projectRoot := t.TempDir()

	// "cancel-during-stream" scenario sleeps 3s during Stream, responding to cancellation
	setup := createTurnExecutor(t, "cancel-during-stream", projectRoot)
	defer setup.proxy.Dispose()

	collector := newEventCollector()
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()

	err := setup.executor.ExecuteTurn(ctx, "test-session", "test-project", "Do something slow",
		collector.chunkCallback, collector.eventCallback)

	// The turn should finish (with or without error) — cancellation is handled gracefully
	// Either: context.Canceled error, or the turn completes before cancellation
	if err != nil {
		assert.ErrorIs(t, err, context.Canceled, "error should be context.Canceled")
	}
}

// TC-A-08: multi-agent spawn — supervisor spawns code agent, code agent completes subtask
func TestToolExecution_MultiAgent(t *testing.T) {
	projectRoot := t.TempDir()

	setup := createTurnExecutor(t, "multi-agent", projectRoot)
	defer setup.proxy.Dispose()

	// Pre-seed subtask that the supervisor will spawn a code agent for.
	// The "multi-agent" MockChatModel scenario uses subtask_id="test-subtask-1".
	setup.subtaskMgr.Subtasks["test-subtask-1"] = &domain.Subtask{
		ID:          "test-subtask-1",
		SessionID:   "test-session",
		TaskID:      "task-1",
		Title:       "Implement feature X",
		Description: "Write the implementation for feature X",
		Status:      domain.SubtaskStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Code agent needs a proxy for the session to resolve tools
	setup.agentPool.SetProxyForSession("test-session", setup.proxy)

	collector := newEventCollector()
	ctx := context.Background()

	err := setup.executor.ExecuteTurn(ctx, "test-session", "test-project", "Spawn a code agent",
		collector.chunkCallback, collector.eventCallback)
	require.NoError(t, err)

	// Verify spawn_agent was called by the supervisor
	toolNames := collector.toolCallNames()
	assert.Contains(t, toolNames, "spawn_agent", "spawn_agent should be called")

	// Verify final answer from supervisor confirms completion
	answer := collector.finalAnswer()
	assert.Contains(t, answer, "agents completed", "answer should confirm all agents completed")

	// Verify subtask was assigned to an agent
	subtask := setup.subtaskMgr.Subtasks["test-subtask-1"]
	assert.NotEmpty(t, subtask.AssignedAgentID, "subtask should be assigned to an agent")

	// Verify agent lifecycle events via AgentPool's session event callback.
	// These events are emitted through AgentPool.emitEventForSession, not through
	// the TurnExecutor's event callback. Register a separate collector to verify.
	poolEvents := collector.getEvents()

	// Verify we have tool_call and tool_result events for spawn_agent
	hasToolCall := false
	hasToolResult := false
	for _, e := range poolEvents {
		if e.Type == domain.EventTypeToolCall && e.Content == "spawn_agent" {
			hasToolCall = true
		}
		if e.Type == domain.EventTypeToolResult {
			hasToolResult = true
		}
	}
	assert.True(t, hasToolCall, "should have tool_call event for spawn_agent")
	assert.True(t, hasToolResult, "should have tool_result event for spawn_agent")

	// Verify subtask was completed by the code agent
	assert.Equal(t, domain.SubtaskStatusCompleted, subtask.Status, "subtask should be completed")
}

// TestToolExecution_EventSequence verifies the event ordering:
// tool_call -> tool_result -> ... -> final answer (IsComplete=true)
func TestToolExecution_EventSequence(t *testing.T) {
	projectRoot := t.TempDir()
	writeFixture(t, projectRoot, "test.txt", "test content")

	setup := createTurnExecutor(t, "local-read", projectRoot)
	defer setup.proxy.Dispose()

	collector := newEventCollector()
	ctx := context.Background()

	err := setup.executor.ExecuteTurn(ctx, "test-session", "test-project", "Read the file",
		collector.chunkCallback, collector.eventCallback)
	require.NoError(t, err)

	events := collector.getEvents()
	require.NotEmpty(t, events, "should have events")

	// Last event should be answer with IsComplete=true
	lastEvent := events[len(events)-1]
	assert.Equal(t, domain.EventTypeAnswer, lastEvent.Type, "last event should be answer")
	assert.True(t, lastEvent.IsComplete, "last event should be complete")

	// Should have at least one tool_call event before the final answer
	hasToolCall := false
	for _, e := range events {
		if e.Type == domain.EventTypeToolCall {
			hasToolCall = true
			break
		}
	}
	assert.True(t, hasToolCall, "should have at least one tool call event")
}

package agent

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/service/orchestrator"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/pkg/config"
	"github.com/cloudwego/eino/components/model"
)

// Mock SubtaskManager for testing
type mockSubtaskManager struct {
	subtasks map[string]*domain.Subtask
	mu       sync.RWMutex
	// Track method calls
	assignCalls   []assignCall
	completeCalls []completeCall
	failCalls     []failCall
}

type assignCall struct {
	subtaskID string
	agentID   string
}

type completeCall struct {
	subtaskID string
	result    string
}

type failCall struct {
	subtaskID string
	reason    string
}

func newMockSubtaskManager() *mockSubtaskManager {
	return &mockSubtaskManager{
		subtasks:      make(map[string]*domain.Subtask),
		assignCalls:   []assignCall{},
		completeCalls: []completeCall{},
		failCalls:     []failCall{},
	}
}

func (m *mockSubtaskManager) GetSubtask(ctx context.Context, subtaskID string) (*domain.Subtask, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	subtask := m.subtasks[subtaskID]
	return subtask, nil
}

func (m *mockSubtaskManager) AssignSubtaskToAgent(ctx context.Context, subtaskID, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.assignCalls = append(m.assignCalls, assignCall{subtaskID, agentID})
	if subtask, ok := m.subtasks[subtaskID]; ok {
		subtask.AssignToAgent(agentID)
	}
	return nil
}

func (m *mockSubtaskManager) CompleteSubtask(ctx context.Context, subtaskID, result string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.completeCalls = append(m.completeCalls, completeCall{subtaskID, result})
	if subtask, ok := m.subtasks[subtaskID]; ok {
		_ = subtask.Complete(result)
	}
	return nil
}

func (m *mockSubtaskManager) FailSubtask(ctx context.Context, subtaskID, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failCalls = append(m.failCalls, failCall{subtaskID, reason})
	if subtask, ok := m.subtasks[subtaskID]; ok {
		_ = subtask.Fail(reason)
	}
	return nil
}

func (m *mockSubtaskManager) addSubtask(subtaskID string, subtask *domain.Subtask) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subtasks[subtaskID] = subtask
}

// mockModelSelectorForPool implements AgentModelSelector for testing.
type mockModelSelectorForPool struct {
	chatModel *mockChatModel
}

func (m *mockModelSelectorForPool) Select(_ domain.FlowType) model.ToolCallingChatModel {
	return m.chatModel
}

func (m *mockModelSelectorForPool) ModelName(_ domain.FlowType) string {
	return "mock-model"
}

func newMockModelSelector() *mockModelSelectorForPool {
	return &mockModelSelectorForPool{chatModel: &mockChatModel{}}
}

// createTestAgentPool creates an AgentPool for testing
func createTestAgentPool(t *testing.T) (*AgentPool, *mockSubtaskManager) {
	t.Helper()

	subtaskMgr := newMockSubtaskManager()

	pool := NewAgentPool(AgentPoolConfig{
		ModelSelector:  newMockModelSelector(),
		SubtaskManager: subtaskMgr,
		AgentConfig:    &config.AgentConfig{},
		SessionDirName: "test-session",
	})

	return pool, subtaskMgr
}

func TestAgentPool_Spawn(t *testing.T) {
	ctx := context.Background()
	pool, subtaskMgr := createTestAgentPool(t)

	// Add subtask to mock manager
	subtask, _ := domain.NewTaskSubtask(
		"subtask-1",
		"session-1",
		"task-1",
		"Test Subtask",
		"Test description",
		nil,
		nil,
	)
	subtaskMgr.addSubtask("subtask-1", subtask)

	agentID, err := pool.Spawn(ctx, "session-1", "project-1", "subtask-1", false)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}

	if agentID == "" {
		t.Error("Spawn() returned empty agentID")
	}

	// Check agent is in the pool (snapshot is safe to read without lock)
	snap, ok := pool.GetStatus(agentID)
	if !ok {
		t.Fatal("Agent not found in pool after Spawn()")
	}

	// Agent may already be "failed" (no engine configured), both are valid
	if snap.Status != "running" && snap.Status != "failed" {
		t.Errorf("Agent status = %v, want 'running' or 'failed'", snap.Status)
	}

	if snap.SubtaskID != "subtask-1" {
		t.Errorf("Agent.SubtaskID = %v, want 'subtask-1'", snap.SubtaskID)
	}

	if snap.SessionID != "session-1" {
		t.Errorf("Agent.SessionID = %v, want 'session-1'", snap.SessionID)
	}

	if snap.ProjectKey != "project-1" {
		t.Errorf("Agent.ProjectKey = %v, want 'project-1'", snap.ProjectKey)
	}

	// Verify subtask was assigned
	subtaskMgr.mu.RLock()
	assignCalls := len(subtaskMgr.assignCalls)
	subtaskMgr.mu.RUnlock()

	if assignCalls != 1 {
		t.Errorf("AssignSubtaskToAgent called %d times, want 1", assignCalls)
	}
}

func TestAgentPool_Spawn_SubtaskNotFound(t *testing.T) {
	ctx := context.Background()
	pool, _ := createTestAgentPool(t)

	// Don't add subtask to manager (simulates not found)
	_, err := pool.Spawn(ctx, "session-1", "project-1", "nonexistent", false)
	if err == nil {
		t.Error("Spawn() with nonexistent subtask should return error")
	}
}

func TestAgentPool_GetStatus(t *testing.T) {
	ctx := context.Background()
	pool, subtaskMgr := createTestAgentPool(t)

	subtask, _ := domain.NewTaskSubtask(
		"subtask-1",
		"session-1",
		"task-1",
		"Test",
		"Description",
		nil,
		nil,
	)
	subtaskMgr.addSubtask("subtask-1", subtask)

	agentID, err := pool.Spawn(ctx, "session-1", "project-1", "subtask-1", false)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}

	// Test GetStatus for existing agent (snapshot is safe to read)
	snap, ok := pool.GetStatus(agentID)
	if !ok {
		t.Fatal("GetStatus() returned false for existing agent")
	}

	// Agent may already be "failed" (no engine), both are valid
	if snap.Status != "running" && snap.Status != "failed" {
		t.Errorf("Agent.Status = %v, want 'running' or 'failed'", snap.Status)
	}

	// Test GetStatus for non-existent agent
	_, ok = pool.GetStatus("nonexistent")
	if ok {
		t.Error("GetStatus() returned true for nonexistent agent")
	}
}

func TestAgentPool_StopAgent(t *testing.T) {
	ctx := context.Background()
	pool, subtaskMgr := createTestAgentPool(t)

	subtask, _ := domain.NewTaskSubtask(
		"subtask-1",
		"session-1",
		"task-1",
		"Test",
		"Description",
		nil,
		nil,
	)
	subtaskMgr.addSubtask("subtask-1", subtask)

	agentID, err := pool.Spawn(ctx, "session-1", "project-1", "subtask-1", false)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}

	// Wait for agent to start (give it a moment to enter running state)
	time.Sleep(10 * time.Millisecond)

	// Verify agent is in running state before stopping
	snap, ok := pool.GetStatus(agentID)
	if !ok {
		t.Fatal("Agent not found after Spawn()")
	}

	// If agent already failed (e.g., due to missing Engine), skip stop test
	if snap.Status == "failed" {
		t.Skip("Agent failed to start (missing Engine), skipping stop test")
	}

	// Stop the agent
	err = pool.StopAgent(agentID)
	if err != nil {
		t.Errorf("StopAgent() error = %v", err)
	}

	// Verify status changed to stopped
	snap, ok = pool.GetStatus(agentID)
	if !ok {
		t.Fatal("Agent disappeared after StopAgent()")
	}

	if snap.Status != "stopped" {
		t.Errorf("Agent.Status = %v, want 'stopped'", snap.Status)
	}
}

func TestAgentPool_StopAgent_NotRunning(t *testing.T) {
	ctx := context.Background()
	pool, subtaskMgr := createTestAgentPool(t)

	subtask, _ := domain.NewTaskSubtask(
		"subtask-1",
		"session-1",
		"task-1",
		"Test",
		"Description",
		nil,
		nil,
	)
	subtaskMgr.addSubtask("subtask-1", subtask)

	agentID, err := pool.Spawn(ctx, "session-1", "project-1", "subtask-1", false)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}

	// Stop the agent first
	_ = pool.StopAgent(agentID)

	// Try stopping again
	err = pool.StopAgent(agentID)
	if err == nil {
		t.Error("StopAgent() on stopped agent should return error")
	}
}

func TestAgentPool_StopAgent_NotFound(t *testing.T) {
	pool, _ := createTestAgentPool(t)

	err := pool.StopAgent("nonexistent")
	if err == nil {
		t.Error("StopAgent() on nonexistent agent should return error")
	}
}

func TestAgentPool_MarkCompleted_NoAutoRetry(t *testing.T) {
	pool, subtaskMgr := createTestAgentPool(t)

	// Add subtask
	subtask, _ := domain.NewTaskSubtask(
		"subtask-1",
		"session-1",
		"task-1",
		"Test",
		"Description",
		nil,
		nil,
	)
	subtask.Status = domain.SubtaskStatusInProgress
	subtaskMgr.addSubtask("subtask-1", subtask)

	// Create running agent manually
	agentID := "code-agent-test123"
	running := &RunningAgent{
		ID:         agentID,
		SubtaskID:  "subtask-1",
		SessionID:  "session-1",
		ProjectKey: "project-1",
		Status:     "running",
		StartedAt:  time.Now(),
		Cancel:     func() {},
	}

	pool.mu.Lock()
	pool.agents[agentID] = running
	pool.mu.Unlock()

	// Mark as completed
	pool.markCompleted(agentID, "subtask-1", "Task completed successfully")

	// Give it a moment for async operations
	time.Sleep(50 * time.Millisecond)

	// Verify agent status (snapshot is safe to read)
	snap, ok := pool.GetStatus(agentID)
	if !ok {
		t.Fatal("Agent disappeared after markCompleted()")
	}

	if snap.Status != "completed" {
		t.Errorf("Agent.Status = %v, want 'completed'", snap.Status)
	}

	if snap.Result != "Task completed successfully" {
		t.Errorf("Agent.Result = %v, want 'Task completed successfully'", snap.Result)
	}

	// Verify CompleteSubtask was called
	subtaskMgr.mu.RLock()
	completeCalls := len(subtaskMgr.completeCalls)
	subtaskMgr.mu.RUnlock()

	if completeCalls != 1 {
		t.Errorf("CompleteSubtask called %d times, want 1", completeCalls)
	}

	// IMPORTANT: Verify NO auto-retry happened
	// The pool should have exactly 1 agent (the original)
	allAgents := pool.GetAllAgents()
	if len(allAgents) != 1 {
		t.Errorf("Pool has %d agents, want 1 (no auto-retry should happen)", len(allAgents))
	}
}

func TestAgentPool_MarkFailed_NoAutoRetry(t *testing.T) {
	pool, subtaskMgr := createTestAgentPool(t)

	// Add subtask
	subtask, _ := domain.NewTaskSubtask(
		"subtask-1",
		"session-1",
		"task-1",
		"Test",
		"Description",
		nil,
		nil,
	)
	subtask.Status = domain.SubtaskStatusInProgress
	subtaskMgr.addSubtask("subtask-1", subtask)

	// Create running agent manually
	agentID := "code-agent-test456"
	running := &RunningAgent{
		ID:         agentID,
		SubtaskID:  "subtask-1",
		SessionID:  "session-1",
		ProjectKey: "project-1",
		Status:     "running",
		StartedAt:  time.Now(),
		Cancel:     func() {},
	}

	pool.mu.Lock()
	pool.agents[agentID] = running
	pool.mu.Unlock()

	// Mark as failed
	pool.markFailed(agentID, "subtask-1", "Agent execution failed")

	// Give it a moment for async operations
	time.Sleep(50 * time.Millisecond)

	// Verify agent status (snapshot is safe to read)
	snap, ok := pool.GetStatus(agentID)
	if !ok {
		t.Fatal("Agent disappeared after markFailed()")
	}

	if snap.Status != "failed" {
		t.Errorf("Agent.Status = %v, want 'failed'", snap.Status)
	}

	if snap.Error != "Agent execution failed" {
		t.Errorf("Agent.Error = %v, want 'Agent execution failed'", snap.Error)
	}

	// Verify FailSubtask was called
	subtaskMgr.mu.RLock()
	failCalls := len(subtaskMgr.failCalls)
	subtaskMgr.mu.RUnlock()

	if failCalls != 1 {
		t.Errorf("FailSubtask called %d times, want 1", failCalls)
	}

	// CRITICAL: Verify NO auto-retry happened
	// The pool should have exactly 1 agent (the failed one)
	allAgents := pool.GetAllAgents()
	if len(allAgents) != 1 {
		t.Errorf("Pool has %d agents, want 1 (no auto-retry should happen)", len(allAgents))
	}
}

func TestAgentPool_EventBus_Completed(t *testing.T) {
	pool, subtaskMgr := createTestAgentPool(t)

	// Create event bus
	bus := orchestrator.NewSessionEventBus(16)
	defer bus.Close()

	pool.SetEventBus(bus)

	// Add subtask
	subtask, _ := domain.NewTaskSubtask(
		"subtask-1",
		"session-1",
		"task-1",
		"Test",
		"Description",
		nil,
		nil,
	)
	subtask.Status = domain.SubtaskStatusInProgress
	subtaskMgr.addSubtask("subtask-1", subtask)

	// Create running agent
	agentID := "code-agent-bus123"
	running := &RunningAgent{
		ID:         agentID,
		SubtaskID:  "subtask-1",
		SessionID:  "session-1",
		ProjectKey: "project-1",
		Status:     "running",
		StartedAt:  time.Now(),
		Cancel:     func() {},
	}

	pool.mu.Lock()
	pool.agents[agentID] = running
	pool.mu.Unlock()

	// Mark as completed
	pool.markCompleted(agentID, "subtask-1", "Success result")

	// Read event from bus (with timeout)
	select {
	case event := <-bus.Events():
		if event.Type != orchestrator.EventAgentCompleted {
			t.Errorf("Event.Type = %v, want EventAgentCompleted", event.Type)
		}
		if event.AgentID != agentID {
			t.Errorf("Event.AgentID = %v, want %v", event.AgentID, agentID)
		}
		if event.SubtaskID != "subtask-1" {
			t.Errorf("Event.SubtaskID = %v, want 'subtask-1'", event.SubtaskID)
		}
		if event.Content != "Success result" {
			t.Errorf("Event.Content = %v, want 'Success result'", event.Content)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for EventAgentCompleted on bus")
	}
}

func TestAgentPool_EventBus_Failed(t *testing.T) {
	pool, subtaskMgr := createTestAgentPool(t)

	// Create event bus
	bus := orchestrator.NewSessionEventBus(16)
	defer bus.Close()

	pool.SetEventBus(bus)

	// Add subtask
	subtask, _ := domain.NewTaskSubtask(
		"subtask-1",
		"session-1",
		"task-1",
		"Test",
		"Description",
		nil,
		nil,
	)
	subtask.Status = domain.SubtaskStatusInProgress
	subtaskMgr.addSubtask("subtask-1", subtask)

	// Create running agent
	agentID := "code-agent-bus456"
	running := &RunningAgent{
		ID:         agentID,
		SubtaskID:  "subtask-1",
		SessionID:  "session-1",
		ProjectKey: "project-1",
		Status:     "running",
		StartedAt:  time.Now(),
		Cancel:     func() {},
	}

	pool.mu.Lock()
	pool.agents[agentID] = running
	pool.mu.Unlock()

	// Mark as failed
	pool.markFailed(agentID, "subtask-1", "Execution error")

	// Read event from bus (with timeout)
	select {
	case event := <-bus.Events():
		if event.Type != orchestrator.EventAgentFailed {
			t.Errorf("Event.Type = %v, want EventAgentFailed", event.Type)
		}
		if event.AgentID != agentID {
			t.Errorf("Event.AgentID = %v, want %v", event.AgentID, agentID)
		}
		if event.SubtaskID != "subtask-1" {
			t.Errorf("Event.SubtaskID = %v, want 'subtask-1'", event.SubtaskID)
		}
		if event.Content != "Execution error" {
			t.Errorf("Event.Content = %v, want 'Execution error'", event.Content)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for EventAgentFailed on bus")
	}
}

func TestAgentPool_GetAllAgents(t *testing.T) {
	pool, _ := createTestAgentPool(t)

	// Add some agents
	pool.mu.Lock()
	pool.agents["agent-1"] = &RunningAgent{ID: "agent-1", Status: "running"}
	pool.agents["agent-2"] = &RunningAgent{ID: "agent-2", Status: "completed"}
	pool.agents["agent-3"] = &RunningAgent{ID: "agent-3", Status: "failed"}
	pool.mu.Unlock()

	all := pool.GetAllAgents()

	if len(all) != 3 {
		t.Errorf("GetAllAgents() returned %d agents, want 3", len(all))
	}

	// Verify all agents are present
	ids := make(map[string]bool)
	for _, agent := range all {
		ids[agent.ID] = true
	}

	for _, expectedID := range []string{"agent-1", "agent-2", "agent-3"} {
		if !ids[expectedID] {
			t.Errorf("Agent %s not in GetAllAgents() result", expectedID)
		}
	}
}

func TestAgentPool_SetSessionDirName(t *testing.T) {
	pool, _ := createTestAgentPool(t)

	pool.SetSessionDirName("new-session-dir")

	pool.mu.RLock()
	dirName := pool.sessionDirName
	pool.mu.RUnlock()

	if dirName != "new-session-dir" {
		t.Errorf("sessionDirName = %v, want 'new-session-dir'", dirName)
	}
}

func TestAgentPool_SetEventBus(t *testing.T) {
	pool, _ := createTestAgentPool(t)

	bus := orchestrator.NewSessionEventBus(16)
	defer bus.Close()

	pool.SetEventBus(bus)

	pool.mu.RLock()
	gotBus := pool.eventBus
	pool.mu.RUnlock()

	if gotBus != bus {
		t.Error("SetEventBus() did not set the bus correctly")
	}
}

func TestAgentPool_RestartAgent(t *testing.T) {
	ctx := context.Background()
	pool, subtaskMgr := createTestAgentPool(t)

	// Add subtask
	subtask, _ := domain.NewTaskSubtask(
		"subtask-1",
		"session-1",
		"task-1",
		"Test",
		"Description",
		nil,
		nil,
	)
	subtask.Context["project_key"] = "project-1"
	subtaskMgr.addSubtask("subtask-1", subtask)

	// Add a failed agent
	oldAgentID := "code-agent-old"
	pool.mu.Lock()
	pool.agents[oldAgentID] = &RunningAgent{
		ID:         oldAgentID,
		SubtaskID:  "subtask-1",
		SessionID:  "session-1",
		ProjectKey: "project-1",
		Status:     "failed",
		Error:      "Previous failure",
		StartedAt:  time.Now().Add(-5 * time.Minute),
		Cancel:     func() {},
	}
	pool.mu.Unlock()

	// Restart the agent
	newAgentID, err := pool.RestartAgent(ctx, oldAgentID, false)
	if err != nil {
		t.Fatalf("RestartAgent() error = %v", err)
	}

	if newAgentID == "" {
		t.Error("RestartAgent() returned empty agentID")
	}

	if newAgentID == oldAgentID {
		t.Error("RestartAgent() should create new agent, not reuse old ID")
	}

	// Verify new agent exists and is running (may already be "failed" — no engine)
	newSnap, ok := pool.GetStatus(newAgentID)
	if !ok {
		t.Fatal("New agent not found after RestartAgent()")
	}

	if newSnap.Status != "running" && newSnap.Status != "failed" {
		t.Errorf("New agent status = %v, want 'running' or 'failed'", newSnap.Status)
	}

	if newSnap.SubtaskID != "subtask-1" {
		t.Errorf("New agent subtaskID = %v, want 'subtask-1'", newSnap.SubtaskID)
	}

	// Old agent should still exist (not removed by restart)
	oldSnap, ok := pool.GetStatus(oldAgentID)
	if !ok {
		t.Error("Old agent should still exist after RestartAgent()")
	} else if oldSnap.Status != "failed" {
		t.Errorf("Old agent status changed to %v", oldSnap.Status)
	}
}

func TestAgentPool_RestartAgent_StillRunning(t *testing.T) {
	ctx := context.Background()
	pool, subtaskMgr := createTestAgentPool(t)

	// Add subtask
	subtask, _ := domain.NewTaskSubtask(
		"subtask-1",
		"session-1",
		"task-1",
		"Test",
		"Description",
		nil,
		nil,
	)
	subtaskMgr.addSubtask("subtask-1", subtask)

	// Add a running agent
	agentID := "code-agent-running"
	pool.mu.Lock()
	pool.agents[agentID] = &RunningAgent{
		ID:        agentID,
		SubtaskID: "subtask-1",
		SessionID: "session-1",
		Status:    "running",
		StartedAt: time.Now(),
		Cancel:    func() {},
	}
	pool.mu.Unlock()

	// Try restarting a running agent (should fail)
	_, err := pool.RestartAgent(ctx, agentID, false)
	if err == nil {
		t.Error("RestartAgent() on running agent should return error")
	}
}

func TestAgentPool_RestartAgent_NotFound(t *testing.T) {
	ctx := context.Background()
	pool, _ := createTestAgentPool(t)

	_, err := pool.RestartAgent(ctx, "nonexistent", false)
	if err == nil {
		t.Error("RestartAgent() on nonexistent agent should return error")
	}
}

// Mock SubtaskManager that returns error on AssignSubtaskToAgent
type errorSubtaskManager struct {
	*mockSubtaskManager
	assignError error
}

func (m *errorSubtaskManager) AssignSubtaskToAgent(ctx context.Context, subtaskID, agentID string) error {
	if m.assignError != nil {
		return m.assignError
	}
	return m.mockSubtaskManager.AssignSubtaskToAgent(ctx, subtaskID, agentID)
}

func TestAgentPool_Spawn_AssignError(t *testing.T) {
	ctx := context.Background()

	subtaskMgr := &errorSubtaskManager{
		mockSubtaskManager: newMockSubtaskManager(),
		assignError:        errors.New("database connection failed"),
	}

	// Add subtask
	subtask, _ := domain.NewTaskSubtask(
		"subtask-1",
		"session-1",
		"task-1",
		"Test",
		"Description",
		nil,
		nil,
	)
	subtaskMgr.addSubtask("subtask-1", subtask)

	pool := NewAgentPool(AgentPoolConfig{
		ModelSelector:  newMockModelSelector(),
		SubtaskManager: subtaskMgr,
		AgentConfig:    &config.AgentConfig{},
		SessionDirName: "test-session",
	})

	_, err := pool.Spawn(ctx, "session-1", "project-1", "subtask-1", false)
	if err == nil {
		t.Error("Spawn() with AssignSubtaskToAgent error should fail")
	}

	// Verify agent was not added to pool
	all := pool.GetAllAgents()
	if len(all) != 0 {
		t.Errorf("Pool has %d agents, want 0 (spawn failed)", len(all))
	}
}

// Mock AgentRunStorage for testing max concurrent
type mockAgentRunStorage struct {
	runs  map[string]*domain.AgentRun
	mu    sync.RWMutex
	count int
}

func newMockAgentRunStorage() *mockAgentRunStorage {
	return &mockAgentRunStorage{
		runs: make(map[string]*domain.AgentRun),
	}
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
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.runs[id], nil
}

func (m *mockAgentRunStorage) GetRunningBySession(ctx context.Context, sessionID string) ([]*domain.AgentRun, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var running []*domain.AgentRun
	for _, run := range m.runs {
		if run.SessionID == sessionID && run.Status == domain.AgentRunRunning {
			running = append(running, run)
		}
	}
	return running, nil
}

func (m *mockAgentRunStorage) CountRunningBySession(ctx context.Context, sessionID string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.count > 0 {
		return m.count, nil
	}
	count := 0
	for _, run := range m.runs {
		if run.SessionID == sessionID && run.Status == domain.AgentRunRunning {
			count++
		}
	}
	return count, nil
}

func (m *mockAgentRunStorage) CleanupOrphanedRuns(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *mockAgentRunStorage) setCount(n int) {
	m.mu.Lock()
	m.count = n
	m.mu.Unlock()
}

func TestAgentPool_MaxConcurrent_Enforcement(t *testing.T) {
	ctx := context.Background()
	subtaskMgr := newMockSubtaskManager()
	storage := newMockAgentRunStorage()

	// Create pool with max_concurrent=2
	pool := NewAgentPool(AgentPoolConfig{
		ModelSelector:   newMockModelSelector(),
		SubtaskManager:  subtaskMgr,
		AgentRunStorage: storage,
		AgentConfig:     &config.AgentConfig{},
		MaxConcurrent:   2,
	})

	// Add subtasks
	for i := 1; i <= 3; i++ {
		subtask, _ := domain.NewTaskSubtask(
			"subtask-"+string(rune('0'+i)),
			"session-1",
			"task-1",
			"Test",
			"Description",
			nil,
			nil,
		)
		subtaskMgr.addSubtask("subtask-"+string(rune('0'+i)), subtask)
	}

	// Simulate 2 agents already running (via setCount)
	storage.setCount(2)

	// Spawn third agent - should FAIL (max_concurrent=2, 2 already running)
	agentID3, err := pool.Spawn(ctx, "session-1", "project-1", "subtask-3", false)
	if err == nil {
		t.Error("Spawn() should fail due to max_concurrent limit")
	}
	if agentID3 != "" {
		t.Errorf("Spawn() returned agentID = %v, want empty", agentID3)
	}

	// Verify error message contains "max concurrent"
	if err != nil && !strings.Contains(err.Error(), "max concurrent") {
		t.Errorf("Error message = %v, want 'max concurrent'", err.Error())
	}

	// Now test successful spawn when under limit
	storage.setCount(1) // only 1 running
	agentID1, err := pool.Spawn(ctx, "session-1", "project-1", "subtask-1", false)
	if err != nil {
		t.Fatalf("Spawn() with count=1 should succeed, error = %v", err)
	}
	if agentID1 == "" {
		t.Error("Spawn() returned empty agentID")
	}
}

func TestAgentPool_MaxConcurrent_Zero_NoLimit(t *testing.T) {
	ctx := context.Background()
	subtaskMgr := newMockSubtaskManager()
	storage := newMockAgentRunStorage()

	// Create pool with max_concurrent=0 (no limit)
	pool := NewAgentPool(AgentPoolConfig{
		ModelSelector:   newMockModelSelector(),
		SubtaskManager:  subtaskMgr,
		AgentRunStorage: storage,
		AgentConfig:     &config.AgentConfig{},
		MaxConcurrent:   0, // no limit
	})

	// Add subtasks
	for i := 1; i <= 5; i++ {
		subtask, _ := domain.NewTaskSubtask(
			"subtask-"+string(rune('0'+i)),
			"session-1",
			"task-1",
			"Test",
			"Description",
			nil,
			nil,
		)
		subtaskMgr.addSubtask("subtask-"+string(rune('0'+i)), subtask)
	}

	// Spawn 5 agents - all should succeed
	for i := 1; i <= 5; i++ {
		agentID, err := pool.Spawn(ctx, "session-1", "project-1", "subtask-"+string(rune('0'+i)), false)
		if err != nil {
			t.Fatalf("Spawn #%d error = %v", i, err)
		}
		if agentID == "" {
			t.Errorf("Spawn #%d returned empty agentID", i)
		}
	}

	// Verify all 5 agents in pool
	all := pool.GetAllAgents()
	if len(all) != 5 {
		t.Errorf("Pool has %d agents, want 5", len(all))
	}
}

func TestAgentPool_MaxConcurrent_NoStorage_NoLimit(t *testing.T) {
	ctx := context.Background()
	subtaskMgr := newMockSubtaskManager()

	// Create pool with max_concurrent=2 but NO storage (backward compatibility)
	pool := NewAgentPool(AgentPoolConfig{
		ModelSelector:   newMockModelSelector(),
		SubtaskManager:  subtaskMgr,
		AgentRunStorage: nil, // no storage
		AgentConfig:     &config.AgentConfig{},
		MaxConcurrent:   2,
	})

	// Add subtasks
	for i := 1; i <= 3; i++ {
		subtask, _ := domain.NewTaskSubtask(
			"subtask-"+string(rune('0'+i)),
			"session-1",
			"task-1",
			"Test",
			"Description",
			nil,
			nil,
		)
		subtaskMgr.addSubtask("subtask-"+string(rune('0'+i)), subtask)
	}

	// Spawn 3 agents - all should succeed (no storage = no enforcement)
	for i := 1; i <= 3; i++ {
		agentID, err := pool.Spawn(ctx, "session-1", "project-1", "subtask-"+string(rune('0'+i)), false)
		if err != nil {
			t.Fatalf("Spawn #%d error = %v", i, err)
		}
		if agentID == "" {
			t.Errorf("Spawn #%d returned empty agentID", i)
		}
	}

	// Verify all 3 agents in pool
	all := pool.GetAllAgents()
	if len(all) != 3 {
		t.Errorf("Pool has %d agents, want 3 (no enforcement without storage)", len(all))
	}
}

// --- Safety net tests for interrupt mechanism (Этап 0) ---

// TestAgentPool_InterruptFlow verifies the full interrupt lifecycle:
// Setup interrupt → agents running → NotifyUserMessage → WaitForAllSessionAgents returns interrupted.
func TestAgentPool_InterruptFlow(t *testing.T) {
	pool, _ := createTestAgentPool(t)

	sessionID := "session-interrupt-1"

	// Manually add a "running" agent with completionCh that never closes (simulates long-running agent)
	agentID := "code-agent-int-1"
	running := &RunningAgent{
		ID:           agentID,
		SubtaskID:    "subtask-int-1",
		SessionID:    sessionID,
		ProjectKey:   "project-1",
		Status:       "running",
		StartedAt:    time.Now(),
		Cancel:       func() {},
		completionCh: make(chan struct{}),
		blockingSpawn: true,
	}

	pool.mu.Lock()
	pool.agents[agentID] = running
	pool.mu.Unlock()

	// Start WaitForAllSessionAgents in a goroutine — it should block until interrupt
	waitDone := make(chan WaitResult, 1)
	waitErr := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		result, err := pool.WaitForAllSessionAgents(ctx, sessionID)
		waitDone <- result
		waitErr <- err
	}()

	// Give WaitForAllSessionAgents time to start and register interrupt ctx
	time.Sleep(50 * time.Millisecond)

	// Verify interrupt context was created
	if !pool.HasBlockingWait(sessionID) {
		t.Fatal("HasBlockingWait() should return true after WaitForAllSessionAgents started")
	}

	// Notify user message (triggers interrupt)
	pool.NotifyUserMessage(sessionID, "stop everything")

	// Wait for result
	select {
	case result := <-waitDone:
		err := <-waitErr
		if err != nil {
			t.Fatalf("WaitForAllSessionAgents() error = %v", err)
		}
		if !result.Interrupted {
			t.Error("WaitResult.Interrupted should be true")
		}
		if result.AllDone {
			t.Error("WaitResult.AllDone should be false (agent still running)")
		}
		if result.UserMessage != "stop everything" {
			t.Errorf("WaitResult.UserMessage = %q, want %q", result.UserMessage, "stop everything")
		}
		if !result.IsInterruptResponder {
			t.Error("WaitResult.IsInterruptResponder should be true (first responder)")
		}
		if len(result.StillRunning) != 1 || result.StillRunning[0] != agentID {
			t.Errorf("WaitResult.StillRunning = %v, want [%s]", result.StillRunning, agentID)
		}

	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for WaitForAllSessionAgents to return after interrupt")
	}
}

// TestAgentPool_InterruptFlow_AllDone verifies that WaitForAllSessionAgents returns AllDone
// when all agents complete before any interrupt.
func TestAgentPool_InterruptFlow_AllDone(t *testing.T) {
	pool, _ := createTestAgentPool(t)

	sessionID := "session-alldone-1"

	// Add agent with completionCh that we'll close immediately
	completionCh := make(chan struct{})
	agentID := "code-agent-done-1"
	running := &RunningAgent{
		ID:            agentID,
		SubtaskID:     "subtask-done-1",
		SessionID:     sessionID,
		ProjectKey:    "project-1",
		Status:        "running",
		StartedAt:     time.Now(),
		Cancel:        func() {},
		completionCh:  completionCh,
		blockingSpawn: true,
	}

	pool.mu.Lock()
	pool.agents[agentID] = running
	pool.mu.Unlock()

	// Close completionCh to simulate agent completing
	running.Status = "completed"
	running.Result = "done"
	close(completionCh)

	// WaitForAllSessionAgents should return immediately with AllDone
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := pool.WaitForAllSessionAgents(ctx, sessionID)
	if err != nil {
		t.Fatalf("WaitForAllSessionAgents() error = %v", err)
	}
	if !result.AllDone {
		t.Error("WaitResult.AllDone should be true")
	}
	if result.Interrupted {
		t.Error("WaitResult.Interrupted should be false")
	}
}

// TestAgentPool_InterruptFlow_NoRunningAgents verifies that WaitForAllSessionAgents
// returns immediately with AllDone when there are no running agents.
func TestAgentPool_InterruptFlow_NoRunningAgents(t *testing.T) {
	pool, _ := createTestAgentPool(t)

	sessionID := "session-empty-1"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := pool.WaitForAllSessionAgents(ctx, sessionID)
	if err != nil {
		t.Fatalf("WaitForAllSessionAgents() error = %v", err)
	}
	if !result.AllDone {
		t.Error("WaitResult.AllDone should be true when no running agents")
	}
}

// TestAgentPool_ClaimInterruptResponder_Concurrent verifies that only one goroutine
// can claim the interrupt responder role for a given session.
func TestAgentPool_ClaimInterruptResponder_Concurrent(t *testing.T) {
	pool, _ := createTestAgentPool(t)

	sessionID := "session-claim-1"

	// Setup interrupt context via InterruptManager public API
	pool.interrupt.GetOrCreateInterruptCtx(sessionID)

	// Launch multiple goroutines trying to claim
	const goroutines = 10
	results := make(chan bool, goroutines)
	var wg sync.WaitGroup

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			claimed := pool.interrupt.ClaimInterruptResponder(sessionID)
			results <- claimed
		}()
	}

	wg.Wait()
	close(results)

	// Count how many goroutines claimed
	claimedCount := 0
	for claimed := range results {
		if claimed {
			claimedCount++
		}
	}

	if claimedCount != 1 {
		t.Errorf("Expected exactly 1 goroutine to claim, got %d", claimedCount)
	}
}

// TestAgentPool_ExtractUserMessage verifies user message extraction from interrupt error.
func TestAgentPool_ExtractUserMessage(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		want    string
	}{
		{"nil error", nil, ""},
		{"user message", errors.New("user_message:hello world"), "hello world"},
		{"user message empty", errors.New("user_message:"), ""},
		{"non-user error", errors.New("some other error"), "some other error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractUserMessage(tt.err)
			if got != tt.want {
				t.Errorf("ExtractUserMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestAgentPool_NotifyUserMessage_NoWaiter verifies that NotifyUserMessage
// is a no-op when there's no active waiter for the session.
func TestAgentPool_NotifyUserMessage_NoWaiter(t *testing.T) {
	pool, _ := createTestAgentPool(t)

	// Should not panic or error when no interrupt context exists
	pool.NotifyUserMessage("nonexistent-session", "hello")

	// Verify no interrupt context was created
	if pool.HasBlockingWait("nonexistent-session") {
		t.Error("HasBlockingWait() should return false for session with no waiter")
	}
}

// TestAgentPool_CleanupInterruptCtx verifies that interrupt context is cleaned up
// when no more running agents exist for the session.
func TestAgentPool_CleanupInterruptCtx(t *testing.T) {
	pool, _ := createTestAgentPool(t)

	sessionID := "session-cleanup-1"

	// Setup interrupt context via InterruptManager
	pool.interrupt.GetOrCreateInterruptCtx(sessionID)

	// Verify it exists
	if !pool.HasBlockingWait(sessionID) {
		t.Fatal("HasBlockingWait() should return true after GetOrCreateInterruptCtx")
	}

	// Add a completed (non-running) agent
	pool.mu.Lock()
	pool.agents["agent-completed"] = &RunningAgent{
		ID:        "agent-completed",
		SessionID: sessionID,
		Status:    "completed",
		Cancel:    func() {},
	}
	pool.mu.Unlock()

	// Cleanup — no running agents, should remove interrupt ctx
	pool.cleanupInterruptCtx(sessionID)

	if pool.HasBlockingWait(sessionID) {
		t.Error("HasBlockingWait() should return false after cleanup with no running agents")
	}
}

// TestAgentPool_CleanupInterruptCtx_WithRunningAgent verifies that interrupt context
// is NOT cleaned up when a running agent still exists.
func TestAgentPool_CleanupInterruptCtx_WithRunningAgent(t *testing.T) {
	pool, _ := createTestAgentPool(t)

	sessionID := "session-noclean-1"

	// Setup interrupt context via InterruptManager
	pool.interrupt.GetOrCreateInterruptCtx(sessionID)

	// Add a running agent
	pool.mu.Lock()
	pool.agents["agent-running"] = &RunningAgent{
		ID:        "agent-running",
		SessionID: sessionID,
		Status:    "running",
		Cancel:    func() {},
	}
	pool.mu.Unlock()

	// Cleanup — running agent still exists, should keep interrupt ctx
	pool.cleanupInterruptCtx(sessionID)

	if !pool.HasBlockingWait(sessionID) {
		t.Error("HasBlockingWait() should remain true when running agent exists")
	}
}

// --- Phase 1: SpawnWithDescription tests ---

func TestAgentPool_SpawnWithDescription(t *testing.T) {
	ctx := context.Background()
	pool, _ := createTestAgentPool(t)

	agentID, err := pool.SpawnWithDescription(ctx, "session-1", "project-1", domain.FlowType("researcher"), "Research codebase structure", false)
	if err != nil {
		t.Fatalf("SpawnWithDescription() error = %v", err)
	}

	if agentID == "" {
		t.Error("SpawnWithDescription() returned empty agentID")
	}

	if !strings.HasPrefix(agentID, "researcher-") {
		t.Errorf("agentID = %v, want prefix 'researcher-'", agentID)
	}

	snap, ok := pool.GetStatus(agentID)
	if !ok {
		t.Fatal("Agent not found in pool after SpawnWithDescription()")
	}

	if snap.Status != "running" && snap.Status != "failed" {
		t.Errorf("Agent status = %v, want 'running' or 'failed'", snap.Status)
	}

	if snap.SubtaskID != "" {
		t.Errorf("Agent.SubtaskID = %v, want empty (no subtask)", snap.SubtaskID)
	}

	if snap.SessionID != "session-1" {
		t.Errorf("Agent.SessionID = %v, want 'session-1'", snap.SessionID)
	}
}

func TestAgentPool_SpawnWithDescription_Reviewer(t *testing.T) {
	ctx := context.Background()
	pool, _ := createTestAgentPool(t)

	agentID, err := pool.SpawnWithDescription(ctx, "session-1", "project-1", domain.FlowType("reviewer"), "Review code quality", false)
	if err != nil {
		t.Fatalf("SpawnWithDescription() error = %v", err)
	}

	if !strings.HasPrefix(agentID, "reviewer-") {
		t.Errorf("agentID = %v, want prefix 'reviewer-'", agentID)
	}
}

func TestAgentPool_MarkCompleted_EmptySubtaskID(t *testing.T) {
	pool, subtaskMgr := createTestAgentPool(t)

	// Create a researcher agent manually (no subtask)
	agentID := "researcher-test123"
	running := &RunningAgent{
		ID:        agentID,
		SubtaskID: "", // no subtask
		SessionID: "session-1",
		Status:    "running",
		StartedAt: time.Now(),
		Cancel:    func() {},
		flowType:  domain.FlowType("researcher"),
	}

	pool.mu.Lock()
	pool.agents[agentID] = running
	pool.mu.Unlock()

	// Mark as completed
	pool.markCompleted(agentID, "", "Research results")

	time.Sleep(50 * time.Millisecond)

	snap, ok := pool.GetStatus(agentID)
	if !ok {
		t.Fatal("Agent disappeared after markCompleted()")
	}

	if snap.Status != "completed" {
		t.Errorf("Agent.Status = %v, want 'completed'", snap.Status)
	}

	// CompleteSubtask should NOT have been called (empty subtaskID guard)
	subtaskMgr.mu.RLock()
	completeCalls := len(subtaskMgr.completeCalls)
	subtaskMgr.mu.RUnlock()

	if completeCalls != 0 {
		t.Errorf("CompleteSubtask called %d times, want 0 (no subtask for researcher)", completeCalls)
	}
}

func TestAgentPool_MarkFailed_EmptySubtaskID(t *testing.T) {
	pool, subtaskMgr := createTestAgentPool(t)

	// Create a reviewer agent manually (no subtask)
	agentID := "reviewer-test456"
	running := &RunningAgent{
		ID:        agentID,
		SubtaskID: "", // no subtask
		SessionID: "session-1",
		Status:    "running",
		StartedAt: time.Now(),
		Cancel:    func() {},
		flowType:  domain.FlowType("reviewer"),
	}

	pool.mu.Lock()
	pool.agents[agentID] = running
	pool.mu.Unlock()

	pool.markFailed(agentID, "", "Review failed: timeout")

	time.Sleep(50 * time.Millisecond)

	snap, ok := pool.GetStatus(agentID)
	if !ok {
		t.Fatal("Agent disappeared after markFailed()")
	}

	if snap.Status != "failed" {
		t.Errorf("Agent.Status = %v, want 'failed'", snap.Status)
	}

	// FailSubtask should NOT have been called (empty subtaskID guard)
	subtaskMgr.mu.RLock()
	failCalls := len(subtaskMgr.failCalls)
	subtaskMgr.mu.RUnlock()

	if failCalls != 0 {
		t.Errorf("FailSubtask called %d times, want 0 (no subtask for reviewer)", failCalls)
	}
}

func TestAgentPool_RestartAgent_Researcher(t *testing.T) {
	ctx := context.Background()
	pool, _ := createTestAgentPool(t)

	// Add a failed researcher agent
	oldAgentID := "researcher-old"
	pool.mu.Lock()
	pool.agents[oldAgentID] = &RunningAgent{
		ID:          oldAgentID,
		SubtaskID:   "", // no subtask
		SessionID:   "session-1",
		ProjectKey:  "project-1",
		Status:      "failed",
		Error:       "Previous failure",
		StartedAt:   time.Now().Add(-5 * time.Minute),
		Cancel:      func() {},
		flowType:    domain.FlowType("researcher"),
		description: "Research codebase architecture",
	}
	pool.mu.Unlock()

	newAgentID, err := pool.RestartAgent(ctx, oldAgentID, false)
	if err != nil {
		t.Fatalf("RestartAgent() error = %v", err)
	}

	if newAgentID == "" {
		t.Error("RestartAgent() returned empty agentID")
	}

	if newAgentID == oldAgentID {
		t.Error("RestartAgent() should create new agent")
	}

	if !strings.HasPrefix(newAgentID, "researcher-") {
		t.Errorf("New agentID = %v, want prefix 'researcher-'", newAgentID)
	}

	newSnap, ok := pool.GetStatus(newAgentID)
	if !ok {
		t.Fatal("New agent not found after RestartAgent()")
	}

	if newSnap.Status != "running" && newSnap.Status != "failed" {
		t.Errorf("New agent status = %v, want 'running' or 'failed'", newSnap.Status)
	}

	if newSnap.SubtaskID != "" {
		t.Errorf("New agent subtaskID = %v, want empty", newSnap.SubtaskID)
	}
}

func TestFirstLine(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		maxLen int
		want   string
	}{
		{"simple", "hello world", 50, "hello world"},
		{"with newline", "first line\nsecond line", 50, "first line"},
		{"truncated", "very long description text", 10, "very long ..."},
		{"exact length", "12345", 5, "12345"},
		{"empty", "", 50, ""},
		{"unicode", "Исследовать архитектуру", 12, "Исследовать ..."},
		{"unicode no truncate", "Привет", 10, "Привет"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := firstLine(tt.text, tt.maxLen)
			if got != tt.want {
				t.Errorf("firstLine(%q, %d) = %q, want %q", tt.text, tt.maxLen, got, tt.want)
			}
		})
	}
}

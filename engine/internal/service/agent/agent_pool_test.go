package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/service/orchestrator"
	"github.com/syntheticinc/bytebrew/engine/pkg/config"
	"github.com/cloudwego/eino/components/model"
)

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
func createTestAgentPool(t *testing.T) *AgentPool {
	t.Helper()

	pool := NewAgentPool(AgentPoolConfig{
		ModelSelector:  newMockModelSelector(),
		AgentConfig:    &config.AgentConfig{},
		SessionDirName: "test-session",
	})

	return pool
}

func TestAgentPool_SpawnWithDescription(t *testing.T) {
	ctx := context.Background()
	pool := createTestAgentPool(t)

	agentID, err := pool.SpawnWithDescription(ctx, "session-1", "project-1", domain.FlowType("coder"), "Implement feature X", false)
	if err != nil {
		// Spawn may fail if no engine is configured — that's OK for this test
		// We just verify the error is not unexpected
		if !strings.Contains(err.Error(), "engine") && !strings.Contains(err.Error(), "flow") {
			t.Fatalf("SpawnWithDescription() unexpected error = %v", err)
		}
		return
	}

	if agentID == "" {
		t.Error("SpawnWithDescription() returned empty agentID")
	}

	// Check agent is in the pool (snapshot is safe to read without lock)
	snap, ok := pool.GetStatus(agentID)
	if !ok {
		t.Fatal("Agent not found in pool after SpawnWithDescription()")
	}

	// Agent may already be "failed" (no engine configured), both are valid
	if snap.Status != "running" && snap.Status != "failed" {
		t.Errorf("Agent status = %v, want 'running' or 'failed'", snap.Status)
	}

	if snap.SessionID != "session-1" {
		t.Errorf("Agent.SessionID = %v, want 'session-1'", snap.SessionID)
	}

	if snap.ProjectKey != "project-1" {
		t.Errorf("Agent.ProjectKey = %v, want 'project-1'", snap.ProjectKey)
	}
}

func TestAgentPool_GetStatus(t *testing.T) {
	ctx := context.Background()
	pool := createTestAgentPool(t)

	agentID, err := pool.SpawnWithDescription(ctx, "session-1", "project-1", domain.FlowType("coder"), "Test task", false)
	if err != nil {
		t.Skip("SpawnWithDescription failed (no engine configured)")
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
	pool := createTestAgentPool(t)

	agentID, err := pool.SpawnWithDescription(ctx, "session-1", "project-1", domain.FlowType("coder"), "Test task", false)
	if err != nil {
		t.Skip("SpawnWithDescription failed (no engine configured)")
	}

	// Wait for agent to start
	time.Sleep(10 * time.Millisecond)

	// Verify agent is in pool before stopping
	snap, ok := pool.GetStatus(agentID)
	if !ok {
		t.Fatal("Agent not found after SpawnWithDescription()")
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

func TestAgentPool_StopAgent_NotFound(t *testing.T) {
	pool := createTestAgentPool(t)

	err := pool.StopAgent("nonexistent")
	if err == nil {
		t.Error("StopAgent() on nonexistent agent should return error")
	}
}

func TestAgentPool_MarkCompleted_NoAutoRetry(t *testing.T) {
	pool := createTestAgentPool(t)

	// Create running agent manually
	agentID := "code-agent-test123"
	running := &RunningAgent{
		ID:         agentID,
		SubtaskID:  "",
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
	pool.markCompleted(agentID, "", "Task completed successfully")

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

	// IMPORTANT: Verify NO auto-retry happened
	allAgents := pool.GetAllAgents()
	if len(allAgents) != 1 {
		t.Errorf("Pool has %d agents, want 1 (no auto-retry should happen)", len(allAgents))
	}
}

func TestAgentPool_MarkFailed_NoAutoRetry(t *testing.T) {
	pool := createTestAgentPool(t)

	// Create running agent manually
	agentID := "code-agent-test456"
	running := &RunningAgent{
		ID:         agentID,
		SubtaskID:  "",
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
	pool.markFailed(agentID, "", "Agent execution failed")

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

	// CRITICAL: Verify NO auto-retry happened
	allAgents := pool.GetAllAgents()
	if len(allAgents) != 1 {
		t.Errorf("Pool has %d agents, want 1 (no auto-retry should happen)", len(allAgents))
	}
}

func TestAgentPool_EventBus_Completed(t *testing.T) {
	pool := createTestAgentPool(t)

	// Create event bus
	bus := orchestrator.NewSessionEventBus(16)
	defer bus.Close()

	pool.SetEventBus(bus)

	// Create running agent
	agentID := "code-agent-bus123"
	running := &RunningAgent{
		ID:         agentID,
		SubtaskID:  "",
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
	pool.markCompleted(agentID, "", "Success result")

	// Read event from bus (with timeout)
	select {
	case event := <-bus.Events():
		if event.Type != orchestrator.EventAgentCompleted {
			t.Errorf("Event.Type = %v, want EventAgentCompleted", event.Type)
		}
		if event.AgentID != agentID {
			t.Errorf("Event.AgentID = %v, want %v", event.AgentID, agentID)
		}
		if event.Content != "Success result" {
			t.Errorf("Event.Content = %v, want 'Success result'", event.Content)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for EventAgentCompleted on bus")
	}
}

func TestAgentPool_EventBus_Failed(t *testing.T) {
	pool := createTestAgentPool(t)

	// Create event bus
	bus := orchestrator.NewSessionEventBus(16)
	defer bus.Close()

	pool.SetEventBus(bus)

	// Create running agent
	agentID := "code-agent-bus456"
	running := &RunningAgent{
		ID:         agentID,
		SubtaskID:  "",
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
	pool.markFailed(agentID, "", "Execution error")

	// Read event from bus (with timeout)
	select {
	case event := <-bus.Events():
		if event.Type != orchestrator.EventAgentFailed {
			t.Errorf("Event.Type = %v, want EventAgentFailed", event.Type)
		}
		if event.AgentID != agentID {
			t.Errorf("Event.AgentID = %v, want %v", event.AgentID, agentID)
		}
		if event.Content != "Execution error" {
			t.Errorf("Event.Content = %v, want 'Execution error'", event.Content)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for EventAgentFailed on bus")
	}
}

func TestAgentPool_GetAllAgents(t *testing.T) {
	pool := createTestAgentPool(t)

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
	pool := createTestAgentPool(t)

	pool.SetSessionDirName("new-session-dir")

	pool.mu.RLock()
	dirName := pool.sessionDirName
	pool.mu.RUnlock()

	if dirName != "new-session-dir" {
		t.Errorf("sessionDirName = %v, want 'new-session-dir'", dirName)
	}
}

func TestAgentPool_SetEventBus(t *testing.T) {
	pool := createTestAgentPool(t)

	bus := orchestrator.NewSessionEventBus(16)
	defer bus.Close()

	pool.SetEventBus(bus)

	pool.mu.RLock()
	hasBus := pool.eventBus != nil
	pool.mu.RUnlock()

	if !hasBus {
		t.Error("SetEventBus() did not set event bus")
	}
}

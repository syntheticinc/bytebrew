package lifecycle

import (
	"context"
	"fmt"
	"testing"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

type mockRunner struct {
	outputs map[string]string
	err     error
}

func (m *mockRunner) RunAgent(_ context.Context, agentName, input, sessionID string, _ domain.AgentEventStream) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if out, ok := m.outputs[agentName]; ok {
		return out, nil
	}
	return fmt.Sprintf("output from %s", agentName), nil
}

func TestManager_SpawnAgent_ContextDestroyed(t *testing.T) {
	runner := &mockRunner{outputs: map[string]string{"agent-a": "result-1"}}
	mgr := NewManager(runner)

	// First execution
	out, err := mgr.ExecuteTask(context.Background(), "agent-a", "session-1", "task 1",
		domain.LifecycleModeSpawn, 16000, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "result-1" {
		t.Errorf("expected result-1, got %q", out)
	}

	// After spawn, instance should be cleaned up
	_, exists := mgr.GetInstance("agent-a", "session-1")
	if exists {
		t.Error("expected spawn instance to be cleaned up after task")
	}

	// Context should be zero
	if mgr.ContextSize("agent-a", "session-1") != 0 {
		t.Error("expected zero context after spawn")
	}
}

func TestManager_PersistentAgent_ContextPreserved(t *testing.T) {
	runner := &mockRunner{outputs: map[string]string{"agent-a": "result-1"}}
	mgr := NewManager(runner)

	// First execution
	_, err := mgr.ExecuteTask(context.Background(), "agent-a", "session-1", "task 1",
		domain.LifecycleModePersistent, 16000, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Instance should still exist
	inst, exists := mgr.GetInstance("agent-a", "session-1")
	if !exists {
		t.Fatal("expected persistent instance to exist")
	}
	if inst.State() != domain.LifecycleReady {
		t.Errorf("expected ready state, got %s", inst.State())
	}

	// Context should be preserved
	if mgr.ContextSize("agent-a", "session-1") != 2 { // "User: task 1" + "Agent: result-1"
		t.Errorf("expected 2 context entries, got %d", mgr.ContextSize("agent-a", "session-1"))
	}

	// Second execution — should have context
	runner.outputs["agent-a"] = "result-2"
	_, err = mgr.ExecuteTask(context.Background(), "agent-a", "session-1", "task 2",
		domain.LifecycleModePersistent, 16000, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mgr.ContextSize("agent-a", "session-1") != 4 {
		t.Errorf("expected 4 context entries, got %d", mgr.ContextSize("agent-a", "session-1"))
	}
}

func TestManager_PersistentAgent_MultiTask(t *testing.T) {
	runner := &mockRunner{outputs: map[string]string{"agent-a": "ok"}}
	mgr := NewManager(runner)

	for i := 0; i < 3; i++ {
		_, err := mgr.ExecuteTask(context.Background(), "agent-a", "session-1", fmt.Sprintf("task %d", i),
			domain.LifecycleModePersistent, 16000, nil)
		if err != nil {
			t.Fatalf("task %d: unexpected error: %v", i, err)
		}
	}

	inst, _ := mgr.GetInstance("agent-a", "session-1")
	if inst.TasksHandled != 3 {
		t.Errorf("expected 3 tasks handled, got %d", inst.TasksHandled)
	}
}

func TestManager_ResetAgent(t *testing.T) {
	runner := &mockRunner{outputs: map[string]string{"agent-a": "ok"}}
	mgr := NewManager(runner)

	mgr.ExecuteTask(context.Background(), "agent-a", "session-1", "task 1",
		domain.LifecycleModePersistent, 16000, nil)

	mgr.ResetAgent("agent-a", "session-1")

	if mgr.ContextSize("agent-a", "session-1") != 0 {
		t.Error("expected zero context after reset")
	}
}

func TestManager_SpawnAgent_ReSpawn(t *testing.T) {
	callCount := 0
	runner := &mockRunner{outputs: map[string]string{"agent-a": "ok"}}
	_ = callCount

	mgr := NewManager(runner)

	// First spawn
	mgr.ExecuteTask(context.Background(), "agent-a", "session-1", "task 1",
		domain.LifecycleModeSpawn, 16000, nil)

	// Second spawn — fresh instance
	mgr.ExecuteTask(context.Background(), "agent-a", "session-1", "task 2",
		domain.LifecycleModeSpawn, 16000, nil)

	// Instance should be gone (spawn cleans up)
	_, exists := mgr.GetInstance("agent-a", "session-1")
	if exists {
		t.Error("expected no instance after spawn task")
	}
}

func TestManager_AgentFailure(t *testing.T) {
	runner := &mockRunner{err: fmt.Errorf("LLM error")}
	mgr := NewManager(runner)

	_, err := mgr.ExecuteTask(context.Background(), "agent-a", "session-1", "task",
		domain.LifecycleModeSpawn, 16000, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

package domain

import (
	"testing"
)

func TestNewAgentInstance(t *testing.T) {
	ai := NewAgentInstance("agent-a", LifecycleModeSpawn, 16000)
	if ai.AgentName != "agent-a" {
		t.Errorf("expected agent-a, got %s", ai.AgentName)
	}
	if ai.State() != LifecycleInitializing {
		t.Errorf("expected initializing, got %s", ai.State())
	}
	if ai.IsPersistent() {
		t.Error("expected not persistent for spawn mode")
	}
}

func TestAgentInstance_SpawnLifecycle(t *testing.T) {
	ai := NewAgentInstance("agent-a", LifecycleModeSpawn, 16000)

	if err := ai.MarkReady(); err != nil {
		t.Fatalf("mark ready: %v", err)
	}
	if ai.State() != LifecycleReady {
		t.Errorf("expected ready, got %s", ai.State())
	}

	if err := ai.MarkRunning(); err != nil {
		t.Fatalf("mark running: %v", err)
	}
	if ai.State() != LifecycleRunning {
		t.Errorf("expected running, got %s", ai.State())
	}

	if err := ai.FinishTask(); err != nil {
		t.Fatalf("finish task: %v", err)
	}
	// Spawn agent → finished (terminal)
	if ai.State() != LifecycleFinished {
		t.Errorf("expected finished for spawn, got %s", ai.State())
	}
	if ai.TasksHandled != 1 {
		t.Errorf("expected 1 task handled, got %d", ai.TasksHandled)
	}
}

func TestAgentInstance_PersistentLifecycle(t *testing.T) {
	ai := NewAgentInstance("agent-a", LifecycleModePersistent, 16000)

	ai.MarkReady()
	ai.MarkRunning()

	if err := ai.FinishTask(); err != nil {
		t.Fatalf("finish task: %v", err)
	}
	// Persistent agent → back to ready
	if ai.State() != LifecycleReady {
		t.Errorf("expected ready for persistent after finish, got %s", ai.State())
	}

	// Can execute another task
	ai.MarkRunning()
	if err := ai.FinishTask(); err != nil {
		t.Fatalf("finish second task: %v", err)
	}
	if ai.TasksHandled != 2 {
		t.Errorf("expected 2 tasks handled, got %d", ai.TasksHandled)
	}
}

func TestAgentInstance_NeedsCompaction(t *testing.T) {
	ai := NewAgentInstance("agent-a", LifecycleModePersistent, 1000)

	ai.ContextTokens = 500
	if ai.NeedsCompaction() {
		t.Error("expected no compaction at 50%")
	}

	ai.ContextTokens = 850
	if !ai.NeedsCompaction() {
		t.Error("expected compaction at 85%")
	}
}

func TestAgentInstance_NeedsCompaction_NoMax(t *testing.T) {
	ai := NewAgentInstance("agent-a", LifecycleModePersistent, 0)
	ai.ContextTokens = 999999
	if ai.NeedsCompaction() {
		t.Error("expected no compaction with maxContext=0")
	}
}

func TestAgentInstance_ResetContext(t *testing.T) {
	ai := NewAgentInstance("agent-a", LifecycleModeSpawn, 16000)
	ai.ContextTokens = 5000
	ai.ResetContext()
	if ai.ContextTokens != 0 {
		t.Errorf("expected 0 tokens after reset, got %d", ai.ContextTokens)
	}
}

func TestAgentInstance_IsPersistent(t *testing.T) {
	spawn := NewAgentInstance("a", LifecycleModeSpawn, 0)
	persistent := NewAgentInstance("b", LifecycleModePersistent, 0)

	if spawn.IsPersistent() {
		t.Error("spawn should not be persistent")
	}
	if !persistent.IsPersistent() {
		t.Error("persistent should be persistent")
	}
}

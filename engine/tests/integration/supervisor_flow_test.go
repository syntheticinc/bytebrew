package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence"
	"github.com/syntheticinc/bytebrew/engine/internal/service/orchestrator"
	"github.com/syntheticinc/bytebrew/engine/internal/service/work"
)

// setupWorkManager creates a work.Manager with real SQLite storage for integration tests
func setupWorkManager(t *testing.T) (*work.Manager, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "integration_test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "integration.db")
	db, err := persistence.NewWorkDB(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("open work db: %v", err)
	}

	taskStorage, err := persistence.NewSQLiteTaskStorage(db)
	if err != nil {
		db.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("create task storage: %v", err)
	}

	subtaskStorage, err := persistence.NewSQLiteSubtaskStorage(db)
	if err != nil {
		db.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("create subtask storage: %v", err)
	}

	manager := work.New(taskStorage, subtaskStorage)
	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return manager, cleanup
}

// TestSupervisorFlow_FullLifecycle tests the complete task lifecycle:
// Create → Approve → Start → Create Subtasks → Assign → Complete Subtasks → Complete Task
func TestSupervisorFlow_FullLifecycle(t *testing.T) {
	manager, cleanup := setupWorkManager(t)
	defer cleanup()
	ctx := context.Background()

	// Step 1: Create task
	task, err := manager.CreateTask(ctx, "sess-1", "Add auth endpoint", "JWT auth for API", []string{"Tests pass", "Login works"})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if task.Status != domain.TaskStatusDraft {
		t.Errorf("expected draft status, got %s", task.Status)
	}

	// Step 2: Approve task
	if err := manager.ApproveTask(ctx, task.ID); err != nil {
		t.Fatalf("approve task: %v", err)
	}

	// Step 3: Start task
	if err := manager.StartTask(ctx, task.ID); err != nil {
		t.Fatalf("start task: %v", err)
	}

	// Step 4: Create subtasks
	sub1, err := manager.CreateSubtask(ctx, "sess-1", task.ID, "Create proto schema", "Define auth proto", nil, []string{"api/auth.proto"})
	if err != nil {
		t.Fatalf("create subtask 1: %v", err)
	}

	sub2, err := manager.CreateSubtask(ctx, "sess-1", task.ID, "Implement handler", "Write handler code",
		[]string{sub1.ID}, []string{"internal/delivery/auth_handler.go"})
	if err != nil {
		t.Fatalf("create subtask 2: %v", err)
	}

	sub3, err := manager.CreateSubtask(ctx, "sess-1", task.ID, "Write tests", "E2E tests",
		[]string{sub2.ID}, []string{"tests/auth_test.go"})
	if err != nil {
		t.Fatalf("create subtask 3: %v", err)
	}

	// Step 5: Only sub1 should be ready (no blockers)
	ready, err := manager.GetReadySubtasks(ctx, task.ID)
	if err != nil {
		t.Fatalf("get ready subtasks: %v", err)
	}
	if len(ready) != 1 || ready[0].ID != sub1.ID {
		t.Fatalf("expected only sub1 to be ready, got %d ready subtasks", len(ready))
	}

	// Step 6: Assign and complete sub1
	if err := manager.AssignSubtaskToAgent(ctx, sub1.ID, "code-agent-001"); err != nil {
		t.Fatalf("assign sub1: %v", err)
	}
	if err := manager.CompleteSubtask(ctx, sub1.ID, "Proto schema created"); err != nil {
		t.Fatalf("complete sub1: %v", err)
	}

	// Step 7: Now sub2 should be ready
	ready2, err := manager.GetReadySubtasks(ctx, task.ID)
	if err != nil {
		t.Fatalf("get ready subtasks after sub1: %v", err)
	}
	if len(ready2) != 1 || ready2[0].ID != sub2.ID {
		t.Fatalf("expected only sub2 to be ready, got %d", len(ready2))
	}

	// Step 8: Assign and complete sub2
	if err := manager.AssignSubtaskToAgent(ctx, sub2.ID, "code-agent-002"); err != nil {
		t.Fatalf("assign sub2: %v", err)
	}
	if err := manager.CompleteSubtask(ctx, sub2.ID, "Handler implemented"); err != nil {
		t.Fatalf("complete sub2: %v", err)
	}

	// Step 9: Now sub3 should be ready
	ready3, err := manager.GetReadySubtasks(ctx, task.ID)
	if err != nil {
		t.Fatalf("get ready subtasks after sub2: %v", err)
	}
	if len(ready3) != 1 || ready3[0].ID != sub3.ID {
		t.Fatalf("expected only sub3 to be ready, got %d", len(ready3))
	}

	// Step 10: Complete sub3
	if err := manager.AssignSubtaskToAgent(ctx, sub3.ID, "code-agent-003"); err != nil {
		t.Fatalf("assign sub3: %v", err)
	}
	if err := manager.CompleteSubtask(ctx, sub3.ID, "Tests written and pass"); err != nil {
		t.Fatalf("complete sub3: %v", err)
	}

	// Step 11: Complete task (all subtasks done)
	if err := manager.CompleteTask(ctx, task.ID); err != nil {
		t.Fatalf("complete task: %v", err)
	}

	// Verify final state
	finalTask, err := manager.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("get final task: %v", err)
	}
	if finalTask.Status != domain.TaskStatusCompleted {
		t.Errorf("expected completed, got %s", finalTask.Status)
	}
	if finalTask.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
}

// TestSupervisorFlow_SubtaskBlocking tests that blocked_by dependencies are respected
func TestSupervisorFlow_SubtaskBlocking(t *testing.T) {
	manager, cleanup := setupWorkManager(t)
	defer cleanup()
	ctx := context.Background()

	task, _ := manager.CreateTask(ctx, "sess-1", "Task", "desc", nil)
	manager.ApproveTask(ctx, task.ID)
	manager.StartTask(ctx, task.ID)

	// Create chain: A → B → C (B blocked by A, C blocked by B)
	subA, _ := manager.CreateSubtask(ctx, "sess-1", task.ID, "A", "first", nil, nil)
	subB, _ := manager.CreateSubtask(ctx, "sess-1", task.ID, "B", "second", []string{subA.ID}, nil)
	subC, _ := manager.CreateSubtask(ctx, "sess-1", task.ID, "C", "third", []string{subB.ID}, nil)

	// Initially only A is ready
	ready, _ := manager.GetReadySubtasks(ctx, task.ID)
	if len(ready) != 1 || ready[0].ID != subA.ID {
		t.Fatalf("expected only A ready, got %d subtasks", len(ready))
	}

	// Complete A → B should become ready, C still blocked
	manager.AssignSubtaskToAgent(ctx, subA.ID, "agent-1")
	manager.CompleteSubtask(ctx, subA.ID, "done")

	ready, _ = manager.GetReadySubtasks(ctx, task.ID)
	if len(ready) != 1 || ready[0].ID != subB.ID {
		t.Fatalf("expected only B ready after A completed, got %d subtasks", len(ready))
	}

	// Complete B → C should become ready
	manager.AssignSubtaskToAgent(ctx, subB.ID, "agent-2")
	manager.CompleteSubtask(ctx, subB.ID, "done")

	ready, _ = manager.GetReadySubtasks(ctx, task.ID)
	if len(ready) != 1 || ready[0].ID != subC.ID {
		t.Fatalf("expected only C ready after B completed, got %d subtasks", len(ready))
	}
}

// TestSupervisorFlow_TaskCompletionBlocked tests that task cannot be completed
// when subtasks are still pending/in_progress
func TestSupervisorFlow_TaskCompletionBlocked(t *testing.T) {
	manager, cleanup := setupWorkManager(t)
	defer cleanup()
	ctx := context.Background()

	task, _ := manager.CreateTask(ctx, "sess-1", "Task", "desc", nil)
	manager.ApproveTask(ctx, task.ID)
	manager.StartTask(ctx, task.ID)

	// Create two subtasks
	sub1, _ := manager.CreateSubtask(ctx, "sess-1", task.ID, "Sub 1", "desc", nil, nil)
	sub2, _ := manager.CreateSubtask(ctx, "sess-1", task.ID, "Sub 2", "desc", nil, nil)

	// Try to complete with both pending — should fail
	err := manager.CompleteTask(ctx, task.ID)
	if err == nil {
		t.Fatal("expected error: cannot complete with pending subtasks")
	}

	// Complete sub1 only
	manager.AssignSubtaskToAgent(ctx, sub1.ID, "agent-1")
	manager.CompleteSubtask(ctx, sub1.ID, "done")

	// Still should fail — sub2 is pending
	err = manager.CompleteTask(ctx, task.ID)
	if err == nil {
		t.Fatal("expected error: cannot complete with sub2 still pending")
	}

	// Fail sub2 (terminal state) — now should succeed
	manager.AssignSubtaskToAgent(ctx, sub2.ID, "agent-2")
	manager.FailSubtask(ctx, sub2.ID, "build error")

	// Now complete should work (both subtasks in terminal states)
	if err := manager.CompleteTask(ctx, task.ID); err != nil {
		t.Fatalf("expected task completion to succeed: %v", err)
	}
}

// TestSupervisorFlow_EventBusIntegration tests that events flow correctly through the bus
func TestSupervisorFlow_EventBusIntegration(t *testing.T) {
	bus := orchestrator.NewSessionEventBus(16)
	defer bus.Close()

	// Publish agent completed event
	err := bus.Publish(orchestrator.OrchestratorEvent{
		Type:      orchestrator.EventAgentCompleted,
		AgentID:   "code-agent-001",
		SubtaskID: "sub-1",
		Content:   "Code written successfully",
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}

	// Publish agent failed event
	err = bus.Publish(orchestrator.OrchestratorEvent{
		Type:      orchestrator.EventAgentFailed,
		AgentID:   "code-agent-002",
		SubtaskID: "sub-2",
		Content:   "Compilation error",
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}

	// Consume events
	events := bus.Events()

	event1 := <-events
	if event1.Type != orchestrator.EventAgentCompleted {
		t.Errorf("expected EventAgentCompleted, got %s", event1.Type)
	}
	if event1.AgentID != "code-agent-001" {
		t.Errorf("expected agent code-agent-001, got %s", event1.AgentID)
	}

	event2 := <-events
	if event2.Type != orchestrator.EventAgentFailed {
		t.Errorf("expected EventAgentFailed, got %s", event2.Type)
	}
	if event2.Content != "Compilation error" {
		t.Errorf("expected 'Compilation error', got '%s'", event2.Content)
	}
}

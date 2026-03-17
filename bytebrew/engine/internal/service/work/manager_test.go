package work

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence"
)

func setupManager(t *testing.T) (*Manager, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "manager_test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test_manager.db")
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

	manager := New(taskStorage, subtaskStorage)

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return manager, cleanup
}

// --- Task Tests ---

func TestManager_CreateTask(t *testing.T) {
	manager, cleanup := setupManager(t)
	defer cleanup()
	ctx := context.Background()

	task, err := manager.CreateTask(ctx, "sess1", "Add auth", "JWT implementation", []string{"Tests pass"})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	if task == nil {
		t.Fatal("expected task, got nil")
	}
	if task.ID == "" {
		t.Error("expected task ID to be generated")
	}
	if task.SessionID != "sess1" {
		t.Errorf("expected session_id 'sess1', got '%s'", task.SessionID)
	}
	if task.Title != "Add auth" {
		t.Errorf("expected title 'Add auth', got '%s'", task.Title)
	}
	if task.Description != "JWT implementation" {
		t.Errorf("expected description 'JWT implementation', got '%s'", task.Description)
	}
	if len(task.AcceptanceCriteria) != 1 || task.AcceptanceCriteria[0] != "Tests pass" {
		t.Errorf("unexpected criteria: %v", task.AcceptanceCriteria)
	}
	if task.Status != domain.TaskStatusDraft {
		t.Errorf("expected draft status, got %s", task.Status)
	}

	// Verify persistence
	retrieved, err := manager.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if retrieved == nil {
		t.Fatal("task not persisted")
	}
	if retrieved.Title != "Add auth" {
		t.Errorf("persisted task has wrong title: %s", retrieved.Title)
	}
}

func TestManager_ApproveTask(t *testing.T) {
	manager, cleanup := setupManager(t)
	defer cleanup()
	ctx := context.Background()

	task, _ := manager.CreateTask(ctx, "sess1", "Task", "desc", nil)

	if err := manager.ApproveTask(ctx, task.ID); err != nil {
		t.Fatalf("approve task: %v", err)
	}

	// Verify status changed
	retrieved, err := manager.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if retrieved.Status != domain.TaskStatusApproved {
		t.Errorf("expected approved status, got %s", retrieved.Status)
	}
	if retrieved.ApprovedAt == nil {
		t.Error("expected ApprovedAt to be set")
	}
}

func TestManager_ApproveTask_NotFound(t *testing.T) {
	manager, cleanup := setupManager(t)
	defer cleanup()
	ctx := context.Background()

	err := manager.ApproveTask(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent task")
	}
}

func TestManager_StartTask(t *testing.T) {
	manager, cleanup := setupManager(t)
	defer cleanup()
	ctx := context.Background()

	task, _ := manager.CreateTask(ctx, "sess1", "Task", "desc", nil)
	manager.ApproveTask(ctx, task.ID)

	if err := manager.StartTask(ctx, task.ID); err != nil {
		t.Fatalf("start task: %v", err)
	}

	retrieved, err := manager.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if retrieved.Status != domain.TaskStatusInProgress {
		t.Errorf("expected in_progress status, got %s", retrieved.Status)
	}
}

func TestManager_StartTask_NotApproved(t *testing.T) {
	manager, cleanup := setupManager(t)
	defer cleanup()
	ctx := context.Background()

	task, _ := manager.CreateTask(ctx, "sess1", "Task", "desc", nil)

	err := manager.StartTask(ctx, task.ID)
	if err == nil {
		t.Fatal("expected error when starting non-approved task")
	}
}

func TestManager_CompleteTask(t *testing.T) {
	manager, cleanup := setupManager(t)
	defer cleanup()
	ctx := context.Background()

	// Create and approve task
	task, _ := manager.CreateTask(ctx, "sess1", "Task", "desc", nil)
	manager.ApproveTask(ctx, task.ID)
	manager.StartTask(ctx, task.ID)

	// Create subtask and complete it
	subtask, _ := manager.CreateSubtask(ctx, "sess1", task.ID, "Subtask", "desc", nil, nil)
	manager.AssignSubtaskToAgent(ctx, subtask.ID, "agent-1")
	manager.CompleteSubtask(ctx, subtask.ID, "done")

	// Now complete the task
	if err := manager.CompleteTask(ctx, task.ID); err != nil {
		t.Fatalf("complete task: %v", err)
	}

	retrieved, err := manager.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if retrieved.Status != domain.TaskStatusCompleted {
		t.Errorf("expected completed status, got %s", retrieved.Status)
	}
	if retrieved.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
}

func TestManager_CompleteTask_NotAllDone(t *testing.T) {
	manager, cleanup := setupManager(t)
	defer cleanup()
	ctx := context.Background()

	task, _ := manager.CreateTask(ctx, "sess1", "Task", "desc", nil)
	manager.ApproveTask(ctx, task.ID)
	manager.StartTask(ctx, task.ID)

	// Create subtask but DON'T complete it
	manager.CreateSubtask(ctx, "sess1", task.ID, "Subtask", "desc", nil, nil)

	// Try to complete task — should fail
	err := manager.CompleteTask(ctx, task.ID)
	if err == nil {
		t.Fatal("expected error when completing task with pending subtasks")
	}
}

func TestManager_FailTask(t *testing.T) {
	manager, cleanup := setupManager(t)
	defer cleanup()
	ctx := context.Background()

	task, _ := manager.CreateTask(ctx, "sess1", "Task", "desc", nil)
	manager.ApproveTask(ctx, task.ID)
	manager.StartTask(ctx, task.ID)

	if err := manager.FailTask(ctx, task.ID, "timeout"); err != nil {
		t.Fatalf("fail task: %v", err)
	}

	retrieved, err := manager.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if retrieved.Status != domain.TaskStatusFailed {
		t.Errorf("expected failed status, got %s", retrieved.Status)
	}
}

func TestManager_GetTask(t *testing.T) {
	manager, cleanup := setupManager(t)
	defer cleanup()
	ctx := context.Background()

	task, _ := manager.CreateTask(ctx, "sess1", "Task", "desc", nil)

	retrieved, err := manager.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected task, got nil")
	}
	if retrieved.ID != task.ID {
		t.Errorf("expected ID %s, got %s", task.ID, retrieved.ID)
	}
}

func TestManager_GetTasks(t *testing.T) {
	manager, cleanup := setupManager(t)
	defer cleanup()
	ctx := context.Background()

	manager.CreateTask(ctx, "sess1", "Task 1", "desc", nil)
	manager.CreateTask(ctx, "sess1", "Task 2", "desc", nil)
	manager.CreateTask(ctx, "sess2", "Task 3", "desc", nil)

	tasks, err := manager.GetTasks(ctx, "sess1")
	if err != nil {
		t.Fatalf("get tasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks for sess1, got %d", len(tasks))
	}
}

// --- Subtask Tests ---

func TestManager_CreateSubtask(t *testing.T) {
	manager, cleanup := setupManager(t)
	defer cleanup()
	ctx := context.Background()

	task, _ := manager.CreateTask(ctx, "sess1", "Task", "desc", nil)

	subtask, err := manager.CreateSubtask(ctx, "sess1", task.ID, "Create proto",
		"Generate files", []string{}, []string{"api/auth.proto"})
	if err != nil {
		t.Fatalf("create subtask: %v", err)
	}

	if subtask == nil {
		t.Fatal("expected subtask, got nil")
	}
	if subtask.ID == "" {
		t.Error("expected subtask ID to be generated")
	}
	if subtask.SessionID != "sess1" {
		t.Errorf("expected session_id 'sess1', got '%s'", subtask.SessionID)
	}
	if subtask.TaskID != task.ID {
		t.Errorf("expected task_id '%s', got '%s'", task.ID, subtask.TaskID)
	}
	if subtask.Title != "Create proto" {
		t.Errorf("expected title 'Create proto', got '%s'", subtask.Title)
	}
	if subtask.Status != domain.SubtaskStatusPending {
		t.Errorf("expected pending status, got %s", subtask.Status)
	}
	if len(subtask.FilesInvolved) != 1 {
		t.Errorf("expected 1 file, got %d", len(subtask.FilesInvolved))
	}
}

func TestManager_GetSubtasksByTask(t *testing.T) {
	manager, cleanup := setupManager(t)
	defer cleanup()
	ctx := context.Background()

	task, _ := manager.CreateTask(ctx, "sess1", "Task", "desc", nil)
	manager.CreateSubtask(ctx, "sess1", task.ID, "Subtask 1", "desc", nil, nil)
	manager.CreateSubtask(ctx, "sess1", task.ID, "Subtask 2", "desc", nil, nil)

	subtasks, err := manager.GetSubtasksByTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("get subtasks: %v", err)
	}
	if len(subtasks) != 2 {
		t.Errorf("expected 2 subtasks, got %d", len(subtasks))
	}
}

func TestManager_GetReadySubtasks(t *testing.T) {
	manager, cleanup := setupManager(t)
	defer cleanup()
	ctx := context.Background()

	task, _ := manager.CreateTask(ctx, "sess1", "Task", "desc", nil)

	// t1: no blockers — ready
	t1, _ := manager.CreateSubtask(ctx, "sess1", task.ID, "Subtask 1", "desc", nil, nil)

	// t2: blocked by t1 (not completed) — not ready
	manager.CreateSubtask(ctx, "sess1", task.ID, "Subtask 2", "desc", []string{t1.ID}, nil)

	ready, err := manager.GetReadySubtasks(ctx, task.ID)
	if err != nil {
		t.Fatalf("get ready subtasks: %v", err)
	}
	if len(ready) != 1 {
		t.Fatalf("expected 1 ready subtask, got %d", len(ready))
	}
	if ready[0].ID != t1.ID {
		t.Errorf("expected t1 to be ready, got %s", ready[0].ID)
	}

	// Complete t1 — now t2 should be ready
	manager.AssignSubtaskToAgent(ctx, t1.ID, "agent-1")
	manager.CompleteSubtask(ctx, t1.ID, "done")

	ready2, err := manager.GetReadySubtasks(ctx, task.ID)
	if err != nil {
		t.Fatalf("get ready subtasks after completion: %v", err)
	}
	if len(ready2) != 1 {
		t.Fatalf("expected 1 ready subtask after completion, got %d", len(ready2))
	}
	// t2 should now be ready
	if ready2[0].Title != "Subtask 2" {
		t.Errorf("expected Subtask 2 to be ready, got %s", ready2[0].Title)
	}
}

func TestManager_AssignSubtaskToAgent(t *testing.T) {
	manager, cleanup := setupManager(t)
	defer cleanup()
	ctx := context.Background()

	task, _ := manager.CreateTask(ctx, "sess1", "Task", "desc", nil)
	subtask, _ := manager.CreateSubtask(ctx, "sess1", task.ID, "Subtask", "desc", nil, nil)

	if err := manager.AssignSubtaskToAgent(ctx, subtask.ID, "agent-123"); err != nil {
		t.Fatalf("assign subtask: %v", err)
	}

	retrieved, err := manager.GetSubtask(ctx, subtask.ID)
	if err != nil {
		t.Fatalf("get subtask: %v", err)
	}
	if retrieved.Status != domain.SubtaskStatusInProgress {
		t.Errorf("expected in_progress status, got %s", retrieved.Status)
	}
	if retrieved.AssignedAgentID != "agent-123" {
		t.Errorf("expected agent_id 'agent-123', got '%s'", retrieved.AssignedAgentID)
	}
}

func TestManager_CompleteSubtask(t *testing.T) {
	manager, cleanup := setupManager(t)
	defer cleanup()
	ctx := context.Background()

	task, _ := manager.CreateTask(ctx, "sess1", "Task", "desc", nil)
	subtask, _ := manager.CreateSubtask(ctx, "sess1", task.ID, "Subtask", "desc", nil, nil)
	manager.AssignSubtaskToAgent(ctx, subtask.ID, "agent-1")

	if err := manager.CompleteSubtask(ctx, subtask.ID, "finished successfully"); err != nil {
		t.Fatalf("complete subtask: %v", err)
	}

	retrieved, err := manager.GetSubtask(ctx, subtask.ID)
	if err != nil {
		t.Fatalf("get subtask: %v", err)
	}
	if retrieved.Status != domain.SubtaskStatusCompleted {
		t.Errorf("expected completed status, got %s", retrieved.Status)
	}
	if retrieved.Result != "finished successfully" {
		t.Errorf("expected result 'finished successfully', got '%s'", retrieved.Result)
	}
	if retrieved.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
}

func TestManager_FailSubtask(t *testing.T) {
	manager, cleanup := setupManager(t)
	defer cleanup()
	ctx := context.Background()

	task, _ := manager.CreateTask(ctx, "sess1", "Task", "desc", nil)
	subtask, _ := manager.CreateSubtask(ctx, "sess1", task.ID, "Subtask", "desc", nil, nil)
	manager.AssignSubtaskToAgent(ctx, subtask.ID, "agent-1")

	if err := manager.FailSubtask(ctx, subtask.ID, "timeout error"); err != nil {
		t.Fatalf("fail subtask: %v", err)
	}

	retrieved, err := manager.GetSubtask(ctx, subtask.ID)
	if err != nil {
		t.Fatalf("get subtask: %v", err)
	}
	if retrieved.Status != domain.SubtaskStatusFailed {
		t.Errorf("expected failed status, got %s", retrieved.Status)
	}
	if retrieved.Result != "timeout error" {
		t.Errorf("expected result 'timeout error', got '%s'", retrieved.Result)
	}
}

func TestManager_GetSubtaskByAgentID(t *testing.T) {
	manager, cleanup := setupManager(t)
	defer cleanup()
	ctx := context.Background()

	task, _ := manager.CreateTask(ctx, "sess1", "Task", "desc", nil)
	subtask, _ := manager.CreateSubtask(ctx, "sess1", task.ID, "Subtask", "desc", nil, nil)
	manager.AssignSubtaskToAgent(ctx, subtask.ID, "agent-xyz")

	retrieved, err := manager.GetSubtaskByAgentID(ctx, "agent-xyz")
	if err != nil {
		t.Fatalf("get subtask by agent: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected subtask, got nil")
	}
	if retrieved.ID != subtask.ID {
		t.Errorf("expected subtask ID %s, got %s", subtask.ID, retrieved.ID)
	}
}

func TestManager_GetSubtaskByAgentID_NotFound(t *testing.T) {
	manager, cleanup := setupManager(t)
	defer cleanup()
	ctx := context.Background()

	retrieved, err := manager.GetSubtaskByAgentID(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved != nil {
		t.Fatal("expected nil for nonexistent agent")
	}
}

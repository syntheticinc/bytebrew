package persistence

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
)

func setupWorkDB(t *testing.T) (*SQLiteTaskStorage, *SQLiteSubtaskStorage, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "work_db_test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test_work.db")
	db, err := NewWorkDB(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("open work db: %v", err)
	}

	taskStorage, err := NewSQLiteTaskStorage(db)
	if err != nil {
		db.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("create task storage: %v", err)
	}

	subtaskStorage, err := NewSQLiteSubtaskStorage(db)
	if err != nil {
		db.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("create subtask storage: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return taskStorage, subtaskStorage, cleanup
}

// --- Task Storage Tests ---

func TestTaskStorage_SaveAndGetByID(t *testing.T) {
	tasks, _, cleanup := setupWorkDB(t)
	defer cleanup()
	ctx := context.Background()

	task, _ := domain.NewTask("s1", "sess1", "Add auth", "JWT implementation", []string{"Tests pass"})

	if err := tasks.Save(ctx, task); err != nil {
		t.Fatalf("save task: %v", err)
	}

	got, err := tasks.GetByID(ctx, "s1")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if got == nil {
		t.Fatal("expected task, got nil")
	}
	if got.Title != "Add auth" {
		t.Errorf("expected title 'Add auth', got '%s'", got.Title)
	}
	if got.Description != "JWT implementation" {
		t.Errorf("expected description 'JWT implementation', got '%s'", got.Description)
	}
	if len(got.AcceptanceCriteria) != 1 || got.AcceptanceCriteria[0] != "Tests pass" {
		t.Errorf("unexpected criteria: %v", got.AcceptanceCriteria)
	}
	if got.Status != domain.TaskStatusDraft {
		t.Errorf("expected draft, got %s", got.Status)
	}
}

func TestTaskStorage_GetByID_NotFound(t *testing.T) {
	tasks, _, cleanup := setupWorkDB(t)
	defer cleanup()
	ctx := context.Background()

	got, err := tasks.GetByID(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for nonexistent task")
	}
}

func TestTaskStorage_Update(t *testing.T) {
	tasks, _, cleanup := setupWorkDB(t)
	defer cleanup()
	ctx := context.Background()

	task, _ := domain.NewTask("s1", "sess1", "title", "", nil)
	tasks.Save(ctx, task)

	task.Approve()
	if err := tasks.Update(ctx, task); err != nil {
		t.Fatalf("update task: %v", err)
	}

	got, _ := tasks.GetByID(ctx, "s1")
	if got.Status != domain.TaskStatusApproved {
		t.Errorf("expected approved, got %s", got.Status)
	}
	if got.ApprovedAt == nil {
		t.Error("expected ApprovedAt to be set")
	}
}

func TestTaskStorage_GetBySessionID(t *testing.T) {
	tasks, _, cleanup := setupWorkDB(t)
	defer cleanup()
	ctx := context.Background()

	s1, _ := domain.NewTask("s1", "sess1", "Task 1", "", nil)
	s2, _ := domain.NewTask("s2", "sess1", "Task 2", "", nil)
	s3, _ := domain.NewTask("s3", "sess2", "Task 3", "", nil)
	tasks.Save(ctx, s1)
	tasks.Save(ctx, s2)
	tasks.Save(ctx, s3)

	got, err := tasks.GetBySessionID(ctx, "sess1")
	if err != nil {
		t.Fatalf("get by session: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(got))
	}
}

func TestTaskStorage_GetByStatus(t *testing.T) {
	tasks, _, cleanup := setupWorkDB(t)
	defer cleanup()
	ctx := context.Background()

	s1, _ := domain.NewTask("s1", "sess1", "Draft", "", nil)
	s2, _ := domain.NewTask("s2", "sess1", "Approved", "", nil)
	s2.Approve()
	tasks.Save(ctx, s1)
	tasks.Save(ctx, s2)

	drafts, _ := tasks.GetByStatus(ctx, "sess1", domain.TaskStatusDraft)
	if len(drafts) != 1 {
		t.Errorf("expected 1 draft, got %d", len(drafts))
	}

	approved, _ := tasks.GetByStatus(ctx, "sess1", domain.TaskStatusApproved)
	if len(approved) != 1 {
		t.Errorf("expected 1 approved, got %d", len(approved))
	}
}

// --- Subtask Storage Tests ---

func TestSubtaskStorage_SaveAndGetByID(t *testing.T) {
	tasks, subtasks, cleanup := setupWorkDB(t)
	defer cleanup()
	ctx := context.Background()

	// Must create task first (FK)
	task, _ := domain.NewTask("s1", "sess1", "Task", "", nil)
	tasks.Save(ctx, task)

	subtask, _ := domain.NewTaskSubtask("t1", "sess1", "s1", "Create proto", "Generate files",
		[]string{"t0"}, []string{"api/auth.proto"})

	if err := subtasks.Save(ctx, subtask); err != nil {
		t.Fatalf("save subtask: %v", err)
	}

	got, err := subtasks.GetByID(ctx, "t1")
	if err != nil {
		t.Fatalf("get subtask: %v", err)
	}
	if got == nil {
		t.Fatal("expected subtask, got nil")
	}
	if got.Title != "Create proto" {
		t.Errorf("expected title 'Create proto', got '%s'", got.Title)
	}
	if got.TaskID != "s1" {
		t.Errorf("expected task_id s1, got %s", got.TaskID)
	}
	if len(got.BlockedBy) != 1 || got.BlockedBy[0] != "t0" {
		t.Errorf("unexpected blocked_by: %v", got.BlockedBy)
	}
	if len(got.FilesInvolved) != 1 {
		t.Errorf("expected 1 file, got %d", len(got.FilesInvolved))
	}
}

func TestSubtaskStorage_FK_Constraint(t *testing.T) {
	_, subtasks, cleanup := setupWorkDB(t)
	defer cleanup()
	ctx := context.Background()

	// Try to insert subtask with nonexistent task_id — should fail
	subtask, _ := domain.NewTaskSubtask("t1", "sess1", "nonexistent_task", "title", "desc", nil, nil)
	err := subtasks.Save(ctx, subtask)
	if err == nil {
		t.Fatal("expected FK constraint error for nonexistent task_id")
	}
}

func TestSubtaskStorage_Update(t *testing.T) {
	tasks, subtasks, cleanup := setupWorkDB(t)
	defer cleanup()
	ctx := context.Background()

	task, _ := domain.NewTask("s1", "sess1", "Task", "", nil)
	tasks.Save(ctx, task)

	subtask, _ := domain.NewTaskSubtask("t1", "sess1", "s1", "title", "desc", nil, nil)
	subtasks.Save(ctx, subtask)

	subtask.Start()
	subtask.AssignToAgent("code-agent-abc")
	if err := subtasks.Update(ctx, subtask); err != nil {
		t.Fatalf("update subtask: %v", err)
	}

	got, _ := subtasks.GetByID(ctx, "t1")
	if got.Status != domain.SubtaskStatusInProgress {
		t.Errorf("expected in_progress, got %s", got.Status)
	}
	if got.AssignedAgentID != "code-agent-abc" {
		t.Errorf("expected agent id 'code-agent-abc', got '%s'", got.AssignedAgentID)
	}
}

func TestSubtaskStorage_GetByTaskID(t *testing.T) {
	tasks, subtasks, cleanup := setupWorkDB(t)
	defer cleanup()
	ctx := context.Background()

	task, _ := domain.NewTask("s1", "sess1", "Task", "", nil)
	tasks.Save(ctx, task)

	t1, _ := domain.NewTaskSubtask("t1", "sess1", "s1", "Subtask 1", "desc", nil, nil)
	t2, _ := domain.NewTaskSubtask("t2", "sess1", "s1", "Subtask 2", "desc", nil, nil)
	subtasks.Save(ctx, t1)
	subtasks.Save(ctx, t2)

	got, err := subtasks.GetByTaskID(ctx, "s1")
	if err != nil {
		t.Fatalf("get by task: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 subtasks, got %d", len(got))
	}
}

func TestSubtaskStorage_GetReadySubtasks(t *testing.T) {
	tasks, subtasks, cleanup := setupWorkDB(t)
	defer cleanup()
	ctx := context.Background()

	task, _ := domain.NewTask("s1", "sess1", "Task", "", nil)
	tasks.Save(ctx, task)

	// t1: no blockers — should be ready
	t1, _ := domain.NewTaskSubtask("t1", "sess1", "s1", "Subtask 1", "desc", nil, nil)
	subtasks.Save(ctx, t1)

	// t2: blocked by t1 (not completed) — should NOT be ready
	t2, _ := domain.NewTaskSubtask("t2", "sess1", "s1", "Subtask 2", "desc", []string{"t1"}, nil)
	subtasks.Save(ctx, t2)

	ready, err := subtasks.GetReadySubtasks(ctx, "s1")
	if err != nil {
		t.Fatalf("get ready subtasks: %v", err)
	}
	if len(ready) != 1 {
		t.Fatalf("expected 1 ready subtask, got %d", len(ready))
	}
	if ready[0].ID != "t1" {
		t.Errorf("expected t1 to be ready, got %s", ready[0].ID)
	}

	// Complete t1 — now t2 should be ready
	t1.Start()
	t1.Complete("done")
	subtasks.Update(ctx, t1)

	ready2, err := subtasks.GetReadySubtasks(ctx, "s1")
	if err != nil {
		t.Fatalf("get ready subtasks after completion: %v", err)
	}
	if len(ready2) != 1 {
		t.Fatalf("expected 1 ready subtask after completion, got %d", len(ready2))
	}
	if ready2[0].ID != "t2" {
		t.Errorf("expected t2 to be ready, got %s", ready2[0].ID)
	}
}

func TestSubtaskStorage_GetByAgentID(t *testing.T) {
	tasks, subtasks, cleanup := setupWorkDB(t)
	defer cleanup()
	ctx := context.Background()

	task, _ := domain.NewTask("s1", "sess1", "Task", "", nil)
	tasks.Save(ctx, task)

	subtask, _ := domain.NewTaskSubtask("t1", "sess1", "s1", "title", "desc", nil, nil)
	subtask.Start()
	subtask.AssignToAgent("code-agent-xyz")
	subtasks.Save(ctx, subtask)

	got, err := subtasks.GetByAgentID(ctx, "code-agent-xyz")
	if err != nil {
		t.Fatalf("get by agent: %v", err)
	}
	if got == nil {
		t.Fatal("expected subtask, got nil")
	}
	if got.ID != "t1" {
		t.Errorf("expected t1, got %s", got.ID)
	}
}

func TestSubtaskStorage_GetByAgentID_NotFound(t *testing.T) {
	_, subtasks, cleanup := setupWorkDB(t)
	defer cleanup()
	ctx := context.Background()

	got, err := subtasks.GetByAgentID(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for nonexistent agent")
	}
}

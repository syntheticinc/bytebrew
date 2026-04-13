package domain

import (
	"testing"
)

func TestNewTask(t *testing.T) {
	task, err := NewTask("s1", "sess1", "Add auth", "Implement JWT auth", []string{"Tests pass", "Endpoint works"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.ID != "s1" {
		t.Errorf("expected ID s1, got %s", task.ID)
	}
	if task.Status != TaskStatusDraft {
		t.Errorf("expected draft status, got %s", task.Status)
	}
	if len(task.AcceptanceCriteria) != 2 {
		t.Errorf("expected 2 criteria, got %d", len(task.AcceptanceCriteria))
	}
}

func TestNewTask_ValidationError(t *testing.T) {
	_, err := NewTask("", "sess1", "title", "", nil)
	if err == nil {
		t.Fatal("expected error for empty ID")
	}

	_, err = NewTask("s1", "", "title", "", nil)
	if err == nil {
		t.Fatal("expected error for empty session_id")
	}

	_, err = NewTask("s1", "sess1", "", "", nil)
	if err == nil {
		t.Fatal("expected error for empty title")
	}
}

func TestTask_Approve(t *testing.T) {
	task, _ := NewTask("s1", "sess1", "title", "", nil)

	if err := task.Approve(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Status != TaskStatusApproved {
		t.Errorf("expected approved, got %s", task.Status)
	}
	if task.ApprovedAt == nil {
		t.Error("expected ApprovedAt to be set")
	}

	// Cannot approve again
	if err := task.Approve(); err == nil {
		t.Error("expected error approving non-draft task")
	}
}

func TestTask_Start(t *testing.T) {
	task, _ := NewTask("s1", "sess1", "title", "", nil)
	task.Approve()

	if err := task.Start(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Status != TaskStatusInProgress {
		t.Errorf("expected in_progress, got %s", task.Status)
	}

	// Cannot start again
	if err := task.Start(); err == nil {
		t.Error("expected error starting non-approved task")
	}
}

func TestTask_Complete(t *testing.T) {
	task, _ := NewTask("s1", "sess1", "title", "", nil)
	task.Approve()
	task.Start()

	if err := task.Complete(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Status != TaskStatusCompleted {
		t.Errorf("expected completed, got %s", task.Status)
	}
	if task.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
}

func TestTask_Fail(t *testing.T) {
	task, _ := NewTask("s1", "sess1", "title", "", nil)
	task.Approve()
	task.Start()

	if err := task.Fail(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Status != TaskStatusFailed {
		t.Errorf("expected failed, got %s", task.Status)
	}
}

func TestTask_Cancel(t *testing.T) {
	task, _ := NewTask("s1", "sess1", "title", "", nil)

	if err := task.Cancel(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Status != TaskStatusCancelled {
		t.Errorf("expected cancelled, got %s", task.Status)
	}

	// Cannot cancel completed task
	task2, _ := NewTask("s2", "sess1", "title2", "", nil)
	task2.Approve()
	task2.Start()
	task2.Complete()
	if err := task2.Cancel(); err == nil {
		t.Error("expected error cancelling completed task")
	}
}

func TestTask_IsTerminal(t *testing.T) {
	task, _ := NewTask("s1", "sess1", "title", "", nil)
	if task.IsTerminal() {
		t.Error("draft should not be terminal")
	}

	task.Approve()
	task.Start()
	task.Complete()
	if !task.IsTerminal() {
		t.Error("completed should be terminal")
	}
}

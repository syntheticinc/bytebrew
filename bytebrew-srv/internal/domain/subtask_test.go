package domain

import (
	"testing"
)

func TestNewSubtask(t *testing.T) {
	tests := []struct {
		name        string
		sessionID   string
		description string
		wantErr     bool
	}{
		{
			name:        "valid subtask",
			sessionID:   "session-1",
			description: "Test subtask",
			wantErr:     false,
		},
		{
			name:        "missing session_id",
			sessionID:   "",
			description: "Test subtask",
			wantErr:     true,
		},
		{
			name:        "missing description",
			sessionID:   "session-1",
			description: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subtask, err := NewSubtask(tt.sessionID, tt.description)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewSubtask() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if subtask == nil {
					t.Error("NewSubtask() returned nil subtask without error")
				}
				if subtask.Status != SubtaskStatusPending {
					t.Errorf("NewSubtask() status = %v, want %v", subtask.Status, SubtaskStatusPending)
				}
			}
		})
	}
}

func TestSubtask_Start(t *testing.T) {
	subtask, _ := NewSubtask("session-1", "Test subtask")

	if err := subtask.Start(); err != nil {
		t.Errorf("Start() error = %v", err)
	}

	if subtask.Status != SubtaskStatusInProgress {
		t.Errorf("Start() status = %v, want %v", subtask.Status, SubtaskStatusInProgress)
	}

	// Try to start again - should fail
	if err := subtask.Start(); err == nil {
		t.Error("Start() should fail when subtask is already in progress")
	}
}

func TestSubtask_Complete(t *testing.T) {
	subtask, _ := NewSubtask("session-1", "Test subtask")
	subtask.Start()

	result := "Subtask completed successfully"
	if err := subtask.Complete(result); err != nil {
		t.Errorf("Complete() error = %v", err)
	}

	if subtask.Status != SubtaskStatusCompleted {
		t.Errorf("Complete() status = %v, want %v", subtask.Status, SubtaskStatusCompleted)
	}

	if subtask.Result != result {
		t.Errorf("Complete() result = %v, want %v", subtask.Result, result)
	}

	if subtask.CompletedAt == nil {
		t.Error("Complete() should set CompletedAt")
	}
}

func TestSubtask_Cancel(t *testing.T) {
	subtask, _ := NewSubtask("session-1", "Test subtask")

	if err := subtask.Cancel(); err != nil {
		t.Errorf("Cancel() error = %v", err)
	}

	if subtask.Status != SubtaskStatusCancelled {
		t.Errorf("Cancel() status = %v, want %v", subtask.Status, SubtaskStatusCancelled)
	}

	// Cannot cancel completed subtask
	subtask2, _ := NewSubtask("session-2", "Test subtask 2")
	subtask2.Start()
	subtask2.Complete("Done")

	if err := subtask2.Cancel(); err == nil {
		t.Error("Cancel() should fail for completed subtask")
	}
}

func TestSubtask_WaitForInput(t *testing.T) {
	subtask, _ := NewSubtask("session-1", "Test subtask")
	subtask.Start()

	if err := subtask.WaitForInput(); err != nil {
		t.Errorf("WaitForInput() error = %v", err)
	}

	if subtask.Status != SubtaskStatusWaitingForInput {
		t.Errorf("WaitForInput() status = %v, want %v", subtask.Status, SubtaskStatusWaitingForInput)
	}
}

func TestSubtask_Resume(t *testing.T) {
	subtask, _ := NewSubtask("session-1", "Test subtask")
	subtask.Start()
	subtask.WaitForInput()

	if err := subtask.Resume(); err != nil {
		t.Errorf("Resume() error = %v", err)
	}

	if subtask.Status != SubtaskStatusInProgress {
		t.Errorf("Resume() status = %v, want %v", subtask.Status, SubtaskStatusInProgress)
	}
}

func TestSubtask_AddContext(t *testing.T) {
	subtask, _ := NewSubtask("session-1", "Test subtask")

	subtask.AddContext("key1", "value1")
	subtask.AddContext("key2", "value2")

	if len(subtask.Context) != 2 {
		t.Errorf("AddContext() context length = %v, want 2", len(subtask.Context))
	}

	if subtask.Context["key1"] != "value1" {
		t.Errorf("AddContext() context[key1] = %v, want value1", subtask.Context["key1"])
	}
}

func TestSubtask_IsCompleted(t *testing.T) {
	subtask, _ := NewSubtask("session-1", "Test subtask")

	if subtask.IsCompleted() {
		t.Error("IsCompleted() should return false for pending subtask")
	}

	subtask.Start()
	subtask.Complete("Done")

	if !subtask.IsCompleted() {
		t.Error("IsCompleted() should return true for completed subtask")
	}
}

func TestSubtask_IsCancelled(t *testing.T) {
	subtask, _ := NewSubtask("session-1", "Test subtask")

	if subtask.IsCancelled() {
		t.Error("IsCancelled() should return false for pending subtask")
	}

	subtask.Cancel()

	if !subtask.IsCancelled() {
		t.Error("IsCancelled() should return true for cancelled subtask")
	}
}

func TestNewTaskSubtask(t *testing.T) {
	subtask, err := NewTaskSubtask("st1", "sess1", "t1", "Create proto", "Generate proto files",
		[]string{"st0"}, []string{"api/proto/auth.proto"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subtask.ID != "st1" {
		t.Errorf("expected ID st1, got %s", subtask.ID)
	}
	if subtask.TaskID != "t1" {
		t.Errorf("expected TaskID t1, got %s", subtask.TaskID)
	}
	if subtask.Status != SubtaskStatusPending {
		t.Errorf("expected pending, got %s", subtask.Status)
	}
	if len(subtask.BlockedBy) != 1 || subtask.BlockedBy[0] != "st0" {
		t.Errorf("unexpected blocked_by: %v", subtask.BlockedBy)
	}
	if len(subtask.FilesInvolved) != 1 {
		t.Errorf("expected 1 file, got %d", len(subtask.FilesInvolved))
	}
}

func TestNewTaskSubtask_ValidationError(t *testing.T) {
	_, err := NewTaskSubtask("st1", "", "t1", "title", "desc", nil, nil)
	if err == nil {
		t.Fatal("expected error for empty session_id")
	}

	_, err = NewTaskSubtask("st1", "sess1", "t1", "", "", nil, nil)
	if err == nil {
		t.Fatal("expected error for empty title and description")
	}
}

func TestSubtask_Fail(t *testing.T) {
	subtask, _ := NewTaskSubtask("st1", "sess1", "t1", "title", "desc", nil, nil)
	subtask.Start()

	if err := subtask.Fail("something broke"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subtask.Status != SubtaskStatusFailed {
		t.Errorf("expected failed, got %s", subtask.Status)
	}
	if subtask.Result != "something broke" {
		t.Errorf("expected failure reason in result")
	}
}

func TestSubtask_Fail_NotInProgress(t *testing.T) {
	subtask, _ := NewTaskSubtask("st1", "sess1", "t1", "title", "desc", nil, nil)
	if err := subtask.Fail("reason"); err == nil {
		t.Error("expected error failing pending subtask")
	}
}

func TestSubtask_AssignToAgent(t *testing.T) {
	subtask, _ := NewTaskSubtask("st1", "sess1", "t1", "title", "desc", nil, nil)
	subtask.AssignToAgent("code-agent-abc123")
	if subtask.AssignedAgentID != "code-agent-abc123" {
		t.Errorf("expected agent ID code-agent-abc123, got %s", subtask.AssignedAgentID)
	}
}

func TestSubtask_IsBlocked(t *testing.T) {
	subtask1, _ := NewTaskSubtask("st1", "sess1", "t1", "title", "desc", nil, nil)
	if subtask1.IsBlocked() {
		t.Error("subtask without blockers should not be blocked")
	}

	subtask2, _ := NewTaskSubtask("st2", "sess1", "t1", "title", "desc", []string{"st1"}, nil)
	if !subtask2.IsBlocked() {
		t.Error("subtask with blockers should be blocked")
	}
}

func TestSubtask_IsTerminal(t *testing.T) {
	subtask, _ := NewTaskSubtask("st1", "sess1", "t1", "title", "desc", nil, nil)
	if subtask.IsTerminal() {
		t.Error("pending should not be terminal")
	}

	subtask.Start()
	subtask.Complete("done")
	if !subtask.IsTerminal() {
		t.Error("completed should be terminal")
	}

	subtask2, _ := NewTaskSubtask("st2", "sess1", "t1", "title", "desc", nil, nil)
	subtask2.Start()
	subtask2.Fail("err")
	if !subtask2.IsTerminal() {
		t.Error("failed should be terminal")
	}
}

func TestSubtask_LegacyConstructor(t *testing.T) {
	subtask, err := NewSubtask("sess1", "do something")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subtask.TaskID != "" {
		t.Errorf("legacy subtask should have empty TaskID, got %s", subtask.TaskID)
	}
	if subtask.Title != "" {
		t.Errorf("legacy subtask should have empty Title, got %s", subtask.Title)
	}
}

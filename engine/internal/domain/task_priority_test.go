package domain

import (
	"testing"
)

func TestTask_SetPriority(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		wantErr  bool
	}{
		{"normal priority", 0, false},
		{"high priority", 1, false},
		{"critical priority", 2, false},
		{"invalid negative", -1, true},
		{"invalid too high", 3, true},
		{"invalid large number", 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := NewTask("task-1", "session-1", "Test Task", "Test", nil)
			if err != nil {
				t.Fatalf("NewTask failed: %v", err)
			}

			err = task.SetPriority(tt.priority)
			if tt.wantErr {
				if err == nil {
					t.Errorf("SetPriority(%d) expected error, got nil", tt.priority)
				}
			} else {
				if err != nil {
					t.Errorf("SetPriority(%d) unexpected error: %v", tt.priority, err)
				}
				if task.Priority != tt.priority {
					t.Errorf("Priority = %d, want %d", task.Priority, tt.priority)
				}
			}
		})
	}
}

func TestTask_DefaultPriority(t *testing.T) {
	task, err := NewTask("task-1", "session-1", "Test Task", "Test", nil)
	if err != nil {
		t.Fatalf("NewTask failed: %v", err)
	}

	if task.Priority != 0 {
		t.Errorf("Default priority = %d, want 0", task.Priority)
	}
}

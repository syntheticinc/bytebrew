package domain

import (
	"testing"
	"time"
)

func TestNewTaskPacket_Valid(t *testing.T) {
	tp, err := NewTaskPacket("task-1", "parent", "child", "do stuff", 30*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tp.Status != TaskPacketPending {
		t.Errorf("expected pending, got %s", tp.Status)
	}
}

func TestNewTaskPacket_MissingFields(t *testing.T) {
	tests := []struct {
		name   string
		id     string
		parent string
		child  string
		input  string
	}{
		{"missing id", "", "p", "c", "input"},
		{"missing parent", "id", "", "c", "input"},
		{"missing child", "id", "p", "", "input"},
		{"missing input", "id", "p", "c", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTaskPacket(tt.id, tt.parent, tt.child, tt.input, 0)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestTaskPacket_Lifecycle(t *testing.T) {
	tp, _ := NewTaskPacket("t1", "p", "c", "input", 0)

	if err := tp.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	if tp.Status != TaskPacketRunning {
		t.Errorf("expected running, got %s", tp.Status)
	}

	if err := tp.Complete("result"); err != nil {
		t.Fatalf("complete: %v", err)
	}
	if tp.Status != TaskPacketCompleted {
		t.Errorf("expected completed, got %s", tp.Status)
	}
	if tp.Result != "result" {
		t.Errorf("expected result %q, got %q", "result", tp.Result)
	}
}

func TestTaskPacket_Fail(t *testing.T) {
	tp, _ := NewTaskPacket("t1", "p", "c", "input", 0)
	tp.Start()

	if err := tp.Fail("broken"); err != nil {
		t.Fatalf("fail: %v", err)
	}
	if tp.Status != TaskPacketFailed {
		t.Errorf("expected failed, got %s", tp.Status)
	}
	if tp.Error != "broken" {
		t.Errorf("expected error %q, got %q", "broken", tp.Error)
	}
}

func TestTaskPacket_Timeout(t *testing.T) {
	tp, _ := NewTaskPacket("t1", "p", "c", "input", 0)
	tp.Start()

	if err := tp.MarkTimeout(); err != nil {
		t.Fatalf("timeout: %v", err)
	}
	if tp.Status != TaskPacketTimeout {
		t.Errorf("expected timeout, got %s", tp.Status)
	}
}

func TestTaskPacket_IsTerminal(t *testing.T) {
	tests := []struct {
		status   TaskPacketStatus
		terminal bool
	}{
		{TaskPacketPending, false},
		{TaskPacketRunning, false},
		{TaskPacketCompleted, true},
		{TaskPacketFailed, true},
		{TaskPacketTimeout, true},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			tp := &TaskPacket{Status: tt.status}
			if tp.IsTerminal() != tt.terminal {
				t.Errorf("IsTerminal() = %v, want %v", tp.IsTerminal(), tt.terminal)
			}
		})
	}
}

func TestTaskPacket_IsExpired(t *testing.T) {
	tp, _ := NewTaskPacket("t1", "p", "c", "input", 1*time.Millisecond)
	tp.Start()
	time.Sleep(5 * time.Millisecond)

	if !tp.IsExpired() {
		t.Error("expected expired after timeout")
	}
}

func TestTaskPacket_NotExpired_NoTimeout(t *testing.T) {
	tp, _ := NewTaskPacket("t1", "p", "c", "input", 0)
	tp.Start()
	if tp.IsExpired() {
		t.Error("expected not expired with 0 timeout")
	}
}

func TestTaskPacket_InvalidTransitions(t *testing.T) {
	tp, _ := NewTaskPacket("t1", "p", "c", "input", 0)

	// Can't complete from pending
	if err := tp.Complete("x"); err == nil {
		t.Error("expected error completing pending task")
	}
	// Can't timeout from pending
	if err := tp.MarkTimeout(); err == nil {
		t.Error("expected error timing out pending task")
	}

	tp.Start()
	// Can't start again
	if err := tp.Start(); err == nil {
		t.Error("expected error starting running task")
	}
}

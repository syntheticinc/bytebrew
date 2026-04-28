package orchestrator

import (
	"sync"
	"testing"
	"time"
)

func TestEventBus_PublishConsume(t *testing.T) {
	bus := NewSessionEventBus(8)
	defer bus.Close()

	event := OrchestratorEvent{
		Type:    EventUserMessage,
		Content: "hello",
	}

	if err := bus.Publish(event); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case received := <-bus.Events():
		if received.Type != EventUserMessage {
			t.Errorf("type = %v, want %v", received.Type, EventUserMessage)
		}
		if received.Content != "hello" {
			t.Errorf("content = %v, want %v", received.Content, "hello")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestEventBus_Buffered(t *testing.T) {
	bus := NewSessionEventBus(4)
	defer bus.Close()

	// Publish multiple events before consuming
	for i := 0; i < 4; i++ {
		err := bus.Publish(OrchestratorEvent{
			Type:    EventUserMessage,
			Content: "msg",
		})
		if err != nil {
			t.Fatalf("publish %d: %v", i, err)
		}
	}

	// Consume all
	for i := 0; i < 4; i++ {
		select {
		case <-bus.Events():
		case <-time.After(time.Second):
			t.Fatalf("timeout on event %d", i)
		}
	}
}

func TestEventBus_BufferFull(t *testing.T) {
	bus := NewSessionEventBus(2)
	defer bus.Close()

	// Fill buffer
	bus.Publish(OrchestratorEvent{Type: EventUserMessage})
	bus.Publish(OrchestratorEvent{Type: EventUserMessage})

	// Third should fail
	err := bus.Publish(OrchestratorEvent{Type: EventUserMessage})
	if err == nil {
		t.Fatal("expected error when buffer full")
	}
}

func TestEventBus_Close(t *testing.T) {
	bus := NewSessionEventBus(8)
	bus.Close()

	err := bus.Publish(OrchestratorEvent{Type: EventUserMessage})
	if err == nil {
		t.Fatal("expected error after close")
	}
}

func TestEventBus_CloseMultiple(t *testing.T) {
	bus := NewSessionEventBus(8)
	// Should not panic
	bus.Close()
	bus.Close()
	bus.Close()
}

func TestEventBus_ConcurrentPublish(t *testing.T) {
	bus := NewSessionEventBus(64)
	defer bus.Close()

	var wg sync.WaitGroup
	errCount := 0
	var mu sync.Mutex

	// 10 goroutines publishing 5 events each
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				if err := bus.Publish(OrchestratorEvent{Type: EventAgentCompleted}); err != nil {
					mu.Lock()
					errCount++
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	// All 50 events should succeed (buffer=64)
	if errCount != 0 {
		t.Errorf("unexpected errors: %d", errCount)
	}

	// Drain and count
	count := 0
	for {
		select {
		case <-bus.Events():
			count++
		default:
			goto done
		}
	}
done:
	if count != 50 {
		t.Errorf("consumed %d events, want 50", count)
	}
}

func TestEventBus_CloseUnblocks(t *testing.T) {
	bus := NewSessionEventBus(8)

	done := make(chan struct{})
	go func() {
		for range bus.Events() {
			// drain
		}
		close(done)
	}()

	bus.Close()

	select {
	case <-done:
		// Consumer unblocked by close
	case <-time.After(time.Second):
		t.Fatal("consumer not unblocked after close")
	}
}

func TestEventBus_AgentEvents(t *testing.T) {
	bus := NewSessionEventBus(8)
	defer bus.Close()

	bus.Publish(OrchestratorEvent{
		Type:      EventAgentCompleted,
		AgentID:   "code-agent-abc",
		SubtaskID: "st-1",
		Content:   "files modified: main.go",
	})

	bus.Publish(OrchestratorEvent{
		Type:      EventAgentFailed,
		AgentID:   "code-agent-def",
		SubtaskID: "st-2",
		Content:   "compilation error",
	})

	e1 := <-bus.Events()
	if e1.Type != EventAgentCompleted || e1.AgentID != "code-agent-abc" {
		t.Errorf("event 1: type=%v agent=%v", e1.Type, e1.AgentID)
	}

	e2 := <-bus.Events()
	if e2.Type != EventAgentFailed || e2.SubtaskID != "st-2" {
		t.Errorf("event 2: type=%v subtask=%v", e2.Type, e2.SubtaskID)
	}
}

func TestEventBus_PublishInterrupt_SignalsChannel(t *testing.T) {
	bus := NewSessionEventBus(8)
	defer bus.Close()

	err := bus.PublishInterrupt(OrchestratorEvent{
		Type:    EventUserMessage,
		Content: "interrupt me",
	})
	if err != nil {
		t.Fatalf("publish interrupt: %v", err)
	}

	// Event should be in the event channel
	select {
	case e := <-bus.Events():
		if e.Content != "interrupt me" {
			t.Errorf("content = %q, want %q", e.Content, "interrupt me")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}

	// Interrupt signal should be in the interrupt channel
	select {
	case <-bus.Interrupts():
		// ok
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for interrupt signal")
	}
}

func TestEventBus_PublishInterrupt_NonBlockingWhenFull(t *testing.T) {
	bus := NewSessionEventBus(8)
	defer bus.Close()

	// First interrupt fills the buffer (capacity 1)
	err := bus.PublishInterrupt(OrchestratorEvent{Type: EventUserMessage, Content: "first"})
	if err != nil {
		t.Fatalf("first interrupt: %v", err)
	}

	// Second interrupt should not block (interrupt channel already has a signal)
	err = bus.PublishInterrupt(OrchestratorEvent{Type: EventUserMessage, Content: "second"})
	if err != nil {
		t.Fatalf("second interrupt: %v", err)
	}

	// Only one interrupt signal in channel
	select {
	case <-bus.Interrupts():
	default:
		t.Fatal("expected interrupt signal")
	}

	select {
	case <-bus.Interrupts():
		t.Fatal("expected no second interrupt signal")
	default:
		// ok — only one signal
	}
}

func TestEventBus_PublishInterrupt_ErrorOnClosedBus(t *testing.T) {
	bus := NewSessionEventBus(8)
	bus.Close()

	err := bus.PublishInterrupt(OrchestratorEvent{Type: EventUserMessage, Content: "late"})
	if err == nil {
		t.Fatal("expected error after close")
	}

	// No interrupt signal should be sent on error
	select {
	case <-bus.Interrupts():
		t.Fatal("should not signal interrupt on publish error")
	default:
		// ok
	}
}

func TestEventBus_DrainInterrupts_ClearsSignals(t *testing.T) {
	bus := NewSessionEventBus(8)
	defer bus.Close()

	// Send an interrupt signal
	bus.PublishInterrupt(OrchestratorEvent{Type: EventUserMessage, Content: "msg"})

	// Drain should clear it
	bus.DrainInterrupts()

	// Channel should be empty now
	select {
	case <-bus.Interrupts():
		t.Fatal("interrupt channel should be empty after drain")
	default:
		// ok
	}
}

func TestEventBus_DrainInterrupts_NoopWhenEmpty(t *testing.T) {
	bus := NewSessionEventBus(8)
	defer bus.Close()

	// Should not block or panic on empty channel
	bus.DrainInterrupts()
}

func TestEventBus_Interrupts_ReturnsReadOnlyChannel(t *testing.T) {
	bus := NewSessionEventBus(8)
	defer bus.Close()

	ch := bus.Interrupts()
	if ch == nil {
		t.Fatal("Interrupts() returned nil channel")
	}

	// Should be empty initially
	select {
	case <-ch:
		t.Fatal("channel should be empty initially")
	default:
		// ok
	}
}

func TestEventBus_DefaultBufferSize(t *testing.T) {
	// Zero or negative should still work
	bus := NewSessionEventBus(0)
	defer bus.Close()

	if err := bus.Publish(OrchestratorEvent{Type: EventWorkReminder}); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case e := <-bus.Events():
		if e.Type != EventWorkReminder {
			t.Errorf("type = %v, want %v", e.Type, EventWorkReminder)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

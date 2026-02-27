package flow_registry

import (
	"testing"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
)

func TestRegister_Success(t *testing.T) {
	registry := NewInMemoryRegistry()

	flow, err := domain.NewActiveFlow("session-1", "project-1", "user-1", "test task")
	if err != nil {
		t.Fatalf("failed to create flow: %v", err)
	}

	err = registry.Register(flow.SessionID, flow)
	if err != nil {
		t.Fatalf("failed to register flow: %v", err)
	}

	if !registry.IsActive(flow.SessionID) {
		t.Error("flow should be active")
	}
}

func TestRegister_AlreadyExists(t *testing.T) {
	registry := NewInMemoryRegistry()

	flow, err := domain.NewActiveFlow("session-1", "project-1", "user-1", "test task")
	if err != nil {
		t.Fatalf("failed to create flow: %v", err)
	}

	err = registry.Register(flow.SessionID, flow)
	if err != nil {
		t.Fatalf("failed to register flow: %v", err)
	}

	err = registry.Register(flow.SessionID, flow)
	if err == nil {
		t.Error("expected error when registering existing flow")
	}
}

func TestUnregister_Success(t *testing.T) {
	registry := NewInMemoryRegistry()

	flow, err := domain.NewActiveFlow("session-1", "project-1", "user-1", "test task")
	if err != nil {
		t.Fatalf("failed to create flow: %v", err)
	}

	err = registry.Register(flow.SessionID, flow)
	if err != nil {
		t.Fatalf("failed to register flow: %v", err)
	}

	err = registry.Unregister(flow.SessionID)
	if err != nil {
		t.Fatalf("failed to unregister flow: %v", err)
	}

	if registry.IsActive(flow.SessionID) {
		t.Error("flow should not be active after unregister")
	}
}

func TestGet_Found(t *testing.T) {
	registry := NewInMemoryRegistry()

	flow, err := domain.NewActiveFlow("session-1", "project-1", "user-1", "test task")
	if err != nil {
		t.Fatalf("failed to create flow: %v", err)
	}

	err = registry.Register(flow.SessionID, flow)
	if err != nil {
		t.Fatalf("failed to register flow: %v", err)
	}

	retrieved, found := registry.Get(flow.SessionID)
	if !found {
		t.Fatal("flow not found")
	}
	if retrieved.SessionID != flow.SessionID {
		t.Error("session_id mismatch")
	}
}

func TestGet_NotFound(t *testing.T) {
	registry := NewInMemoryRegistry()

	_, found := registry.Get("non-existent")
	if found {
		t.Error("expected not found")
	}
}

func TestIsActive_True(t *testing.T) {
	registry := NewInMemoryRegistry()

	flow, err := domain.NewActiveFlow("session-1", "project-1", "user-1", "test task")
	if err != nil {
		t.Fatalf("failed to create flow: %v", err)
	}

	err = registry.Register(flow.SessionID, flow)
	if err != nil {
		t.Fatalf("failed to register flow: %v", err)
	}

	if !registry.IsActive(flow.SessionID) {
		t.Error("flow should be active")
	}
}

func TestIsActive_False(t *testing.T) {
	registry := NewInMemoryRegistry()

	if registry.IsActive("non-existent") {
		t.Error("flow should not be active")
	}
}

func TestSubscribe_Success(t *testing.T) {
	registry := NewInMemoryRegistry()

	flow, err := domain.NewActiveFlow("session-1", "project-1", "user-1", "test task")
	if err != nil {
		t.Fatalf("failed to create flow: %v", err)
	}

	err = registry.Register(flow.SessionID, flow)
	if err != nil {
		t.Fatalf("failed to register flow: %v", err)
	}

	subscriber := &mockSubscriber{id: "sub-1"}
	err = registry.Subscribe(flow.SessionID, subscriber)
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}
}

func TestSubscribe_FlowNotFound(t *testing.T) {
	registry := NewInMemoryRegistry()

	subscriber := &mockSubscriber{id: "sub-1"}
	err := registry.Subscribe("non-existent", subscriber)
	if err == nil {
		t.Error("expected error when subscribing to non-existent flow")
	}
}

func TestUnsubscribe_Success(t *testing.T) {
	registry := NewInMemoryRegistry()

	flow, err := domain.NewActiveFlow("session-1", "project-1", "user-1", "test task")
	if err != nil {
		t.Fatalf("failed to create flow: %v", err)
	}

	err = registry.Register(flow.SessionID, flow)
	if err != nil {
		t.Fatalf("failed to register flow: %v", err)
	}

	subscriber := &mockSubscriber{id: "sub-1"}
	err = registry.Subscribe(flow.SessionID, subscriber)
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}

	err = registry.Unsubscribe(flow.SessionID, subscriber.ID())
	if err != nil {
		t.Fatalf("failed to unsubscribe: %v", err)
	}
}

func TestConcurrentAccess(t *testing.T) {
	registry := NewInMemoryRegistry()

	done := make(chan bool)

	// Concurrent registrations
	for i := 0; i < 10; i++ {
		go func(id int) {
			flow, err := domain.NewActiveFlow(
				"session-"+string(rune('0'+id)),
				"project-1",
				"user-1",
				"test task",
			)
			if err != nil {
				t.Errorf("failed to create flow: %v", err)
				done <- false
				return
			}

			registry.Register(flow.SessionID, flow)
			registry.IsActive(flow.SessionID)
			registry.Get(flow.SessionID)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		if !<-done {
			t.Error("concurrent operation failed")
		}
	}
}

// mockSubscriber implements FlowSubscriber for testing
type mockSubscriber struct {
	id string
}

func (m *mockSubscriber) ID() string {
	return m.id
}

func (m *mockSubscriber) OnEvent(event *domain.AgentEvent) error {
	return nil
}

func (m *mockSubscriber) OnComplete() error {
	return nil
}

func (m *mockSubscriber) OnError(err error) error {
	return nil
}

// TestBroadcastEvent tests broadcasting events to subscribers
func TestBroadcastEvent(t *testing.T) {
	registry := NewInMemoryRegistry()

	flow, err := domain.NewActiveFlow("session-1", "project-1", "user-1", "test task")
	if err != nil {
		t.Fatalf("failed to create flow: %v", err)
	}

	err = registry.Register(flow.SessionID, flow)
	if err != nil {
		t.Fatalf("failed to register flow: %v", err)
	}

	subscriber1 := &mockSubscriber{id: "sub-1"}
	subscriber2 := &mockSubscriber{id: "sub-2"}

	err = registry.Subscribe(flow.SessionID, subscriber1)
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}

	err = registry.Subscribe(flow.SessionID, subscriber2)
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}

	// Broadcast event
	event := &domain.AgentEvent{
		Type: "test_event",
	}

	err = registry.BroadcastEvent(flow.SessionID, event)
	if err != nil {
		t.Fatalf("failed to broadcast event: %v", err)
	}
}

// TestBroadcastEvent_FlowNotFound tests broadcasting to non-existent flow
func TestBroadcastEvent_FlowNotFound(t *testing.T) {
	registry := NewInMemoryRegistry()

	event := &domain.AgentEvent{
		Type: "test_event",
	}

	err := registry.BroadcastEvent("non-existent", event)
	if err == nil {
		t.Error("expected error when broadcasting to non-existent flow")
	}
}

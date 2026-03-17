package grpc

import (
	"testing"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/service/orchestrator"
	"github.com/stretchr/testify/assert"
)

// mockMessageRouter implements MessageRouter for testing
type mockMessageRouter struct {
	blockingWait    bool
	notifiedSession string
	notifiedMessage string
}

func (m *mockMessageRouter) HasBlockingWait(sessionID string) bool {
	return m.blockingWait
}

func (m *mockMessageRouter) NotifyUserMessage(sessionID, message string) {
	m.notifiedSession = sessionID
	m.notifiedMessage = message
}

func TestRouteUserMessage_BlockingWait_RoutesToInterrupt(t *testing.T) {
	router := &mockMessageRouter{blockingWait: true}
	eventBus := orchestrator.NewSessionEventBus(64)
	defer eventBus.Close()

	viaInterrupt := routeUserMessage("session-1", "fix bug #42", router, eventBus)

	assert.True(t, viaInterrupt, "should route via interrupt when blocking wait active")
	assert.Equal(t, "session-1", router.notifiedSession)
	assert.Equal(t, "fix bug #42", router.notifiedMessage)

	// EventBus should NOT have received the message
	select {
	case evt := <-eventBus.Events():
		t.Errorf("expected no EventBus events, got: %+v", evt)
	default:
		// Good — no events
	}
}

func TestRouteUserMessage_NoBlockingWait_RoutesToEventBus(t *testing.T) {
	router := &mockMessageRouter{blockingWait: false}
	eventBus := orchestrator.NewSessionEventBus(64)
	defer eventBus.Close()

	viaInterrupt := routeUserMessage("session-1", "new task", router, eventBus)

	assert.False(t, viaInterrupt, "should route via EventBus when no blocking wait")
	assert.Empty(t, router.notifiedSession, "should NOT notify via interrupt")

	// EventBus SHOULD have received the message
	select {
	case evt := <-eventBus.Events():
		assert.Equal(t, orchestrator.EventUserMessage, evt.Type)
		assert.Equal(t, "new task", evt.Content)
	default:
		t.Error("expected EventBus event, got none")
	}
}

func TestRouteUserMessage_NilRouter_RoutesToEventBus(t *testing.T) {
	eventBus := orchestrator.NewSessionEventBus(64)
	defer eventBus.Close()

	viaInterrupt := routeUserMessage("session-1", "hello", nil, eventBus)

	assert.False(t, viaInterrupt, "should route via EventBus when router is nil")

	// EventBus SHOULD have received the message
	select {
	case evt := <-eventBus.Events():
		assert.Equal(t, orchestrator.EventUserMessage, evt.Type)
		assert.Equal(t, "hello", evt.Content)
	default:
		t.Error("expected EventBus event, got none")
	}
}

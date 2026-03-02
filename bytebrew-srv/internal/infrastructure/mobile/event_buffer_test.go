package mobile

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
)

func newTestEvent(sessionID string) *pb.SessionEvent {
	return &pb.SessionEvent{
		SessionId: sessionID,
		Type:      pb.SessionEventType_SESSION_EVENT_TYPE_AGENT_MESSAGE,
		Payload: &pb.SessionEvent_AgentMessage{
			AgentMessage: &pb.AgentMessageEvent{
				Content: "test message",
			},
		},
	}
}

func TestEventBuffer_AppendAndGetAfter(t *testing.T) {
	buf := NewEventBuffer(10)

	e1 := newTestEvent("s1")
	e2 := newTestEvent("s1")
	e3 := newTestEvent("s1")

	id1 := buf.Append("s1", e1)
	id2 := buf.Append("s1", e2)
	id3 := buf.Append("s1", e3)

	assert.Equal(t, "s1-1", id1)
	assert.Equal(t, "s1-2", id2)
	assert.Equal(t, "s1-3", id3)

	// EventId should be set on the proto event
	assert.Equal(t, "s1-1", e1.EventId)
	assert.Equal(t, "s1-2", e2.EventId)
	assert.Equal(t, "s1-3", e3.EventId)

	// GetAfter with id1 should return e2 and e3
	events := buf.GetAfter("s1", id1)
	require.Len(t, events, 2)
	assert.Equal(t, "s1-2", events[0].EventId)
	assert.Equal(t, "s1-3", events[1].EventId)

	// GetAfter with id2 should return only e3
	events = buf.GetAfter("s1", id2)
	require.Len(t, events, 1)
	assert.Equal(t, "s1-3", events[0].EventId)

	// GetAfter with id3 should return nothing
	events = buf.GetAfter("s1", id3)
	assert.Empty(t, events)
}

func TestEventBuffer_GetAfterEmptyLastEventID(t *testing.T) {
	buf := NewEventBuffer(10)

	buf.Append("s1", newTestEvent("s1"))
	buf.Append("s1", newTestEvent("s1"))

	// Empty lastEventID returns all events for the session
	events := buf.GetAfter("s1", "")
	require.Len(t, events, 2)
	assert.Equal(t, "s1-1", events[0].EventId)
	assert.Equal(t, "s1-2", events[1].EventId)
}

func TestEventBuffer_GetAfterNonExistentLastEventID(t *testing.T) {
	buf := NewEventBuffer(10)

	buf.Append("s1", newTestEvent("s1"))
	buf.Append("s1", newTestEvent("s1"))

	// Non-existent lastEventID returns all events (full resync)
	events := buf.GetAfter("s1", "s1-999")
	require.Len(t, events, 2)
	assert.Equal(t, "s1-1", events[0].EventId)
	assert.Equal(t, "s1-2", events[1].EventId)
}

func TestEventBuffer_EmptyBuffer(t *testing.T) {
	buf := NewEventBuffer(10)

	events := buf.GetAfter("s1", "")
	assert.Nil(t, events)

	events = buf.GetAfter("s1", "s1-1")
	assert.Nil(t, events)
}

func TestEventBuffer_CircularWrapAround(t *testing.T) {
	buf := NewEventBuffer(3) // small buffer to test wrap-around

	// Append 5 events, buffer only holds 3
	for i := 0; i < 5; i++ {
		buf.Append("s1", newTestEvent("s1"))
	}

	// Only events 3, 4, 5 should remain (oldest 1, 2 overwritten)
	events := buf.GetAfter("s1", "")
	require.Len(t, events, 3)
	assert.Equal(t, "s1-3", events[0].EventId)
	assert.Equal(t, "s1-4", events[1].EventId)
	assert.Equal(t, "s1-5", events[2].EventId)

	// GetAfter with s1-3 should return events 4 and 5
	events = buf.GetAfter("s1", "s1-3")
	require.Len(t, events, 2)
	assert.Equal(t, "s1-4", events[0].EventId)
	assert.Equal(t, "s1-5", events[1].EventId)

	// GetAfter with evicted event (s1-1) returns all (full resync)
	events = buf.GetAfter("s1", "s1-1")
	require.Len(t, events, 3)
}

func TestEventBuffer_MultipleSessionsIsolated(t *testing.T) {
	buf := NewEventBuffer(10)

	buf.Append("s1", newTestEvent("s1"))
	buf.Append("s2", newTestEvent("s2"))
	buf.Append("s1", newTestEvent("s1"))
	buf.Append("s2", newTestEvent("s2"))

	// Session 1 events
	events := buf.GetAfter("s1", "")
	require.Len(t, events, 2)
	assert.Equal(t, "s1-1", events[0].EventId)
	assert.Equal(t, "s1-2", events[1].EventId)

	// Session 2 events
	events = buf.GetAfter("s2", "")
	require.Len(t, events, 2)
	assert.Equal(t, "s2-1", events[0].EventId)
	assert.Equal(t, "s2-2", events[1].EventId)

	// GetAfter for session 1, after first event
	events = buf.GetAfter("s1", "s1-1")
	require.Len(t, events, 1)
	assert.Equal(t, "s1-2", events[0].EventId)

	// Non-existent session
	events = buf.GetAfter("s3", "")
	assert.Nil(t, events)
}

func TestEventBuffer_CountersArePerSession(t *testing.T) {
	buf := NewEventBuffer(10)

	id1 := buf.Append("s1", newTestEvent("s1"))
	id2 := buf.Append("s2", newTestEvent("s2"))
	id3 := buf.Append("s1", newTestEvent("s1"))
	id4 := buf.Append("s2", newTestEvent("s2"))

	// Each session has independent counters
	assert.Equal(t, "s1-1", id1)
	assert.Equal(t, "s2-1", id2)
	assert.Equal(t, "s1-2", id3)
	assert.Equal(t, "s2-2", id4)
}

func TestEventBuffer_DefaultSize(t *testing.T) {
	buf := NewEventBuffer(0)
	assert.Equal(t, defaultBufferSize, buf.size)

	buf = NewEventBuffer(-1)
	assert.Equal(t, defaultBufferSize, buf.size)
}

func TestEventBuffer_ConcurrentAccess(t *testing.T) {
	// Buffer large enough to hold all events so none get evicted
	const goroutines = 10
	const eventsPerGoroutine = 20
	buf := NewEventBuffer(goroutines * eventsPerGoroutine)

	var wg sync.WaitGroup

	// Concurrent writers
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(sessionIdx int) {
			defer wg.Done()
			sessionID := fmt.Sprintf("session-%d", sessionIdx)
			for i := 0; i < eventsPerGoroutine; i++ {
				buf.Append(sessionID, newTestEvent(sessionID))
			}
		}(g)
	}

	// Concurrent readers
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(sessionIdx int) {
			defer wg.Done()
			sessionID := fmt.Sprintf("session-%d", sessionIdx)
			for i := 0; i < eventsPerGoroutine; i++ {
				buf.GetAfter(sessionID, "")
			}
		}(g)
	}

	wg.Wait()

	// Verify each session has the expected number of events
	for g := 0; g < goroutines; g++ {
		sessionID := fmt.Sprintf("session-%d", g)
		events := buf.GetAfter(sessionID, "")
		assert.Len(t, events, eventsPerGoroutine, "session %s should have %d events", sessionID, eventsPerGoroutine)
	}
}

func TestEventBuffer_WrapAroundPreservesOrder(t *testing.T) {
	buf := NewEventBuffer(5)

	// Fill buffer completely
	for i := 0; i < 5; i++ {
		buf.Append("s1", newTestEvent("s1"))
	}

	// Overwrite 3 entries
	for i := 0; i < 3; i++ {
		buf.Append("s1", newTestEvent("s1"))
	}

	// Should have events 4,5,6,7,8 (first 3 overwritten)
	events := buf.GetAfter("s1", "")
	require.Len(t, events, 5)
	assert.Equal(t, "s1-4", events[0].EventId)
	assert.Equal(t, "s1-5", events[1].EventId)
	assert.Equal(t, "s1-6", events[2].EventId)
	assert.Equal(t, "s1-7", events[3].EventId)
	assert.Equal(t, "s1-8", events[4].EventId)
}

func TestParseEventCounter(t *testing.T) {
	tests := []struct {
		name    string
		eventID string
		want    uint64
	}{
		{"valid", "session-123-42", 42},
		{"valid simple", "s1-1", 1},
		{"valid large", "s1-999999", 999999},
		{"empty", "", 0},
		{"no dash", "nodash", 0},
		{"trailing dash", "s1-", 0},
		{"non-numeric", "s1-abc", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseEventCounter(tt.eventID)
			assert.Equal(t, tt.want, got)
		})
	}
}

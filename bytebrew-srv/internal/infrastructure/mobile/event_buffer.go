package mobile

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
)

const defaultBufferSize = 1000

// BufferedEvent holds a proto event with its session and event IDs.
type BufferedEvent struct {
	SessionID string
	EventID   string
	Event     *pb.SessionEvent
}

// EventBuffer is a thread-safe circular buffer that stores recent session events
// for backfill on reconnect. Each session has an independent monotonic counter
// used to generate event IDs in the format "{sessionID}-{counter}".
type EventBuffer struct {
	mu       sync.RWMutex
	events   []BufferedEvent
	head     int              // next write position
	count    int              // current number of stored events
	size     int              // max capacity
	counters map[string]uint64 // per-session monotonic counter
}

// NewEventBuffer creates a new EventBuffer with the given capacity.
// If size <= 0, defaultBufferSize is used.
func NewEventBuffer(size int) *EventBuffer {
	if size <= 0 {
		size = defaultBufferSize
	}
	return &EventBuffer{
		events:   make([]BufferedEvent, size),
		size:     size,
		counters: make(map[string]uint64),
	}
}

// Append stores an event in the buffer and returns the generated eventID.
// The eventID is set on the proto event's EventId field before storing.
func (b *EventBuffer) Append(sessionID string, event *pb.SessionEvent) string {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.counters[sessionID]++
	eventID := fmt.Sprintf("%s-%d", sessionID, b.counters[sessionID])

	event.EventId = eventID

	b.events[b.head] = BufferedEvent{
		SessionID: sessionID,
		EventID:   eventID,
		Event:     event,
	}
	b.head = (b.head + 1) % b.size
	if b.count < b.size {
		b.count++
	}

	return eventID
}

// GetAfter returns all buffered events for the given sessionID that were
// appended after the event with lastEventID, in chronological order.
// If lastEventID is empty, all buffered events for the session are returned.
// If lastEventID is not found, all buffered events for the session are returned
// (the client likely missed too many events and needs a full resync).
func (b *EventBuffer) GetAfter(sessionID, lastEventID string) []*pb.SessionEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.count == 0 {
		return nil
	}

	// Collect all events for this session in chronological order
	sessionEvents := b.collectSessionEvents(sessionID)
	if len(sessionEvents) == 0 {
		return nil
	}

	if lastEventID == "" {
		return sessionEvents
	}

	// Find the position of lastEventID
	foundIdx := -1
	for i, evt := range sessionEvents {
		if evt.EventId == lastEventID {
			foundIdx = i
			break
		}
	}

	// If not found, return all (client missed too many events)
	if foundIdx < 0 {
		return sessionEvents
	}

	// Return events after the found position
	if foundIdx+1 >= len(sessionEvents) {
		return nil
	}
	return sessionEvents[foundIdx+1:]
}

// collectSessionEvents returns all buffered events for a session in chronological order.
// Must be called with b.mu held (at least RLock).
func (b *EventBuffer) collectSessionEvents(sessionID string) []*pb.SessionEvent {
	// Start index: oldest event in the circular buffer
	start := 0
	if b.count == b.size {
		start = b.head // head points to the oldest when buffer is full
	}

	result := make([]*pb.SessionEvent, 0)
	for i := 0; i < b.count; i++ {
		idx := (start + i) % b.size
		entry := b.events[idx]
		if entry.SessionID == sessionID {
			result = append(result, entry.Event)
		}
	}
	return result
}

// parseEventCounter extracts the monotonic counter from an eventID.
// EventID format: "{sessionID}-{counter}". Returns 0 if parsing fails.
// Note: used in event_buffer_test.go for verifying eventID format.
func parseEventCounter(eventID string) uint64 {
	lastDash := strings.LastIndex(eventID, "-")
	if lastDash < 0 || lastDash == len(eventID)-1 {
		return 0
	}
	counter, err := strconv.ParseUint(eventID[lastDash+1:], 10, 64)
	if err != nil {
		return 0
	}
	return counter
}

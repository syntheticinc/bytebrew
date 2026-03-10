package bridge

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// BufferedEvent holds a serialized event with metadata for backfill on reconnect.
type BufferedEvent struct {
	EventID   string
	SessionID string
	Event     map[string]interface{}
}

// EventBuffer is a thread-safe ring buffer that stores recent events for
// reconnect backfill. When full, the oldest events are overwritten.
type EventBuffer struct {
	events []BufferedEvent
	head   int // write position (next slot to overwrite)
	count  int // number of valid entries
	nextID int // monotonically increasing event counter
	mu     sync.RWMutex
}

// NewEventBuffer creates a new EventBuffer with the given capacity.
// If maxSize <= 0, it defaults to 1000.
func NewEventBuffer(maxSize int) *EventBuffer {
	if maxSize <= 0 {
		maxSize = 1000
	}
	return &EventBuffer{
		events: make([]BufferedEvent, maxSize),
		nextID: 1,
	}
}

// Append adds an event to the buffer and returns its event ID (e.g., "mevt-1").
func (b *EventBuffer) Append(sessionID string, event map[string]interface{}) string {
	b.mu.Lock()
	defer b.mu.Unlock()

	eventID := fmt.Sprintf("mevt-%d", b.nextID)
	b.nextID++

	b.events[b.head] = BufferedEvent{
		EventID:   eventID,
		SessionID: sessionID,
		Event:     event,
	}

	b.head = (b.head + 1) % len(b.events)
	if b.count < len(b.events) {
		b.count++
	}

	return eventID
}

// GetAfter returns all events buffered after the given event ID.
// Returns nil if the event ID is not found or is empty.
func (b *EventBuffer) GetAfter(lastEventID string) []BufferedEvent {
	if lastEventID == "" {
		return nil
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.count == 0 {
		return nil
	}

	// Parse the numeric part of the event ID.
	targetNum, err := parseEventNum(lastEventID)
	if err != nil {
		return nil
	}

	// Collect all valid entries in chronological order.
	start := (b.head - b.count + len(b.events)) % len(b.events)
	var result []BufferedEvent
	found := false

	for i := 0; i < b.count; i++ {
		idx := (start + i) % len(b.events)
		entry := b.events[idx]

		if found {
			result = append(result, entry)
			continue
		}

		num, err := parseEventNum(entry.EventID)
		if err != nil {
			continue
		}
		if num == targetNum {
			found = true
		}
	}

	return result
}

// GetAllForSession returns all buffered events for the given session in chronological order.
// Used for initial subscribe when no lastEventID is provided.
func (b *EventBuffer) GetAllForSession(sessionID string) []BufferedEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.count == 0 {
		return nil
	}

	start := (b.head - b.count + len(b.events)) % len(b.events)
	var result []BufferedEvent

	for i := 0; i < b.count; i++ {
		idx := (start + i) % len(b.events)
		entry := b.events[idx]
		if entry.SessionID == sessionID {
			result = append(result, entry)
		}
	}

	return result
}

// parseEventNum extracts the numeric suffix from an event ID like "mevt-42".
func parseEventNum(eventID string) (int, error) {
	parts := strings.SplitN(eventID, "-", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid event id format: %s", eventID)
	}
	return strconv.Atoi(parts[1])
}

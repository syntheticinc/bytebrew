package bridge

import (
	"log/slog"
	"strconv"
	"sync"

	"github.com/google/uuid"
	pb "github.com/syntheticinc/bytebrew/bytebrew/engine/api/proto/gen"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/service/eventformat"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/service/eventstore"
)

// DeviceSubscription tracks a mobile device's subscription to a session.
type DeviceSubscription struct {
	DeviceID    string
	SessionID   string
	LastEventID string
}

// MessageSender sends a message to a specific device.
type MessageSender interface {
	SendMessage(deviceID string, msg *MobileMessage) error
}

// EventStoreReader reads persisted events for backfill on reconnect (consumer-side interface).
type EventStoreReader interface {
	GetAfter(sessionID string, lastEventID int64) ([]eventstore.StoredEvent, error)
	GetAll(sessionID string) ([]eventstore.StoredEvent, error)
}

// EventBroadcaster serializes SessionEvents into the flat mobile format
// and sends them to subscribed devices via the MessageSender.
// Events are persisted by EventStream; this broadcaster reads from the
// event store for backfill on reconnect.
type EventBroadcaster struct {
	sender MessageSender
	store  EventStoreReader

	subscribers map[string]*DeviceSubscription // deviceID → subscription
	mu          sync.RWMutex
}

// NewEventBroadcaster creates a new EventBroadcaster.
func NewEventBroadcaster(sender MessageSender, store EventStoreReader) *EventBroadcaster {
	return &EventBroadcaster{
		sender:      sender,
		store:       store,
		subscribers: make(map[string]*DeviceSubscription),
	}
}

// Subscribe registers a device to receive events for the given session.
// If lastEventID is provided, missed events are backfilled from the event store.
func (b *EventBroadcaster) Subscribe(deviceID, sessionID, lastEventID string) {
	b.mu.Lock()
	b.subscribers[deviceID] = &DeviceSubscription{
		DeviceID:    deviceID,
		SessionID:   sessionID,
		LastEventID: lastEventID,
	}
	b.mu.Unlock()

	slog.Info("device subscribed to session", "device_id", deviceID, "session_id", sessionID, "last_event_id", lastEventID)

	// Backfill missed events from event store.
	lastID, _ := strconv.ParseInt(lastEventID, 10, 64)

	var missed []eventstore.StoredEvent
	var err error
	if lastID == 0 {
		missed, err = b.store.GetAll(sessionID)
	} else {
		missed, err = b.store.GetAfter(sessionID, lastID)
	}

	if err != nil {
		slog.Error("backfill from event store failed", "device_id", deviceID, "session_id", sessionID, "error", err)
	}

	for _, evt := range missed {
		b.sendToDevice(deviceID, sessionID, evt.JSON, strconv.FormatInt(evt.ID, 10))
	}

	// BackfillComplete marker so the client knows backfill is done.
	b.sendToDevice(deviceID, sessionID, map[string]interface{}{"type": "BackfillComplete"}, "")
}

// Unsubscribe removes a device's subscription.
func (b *EventBroadcaster) Unsubscribe(deviceID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.subscribers, deviceID)
	slog.Info("device unsubscribed", "device_id", deviceID)
}

// BroadcastEvent serializes a SessionEvent to the flat mobile format
// and fans out to all devices subscribed to the event's session.
// The event already has an EventId assigned by EventStream.
func (b *EventBroadcaster) BroadcastEvent(sessionID string, event *pb.SessionEvent) {
	serialized := eventformat.SerializeForMobile(event)
	if serialized == nil {
		return
	}

	eventID := event.GetEventId()

	b.mu.RLock()
	var targets []*DeviceSubscription
	for _, sub := range b.subscribers {
		if sub.SessionID == sessionID {
			targets = append(targets, sub)
		}
	}
	b.mu.RUnlock()

	for _, sub := range targets {
		b.sendToDevice(sub.DeviceID, sessionID, serialized, eventID)
	}
}

// SendSessionStatus sends a synthetic session status event to a specific device.
// Used after subscribe to ensure the device knows the current processing state,
// preventing stuck-spinner when ProcessingStopped was lost during TCP death.
//
// NOT buffered: synthetic events are sent without event_id so the mobile client's
// dedup logic (based on event_id) won't skip them. This avoids collisions when
// the server restarts and the mevt counter resets (mobile may already have old
// mevt-1 in its seen set).
func (b *EventBroadcaster) SendSessionStatus(deviceID, sessionID string, processing bool) {
	eventType := "ProcessingStopped"
	state := "idle"
	if processing {
		eventType = "ProcessingStarted"
		state = "processing"
	}

	statusEvent := map[string]interface{}{
		"type":  eventType,
		"state": state,
	}
	// Empty event_id → mobile skips dedup check → always processed.
	b.sendToDevice(deviceID, sessionID, statusEvent, "")
}

func (b *EventBroadcaster) sendToDevice(deviceID, sessionID string, event map[string]interface{}, eventID string) {
	msg := &MobileMessage{
		Type:      "session_event",
		RequestID: uuid.New().String(),
		DeviceID:  deviceID,
		Payload: map[string]interface{}{
			"session_id": sessionID,
			"event":      event,
			"event_id":   eventID,
		},
	}

	if err := b.sender.SendMessage(deviceID, msg); err != nil {
		slog.Error("broadcast to device failed", "device_id", deviceID, "event_id", eventID, "error", err)
	}
}

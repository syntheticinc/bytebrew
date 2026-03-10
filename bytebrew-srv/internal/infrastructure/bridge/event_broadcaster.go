package bridge

import (
	"log/slog"
	"sync"

	"github.com/google/uuid"
	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
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

// EventBroadcaster serializes SessionEvents into the flat mobile format,
// buffers them for reconnect backfill, and sends them to subscribed devices
// via the MessageSender.
type EventBroadcaster struct {
	sender MessageSender
	buffer *EventBuffer

	subscribers map[string]*DeviceSubscription // deviceID → subscription
	mu          sync.RWMutex
}

// NewEventBroadcaster creates a new EventBroadcaster.
func NewEventBroadcaster(sender MessageSender) *EventBroadcaster {
	return &EventBroadcaster{
		sender:      sender,
		buffer:      NewEventBuffer(1000),
		subscribers: make(map[string]*DeviceSubscription),
	}
}

// Subscribe registers a device to receive events for the given session.
// If lastEventID is provided, missed events are backfilled immediately.
func (b *EventBroadcaster) Subscribe(deviceID, sessionID, lastEventID string) {
	b.mu.Lock()
	b.subscribers[deviceID] = &DeviceSubscription{
		DeviceID:    deviceID,
		SessionID:   sessionID,
		LastEventID: lastEventID,
	}
	b.mu.Unlock()

	slog.Info("device subscribed to session", "device_id", deviceID, "session_id", sessionID)

	if lastEventID == "" {
		return
	}

	// Backfill missed events.
	missed := b.buffer.GetAfter(lastEventID)
	for _, evt := range missed {
		if evt.SessionID != sessionID {
			continue
		}
		b.sendToDevice(deviceID, sessionID, evt.Event, evt.EventID)
	}
}

// Unsubscribe removes a device's subscription.
func (b *EventBroadcaster) Unsubscribe(deviceID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.subscribers, deviceID)
	slog.Info("device unsubscribed", "device_id", deviceID)
}

// BroadcastEvent serializes a SessionEvent to the flat mobile format, buffers
// it, and fans out to all devices subscribed to the event's session.
func (b *EventBroadcaster) BroadcastEvent(sessionID string, event *pb.SessionEvent) {
	serialized := serializeEventForMobile(event)
	if serialized == nil {
		return
	}

	eventID := b.buffer.Append(sessionID, serialized)

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

// serializeEventForMobile converts a proto SessionEvent into the flat JSON
// format expected by mobile clients.
func serializeEventForMobile(event *pb.SessionEvent) map[string]interface{} {
	switch event.GetType() {
	case pb.SessionEventType_SESSION_EVENT_ANSWER:
		return map[string]interface{}{
			"type":     "MessageCompleted",
			"content":  event.GetContent(),
			"role":     "assistant",
			"agent_id": event.GetAgentId(),
		}

	case pb.SessionEventType_SESSION_EVENT_ANSWER_CHUNK:
		return map[string]interface{}{
			"type":     "StreamingProgress",
			"content":  event.GetContent(),
			"agent_id": event.GetAgentId(),
		}

	case pb.SessionEventType_SESSION_EVENT_TOOL_EXECUTION_START:
		args := make(map[string]interface{}, len(event.GetToolArguments()))
		for k, v := range event.GetToolArguments() {
			args[k] = v
		}
		return map[string]interface{}{
			"type":      "ToolExecutionStarted",
			"call_id":   event.GetCallId(),
			"tool_name": event.GetToolName(),
			"arguments": args,
			"agent_id":  event.GetAgentId(),
		}

	case pb.SessionEventType_SESSION_EVENT_TOOL_EXECUTION_END:
		return map[string]interface{}{
			"type":           "ToolExecutionCompleted",
			"call_id":        event.GetCallId(),
			"tool_name":      event.GetToolName(),
			"result_summary": event.GetToolResultSummary(),
			"has_error":      event.GetToolHasError(),
			"agent_id":       event.GetAgentId(),
		}

	case pb.SessionEventType_SESSION_EVENT_REASONING:
		return map[string]interface{}{
			"type":     "ReasoningChunk",
			"content":  event.GetContent(),
			"agent_id": event.GetAgentId(),
		}

	case pb.SessionEventType_SESSION_EVENT_ASK_USER:
		return map[string]interface{}{
			"type":     "AskUserRequested",
			"question": event.GetQuestion(),
			"options":  event.GetOptions(),
			"agent_id": event.GetAgentId(),
		}

	case pb.SessionEventType_SESSION_EVENT_PROCESSING_STARTED:
		return map[string]interface{}{
			"type":  "ProcessingStarted",
			"state": "processing",
		}

	case pb.SessionEventType_SESSION_EVENT_PROCESSING_STOPPED:
		return map[string]interface{}{
			"type":  "ProcessingStopped",
			"state": "idle",
		}

	case pb.SessionEventType_SESSION_EVENT_ERROR:
		msg := event.GetContent()
		if detail := event.GetErrorDetail(); detail != nil {
			msg = detail.GetMessage()
		}
		return map[string]interface{}{
			"type":    "Error",
			"message": msg,
			"code":    "error",
		}

	case pb.SessionEventType_SESSION_EVENT_PLAN_UPDATE:
		steps := make([]map[string]interface{}, 0, len(event.GetPlanSteps()))
		for _, s := range event.GetPlanSteps() {
			steps = append(steps, map[string]interface{}{
				"title":  s.GetTitle(),
				"status": s.GetStatus(),
			})
		}
		return map[string]interface{}{
			"type":      "PlanUpdated",
			"plan_name": event.GetPlanName(),
			"steps":     steps,
			"agent_id":  event.GetAgentId(),
		}

	default:
		slog.Warn("unknown session event type", "type", event.GetType().String())
		return nil
	}
}

package mobile

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/google/uuid"
	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
)

const (
	// subscriberChannelSize is the buffered channel size for each mobile subscriber.
	subscriberChannelSize = 100
)

// MobileSubscriber represents a mobile client subscribed to session events.
type MobileSubscriber struct {
	id      string
	eventCh chan *pb.SessionEvent
	done    chan struct{}
}

// EventBroadcaster converts domain.AgentEvent to proto SessionEvent and
// manages mobile subscribers per session. It stores recent events in a ring
// buffer so that reconnecting clients can receive missed events via backfill.
type EventBroadcaster struct {
	mu          sync.RWMutex
	subscribers map[string]map[string]*MobileSubscriber // sessionID -> subscriberID -> subscriber
	buffer      *EventBuffer
}

// NewEventBroadcaster creates a new EventBroadcaster with an event buffer for backfill.
func NewEventBroadcaster(buffer *EventBuffer) *EventBroadcaster {
	return &EventBroadcaster{
		subscribers: make(map[string]map[string]*MobileSubscriber),
		buffer:      buffer,
	}
}

// Subscribe registers a new subscriber for a session. Returns an event channel
// and an unsubscribe function. The caller must call unsubscribe when done.
func (b *EventBroadcaster) Subscribe(sessionID, subscriberID string) (<-chan *pb.SessionEvent, func()) {
	if subscriberID == "" {
		subscriberID = uuid.New().String()
	}

	sub := &MobileSubscriber{
		id:      subscriberID,
		eventCh: make(chan *pb.SessionEvent, subscriberChannelSize),
		done:    make(chan struct{}),
	}

	b.mu.Lock()
	if _, ok := b.subscribers[sessionID]; !ok {
		b.subscribers[sessionID] = make(map[string]*MobileSubscriber)
	}
	b.subscribers[sessionID][subscriberID] = sub
	b.mu.Unlock()

	slog.Info("mobile subscriber added", "session_id", sessionID, "subscriber_id", subscriberID)

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		if subs, ok := b.subscribers[sessionID]; ok {
			if s, exists := subs[subscriberID]; exists {
				close(s.done)
				delete(subs, subscriberID)
				if len(subs) == 0 {
					delete(b.subscribers, sessionID)
				}
				slog.Info("mobile subscriber removed", "session_id", sessionID, "subscriber_id", subscriberID)
			}
		}
	}

	return sub.eventCh, unsubscribe
}

// Broadcast converts a domain.AgentEvent to a proto SessionEvent, stores it
// in the ring buffer for backfill, and sends it to all subscribers for the
// given session. Events are dropped (with a warning) if a subscriber's channel
// is full.
func (b *EventBroadcaster) Broadcast(sessionID string, event *domain.AgentEvent) {
	protoEvent := convertEvent(sessionID, event)
	if protoEvent == nil {
		return
	}

	// Store in ring buffer and assign event ID
	eventID := b.buffer.Append(sessionID, protoEvent)
	event.EventID = eventID

	b.mu.RLock()
	subs, ok := b.subscribers[sessionID]
	if !ok || len(subs) == 0 {
		b.mu.RUnlock()
		return
	}

	// Copy subscriber references under read lock to avoid holding lock during send
	targets := make([]*MobileSubscriber, 0, len(subs))
	for _, sub := range subs {
		targets = append(targets, sub)
	}
	b.mu.RUnlock()

	for _, sub := range targets {
		select {
		case sub.eventCh <- protoEvent:
			// sent successfully
		case <-sub.done:
			// subscriber already unsubscribed
		default:
			slog.Warn("mobile subscriber channel full, dropping event",
				"session_id", sessionID,
				"subscriber_id", sub.id,
				"event_type", event.Type)
		}
	}
}

// GetMissedEvents returns buffered events for the session that occurred after
// the given lastEventID. Used for backfill on client reconnect.
func (b *EventBroadcaster) GetMissedEvents(sessionID, lastEventID string) []*pb.SessionEvent {
	return b.buffer.GetAfter(sessionID, lastEventID)
}

// HasSubscribers returns true if the session has at least one subscriber.
func (b *EventBroadcaster) HasSubscribers(sessionID string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	subs, ok := b.subscribers[sessionID]
	return ok && len(subs) > 0
}

// convertEvent maps a domain.AgentEvent to a proto SessionEvent.
func convertEvent(sessionID string, event *domain.AgentEvent) *pb.SessionEvent {
	protoEvent := &pb.SessionEvent{
		EventId:   uuid.New().String(),
		SessionId: sessionID,
		Timestamp: event.Timestamp.Unix(),
		AgentId:   event.AgentID,
		Step:      int32(event.Step),
	}

	if protoEvent.AgentId == "" {
		protoEvent.AgentId = "supervisor"
	}

	switch event.Type {
	case domain.EventTypeAnswer:
		protoEvent.Type = pb.SessionEventType_SESSION_EVENT_TYPE_AGENT_MESSAGE
		protoEvent.Payload = &pb.SessionEvent_AgentMessage{
			AgentMessage: &pb.AgentMessageEvent{
				Content:    sanitizeUTF8(event.Content),
				IsComplete: event.IsComplete,
			},
		}

	case domain.EventTypeAnswerChunk:
		protoEvent.Type = pb.SessionEventType_SESSION_EVENT_TYPE_ANSWER_CHUNK
		protoEvent.Payload = &pb.SessionEvent_AgentMessage{
			AgentMessage: &pb.AgentMessageEvent{
				Content:    sanitizeUTF8(event.Content),
				IsComplete: false,
			},
		}

	case domain.EventTypeToolCall:
		protoEvent.Type = pb.SessionEventType_SESSION_EVENT_TYPE_TOOL_CALL_START
		toolName := extractToolName(event)
		callID := fmt.Sprintf("server-%s-%d", toolName, event.Step)
		args := extractToolArguments(event)
		protoEvent.Payload = &pb.SessionEvent_ToolCallStart{
			ToolCallStart: &pb.ToolCallStartEvent{
				CallId:    callID,
				ToolName:  toolName,
				Arguments: args,
			},
		}

	case domain.EventTypeToolResult:
		protoEvent.Type = pb.SessionEventType_SESSION_EVENT_TYPE_TOOL_CALL_END
		toolName := extractMetadataString(event.Metadata, "tool_name")
		callID := fmt.Sprintf("server-%s-%d", toolName, event.Step)
		summary := extractMetadataString(event.Metadata, "summary")
		protoEvent.Payload = &pb.SessionEvent_ToolCallEnd{
			ToolCallEnd: &pb.ToolCallEndEvent{
				CallId:        callID,
				ToolName:      toolName,
				ResultSummary: sanitizeUTF8(summary),
				HasError:      false,
			},
		}

	case domain.EventTypeReasoning:
		protoEvent.Type = pb.SessionEventType_SESSION_EVENT_TYPE_REASONING
		protoEvent.Payload = &pb.SessionEvent_Reasoning{
			Reasoning: &pb.ReasoningEvent{
				Content:    sanitizeUTF8(event.Content),
				IsComplete: event.IsComplete,
			},
		}

	case domain.EventTypeUserQuestion:
		protoEvent.Type = pb.SessionEventType_SESSION_EVENT_TYPE_ASK_USER
		protoEvent.Payload = &pb.SessionEvent_AskUser{
			AskUser: &pb.AskUserEvent{
				Question: sanitizeUTF8(event.Content),
			},
		}

	case domain.EventTypePlanCreated, domain.EventTypePlanProgress, domain.EventTypePlanCompleted:
		protoEvent.Type = pb.SessionEventType_SESSION_EVENT_TYPE_PLAN_UPDATE
		protoEvent.Payload = &pb.SessionEvent_Plan{
			Plan: buildPlanEvent(event),
		}

	case domain.EventTypeError:
		protoEvent.Type = pb.SessionEventType_SESSION_EVENT_TYPE_ERROR
		errEvent := &pb.ErrorEvent{}
		if event.Error != nil {
			errEvent.Code = event.Error.Code
			errEvent.Message = sanitizeUTF8(event.Error.Message)
		} else {
			errEvent.Message = sanitizeUTF8(event.Content)
		}
		protoEvent.Payload = &pb.SessionEvent_ErrorEvent{
			ErrorEvent: errEvent,
		}

	case domain.EventTypeAgentSpawned, domain.EventTypeAgentCompleted, domain.EventTypeAgentFailed:
		// Agent lifecycle events sent as status updates
		protoEvent.Type = pb.SessionEventType_SESSION_EVENT_TYPE_SESSION_STATUS
		msg := fmt.Sprintf("[%s] %s: %s", event.Type, protoEvent.AgentId, sanitizeUTF8(event.Content))
		protoEvent.Payload = &pb.SessionEvent_SessionStatus{
			SessionStatus: &pb.SessionStatusEvent{
				State:   pb.SessionState_SESSION_STATE_ACTIVE,
				Message: msg,
			},
		}

	default:
		slog.Warn("unknown agent event type for mobile broadcast", "type", event.Type)
		return nil
	}

	return protoEvent
}

// extractToolName gets the tool name from event content or metadata.
func extractToolName(event *domain.AgentEvent) string {
	if name := extractMetadataString(event.Metadata, "tool_name"); name != "" {
		return name
	}
	return event.Content
}

// extractToolArguments parses function_arguments metadata into map[string]string.
func extractToolArguments(event *domain.AgentEvent) map[string]string {
	args := make(map[string]string)

	argsJSON := extractMetadataString(event.Metadata, "function_arguments")
	if argsJSON == "" {
		return args
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &parsed); err != nil {
		args["_json"] = sanitizeUTF8(argsJSON)
		return args
	}

	for k, v := range parsed {
		switch val := v.(type) {
		case string:
			args[k] = sanitizeUTF8(val)
		case float64:
			args[k] = fmt.Sprintf("%.0f", val)
		case bool:
			args[k] = fmt.Sprintf("%v", val)
		case []interface{}:
			var parts []string
			for _, item := range val {
				parts = append(parts, sanitizeUTF8(fmt.Sprintf("%v", item)))
			}
			args[k] = strings.Join(parts, "\n")
		default:
			if jsonVal, err := json.Marshal(val); err == nil {
				args[k] = sanitizeUTF8(string(jsonVal))
			}
		}
	}

	return args
}

// extractMetadataString safely extracts a string value from metadata.
func extractMetadataString(metadata map[string]interface{}, key string) string {
	if metadata == nil {
		return ""
	}
	if v, ok := metadata[key].(string); ok {
		return v
	}
	return ""
}

// buildPlanEvent constructs a PlanEvent from agent event metadata.
func buildPlanEvent(event *domain.AgentEvent) *pb.PlanEvent {
	planEvent := &pb.PlanEvent{
		PlanName: sanitizeUTF8(event.Content),
	}

	// Try to extract steps from metadata
	if stepsRaw, ok := event.Metadata["steps"]; ok {
		if steps, ok := stepsRaw.([]interface{}); ok {
			for _, stepRaw := range steps {
				stepMap, ok := stepRaw.(map[string]interface{})
				if !ok {
					continue
				}

				stepEvent := &pb.PlanStepEvent{}
				if title, ok := stepMap["description"].(string); ok {
					stepEvent.Title = sanitizeUTF8(title)
				}
				if status, ok := stepMap["status"].(string); ok {
					stepEvent.Status = mapPlanStepStatus(status)
				}
				planEvent.Steps = append(planEvent.Steps, stepEvent)
			}
		}
	}

	// If no steps from metadata, try to build from progress info
	if len(planEvent.Steps) == 0 {
		if currentStep := extractMetadataString(event.Metadata, "current_step"); currentStep != "" {
			planEvent.Steps = append(planEvent.Steps, &pb.PlanStepEvent{
				Title:  sanitizeUTF8(currentStep),
				Status: pb.PlanStepStatus_PLAN_STEP_STATUS_IN_PROGRESS,
			})
		}
	}

	return planEvent
}

// mapPlanStepStatus converts a domain plan step status string to proto enum.
func mapPlanStepStatus(status string) pb.PlanStepStatus {
	switch status {
	case string(domain.StepStatusPending):
		return pb.PlanStepStatus_PLAN_STEP_STATUS_PENDING
	case string(domain.StepStatusInProgress):
		return pb.PlanStepStatus_PLAN_STEP_STATUS_IN_PROGRESS
	case string(domain.StepStatusCompleted):
		return pb.PlanStepStatus_PLAN_STEP_STATUS_COMPLETED
	case string(domain.StepStatusFailed):
		return pb.PlanStepStatus_PLAN_STEP_STATUS_FAILED
	default:
		return pb.PlanStepStatus_PLAN_STEP_STATUS_UNSPECIFIED
	}
}

// sanitizeUTF8 removes invalid UTF-8 characters from a string.
func sanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	return string([]rune(s))
}

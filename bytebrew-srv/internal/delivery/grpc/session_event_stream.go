package grpc

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"

	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
)

// SessionPublisher publishes session events (consumer-side interface).
type SessionPublisher interface {
	PublishEvent(sessionID string, event *pb.SessionEvent)
}

// SessionEventStream converts domain.AgentEvent to pb.SessionEvent and publishes via SessionPublisher.
// Implements domain.AgentEventStream.
type SessionEventStream struct {
	sessionID string
	publisher SessionPublisher
	eventSeq  atomic.Int64
}

// NewSessionEventStream creates a new event stream that publishes to a SessionPublisher.
func NewSessionEventStream(sessionID string, publisher SessionPublisher) *SessionEventStream {
	return &SessionEventStream{
		sessionID: sessionID,
		publisher: publisher,
	}
}

// Send converts a domain AgentEvent to a proto SessionEvent and publishes it.
func (s *SessionEventStream) Send(event *domain.AgentEvent) error {
	pbEvent := s.convertEvent(event)
	if pbEvent == nil {
		return nil
	}

	pbEvent.EventId = fmt.Sprintf("evt-%d", s.eventSeq.Add(1))
	pbEvent.SessionId = s.sessionID

	s.publisher.PublishEvent(s.sessionID, pbEvent)
	return nil
}

// PublishProcessingStarted sends a PROCESSING_STARTED event.
func (s *SessionEventStream) PublishProcessingStarted() {
	s.publisher.PublishEvent(s.sessionID, &pb.SessionEvent{
		EventId:   fmt.Sprintf("evt-%d", s.eventSeq.Add(1)),
		SessionId: s.sessionID,
		Type:      pb.SessionEventType_SESSION_EVENT_PROCESSING_STARTED,
	})
}

// PublishProcessingStopped sends a PROCESSING_STOPPED event.
func (s *SessionEventStream) PublishProcessingStopped() {
	s.publisher.PublishEvent(s.sessionID, &pb.SessionEvent{
		EventId:   fmt.Sprintf("evt-%d", s.eventSeq.Add(1)),
		SessionId: s.sessionID,
		Type:      pb.SessionEventType_SESSION_EVENT_PROCESSING_STOPPED,
	})
}

// PublishError sends an ERROR event.
func (s *SessionEventStream) PublishError(err error) {
	s.publisher.PublishEvent(s.sessionID, &pb.SessionEvent{
		EventId:   fmt.Sprintf("evt-%d", s.eventSeq.Add(1)),
		SessionId: s.sessionID,
		Type:      pb.SessionEventType_SESSION_EVENT_ERROR,
		Content:   err.Error(),
		ErrorDetail: &pb.Error{
			Code:    "internal",
			Message: err.Error(),
		},
	})
}

// PublishAnswerChunk sends an ANSWER_CHUNK event.
func (s *SessionEventStream) PublishAnswerChunk(chunk string) {
	s.publisher.PublishEvent(s.sessionID, &pb.SessionEvent{
		EventId:   fmt.Sprintf("evt-%d", s.eventSeq.Add(1)),
		SessionId: s.sessionID,
		Type:      pb.SessionEventType_SESSION_EVENT_ANSWER_CHUNK,
		Content:   chunk,
	})
}

func (s *SessionEventStream) convertEvent(event *domain.AgentEvent) *pb.SessionEvent {
	agentID := event.AgentID
	if agentID == "" {
		agentID = "supervisor"
	}

	switch event.Type {
	case domain.EventTypeAnswerChunk:
		return &pb.SessionEvent{
			Type:    pb.SessionEventType_SESSION_EVENT_ANSWER_CHUNK,
			Content: sanitizeUTF8(event.Content),
			AgentId: agentID,
			Step:    int32(event.Step),
		}

	case domain.EventTypeAnswer:
		return &pb.SessionEvent{
			Type:    pb.SessionEventType_SESSION_EVENT_ANSWER,
			Content: sanitizeUTF8(event.Content),
			AgentId: agentID,
		}

	case domain.EventTypeToolCall:
		args := parseToolArguments(event)
		callID := fmt.Sprintf("server-%s-%d", event.Content, event.Step)
		return &pb.SessionEvent{
			Type:          pb.SessionEventType_SESSION_EVENT_TOOL_EXECUTION_START,
			ToolName:      event.Content,
			CallId:        callID,
			ToolArguments: args,
			AgentId:       agentID,
			Step:          int32(event.Step),
		}

	case domain.EventTypeToolResult:
		toolName := ""
		if name, ok := event.Metadata["tool_name"].(string); ok {
			toolName = name
		}
		callID := fmt.Sprintf("server-%s-%d", toolName, event.Step)

		summary := sanitizeUTF8(event.Content)
		if s, ok := event.Metadata["summary"].(string); ok {
			summary = sanitizeUTF8(s)
		}

		return &pb.SessionEvent{
			Type:             pb.SessionEventType_SESSION_EVENT_TOOL_EXECUTION_END,
			ToolName:         toolName,
			CallId:           callID,
			ToolResultSummary: summary,
			ToolHasError:     event.Error != nil,
			AgentId:          agentID,
			Step:             int32(event.Step),
		}

	case domain.EventTypeReasoning:
		return &pb.SessionEvent{
			Type:    pb.SessionEventType_SESSION_EVENT_REASONING,
			Content: sanitizeUTF8(event.Content),
			AgentId: agentID,
			Step:    int32(event.Step),
		}

	case domain.EventTypePlanCreated, domain.EventTypePlanProgress, domain.EventTypePlanCompleted:
		return s.convertPlanEvent(event, agentID)

	case domain.EventTypeUserQuestion:
		question := sanitizeUTF8(event.Content)
		callID := ""
		if id, ok := event.Metadata["call_id"].(string); ok {
			callID = id
		}
		return &pb.SessionEvent{
			Type:     pb.SessionEventType_SESSION_EVENT_ASK_USER,
			Content:  question,
			Question: question,
			CallId:   callID,
			AgentId:  agentID,
		}

	case domain.EventTypeError:
		content := sanitizeUTF8(event.Content)
		var errDetail *pb.Error
		if event.Error != nil {
			content = sanitizeUTF8(event.Error.Message)
			errDetail = &pb.Error{
				Code:    event.Error.Code,
				Message: sanitizeUTF8(event.Error.Message),
			}
		}
		return &pb.SessionEvent{
			Type:        pb.SessionEventType_SESSION_EVENT_ERROR,
			Content:     content,
			ErrorDetail: errDetail,
		}

	case domain.EventTypeAgentSpawned, domain.EventTypeAgentCompleted, domain.EventTypeAgentFailed:
		// Agent lifecycle events sent as answer chunks with descriptive content
		eventTypeStr := string(event.Type)
		content := fmt.Sprintf("[%s] %s: %s", eventTypeStr, agentID, sanitizeUTF8(event.Content))
		return &pb.SessionEvent{
			Type:    pb.SessionEventType_SESSION_EVENT_ANSWER_CHUNK,
			Content: content,
			AgentId: agentID,
		}

	default:
		return nil
	}
}

// convertPlanEvent converts plan-related domain events to SessionEvent.
func (s *SessionEventStream) convertPlanEvent(event *domain.AgentEvent, agentID string) *pb.SessionEvent {
	pbEvent := &pb.SessionEvent{
		Type:    pb.SessionEventType_SESSION_EVENT_PLAN_UPDATE,
		AgentId: agentID,
	}

	if name, ok := event.Metadata["plan_name"].(string); ok {
		pbEvent.PlanName = name
	}

	// Extract plan steps from metadata
	if stepsRaw, ok := event.Metadata["plan_steps"]; ok {
		pbEvent.PlanSteps = extractPlanSteps(stepsRaw)
	}

	pbEvent.Content = sanitizeUTF8(event.Content)
	return pbEvent
}

// parseToolArguments extracts tool arguments from event metadata.
func parseToolArguments(event *domain.AgentEvent) map[string]string {
	args := make(map[string]string)
	argsJSON, ok := event.Metadata["function_arguments"].(string)
	if !ok || argsJSON == "" {
		return args
	}

	var parsedArgs map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &parsedArgs); err != nil {
		args["_json"] = sanitizeUTF8(argsJSON)
		return args
	}

	for k, v := range parsedArgs {
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

// extractPlanSteps converts raw metadata into PlanStep proto messages.
func extractPlanSteps(stepsRaw interface{}) []*pb.PlanStep {
	// Try as []interface{} (common from JSON)
	stepsSlice, ok := stepsRaw.([]interface{})
	if !ok {
		return nil
	}

	steps := make([]*pb.PlanStep, 0, len(stepsSlice))
	for _, raw := range stepsSlice {
		stepMap, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		step := &pb.PlanStep{}
		if title, ok := stepMap["title"].(string); ok {
			step.Title = title
		}
		if status, ok := stepMap["status"].(string); ok {
			step.Status = status
		}
		steps = append(steps, step)
	}
	return steps
}

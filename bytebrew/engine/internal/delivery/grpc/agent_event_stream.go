package grpc

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	pb "github.com/syntheticinc/bytebrew/bytebrew/engine/api/proto/gen"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
)

// GrpcAgentEventStream implements AgentEventStream interface for gRPC streaming
type GrpcAgentEventStream struct {
	stream         pb.FlowService_ExecuteFlowServer
	streamWriter   *StreamWriter
	sessionID      string
	toolClassifier domain.ToolClassifier
}

// NewGrpcAgentEventStream creates a new gRPC agent event stream
func NewGrpcAgentEventStream(stream pb.FlowService_ExecuteFlowServer, sessionID string, toolClassifier domain.ToolClassifier, streamWriter *StreamWriter) *GrpcAgentEventStream {
	return &GrpcAgentEventStream{
		stream:         stream,
		streamWriter:   streamWriter,
		sessionID:      sessionID,
		toolClassifier: toolClassifier,
	}
}

// Send sends an agent event to the gRPC stream
func (s *GrpcAgentEventStream) Send(event *domain.AgentEvent) error {
	ctx := s.stream.Context()

	// Map agent event type to proto response type
	var responseType pb.ResponseType
	var content string
	var toolCall *pb.ToolCall
	var toolResult *pb.ToolResult
	var thought *pb.ThoughtStep
	var reasoning *pb.ReasoningContent

	switch event.Type {
	case domain.EventTypeAnswer:
		responseType = pb.ResponseType_RESPONSE_TYPE_ANSWER
		content = sanitizeUTF8(event.Content)
		thought = &pb.ThoughtStep{
			Content: content,
		}

	case domain.EventTypeReasoning:
		responseType = pb.ResponseType_RESPONSE_TYPE_REASONING
		content = sanitizeUTF8(event.Content)
		reasoning = &pb.ReasoningContent{
			Thinking:   content,
			IsComplete: event.IsComplete,
		}

	case domain.EventTypeToolCall:
		toolName := event.Content

		// Client-side tools (proxied): StreamBasedClientOperationsProxy already sends TOOL_CALL
		// Server-side tools: We send TOOL_CALL from here
		if s.toolClassifier.ClassifyTool(toolName) == domain.ToolTypeProxied {
			// Skip - proxy handles this
			slog.DebugContext(ctx, "proxied tool call (not sent - proxy handles)",
				"tool_name", toolName,
				"step", event.Step)
			return nil
		}

		// Server-side tool - send TOOL_CALL to client
		responseType = pb.ResponseType_RESPONSE_TYPE_TOOL_CALL

		// Generate call ID
		callID := fmt.Sprintf("server-%s-%d", toolName, event.Step)

		// Parse arguments from metadata - convert JSON to map[string]string
		args := make(map[string]string)
		if argsJSON, ok := event.Metadata["function_arguments"].(string); ok && argsJSON != "" {
			// Try to parse JSON and extract individual arguments
			var parsedArgs map[string]interface{}
			if err := json.Unmarshal([]byte(argsJSON), &parsedArgs); err == nil {
				for k, v := range parsedArgs {
					switch val := v.(type) {
					case string:
						args[k] = sanitizeUTF8(val)
					case float64:
						args[k] = fmt.Sprintf("%.0f", val)
					case bool:
						args[k] = fmt.Sprintf("%v", val)
					case []interface{}:
						// Handle arrays: join elements with newlines
						var parts []string
						for _, item := range val {
							parts = append(parts, sanitizeUTF8(fmt.Sprintf("%v", item)))
						}
						args[k] = strings.Join(parts, "\n")
					default:
						// For complex types, store as JSON string
						if jsonVal, err := json.Marshal(val); err == nil {
							args[k] = sanitizeUTF8(string(jsonVal))
						}
					}
				}
			} else {
				// Fallback: store raw JSON
				args["_json"] = sanitizeUTF8(argsJSON)
			}
		}

		toolCall = &pb.ToolCall{
			ToolName:  toolName,
			Arguments: args,
			CallId:    callID,
		}

		slog.InfoContext(ctx, "server-side tool call (sending to client)",
			"tool_name", toolName,
			"call_id", callID,
			"step", event.Step)

	case domain.EventTypeToolResult:
		responseType = pb.ResponseType_RESPONSE_TYPE_TOOL_RESULT
		content = sanitizeUTF8(event.Content)

		// Extract tool name from metadata
		toolName := ""
		if name, ok := event.Metadata["tool_name"].(string); ok {
			toolName = name
		}

		// Check if this is a server-side tool result
		if s.toolClassifier.ClassifyTool(toolName) == domain.ToolTypeServerSide {
			// Generate same call ID as was used in TOOL_CALL event
			callID := fmt.Sprintf("server-%s-%d", toolName, event.Step)

			// Get full result from metadata (event.Content is just a preview)
			fullResult := event.Content
			if full, ok := event.Metadata["full_result"].(string); ok {
				fullResult = full
			}
			fullResult = sanitizeUTF8(fullResult)

			// Get summary from metadata
			var summary string
			if s, ok := event.Metadata["summary"].(string); ok {
				summary = s
			}
			summary = sanitizeUTF8(summary)

			// Create ToolResult message for server-side tools
			toolResult = &pb.ToolResult{
				CallId:  callID,
				Result:  fullResult,
				Summary: summary,
			}

			slog.InfoContext(ctx, "server-side tool result (sending to client)",
				"tool_name", toolName,
				"call_id", callID,
				"result_length", len(fullResult))
		} else {
			// For client-side tools, TOOL_RESULT is not sent
			// Client already displayed the result locally
			slog.DebugContext(ctx, "client-side tool result (not sent - client handles this)",
				"tool_name", toolName,
				"step", event.Step)
			return nil
		}

	case domain.EventTypeAnswerChunk:
		responseType = pb.ResponseType_RESPONSE_TYPE_ANSWER_CHUNK
		content = sanitizeUTF8(event.Content)

	case domain.EventTypeError:
		responseType = pb.ResponseType_RESPONSE_TYPE_ERROR
		if event.Error != nil {
			content = sanitizeUTF8(event.Error.Message)
			slog.WarnContext(ctx, "sending error event to client",
				"code", event.Error.Code,
				"message", event.Error.Message)
		} else {
			content = sanitizeUTF8(event.Content)
			slog.WarnContext(ctx, "sending error event to client",
				"content", content)
		}

	case domain.EventTypeAgentSpawned, domain.EventTypeAgentCompleted,
		domain.EventTypeAgentFailed:
		// Agent lifecycle events — send as ANSWER_CHUNK with metadata
		// Client identifies them by agentId + content pattern
		responseType = pb.ResponseType_RESPONSE_TYPE_ANSWER_CHUNK

		// Build descriptive content for the agent event
		eventTypeStr := string(event.Type)
		agentID := event.AgentID
		if agentID == "" {
			agentID = "unknown"
		}
		content = fmt.Sprintf("[%s] %s: %s", eventTypeStr, agentID, sanitizeUTF8(event.Content))

		slog.InfoContext(ctx, "agent lifecycle event (sending to client)",
			"event_type", eventTypeStr,
			"agent_id", agentID,
			"content", event.Content)
	}

	// Create response
	var pbError *pb.Error
	if event.Type == domain.EventTypeError && event.Error != nil {
		pbError = &pb.Error{
			Code:    event.Error.Code,
			Message: sanitizeUTF8(event.Error.Message),
		}
	}

	// Determine agent ID: default to "supervisor" if not set
	agentID := event.AgentID
	if agentID == "" {
		agentID = "supervisor"
	}

	resp := &pb.FlowResponse{
		SessionId:  s.sessionID,
		Type:       responseType,
		Content:    content,
		ToolCall:   toolCall,
		ToolResult: toolResult,
		Thought:    thought,
		Reasoning:  reasoning,
		Error:      pbError,
		IsFinal:    event.Type == domain.EventTypeAnswer && event.IsComplete,
		Step:       int32(event.Step),
		AgentId:    agentID,
	}

	// Send response via StreamWriter (thread-safe)
	if err := s.streamWriter.Send(resp); err != nil {
		slog.ErrorContext(ctx, "failed to send agent event", "error", err, "type", event.Type)
		return err
	}

	slog.DebugContext(ctx, "agent event sent", "type", event.Type, "step", event.Step)
	return nil
}

// sanitizeUTF8 removes invalid UTF-8 characters from a string
func sanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	// Convert to valid UTF-8 by replacing invalid sequences
	return string([]rune(s))
}

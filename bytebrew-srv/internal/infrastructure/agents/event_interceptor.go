package agents

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/cloudwego/eino/schema"
)

// AgentEventInterceptor intercepts and logs agent events
type AgentEventInterceptor struct {
	contextLogger *ContextLogger
	eventStream   domain.AgentEventStream
	step          int
}

// NewAgentEventInterceptor creates a new agent event interceptor
func NewAgentEventInterceptor(contextLogger *ContextLogger, eventStream domain.AgentEventStream) *AgentEventInterceptor {
	return &AgentEventInterceptor{
		contextLogger: contextLogger,
		eventStream:   eventStream,
		step:          0,
	}
}

// OnAnswer handles answer events from the agent
func (a *AgentEventInterceptor) OnAnswer(ctx context.Context, answer string) {
	event := &domain.AgentEvent{
		Type:      domain.EventTypeAnswer,
		Timestamp: time.Now(),
		Step:      a.step,
		Content:   answer,
		Metadata:  make(map[string]interface{}),
	}

	a.logAndSendEvent(ctx, event)
}

// OnToolCall handles tool call events from the agent
func (a *AgentEventInterceptor) OnToolCall(ctx context.Context, toolName string, args map[string]interface{}) {
	event := &domain.AgentEvent{
		Type:      domain.EventTypeToolCall,
		Timestamp: time.Now(),
		Step:      a.step,
		Content:   toolName,
		Metadata: map[string]interface{}{
			"tool_name": toolName,
			"args":      args,
		},
	}

	a.logAndSendEvent(ctx, event)
}

// OnToolResult handles tool result events from the agent
func (a *AgentEventInterceptor) OnToolResult(ctx context.Context, toolName string, result string) {
	// Create preview of result (first 500 chars)
	preview := result
	if len(result) > 500 {
		preview = result[:500] + "..."
	}

	event := &domain.AgentEvent{
		Type:      domain.EventTypeToolResult,
		Timestamp: time.Now(),
		Step:      a.step,
		Content:   preview,
		Metadata: map[string]interface{}{
			"tool_name":     toolName,
			"result_length": len(result),
			"preview":       preview,
		},
	}

	a.logAndSendEvent(ctx, event)
}

// OnReasoning handles reasoning content events from models like GLM 4.7
func (a *AgentEventInterceptor) OnReasoning(ctx context.Context, reasoning string) {
	event := &domain.AgentEvent{
		Type:      domain.EventTypeReasoning,
		Timestamp: time.Now(),
		Step:      a.step,
		Content:   reasoning,
		Metadata: map[string]interface{}{
			"reasoning_length": len(reasoning),
		},
	}

	a.logAndSendEvent(ctx, event)
}

// OnAnswerChunk handles streaming answer chunk events
func (a *AgentEventInterceptor) OnAnswerChunk(ctx context.Context, chunk string) {
	event := &domain.AgentEvent{
		Type:      domain.EventTypeAnswerChunk,
		Timestamp: time.Now(),
		Step:      a.step,
		Content:   chunk,
		Metadata:  make(map[string]interface{}),
	}

	a.logAndSendEvent(ctx, event)
}

// IncrementStep increments the step counter
func (a *AgentEventInterceptor) IncrementStep() {
	a.step++
}

// logAndSendEvent logs the event and sends it to the stream
func (a *AgentEventInterceptor) logAndSendEvent(ctx context.Context, event *domain.AgentEvent) {
	// Log the event
	slog.InfoContext(ctx, "agent event",
		"type", event.Type,
		"step", event.Step,
		"content_preview", truncateString(event.Content, 100))

	// Send to stream if available
	if a.eventStream != nil {
		if err := a.eventStream.Send(event); err != nil {
			slog.ErrorContext(ctx, "failed to send agent event", "error", err, "type", event.Type)
		}
	}

	// Save to context logger if available
	if a.contextLogger != nil {
		a.saveEventToContextLog(ctx, event)
	}
}

// saveEventToContextLog saves the event to the context log
func (a *AgentEventInterceptor) saveEventToContextLog(ctx context.Context, event *domain.AgentEvent) {
	// Log the event to context logger
	// In production, this would save to a file in the session directory
	slog.DebugContext(ctx, "agent event saved", "type", event.Type, "step", event.Step)
}

// ParseToolCallsFromMessage extracts tool calls from an assistant message
func ParseToolCallsFromMessage(msg *schema.Message) []schema.ToolCall {
	if msg == nil || len(msg.ToolCalls) == 0 {
		return nil
	}
	return msg.ToolCalls
}

// CreateToolCallMetadata creates metadata for a tool call event
// Handles both OpenAI-style (ID) and Ollama-style (Index) tool calls
func CreateToolCallMetadata(tc *schema.ToolCall) map[string]interface{} {
	return CreateToolCallMetadataWithIndex(tc, 0)
}

// CreateToolCallMetadataWithIndex creates metadata for a tool call event with array position fallback
// Handles OpenAI (ID), Ollama with Index, and Ollama without Index (uses arrayIdx)
func CreateToolCallMetadataWithIndex(tc *schema.ToolCall, arrayIdx int) map[string]interface{} {
	metadata := make(map[string]interface{})
	metadata["type"] = tc.Type

	// Use ID if available (OpenAI), otherwise generate from Index (Ollama),
	// or fall back to array position (Ollama without Index)
	toolCallID := tc.ID
	if toolCallID == "" {
		if tc.Index != nil {
			toolCallID = fmt.Sprintf("call_%d_%s", *tc.Index, tc.Function.Name)
		} else {
			// Ollama doesn't provide ID or Index - use array position
			toolCallID = fmt.Sprintf("call_%d_%s", arrayIdx, tc.Function.Name)
		}
	}
	metadata["id"] = toolCallID

	if tc.Function.Name != "" {
		metadata["function_name"] = tc.Function.Name
		metadata["function_arguments"] = tc.Function.Arguments
	}

	// Also store index for debugging/tracing
	if tc.Index != nil {
		metadata["index"] = *tc.Index
	}
	metadata["array_index"] = arrayIdx

	return metadata
}

package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/cloudwego/eino/schema"
)

// MessageCollector accumulates schema.Message from AgentEvents and saves them to history
type MessageCollector struct {
	sessionID   string
	agentID     string
	historyRepo HistoryRepository
	messages    []*schema.Message
	stepCount   int
	mu          sync.Mutex

	// Track pending tool call for pairing with result
	pendingToolCall *pendingToolCallInfo
}

type pendingToolCallInfo struct {
	toolCallID string
	toolName   string
}

// NewMessageCollector creates a new MessageCollector
func NewMessageCollector(sessionID, agentID string, historyRepo HistoryRepository) *MessageCollector {
	return &MessageCollector{
		sessionID:   sessionID,
		agentID:     agentID,
		historyRepo: historyRepo,
		messages:    make([]*schema.Message, 0),
	}
}

// WrapEventCallback returns a wrapper that intercepts events to collect messages
func (mc *MessageCollector) WrapEventCallback(original func(*domain.AgentEvent) error) func(*domain.AgentEvent) error {
	return func(event *domain.AgentEvent) error {
		// Collect messages from events
		mc.handleEvent(event)

		// Pass through to original callback
		if original != nil {
			return original(event)
		}
		return nil
	}
}

// handleEvent processes events to extract and save messages
func (mc *MessageCollector) handleEvent(event *domain.AgentEvent) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	ctx := context.Background()

	switch event.Type {
	case domain.EventTypeToolCall:
		mc.handleToolCall(ctx, event)
	case domain.EventTypeToolResult:
		mc.handleToolResult(ctx, event)
	case domain.EventTypeAnswer:
		mc.handleAnswer(ctx, event)
	}
}

// handleToolCall creates assistant message with tool call
func (mc *MessageCollector) handleToolCall(ctx context.Context, event *domain.AgentEvent) {
	metadata := event.Metadata
	if metadata == nil {
		slog.WarnContext(ctx, "tool_call event without metadata")
		return
	}

	toolCallID, _ := metadata["id"].(string)
	toolName, _ := metadata["tool_name"].(string)
	argsJSON, _ := metadata["function_arguments"].(string)
	assistantContent, _ := metadata["assistant_content"].(string)

	if toolCallID == "" || toolName == "" {
		slog.WarnContext(ctx, "tool_call event missing required fields",
			"id", toolCallID, "name", toolName)
		return
	}

	// Parse arguments JSON to map for domain.ToolCallInfo
	var argsMap map[string]string
	if argsJSON != "" {
		var rawArgs map[string]interface{}
		if err := json.Unmarshal([]byte(argsJSON), &rawArgs); err == nil {
			argsMap = make(map[string]string)
			for k, v := range rawArgs {
				argsMap[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	// Create assistant message with tool call
	msg := &schema.Message{
		Role:    schema.Assistant,
		Content: assistantContent,
		ToolCalls: []schema.ToolCall{{
			ID: toolCallID,
			Function: schema.FunctionCall{
				Name:      toolName,
				Arguments: argsJSON,
			},
		}},
	}

	mc.messages = append(mc.messages, msg)

	// Save to history
	domainToolCalls := []domain.ToolCallInfo{{
		ID:        toolCallID,
		Name:      toolName,
		Arguments: argsMap,
	}}

	histMsg, err := domain.NewAssistantMessageWithToolCalls(mc.sessionID, assistantContent, domainToolCalls)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create assistant message", "error", err)
		return
	}
	histMsg.AgentID = mc.agentID

	if mc.historyRepo != nil {
		if err := mc.historyRepo.Create(ctx, histMsg); err != nil {
			slog.ErrorContext(ctx, "failed to save tool call message", "error", err)
		}
	}

	// Track pending tool call for result pairing
	mc.pendingToolCall = &pendingToolCallInfo{
		toolCallID: toolCallID,
		toolName:   toolName,
	}

	slog.InfoContext(ctx, "collected tool call message",
		"tool_name", toolName, "id", toolCallID, "agent_id", mc.agentID)
}

// handleToolResult creates tool message
func (mc *MessageCollector) handleToolResult(ctx context.Context, event *domain.AgentEvent) {
	metadata := event.Metadata
	if metadata == nil {
		slog.WarnContext(ctx, "tool_result event without metadata")
		return
	}

	toolName, _ := metadata["tool_name"].(string)
	fullResult, _ := metadata["full_result"].(string)

	// Use full_result if available, otherwise use event.Content
	content := fullResult
	if content == "" {
		content = event.Content
	}

	// Get tool call ID from pending call
	toolCallID := ""
	if mc.pendingToolCall != nil && mc.pendingToolCall.toolName == toolName {
		toolCallID = mc.pendingToolCall.toolCallID
		mc.pendingToolCall = nil // Clear pending
	} else {
		// Fallback: generate ID if not found
		toolCallID = fmt.Sprintf("server-%s-%d", toolName, mc.stepCount)
		slog.WarnContext(ctx, "no pending tool call found, using fallback ID",
			"tool_name", toolName, "id", toolCallID)
	}

	// Create tool message
	msg := &schema.Message{
		Role:       schema.Tool,
		Content:    content,
		ToolCallID: toolCallID,
		Name:       toolName,
	}

	mc.messages = append(mc.messages, msg)

	// Save to history
	histMsg, err := domain.NewToolMessage(mc.sessionID, toolCallID, toolName, content)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create tool message", "error", err)
		return
	}
	histMsg.AgentID = mc.agentID

	if mc.historyRepo != nil {
		if err := mc.historyRepo.Create(ctx, histMsg); err != nil {
			slog.ErrorContext(ctx, "failed to save tool result message", "error", err)
		}
	}

	// Increment step after tool result
	mc.stepCount++

	slog.InfoContext(ctx, "collected tool result message",
		"tool_name", toolName, "step", mc.stepCount, "agent_id", mc.agentID)
}

// handleAnswer creates final assistant message
func (mc *MessageCollector) handleAnswer(ctx context.Context, event *domain.AgentEvent) {
	if event.Content == "" {
		return
	}

	// Create assistant message
	msg := &schema.Message{
		Role:    schema.Assistant,
		Content: event.Content,
	}

	mc.messages = append(mc.messages, msg)

	// Save to history
	histMsg, err := domain.NewMessage(mc.sessionID, domain.MessageTypeAgent, "assistant", event.Content)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create answer message", "error", err)
		return
	}
	histMsg.AgentID = mc.agentID

	if mc.historyRepo != nil {
		if err := mc.historyRepo.Create(ctx, histMsg); err != nil {
			slog.ErrorContext(ctx, "failed to save answer message", "error", err)
		}
	}

	slog.InfoContext(ctx, "collected answer message",
		"length", len(event.Content), "agent_id", mc.agentID)
}

// GetAccumulatedMessages returns all collected messages (thread-safe)
func (mc *MessageCollector) GetAccumulatedMessages() []*schema.Message {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Return copy to prevent concurrent modification
	result := make([]*schema.Message, len(mc.messages))
	copy(result, mc.messages)
	return result
}

// StepCount returns the current step count (thread-safe)
func (mc *MessageCollector) StepCount() int {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	return mc.stepCount
}

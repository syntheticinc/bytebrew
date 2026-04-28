package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// DebugChatModelWrapper wraps a ToolCallingChatModel and logs exact input/output
// This allows verifying what's actually sent to the model
type DebugChatModelWrapper struct {
	inner     model.ToolCallingChatModel
	logDir    string
	sessionID string
	callCount uint64
}

// NewDebugChatModelWrapper creates a wrapper that logs all model interactions
func NewDebugChatModelWrapper(inner model.ToolCallingChatModel, logDir, sessionID string) model.ToolCallingChatModel {
	return &DebugChatModelWrapper{
		inner:     inner,
		logDir:    logDir,
		sessionID: sessionID,
	}
}

// messageToMap converts schema.Message to a map for JSON serialization
func messageToMap(msg *schema.Message) map[string]interface{} {
	result := map[string]interface{}{
		"role":    string(msg.Role),
		"content": msg.Content,
	}

	if msg.Name != "" {
		result["name"] = msg.Name
	}

	if msg.ToolCallID != "" {
		result["tool_call_id"] = msg.ToolCallID
	}

	if len(msg.ToolCalls) > 0 {
		toolCalls := make([]map[string]interface{}, 0, len(msg.ToolCalls))
		for _, tc := range msg.ToolCalls {
			toolCalls = append(toolCalls, map[string]interface{}{
				"id": tc.ID,
				"function": map[string]interface{}{
					"name":      tc.Function.Name,
					"arguments": tc.Function.Arguments,
				},
			})
		}
		result["tool_calls"] = toolCalls
	}

	// Include MultiContent if present
	if len(msg.MultiContent) > 0 {
		result["multi_content_count"] = len(msg.MultiContent)
	}

	return result
}

// logRequest logs the exact request being sent to the model
func (w *DebugChatModelWrapper) logRequest(ctx context.Context, input []*schema.Message, method string) uint64 {
	callNum := atomic.AddUint64(&w.callCount, 1)

	// Convert messages to JSON-serializable format
	messages := make([]map[string]interface{}, 0, len(input))
	for _, msg := range input {
		messages = append(messages, messageToMap(msg))
	}

	logData := map[string]interface{}{
		"timestamp":      time.Now().Format(time.RFC3339Nano),
		"session_id":     w.sessionID,
		"call_number":    callNum,
		"method":         method,
		"message_count":  len(input),
		"messages":       messages,
	}

	// Log to slog for immediate visibility
	slog.InfoContext(ctx, "[DEBUG_WRAPPER] Model request",
		"call_number", callNum,
		"method", method,
		"message_count", len(input))

	// Also save to file for detailed inspection
	if w.logDir != "" {
		w.saveToFile(fmt.Sprintf("request_%d", callNum), logData)
	}

	return callNum
}

// logResponse logs the model's response
func (w *DebugChatModelWrapper) logResponse(ctx context.Context, callNum uint64, response *schema.Message, err error) {
	logData := map[string]interface{}{
		"timestamp":   time.Now().Format(time.RFC3339Nano),
		"session_id":  w.sessionID,
		"call_number": callNum,
	}

	if err != nil {
		logData["error"] = err.Error()
	} else if response != nil {
		logData["response"] = messageToMap(response)
	}

	// Log to slog
	if err != nil {
		slog.ErrorContext(ctx, "[DEBUG_WRAPPER] Model response error",
			"call_number", callNum,
			"error", err)
	} else {
		hasToolCalls := response != nil && len(response.ToolCalls) > 0
		contentLen := 0
		if response != nil {
			contentLen = len(response.Content)
		}
		slog.InfoContext(ctx, "[DEBUG_WRAPPER] Model response",
			"call_number", callNum,
			"content_length", contentLen,
			"has_tool_calls", hasToolCalls)
	}

	// Save to file
	if w.logDir != "" {
		w.saveToFile(fmt.Sprintf("response_%d", callNum), logData)
	}
}

// saveToFile saves log data to JSON file
func (w *DebugChatModelWrapper) saveToFile(name string, data map[string]interface{}) {
	if w.logDir == "" {
		return
	}

	// Create session subdirectory
	sessionDir := filepath.Join(w.logDir, w.sessionID)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		slog.ErrorContext(context.Background(), "failed to create debug log directory", "error", err)
		return
	}

	filename := filepath.Join(sessionDir, fmt.Sprintf("%s.json", name))
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		slog.ErrorContext(context.Background(), "failed to marshal debug log", "error", err)
		return
	}

	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		slog.ErrorContext(context.Background(), "failed to write debug log", "error", err)
	}
}

// Generate implements model.ChatModel
func (w *DebugChatModelWrapper) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	callNum := w.logRequest(ctx, input, "Generate")

	response, err := w.inner.Generate(ctx, input, opts...)

	w.logResponse(ctx, callNum, response, err)

	return response, err
}

// Stream implements model.ChatModel
func (w *DebugChatModelWrapper) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	callNum := w.logRequest(ctx, input, "Stream")

	slog.InfoContext(ctx, "[DEBUG_WRAPPER] Starting stream", "call_number", callNum)

	reader, err := w.inner.Stream(ctx, input, opts...)
	if err != nil {
		w.logResponse(ctx, callNum, nil, err)
		return nil, err
	}

	// Note: For streaming, we can't easily log the full response without consuming it
	// The callback handler already logs streaming events
	slog.InfoContext(ctx, "[DEBUG_WRAPPER] Stream started successfully", "call_number", callNum)

	return reader, nil
}

// WithTools implements model.ToolCallingChatModel
func (w *DebugChatModelWrapper) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	slog.InfoContext(context.Background(), "[DEBUG_WRAPPER] WithTools called", "tool_count", len(tools))
	for i, t := range tools {
		slog.DebugContext(context.Background(), "[DEBUG_WRAPPER] Tool added", "index", i, "name", t.Name)
	}
	newInner, err := w.inner.WithTools(tools)
	if err != nil {
		return nil, err
	}
	return &DebugChatModelWrapper{
		inner:     newInner,
		logDir:    w.logDir,
		sessionID: w.sessionID,
	}, nil
}

// IsCallbacksEnabled forwards the inner model's callback aspect status so eino's
// components.Checker type-assertion succeeds on the wrapper and the framework
// does not auto-inject a duplicate aspect on top of the inner model's manual
// callbacks dispatch (which would emit every streamed chunk twice).
func (w *DebugChatModelWrapper) IsCallbacksEnabled() bool {
	return components.IsCallbacksEnabled(w.inner)
}

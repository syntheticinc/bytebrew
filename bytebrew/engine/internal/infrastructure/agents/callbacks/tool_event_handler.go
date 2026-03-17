package callbacks

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/tools"
	"github.com/cloudwego/eino/callbacks"
	einotool "github.com/cloudwego/eino/components/tool"
)

// ToolCallRecorder defines interface for recording tool calls and results.
// Consumer-side interface: defined here where it's used.
type ToolCallRecorder interface {
	RecordToolCall(sessionID, toolName string)
	RecordToolResult(sessionID, toolName, result string)
}

// ToolEventHandler handles tool start/end callbacks.
type ToolEventHandler struct {
	emitter     *EventEmitter
	counter     *StepCounter
	model       *ModelEventHandler
	planEmitter *PlanProgressEmitter
	recorder    ToolCallRecorder
	sessionID   string
}

// NewToolEventHandler creates a new ToolEventHandler.
func NewToolEventHandler(
	emitter *EventEmitter,
	counter *StepCounter,
	model *ModelEventHandler,
	planEmitter *PlanProgressEmitter,
	recorder ToolCallRecorder,
	sessionID string,
) *ToolEventHandler {
	return &ToolEventHandler{
		emitter:     emitter,
		counter:     counter,
		model:       model,
		planEmitter: planEmitter,
		recorder:    recorder,
		sessionID:   sessionID,
	}
}

// OnToolStart handles tool execution start.
func (h *ToolEventHandler) OnToolStart(ctx context.Context, info *callbacks.RunInfo, input *einotool.CallbackInput) context.Context {
	currentStep := h.counter.GetStep()
	slog.InfoContext(ctx, "[CALLBACK] onToolStart called", "tool_name", info.Name, "step", currentStep)

	// Record tool call for efficiency reminders
	if h.recorder != nil && h.sessionID != "" && info.Name != "" {
		h.recorder.RecordToolCall(h.sessionID, info.Name)
	}

	// Finalize any accumulated text BEFORE tool call so it appears in chat history first
	h.model.FinalizeAccumulatedText(ctx)

	// Generate call ID for server-side tools
	callID := fmt.Sprintf("server-%s-%d", info.Name, currentStep)

	// Emit ToolCall event for the tool that's about to be executed
	metadata := map[string]interface{}{
		"id":        callID,
		"tool_name": info.Name,
	}

	// Add tool input/arguments if available
	if input != nil && input.ArgumentsInJSON != "" {
		metadata["function_arguments"] = input.ArgumentsInJSON
		slog.InfoContext(ctx, "[CALLBACK] onToolStart: got arguments",
			"tool_name", info.Name,
			"arguments_json", input.ArgumentsInJSON)
	} else {
		slog.WarnContext(ctx, "[CALLBACK] onToolStart: NO arguments",
			"tool_name", info.Name,
			"input_nil", input == nil,
			"args_empty", input != nil && input.ArgumentsInJSON == "")
	}

	// Attach pending assistant content from onModelEnd (only for the first tool call)
	if assistantContent := h.counter.ConsumePendingAssistantContent(); assistantContent != "" {
		metadata["assistant_content"] = assistantContent
		slog.InfoContext(ctx, "[CALLBACK] onToolStart: attached assistant_content",
			"tool_name", info.Name,
			"content_length", len(assistantContent))
	}

	event := &domain.AgentEvent{
		Type:      domain.EventTypeToolCall,
		Timestamp: time.Now(),
		Step:      currentStep,
		Content:   info.Name,
		Metadata:  metadata,
	}

	slog.InfoContext(ctx, "[CALLBACK] emitting ToolCall event",
		"tool_name", info.Name,
		"step", currentStep,
		"call_id", callID)

	h.emitter.Emit(ctx, event)

	slog.InfoContext(ctx, "[CALLBACK] onToolStart completed", "tool_name", info.Name)
	return ctx
}

// OnToolEnd handles tool execution result.
func (h *ToolEventHandler) OnToolEnd(ctx context.Context, info *callbacks.RunInfo, output *einotool.CallbackOutput) context.Context {
	currentStep := h.counter.GetStep()
	slog.InfoContext(ctx, "[CALLBACK] onToolEnd called", "tool_name", info.Name, "step", currentStep)

	if output == nil {
		slog.WarnContext(ctx, "[CALLBACK] onToolEnd: output is nil")
		return ctx
	}

	preview := output.Response
	// Don't truncate smart_search results - client needs full citations
	if len(output.Response) > 500 && info.Name != "smart_search" {
		preview = output.Response[:500] + "..."
	}

	slog.InfoContext(ctx, "[CALLBACK] onToolEnd: emitting ToolResult event", "tool_name", info.Name, "result_length", len(output.Response))

	// Store full result in metadata for server-side tools
	// agent_event_stream.go will use this for TOOL_RESULT
	metadata := map[string]interface{}{
		"tool_name":     info.Name,
		"result_length": len(output.Response),
		"full_result":   output.Response, // Full result for client display
	}

	// Compute display summary for client
	summary := tools.SummarizeToolResult(info.Name, output.Response)
	if summary != "" {
		metadata["summary"] = summary
	}

	event := &domain.AgentEvent{
		Type:      domain.EventTypeToolResult,
		Timestamp: time.Now(),
		Step:      currentStep,
		Content:   preview, // Preview for logs
		Metadata:  metadata,
	}

	slog.InfoContext(ctx, "[CALLBACK] emitting ToolResult event",
		"tool_name", info.Name,
		"step", currentStep,
		"expected_call_id", fmt.Sprintf("server-%s-%d", info.Name, currentStep))

	h.emitter.Emit(ctx, event)

	// Record tool result for error loop detection
	if h.recorder != nil && h.sessionID != "" && info.Name != "" {
		h.recorder.RecordToolResult(h.sessionID, info.Name, output.Response)
	}

	// If manage_plan was called, emit plan progress
	if info.Name == "manage_plan" {
		h.planEmitter.EmitPlanProgress(ctx)
	}

	// Increment step after tool execution completes
	// This ensures onToolStart and onToolEnd use the same step number for callId
	h.counter.IncrementStep()
	slog.InfoContext(ctx, "[CALLBACK] onToolEnd completed, step incremented", "tool_name", info.Name, "new_step", h.counter.GetStep())
	return ctx
}

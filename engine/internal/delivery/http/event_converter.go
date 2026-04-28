package http

// SSEEvent represents a converted event ready to be sent over SSE.
type SSEEvent struct {
	Type string // "thinking", "tool_call", "tool_result", "message", "confirmation", "done", "error", "agent_spawn", "agent_result", "user_input_required"
	Data string // JSON payload
}

// domainToSSE maps domain event type strings to SSE event type strings.
var domainToSSE = map[string]string{
	"MessageStarted":         "thinking",
	"StreamingProgress":      "message_delta",
	"MessageCompleted":       "message",
	"ToolExecutionStarted":   "tool_call",
	"ToolExecutionCompleted": "tool_result",
	"ConfirmationRequired":   "confirmation",
	"ProcessingStopped":      "done",
	"Error":                  "error",
	// Agent lifecycle events (emitted by AgentPool on spawn/complete/fail)
	"agent_spawned":   "agent_spawn",
	"agent_completed": "agent_result",
	"agent_failed":    "agent_result",
	// User confirmation prompt for confirm_before tools (reply channel awaits user decision)
	"user_question": "user_input_required",
	// Structured output (summary tables, action buttons)
	"structured_output": "structured_output",
	// Agent lifecycle state transition (AC-STATE-02)
	"state_changed": "agent.state_changed",
}

// ConvertDomainEvent maps a domain event type and payload to an SSEEvent.
// Returns nil if the domain event type is not recognized.
func ConvertDomainEvent(eventType string, payload string) *SSEEvent {
	sseType, ok := domainToSSE[eventType]
	if !ok {
		return nil
	}
	return &SSEEvent{
		Type: sseType,
		Data: payload,
	}
}

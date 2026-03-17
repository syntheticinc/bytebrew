package http

// SSEEvent represents a converted event ready to be sent over SSE.
type SSEEvent struct {
	Type string // "thinking", "tool_call", "tool_result", "message", "confirmation", "done", "error"
	Data string // JSON payload
}

// domainToSSE maps domain event type strings to SSE event type strings.
var domainToSSE = map[string]string{
	"MessageStarted":        "thinking",
	"StreamingProgress":     "message",
	"MessageCompleted":      "message",
	"ToolExecutionStarted":  "tool_call",
	"ToolExecutionCompleted": "tool_result",
	"ConfirmationRequired":  "confirmation",
	"ProcessingStopped":     "done",
	"Error":                 "error",
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

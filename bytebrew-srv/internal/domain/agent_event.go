package domain

import (
	"time"
)

// AgentEventType represents the type of agent event
type AgentEventType string

const (
	EventTypeAnswer         AgentEventType = "answer"          // Standard agent answer
	EventTypeReasoning      AgentEventType = "reasoning"       // Reasoning content (for GLM 4.7, Claude thinking)
	EventTypeToolCall       AgentEventType = "tool_call"       // Tool invocation
	EventTypeToolResult     AgentEventType = "tool_result"     // Tool result
	EventTypeAnswerChunk    AgentEventType = "answer_chunk"    // Streaming answer chunk
	EventTypePlanCreated    AgentEventType = "plan_created"    // Plan created
	EventTypePlanProgress   AgentEventType = "plan_progress"   // Plan progress update
	EventTypePlanCompleted  AgentEventType = "plan_completed"  // Plan completed
	EventTypeError          AgentEventType = "error"           // Agent error (XML parsing, etc.)
	EventTypeAgentSpawned   AgentEventType = "agent_spawned"   // Code Agent spawned
	EventTypeAgentCompleted AgentEventType = "agent_completed" // Code Agent completed
	EventTypeAgentFailed    AgentEventType = "agent_failed"    // Code Agent failed
	EventTypeUserQuestion AgentEventType = "user_question" // ask_user question to client
)

// AgentError represents error information for EventTypeError
type AgentError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// AgentEvent represents an event from the agent execution
type AgentEvent struct {
	Type       AgentEventType         `json:"type"`
	Timestamp  time.Time              `json:"timestamp"`
	Step       int                    `json:"step"`
	Content    string                 `json:"content"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	IsComplete bool                   `json:"is_complete,omitempty"` // For streaming: false during stream, true when complete
	Error      *AgentError            `json:"error,omitempty"`      // Error details for EventTypeError
	AgentID    string                 `json:"agent_id,omitempty"`   // "supervisor" | "code-agent-{uuid[:8]}"
}

// AgentEventStream defines interface for sending agent events
type AgentEventStream interface {
	Send(event *AgentEvent) error
}

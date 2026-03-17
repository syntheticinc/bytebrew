package callbacks

import (
	"context"
	"log/slog"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
)

// EventEmitter sends agent events via callback, automatically setting AgentID.
type EventEmitter struct {
	eventCallback func(event *domain.AgentEvent) error
	agentID       string
}

// NewEventEmitter creates a new EventEmitter.
func NewEventEmitter(cb func(event *domain.AgentEvent) error, agentID string) *EventEmitter {
	return &EventEmitter{
		eventCallback: cb,
		agentID:       agentID,
	}
}

// Emit sends an event via the callback, automatically setting AgentID.
// Logs only important events (skip intermediate reasoning chunks to avoid log spam).
func (e *EventEmitter) Emit(ctx context.Context, event *domain.AgentEvent) {
	if e.eventCallback == nil {
		slog.WarnContext(ctx, "[CALLBACK] emitEvent: eventCallback is nil, skipping", "type", event.Type)
		return
	}

	// Set AgentID on all events
	if event.AgentID == "" {
		event.AgentID = e.agentID
	}

	// Log only important events (skip intermediate reasoning chunks to avoid log spam)
	if event.Type != domain.EventTypeReasoning || event.IsComplete {
		slog.InfoContext(ctx, "[CALLBACK] emitEvent: sending event", "type", event.Type, "step", event.Step, "agent_id", event.AgentID, "content_length", len(event.Content), "is_complete", event.IsComplete)
	}

	if err := e.eventCallback(event); err != nil {
		slog.ErrorContext(ctx, "[CALLBACK] emitEvent: failed to emit agent event", "error", err, "type", event.Type)
	}
}

package orchestrator

import (
	"fmt"
	"sync"
)

// EventType represents the type of orchestrator event
type EventType string

const (
	EventUserMessage    EventType = "user_message"
	EventAgentCompleted EventType = "agent_completed"
	EventAgentFailed    EventType = "agent_failed"
	EventWorkReminder   EventType = "work_reminder"
	EventUserResponded  EventType = "user_responded"
)

// OrchestratorEvent carries data between producers (AgentPool, FlowHandler)
// and the single consumer (SupervisorOrchestrator).
type OrchestratorEvent struct {
	Type      EventType
	Content   string // UserMessage: question. Agent*: result/error.
	AgentID   string // Agent* events
	SubtaskID string // Agent* events
}

// SessionEventBus is a per-session, channel-based event bus.
// Single consumer reads via Events(), multiple producers write via Publish().
// Supports interrupt signalling for cancelling in-progress REACT turns.
type SessionEventBus struct {
	eventCh     chan OrchestratorEvent
	interruptCh chan struct{}
	done        chan struct{}
	once        sync.Once
}

// NewSessionEventBus creates a new event bus with the given buffer size.
func NewSessionEventBus(bufferSize int) *SessionEventBus {
	if bufferSize < 1 {
		bufferSize = 1
	}
	return &SessionEventBus{
		eventCh:     make(chan OrchestratorEvent, bufferSize),
		interruptCh: make(chan struct{}, 1),
		done:        make(chan struct{}),
	}
}

// Publish sends an event to the bus. Non-blocking if buffer is not full.
// Returns error if the bus is closed or the buffer is full.
func (b *SessionEventBus) Publish(event OrchestratorEvent) error {
	select {
	case <-b.done:
		return fmt.Errorf("event bus closed")
	default:
	}

	select {
	case b.eventCh <- event:
		return nil
	case <-b.done:
		return fmt.Errorf("event bus closed")
	default:
		return fmt.Errorf("event bus buffer full")
	}
}

// PublishInterrupt publishes an event and signals the interrupt channel.
// Used for user messages that should cancel the current REACT turn.
// The event is published via Publish(); if that fails, the error is returned
// without signalling interrupt.
func (b *SessionEventBus) PublishInterrupt(event OrchestratorEvent) error {
	if err := b.Publish(event); err != nil {
		return err
	}

	// Signal interrupt (non-blocking — if channel already has a signal, skip)
	select {
	case b.interruptCh <- struct{}{}:
	default:
	}

	return nil
}

// Interrupts returns the read-only interrupt signal channel.
// The orchestrator selects on this to detect mid-turn user messages.
func (b *SessionEventBus) Interrupts() <-chan struct{} {
	return b.interruptCh
}

// DrainInterrupts clears any stale interrupt signals from the channel.
// Called before starting a new turn to avoid reacting to already-processed interrupts.
func (b *SessionEventBus) DrainInterrupts() {
	for {
		select {
		case <-b.interruptCh:
		default:
			return
		}
	}
}

// Events returns the read-only channel for consuming events.
// The consumer uses this in a select{} statement.
func (b *SessionEventBus) Events() <-chan OrchestratorEvent {
	return b.eventCh
}

// Close stops accepting new events and closes the channel.
// Safe to call multiple times.
func (b *SessionEventBus) Close() {
	b.once.Do(func() {
		close(b.done)
		close(b.eventCh)
	})
}

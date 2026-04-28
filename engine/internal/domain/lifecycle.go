package domain

import "fmt"

// LifecycleState represents the lifecycle state of an agent during execution.
type LifecycleState string

const (
	LifecycleInitializing LifecycleState = "initializing"
	LifecycleReady        LifecycleState = "ready"
	LifecycleRunning      LifecycleState = "running"
	LifecycleNeedsInput   LifecycleState = "needs_input"
	LifecycleBlocked      LifecycleState = "blocked"
	LifecycleDegraded     LifecycleState = "degraded"
	LifecycleFinished     LifecycleState = "finished"
)

// AllLifecycleStates returns all valid lifecycle states.
func AllLifecycleStates() []LifecycleState {
	return []LifecycleState{
		LifecycleInitializing,
		LifecycleReady,
		LifecycleRunning,
		LifecycleNeedsInput,
		LifecycleBlocked,
		LifecycleDegraded,
		LifecycleFinished,
	}
}

// IsValid returns true if the lifecycle state is one of the known states.
func (s LifecycleState) IsValid() bool {
	switch s {
	case LifecycleInitializing, LifecycleReady, LifecycleRunning,
		LifecycleNeedsInput, LifecycleBlocked, LifecycleDegraded, LifecycleFinished:
		return true
	}
	return false
}

// validTransitions defines allowed state transitions.
// Key = current state, value = set of allowed next states.
var validTransitions = map[LifecycleState]map[LifecycleState]bool{
	LifecycleInitializing: {
		LifecycleReady:   true,
		LifecycleBlocked: true, // initialization failure
	},
	LifecycleReady: {
		LifecycleRunning:  true,
		LifecycleFinished: true, // shutdown without running
	},
	LifecycleRunning: {
		LifecycleNeedsInput: true,
		LifecycleBlocked:    true,
		LifecycleDegraded:   true,
		LifecycleFinished:   true,
	},
	LifecycleNeedsInput: {
		LifecycleRunning:  true, // user provided input
		LifecycleFinished: true, // timeout / cancel
	},
	LifecycleBlocked: {
		LifecycleRunning:  true, // unblocked
		LifecycleFinished: true, // gave up
	},
	LifecycleDegraded: {
		LifecycleRunning:  true, // recovered
		LifecycleFinished: true,
	},
	LifecycleFinished: {
		// Terminal state for spawn agents.
		// Persistent agents can transition back to ready.
		LifecycleReady: true,
	},
}

// CanTransitionTo returns true if transitioning from current state to next is valid.
func (s LifecycleState) CanTransitionTo(next LifecycleState) bool {
	allowed, ok := validTransitions[s]
	if !ok {
		return false
	}
	return allowed[next]
}

// BlockedReason describes why an agent is in the blocked state.
type BlockedReason struct {
	Code    string // machine-readable: "mcp_connection_failed", "model_unavailable", etc.
	Message string // human-readable description
}

// AgentLifecycle tracks the lifecycle state of a running agent instance.
type AgentLifecycle struct {
	AgentName     string
	SessionID     string
	State         LifecycleState
	BlockedReason *BlockedReason
}

// NewAgentLifecycle creates a new AgentLifecycle in the initializing state.
func NewAgentLifecycle(agentName, sessionID string) *AgentLifecycle {
	return &AgentLifecycle{
		AgentName: agentName,
		SessionID: sessionID,
		State:     LifecycleInitializing,
	}
}

// TransitionTo attempts to transition to a new state.
// Returns an error if the transition is invalid.
func (l *AgentLifecycle) TransitionTo(next LifecycleState) error {
	if !next.IsValid() {
		return fmt.Errorf("invalid lifecycle state: %s", next)
	}
	if !l.State.CanTransitionTo(next) {
		return fmt.Errorf("invalid transition: %s -> %s", l.State, next)
	}
	l.State = next
	if next != LifecycleBlocked {
		l.BlockedReason = nil
	}
	return nil
}

// TransitionToBlocked transitions to blocked state with a reason.
func (l *AgentLifecycle) TransitionToBlocked(reason BlockedReason) error {
	if err := l.TransitionTo(LifecycleBlocked); err != nil {
		return err
	}
	l.BlockedReason = &reason
	return nil
}

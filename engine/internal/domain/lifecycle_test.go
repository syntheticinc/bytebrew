package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLifecycleState_IsValid(t *testing.T) {
	tests := []struct {
		state LifecycleState
		valid bool
	}{
		{LifecycleInitializing, true},
		{LifecycleReady, true},
		{LifecycleRunning, true},
		{LifecycleNeedsInput, true},
		{LifecycleBlocked, true},
		{LifecycleDegraded, true},
		{LifecycleFinished, true},
		{LifecycleState("invalid"), false},
		{LifecycleState(""), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.state.IsValid())
		})
	}
}

func TestLifecycleState_ValidTransitions(t *testing.T) {
	tests := []struct {
		name string
		from LifecycleState
		to   LifecycleState
		ok   bool
	}{
		// initializing transitions
		{"init->ready", LifecycleInitializing, LifecycleReady, true},
		{"init->blocked", LifecycleInitializing, LifecycleBlocked, true},
		{"init->running FAIL", LifecycleInitializing, LifecycleRunning, false},

		// ready transitions
		{"ready->running", LifecycleReady, LifecycleRunning, true},
		{"ready->finished", LifecycleReady, LifecycleFinished, true},
		{"ready->blocked FAIL", LifecycleReady, LifecycleBlocked, false},

		// running transitions
		{"running->needs_input", LifecycleRunning, LifecycleNeedsInput, true},
		{"running->blocked", LifecycleRunning, LifecycleBlocked, true},
		{"running->degraded", LifecycleRunning, LifecycleDegraded, true},
		{"running->finished", LifecycleRunning, LifecycleFinished, true},
		{"running->init FAIL", LifecycleRunning, LifecycleInitializing, false},

		// needs_input transitions
		{"needs_input->running", LifecycleNeedsInput, LifecycleRunning, true},
		{"needs_input->finished", LifecycleNeedsInput, LifecycleFinished, true},
		{"needs_input->blocked FAIL", LifecycleNeedsInput, LifecycleBlocked, false},

		// blocked transitions
		{"blocked->running", LifecycleBlocked, LifecycleRunning, true},
		{"blocked->finished", LifecycleBlocked, LifecycleFinished, true},
		{"blocked->ready FAIL", LifecycleBlocked, LifecycleReady, false},

		// degraded transitions
		{"degraded->running", LifecycleDegraded, LifecycleRunning, true},
		{"degraded->finished", LifecycleDegraded, LifecycleFinished, true},
		{"degraded->blocked FAIL", LifecycleDegraded, LifecycleBlocked, false},

		// finished transitions (persistent agent restart)
		{"finished->ready", LifecycleFinished, LifecycleReady, true},
		{"finished->running FAIL", LifecycleFinished, LifecycleRunning, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.ok, tt.from.CanTransitionTo(tt.to))
		})
	}
}

func TestAgentLifecycle_TransitionTo(t *testing.T) {
	lc := NewAgentLifecycle("test-agent", "session-1")
	assert.Equal(t, LifecycleInitializing, lc.State)

	// Valid transition: initializing -> ready
	require.NoError(t, lc.TransitionTo(LifecycleReady))
	assert.Equal(t, LifecycleReady, lc.State)

	// Valid transition: ready -> running
	require.NoError(t, lc.TransitionTo(LifecycleRunning))
	assert.Equal(t, LifecycleRunning, lc.State)

	// Invalid transition: running -> initializing
	err := lc.TransitionTo(LifecycleInitializing)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transition")
	assert.Equal(t, LifecycleRunning, lc.State) // state unchanged
}

func TestAgentLifecycle_TransitionToBlocked(t *testing.T) {
	lc := NewAgentLifecycle("test-agent", "session-1")
	require.NoError(t, lc.TransitionTo(LifecycleReady))
	require.NoError(t, lc.TransitionTo(LifecycleRunning))

	// AC-STATE-04: blocked contains reason
	reason := BlockedReason{
		Code:    "model_unavailable",
		Message: "Model provider returned 503",
	}
	require.NoError(t, lc.TransitionToBlocked(reason))
	assert.Equal(t, LifecycleBlocked, lc.State)
	require.NotNil(t, lc.BlockedReason)
	assert.Equal(t, "model_unavailable", lc.BlockedReason.Code)
	assert.Equal(t, "Model provider returned 503", lc.BlockedReason.Message)

	// Transition out of blocked clears reason
	require.NoError(t, lc.TransitionTo(LifecycleRunning))
	assert.Nil(t, lc.BlockedReason)
}

func TestAgentLifecycle_InvalidState(t *testing.T) {
	lc := NewAgentLifecycle("test-agent", "session-1")
	err := lc.TransitionTo(LifecycleState("unknown"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid lifecycle state")
}

func TestAllLifecycleStates(t *testing.T) {
	states := AllLifecycleStates()
	assert.Len(t, states, 7)
	for _, s := range states {
		assert.True(t, s.IsValid())
	}
}

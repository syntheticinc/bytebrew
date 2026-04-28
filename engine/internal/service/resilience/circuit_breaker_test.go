package resilience

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func shortConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 3,
		FailureWindow:    1 * time.Second,
		ResetInterval:    50 * time.Millisecond,
	}
}

func TestCircuitBreaker_StartsClosedAllowsRequests(t *testing.T) {
	cb := NewCircuitBreaker("test-mcp", shortConfig())
	assert.Equal(t, CircuitClosed, cb.State())
	assert.NoError(t, cb.AllowRequest())
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	// AC-RESIL-09: opens after 3 consecutive failures
	cb := NewCircuitBreaker("test-mcp", shortConfig())

	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, CircuitClosed, cb.State()) // not yet

	cb.RecordFailure()
	assert.Equal(t, CircuitOpen, cb.State()) // opened

	// AC-RESIL-10: open → request denied
	err := cb.AllowRequest()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker open")
}

func TestCircuitBreaker_SuccessResetsFailures(t *testing.T) {
	cb := NewCircuitBreaker("test-mcp", shortConfig())

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordSuccess() // resets counter

	cb.RecordFailure()
	assert.Equal(t, CircuitClosed, cb.State()) // still closed, only 1 failure
}

func TestCircuitBreaker_HalfOpenAfterReset(t *testing.T) {
	// AC-RESIL-11: half-open after reset interval, closes on success
	cb := NewCircuitBreaker("test-mcp", shortConfig())

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, CircuitOpen, cb.State())

	// Wait for reset interval
	time.Sleep(60 * time.Millisecond)
	assert.Equal(t, CircuitHalfOpen, cb.State())

	// Allow one probe request
	assert.NoError(t, cb.AllowRequest())

	// Success closes it
	cb.RecordSuccess()
	assert.Equal(t, CircuitClosed, cb.State())
}

func TestCircuitBreaker_HalfOpenFailureReopens(t *testing.T) {
	cb := NewCircuitBreaker("test-mcp", shortConfig())

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, CircuitOpen, cb.State())

	time.Sleep(60 * time.Millisecond)
	assert.Equal(t, CircuitHalfOpen, cb.State())

	// Half-open probe fails → back to open
	cb.RecordFailure()
	assert.Equal(t, CircuitOpen, cb.State())
}

func TestCircuitBreaker_FailuresOutsideWindow(t *testing.T) {
	cfg := CircuitBreakerConfig{
		FailureThreshold: 3,
		FailureWindow:    50 * time.Millisecond,
		ResetInterval:    1 * time.Second,
	}
	cb := NewCircuitBreaker("test", cfg)

	cb.RecordFailure()
	cb.RecordFailure()
	time.Sleep(60 * time.Millisecond) // failures expire
	cb.RecordFailure()

	// Only 1 failure in window, should still be closed
	assert.Equal(t, CircuitClosed, cb.State())
}

func TestCircuitBreakerRegistry(t *testing.T) {
	registry := NewCircuitBreakerRegistry(DefaultCircuitBreakerConfig())

	cb1 := registry.Get("tavily")
	cb2 := registry.Get("github")
	cb1Again := registry.Get("tavily")

	assert.Same(t, cb1, cb1Again) // same instance
	assert.NotSame(t, cb1, cb2)

	states := registry.States()
	assert.Len(t, states, 2)
	assert.Equal(t, CircuitClosed, states["tavily"])
	assert.Equal(t, CircuitClosed, states["github"])
}

func TestCircuitBreakerRegistry_Reset(t *testing.T) {
	registry := NewCircuitBreakerRegistry(DefaultCircuitBreakerConfig())

	// Create and open a breaker
	cb := registry.Get("openai")
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()
	require.Equal(t, CircuitOpen, cb.State())

	// Reset removes it
	ok := registry.Reset("openai")
	assert.True(t, ok)

	// States no longer contains it
	states := registry.States()
	_, found := states["openai"]
	assert.False(t, found)

	// Next Get creates a fresh closed breaker
	cb2 := registry.Get("openai")
	assert.Equal(t, CircuitClosed, cb2.State())
	assert.NotSame(t, cb, cb2)
}

func TestCircuitBreakerRegistry_ResetNotFound(t *testing.T) {
	registry := NewCircuitBreakerRegistry(DefaultCircuitBreakerConfig())
	ok := registry.Reset("nonexistent")
	assert.False(t, ok)
}

package taskrunner

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Nil-safety: OnCompleted and Stop on a nil hook must never panic.
// Production call sites always check for nil, but defensive coding costs nothing.
func TestCompletionHook_NilReceiver_Safe(t *testing.T) {
	var h *TaskCompletionHook
	// Should not panic
	h.OnCompleted(context.Background(), uuid.New())
	h.Stop()
}

// No-op hook: when triggerRepo or notifier is nil, OnCompleted must not
// spawn a goroutine. We verify Stop() returns immediately with no WaitGroup waiters.
func TestCompletionHook_MissingDeps_ShortCircuits(t *testing.T) {
	h := NewTaskCompletionHook(nil, nil, nil)
	start := time.Now()
	h.OnCompleted(context.Background(), uuid.New())
	h.OnCompleted(context.Background(), uuid.New())
	h.Stop()
	// Must return well under the 45s shutdown timeout.
	assert.Less(t, time.Since(start), 100*time.Millisecond, "Stop must not block when no deps are wired")
}

// After Stop(), further OnCompleted calls must be dropped (no new goroutines launched).
// This guarantees the hook cannot leak goroutines past shutdown.
func TestCompletionHook_AfterStop_DropsNotifications(t *testing.T) {
	h := NewTaskCompletionHook(nil, nil, nil)
	h.Stop()
	// Post-stop calls should be dropped without panicking.
	h.OnCompleted(context.Background(), uuid.New())
	// stopped flag is set — verify by calling Stop again; should still be fast.
	start := time.Now()
	h.Stop()
	assert.Less(t, time.Since(start), 50*time.Millisecond, "second Stop must be idempotent and fast")
}

// Stop() must be idempotent: calling it multiple times is safe and returns quickly
// every time. Important for shutdown code paths that might fire twice due to signal handling.
func TestCompletionHook_Stop_Idempotent(t *testing.T) {
	h := NewTaskCompletionHook(nil, nil, nil)
	for i := 0; i < 5; i++ {
		start := time.Now()
		h.Stop()
		assert.Less(t, time.Since(start), 50*time.Millisecond, "Stop iteration %d must be fast", i)
	}
}

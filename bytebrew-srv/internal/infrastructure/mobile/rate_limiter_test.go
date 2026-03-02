package mobile

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlidingWindowRateLimiter_UnderLimit(t *testing.T) {
	rl := NewSlidingWindowRateLimiter(3, time.Minute)

	for i := 0; i < 3; i++ {
		err := rl.Allow("192.168.1.1")
		require.NoError(t, err, "request %d should be allowed", i+1)
	}
}

func TestSlidingWindowRateLimiter_OverLimit(t *testing.T) {
	rl := NewSlidingWindowRateLimiter(3, time.Minute)

	for i := 0; i < 3; i++ {
		require.NoError(t, rl.Allow("192.168.1.1"))
	}

	err := rl.Allow("192.168.1.1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate limit exceeded")
}

func TestSlidingWindowRateLimiter_WindowExpiry(t *testing.T) {
	// Use a very short window so we can test expiry
	rl := NewSlidingWindowRateLimiter(2, 50*time.Millisecond)

	require.NoError(t, rl.Allow("key1"))
	require.NoError(t, rl.Allow("key1"))

	// Over limit
	require.Error(t, rl.Allow("key1"))

	// Wait for window to expire
	time.Sleep(60 * time.Millisecond)

	// Should be allowed again
	err := rl.Allow("key1")
	require.NoError(t, err)
}

func TestSlidingWindowRateLimiter_DifferentKeys(t *testing.T) {
	rl := NewSlidingWindowRateLimiter(1, time.Minute)

	require.NoError(t, rl.Allow("ip-a"))
	require.Error(t, rl.Allow("ip-a"))

	// Different key should still be allowed
	require.NoError(t, rl.Allow("ip-b"))
}

func TestSlidingWindowRateLimiter_Concurrent(t *testing.T) {
	rl := NewSlidingWindowRateLimiter(100, time.Minute)

	var wg sync.WaitGroup
	errors := make(chan error, 200)

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := rl.Allow("shared-key"); err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Exactly 100 should succeed, 100 should fail
	errorCount := 0
	for range errors {
		errorCount++
	}
	assert.Equal(t, 100, errorCount, "exactly 100 requests should be rejected")
}

func TestSlidingWindowRateLimiter_Cleanup(t *testing.T) {
	rl := NewSlidingWindowRateLimiter(2, 50*time.Millisecond)

	require.NoError(t, rl.Allow("cleanup-key"))
	require.NoError(t, rl.Allow("cleanup-key"))

	// Wait for entries to expire
	time.Sleep(60 * time.Millisecond)

	// Run cleanup
	rl.cleanup()

	// Key should be removed from the map
	rl.mu.Lock()
	_, exists := rl.attempts["cleanup-key"]
	rl.mu.Unlock()
	assert.False(t, exists, "expired key should be removed after cleanup")
}

func TestSlidingWindowRateLimiter_StartCleanup_StopsOnCancel(t *testing.T) {
	rl := NewSlidingWindowRateLimiter(5, time.Minute)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		rl.StartCleanup(ctx)
		close(done)
	}()

	// Cancel should stop the cleanup goroutine
	cancel()

	select {
	case <-done:
		// Success
	case <-time.After(time.Second):
		t.Fatal("StartCleanup did not stop after context cancellation")
	}
}

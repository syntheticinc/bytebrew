package ratelimit

import (
	"strings"
	"testing"
	"time"
)

func TestTrialLimiter_UnderLimit(t *testing.T) {
	limiter := NewTrialLimiter(5)

	for i := 0; i < 5; i++ {
		if err := limiter.Check("user-1"); err != nil {
			t.Fatalf("step %d: unexpected error: %v", i, err)
		}
	}
}

func TestTrialLimiter_AtLimit(t *testing.T) {
	limiter := NewTrialLimiter(3)

	for i := 0; i < 3; i++ {
		if err := limiter.Check("user-1"); err != nil {
			t.Fatalf("step %d: unexpected error: %v", i, err)
		}
	}

	err := limiter.Check("user-1")
	if err == nil {
		t.Fatal("expected error when at limit, got nil")
	}
	if !strings.Contains(err.Error(), "rate limit exceeded") {
		t.Errorf("error = %q, want containing %q", err.Error(), "rate limit exceeded")
	}
}

func TestTrialLimiter_WindowExpiry(t *testing.T) {
	limiter := &TrialLimiter{
		windows: make(map[string][]time.Time),
		limit:   2,
		window:  100 * time.Millisecond,
	}

	// Fill to limit
	if err := limiter.Check("user-1"); err != nil {
		t.Fatalf("step 1: unexpected error: %v", err)
	}
	if err := limiter.Check("user-1"); err != nil {
		t.Fatalf("step 2: unexpected error: %v", err)
	}

	// Should be blocked
	if err := limiter.Check("user-1"); err == nil {
		t.Fatal("expected rate limit error, got nil")
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	if err := limiter.Check("user-1"); err != nil {
		t.Fatalf("after window expiry: unexpected error: %v", err)
	}
}

func TestTrialLimiter_IndependentUsers(t *testing.T) {
	limiter := NewTrialLimiter(1)

	if err := limiter.Check("user-1"); err != nil {
		t.Fatalf("user-1: unexpected error: %v", err)
	}

	// user-1 is at limit
	if err := limiter.Check("user-1"); err == nil {
		t.Fatal("user-1: expected rate limit error, got nil")
	}

	// user-2 should be independent
	if err := limiter.Check("user-2"); err != nil {
		t.Fatalf("user-2: unexpected error: %v", err)
	}
}

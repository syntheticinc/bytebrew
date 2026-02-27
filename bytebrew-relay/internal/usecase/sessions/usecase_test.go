package sessions

import (
	"fmt"
	"testing"
	"time"
)

func TestRegister_Success(t *testing.T) {
	uc := New(5 * time.Minute)

	err := uc.Register("user-1", "session-1", "personal", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	count := uc.ActiveCount("user-1")
	if count != 1 {
		t.Fatalf("expected 1 active session, got %d", count)
	}
}

func TestRegister_SeatLimit_Personal(t *testing.T) {
	uc := New(5 * time.Minute)

	// First session succeeds
	if err := uc.Register("user-1", "session-1", "personal", 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second session fails (personal = 1 seat)
	err := uc.Register("user-1", "session-2", "personal", 1)
	if err == nil {
		t.Fatal("expected error for seat limit")
	}
}

func TestRegister_SeatLimit_Teams(t *testing.T) {
	uc := New(5 * time.Minute)

	// Register 3 sessions (teams with 3 seats)
	for i := 0; i < 3; i++ {
		err := uc.Register("user-1", fmt.Sprintf("session-%d", i), "teams", 3)
		if err != nil {
			t.Fatalf("unexpected error on session %d: %v", i, err)
		}
	}

	// 4th session fails
	err := uc.Register("user-1", "session-3", "teams", 3)
	if err == nil {
		t.Fatal("expected error for seat limit")
	}

	if uc.ActiveCount("user-1") != 3 {
		t.Fatalf("expected 3 active sessions, got %d", uc.ActiveCount("user-1"))
	}
}

func TestRegister_DifferentUsers(t *testing.T) {
	uc := New(5 * time.Minute)

	// Each user gets their own seat
	if err := uc.Register("user-1", "session-1", "personal", 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := uc.Register("user-2", "session-2", "personal", 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if uc.ActiveCount("user-1") != 1 {
		t.Fatalf("expected 1 session for user-1, got %d", uc.ActiveCount("user-1"))
	}
	if uc.ActiveCount("user-2") != 1 {
		t.Fatalf("expected 1 session for user-2, got %d", uc.ActiveCount("user-2"))
	}
}

func TestHeartbeat_Success(t *testing.T) {
	uc := New(5 * time.Minute)

	if err := uc.Register("user-1", "session-1", "personal", 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := uc.Heartbeat("session-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHeartbeat_NotFound(t *testing.T) {
	uc := New(5 * time.Minute)

	err := uc.Heartbeat("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestRelease_Success(t *testing.T) {
	uc := New(5 * time.Minute)

	if err := uc.Register("user-1", "session-1", "personal", 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := uc.Release("session-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if uc.ActiveCount("user-1") != 0 {
		t.Fatalf("expected 0 sessions after release, got %d", uc.ActiveCount("user-1"))
	}
}

func TestRelease_NotFound(t *testing.T) {
	uc := New(5 * time.Minute)

	err := uc.Release("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestRelease_FreesSlot(t *testing.T) {
	uc := New(5 * time.Minute)

	// Fill slot
	if err := uc.Register("user-1", "session-1", "personal", 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Release slot
	if err := uc.Release("session-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// New session should succeed
	if err := uc.Register("user-1", "session-2", "personal", 1); err != nil {
		t.Fatalf("unexpected error after release: %v", err)
	}
}

func TestCleanExpired(t *testing.T) {
	timeout := 5 * time.Minute
	uc := New(timeout)

	// Override nowFunc to control time
	now := time.Now()
	uc.nowFunc = func() time.Time { return now }

	// Register session
	if err := uc.Register("user-1", "session-1", "personal", 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Move time forward past timeout
	now = now.Add(6 * time.Minute)

	uc.CleanExpired()

	if uc.ActiveCount("user-1") != 0 {
		t.Fatalf("expected 0 sessions after cleanup, got %d", uc.ActiveCount("user-1"))
	}
}

func TestCleanExpired_KeepsFresh(t *testing.T) {
	timeout := 5 * time.Minute
	uc := New(timeout)

	now := time.Now()
	uc.nowFunc = func() time.Time { return now }

	// Register session
	if err := uc.Register("user-1", "session-1", "personal", 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Move time forward but within timeout
	now = now.Add(3 * time.Minute)

	uc.CleanExpired()

	if uc.ActiveCount("user-1") != 1 {
		t.Fatalf("expected 1 session (still fresh), got %d", uc.ActiveCount("user-1"))
	}
}

func TestCleanExpired_FreesSlot(t *testing.T) {
	timeout := 5 * time.Minute
	uc := New(timeout)

	now := time.Now()
	uc.nowFunc = func() time.Time { return now }

	// Fill seat
	if err := uc.Register("user-1", "session-1", "personal", 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Move time past timeout
	now = now.Add(6 * time.Minute)

	// Register should succeed because CleanExpired runs inside Register
	if err := uc.Register("user-1", "session-2", "personal", 1); err != nil {
		t.Fatalf("expected success after expired session cleanup: %v", err)
	}

	if uc.TotalActive() != 1 {
		t.Fatalf("expected 1 total session, got %d", uc.TotalActive())
	}
}

func TestTotalActive(t *testing.T) {
	uc := New(5 * time.Minute)

	if uc.TotalActive() != 0 {
		t.Fatalf("expected 0 initially, got %d", uc.TotalActive())
	}

	if err := uc.Register("user-1", "s1", "teams", 5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := uc.Register("user-1", "s2", "teams", 5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := uc.Register("user-2", "s3", "teams", 5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if uc.TotalActive() != 3 {
		t.Fatalf("expected 3 total, got %d", uc.TotalActive())
	}
}

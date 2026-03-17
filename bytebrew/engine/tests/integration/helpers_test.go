//go:build integration

package integration

import (
	"testing"
	"time"
)

// waitForCondition polls check every 100ms until it returns true or timeout expires.
func waitForCondition(t *testing.T, timeout time.Duration, check func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("condition not met within %v", timeout)
}

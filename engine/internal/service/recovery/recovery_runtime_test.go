package recovery

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// TestRecoveryExecutor_MCPConnectionFailed verifies that mcp_connection_failed
// triggers a retry recovery action.
func TestRecoveryExecutor_MCPConnectionFailed(t *testing.T) {
	exec := New(nil)

	result := exec.Execute(context.Background(), "session-1",
		domain.FailureMCPConnectionFailed, "server unreachable")

	assert.True(t, result.Recovered)
	assert.Equal(t, domain.RecoveryRetry, result.Action)
}

// TestRecoveryExecutor_UnknownFailure verifies that an unrecognized failure type
// results in no recovery (Recovered=false).
func TestRecoveryExecutor_UnknownFailure(t *testing.T) {
	exec := New(nil)

	result := exec.Execute(context.Background(), "session-1",
		domain.FailureType("unknown_xyz"), "something unexpected")

	assert.False(t, result.Recovered)
	assert.Equal(t, domain.RecoveryBlock, result.Action)
	assert.Equal(t, domain.EscalationAlertHuman, result.Escalation)
}

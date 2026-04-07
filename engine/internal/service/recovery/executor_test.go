package recovery

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

type mockEventRecorder struct {
	events []domain.RecoveryEvent
}

func (m *mockEventRecorder) RecordRecoveryEvent(ctx context.Context, sessionID string, event domain.RecoveryEvent) {
	m.events = append(m.events, event)
}

func TestExecutor_MCPConnectionFailed(t *testing.T) {
	recorder := &mockEventRecorder{}
	exec := New(recorder)

	result := exec.Execute(context.Background(), "session-1",
		domain.FailureMCPConnectionFailed, "tavily server unreachable")

	assert.True(t, result.Recovered)
	assert.Equal(t, domain.RecoveryRetry, result.Action)
	assert.Equal(t, domain.EscalationLogAndContinue, result.Escalation)
	// AC-REC-04: events recorded
	assert.GreaterOrEqual(t, len(recorder.events), 1)
}

func TestExecutor_ModelUnavailable(t *testing.T) {
	recorder := &mockEventRecorder{}
	exec := New(recorder)

	result := exec.Execute(context.Background(), "session-1",
		domain.FailureModelUnavailable, "provider returned 503")

	assert.True(t, result.Recovered)
	assert.Equal(t, domain.RecoveryRetry, result.Action)
	assert.Equal(t, domain.EscalationAlertHuman, result.Escalation)
}

func TestExecutor_ToolAuthFailure_NoRetry(t *testing.T) {
	recorder := &mockEventRecorder{}
	exec := New(recorder)

	// tool_auth_failure: no retry, block immediately
	result := exec.Execute(context.Background(), "session-1",
		domain.FailureToolAuthFailure, "invalid API key")

	assert.False(t, result.Recovered)
	assert.Equal(t, domain.RecoveryBlock, result.Action)
	assert.Equal(t, domain.EscalationAlertHuman, result.Escalation)
	// AC-REC-04: event recorded even for no-retry
	assert.Len(t, recorder.events, 1)
}

func TestExecutor_ToolTimeout(t *testing.T) {
	recorder := &mockEventRecorder{}
	exec := New(recorder)

	result := exec.Execute(context.Background(), "session-1",
		domain.FailureToolTimeout, "execute_command timed out")

	assert.True(t, result.Recovered)
	assert.Equal(t, domain.RecoveryRetry, result.Action)
	assert.Equal(t, domain.EscalationLogAndContinue, result.Escalation)
}

func TestExecutor_ContextOverflow(t *testing.T) {
	recorder := &mockEventRecorder{}
	exec := New(recorder)

	result := exec.Execute(context.Background(), "session-1",
		domain.FailureContextOverflow, "context exceeded 128k tokens")

	assert.True(t, result.Recovered)
	assert.Equal(t, domain.RecoveryCompact, result.Action)
	assert.Equal(t, domain.EscalationAbort, result.Escalation)
}

func TestExecutor_UnknownFailureType(t *testing.T) {
	recorder := &mockEventRecorder{}
	exec := New(recorder)

	result := exec.Execute(context.Background(), "session-1",
		domain.FailureType("unknown_error"), "something weird happened")

	assert.False(t, result.Recovered)
	assert.Equal(t, domain.RecoveryBlock, result.Action)
}

func TestExecutor_GetRecipe(t *testing.T) {
	exec := New(nil)

	recipe, ok := exec.GetRecipe(domain.FailureMCPConnectionFailed)
	require.True(t, ok)
	assert.Equal(t, domain.RecoveryRetry, recipe.Action)
	assert.Equal(t, 1, recipe.RetryCount)

	_, ok = exec.GetRecipe(domain.FailureType("unknown"))
	assert.False(t, ok)
}

func TestExecutor_CustomRecipes(t *testing.T) {
	custom := map[domain.FailureType]*domain.RecoveryRecipe{
		domain.FailureToolTimeout: {
			FailureType:   domain.FailureToolTimeout,
			Action:        domain.RecoverySkip,
			RetryCount:    0,
			Escalation:    domain.EscalationLogAndContinue,
		},
	}

	exec := NewWithRecipes(custom, nil)

	recipe, ok := exec.GetRecipe(domain.FailureToolTimeout)
	require.True(t, ok)
	assert.Equal(t, domain.RecoverySkip, recipe.Action)

	// MCP recipe not present in custom set
	_, ok = exec.GetRecipe(domain.FailureMCPConnectionFailed)
	assert.False(t, ok)
}

func TestExecutor_NilRecorder(t *testing.T) {
	// Should not panic with nil recorder
	exec := New(nil)
	result := exec.Execute(context.Background(), "session-1",
		domain.FailureToolTimeout, "timeout")
	assert.True(t, result.Recovered)
}

func TestExecutor_BackoffCalculation(t *testing.T) {
	exec := New(nil)

	// Fixed backoff
	fixedRecipe := &domain.RecoveryRecipe{
		Backoff:       domain.BackoffFixed,
		BackoffBaseMs: 1000,
	}
	assert.Equal(t, int64(1000), exec.calculateBackoff(fixedRecipe, 1).Milliseconds())
	assert.Equal(t, int64(1000), exec.calculateBackoff(fixedRecipe, 3).Milliseconds())

	// Exponential backoff
	expRecipe := &domain.RecoveryRecipe{
		Backoff:       domain.BackoffExponential,
		BackoffBaseMs: 1000,
	}
	assert.Equal(t, int64(1000), exec.calculateBackoff(expRecipe, 1).Milliseconds())
	assert.Equal(t, int64(2000), exec.calculateBackoff(expRecipe, 2).Milliseconds())
	assert.Equal(t, int64(4000), exec.calculateBackoff(expRecipe, 3).Milliseconds())

	// Zero base
	zeroRecipe := &domain.RecoveryRecipe{
		Backoff:       domain.BackoffFixed,
		BackoffBaseMs: 0,
	}
	assert.Equal(t, int64(0), exec.calculateBackoff(zeroRecipe, 1).Milliseconds())
}

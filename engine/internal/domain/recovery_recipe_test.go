package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFailureType_IsValid(t *testing.T) {
	tests := []struct {
		ft    FailureType
		valid bool
	}{
		{FailureMCPConnectionFailed, true},
		{FailureModelUnavailable, true},
		{FailureToolTimeout, true},
		{FailureToolAuthFailure, true},
		{FailureContextOverflow, true},
		{FailureType("unknown"), false},
		{FailureType(""), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.ft), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.ft.IsValid())
		})
	}
}

func TestRecoveryRecipe_Validate(t *testing.T) {
	tests := []struct {
		name    string
		recipe  RecoveryRecipe
		wantErr bool
	}{
		{"valid", RecoveryRecipe{FailureType: FailureToolTimeout, RetryCount: 1}, false},
		{"invalid failure type", RecoveryRecipe{FailureType: "bogus"}, true},
		{"negative retry", RecoveryRecipe{FailureType: FailureToolTimeout, RetryCount: -1}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.recipe.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestDefaultRecoveryRecipes(t *testing.T) {
	recipes := DefaultRecoveryRecipes()
	assert.Len(t, recipes, 5)

	// mcp_connection_failed: retry 1x then degrade (log_and_continue)
	mcp := recipes[FailureMCPConnectionFailed]
	require.NotNil(t, mcp)
	assert.Equal(t, RecoveryRetry, mcp.Action)
	assert.Equal(t, 1, mcp.RetryCount)
	assert.Equal(t, EscalationLogAndContinue, mcp.Escalation)

	// model_unavailable: retry with exponential backoff then alert human
	model := recipes[FailureModelUnavailable]
	require.NotNil(t, model)
	assert.Equal(t, RecoveryRetry, model.Action)
	assert.Equal(t, BackoffExponential, model.Backoff)
	assert.Equal(t, EscalationAlertHuman, model.Escalation)

	// tool_auth_failure: no retry, block immediately
	auth := recipes[FailureToolAuthFailure]
	require.NotNil(t, auth)
	assert.Equal(t, RecoveryBlock, auth.Action)
	assert.Equal(t, 0, auth.RetryCount)

	// context_overflow: compact and retry
	ctx := recipes[FailureContextOverflow]
	require.NotNil(t, ctx)
	assert.Equal(t, RecoveryCompact, ctx.Action)
	assert.Equal(t, 1, ctx.RetryCount)
}

func TestSessionDegradeState(t *testing.T) {
	// AC-REC-01: Degrade scope per-session
	state := NewSessionDegradeState()

	assert.False(t, state.IsMCPDegraded("tavily"))
	assert.False(t, state.IsToolDegraded("web_search"))

	state.DegradeMCP("tavily")
	state.DegradeTool("web_search")

	assert.True(t, state.IsMCPDegraded("tavily"))
	assert.True(t, state.IsToolDegraded("web_search"))
	assert.False(t, state.IsMCPDegraded("github"))

	// AC-REC-02: New session starts fresh
	fresh := NewSessionDegradeState()
	assert.False(t, fresh.IsMCPDegraded("tavily"))
	assert.False(t, fresh.IsToolDegraded("web_search"))
}

func TestAllFailureTypes(t *testing.T) {
	types := AllFailureTypes()
	assert.Len(t, types, 5)
	for _, ft := range types {
		assert.True(t, ft.IsValid())
	}
}

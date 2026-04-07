package domain

import "fmt"

// FailureType represents a type of failure that can occur during agent execution.
type FailureType string

const (
	FailureMCPConnectionFailed FailureType = "mcp_connection_failed"
	FailureModelUnavailable    FailureType = "model_unavailable"
	FailureToolTimeout         FailureType = "tool_timeout"
	FailureToolAuthFailure     FailureType = "tool_auth_failure"
	FailureContextOverflow     FailureType = "context_overflow"
)

// AllFailureTypes returns all defined failure types.
func AllFailureTypes() []FailureType {
	return []FailureType{
		FailureMCPConnectionFailed,
		FailureModelUnavailable,
		FailureToolTimeout,
		FailureToolAuthFailure,
		FailureContextOverflow,
	}
}

// IsValid returns true if the failure type is recognized.
func (ft FailureType) IsValid() bool {
	switch ft {
	case FailureMCPConnectionFailed, FailureModelUnavailable,
		FailureToolTimeout, FailureToolAuthFailure, FailureContextOverflow:
		return true
	}
	return false
}

// RecoveryAction is the action to take when recovering from a failure.
type RecoveryAction string

const (
	RecoveryRetry    RecoveryAction = "retry"
	RecoveryFallback RecoveryAction = "fallback"
	RecoveryDegrade  RecoveryAction = "degrade"
	RecoveryBlock    RecoveryAction = "block"
	RecoverySkip     RecoveryAction = "skip"
	RecoveryCompact  RecoveryAction = "compact"
)

// BackoffStrategy defines how retry delays are calculated.
type BackoffStrategy string

const (
	BackoffFixed       BackoffStrategy = "fixed"
	BackoffExponential BackoffStrategy = "exponential"
)

// EscalationAction defines what happens after recovery fails.
type EscalationAction string

const (
	EscalationAlertHuman      EscalationAction = "alert_human"
	EscalationLogAndContinue  EscalationAction = "log_and_continue"
	EscalationAbort           EscalationAction = "abort"
)

// RecoveryRecipe defines how to recover from a specific failure type.
type RecoveryRecipe struct {
	FailureType    FailureType
	Action         RecoveryAction
	RetryCount     int             // max retries (0 = no retry)
	Backoff        BackoffStrategy
	BackoffBaseMs  int             // base delay in ms for backoff
	FallbackModel  string          // model name to fall back to (for model_unavailable)
	Escalation     EscalationAction
}

// Validate validates the RecoveryRecipe.
func (r *RecoveryRecipe) Validate() error {
	if !r.FailureType.IsValid() {
		return fmt.Errorf("invalid failure type: %s", r.FailureType)
	}
	if r.RetryCount < 0 {
		return fmt.Errorf("retry_count must be >= 0")
	}
	return nil
}

// DefaultRecoveryRecipes returns the built-in recovery recipes per PRD §8.4.
func DefaultRecoveryRecipes() map[FailureType]*RecoveryRecipe {
	return map[FailureType]*RecoveryRecipe{
		FailureMCPConnectionFailed: {
			FailureType:   FailureMCPConnectionFailed,
			Action:        RecoveryRetry,
			RetryCount:    1,
			Backoff:       BackoffFixed,
			BackoffBaseMs: 1000,
			Escalation:    EscalationLogAndContinue, // degrade mode
		},
		FailureModelUnavailable: {
			FailureType:   FailureModelUnavailable,
			Action:        RecoveryRetry,
			RetryCount:    1,
			Backoff:       BackoffExponential,
			BackoffBaseMs: 2000,
			Escalation:    EscalationAlertHuman, // block
		},
		FailureToolTimeout: {
			FailureType:   FailureToolTimeout,
			Action:        RecoveryRetry,
			RetryCount:    1,
			Backoff:       BackoffFixed,
			BackoffBaseMs: 500,
			Escalation:    EscalationLogAndContinue, // skip tool
		},
		FailureToolAuthFailure: {
			FailureType:   FailureToolAuthFailure,
			Action:        RecoveryBlock,
			RetryCount:    0, // no retry
			Escalation:    EscalationAlertHuman,
		},
		FailureContextOverflow: {
			FailureType:   FailureContextOverflow,
			Action:        RecoveryCompact,
			RetryCount:    1,
			Backoff:       BackoffFixed,
			BackoffBaseMs: 0,
			Escalation:    EscalationAbort,
		},
	}
}

// RecoveryEvent records a recovery attempt for agent inspection (AC-REC-04).
type RecoveryEvent struct {
	FailureType FailureType
	Action      RecoveryAction
	Attempt     int    // 1-based attempt number
	Success     bool
	Detail      string // human-readable description
}

// SessionDegradeState tracks degraded components within a session (AC-REC-01).
// Reset on new session (AC-REC-02).
type SessionDegradeState struct {
	DegradedMCPServers map[string]bool // MCP server name → degraded
	DegradedTools      map[string]bool // tool name → degraded
}

// NewSessionDegradeState creates a fresh degrade state for a new session.
func NewSessionDegradeState() *SessionDegradeState {
	return &SessionDegradeState{
		DegradedMCPServers: make(map[string]bool),
		DegradedTools:      make(map[string]bool),
	}
}

// DegradeMCP marks an MCP server as degraded for this session.
func (s *SessionDegradeState) DegradeMCP(serverName string) {
	s.DegradedMCPServers[serverName] = true
}

// IsMCPDegraded returns true if the MCP server is degraded.
func (s *SessionDegradeState) IsMCPDegraded(serverName string) bool {
	return s.DegradedMCPServers[serverName]
}

// DegradeTool marks a tool as degraded for this session.
func (s *SessionDegradeState) DegradeTool(toolName string) {
	s.DegradedTools[toolName] = true
}

// IsToolDegraded returns true if the tool is degraded.
func (s *SessionDegradeState) IsToolDegraded(toolName string) bool {
	return s.DegradedTools[toolName]
}

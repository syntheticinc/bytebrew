package recovery

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// EventRecorder records recovery events for agent inspection (AC-REC-04).
type EventRecorder interface {
	RecordRecoveryEvent(ctx context.Context, sessionID string, event domain.RecoveryEvent)
}

// Executor executes recovery recipes for failures.
// It applies 1 automatic recovery attempt before escalation (AC-REC-03).
type Executor struct {
	recipes  map[domain.FailureType]*domain.RecoveryRecipe
	recorder EventRecorder
}

// New creates a new recovery executor with default recipes.
func New(recorder EventRecorder) *Executor {
	return &Executor{
		recipes:  domain.DefaultRecoveryRecipes(),
		recorder: recorder,
	}
}

// NewWithRecipes creates a new recovery executor with custom recipes.
func NewWithRecipes(recipes map[domain.FailureType]*domain.RecoveryRecipe, recorder EventRecorder) *Executor {
	return &Executor{
		recipes:  recipes,
		recorder: recorder,
	}
}

// RecoveryResult contains the outcome of a recovery attempt.
type RecoveryResult struct {
	Recovered  bool
	Action     domain.RecoveryAction
	Escalation domain.EscalationAction
	Detail     string
}

// Execute attempts recovery for the given failure type.
// Returns a result indicating whether recovery succeeded and what action was taken.
func (e *Executor) Execute(ctx context.Context, sessionID string, failureType domain.FailureType, detail string) RecoveryResult {
	recipe, ok := e.recipes[failureType]
	if !ok {
		slog.WarnContext(ctx, "[Recovery] no recipe for failure type", "failure_type", failureType)
		return RecoveryResult{
			Recovered:  false,
			Action:     domain.RecoveryBlock,
			Escalation: domain.EscalationAlertHuman,
			Detail:     fmt.Sprintf("no recovery recipe for %s", failureType),
		}
	}

	slog.InfoContext(ctx, "[Recovery] executing recipe",
		"failure_type", failureType, "action", recipe.Action,
		"retry_count", recipe.RetryCount, "session_id", sessionID)

	// No retry actions — immediate result
	if recipe.Action == domain.RecoveryBlock {
		event := domain.RecoveryEvent{
			FailureType: failureType,
			Action:      recipe.Action,
			Attempt:     0,
			Success:     false,
			Detail:      detail,
		}
		e.recordEvent(ctx, sessionID, event)

		return RecoveryResult{
			Recovered:  false,
			Action:     recipe.Action,
			Escalation: recipe.Escalation,
			Detail:     detail,
		}
	}

	// Attempt recovery (AC-REC-03: 1 auto attempt before escalation)
	maxAttempts := recipe.RetryCount
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Apply backoff delay before retry (not on first attempt)
		if attempt > 1 {
			delay := e.calculateBackoff(recipe, attempt)
			slog.DebugContext(ctx, "[Recovery] backoff delay",
				"attempt", attempt, "delay_ms", delay.Milliseconds())

			select {
			case <-ctx.Done():
				return RecoveryResult{
					Recovered:  false,
					Action:     recipe.Action,
					Escalation: recipe.Escalation,
					Detail:     "recovery cancelled",
				}
			case <-time.After(delay):
			}
		}

		event := domain.RecoveryEvent{
			FailureType: failureType,
			Action:      recipe.Action,
			Attempt:     attempt,
			Success:     true, // optimistic — caller determines actual success
			Detail:      fmt.Sprintf("attempt %d/%d: %s", attempt, maxAttempts, detail),
		}
		e.recordEvent(ctx, sessionID, event)
	}

	return RecoveryResult{
		Recovered:  true,
		Action:     recipe.Action,
		Escalation: recipe.Escalation,
		Detail:     fmt.Sprintf("recovery attempted: %s", recipe.Action),
	}
}

// GetRecipe returns the recipe for a failure type.
func (e *Executor) GetRecipe(failureType domain.FailureType) (*domain.RecoveryRecipe, bool) {
	recipe, ok := e.recipes[failureType]
	return recipe, ok
}

// calculateBackoff computes the delay for the given attempt.
func (e *Executor) calculateBackoff(recipe *domain.RecoveryRecipe, attempt int) time.Duration {
	baseMs := recipe.BackoffBaseMs
	if baseMs <= 0 {
		return 0
	}

	switch recipe.Backoff {
	case domain.BackoffExponential:
		// 2^(attempt-1) * base
		multiplier := 1 << (attempt - 1)
		return time.Duration(baseMs*multiplier) * time.Millisecond
	default: // fixed
		return time.Duration(baseMs) * time.Millisecond
	}
}

func (e *Executor) recordEvent(ctx context.Context, sessionID string, event domain.RecoveryEvent) {
	if e.recorder == nil {
		return
	}
	e.recorder.RecordRecoveryEvent(ctx, sessionID, event)
}

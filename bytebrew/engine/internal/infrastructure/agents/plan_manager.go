package agents

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/google/uuid"
)

// PlanStorage defines the interface for persisting plans
type PlanStorage interface {
	Save(ctx context.Context, plan *domain.Plan) error
	GetBySessionID(ctx context.Context, sessionID string) (*domain.Plan, error)
	Update(ctx context.Context, plan *domain.Plan) error
}

// PlanManager handles plan lifecycle and caching
type PlanManager struct {
	storage PlanStorage
	cache   map[string]*domain.Plan // sessionID -> Plan
	mu      sync.RWMutex
}

// NewPlanManager creates a new PlanManager instance
func NewPlanManager(storage PlanStorage) *PlanManager {
	return &PlanManager{
		storage: storage,
		cache:   make(map[string]*domain.Plan),
	}
}

// CreatePlan creates a new plan and persists it
func (pm *PlanManager) CreatePlan(ctx context.Context, sessionID, goal string, steps []*domain.PlanStep) (*domain.Plan, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("sessionID is required")
	}

	if goal == "" {
		return nil, fmt.Errorf("goal is required")
	}

	if len(steps) == 0 {
		return nil, fmt.Errorf("at least one step is required")
	}

	// Generate unique ID
	planID := uuid.New().String()

	// Create plan (first step auto-marked as in_progress)
	plan := domain.NewPlan(planID, sessionID, goal, steps)

	// Persist to storage
	if pm.storage != nil {
		if err := pm.storage.Save(ctx, plan); err != nil {
			slog.ErrorContext(ctx, "failed to save plan to storage",
				"plan_id", planID,
				"session_id", sessionID,
				"error", err)
			return nil, fmt.Errorf("failed to save plan: %w", err)
		}
	}

	// Cache in memory
	pm.mu.Lock()
	pm.cache[sessionID] = plan
	pm.mu.Unlock()

	slog.InfoContext(ctx, "plan created",
		"plan_id", planID,
		"session_id", sessionID,
		"goal", goal,
		"steps", len(steps))

	return plan, nil
}

// GetActivePlan retrieves the active plan for a session
func (pm *PlanManager) GetActivePlan(ctx context.Context, sessionID string) (*domain.Plan, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("sessionID is required")
	}

	// Check cache first
	pm.mu.RLock()
	if plan, ok := pm.cache[sessionID]; ok {
		pm.mu.RUnlock()
		return plan, nil
	}
	pm.mu.RUnlock()

	// Load from storage
	if pm.storage != nil {
		plan, err := pm.storage.GetBySessionID(ctx, sessionID)
		if err != nil {
			// Not an error if plan doesn't exist
			slog.DebugContext(ctx, "no plan found for session",
				"session_id", sessionID,
				"error", err)
			return nil, nil
		}

		// Cache it
		pm.mu.Lock()
		pm.cache[sessionID] = plan
		pm.mu.Unlock()

		return plan, nil
	}

	return nil, nil
}

// UpdateStepStatus updates a step's status and auto-advances to next step
func (pm *PlanManager) UpdateStepStatus(ctx context.Context, sessionID string, stepIdx int, status domain.PlanStepStatus, result string) error {
	plan, err := pm.GetActivePlan(ctx, sessionID)
	if err != nil {
		return err
	}

	if plan == nil {
		return fmt.Errorf("no active plan for session %s", sessionID)
	}

	// Update step
	if err := plan.UpdateStepStatus(stepIdx, status, result); err != nil {
		return err
	}

	// Persist changes
	if pm.storage != nil {
		if err := pm.storage.Update(ctx, plan); err != nil {
			slog.ErrorContext(ctx, "failed to update plan in storage",
				"plan_id", plan.ID,
				"session_id", sessionID,
				"step_idx", stepIdx,
				"error", err)
			return fmt.Errorf("failed to update plan: %w", err)
		}
	}

	slog.DebugContext(ctx, "plan step updated",
		"plan_id", plan.ID,
		"session_id", sessionID,
		"step_idx", stepIdx,
		"status", status)

	return nil
}

// UpdatePlanStatus updates the plan's overall status
func (pm *PlanManager) UpdatePlanStatus(ctx context.Context, sessionID string, status domain.PlanStatus) error {
	plan, err := pm.GetActivePlan(ctx, sessionID)
	if err != nil {
		return err
	}

	if plan == nil {
		return fmt.Errorf("no active plan for session %s", sessionID)
	}

	// Update status
	plan.SetStatus(status)

	// Persist changes
	if pm.storage != nil {
		if err := pm.storage.Update(ctx, plan); err != nil {
			slog.ErrorContext(ctx, "failed to update plan status in storage",
				"plan_id", plan.ID,
				"session_id", sessionID,
				"error", err)
			return fmt.Errorf("failed to update plan status: %w", err)
		}
	}

	slog.InfoContext(ctx, "plan status updated",
		"plan_id", plan.ID,
		"session_id", sessionID,
		"status", status)

	// Clear from cache if completed/abandoned
	if status == domain.PlanStatusCompleted || status == domain.PlanStatusAbandoned {
		pm.mu.Lock()
		delete(pm.cache, sessionID)
		pm.mu.Unlock()
	}

	return nil
}

// AddStep adds a new step to the plan
func (pm *PlanManager) AddStep(ctx context.Context, sessionID, description, reasoning string) error {
	plan, err := pm.GetActivePlan(ctx, sessionID)
	if err != nil {
		return err
	}

	if plan == nil {
		return fmt.Errorf("no active plan for session %s", sessionID)
	}

	// Add step
	plan.AddStep(description, reasoning)

	// Persist changes
	if pm.storage != nil {
		if err := pm.storage.Update(ctx, plan); err != nil {
			slog.ErrorContext(ctx, "failed to update plan after adding step",
				"plan_id", plan.ID,
				"session_id", sessionID,
				"error", err)
			return fmt.Errorf("failed to update plan: %w", err)
		}
	}

	slog.DebugContext(ctx, "step added to plan",
		"plan_id", plan.ID,
		"session_id", sessionID,
		"description", description)

	return nil
}

// RemoveStep removes a step from the plan
func (pm *PlanManager) RemoveStep(ctx context.Context, sessionID string, stepIndex int) error {
	plan, err := pm.GetActivePlan(ctx, sessionID)
	if err != nil {
		return err
	}

	if plan == nil {
		return fmt.Errorf("no active plan for session %s", sessionID)
	}

	// Remove step
	if err := plan.RemoveStep(stepIndex); err != nil {
		return err
	}

	// Persist changes
	if pm.storage != nil {
		if err := pm.storage.Update(ctx, plan); err != nil {
			slog.ErrorContext(ctx, "failed to update plan after removing step",
				"plan_id", plan.ID,
				"session_id", sessionID,
				"error", err)
			return fmt.Errorf("failed to update plan: %w", err)
		}
	}

	slog.DebugContext(ctx, "step removed from plan",
		"plan_id", plan.ID,
		"session_id", sessionID,
		"step_index", stepIndex)

	return nil
}

// ModifyStep modifies a step's description and reasoning
func (pm *PlanManager) ModifyStep(ctx context.Context, sessionID string, stepIndex int, description, reasoning string) error {
	plan, err := pm.GetActivePlan(ctx, sessionID)
	if err != nil {
		return err
	}

	if plan == nil {
		return fmt.Errorf("no active plan for session %s", sessionID)
	}

	// Modify step
	if err := plan.ModifyStep(stepIndex, description, reasoning); err != nil {
		return err
	}

	// Persist changes
	if pm.storage != nil {
		if err := pm.storage.Update(ctx, plan); err != nil {
			slog.ErrorContext(ctx, "failed to update plan after modifying step",
				"plan_id", plan.ID,
				"session_id", sessionID,
				"error", err)
			return fmt.Errorf("failed to update plan: %w", err)
		}
	}

	slog.DebugContext(ctx, "step modified in plan",
		"plan_id", plan.ID,
		"session_id", sessionID,
		"step_index", stepIndex)

	return nil
}

// ClearCache removes a plan from the cache (useful for testing)
func (pm *PlanManager) ClearCache(sessionID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.cache, sessionID)
}

// ClearAllCache clears the entire cache
func (pm *PlanManager) ClearAllCache() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.cache = make(map[string]*domain.Plan)
}

// GetContextReminder implements react.ContextReminderProvider
// Returns a reminder about current plan state to inject into LLM context
func (pm *PlanManager) GetContextReminder(ctx context.Context, sessionID string) (content string, priority int, ok bool) {
	if sessionID == "" {
		return "", 0, false
	}

	plan, err := pm.GetActivePlan(ctx, sessionID)
	if err != nil || plan == nil || plan.Status != domain.PlanStatusActive {
		return "", 0, false
	}

	// Build action instruction based on current step
	actionInstruction := ""
	if currentStep := plan.CurrentStep(); currentStep != nil {
		actionInstruction = fmt.Sprintf(
			"\n\n🚨 **ACTION REQUIRED NOW:** You gathered enough info for Step %d. "+
				"YOUR NEXT CALL MUST BE `manage_plan` with `\"status\": \"completed\"` for step %d!\n"+
				"Example: manage_plan({\"goal\": \"%s\", \"steps\": [{\"index\": %d, \"status\": \"completed\"}, ...]})",
			currentStep.Index+1,
			currentStep.Index,
			plan.Goal,
			currentStep.Index,
		)
	}

	content = fmt.Sprintf(
		"\n**REMEMBER YOUR GOAL:** %s\n%s%s\n**Stay focused on this goal!**",
		plan.Goal,
		plan.ToCompactContext(),
		actionInstruction,
	)

	return content, 100, true // High priority - plan reminders should be at the end
}

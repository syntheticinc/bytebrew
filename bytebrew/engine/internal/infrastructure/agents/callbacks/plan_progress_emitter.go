package callbacks

import (
	"context"
	"fmt"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
)

// PlanProvider is a consumer-side interface for plan access.
type PlanProvider interface {
	GetActivePlan(ctx context.Context, sessionID string) (*domain.Plan, error)
}

// PlanProgressEmitter emits plan progress events after manage_plan tool calls.
type PlanProgressEmitter struct {
	emitter     *EventEmitter
	counter     *StepCounter
	planManager PlanProvider
	sessionID   string
}

// NewPlanProgressEmitter creates a new PlanProgressEmitter.
func NewPlanProgressEmitter(
	emitter *EventEmitter,
	counter *StepCounter,
	planManager PlanProvider,
	sessionID string,
) *PlanProgressEmitter {
	return &PlanProgressEmitter{
		emitter:     emitter,
		counter:     counter,
		planManager: planManager,
		sessionID:   sessionID,
	}
}

// EmitPlanProgress emits a plan progress event.
func (p *PlanProgressEmitter) EmitPlanProgress(ctx context.Context) {
	if p.planManager == nil || p.sessionID == "" {
		return
	}

	plan, err := p.planManager.GetActivePlan(ctx, p.sessionID)
	if err != nil || plan == nil {
		return
	}

	completed, total := plan.Progress()
	current := plan.CurrentStep()

	currentStepDesc := ""
	if current != nil {
		currentStepDesc = fmt.Sprintf("Step %d: %s", current.Index+1, current.Description)
	} else {
		// No current step means all steps completed
		currentStepDesc = "All steps completed"
	}

	event := &domain.AgentEvent{
		Type:      domain.EventTypePlanProgress,
		Timestamp: time.Now(),
		Step:      p.counter.GetStep(),
		Content:   plan.Goal,
		Metadata: map[string]interface{}{
			"plan_id":      plan.ID,
			"goal":         plan.Goal,
			"completed":    completed,
			"total":        total,
			"progress":     fmt.Sprintf("%d/%d", completed, total),
			"current_step": currentStepDesc,
			"plan_status":  string(plan.Status),
		},
	}

	p.emitter.Emit(ctx, event)
}

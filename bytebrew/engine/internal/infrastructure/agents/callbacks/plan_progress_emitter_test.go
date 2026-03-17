package callbacks

import (
	"context"
	"testing"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPlanProvider implements PlanProvider for tests
type mockPlanProvider struct {
	plan *domain.Plan
	err  error
}

func (m *mockPlanProvider) GetActivePlan(ctx context.Context, sessionID string) (*domain.Plan, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.plan, nil
}

func TestPlanProgressEmitter_EmitPlanProgress(t *testing.T) {
	collector := newEventCollector()
	counter := NewStepCounter()
	emitter := NewEventEmitter(collector.Callback, "supervisor")

	// NewPlan auto-marks first step as InProgress, so create with pending steps
	// then manually update statuses
	plan := domain.NewPlan("plan-1", "session-1", "Test goal", []*domain.PlanStep{
		{Index: 0, Description: "Step 1", Status: domain.StepStatusPending},
		{Index: 1, Description: "Step 2", Status: domain.StepStatusPending},
		{Index: 2, Description: "Step 3", Status: domain.StepStatusPending},
	})
	// Complete step 0 (auto-advances step 1 to InProgress)
	_ = plan.UpdateStepStatus(0, domain.StepStatusCompleted, "done")

	provider := &mockPlanProvider{plan: plan}
	pe := NewPlanProgressEmitter(emitter, counter, provider, "session-1")

	pe.EmitPlanProgress(context.Background())

	events := collector.GetEventsByType(domain.EventTypePlanProgress)
	require.Len(t, events, 1)
	assert.Equal(t, "Test goal", events[0].Content)
	assert.Equal(t, "1/3", events[0].Metadata["progress"])
	assert.Equal(t, "Step 2: Step 2", events[0].Metadata["current_step"])
}

func TestPlanProgressEmitter_NilPlanManager(t *testing.T) {
	collector := newEventCollector()
	counter := NewStepCounter()
	emitter := NewEventEmitter(collector.Callback, "supervisor")

	pe := NewPlanProgressEmitter(emitter, counter, nil, "session-1")

	// Should not panic and should not emit
	pe.EmitPlanProgress(context.Background())
	assert.Empty(t, collector.GetEvents())
}

func TestPlanProgressEmitter_EmptySessionID(t *testing.T) {
	collector := newEventCollector()
	counter := NewStepCounter()
	emitter := NewEventEmitter(collector.Callback, "supervisor")
	provider := &mockPlanProvider{plan: &domain.Plan{}}

	pe := NewPlanProgressEmitter(emitter, counter, provider, "")

	// Should not emit
	pe.EmitPlanProgress(context.Background())
	assert.Empty(t, collector.GetEvents())
}

func TestPlanProgressEmitter_AllStepsCompleted(t *testing.T) {
	collector := newEventCollector()
	counter := NewStepCounter()
	emitter := NewEventEmitter(collector.Callback, "supervisor")

	plan := domain.NewPlan("plan-1", "session-1", "Done goal", []*domain.PlanStep{
		{Index: 0, Description: "Step 1", Status: domain.StepStatusCompleted},
		{Index: 1, Description: "Step 2", Status: domain.StepStatusCompleted},
	})
	// Mark all as completed for this test
	plan.Steps[0].Status = domain.StepStatusCompleted
	plan.Steps[1].Status = domain.StepStatusCompleted

	provider := &mockPlanProvider{plan: plan}
	pe := NewPlanProgressEmitter(emitter, counter, provider, "session-1")

	pe.EmitPlanProgress(context.Background())

	events := collector.GetEventsByType(domain.EventTypePlanProgress)
	require.Len(t, events, 1)
	assert.Equal(t, "All steps completed", events[0].Metadata["current_step"])
}

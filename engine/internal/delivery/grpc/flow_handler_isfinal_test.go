package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/stretchr/testify/assert"
)

// mockWorkChecker implements orchestrator.ActiveWorkChecker for testing
type mockWorkChecker struct {
	hasActiveWork    bool
	isWaitingForUser bool
	summary          string
}

func (m *mockWorkChecker) HasActiveWork(ctx context.Context) bool {
	return m.hasActiveWork
}

func (m *mockWorkChecker) IsWaitingForUser(ctx context.Context) bool {
	return m.isWaitingForUser
}

func (m *mockWorkChecker) ActiveWorkSummary(ctx context.Context) string {
	return m.summary
}

func TestShouldSuppressIsFinal_AnswerComplete_NoWorkChecker(t *testing.T) {
	ctx := context.Background()
	event := &domain.AgentEvent{
		Type:       domain.EventTypeAnswer,
		IsComplete: true,
		Content:    "Task completed",
		Timestamp:  time.Now(),
	}

	result := shouldSuppressIsFinal(event, nil, ctx)

	assert.False(t, result, "should NOT suppress when workChecker is nil")
}

func TestShouldSuppressIsFinal_AnswerComplete_NoActiveWork(t *testing.T) {
	ctx := context.Background()
	event := &domain.AgentEvent{
		Type:       domain.EventTypeAnswer,
		IsComplete: true,
		Content:    "All work done",
		Timestamp:  time.Now(),
	}
	workChecker := &mockWorkChecker{
		hasActiveWork:    false,
		isWaitingForUser: false,
		summary:          "none",
	}

	result := shouldSuppressIsFinal(event, workChecker, ctx)

	assert.False(t, result, "should NOT suppress when no active work and not waiting for user")
}

func TestShouldSuppressIsFinal_AnswerComplete_HasActiveWork(t *testing.T) {
	ctx := context.Background()
	event := &domain.AgentEvent{
		Type:       domain.EventTypeAnswer,
		IsComplete: true,
		Content:    "Turn completed but work remains",
		Timestamp:  time.Now(),
	}
	workChecker := &mockWorkChecker{
		hasActiveWork:    true,
		isWaitingForUser: false,
		summary:          "[task-1] \"Implement feature\" (in_progress)",
	}

	result := shouldSuppressIsFinal(event, workChecker, ctx)

	assert.True(t, result, "should suppress when there is active work")
}

func TestShouldSuppressIsFinal_AnswerComplete_WaitingForUser(t *testing.T) {
	ctx := context.Background()
	event := &domain.AgentEvent{
		Type:       domain.EventTypeAnswer,
		IsComplete: true,
		Content:    "Asked user a question",
		Timestamp:  time.Now(),
	}
	workChecker := &mockWorkChecker{
		hasActiveWork:    false,
		isWaitingForUser: true,
		summary:          "waiting for user response",
	}

	result := shouldSuppressIsFinal(event, workChecker, ctx)

	assert.True(t, result, "should suppress when waiting for user response")
}

func TestShouldSuppressIsFinal_AnswerComplete_BothActiveAndWaiting(t *testing.T) {
	ctx := context.Background()
	event := &domain.AgentEvent{
		Type:       domain.EventTypeAnswer,
		IsComplete: true,
		Content:    "Asked user and have active tasks",
		Timestamp:  time.Now(),
	}
	workChecker := &mockWorkChecker{
		hasActiveWork:    true,
		isWaitingForUser: true,
		summary:          "[task-1] \"Implement feature\" (in_progress), waiting for user",
	}

	result := shouldSuppressIsFinal(event, workChecker, ctx)

	assert.True(t, result, "should suppress when both active work and waiting for user")
}

func TestShouldSuppressIsFinal_ToolCallEvent(t *testing.T) {
	ctx := context.Background()
	event := &domain.AgentEvent{
		Type:       domain.EventTypeToolCall,
		IsComplete: false,
		Content:    "read_file",
		Timestamp:  time.Now(),
	}
	workChecker := &mockWorkChecker{
		hasActiveWork:    true,
		isWaitingForUser: false,
	}

	result := shouldSuppressIsFinal(event, workChecker, ctx)

	assert.False(t, result, "should NOT suppress tool call events")
}

func TestShouldSuppressIsFinal_AnswerNotComplete(t *testing.T) {
	ctx := context.Background()
	event := &domain.AgentEvent{
		Type:       domain.EventTypeAnswer,
		IsComplete: false, // streaming chunk
		Content:    "partial answer...",
		Timestamp:  time.Now(),
	}
	workChecker := &mockWorkChecker{
		hasActiveWork:    true,
		isWaitingForUser: false,
	}

	result := shouldSuppressIsFinal(event, workChecker, ctx)

	assert.False(t, result, "should NOT suppress incomplete answer events")
}

func TestShouldSuppressIsFinal_ReasoningEvent(t *testing.T) {
	ctx := context.Background()
	event := &domain.AgentEvent{
		Type:       domain.EventTypeReasoning,
		IsComplete: true,
		Content:    "thinking about the problem...",
		Timestamp:  time.Now(),
	}
	workChecker := &mockWorkChecker{
		hasActiveWork:    true,
		isWaitingForUser: false,
	}

	result := shouldSuppressIsFinal(event, workChecker, ctx)

	assert.False(t, result, "should NOT suppress reasoning events")
}

func TestShouldSuppressIsFinal_PlanCreatedEvent(t *testing.T) {
	ctx := context.Background()
	event := &domain.AgentEvent{
		Type:       domain.EventTypePlanCreated,
		IsComplete: true,
		Content:    "plan created",
		Timestamp:  time.Now(),
	}
	workChecker := &mockWorkChecker{
		hasActiveWork:    true,
		isWaitingForUser: false,
	}

	result := shouldSuppressIsFinal(event, workChecker, ctx)

	assert.False(t, result, "should NOT suppress plan events")
}

func TestShouldSuppressIsFinal_UserQuestionEvent(t *testing.T) {
	ctx := context.Background()
	event := &domain.AgentEvent{
		Type:       domain.EventTypeUserQuestion,
		IsComplete: true,
		Content:    "Do you approve this plan?",
		Timestamp:  time.Now(),
	}
	workChecker := &mockWorkChecker{
		hasActiveWork:    false,
		isWaitingForUser: true,
	}

	result := shouldSuppressIsFinal(event, workChecker, ctx)

	assert.False(t, result, "should NOT suppress user question events")
}

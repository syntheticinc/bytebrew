package assistant

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

const (
	// BuilderAgentName is the internal name of the builder assistant.
	// Not visible in user agent list, not editable, not deletable.
	BuilderAgentName = "__bytebrew_builder_assistant__"
)

// Builder is the system-level builder assistant agent.
// It helps users create and configure agent workflows through natural language.
type Builder struct {
	ops        AdminOperations
	assembler  *Assembler
	interviews map[string]*InterviewState // sessionID -> state
}

// NewBuilder creates a new Builder assistant.
func NewBuilder(ops AdminOperations) *Builder {
	return &Builder{
		ops:        ops,
		assembler:  NewAssembler(ops),
		interviews: make(map[string]*InterviewState),
	}
}

// HandleMessage processes a user message in the builder assistant context.
// Returns the assistant's response text.
func (b *Builder) HandleMessage(ctx context.Context, sessionID, message string,
	hasSchemas bool, eventStream domain.AgentEventStream) (string, error) {

	isFirstVisit := !hasSchemas && b.interviews[sessionID] == nil

	router := NewRouter(hasSchemas, isFirstVisit)
	intent := router.Classify(message)

	slog.InfoContext(ctx, "builder: classified message", "intent", intent, "session", sessionID)

	switch intent {
	case IntentInterview:
		return b.handleInterview(ctx, sessionID, message, eventStream)
	case IntentDirect:
		return b.handleDirect(ctx, sessionID, message, eventStream)
	case IntentAnswer:
		return b.handleAnswer(message)
	default:
		return b.handleDirect(ctx, sessionID, message, eventStream)
	}
}

// IsSystemAgent returns true if the agent name is the builder assistant.
func IsSystemAgent(agentName string) bool {
	return agentName == BuilderAgentName
}

func (b *Builder) handleInterview(ctx context.Context, sessionID, message string,
	eventStream domain.AgentEventStream) (string, error) {

	state, exists := b.interviews[sessionID]
	if !exists {
		state = NewInterviewState()
		b.interviews[sessionID] = state

		// Return first question
		question := state.NextQuestion()
		return fmt.Sprintf("I'll help you set up your workflow! Let me ask a few questions first.\n\n%s", question), nil
	}

	// Process the answer to the current question
	currentQuestion := state.NextQuestion()
	state.ProcessAnswer(currentQuestion, message)

	// Check if interview is complete
	if state.IsComplete() {
		// Proceed to assembly
		plan := b.assembler.PlanFromInterview(state)

		response := fmt.Sprintf("Great! I have enough information. Here's what I'll create:\n\n"+
			"Schema: %s\n"+
			"Agents: %d\n\n"+
			"Assembling now...", plan.SchemaName, len(plan.Agents))

		// Execute assembly
		if err := b.assembler.Execute(ctx, plan, eventStream); err != nil {
			return "", fmt.Errorf("assembly failed: %w", err)
		}

		// Clean up interview state
		delete(b.interviews, sessionID)

		return response + "\n\nDone! Your workflow is ready. You can see it on the canvas.", nil
	}

	// Ask next question
	nextQuestion := state.NextQuestion()
	return nextQuestion, nil
}

func (b *Builder) handleDirect(_ context.Context, _ string, message string,
	_ domain.AgentEventStream) (string, error) {
	// In the full implementation, this would parse the request and execute admin operations.
	// For V2, we return a message indicating direct execution mode.
	return fmt.Sprintf("I'll execute that request directly: %s\n\nProcessing...", message), nil
}

func (b *Builder) handleAnswer(message string) (string, error) {
	// In the full implementation, this would use the LLM to answer questions.
	// For V2, we provide a helpful response template.
	return fmt.Sprintf("That's a great question about ByteBrew! Let me help you understand.\n\n"+
		"ByteBrew uses schemas to organize agents into workflows. "+
		"Agents can be connected with flow edges (sequential), transfer edges (hand-off), "+
		"or loop edges (retry with gates). Each agent can have capabilities like memory, "+
		"knowledge, guardrails, and escalation.\n\n"+
		"Would you like me to help you set something up?"), nil
}

// GetInterviewState returns the current interview state for a session (for testing).
func (b *Builder) GetInterviewState(sessionID string) (*InterviewState, bool) {
	state, ok := b.interviews[sessionID]
	return state, ok
}

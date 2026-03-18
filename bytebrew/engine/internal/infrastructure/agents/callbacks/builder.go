package callbacks

import (
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/agents"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	ucb "github.com/cloudwego/eino/utils/callbacks"
	"context"
)

// BuilderConfig holds configuration for constructing the callback builder.
type BuilderConfig struct {
	EventCallback    func(event *domain.AgentEvent) error
	ChunkCallback    func(chunk string) error
	Store            *agents.StepContentStore
	SessionID        string
	AgentID          string // "supervisor" or "code-agent-{uuid}"
	ToolCallRecorder ToolCallRecorder
}

// AgentCallbackBuilder wires together all callback sub-components
// and exposes the public API consumed by the agent.
type AgentCallbackBuilder struct {
	counter      *StepCounter
	emitter      *EventEmitter
	modelHandler *ModelEventHandler
	toolHandler  *ToolEventHandler
}

// NewBuilder creates and wires all callback components.
func NewBuilder(cfg BuilderConfig) *AgentCallbackBuilder {
	agentID := cfg.AgentID
	if agentID == "" {
		agentID = "supervisor"
	}

	counter := NewStepCounter()
	emitter := NewEventEmitter(cfg.EventCallback, agentID)
	extractor := agents.NewReasoningExtractor()

	modelHandler := NewModelEventHandler(emitter, counter, extractor, cfg.Store, cfg.ChunkCallback)
	toolHandler := NewToolEventHandler(emitter, counter, modelHandler, cfg.ToolCallRecorder, cfg.SessionID)

	return &AgentCallbackBuilder{
		counter:      counter,
		emitter:      emitter,
		modelHandler: modelHandler,
		toolHandler:  toolHandler,
	}
}

// BuildCallbackOption creates an Eino agent option with the callback handler.
func (b *AgentCallbackBuilder) BuildCallbackOption() agent.AgentOption {
	modelHandler := &ucb.ModelCallbackHandler{
		OnEnd:                 b.modelHandler.OnModelEnd,
		OnEndWithStreamOutput: b.modelHandler.OnModelEndWithStreamOutput,
	}
	toolHandler := &ucb.ToolCallbackHandler{
		OnStart: b.toolHandler.OnToolStart,
		OnEnd:   b.toolHandler.OnToolEnd,
	}
	handler := ucb.NewHandlerHelper().
		ChatModel(modelHandler).
		Tool(toolHandler).
		Handler()
	return agent.WithComposeOptions(compose.WithCallbacks(handler))
}

// GetStep returns the current step (thread-safe, public method).
func (b *AgentCallbackBuilder) GetStep() int {
	return b.counter.GetStep()
}

// FinalizeAccumulatedText emits EventTypeAnswer for any accumulated streamed text.
func (b *AgentCallbackBuilder) FinalizeAccumulatedText(ctx context.Context) {
	b.modelHandler.FinalizeAccumulatedText(ctx)
}

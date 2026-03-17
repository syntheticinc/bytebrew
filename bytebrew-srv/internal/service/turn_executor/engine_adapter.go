package turn_executor

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/agents"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/agents/react"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/tools"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/service/engine"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/config"
	"github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
)

// Consumer-side interfaces (defined here where they're used)

// AgentEngine executes agents with persistence
type AgentEngine interface {
	Execute(ctx context.Context, cfg engine.ExecutionConfig) (*engine.ExecutionResult, error)
}

// FlowProvider provides flow configurations
type FlowProvider interface {
	GetFlow(ctx context.Context, flowType domain.FlowType) (*domain.Flow, error)
}

// ToolResolver resolves tool names to instances
type ToolResolver interface {
	Resolve(ctx context.Context, toolNames []string, deps tools.ToolDependencies) ([]einotool.InvokableTool, error)
}

// ToolDependenciesProvider creates tool deps for a given session
type ToolDependenciesProvider interface {
	GetDependencies(sessionID, projectKey string) tools.ToolDependencies
}

// PlanManager provides plan management (pass-through to engine)
type PlanManager interface {
	CreatePlan(ctx context.Context, sessionID, goal string, steps []*domain.PlanStep) (*domain.Plan, error)
	GetActivePlan(ctx context.Context, sessionID string) (*domain.Plan, error)
	UpdateStepStatus(ctx context.Context, sessionID string, stepIdx int, status domain.PlanStepStatus, result string) error
	UpdatePlanStatus(ctx context.Context, sessionID string, status domain.PlanStatus) error
	AddStep(ctx context.Context, sessionID, description, reasoning string) error
	RemoveStep(ctx context.Context, sessionID string, stepIndex int) error
	ModifyStep(ctx context.Context, sessionID string, stepIndex int, description, reasoning string) error
}

// ToolCallRecorder records tool calls (pass-through to engine)
type ToolCallRecorder interface {
	RecordToolCall(sessionID, toolName string)
	RecordToolResult(sessionID, toolName, result string)
}

// ContextReminderProvider provides context reminders (pass-through to engine)
type ContextReminderProvider interface {
	GetContextReminder(ctx context.Context, sessionID string) (string, int, bool)
}

// EngineAdapter adapts Engine to TurnExecutor interface (orchestrator.TurnExecutor)
// It bridges the Orchestrator event loop with the new Engine
type EngineAdapter struct {
	engine           AgentEngine
	flowProvider     FlowProvider
	toolResolver     ToolResolver
	toolDeps         ToolDependenciesProvider
	chatModel        model.ToolCallingChatModel
	agentConfig      *config.AgentConfig
	modelName        string
	// pass-through deps
	planManager      PlanManager
	contextReminders []ContextReminderProvider
	toolCallRecorder ToolCallRecorder
}

// Config holds configuration for EngineAdapter
type Config struct {
	Engine           AgentEngine
	FlowProvider     FlowProvider
	ToolResolver     ToolResolver
	ToolDeps         ToolDependenciesProvider
	ChatModel        model.ToolCallingChatModel
	AgentConfig      *config.AgentConfig
	ModelName        string
	PlanManager      PlanManager
	ContextReminders []ContextReminderProvider
	ToolCallRecorder ToolCallRecorder
}

// NewEngineAdapter creates a new EngineAdapter
func NewEngineAdapter(cfg Config) (*EngineAdapter, error) {
	if cfg.Engine == nil {
		return nil, fmt.Errorf("engine is required")
	}
	if cfg.FlowProvider == nil {
		return nil, fmt.Errorf("flow provider is required")
	}
	if cfg.ToolResolver == nil {
		return nil, fmt.Errorf("tool resolver is required")
	}
	if cfg.ToolDeps == nil {
		return nil, fmt.Errorf("tool dependencies provider is required")
	}
	if cfg.ChatModel == nil {
		return nil, fmt.Errorf("chat model is required")
	}

	return &EngineAdapter{
		engine:           cfg.Engine,
		flowProvider:     cfg.FlowProvider,
		toolResolver:     cfg.ToolResolver,
		toolDeps:         cfg.ToolDeps,
		chatModel:        cfg.ChatModel,
		agentConfig:      cfg.AgentConfig,
		modelName:        cfg.ModelName,
		planManager:      cfg.PlanManager,
		contextReminders: cfg.ContextReminders,
		toolCallRecorder: cfg.ToolCallRecorder,
	}, nil
}

// ExecuteTurn implements orchestrator.TurnExecutor interface
func (e *EngineAdapter) ExecuteTurn(
	ctx context.Context,
	sessionID, projectKey, question string,
	chunkCallback func(chunk string) error,
	eventCallback func(event *domain.AgentEvent) error,
) error {
	// 1. Get flow config for supervisor
	flow, err := e.flowProvider.GetFlow(ctx, domain.FlowType("supervisor"))
	if err != nil {
		return fmt.Errorf("get supervisor flow: %w", err)
	}

	// 2. Get tool dependencies
	toolDeps := e.toolDeps.GetDependencies(sessionID, projectKey)

	// 3. Resolve tools from flow.ToolNames
	resolvedTools, err := e.toolResolver.Resolve(ctx, flow.ToolNames, toolDeps)
	if err != nil {
		return fmt.Errorf("resolve tools: %w", err)
	}

	// 4. Convert InvokableTool to BaseTool (slice casting)
	baseTools := convertToBaseTools(resolvedTools)

	// 5. Convert context reminders to engine-compatible interface
	engineReminders := convertContextRemindersToEngine(e.contextReminders)

	// 6. Build ExecutionConfig
	var compressor engine.MessageCompressor
	if flow.MaxContextSize > 0 {
		compressor = engine.MessageCompressor(agents.NewContextRewriter(flow.MaxContextSize))
	}
	execCfg := engine.ExecutionConfig{
		SessionID:         sessionID,
		AgentID:           "supervisor",
		Flow:              flow,
		Tools:             baseTools,
		Input:             question,
		ChatModel:         e.chatModel,
		Streaming:         true,
		ChunkCallback:     chunkCallback,
		EventCallback:     eventCallback,
		ContextReminders:  engineReminders,
		PlanManager:       convertPlanManagerToEngine(e.planManager),
		ToolCallRecorder:  convertToolCallRecorderToEngine(e.toolCallRecorder),
		ModelName:         e.modelName,
		AgentConfig:       e.agentConfig,
		MessageCompressor: compressor,
	}

	// 7. Execute via Engine
	result, err := e.engine.Execute(ctx, execCfg)
	if err != nil {
		return fmt.Errorf("execute engine: %w", err)
	}

	// Log result status
	slog.InfoContext(ctx, "[EngineAdapter] engine execution completed",
		"status", result.Status,
		"suspended_at", result.SuspendedAt)

	// 8. Send final completion signal so the client knows the turn is done.
	// agent.Stream() only emits IsComplete=false; we must emit IsComplete=true
	// after the engine finishes so the gRPC layer sends IsFinal=true to the client.
	if eventCallback != nil {
		eventCallback(&domain.AgentEvent{
			Type:       domain.EventTypeAnswer,
			Timestamp:  time.Now(),
			Content:    result.Answer,
			IsComplete: true,
			AgentID:    "supervisor",
		})
	}

	return nil
}

// convertToBaseTools converts []InvokableTool to []BaseTool
func convertToBaseTools(invokableTools []einotool.InvokableTool) []einotool.BaseTool {
	baseTools := make([]einotool.BaseTool, len(invokableTools))
	for i, t := range invokableTools {
		baseTools[i] = t // InvokableTool embeds BaseTool, so implicit conversion
	}
	return baseTools
}

// Adapters for converting consumer-side interfaces to engine-compatible types

// contextReminderEngineAdapter adapts turn_executor.ContextReminderProvider to react.ContextReminderProvider
type contextReminderEngineAdapter struct {
	provider ContextReminderProvider
}

func (a *contextReminderEngineAdapter) GetContextReminder(ctx context.Context, sessionID string) (string, int, bool) {
	return a.provider.GetContextReminder(ctx, sessionID)
}

func convertContextRemindersToEngine(providers []ContextReminderProvider) []react.ContextReminderProvider {
	if providers == nil {
		return nil
	}
	result := make([]react.ContextReminderProvider, len(providers))
	for i, p := range providers {
		result[i] = &contextReminderEngineAdapter{provider: p}
	}
	return result
}

// planManagerEngineAdapter adapts turn_executor.PlanManager to react.PlanManager interface
type planManagerEngineAdapter struct {
	pm PlanManager
}

func (a *planManagerEngineAdapter) CreatePlan(ctx context.Context, sessionID, goal string, steps []*domain.PlanStep) (*domain.Plan, error) {
	return a.pm.CreatePlan(ctx, sessionID, goal, steps)
}

func (a *planManagerEngineAdapter) GetActivePlan(ctx context.Context, sessionID string) (*domain.Plan, error) {
	return a.pm.GetActivePlan(ctx, sessionID)
}

func (a *planManagerEngineAdapter) UpdateStepStatus(ctx context.Context, sessionID string, stepIdx int, status domain.PlanStepStatus, result string) error {
	return a.pm.UpdateStepStatus(ctx, sessionID, stepIdx, status, result)
}

func (a *planManagerEngineAdapter) UpdatePlanStatus(ctx context.Context, sessionID string, status domain.PlanStatus) error {
	return a.pm.UpdatePlanStatus(ctx, sessionID, status)
}

func (a *planManagerEngineAdapter) AddStep(ctx context.Context, sessionID, description, reasoning string) error {
	return a.pm.AddStep(ctx, sessionID, description, reasoning)
}

func (a *planManagerEngineAdapter) RemoveStep(ctx context.Context, sessionID string, stepIndex int) error {
	return a.pm.RemoveStep(ctx, sessionID, stepIndex)
}

func (a *planManagerEngineAdapter) ModifyStep(ctx context.Context, sessionID string, stepIndex int, description, reasoning string) error {
	return a.pm.ModifyStep(ctx, sessionID, stepIndex, description, reasoning)
}

func convertPlanManagerToEngine(pm PlanManager) react.PlanManager {
	if pm == nil {
		return nil
	}
	return &planManagerEngineAdapter{pm: pm}
}

// toolCallRecorderEngineAdapter adapts turn_executor.ToolCallRecorder to react.ToolCallRecorder
type toolCallRecorderEngineAdapter struct {
	recorder ToolCallRecorder
}

func (a *toolCallRecorderEngineAdapter) RecordToolCall(sessionID, toolName string) {
	a.recorder.RecordToolCall(sessionID, toolName)
}

func (a *toolCallRecorderEngineAdapter) RecordToolResult(sessionID, toolName, result string) {
	a.recorder.RecordToolResult(sessionID, toolName, result)
}

func convertToolCallRecorderToEngine(recorder ToolCallRecorder) react.ToolCallRecorder {
	if recorder == nil {
		return nil
	}
	return &toolCallRecorderEngineAdapter{recorder: recorder}
}

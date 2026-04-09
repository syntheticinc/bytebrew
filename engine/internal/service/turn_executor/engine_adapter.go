package turn_executor

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/agents"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/agents/react"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/tools"
	"github.com/syntheticinc/bytebrew/engine/internal/service/engine"
	"github.com/syntheticinc/bytebrew/engine/pkg/config"
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

// ToolCallRecorder records tool calls (pass-through to engine)
type ToolCallRecorder interface {
	RecordToolCall(sessionID, toolName string)
	RecordToolResult(sessionID, toolName, result string)
}

// ContextReminderProvider provides context reminders (pass-through to engine)
type ContextReminderProvider interface {
	GetContextReminder(ctx context.Context, sessionID string) (string, int, bool)
}

// GuardrailChecker evaluates agent output against guardrail rules (consumer-side interface).
type GuardrailChecker interface {
	Evaluate(ctx context.Context, config *GuardrailCheckConfig, output string) (*GuardrailCheckResult, error)
}

// GuardrailCheckConfig holds guardrail configuration for a check.
type GuardrailCheckConfig struct {
	Mode         string
	OnFailure    string
	MaxRetries   int
	FallbackText string
	JSONSchema   string
	JudgePrompt  string
	JudgeModel   string
	WebhookURL   string
}

// GuardrailCheckResult holds the result of a guardrail check.
type GuardrailCheckResult struct {
	Passed bool
	Reason string
}

// FlowExecutor executes multi-agent flow pipelines (consumer-side interface).
type FlowExecutor interface {
	HasOutgoingEdges(ctx context.Context, schemaID uint, agentName string) (bool, error)
	Execute(ctx context.Context, cfg FlowExecConfig, entryAgent, input string) error
}

// FlowExecConfig holds flow execution configuration.
type FlowExecConfig struct {
	SchemaID    uint
	SessionID   string
	EventStream domain.AgentEventStream
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
	agentName        string
	// pass-through deps
	contextReminders []ContextReminderProvider
	toolCallRecorder ToolCallRecorder
	// US-002: Flow executor for multi-agent pipelines
	flowExecutor FlowExecutor
	schemaID     uint
	// US-003: Guardrail pipeline
	guardrail       GuardrailChecker
	guardrailConfig *GuardrailCheckConfig
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
	AgentName        string
	ContextReminders []ContextReminderProvider
	ToolCallRecorder ToolCallRecorder
	// US-002: Flow executor (nil = no flow execution)
	FlowExecutor FlowExecutor
	SchemaID     uint
	// US-003: Guardrail pipeline (nil = no guardrails)
	Guardrail       GuardrailChecker
	GuardrailConfig *GuardrailCheckConfig
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
		agentName:        cfg.AgentName,
		contextReminders: cfg.ContextReminders,
		toolCallRecorder: cfg.ToolCallRecorder,
		flowExecutor:     cfg.FlowExecutor,
		schemaID:         cfg.SchemaID,
		guardrail:        cfg.Guardrail,
		guardrailConfig:  cfg.GuardrailConfig,
	}, nil
}

// ExecuteTurn implements orchestrator.TurnExecutor interface
func (e *EngineAdapter) ExecuteTurn(
	ctx context.Context,
	sessionID, projectKey, question string,
	chunkCallback func(chunk string) error,
	eventCallback func(event *domain.AgentEvent) error,
) error {
	// 1. Get flow config for the agent
	flow, err := e.flowProvider.GetFlow(ctx, domain.FlowType(e.agentName))
	if err != nil {
		return fmt.Errorf("get flow %q: %w", e.agentName, err)
	}

	// 2. Get tool dependencies
	toolDeps := e.toolDeps.GetDependencies(sessionID, projectKey)
	toolDeps.AgentName = flow.Name
	toolDeps.KnowledgePath = flow.KnowledgePath
	toolDeps.MCPServers = flow.MCPServers
	// Set schema scope for memory tools (0 = no explicit schema context)
	toolDeps.SchemaID = strconv.FormatUint(uint64(e.schemaID), 10)

	// Populate spawn targets from flow's SpawnPolicy
	canSpawn := make([]string, len(flow.Spawn.AllowedFlows))
	for i, ft := range flow.Spawn.AllowedFlows {
		canSpawn[i] = string(ft)
	}
	toolDeps.CanSpawn = canSpawn

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
		AgentID:           e.agentName,
		Flow:              flow,
		Tools:             baseTools,
		Input:             question,
		ChatModel:         e.chatModel,
		Streaming:         true,
		ChunkCallback:     chunkCallback,
		EventCallback:     eventCallback,
		ContextReminders:  engineReminders,
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

	answer := result.Answer

	// US-003: Guardrail check on agent output before sending to user
	if e.guardrail != nil && e.guardrailConfig != nil && answer != "" {
		checkResult, grErr := e.guardrail.Evaluate(ctx, e.guardrailConfig, answer)
		if grErr != nil {
			slog.ErrorContext(ctx, "[EngineAdapter] guardrail evaluation failed",
				"agent", e.agentName, "error", grErr)
			// On guardrail error with fallback configured, use fallback text
			if e.guardrailConfig.OnFailure == "fallback" && e.guardrailConfig.FallbackText != "" {
				answer = e.guardrailConfig.FallbackText
			} else {
				return fmt.Errorf("guardrail check failed: %w", grErr)
			}
		} else if !checkResult.Passed {
			slog.WarnContext(ctx, "[EngineAdapter] guardrail check failed",
				"agent", e.agentName, "reason", checkResult.Reason)
			if e.guardrailConfig.OnFailure == "fallback" && e.guardrailConfig.FallbackText != "" {
				answer = e.guardrailConfig.FallbackText
			}
		}
	}

	// 8. Send final completion signal so the client knows the turn is done.
	// agent.Stream() only emits IsComplete=false; we must emit IsComplete=true
	// after the engine finishes so the gRPC layer sends IsFinal=true to the client.
	if eventCallback != nil {
		eventCallback(&domain.AgentEvent{
			Type:       domain.EventTypeAnswer,
			Timestamp:  time.Now(),
			Content:    answer,
			IsComplete: true,
			AgentID:    e.agentName,
		})
	}

	// US-002: Execute flow pipeline if agent has outgoing edges
	if e.flowExecutor != nil && e.schemaID > 0 {
		hasEdges, edgeErr := e.flowExecutor.HasOutgoingEdges(ctx, e.schemaID, e.agentName)
		if edgeErr != nil {
			slog.WarnContext(ctx, "[EngineAdapter] failed to check outgoing edges",
				"agent", e.agentName, "schema_id", e.schemaID, "error", edgeErr)
		} else if hasEdges {
			slog.InfoContext(ctx, "[EngineAdapter] executing flow pipeline",
				"agent", e.agentName, "schema_id", e.schemaID)
			flowCfg := FlowExecConfig{
				SchemaID:    e.schemaID,
				SessionID:   sessionID,
				EventStream: eventCallbackStream(eventCallback),
			}
			if flowErr := e.flowExecutor.Execute(ctx, flowCfg, e.agentName, answer); flowErr != nil {
				slog.ErrorContext(ctx, "[EngineAdapter] flow execution failed",
					"agent", e.agentName, "error", flowErr)
				// Flow failure is not fatal — the primary agent answer was already sent
			}
		}
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

// eventCallbackStreamAdapter wraps eventCallback as domain.AgentEventStream.
type eventCallbackStreamAdapter struct {
	cb func(event *domain.AgentEvent) error
}

func (a *eventCallbackStreamAdapter) Send(event *domain.AgentEvent) error {
	if a.cb == nil {
		return nil
	}
	return a.cb(event)
}

// eventCallbackStream wraps eventCallback as domain.AgentEventStream.
// Returns nil if eventCallback is nil.
func eventCallbackStream(cb func(event *domain.AgentEvent) error) domain.AgentEventStream {
	if cb == nil {
		return nil
	}
	return &eventCallbackStreamAdapter{cb: cb}
}

package turnexecutor

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/agents"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/agents/react"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/llm"
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
	GetFlow(ctx context.Context, agentName string) (*domain.Flow, error)
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
	Strict       bool // when true, overrides OnFailure to "error" (block output)
}

// GuardrailCheckResult holds the result of a guardrail check.
type GuardrailCheckResult struct {
	Passed bool
	Reason string
}

// EngineAdapter adapts Engine to TurnExecutor interface (orchestrator.TurnExecutor)
// It bridges the Orchestrator event loop with the new Engine.
//
// V2 note: post-Group A.1 the schema-level multi-agent pipeline executor was
// removed. Multi-agent delegation in V2 is expressed by the agent itself via
// tool calls (see docs/architecture/agent-first-runtime.md §3.1), not by a
// separate flow Executor walking edge types.
type EngineAdapter struct {
	engine           AgentEngine
	flowProvider     FlowProvider
	toolResolver     ToolResolver
	toolDeps         ToolDependenciesProvider
	chatModel        model.ToolCallingChatModel
	agentConfig      *config.AgentConfig
	modelName        string
	agentName        string
	agentUUID        string // uuid FK → agents.id (for engine execution context)
	// pass-through deps
	contextReminders []ContextReminderProvider
	toolCallRecorder ToolCallRecorder
	// Schema scope for memory tools (empty = no explicit schema context)
	schemaID string
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
	AgentUUID        string // uuid FK → agents.id (for engine execution context)
	ContextReminders []ContextReminderProvider
	ToolCallRecorder ToolCallRecorder
	// Schema scope (empty = no explicit schema context)
	SchemaID string
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
		agentUUID:        cfg.AgentUUID,
		contextReminders: cfg.ContextReminders,
		toolCallRecorder: cfg.ToolCallRecorder,
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
	flow, err := e.flowProvider.GetFlow(ctx, e.agentName)
	if err != nil {
		return fmt.Errorf("get flow %q: %w", e.agentName, err)
	}

	// 2. Get tool dependencies
	toolDeps := e.toolDeps.GetDependencies(sessionID, projectKey)
	toolDeps.AgentName = flow.Name
	toolDeps.MCPServers = flow.MCPServers
	// Set schema scope for memory tools (0 = no explicit schema context)
	toolDeps.SchemaID = e.schemaID

	toolDeps.ConfirmBefore = flow.ConfirmBefore

	// Pull ConfirmRequester from proxy if available (set by processor for SSE path)
	if cr, ok := toolDeps.Proxy.(interface{ ConfirmRequester() tools.ConfirmationRequester }); ok {
		toolDeps.ConfirmRequester = cr.ConfirmRequester()
	}

	// Populate spawn targets from flow's SpawnPolicy
	toolDeps.CanSpawn = flow.Spawn.AllowedFlows

	// 3. Resolve tools from flow.ToolNames
	slog.InfoContext(ctx, "[EngineAdapter] resolving tools", "agent", e.agentName, "flow_tool_names_count", len(flow.ToolNames), "flow_tool_names", flow.ToolNames)
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
	// US-003: When guardrail is configured, intercept answer events from the stream
	// to collect the full answer text for post-execution validation.
	var collectedAnswer strings.Builder
	wrappedEventCallback := eventCallback
	if e.guardrail != nil && e.guardrailConfig != nil && eventCallback != nil {
		wrappedEventCallback = func(event *domain.AgentEvent) error {
			if event.Type == domain.EventTypeAnswer && !event.IsComplete {
				collectedAnswer.WriteString(event.Content)
			}
			return eventCallback(event)
		}
	}

	// Wrap ChatModel with per-agent model parameters (temperature, top_p, etc.)
	chatModel := llm.WrapWithModelParams(e.chatModel, llm.ModelParams{
		Temperature: flow.Temperature,
		TopP:        flow.TopP,
		MaxTokens:   flow.MaxTokens,
		Stop:        flow.StopSequences,
	})

	execCfg := engine.ExecutionConfig{
		SessionID:         sessionID,
		AgentID:           e.agentUUID,
		Flow:              flow,
		Tools:             baseTools,
		Input:             question,
		ChatModel:         chatModel,
		Streaming:         true,
		ChunkCallback:     chunkCallback,
		EventCallback:     wrappedEventCallback,
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

	// Use result.Answer for non-streaming, collected answer for streaming.
	answer := result.Answer
	if answer == "" {
		answer = collectedAnswer.String()
	}

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

	// V2 (Group A.1): no schema-level pipeline dispatch happens here. Multi-agent
	// delegation is expressed by the agent itself through tool calls (see
	// docs/architecture/agent-first-runtime.md §3.1).

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

// contextReminderEngineAdapter adapts turnexecutor.ContextReminderProvider to react.ContextReminderProvider
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

// toolCallRecorderEngineAdapter adapts turnexecutor.ToolCallRecorder to react.ToolCallRecorder
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


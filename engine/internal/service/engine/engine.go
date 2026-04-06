package engine

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/agents/react"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/adapters"
	"github.com/syntheticinc/bytebrew/engine/pkg/config"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// MessageCompressor compresses message history to fit within token budget.
type MessageCompressor func(ctx context.Context, messages []*schema.Message) []*schema.Message

// Consumer-side interfaces

// SnapshotRepository provides persistence for agent context snapshots
type SnapshotRepository interface {
	Save(ctx context.Context, snapshot *domain.AgentContextSnapshot) error
	Load(ctx context.Context, sessionID, agentID string) (*domain.AgentContextSnapshot, error)
	Delete(ctx context.Context, sessionID, agentID string) error
	FindActive(ctx context.Context) ([]*domain.AgentContextSnapshot, error)
}

// HistoryRepository provides persistence for messages
type HistoryRepository interface {
	Create(ctx context.Context, message *domain.Message) error
}

// ExecutionConfig defines what the caller passes to Execute
type ExecutionConfig struct {
	SessionID      string
	AgentID        string // "supervisor" | "code-agent-{uuid}"
	Flow           *domain.Flow
	Tools          []tool.BaseTool
	Input          string
	ChatModel      model.ToolCallingChatModel
	Streaming      bool
	ChunkCallback  func(chunk string) error
	EventCallback  func(event *domain.AgentEvent) error

	// Pass-through to react.Agent
	ContextReminders []react.ContextReminderProvider
	ToolCallRecorder react.ToolCallRecorder
	ModelName        string
	AgentConfig      *config.AgentConfig
	SessionDirName   string

	// Code agent specific
	ParentAgentID string
	SubtaskID     string

	// MessageCompressor compresses message history before saving snapshot.
	// If nil, no compression is applied.
	MessageCompressor MessageCompressor
}

// ExecutionResult contains the result of agent execution
type ExecutionResult struct {
	Status      ExecutionStatus
	Answer      string
	SuspendedAt string
}

// ExecutionStatus represents the execution outcome
type ExecutionStatus string

const (
	StatusCompleted ExecutionStatus = "completed"
	StatusSuspended ExecutionStatus = "suspended"
	StatusFailed    ExecutionStatus = "failed"
)

// Engine provides unified agent execution with suspend/resume and persistence
type Engine struct {
	snapshotRepo SnapshotRepository
	historyRepo  HistoryRepository
}

// New creates a new Engine
func New(snapshotRepo SnapshotRepository, historyRepo HistoryRepository) *Engine {
	return &Engine{
		snapshotRepo: snapshotRepo,
		historyRepo:  historyRepo,
	}
}

// Execute runs an agent with full persistence support
func (e *Engine) Execute(ctx context.Context, cfg ExecutionConfig) (*ExecutionResult, error) {
	if err := e.validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	// 1. Load snapshot for resume
	snapshot, err := e.snapshotRepo.Load(ctx, cfg.SessionID, cfg.AgentID)
	if err != nil {
		return nil, fmt.Errorf("load snapshot: %w", err)
	}

	if snapshot == nil {
		slog.InfoContext(ctx, "no snapshot found, starting fresh",
			"session_id", cfg.SessionID, "agent_id", cfg.AgentID)
	} else {
		slog.InfoContext(ctx, "snapshot found",
			"session_id", cfg.SessionID, "agent_id", cfg.AgentID,
			"status", snapshot.Status, "schema_version", snapshot.SchemaVersion,
			"compatible", snapshot.IsCompatible())
	}

	var historyMessages []*schema.Message
	if snapshot != nil && snapshot.IsCompatible() {
		historyMessages, err = adapters.DeserializeSchemaMessages(snapshot.ContextData)
		if err != nil {
			slog.WarnContext(ctx, "failed to deserialize snapshot, starting fresh",
				"agent_id", cfg.AgentID, "error", err)
			historyMessages = nil
		} else {
			slog.InfoContext(ctx, "loaded snapshot for resume",
				"agent_id", cfg.AgentID,
				"message_count", len(historyMessages),
				"step", snapshot.StepNumber)
		}
	}

	// 2. Build react.AgentConfig
	// Supervisor executes tools sequentially to prevent ask_user + spawn_agent
	// from running in parallel. Code agents keep parallel execution for performance.
	isSupervisor := cfg.AgentID == "" || cfg.AgentID == "supervisor"

	agentConfig := &react.AgentConfig{
		ChatModel:                cfg.ChatModel,
		Tools:                    cfg.Tools,
		MaxSteps:                 cfg.Flow.MaxSteps,
		SessionID:                cfg.SessionID,
		AgentConfig:              e.buildEffectiveAgentConfig(cfg),
		ModelName:                cfg.ModelName,
		HistoryMessages:          historyMessages,
		ContextReminderProviders: cfg.ContextReminders,
		ToolCallRecorder:         cfg.ToolCallRecorder,
		AgentID:                  cfg.AgentID,
		ParentAgentID:            cfg.ParentAgentID,
		SubtaskID:                cfg.SubtaskID,
		SessionDirName:           cfg.SessionDirName,
		SequentialTools:          isSupervisor,
	}

	// 3. Create message collector (wraps EventCallback for per-step persistence)
	collector := NewMessageCollector(cfg.SessionID, cfg.AgentID, e.historyRepo)
	wrappedEventCb := collector.WrapEventCallback(cfg.EventCallback)

	// 3b. Persist user message to history (so it appears on session reload)
	collector.CollectUserMessage(ctx, cfg.Input)

	// 4. Create and run agent
	agent, err := react.NewAgent(ctx, *agentConfig)
	if err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}

	var answer string
	var execErr error
	if cfg.Streaming {
		execErr = agent.Stream(ctx, cfg.Input, cfg.ChunkCallback, wrappedEventCb)
	} else {
		answer, execErr = agent.RunWithCallbacks(ctx, cfg.Input, wrappedEventCb)
	}

	// 5. Determine status
	status := StatusCompleted
	suspendedAt := ""
	if cfg.Flow.ShouldSuspendOn("final_answer") && execErr == nil {
		status = StatusSuspended
		suspendedAt = "final_answer"
	}
	if execErr != nil {
		if ctx.Err() != nil {
			// Client-initiated cancel — save as suspended for resume
			status = StatusSuspended
			slog.InfoContext(ctx, "execution cancelled, saving as suspended",
				"agent_id", cfg.AgentID)
		} else {
			status = StatusFailed
			slog.ErrorContext(ctx, "agent execution failed",
				"agent_id", cfg.AgentID, "error", execErr)
		}
	}

	// 6. Save snapshot — use background context if original is canceled
	// (we still want to persist state for session resume)
	saveCtx := ctx
	if ctx.Err() != nil {
		saveCtx = context.Background()
	}
	if saveErr := e.saveSnapshot(saveCtx, cfg, collector, historyMessages, status); saveErr != nil {
		slog.ErrorContext(ctx, "failed to save snapshot", "error", saveErr)
	}

	// 7. Return result
	return &ExecutionResult{
		Status:      status,
		Answer:      answer,
		SuspendedAt: suspendedAt,
	}, execErr
}

// RecoverInterrupted marks all active snapshots as interrupted
func (e *Engine) RecoverInterrupted(ctx context.Context) error {
	active, err := e.snapshotRepo.FindActive(ctx)
	if err != nil {
		return fmt.Errorf("find active snapshots: %w", err)
	}

	for _, snap := range active {
		if snap.Status == domain.AgentContextStatusActive {
			snap.MarkInterrupted()
			if err := e.snapshotRepo.Save(ctx, snap); err != nil {
				slog.ErrorContext(ctx, "failed to mark snapshot as interrupted",
					"agent_id", snap.AgentID, "error", err)
				continue
			}
			slog.WarnContext(ctx, "found interrupted agent",
				"agent_id", snap.AgentID, "session_id", snap.SessionID)
		}
	}
	return nil
}

// validateConfig validates ExecutionConfig
func (e *Engine) validateConfig(cfg ExecutionConfig) error {
	if cfg.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	if cfg.AgentID == "" {
		return fmt.Errorf("agent_id is required")
	}
	if cfg.Flow == nil {
		return fmt.Errorf("flow is required")
	}
	if cfg.ChatModel == nil {
		return fmt.Errorf("chat_model is required")
	}
	if err := cfg.Flow.Validate(); err != nil {
		return fmt.Errorf("invalid flow: %w", err)
	}
	return nil
}

// buildEffectiveAgentConfig creates effective agent config from ExecutionConfig.
// Flow.MaxContextSize (from DB) always takes priority over global AgentConfig default.
func (e *Engine) buildEffectiveAgentConfig(cfg ExecutionConfig) *config.AgentConfig {
	if cfg.AgentConfig == nil {
		// Build minimal config from Flow if AgentConfig not provided
		return &config.AgentConfig{
			Prompts: &config.PromptsConfig{
				SystemPrompt: cfg.Flow.SystemPrompt,
			},
			MaxContextSize:                cfg.Flow.MaxContextSize,
			MaxTurnDuration:               cfg.Flow.MaxTurnDuration,
			EnableEnhancedToolCallChecker: true,
		}
	}

	// Copy to avoid mutating shared global config pointer
	result := *cfg.AgentConfig
	result.EnableEnhancedToolCallChecker = true

	// Per-agent values from Flow (DB) override global defaults
	if cfg.Flow.MaxContextSize > 0 {
		result.MaxContextSize = cfg.Flow.MaxContextSize
	}
	if cfg.Flow.MaxTurnDuration > 0 {
		result.MaxTurnDuration = cfg.Flow.MaxTurnDuration
	}

	// Overlay Flow's system prompt when AgentConfig doesn't provide one
	hasFlowPrompt := cfg.Flow.SystemPrompt != ""
	hasConfigPrompt := result.Prompts != nil && result.Prompts.SystemPrompt != ""
	if hasFlowPrompt && !hasConfigPrompt {
		if result.Prompts == nil {
			result.Prompts = &config.PromptsConfig{}
		} else {
			promptsCopy := *result.Prompts
			result.Prompts = &promptsCopy
		}
		result.Prompts.SystemPrompt = cfg.Flow.SystemPrompt
	}

	return &result
}

// saveSnapshot saves the current execution state
func (e *Engine) saveSnapshot(
	ctx context.Context,
	cfg ExecutionConfig,
	collector *MessageCollector,
	historyMessages []*schema.Message,
	status ExecutionStatus,
) error {
	// Merge history + current user message + new messages from this execution.
	// The user message must be persisted so the next turn sees it in history —
	// MessageCollector only captures agent events (tool_call, tool_result, answer),
	// not the user input.
	allMessages := historyMessages
	if cfg.Input != "" {
		allMessages = append(allMessages, &schema.Message{
			Role:    schema.User,
			Content: cfg.Input,
		})
	}
	newMessages := collector.GetAccumulatedMessages()
	if len(newMessages) > 0 {
		allMessages = append(allMessages, newMessages...)
	}

	// Sanitize assistant messages: some LLM providers require a non-empty "content"
	// field on assistant messages with tool_calls. In streaming mode, the content may
	// be lost. Set a minimal placeholder so serialization (e.g. go-openai's omitempty)
	// doesn't strip the field entirely.
	for _, msg := range allMessages {
		if msg.Role == schema.Assistant && msg.Content == "" && len(msg.ToolCalls) > 0 {
			msg.Content = " "
		}
	}

	// Compress snapshot to prevent unbounded growth across session resumes.
	// Without compression, each resume adds new messages but never removes old ones,
	// causing the snapshot to grow indefinitely.
	if cfg.MessageCompressor != nil && len(allMessages) > 0 {
		beforeCount := len(allMessages)
		allMessages = cfg.MessageCompressor(ctx, allMessages)
		if len(allMessages) != beforeCount {
			slog.InfoContext(ctx, "compressed snapshot before saving",
				"before", beforeCount,
				"after", len(allMessages))
		}
	}

	contextData, err := adapters.SerializeSchemaMessages(allMessages)
	if err != nil {
		return fmt.Errorf("serialize messages: %w", err)
	}

	snap := &domain.AgentContextSnapshot{
		SessionID:     cfg.SessionID,
		AgentID:       cfg.AgentID,
		FlowType:      cfg.Flow.Type,
		SchemaVersion: domain.CurrentSchemaVersion,
		ContextData:   contextData,
		StepNumber:    collector.StepCount(),
		Status:        mapExecutionStatusToContextStatus(status),
	}

	if err := snap.Validate(); err != nil {
		return fmt.Errorf("invalid snapshot: %w", err)
	}

	if err := e.snapshotRepo.Save(ctx, snap); err != nil {
		return fmt.Errorf("save snapshot: %w", err)
	}

	slog.InfoContext(ctx, "saved snapshot",
		"agent_id", cfg.AgentID,
		"message_count", len(allMessages),
		"step", snap.StepNumber,
		"status", snap.Status)

	return nil
}

// mapExecutionStatusToContextStatus maps ExecutionStatus to AgentContextStatus
func mapExecutionStatusToContextStatus(status ExecutionStatus) domain.AgentContextStatus {
	switch status {
	case StatusCompleted:
		return domain.AgentContextStatusCompleted
	case StatusSuspended:
		return domain.AgentContextStatusSuspended
	case StatusFailed:
		return domain.AgentContextStatusInterrupted
	default:
		return domain.AgentContextStatusActive
	}
}

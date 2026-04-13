package agent

import (
	"context"
	"log/slog"

	pb "github.com/syntheticinc/bytebrew/engine/api/proto/gen"
	"github.com/syntheticinc/bytebrew/engine/internal/service/turn_executor"
	"github.com/syntheticinc/bytebrew/engine/pkg/config"
	"github.com/syntheticinc/bytebrew/engine/pkg/errors"
	"github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
)


// ChatModel defines interface for LLM chat model
type ChatModel interface {
	model.ToolCallingChatModel
}

// ClientOperationsProxy defines interface for gRPC client operations
type ClientOperationsProxy interface {
	ReadFile(ctx context.Context, sessionID, filePath string, startLine, endLine int32) (string, error)
	WriteFile(ctx context.Context, sessionID, filePath, content string) (string, error)
	EditFile(ctx context.Context, sessionID, filePath, oldString, newString string, replaceAll bool) (string, error)
	SearchCode(ctx context.Context, sessionID, query, projectKey string, limit int32, minScore float32) ([]byte, error)
	GetProjectTree(ctx context.Context, sessionID, projectKey, path string, maxDepth int) (string, error)
	GrepSearch(ctx context.Context, sessionID, pattern string, limit int32, fileTypes []string, ignoreCase bool) (string, error)
	GlobSearch(ctx context.Context, sessionID, pattern string, limit int32) (string, error)
	SymbolSearch(ctx context.Context, sessionID, symbolName string, limit int32, symbolTypes []string) (string, error)
	ExecuteSubQueries(ctx context.Context, sessionID string, subQueries []*pb.SubQuery) ([]*pb.SubResult, error)
	ExecuteCommand(ctx context.Context, sessionID, command, cwd string, timeout int32) (string, error)
	ExecuteCommandFull(ctx context.Context, sessionID string, arguments map[string]string) (string, error)
	AskUserQuestionnaire(ctx context.Context, sessionID, questionsJSON string) (string, error)
	LspRequest(ctx context.Context, sessionID, symbolName, operation string) (string, error)
}



// Service handles agent orchestration and flow execution
type Service struct {
	chatModel        ChatModel
	contextReminders []turn_executor.ContextReminderProvider
	toolCallHistory  *ToolCallHistoryReminder
	webSearchTool    einotool.InvokableTool
	webFetchTool     einotool.InvokableTool
	maxSteps         int
	agentConfig      *config.AgentConfig
	modelName        string
	streaming        bool // Enable streaming mode
	supervisorMode   bool // Supervisor mode with Code Agents
}

// Config holds configuration for Agent Service
type Config struct {
	ChatModel        ChatModel
	ContextReminders []turn_executor.ContextReminderProvider
	WebSearchTool    einotool.InvokableTool
	WebFetchTool     einotool.InvokableTool
	MaxSteps         int
	AgentConfig      *config.AgentConfig
	ModelName        string // Model name for reasoning extraction
	Streaming        bool   // Enable streaming mode
}

// New creates a new Agent Service
func New(cfg Config) (*Service, error) {
	if cfg.ChatModel == nil {
		return nil, errors.New(errors.CodeInvalidInput, "chat model is required")
	}

	// MaxSteps = 0 means no limit, use value from config as-is
	maxSteps := cfg.MaxSteps

	// Use provided AgentConfig or default
	agentConfig := cfg.AgentConfig
	if agentConfig == nil {
		agentConfig = config.DefaultAgentConfig()
	}

	// Create tool call history reminder
	toolCallHistory := NewToolCallHistoryReminder()

	// Add it to context reminders
	contextReminders := cfg.ContextReminders
	contextReminders = append(contextReminders, toolCallHistory)

	return &Service{
		chatModel:        cfg.ChatModel,
		contextReminders: contextReminders,
		toolCallHistory:  toolCallHistory,
		webSearchTool:    cfg.WebSearchTool,
		webFetchTool:     cfg.WebFetchTool,
		maxSteps:         maxSteps,
		agentConfig:      agentConfig,
		modelName:        cfg.ModelName,
		streaming:        cfg.Streaming,
		supervisorMode:   false,
	}, nil
}

// SetEnvironmentContext sets environment metadata (project root, platform)
// that will be injected into the LLM context as a reminder.
// Replaces any existing EnvironmentContextReminder.
func (s *Service) SetEnvironmentContext(projectRoot, platform string) {
	if projectRoot == "" && platform == "" {
		return
	}

	reminder := NewEnvironmentContextReminder(projectRoot, platform)

	// Replace existing EnvironmentContextReminder if any
	var newReminders []turn_executor.ContextReminderProvider
	for _, r := range s.contextReminders {
		if _, ok := r.(*EnvironmentContextReminder); !ok {
			newReminders = append(newReminders, r)
		}
	}
	newReminders = append(newReminders, reminder)
	s.contextReminders = newReminders

	// Propagate to AgentPool so Code Agents inherit environment context
}

// SetTestingStrategy sets project-level testing strategy
// that will be injected into the LLM context as a reminder.
// Replaces any existing TestingStrategyReminder.
func (s *Service) SetTestingStrategy(yamlContent string) {
	if yamlContent == "" {
		return
	}

	strategy, err := ParseTestingStrategy(yamlContent)
	if err != nil {
		slog.Warn("failed to parse testing strategy", "error", err)
		return
	}

	reminder := NewTestingStrategyReminder(strategy)

	// Replace existing TestingStrategyReminder if any
	var newReminders []turn_executor.ContextReminderProvider
	for _, r := range s.contextReminders {
		if _, ok := r.(*TestingStrategyReminder); !ok {
			newReminders = append(newReminders, r)
		}
	}
	newReminders = append(newReminders, reminder)
	s.contextReminders = newReminders

}


// GetToolCallRecorder returns the tool call recorder for callback integration
func (s *Service) GetToolCallRecorder() ToolCallRecorder {
	return s.toolCallHistory
}

// GetToolCallHistoryReminder returns the tool call history reminder for session cleanup
func (s *Service) GetToolCallHistoryReminder() *ToolCallHistoryReminder {
	return s.toolCallHistory
}

// GetContextReminders returns the context reminders for Engine integration
func (s *Service) GetContextReminders() []turn_executor.ContextReminderProvider {
	return s.contextReminders
}


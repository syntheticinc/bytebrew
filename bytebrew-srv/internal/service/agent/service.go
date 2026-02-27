package agent

import (
	"context"
	"log/slog"

	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/agents/react"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/service/turn_executor"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/config"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/errors"
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

// TaskManager defines interface for task management (consumer-side)
type TaskManager interface {
	CreateTask(ctx context.Context, sessionID, title, description string, criteria []string) (*domain.Task, error)
	ApproveTask(ctx context.Context, taskID string) error
	StartTask(ctx context.Context, taskID string) error
	GetTask(ctx context.Context, taskID string) (*domain.Task, error)
	GetTasks(ctx context.Context, sessionID string) ([]*domain.Task, error)
	CompleteTask(ctx context.Context, taskID string) error
	FailTask(ctx context.Context, taskID, reason string) error
	CancelTask(ctx context.Context, taskID, reason string) error
}

// WorkSubtaskManager defines interface for subtask management (consumer-side)
type WorkSubtaskManager interface {
	CreateSubtask(ctx context.Context, sessionID, taskID, title, description string, blockedBy, files []string) (*domain.Subtask, error)
	GetSubtask(ctx context.Context, subtaskID string) (*domain.Subtask, error)
	GetSubtasksByTask(ctx context.Context, taskID string) ([]*domain.Subtask, error)
	GetReadySubtasks(ctx context.Context, taskID string) ([]*domain.Subtask, error)
	AssignSubtaskToAgent(ctx context.Context, subtaskID, agentID string) error
	CompleteSubtask(ctx context.Context, subtaskID, result string) error
	FailSubtask(ctx context.Context, subtaskID, reason string) error
}

// AgentPoolManager defines interface for Code Agent pool (consumer-side)
type AgentPoolManager interface{}


// PlanManager defines interface for plan orchestration (consumer-side interface)
type PlanManager interface {
	CreatePlan(ctx context.Context, sessionID, goal string, steps []*domain.PlanStep) (*domain.Plan, error)
	GetActivePlan(ctx context.Context, sessionID string) (*domain.Plan, error)
	UpdateStepStatus(ctx context.Context, sessionID string, stepIdx int, status domain.PlanStepStatus, result string) error
	UpdatePlanStatus(ctx context.Context, sessionID string, status domain.PlanStatus) error
	AddStep(ctx context.Context, sessionID, description, reasoning string) error
	RemoveStep(ctx context.Context, sessionID string, stepIndex int) error
	ModifyStep(ctx context.Context, sessionID string, stepIndex int, description, reasoning string) error
}

// Service handles agent orchestration and flow execution
type Service struct {
	chatModel        ChatModel
	planManager      PlanManager
	taskManager      TaskManager
	subtaskManager   WorkSubtaskManager
	agentPool        AgentPoolManager
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
	PlanManager      PlanManager
	TaskManager      TaskManager
	SubtaskManager   WorkSubtaskManager
	AgentPool        AgentPoolManager
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
		planManager:      cfg.PlanManager,
		taskManager:      cfg.TaskManager,
		subtaskManager:   cfg.SubtaskManager,
		agentPool:        cfg.AgentPool,
		contextReminders: contextReminders,
		toolCallHistory:  toolCallHistory,
		webSearchTool:    cfg.WebSearchTool,
		webFetchTool:     cfg.WebFetchTool,
		maxSteps:         maxSteps,
		agentConfig:      agentConfig,
		modelName:        cfg.ModelName,
		streaming:        cfg.Streaming,
		supervisorMode:   cfg.AgentPool != nil,
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
	s.propagateContextRemindersToPool()
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

	s.propagateContextRemindersToPool()
}

// propagateContextRemindersToPool sends current context reminders to AgentPool
// so Code Agents inherit environment context (project root, platform).
func (s *Service) propagateContextRemindersToPool() {
	if s.agentPool == nil {
		return
	}
	pool, ok := s.agentPool.(*AgentPool)
	if !ok {
		return
	}

	var reactReminders []react.ContextReminderProvider
	for _, r := range s.contextReminders {
		reactReminders = append(reactReminders, r)
	}
	pool.SetContextReminders(reactReminders)
}

// GetToolCallRecorder returns the tool call recorder for callback integration
func (s *Service) GetToolCallRecorder() ToolCallRecorder {
	return s.toolCallHistory
}

// GetContextReminders returns the context reminders for Engine integration
func (s *Service) GetContextReminders() []turn_executor.ContextReminderProvider {
	return s.contextReminders
}


package infrastructure

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/agents"
	licenseinfra "github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/license"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/llm"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/tools"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/websearch"
	agentservice "github.com/syntheticinc/bytebrew/bytebrew/engine/internal/service/agent"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/service/engine"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/service/work"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/pkg/config"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/pkg/errors"
	"github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
)

// InfraComponents holds all infrastructure components created during initialization
type InfraComponents struct {
	AgentService     *agentservice.Service
	WorkManager      *work.Manager
	AgentPool        *agentservice.AgentPool
	AgentPoolAdapter *agentservice.AgentPoolAdapter
	SessionStorage   *persistence.SQLiteSessionStorage
	ChatModel        model.ToolCallingChatModel // kept for backward compatibility
	ModelSelector    *llm.ModelSelector
	// Engine components
	Engine           *engine.Engine
	FlowManager      *agentservice.FlowManager
	ToolResolver     *tools.DefaultToolResolver
	ToolDepsProvider *tools.DefaultToolDepsProvider
	// Additional dependencies for TurnExecutorFactory
	ModelName        string
	WebSearchTool    einotool.InvokableTool
	WebFetchTool     einotool.InvokableTool
	AgentConfig      *config.AgentConfig // effective config with defaults applied
	LicenseInfo      *domain.LicenseInfo
}

// NewAgentService creates a new AgentService with Eino Agent
func NewAgentService(cfg config.Config) (*agentservice.Service, error) {
	components, err := NewInfraComponents(cfg)
	if err != nil {
		return nil, err
	}
	return components.AgentService, nil
}

// NewInfraComponents creates all infrastructure components including WorkManager and AgentPool
func NewInfraComponents(cfg config.Config) (*InfraComponents, error) {
	// 1. Create LLM model
	chatModel, err := createChatModel(cfg)
	if err != nil {
		return nil, err
	}

	modelName := getModelName(cfg)
	slog.Info("agent service initialized", "model", modelName, "provider", cfg.LLM.DefaultProvider)

	chatModel = wrapWithDebugModel(chatModel)
	modelSelector := createModelSelector(cfg, chatModel, modelName)

	// 2. Create work storage, agent pool, session storage
	storageCmp := createWorkStorage(cfg, modelSelector)

	var agentPool *agentservice.AgentPool
	var agentPoolAdapter *agentservice.AgentPoolAdapter
	if storageCmp.WorkManager != nil {
		agentPool = agentservice.NewAgentPool(agentservice.AgentPoolConfig{
			ModelSelector:   modelSelector,
			SubtaskManager:  storageCmp.WorkManager,
			AgentRunStorage: storageCmp.AgentRunStorage,
			AgentConfig:     &cfg.Agent,
		})
		agentPoolAdapter = agentservice.NewAgentPoolAdapter(agentPool)
		slog.Info("agent pool initialized")
	}

	// 4. License validation
	licenseInfo := validateLicense(cfg.License)
	slog.Info("license status",
		"tier", licenseInfo.Tier,
		"status", licenseInfo.Status,
		"expires", licenseInfo.ExpiresAt,
	)

	// 5. Fill empty AgentConfig fields with defaults
	agentConfig := applyAgentConfigDefaults(&cfg.Agent)

	// 6. Create web search/fetch tools
	webSearchTool, webFetchTool := createWebTools(cfg)

	// 7. Create Engine and wire to AgentPool
	ec, err := createEngine(cfg, storageCmp.WorkManager, agentPoolAdapter, webSearchTool, webFetchTool)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to initialize engine")
	}

	wireEngineToPool(agentPool, ec)

	// 7b. Create plan manager for AgentService
	planManager := createPlanStorage(cfg)

	// 8. Add security reminder (highest priority -- last in context for max recency bias)
	contextReminders := storageCmp.ContextReminders
	contextReminders = append(contextReminders, agents.NewSecurityReminderProvider())

	// 9. Create AgentService
	agentService, err := agentservice.New(agentservice.Config{
		ChatModel:        chatModel,
		PlanManager:      planManager,
		TaskManager:      storageCmp.WorkManager,
		SubtaskManager:   storageCmp.WorkManager,
		AgentPool:        agentPool,
		ContextReminders: contextReminders,
		WebSearchTool:    webSearchTool,
		WebFetchTool:     webFetchTool,
		MaxSteps:         cfg.Agent.MaxSteps,
		AgentConfig:      agentConfig,
		ModelName:        modelName,
		Streaming:        cfg.LLM.Streaming,
	})
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to create agent service")
	}

	slog.Info("agent service created with multi-agent support",
		"work_manager", storageCmp.WorkManager != nil,
		"agent_pool", agentPool != nil,
		"engine", ec.Engine != nil)

	return &InfraComponents{
		AgentService:     agentService,
		WorkManager:      storageCmp.WorkManager,
		AgentPool:        agentPool,
		AgentPoolAdapter: agentPoolAdapter,
		SessionStorage:   storageCmp.SessionStorage,
		ChatModel:        chatModel,
		ModelSelector:    modelSelector,
		Engine:           ec.Engine,
		FlowManager:      ec.FlowManager,
		ToolResolver:     ec.ToolResolver,
		ToolDepsProvider: ec.ToolDepsProvider,
		ModelName:        modelName,
		WebSearchTool:    webSearchTool,
		WebFetchTool:     webFetchTool,
		AgentConfig:      agentConfig,
		LicenseInfo:      licenseInfo,
	}, nil
}

// applyAgentConfigDefaults fills empty AgentConfig fields with defaults.
func applyAgentConfigDefaults(agentConfig *config.AgentConfig) *config.AgentConfig {
	defaultConfig := config.DefaultAgentConfig()

	if agentConfig.ContextLogPath == "" {
		agentConfig.ContextLogPath = defaultConfig.ContextLogPath
	}
	if agentConfig.MaxSteps == 0 {
		agentConfig.MaxSteps = defaultConfig.MaxSteps
	}
	if agentConfig.MaxContextSize == 0 {
		agentConfig.MaxContextSize = defaultConfig.MaxContextSize
	}
	if agentConfig.ToolReturnDirectly == nil {
		agentConfig.ToolReturnDirectly = defaultConfig.ToolReturnDirectly
	}

	return agentConfig
}

// createWebTools creates web search/fetch tools if API key is available.
func createWebTools(cfg config.Config) (einotool.InvokableTool, einotool.InvokableTool) {
	webSearchAPIKey := cfg.WebSearch.APIKey
	if webSearchAPIKey == "" {
		webSearchAPIKey = os.Getenv("TAVILY_API_KEY")
	}
	if cfg.WebSearch.Provider != "tavily" || webSearchAPIKey == "" {
		return nil, nil
	}

	provider := websearch.NewTavilyProvider(webSearchAPIKey)
	webSearchTool := tools.NewWebSearchTool(provider)
	webFetchTool := tools.NewWebFetchTool(provider)
	slog.Info("web search initialized", "provider", "tavily")
	return webSearchTool, webFetchTool
}

// validateLicense validates the license from config. Always returns a LicenseInfo (fallback to Blocked).
func validateLicense(cfg config.LicenseConfig) *domain.LicenseInfo {
	if cfg.PublicKeyHex == "" {
		return domain.BlockedLicense()
	}

	validator, err := licenseinfra.New(cfg.PublicKeyHex)
	if err != nil {
		slog.Error("invalid license public key, running as Blocked", "error", err)
		return domain.BlockedLicense()
	}

	licensePath := cfg.LicensePath
	if licensePath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			slog.Warn("failed to get home directory for license path", "error", err)
			return domain.BlockedLicense()
		}
		licensePath = filepath.Join(home, ".bytebrew", "license.jwt")
	}

	return validator.Validate(licensePath)
}

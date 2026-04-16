package app

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/agents"
	licenseinfra "github.com/syntheticinc/bytebrew/engine/internal/infrastructure/license"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/llm"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/configrepo"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/taskrunner"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/tools"
	agentservice "github.com/syntheticinc/bytebrew/engine/internal/service/agent"
	"github.com/syntheticinc/bytebrew/engine/internal/service/engine"
	"github.com/syntheticinc/bytebrew/engine/pkg/config"
	"github.com/syntheticinc/bytebrew/engine/pkg/errors"
	"github.com/cloudwego/eino/components/model"
	"gorm.io/gorm"
)

// InfraComponents holds all infrastructure components created during initialization
type InfraComponents struct {
	AgentService     *agentservice.Service
	TaskManager      *taskrunner.EngineTaskManagerAdapter
	TaskRepo         *configrepo.GORMTaskRepository
	AgentPool        *agentservice.AgentPool
	AgentPoolAdapter *agentservice.AgentPoolAdapter
	ChatModel        model.ToolCallingChatModel // kept for backward compatibility
	ModelSelector    *llm.ModelSelector
	// Engine components
	Engine            *engine.Engine
	FlowManager       *agentservice.FlowManager
	AgentToolResolver *tools.AgentToolResolver
	ToolDepsProvider  *tools.DefaultToolDepsProvider
	// Additional dependencies for TurnExecutorFactory
	ModelName   string
	ModelCache  *llm.ModelCache
	AgentConfig *config.AgentConfig // effective config with defaults applied
	LicenseInfo *domain.LicenseInfo
}

// InfraComponentsConfig holds optional parameters for NewInfraComponents.
// LicenseInfo is nil in CE mode (no restrictions).
type InfraComponentsConfig struct {
	Config      config.Config
	LicenseInfo *domain.LicenseInfo // nil = CE mode (all features enabled)
	DB          *gorm.DB            // PostgreSQL GORM DB for runtime storage
}

// NewInfraComponents creates all infrastructure components including WorkManager and AgentPool.
// License is passed via InfraComponentsConfig; nil means CE mode (no license checks).
func NewInfraComponents(icc InfraComponentsConfig) (*InfraComponents, error) {
	cfg := icc.Config

	// 1. Create LLM model
	chatModel, err := createChatModel(cfg)
	if err != nil {
		return nil, err
	}

	modelName := getModelName(cfg)
	slog.Info("agent service initialized", "model", modelName, "provider", cfg.LLM.DefaultProvider)

	chatModel = wrapWithDebugModel(chatModel)
	modelSelector := createModelSelector(cfg, chatModel, modelName)

	// 2. Create model cache (for dynamic model resolution from DB)
	var modelCache *llm.ModelCache
	if icc.DB != nil {
		modelCache = llm.NewModelCache(icc.DB)
	}

	// 3. Create work storage, agent pool, session storage
	storageCmp := createWorkStorage(icc.DB)

	var agentPool *agentservice.AgentPool
	var agentPoolAdapter *agentservice.AgentPoolAdapter
	if storageCmp.TaskManager != nil {
		agentPool = agentservice.NewAgentPool(agentservice.AgentPoolConfig{
			ModelSelector:   modelSelector,
			SubtaskManager:  storageCmp.TaskManager,
			AgentRunStorage: storageCmp.AgentRunStorage,
			AgentConfig:     &cfg.Agent,
		})
		agentPoolAdapter = agentservice.NewAgentPoolAdapter(agentPool)
		slog.Info("agent pool initialized")
	}

	// 4. License info (passed from caller; nil = CE mode)
	licenseInfo := icc.LicenseInfo
	if licenseInfo != nil {
		slog.Info("license status",
			"tier", licenseInfo.Tier,
			"status", licenseInfo.Status,
			"expires", licenseInfo.ExpiresAt,
		)
	} else {
		slog.Info("running in CE mode (no license)")
	}

	// 5. Fill empty AgentConfig fields with defaults
	agentConfig := applyAgentConfigDefaults(&cfg.Agent)

	// 6. Create Engine and wire to AgentPool
	ec, err := createEngine(cfg, icc.DB, storageCmp.TaskManager, agentPoolAdapter)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to initialize engine")
	}

	wireEngineToPool(agentPool, ec)

	// 7. Add security reminder (highest priority -- last in context for max recency bias)
	contextReminders := storageCmp.ContextReminders
	contextReminders = append(contextReminders, agents.NewSecurityReminderProvider())

	// 8. Create AgentService (optional — nil when no LLM configured in Docker/bootstrap mode)
	var agentService *agentservice.Service
	if chatModel != nil {
		var svcErr error
		agentService, svcErr = agentservice.New(agentservice.Config{
			ChatModel:        chatModel,
			AgentPool:        agentPool,
			ContextReminders: contextReminders,
			MaxSteps:         cfg.Agent.MaxSteps,
			AgentConfig:      agentConfig,
			ModelName:        modelName,
			Streaming:        cfg.LLM.Streaming,
		})
		if svcErr != nil {
			return nil, errors.Wrap(svcErr, errors.CodeInternal, "failed to create agent service")
		}
		slog.Info("agent service created with multi-agent support",
			"task_manager", storageCmp.TaskManager != nil,
			"agent_pool", agentPool != nil,
			"engine", ec.Engine != nil)
	} else {
		slog.Info("agent service skipped — no LLM model configured. Configure models via Admin Dashboard to enable chat.")
	}

	return &InfraComponents{
		AgentService:      agentService,
		TaskManager:       storageCmp.TaskManager,
		TaskRepo:          storageCmp.TaskRepo,
		AgentPool:         agentPool,
		AgentPoolAdapter:  agentPoolAdapter,
		ChatModel:         chatModel,
		ModelSelector:     modelSelector,
		Engine:            ec.Engine,
		FlowManager:       ec.FlowManager,
		AgentToolResolver: ec.AgentToolResolver,
		ToolDepsProvider:  ec.ToolDepsProvider,
		ModelName:         modelName,
		ModelCache:        modelCache,
		AgentConfig:       agentConfig,
		LicenseInfo:       licenseInfo,
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

// ValidateLicense validates the license from config. Always returns a LicenseInfo (fallback to Blocked).
// Called from legacy code. CE binary skips this entirely.
func ValidateLicense(cfg config.LicenseConfig) *domain.LicenseInfo {
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

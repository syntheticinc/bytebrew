package infrastructure

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/repository"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/tools"
	agentservice "github.com/syntheticinc/bytebrew/engine/internal/service/agent"
	"github.com/syntheticinc/bytebrew/engine/internal/service/engine"
	"github.com/syntheticinc/bytebrew/engine/internal/service/turn_executor"
	"github.com/syntheticinc/bytebrew/engine/internal/service/work"
	"github.com/syntheticinc/bytebrew/engine/pkg/config"
	"gorm.io/gorm"
)

// storageComponents holds all storage-related components created during initialization.
type storageComponents struct {
	WorkManager      *work.Manager
	SessionStorage   *persistence.SessionStorage
	AgentRunStorage  agentservice.AgentRunStorage
	ContextReminders []turn_executor.ContextReminderProvider
}

// createWorkStorage creates work manager, agent pool, session storage from pgDB.
func createWorkStorage(db *gorm.DB) *storageComponents {
	if db == nil {
		slog.Error("no database connection, multi-agent features disabled")
		return &storageComponents{}
	}
	return initWorkComponents(db)
}

// initWorkComponents initializes all work-related components from a GORM DB.
func initWorkComponents(db *gorm.DB) *storageComponents {
	ctx := context.Background()
	result := &storageComponents{}

	taskStorage := persistence.NewTaskStorage(db)
	subtaskStorage := persistence.NewSubtaskStorage(db)
	agentRunStorage := persistence.NewAgentRunStorage(db)
	result.AgentRunStorage = agentRunStorage

	sessionStorage := persistence.NewSessionStorage(db)
	result.SessionStorage = sessionStorage

	// Startup cleanup: orphaned agent runs and active sessions from previous crash
	cleaned, cleanErr := agentRunStorage.CleanupOrphanedRuns(ctx)
	if cleanErr != nil {
		slog.Error("failed to cleanup orphaned agent runs", "error", cleanErr)
	} else if cleaned > 0 {
		slog.Info("cleaned up orphaned agent runs from previous crash", "count", cleaned)
	}

	suspended, suspendErr := sessionStorage.SuspendActiveSessions(ctx)
	if suspendErr != nil {
		slog.Error("failed to suspend active sessions", "error", suspendErr)
	} else if suspended > 0 {
		slog.Info("suspended active sessions from previous crash", "count", suspended)
	}

	result.WorkManager = work.New(taskStorage, subtaskStorage)
	slog.Info("work manager initialized")

	// Create context reminder for work status
	workReminder := work.NewWorkContextReminder(result.WorkManager)
	result.ContextReminders = append(result.ContextReminders, workReminder)

	return result
}

// engineComponents holds Engine and its associated dependencies.
type engineComponents struct {
	Engine            *engine.Engine
	FlowManager       *agentservice.FlowManager
	AgentToolResolver *tools.AgentToolResolver
	ToolDepsProvider  *tools.DefaultToolDepsProvider
}

// createEngine creates Engine, FlowManager, ToolResolver and ToolDepsProvider.
// Uses the shared PostgreSQL database for message and context snapshot storage.
func createEngine(
	cfg config.Config,
	db *gorm.DB,
	workManager *work.Manager,
	agentPoolAdapter *agentservice.AgentPoolAdapter,
	webSearchTool, webFetchTool einotool.InvokableTool,
) (*engineComponents, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection required for engine")
	}

	snapshotRepo := repository.NewAgentContextRepository(db)
	messageRepo := repository.NewMessageRepositoryImpl(db)
	agentEngine := engine.New(snapshotRepo, messageRepo)
	slog.Info("engine initialized (PostgreSQL)")

	// Load flows.yaml (optional — not required in bootstrap/Docker mode)
	flowsPath := filepath.Join(cfg.ConfigDir, "flows.yaml")
	flowsCfg, err := config.LoadFlowsConfig(flowsPath)
	if err != nil {
		slog.Info("No flows.yaml found — using empty flows config (configure agents via Admin Dashboard)", "path", flowsPath)
		flowsCfg = &config.FlowsConfig{}
	}

	flowManager, err := agentservice.NewFlowManager(flowsCfg, cfg.Agent.Prompts)
	if err != nil {
		return nil, fmt.Errorf("create flow manager: %w", err)
	}
	slog.Info("flow manager initialized", "flows_path", flowsPath)

	// Create ToolDepsProvider
	toolDepsProvider := tools.NewDefaultToolDepsProvider(
		nil, // proxy -- set dynamically per-session
		workManager,
		workManager,
		agentPoolAdapter,
		webSearchTool,
		webFetchTool,
	)

	// Create AgentToolResolver (factory-based tool resolution)
	builtinStore := tools.NewBuiltinToolStore()
	tools.RegisterAllBuiltins(builtinStore)

	// Register spawn_code_agent separately (requires AgentPool, wired later via wireEngineToPool)
	if agentPoolAdapter != nil {
		builtinStore.Register("spawn_code_agent", func(deps tools.ToolDependencies) einotool.InvokableTool {
			return tools.NewSpawnCodeAgentTool(deps.AgentPool, deps.SessionID, deps.ProjectKey)
		})
	}

	agentToolResolver := tools.NewAgentToolResolver(builtinStore)
	slog.Info("agent tool resolver initialized", "builtin_tools", len(builtinStore.Names()))

	return &engineComponents{
		Engine:            agentEngine,
		FlowManager:       flowManager,
		AgentToolResolver: agentToolResolver,
		ToolDepsProvider:  toolDepsProvider,
	}, nil
}

// wireEngineToPool connects Engine to AgentPool and configures max concurrency.
func wireEngineToPool(
	agentPool *agentservice.AgentPool,
	ec *engineComponents,
) {
	if agentPool == nil || ec == nil {
		return
	}

	agentPool.SetEngine(ec.Engine, ec.FlowManager, ec.AgentToolResolver, ec.ToolDepsProvider)
	slog.Info("engine wired to agent pool")

	// Set MaxConcurrent from supervisor flow (legacy: uses "supervisor" as default flow for spawn config)
	ctx := context.Background()
	supervisorFlow, err := ec.FlowManager.GetFlow(ctx, domain.FlowType("supervisor"))
	if err != nil {
		slog.Warn("failed to get supervisor flow for MaxConcurrent config", "error", err)
		return
	}
	if supervisorFlow.Spawn.MaxConcurrent > 0 {
		agentPool.SetMaxConcurrent(supervisorFlow.Spawn.MaxConcurrent)
		slog.Info("max concurrent agents configured", "limit", supervisorFlow.Spawn.MaxConcurrent)
	}
}

// NewRuntimeDB is a convenience function that returns the existing pgDB reference.
// Kept for backward compatibility where a separate runtime DB handle was expected.
func NewRuntimeDB(db *gorm.DB) *gorm.DB {
	return db
}

// MigrateRuntimeTables runs migration for runtime-only tables.
// Called separately from models.AutoMigrate if needed.
func MigrateRuntimeTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.RuntimeSessionModel{},
		&models.RuntimeTaskModel{},
		&models.RuntimeSubtaskModel{},
		&models.RuntimeAgentRunModel{},
		&models.RuntimeDeviceModel{},
		&models.RuntimeConfigKV{},
		&models.RuntimeSessionEventModel{},
	)
}

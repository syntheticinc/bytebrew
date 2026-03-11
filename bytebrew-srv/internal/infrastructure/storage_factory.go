package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/agents"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/persistence"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/persistence/repository"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/tools"
	agentservice "github.com/syntheticinc/bytebrew/bytebrew-srv/internal/service/agent"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/service/engine"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/service/turn_executor"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/service/work"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/config"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// storageComponents holds all storage-related components created during initialization.
type storageComponents struct {
	WorkManager      *work.Manager
	SessionStorage   *persistence.SQLiteSessionStorage
	AgentRunStorage  agentservice.AgentRunStorage
	ContextReminders []turn_executor.ContextReminderProvider
}

// createPlanStorage creates plan manager with SQLite or memory-only storage.
func createPlanStorage(cfg config.Config) *agents.PlanManager {
	planDBPath := cfg.PlanStorage.DBPath
	if planDBPath == "" {
		planDBPath = "./data/plans.db"
	}
	planStorage, err := persistence.NewSQLitePlanStorage(planDBPath)
	if err != nil {
		slog.Error("failed to create plan storage, plans will not be persisted", "error", err)
		planStorage = nil
	}
	if planStorage != nil {
		slog.Info("plan storage initialized", "db_path", planDBPath)
	}

	planManager := agents.NewPlanManager(planStorage)
	if planStorage != nil {
		slog.Info("plan manager initialized with SQLite storage")
	}
	if planStorage == nil {
		slog.Warn("plan manager initialized without storage (memory-only)")
	}

	return planManager
}

// createWorkStorage creates work DB, work manager, agent pool, session storage.
func createWorkStorage(cfg config.Config, modelSelector agentservice.AgentModelSelector) *storageComponents {
	workDBPath := cfg.WorkStorage.DBPath
	if workDBPath == "" {
		workDBPath = "./data/work.db"
	}

	workDB, err := persistence.NewWorkDB(workDBPath)
	if err != nil {
		slog.Error("failed to create work DB, multi-agent features disabled", "error", err)
		return &storageComponents{}
	}
	slog.Info("work DB initialized", "db_path", workDBPath)

	return initWorkComponents(workDB, modelSelector, &cfg.Agent)
}

// initWorkComponents initializes all work-related components from a work DB.
func initWorkComponents(workDB *sql.DB, modelSelector agentservice.AgentModelSelector, agentCfg *config.AgentConfig) *storageComponents {
	ctx := context.Background()
	result := &storageComponents{}

	taskStorage, err := persistence.NewSQLiteTaskStorage(workDB)
	if err != nil {
		slog.Error("failed to create task storage", "error", err)
	}

	subtaskStorage, err := persistence.NewSQLiteSubtaskStorage(workDB)
	if err != nil {
		slog.Error("failed to create subtask storage", "error", err)
	}

	agentRunStorage, err := persistence.NewSQLiteAgentRunStorage(workDB)
	if err != nil {
		slog.Error("failed to create agent run storage", "error", err)
	}
	result.AgentRunStorage = agentRunStorage

	sessionStorage, err := persistence.NewSQLiteSessionStorage(workDB)
	if err != nil {
		slog.Error("failed to create session storage", "error", err)
	}
	result.SessionStorage = sessionStorage

	// Startup cleanup: orphaned agent runs and active sessions from previous crash
	if agentRunStorage != nil {
		cleaned, cleanErr := agentRunStorage.CleanupOrphanedRuns(ctx)
		if cleanErr != nil {
			slog.Error("failed to cleanup orphaned agent runs", "error", cleanErr)
		} else if cleaned > 0 {
			slog.Info("cleaned up orphaned agent runs from previous crash", "count", cleaned)
		}
	}

	if sessionStorage != nil {
		suspended, suspendErr := sessionStorage.SuspendActiveSessions(ctx)
		if suspendErr != nil {
			slog.Error("failed to suspend active sessions", "error", suspendErr)
		} else if suspended > 0 {
			slog.Info("suspended active sessions from previous crash", "count", suspended)
		}
	}

	if taskStorage != nil && subtaskStorage != nil {
		result.WorkManager = work.New(taskStorage, subtaskStorage)
		slog.Info("work manager initialized")

		// Create context reminder for work status
		workReminder := work.NewWorkContextReminder(result.WorkManager)
		result.ContextReminders = append(result.ContextReminders, workReminder)
	}

	return result
}

// engineComponents holds Engine and its associated dependencies.
type engineComponents struct {
	Engine           *engine.Engine
	FlowManager      *agentservice.FlowManager
	ToolResolver     *tools.DefaultToolResolver
	ToolDepsProvider *tools.DefaultToolDepsProvider
}

// createEngine creates Engine, FlowManager, ToolResolver and ToolDepsProvider.
func createEngine(
	cfg config.Config,
	workManager *work.Manager,
	agentPoolAdapter *agentservice.AgentPoolAdapter,
	webSearchTool, webFetchTool einotool.InvokableTool,
) (*engineComponents, error) {
	// 1. Create engine DB (snapshot + message storage)
	engineDBPath := "./data/engine.db"
	if err := os.MkdirAll(filepath.Dir(engineDBPath), 0755); err != nil {
		return nil, fmt.Errorf("create engine data directory: %w", err)
	}

	gormDB, err := gorm.Open(sqlite.Open(engineDBPath), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open engine DB: %w", err)
	}

	if err := createEngineTables(gormDB); err != nil {
		return nil, fmt.Errorf("migrate engine DB: %w", err)
	}

	snapshotRepo := repository.NewAgentContextRepository(gormDB)
	messageRepo := repository.NewMessageRepositoryImpl(gormDB)
	agentEngine := engine.New(snapshotRepo, messageRepo)
	slog.Info("engine initialized", "db_path", engineDBPath)

	// 2. Load flows.yaml
	flowsPath := filepath.Join(cfg.ConfigDir, "flows.yaml")
	flowsCfg, err := config.LoadFlowsConfig(flowsPath)
	if err != nil {
		return nil, fmt.Errorf("load flows config: %w", err)
	}

	flowManager, err := agentservice.NewFlowManager(flowsCfg, cfg.Agent.Prompts)
	if err != nil {
		return nil, fmt.Errorf("create flow manager: %w", err)
	}
	slog.Info("flow manager initialized", "flows_path", flowsPath)

	// 3. Create ToolResolver and ToolDepsProvider
	toolResolver := tools.NewDefaultToolResolver()
	toolDepsProvider := tools.NewDefaultToolDepsProvider(
		nil, // proxy -- set dynamically per-session
		workManager,
		workManager,
		agentPoolAdapter,
		webSearchTool,
		webFetchTool,
	)

	return &engineComponents{
		Engine:           agentEngine,
		FlowManager:      flowManager,
		ToolResolver:     toolResolver,
		ToolDepsProvider: toolDepsProvider,
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

	agentPool.SetEngine(ec.Engine, ec.FlowManager, ec.ToolResolver, ec.ToolDepsProvider)
	slog.Info("engine wired to agent pool")

	// Set MaxConcurrent from supervisor flow
	ctx := context.Background()
	supervisorFlow, err := ec.FlowManager.GetFlow(ctx, domain.FlowTypeSupervisor)
	if err != nil {
		slog.Warn("failed to get supervisor flow for MaxConcurrent config", "error", err)
		return
	}
	if supervisorFlow.Spawn.MaxConcurrent > 0 {
		agentPool.SetMaxConcurrent(supervisorFlow.Spawn.MaxConcurrent)
		slog.Info("max concurrent agents configured", "limit", supervisorFlow.Spawn.MaxConcurrent)
	}
}

// createEngineTables creates SQLite-compatible tables for Engine storage.
// We use raw SQL instead of GORM AutoMigrate because the shared models.Message
// has PostgreSQL-specific defaults (gen_random_uuid) and foreign key relationships
// that are incompatible with SQLite.
func createEngineTables(db *gorm.DB) error {
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS agent_context_snapshot (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		agent_id TEXT NOT NULL,
		flow_type TEXT NOT NULL,
		schema_version INTEGER NOT NULL DEFAULT 1,
		context_data BLOB NOT NULL,
		step_number INTEGER NOT NULL DEFAULT 0,
		token_count INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'active',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`).Error; err != nil {
		return fmt.Errorf("create agent_context_snapshot table: %w", err)
	}

	db.Exec(`CREATE INDEX IF NOT EXISTS idx_snap_session_agent ON agent_context_snapshot(session_id, agent_id)`)
	db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_snap_agent_unique ON agent_context_snapshot(agent_id)`)

	if err := db.Exec(`CREATE TABLE IF NOT EXISTS message (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		message_type TEXT NOT NULL,
		sender TEXT,
		agent_id TEXT,
		content TEXT NOT NULL,
		metadata TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`).Error; err != nil {
		return fmt.Errorf("create message table: %w", err)
	}

	db.Exec(`CREATE INDEX IF NOT EXISTS idx_msg_session ON message(session_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_msg_session_agent ON message(session_id, agent_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_session_created ON message(created_at)`)

	return nil
}

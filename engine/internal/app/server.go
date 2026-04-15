// Package app provides common server setup shared between CE and server (legacy) entry points.
package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"sync/atomic"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/syntheticinc/bytebrew/engine/internal/delivery/grpc"
	deliveryhttp "github.com/syntheticinc/bytebrew/engine/internal/delivery/http"
	"github.com/syntheticinc/bytebrew/engine/internal/delivery/ws"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/embedded"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/agentregistry"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/taskrunner"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/turnexecutorfactory"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/versioncheck"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/audit"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/bridge"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/flowregistry"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/indexing"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/knowledge"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/mcp"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/configrepo"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	admintools "github.com/syntheticinc/bytebrew/engine/internal/infrastructure/tools/admin"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/llm/registry"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/portfile"

	mcpcatalog "github.com/syntheticinc/bytebrew/engine/internal/service/mcp"
	svcschematemplate "github.com/syntheticinc/bytebrew/engine/internal/service/schematemplate"
	ucschematemplate "github.com/syntheticinc/bytebrew/engine/internal/usecase/schematemplate"
	"github.com/syntheticinc/bytebrew/engine/internal/service/capability"
	svcknowledge "github.com/syntheticinc/bytebrew/engine/internal/service/knowledge"
	"github.com/syntheticinc/bytebrew/engine/internal/service/eventstore"
	"github.com/syntheticinc/bytebrew/engine/internal/service/lifecycle"
	"github.com/syntheticinc/bytebrew/engine/internal/service/guardrail"

	"github.com/syntheticinc/bytebrew/engine/internal/service/recovery"
	"github.com/syntheticinc/bytebrew/engine/internal/service/resilience"
	"github.com/syntheticinc/bytebrew/engine/internal/service/sessionprocessor"
	"github.com/syntheticinc/bytebrew/engine/internal/service/turnexecutor"
	"github.com/syntheticinc/bytebrew/engine/pkg/config"
	"github.com/syntheticinc/bytebrew/engine/pkg/logger"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// LicenseProvider gives read access to the current license and its atomic pointer.
// Implemented by license.LicenseWatcher for live-reloading, or nil for CE mode.
type LicenseProvider interface {
	// Current returns the latest validated license, or nil (CE mode).
	Current() *domain.LicenseInfo
	// Pointer returns the atomic pointer for use by HTTP middleware.
	Pointer() *atomic.Pointer[domain.LicenseInfo]
	// Stop terminates the background refresh goroutine.
	Stop()
}

// ServerConfig holds parameters for Run that differ between CE and server (legacy).
type ServerConfig struct {
	// ConfigPath is the path to the config file (resolved by the caller).
	ConfigPath string

	// ConfigExplicit is true when --config was explicitly provided on the command line.
	ConfigExplicit bool

	// Port overrides the config port (0 = use config or random).
	Port int

	// Managed enables managed subprocess mode (random port, READY protocol).
	Managed bool

	// BridgeURL overrides the bridge WebSocket URL from config.
	BridgeURL string

	// LicenseInfo is the validated license. nil = CE mode (no restrictions).
	LicenseInfo *domain.LicenseInfo

	// LicenseProvider enables live license reloading for EE middleware.
	// nil = CE mode (no EE gating). When set, LicenseInfo is also populated
	// from its Current() at startup for backward compatibility with gRPC/WS.
	LicenseProvider LicenseProvider

	// Version, Commit, Date are build-time metadata.
	Version string
	Commit  string
	Date    string
}

// Run starts the ByteBrew server with the given configuration.
// This is the common entry point shared by CE and server (legacy) binaries.
func Run(sc ServerConfig) error {
	// Always resolve data dir (needed for port file discovery)
	dataDir := UserDataDir()
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}

	configPath := sc.ConfigPath

	// In managed mode, create additional subdirs and override paths
	if sc.Managed {
		if err := ensureManagedDirs(dataDir); err != nil {
			return fmt.Errorf("create managed directories: %w", err)
		}

		// If --config was not explicitly provided, use config from data dir
		if !sc.ConfigExplicit {
			managedConfigPath := filepath.Join(dataDir, "config.yaml")
			if _, err := os.Stat(managedConfigPath); os.IsNotExist(err) {
				if err := generateDefaultConfig(managedConfigPath, sc.LicenseInfo != nil); err != nil {
					return fmt.Errorf("generate default config: %w", err)
				}
				log.Printf("Generated default config at %s", managedConfigPath)
			}
			configPath = managedConfigPath
		}

		// Generate default prompts.yaml if missing (from embedded)
		managedPromptsPath := filepath.Join(dataDir, "prompts.yaml")
		if _, err := os.Stat(managedPromptsPath); os.IsNotExist(err) {
			if err := os.WriteFile(managedPromptsPath, embedded.DefaultPrompts, 0644); err != nil {
				return fmt.Errorf("write default prompts: %w", err)
			}
			log.Printf("Generated default prompts at %s", managedPromptsPath)
		}

		// Generate default flows.yaml if missing (from embedded)
		managedFlowsPath := filepath.Join(dataDir, "flows.yaml")
		if _, err := os.Stat(managedFlowsPath); os.IsNotExist(err) {
			if err := os.WriteFile(managedFlowsPath, embedded.DefaultFlows, 0644); err != nil {
				return fmt.Errorf("write default flows: %w", err)
			}
			log.Printf("Generated default flows at %s", managedFlowsPath)
		}
	}

	// Get working directory for config path resolution
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// Resolve config path relative to working directory
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(wd, configPath)
	}

	// Load configuration — if config file doesn't exist, use defaults (env-var mode for Docker)
	var cfg *config.Config
	if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) && !sc.ConfigExplicit {
		log.Printf("No config file at %s — using defaults (configure via environment variables or Admin Dashboard)", configPath)
		cfg = config.DefaultConfig()
	} else {
		var loadErr error
		cfg, loadErr = config.Load(configPath)
		if loadErr != nil {
			return fmt.Errorf("load config: %w", loadErr)
		}
		log.Printf("Config loaded: default_provider=%s, ollama_model=%s", cfg.LLM.DefaultProvider, cfg.LLM.Ollama.Model)
	}

	// Override bridge config from flag
	if sc.BridgeURL != "" {
		cfg.Bridge.URL = sc.BridgeURL
		cfg.Bridge.Enabled = true
	}

	// Check for already running server BEFORE touching log files.
	portReader := portfile.NewReader(dataDir)
	existingInfo, _ := portReader.Read()
	if existingInfo != nil {
		// Skip check if the recorded PID is our own process (Docker restart scenario)
		if existingInfo.PID != os.Getpid() && portfile.IsProcessAlive(existingInfo.PID) {
			return fmt.Errorf("server already running (PID %d, port %d). Kill it first or use a different config",
				existingInfo.PID, existingInfo.Port)
		}
		// Stale port file from a crashed/killed server — clean up.
		stalePortFile := filepath.Join(dataDir, "server.port")
		if err := os.Remove(stalePortFile); err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: failed to remove stale port file: %v", err)
		} else {
			log.Printf("Removed stale port file (PID %d no longer running)", existingInfo.PID)
		}
	}

	// Apply managed mode overrides
	if sc.Managed {
		cfg.Logging.FilePath = filepath.Join(dataDir, "logs", "server.log")
		cfg.Server.Port = sc.Port
	}

	// Clear old logs if configured
	if cfg.Logging.ClearOnStartup {
		logsDir := filepath.Dir(cfg.Logging.FilePath)
		if logsDir == "" || logsDir == "." {
			logsDir = "logs"
		}
		removedCount, err := logger.ClearLogsDir(logsDir)
		if err != nil {
			log.Printf("Warning: failed to clear logs directory: %v", err)
		} else if removedCount > 0 {
			log.Printf("Cleared %d old log files from %s", removedCount, logsDir)
		}
	}

	// Initialize logger
	loggerInstance, err := logger.New(cfg.Logging)
	if err != nil {
		return fmt.Errorf("initialize logger: %w", err)
	}

	slog.SetDefault(loggerInstance.Logger)

	// Start pprof HTTP server for diagnostics
	go func() {
		pprofAddr := "localhost:6060"
		slog.Info("pprof server started", "addr", pprofAddr)
		if err := http.ListenAndServe(pprofAddr, nil); err != nil {
			slog.Error("pprof server failed", "error", err)
		}
	}()

	ctx := context.Background()
	loggerInstance.InfoContext(ctx, "Starting ByteBrew Server",
		"version", sc.Version,
		"commit", sc.Commit,
		"built", sc.Date,
		"config", configPath,
	)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Try loading bootstrap config for PostgreSQL database connection.
	var agentRegistry *agentregistry.AgentRegistry
	var pgDB *gorm.DB
	var taskRepo *configrepo.GORMTaskRepository
	var apiTokenRepo *configrepo.GORMAPITokenRepository
	bootstrapCfg, bootstrapErr := config.LoadBootstrap(configPath)
	if bootstrapErr != nil {
		slog.Info("No bootstrap database config, running in legacy mode", "reason", bootstrapErr.Error())
	} else {
		var pgErr error
		pgDB, pgErr = gorm.Open(postgres.Open(bootstrapCfg.Database.URL), &gorm.Config{
			Logger: gormlogger.Default.LogMode(gormlogger.Silent),
		})
		if pgErr != nil {
			return fmt.Errorf("connect to PostgreSQL: %w", pgErr)
		}

		if migrateErr := models.AutoMigrate(pgDB); migrateErr != nil {
			return fmt.Errorf("run database migrations: %w", migrateErr)
		}

		agentRepo := configrepo.NewGORMAgentRepository(pgDB)
		taskRepo = configrepo.NewGORMTaskRepository(pgDB)
		apiTokenRepo = configrepo.NewGORMAPITokenRepository(pgDB)
		agentRegistry = agentregistry.New(agentRepo)
		if loadErr := agentRegistry.Load(ctx); loadErr != nil {
			return fmt.Errorf("load agents from database: %w", loadErr)
		}

		agentCount := agentRegistry.Count()
		if agentCount > 0 {
			slog.InfoContext(ctx, "Loaded agents from database", "count", agentCount, "agents", agentRegistry.List())
		} else {
			slog.InfoContext(ctx, "No agents configured in database")
		}
	}

	// If no LLM configured in legacy config but models exist in DB, use the first one.
	if cfg.LLM.DefaultProvider == "" && pgDB != nil {
		var firstModel models.LLMProviderModel
		if err := pgDB.First(&firstModel).Error; err == nil {
			slog.InfoContext(ctx, "Auto-configuring LLM from database model",
				"name", firstModel.Name, "provider", firstModel.Type, "model", firstModel.ModelName)
			switch firstModel.Type {
			case "ollama":
				cfg.LLM.DefaultProvider = "ollama"
				cfg.LLM.Ollama.Model = firstModel.ModelName
				cfg.LLM.Ollama.BaseURL = firstModel.BaseURL
			case "openai", "openai_compatible":
				cfg.LLM.DefaultProvider = "openrouter"
				cfg.LLM.OpenRouter.Model = firstModel.ModelName
				cfg.LLM.OpenRouter.APIKey = firstModel.APIKeyEncrypted
				if firstModel.BaseURL != "" {
					cfg.LLM.OpenRouter.BaseURL = firstModel.BaseURL
				}
			case "anthropic":
				cfg.LLM.DefaultProvider = "anthropic"
				cfg.LLM.Anthropic.Model = firstModel.ModelName
				cfg.LLM.Anthropic.APIKey = firstModel.APIKeyEncrypted
			}
		}
	}

	// Create infrastructure components (AgentService + WorkManager + AgentPool)
	components, err := NewInfraComponents(InfraComponentsConfig{
		Config:      *cfg,
		LicenseInfo: sc.LicenseInfo,
		DB:          pgDB,
	})
	if err != nil {
		return fmt.Errorf("create infrastructure components: %w", err)
	}

	// Knowledge indexing infrastructure (created before HTTP so endpoints can use it)
	var knowledgeRepo *configrepo.GORMKnowledgeRepository
	var knowledgeIndexer *knowledge.Indexer
	var embeddingsClient *indexing.EmbeddingsClient
	if pgDB != nil {
		knowledgeRepo = configrepo.NewGORMKnowledgeRepository(pgDB)
		embedURL := indexing.DefaultOllamaURL
		if envURL := os.Getenv("EMBED_URL"); envURL != "" {
			embedURL = envURL
		}
		embedModel := indexing.DefaultEmbedModel
		if envModel := os.Getenv("EMBED_MODEL"); envModel != "" {
			embedModel = envModel
		}
		embedDim := indexing.DefaultDimension
		if envDim := os.Getenv("EMBED_DIM"); envDim != "" {
			if d, err := strconv.Atoi(envDim); err == nil && d > 0 {
				embedDim = d
			}
		}
		embeddingsClient = indexing.NewEmbeddingsClient(
			embedURL,
			embedModel,
			embedDim,
		)
		knowledgeIndexer = knowledge.NewIndexer(embeddingsClient, knowledgeRepo, slog.Default())
	}

	// Initialize MCP client connections from database
	mcpRegistry := mcp.NewClientRegistry()
	var forwardHeadersStore atomic.Value // shared with configReloaderHTTPAdapter for dynamic updates
	forwardHeadersStore.Store([]string(nil))

	// Seed builder-assistant and its MCP server BEFORE connectMCPServers,
	// so the seeded bytebrew-docs MCP server is included in the first connect pass.
	if pgDB != nil {
		seedByteBrewDocsMCP(ctx, pgDB)
		seedBuilderAssistant(ctx, pgDB)
		seedBuilderSchema(ctx, pgDB)
		// V2 Commit Group C (§5.5): the system-wide MCP catalog is now a
		// DB table populated from mcp-catalog.yaml at boot via upsert.
		seedMCPCatalog(ctx, pgDB)
		// V2 Commit Group L (§2.2): schema starter templates catalog is a
		// DB table populated from schema-templates.yaml at boot via upsert.
		seedSchemaTemplates(ctx, pgDB)
	}

	if pgDB != nil {
		mcpServerRepo := configrepo.NewGORMMCPServerRepository(pgDB)
		mcpServers, mcpErr := mcpServerRepo.List(ctx)
		if mcpErr != nil {
			slog.Warn("failed to load MCP servers from database", "error", mcpErr)
		} else {
			connectMCPServers(ctx, mcpServers, mcpRegistry)
			forwardHeadersStore.Store(collectForwardHeaders(mcpServers))
		}
	}

	// Wire MCP provider into AgentToolResolver
	if components.AgentToolResolver != nil {
		components.AgentToolResolver.SetMCPProvider(mcpRegistry)
	}

	// Register admin tools and reload registry.
	if pgDB != nil && agentRegistry != nil {

		// Wire admin tools into builtin store for builder-assistant.
		if components.AgentToolResolver != nil {
			admintools.RegisterAdminTools(components.AgentToolResolver.BuiltinStore(), admintools.AdminToolDependencies{
				AgentRepo:      newAdminAgentRepoAdapter(configrepo.NewGORMAgentRepository(pgDB)),
				SchemaRepo:     newAdminSchemaRepoAdapter(configrepo.NewGORMSchemaRepository(pgDB)),
				TriggerRepo:    newAdminTriggerRepoAdapter(configrepo.NewGORMTriggerRepository(pgDB), pgDB),
				MCPServerRepo:  newAdminMCPServerRepoAdapter(configrepo.NewGORMMCPServerRepository(pgDB)),
				ModelRepo:      newAdminModelRepoAdapter(configrepo.NewGORMLLMProviderRepository(pgDB)),
				AgentRelationRepo: newAdminAgentRelationRepoAdapter(configrepo.NewGORMAgentRelationRepository(pgDB)),
				SessionRepo:    newAdminSessionRepoAdapter(configrepo.NewGORMSessionRepository(pgDB)),
				CapabilityRepo: newAdminCapabilityRepoAdapter(configrepo.NewGORMCapabilityRepository(pgDB)),
				Reloader: func() {
					if agentRegistry != nil {
						if err := agentRegistry.Load(context.Background()); err != nil {
							slog.Warn("admin tools: failed to reload registry", "error", err)
						}
					}
				},
			})
			slog.InfoContext(ctx, "admin tools registered into builtin store")
		}

		// Reload registry so the seeded builder-assistant is available at runtime.
		if err := agentRegistry.Load(ctx); err != nil {
			slog.WarnContext(ctx, "failed to reload agent registry after seed", "error", err)
		}
	}

	// Wire knowledge search into AgentToolResolver.
	// embeddingsClient may be nil (no Ollama) — per-agent resolver provides embedding models.
	if components.AgentToolResolver != nil && knowledgeRepo != nil {
		components.AgentToolResolver.SetKnowledge(knowledgeRepo, embeddingsClient)
		if pgDB != nil {
			components.AgentToolResolver.SetKnowledgeEmbedderResolver(
				&knowledgeEmbedderResolverAdapter{resolver: &embeddingModelResolver{db: pgDB}})
			components.AgentToolResolver.SetKnowledgeKBResolver(
				configrepo.NewGORMKnowledgeBaseRepository(pgDB))
		}
	}

	// Wire spawner into AgentToolResolver for HTTP chat path spawn support.
	// CompositeAgentSpawner routes spawn requests based on agent lifecycle mode:
	// "spawn" agents → pool (unchanged), "persistent" agents → lifecycle.Manager.
	var lifecycleManager *lifecycle.Manager
	var lifecycleDispatcher *lifecycle.Dispatcher
	var agentLifecycleReader AgentLifecycleReader
	if components.AgentPoolAdapter != nil && agentRegistry != nil {
		agentLifecycleReader = newAgentRegistryLifecycleAdapter(agentRegistry)
		poolRunner := &poolBasedRunner{pool: components.AgentPoolAdapter}
		lifecycleManager = lifecycle.NewManager(poolRunner)
		lifecycleDispatcher = lifecycle.NewDispatcher(lifecycleManager)

		if components.AgentToolResolver != nil {
			compositeSpawner := NewCompositeAgentSpawner(
				components.AgentPoolAdapter,
				lifecycleManager,
				agentLifecycleReader,
			)
			components.AgentToolResolver.SetSpawner(compositeSpawner, components.AgentPoolAdapter)
			slog.InfoContext(ctx, "CompositeAgentSpawner wired into AgentToolResolver")
		}
	}

	// US-001: Wire capability injector into AgentToolResolver
	if components.AgentToolResolver != nil && pgDB != nil {
		capRepo := configrepo.NewGORMCapabilityRepository(pgDB)
		injector := capability.NewInjector(&capabilityInjectorAdapter{repo: capRepo})
		components.AgentToolResolver.SetCapabilityInjector(injector)
		slog.InfoContext(ctx, "Capability injector wired into AgentToolResolver")
	}

	// US-004: Wire dynamic policy evaluator — loads rules from capabilities DB per-agent.
	if components.AgentToolResolver != nil && pgDB != nil {
		components.AgentToolResolver.SetPolicyEvaluator(&dynamicPolicyEvaluatorAdapter{db: pgDB})
		slog.InfoContext(ctx, "Dynamic policy evaluator wired into AgentToolResolver")
	}

	// US-006: Wire circuit breaker registry into AgentToolResolver
	var cbRegistry *resilience.CircuitBreakerRegistry
	if components.AgentToolResolver != nil {
		cbRegistry = resilience.NewCircuitBreakerRegistry(resilience.DefaultCircuitBreakerConfig())
		components.AgentToolResolver.SetCircuitBreakerRegistry(&circuitBreakerRegistryAdapter{registry: cbRegistry})
		slog.InfoContext(ctx, "Circuit breaker registry wired into AgentToolResolver")

		// Wire 30s default tool timeout into AgentToolResolver (AC-RESIL-05)
		components.AgentToolResolver.SetToolTimeout(30_000) // 30 seconds in ms
		slog.InfoContext(ctx, "Tool timeout wired into AgentToolResolver", "timeout_ms", 30000)
	}

	// Resilience: HeartbeatMonitor — detects stuck agents (AC-RESIL-01/02)
	heartbeatMonitor := resilience.NewHeartbeatMonitor(resilience.DefaultHeartbeatConfig(), stubHeartbeatCallback)
	heartbeatMonitor.Start(ctx)
	slog.InfoContext(ctx, "Heartbeat monitor started")

	// Resilience: DeadLetterQueue — tracks timed-out tasks (AC-RESIL-07/08)
	deadLetterQueue := resilience.NewDeadLetterQueue(resilience.DefaultDeadLetterConfig(), func(t resilience.TrackedTask, elapsed time.Duration) {
		slog.WarnContext(ctx, "task timed out, moved to dead letter",
			"task_id", t.TaskID, "agent_id", t.AgentID, "elapsed", elapsed)
	})

	// US-005: Wire recovery executor into AgentToolResolver
	if components.AgentToolResolver != nil {
		recoveryExec := recovery.New(nil) // nil recorder — events logged via slog
		components.AgentToolResolver.SetRecoveryExecutor(&recoveryExecutorAdapter{executor: recoveryExec})
		slog.InfoContext(ctx, "Recovery executor wired into AgentToolResolver")
	}

	// Wire per-agent capability config reader (recovery recipes, memory max_entries, knowledge top_k)
	var capReader *capabilityConfigReader
	if pgDB != nil {
		capReader = &capabilityConfigReader{db: pgDB}
		if components.AgentToolResolver != nil {
			components.AgentToolResolver.SetCapabilityConfigReader(capReader)
			slog.InfoContext(ctx, "Capability config reader wired into AgentToolResolver")
		}
	}

	// US-003: Guardrail pipeline — wired into factory after factory creation (see below).
	guardrailPipeline := guardrail.NewPipeline()
	_ = guardrailPipeline

	// HTTP REST API server — starts only when bootstrap config is available.
	// Supports two modes:
	//   Single-port (default): all routes on one port (backward compatible)
	//   Two-port: external (data plane) + internal (control plane)
	var httpServer *deliveryhttp.Server         // main server (single-port) or external (two-port)
	var internalHTTPServer *deliveryhttp.Server  // nil in single-port mode
	var httpPort int
	var internalHTTPPort int
	var httpAuthMW *deliveryhttp.AuthMiddleware
	var configurableRL *deliveryhttp.ConfigurableRateLimiter
	if agentRegistry != nil && bootstrapCfg != nil {
		httpPort = bootstrapCfg.Engine.Port
		if httpPort == 0 {
			httpPort = 8443
		}
		internalHTTPPort = bootstrapCfg.Engine.InternalPort // 0 = single-port mode

		if internalHTTPPort > 0 {
			// Two-port mode: external gets configurable CORS, internal gets permissive CORS
			httpServer = deliveryhttp.NewServerWithCORS(httpPort, bootstrapCfg.Engine.CORSOrigins)
			internalHTTPServer = deliveryhttp.NewServer(internalHTTPPort)
		} else {
			// Single-port mode (backward compatible)
			httpServer = deliveryhttp.NewServer(httpPort)
		}
		r := httpServer.Router()
		// internalRouter is the router for management/admin routes.
		// In single-port mode it points to the same router as r.
		// In two-port mode it points to the internal server's router.
		internalRouter := r
		if internalHTTPServer != nil {
			internalRouter = internalHTTPServer.Router()
		}

		// Metrics middleware — records request count and duration for all routes.
		// Applied before auth so every request is instrumented regardless of auth status.
		r.Use(deliveryhttp.MetricsMiddleware)
		if internalHTTPServer != nil {
			internalRouter.Use(deliveryhttp.MetricsMiddleware)
		}

		// Auth
		jwtSecret := bootstrapCfg.Security.AdminPassword
		authMW := deliveryhttp.NewAuthMiddleware(jwtSecret, &tokenRepoHTTPAdapter{repo: apiTokenRepo})
		httpAuthMW = authMW

		// Audit logger
		auditLogger := audit.NewLogger(pgDB)

		// Update checker (non-blocking, air-gap safe)
		updateChecker := versioncheck.NewUpdateChecker(sc.Version)
		updateChecker.Start(ctx)

		// Health (public) — available on both ports
		healthHandler := deliveryhttp.NewHealthHandler(sc.Version, &agentCounterHTTPAdapter{registry: agentRegistry})
		healthHandler.SetUpdateChecker(updateChecker)
		r.Get("/api/v1/health", healthHandler.ServeHTTP)

		// Model registry (public — read-only catalog, no auth needed)
		modelRegistry := registry.New()
		registryHandler := deliveryhttp.NewModelRegistryHandler(modelRegistry)

		// Auth login (public)
		authHandler := deliveryhttp.NewAuthHandler(
			bootstrapCfg.Security.AdminUser,
			bootstrapCfg.Security.AdminPassword,
			jwtSecret,
		)

		if internalHTTPServer != nil {
			// Two-port mode: register public routes on internal router too
			internalRouter.Get("/api/v1/health", healthHandler.ServeHTTP)
			internalRouter.Get("/api/v1/models/registry", registryHandler.List)
			internalRouter.Get("/api/v1/models/registry/providers", registryHandler.ListProviders)
			internalRouter.Post("/api/v1/auth/login", authHandler.Login)
		}
		// Single-port or external: model registry + login on main router
		r.Get("/api/v1/models/registry", registryHandler.List)
		r.Get("/api/v1/models/registry/providers", registryHandler.ListProviders)
		r.Post("/api/v1/auth/login", authHandler.Login)

		// Protected management routes — on internalRouter (= r in single-port mode)
		internalRouter.Group(func(r chi.Router) {
			r.Use(authMW.Authenticate)
			r.Use(deliveryhttp.AuditMiddleware(&auditHTTPAdapter{logger: auditLogger}))

			// Schema repo (created early because agent manager needs it for used_in_schemas)
			schemaRepo := configrepo.NewGORMSchemaRepository(pgDB)

			// Agents
			agentRepo := configrepo.NewGORMAgentRepository(pgDB)
			agentManager := &agentManagerHTTPAdapter{repo: agentRepo, registry: agentRegistry, db: pgDB, schemaRepo: schemaRepo}
			agentHandler := deliveryhttp.NewAgentHandlerWithManager(agentManager)
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeAgentsRead))
				r.Get("/api/v1/agents", agentHandler.List)
				r.Get("/api/v1/agents/{name}", agentHandler.Get)
			})
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeAgentsWrite))
				r.Post("/api/v1/agents", agentHandler.Create)
				r.Put("/api/v1/agents/{name}", agentHandler.Update)
				r.Delete("/api/v1/agents/{name}", agentHandler.Delete)
			})

			// Agent Lifecycle
			if lifecycleManager != nil && agentLifecycleReader != nil {
				lifecycleProvider := newLifecycleHTTPAdapter(lifecycleManager, agentLifecycleReader)
				lifecycleHandler := deliveryhttp.NewLifecycleHandler(lifecycleProvider)
				r.Group(func(r chi.Router) {
					r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeAgentsRead))
					r.Get("/api/v1/agents/{name}/lifecycle", lifecycleHandler.Status)
				})
			}

			// Agent Capabilities
			capRepo := configrepo.NewGORMCapabilityRepository(pgDB)
			capHandler := deliveryhttp.NewCapabilityHandler(&capabilityServiceHTTPAdapter{repo: capRepo})
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeAgentsRead))
				r.Get("/api/v1/agents/{name}/capabilities", capHandler.List)
			})
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeAgentsWrite))
				r.Post("/api/v1/agents/{name}/capabilities", capHandler.Add)
				r.Put("/api/v1/agents/{name}/capabilities/{capId}", capHandler.Update)
				r.Delete("/api/v1/agents/{name}/capabilities/{capId}", capHandler.Remove)
			})

			// Models
			llmProviderRepo := configrepo.NewGORMLLMProviderRepository(pgDB)
			modelService := &modelServiceHTTPAdapter{repo: llmProviderRepo, modelCache: components.ModelCache}
			modelHandler := deliveryhttp.NewModelHandler(modelService)
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeModelsRead))
				r.Get("/api/v1/models", modelHandler.List)
			})
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeModelsWrite))
				r.Post("/api/v1/models", modelHandler.Create)
				r.Put("/api/v1/models/{name}", modelHandler.Update)
				r.Delete("/api/v1/models/{name}", modelHandler.Delete)
				r.Post("/api/v1/models/{name}/verify", modelHandler.Verify)
			})

			// Tasks
			taskHandler := deliveryhttp.NewTaskHandler(&taskServiceHTTPAdapter{
				repo:    taskRepo,
				manager: components.TaskManager,
			})
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeTasks))
				r.Post("/api/v1/tasks", taskHandler.Create)
				r.Get("/api/v1/tasks", taskHandler.List)
				r.Get("/api/v1/tasks/{id}", taskHandler.Get)
				r.Delete("/api/v1/tasks/{id}", taskHandler.Cancel)
				r.Get("/api/v1/tasks/{id}/subtasks", taskHandler.ListSubtasks)
				r.Post("/api/v1/tasks/{id}/approve", taskHandler.Approve)
				r.Post("/api/v1/tasks/{id}/start", taskHandler.Start)
				r.Post("/api/v1/tasks/{id}/complete", taskHandler.Complete)
				r.Post("/api/v1/tasks/{id}/fail", taskHandler.Fail)
				r.Post("/api/v1/tasks/{id}/priority", taskHandler.SetPriority)
			})

			// Dispatch Tasks (lifecycle dispatcher queries)
			if lifecycleDispatcher != nil {
				dispatchHandler := deliveryhttp.NewDispatchHandler(lifecycleDispatcher)
				r.Group(func(r chi.Router) {
					r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeTasks))
					r.Get("/api/v1/dispatch/tasks/{taskId}", dispatchHandler.Get)
					r.Get("/api/v1/sessions/{sessionId}/dispatch-tasks", dispatchHandler.ListBySession)
				})
			}

			// Config
			configHandler := deliveryhttp.NewConfigHandler(
				&configReloaderHTTPAdapter{registry: agentRegistry, mcpRegistry: mcpRegistry, db: pgDB, forwardHeadersStore: &forwardHeadersStore},
				&configImportExportHTTPAdapter{db: pgDB},
			)
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeConfig))
				r.Post("/api/v1/config/reload", configHandler.Reload)
				r.Post("/api/v1/config/import", configHandler.Import)
				r.Get("/api/v1/config/export", configHandler.Export)
			})

			// Knowledge
			if knowledgeRepo != nil {
				var reindexer deliveryhttp.KnowledgeReindexer
				if knowledgeIndexer != nil {
					reindexer = &knowledgeReindexerHTTPAdapter{
						indexer:  knowledgeIndexer,
						registry: agentRegistry,
					}
				}
				knowledgeHandler := deliveryhttp.NewKnowledgeHandler(
					&knowledgeStatsHTTPAdapter{repo: knowledgeRepo},
					reindexer,
				)

				dataDir := "data"
				if envDir := os.Getenv("DATA_DIR"); envDir != "" {
					dataDir = envDir
				}

				uploadSvc := svcknowledge.NewUploadService(knowledgeRepo, dataDir)
				uploadSvc.SetEmbeddingResolver(&embeddingModelResolver{db: pgDB})
				uploadSvc.SetKBEmbeddingResolver(&kbEmbeddingResolver{db: pgDB})
				knowledgeHandler.SetFileUploader(&knowledgeUploadHTTPAdapter{svc: uploadSvc})
				knowledgeHandler.SetFileLister(&knowledgeFileListerHTTPAdapter{svc: uploadSvc})

				// Knowledge Bases (many-to-many) handler
				kbRepo := configrepo.NewGORMKnowledgeBaseRepository(pgDB)
				kbHandler := deliveryhttp.NewKnowledgeBaseHandler(
					&kbStoreAdapter{repo: kbRepo, db: pgDB},
					&kbFileManagerAdapter{svc: uploadSvc},
				)

				// Legacy agent-scoped knowledge endpoints
				r.Group(func(r chi.Router) {
					r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeAgentsRead))
					r.Get("/api/v1/agents/{name}/knowledge/status", knowledgeHandler.Status)
					r.Get("/api/v1/agents/{name}/knowledge/files", knowledgeHandler.ListFiles)
				})
				r.Group(func(r chi.Router) {
					r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeAgentsWrite))
					r.Post("/api/v1/agents/{name}/knowledge/reindex", knowledgeHandler.Reindex)
					r.Post("/api/v1/agents/{name}/knowledge/files", knowledgeHandler.UploadFile)
					r.Delete("/api/v1/agents/{name}/knowledge/files/{file_id}", knowledgeHandler.DeleteFile)
					r.Post("/api/v1/agents/{name}/knowledge/files/{file_id}/reindex", knowledgeHandler.ReindexFile)
				})

				// Knowledge Base CRUD + file management endpoints
				r.Group(func(r chi.Router) {
					r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeAgentsRead))
					r.Get("/api/v1/knowledge-bases", kbHandler.List)
					r.Get("/api/v1/knowledge-bases/{id}", kbHandler.Get)
					r.Get("/api/v1/knowledge-bases/{id}/files", kbHandler.ListFiles)
				})
				r.Group(func(r chi.Router) {
					r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeAgentsWrite))
					r.Post("/api/v1/knowledge-bases", kbHandler.Create)
					r.Put("/api/v1/knowledge-bases/{id}", kbHandler.Update)
					r.Delete("/api/v1/knowledge-bases/{id}", kbHandler.Delete)
					r.Post("/api/v1/knowledge-bases/{id}/agents/{agent_name}", kbHandler.LinkAgent)
					r.Delete("/api/v1/knowledge-bases/{id}/agents/{agent_name}", kbHandler.UnlinkAgent)
					r.Post("/api/v1/knowledge-bases/{id}/files", kbHandler.UploadFile)
					r.Delete("/api/v1/knowledge-bases/{id}/files/{file_id}", kbHandler.DeleteFile)
					r.Post("/api/v1/knowledge-bases/{id}/files/{file_id}/reindex", kbHandler.ReindexFile)
				})
			}

			// Audit log READ API — always registered so Admin UI doesn't get 404.
			// Returns 403 "EE required" when no license is active.
			auditRepo := configrepo.NewGORMAuditRepository(pgDB)
			auditHandler := deliveryhttp.NewAuditHandler(&auditServiceHTTPAdapter{repo: auditRepo})
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireAdminSession)
				if sc.LicenseProvider != nil {
					eeMWAudit := deliveryhttp.NewEEMiddleware(sc.LicenseProvider.Pointer())
					r.Use(eeMWAudit.RequireEE)
				} else {
					r.Use(func(next http.Handler) http.Handler {
						return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							w.Header().Set("Content-Type", "application/json")
							w.WriteHeader(http.StatusForbidden)
							_, _ = w.Write([]byte(`{"error":"Enterprise Edition license required","upgrade_url":"https://bytebrew.ai/billing"}`))
						})
					})
				}
				r.Get("/api/v1/audit", auditHandler.List)
			})

			// API Tokens (admin-only)
			tokenHandler := deliveryhttp.NewTokenHandler(&tokenRepoHTTPAdapter{repo: apiTokenRepo})
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireAdminSession)
				r.Post("/api/v1/auth/tokens", tokenHandler.CreateToken)
				r.Get("/api/v1/auth/tokens", tokenHandler.ListTokens)
				r.Delete("/api/v1/auth/tokens/{id}", tokenHandler.DeleteToken)
			})

			// MCP Servers
			mcpServerRepo := configrepo.NewGORMMCPServerRepository(pgDB)
			mcpHandler := deliveryhttp.NewMCPHandler(&mcpServiceHTTPAdapter{repo: mcpServerRepo})
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeMCPRead))
				r.Get("/api/v1/mcp-servers", mcpHandler.List)
			})
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeMCPWrite))
				r.Post("/api/v1/mcp-servers", mcpHandler.Create)
				r.Put("/api/v1/mcp-servers/{name}", mcpHandler.Update)
				r.Delete("/api/v1/mcp-servers/{name}", mcpHandler.Delete)
			})

			// Triggers
			triggerRepo := configrepo.NewGORMTriggerRepository(pgDB)
			triggerHandler := deliveryhttp.NewTriggerHandler(&triggerServiceHTTPAdapter{repo: triggerRepo, db: pgDB})
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeTriggersRead))
				r.Get("/api/v1/triggers", triggerHandler.List)
			})
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeTriggersWrite))
				r.Post("/api/v1/triggers", triggerHandler.Create)
				r.Put("/api/v1/triggers/{id}", triggerHandler.Update)
				r.Delete("/api/v1/triggers/{id}", triggerHandler.Delete)
				r.Patch("/api/v1/triggers/{id}/target", triggerHandler.SetTarget)
				r.Delete("/api/v1/triggers/{id}/target", triggerHandler.ClearTarget)
			})

			// Schemas (with agent_relations) — schemaRepo already created above for agent cross-refs.
			// V2: edges→agent_relations rename + drop type column (Group A.1).
			// Gates removed in V2 (see docs/architecture/agent-first-runtime.md §3).
			agentRelationRepo := configrepo.NewGORMAgentRelationRepository(pgDB)
			schemaHandler := deliveryhttp.NewSchemaHandler(
				&schemaServiceHTTPAdapter{repo: schemaRepo},
				&agentRelationServiceHTTPAdapter{repo: agentRelationRepo},
			)
			schemaHandler.SetAgentDetailer(agentManager)
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeSchemasRead))
				r.Get("/api/v1/schemas", schemaHandler.ListSchemas)
				r.Get("/api/v1/schemas/{id}", schemaHandler.GetSchema)
				r.Get("/api/v1/schemas/{id}/agents", schemaHandler.ListSchemaAgents)
				r.Get("/api/v1/schemas/{id}/agent-relations", schemaHandler.ListAgentRelations)
				r.Get("/api/v1/schemas/{id}/agent-relations/{relationId}", schemaHandler.GetAgentRelation)
				r.Get("/api/v1/schemas/{id}/export", schemaHandler.ExportSchema)
			})
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeSchemasWrite))
				r.Post("/api/v1/schemas", schemaHandler.CreateSchema)
				r.Post("/api/v1/schemas/import", schemaHandler.ImportSchema)
				r.Put("/api/v1/schemas/{id}", schemaHandler.UpdateSchema)
				r.Delete("/api/v1/schemas/{id}", schemaHandler.DeleteSchema)
				// V2: schema membership is derived from agent_relations
				// (docs/architecture/agent-first-runtime.md §2.1) — the
				// POST/DELETE schema-agent routes are gone; create or
				// remove a delegation relation to add or remove a member.
				r.Post("/api/v1/schemas/{id}/agent-relations", schemaHandler.CreateAgentRelation)
				r.Put("/api/v1/schemas/{id}/agent-relations/{relationId}", schemaHandler.UpdateAgentRelation)
				r.Delete("/api/v1/schemas/{id}/agent-relations/{relationId}", schemaHandler.DeleteAgentRelation)
			})

			// Widgets: V2 removes the server-side widgets entity. The admin
			// UI is a pure snippet generator (docs/architecture/agent-first-runtime.md
			// §4.3); no /api/v1/widgets routes are registered.

			// Settings (admin-only)
			settingRepo := configrepo.NewGORMSettingRepository(pgDB)
			settingHandler := deliveryhttp.NewSettingHandler(&settingServiceHTTPAdapter{repo: settingRepo})
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireAdminSession)
				r.Get("/api/v1/settings", settingHandler.List)
				r.Put("/api/v1/settings/{key}", settingHandler.Update)
			})

			// Builder-assistant restore (admin-only)
			baHandler := deliveryhttp.NewBuilderAssistantHandler(&builderAssistantRestorerAdapter{db: pgDB, registry: agentRegistry})
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireAdminSession)
				r.Post("/api/v1/admin/builder-assistant/restore", baHandler.Restore)
			})

			// Sessions (admin-only)
			sessionRepo := configrepo.NewGORMSessionRepository(pgDB)
			messageRepo := configrepo.NewGORMMessageRepository(pgDB)
			sessionHandler := deliveryhttp.NewSessionHandler(&sessionServiceHTTPAdapter{repo: sessionRepo, messageRepo: messageRepo})
			sessionHandler.SetEventService(&eventServiceHTTPAdapter{repo: messageRepo})
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireAdminSession)
				r.Mount("/api/v1/sessions", sessionHandler.Routes())
			})

			// Tool metadata (admin-only)
			toolMetaHandler := deliveryhttp.NewToolMetadataHandler(&toolMetadataHTTPAdapter{})
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireAdminSession)
				r.Get("/api/v1/tools/metadata", toolMetaHandler.List)
			})

			// Memory (per-schema)
			memoryStorage := persistence.NewMemoryStorage(pgDB)
			memoryHandler := deliveryhttp.NewMemoryHandler(
				&memoryListerHTTPAdapter{storage: memoryStorage},
				&memoryClearerHTTPAdapter{storage: memoryStorage},
			)
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeSchemasRead))
				r.Get("/api/v1/schemas/{id}/memory", memoryHandler.ListMemories)
			})
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeSchemasWrite))
				r.Delete("/api/v1/schemas/{id}/memory", memoryHandler.ClearMemories)
				r.Delete("/api/v1/schemas/{id}/memory/{entry_id}", memoryHandler.DeleteMemory)
			})

			// MCP Catalog (read-only) — DB-backed (V2 Commit Group C, §5.5).
			if pgDB != nil {
				catalogRepo := configrepo.NewGORMMCPCatalogRepository(pgDB)
				catalogSvc := mcpcatalog.NewCatalogService(catalogRepo)
				catalogHandler := deliveryhttp.NewCatalogHandler(catalogSvc)
				r.Get("/api/v1/mcp/catalog", catalogHandler.ListCatalog)
			}

			// Schema templates catalog + fork — DB-backed (V2 Commit Group L, §2.2).
			// Reads are open to any authenticated user; fork requires the
			// schemas-write scope (it creates new schemas/agents/triggers).
			if pgDB != nil {
				tmplRepo := configrepo.NewGORMSchemaTemplateRepository(pgDB)
				forkSvc := svcschematemplate.NewForkService(pgDB, tmplRepo)
				forkAdapter := svcschematemplate.NewUsecaseForkerAdapter(forkSvc)
				tmplUC := ucschematemplate.New(tmplRepo, forkAdapter)
				tmplHandler := deliveryhttp.NewSchemaTemplateHandler(tmplUC, "1.0")
				r.Get("/api/v1/schema-templates", tmplHandler.List)
				r.Get("/api/v1/schema-templates/{name}", tmplHandler.Get)
				r.Group(func(r chi.Router) {
					r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeSchemasWrite))
					r.Post("/api/v1/schema-templates/{name}/fork", tmplHandler.Fork)
				})
			}

			// Usage (CE mode — unlimited)
			usageHandler := deliveryhttp.NewUsageHandler()
			r.Get("/api/v1/usage", usageHandler.GetUsage)

			// Resilience admin endpoints (AC-RESIL-08: dead letters visible, circuit breaker management)
			resilienceHandler := deliveryhttp.NewResilienceHandler(
				&circuitBreakerQuerierHTTPAdapter{registry: cbRegistry},
				&deadLetterQuerierHTTPAdapter{queue: deadLetterQueue},
				&heartbeatQuerierHTTPAdapter{monitor: heartbeatMonitor},
			)
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireAdminSession)
				r.Get("/api/v1/admin/resilience/circuit-breakers", resilienceHandler.ListCircuitBreakers)
				r.Post("/api/v1/admin/resilience/circuit-breakers/{name}/reset", resilienceHandler.ResetCircuitBreaker)
				r.Get("/api/v1/admin/resilience/dead-letters", resilienceHandler.ListDeadLetters)
				r.Get("/api/v1/admin/resilience/heartbeats", resilienceHandler.ListHeartbeats)
			})

			// Capability injector is wired into AgentToolResolver above (US-001).
			// capRepo is also used here for capability CRUD HTTP handlers.
		})

		// EE-only routes (require auth + valid Enterprise license) — on internal router.
		if sc.LicenseProvider != nil {
			eeMW := deliveryhttp.NewEEMiddleware(sc.LicenseProvider.Pointer())

			// Prometheus /metrics endpoint (EE, no auth — Prometheus scrapes without tokens).
			internalRouter.Group(func(r chi.Router) {
				r.Use(eeMW.RequireEE)
				r.Handle("/metrics", promhttp.Handler())
			})

			// Configurable rate limiter (EE) — per-header rate limiting
			rateLimitRules := cfg.RateLimits
			if len(rateLimitRules) == 0 {
				// Fallback: read rate limits from env var (Docker/env-based deployments)
				if envRL := os.Getenv("BYTEBREW_RATE_LIMITS"); envRL != "" {
					var envRules []config.RateLimitRule
					if err := json.Unmarshal([]byte(envRL), &envRules); err != nil {
						slog.Warn("failed to parse BYTEBREW_RATE_LIMITS env var", "error", err)
					} else {
						rateLimitRules = envRules
					}
				}
			}
			if len(rateLimitRules) > 0 {
				rules := convertRateLimitRules(rateLimitRules)
				configurableRL = deliveryhttp.NewConfigurableRateLimiter(rules, sc.LicenseProvider.Pointer())
			}

			internalRouter.Group(func(r chi.Router) {
				r.Use(authMW.Authenticate)
				r.Use(eeMW.RequireEE)

				// Tool call audit log (EE) — detailed per-tool-call log
				toolCallRepo := configrepo.NewToolCallEventRepository(pgDB)
				toolCallLogHandler := deliveryhttp.NewToolCallLogHandler(&toolCallLogHTTPAdapter{repo: toolCallRepo})
				r.Get("/api/v1/audit/tool-calls", toolCallLogHandler.List)

				// Rate limit usage API (EE)
				if configurableRL != nil {
					usageHandler := deliveryhttp.NewRateLimitUsageHandler(configurableRL)
					r.Get("/api/v1/rate-limits/usage", usageHandler.Usage)
				}
			})
		}

		// Webhook route (internal only — triggered by external services, requires network access)
		//
		// V2 (§4.1): this endpoint resolves the incoming path against
		// `triggers.config->>'webhook_path'`, stamps the trigger's
		// last_fired_at, and creates the task with the trigger's UUID in
		// SourceID so the admin UI can trace the run back to its channel.
		// Paths without a matching enabled trigger return 404.
		webhookTriggerRepo := configrepo.NewGORMTriggerRepository(pgDB)
		internalRouter.Post("/api/v1/webhooks/{path}", func(w http.ResponseWriter, req *http.Request) {
			webhookPath := chi.URLParam(req, "path")
			w.Header().Set("Content-Type", "application/json")

			trigger, err := webhookTriggerRepo.FindByWebhookPath(req.Context(), "/"+webhookPath)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"resolve webhook trigger: ` + err.Error() + `"}`))
				return
			}
			if trigger == nil {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error":"no enabled webhook trigger for path"}`))
				return
			}

			var body struct {
				Title       string `json:"title"`
				Description string `json:"description"`
				Message     string `json:"message"`
			}
			_ = json.NewDecoder(req.Body).Decode(&body)

			title := trigger.Title
			if title == "" {
				title = "Webhook: /" + webhookPath
			}
			if body.Title != "" {
				title = body.Title
			}
			description := trigger.Description
			if body.Description != "" {
				description = body.Description
			}
			if body.Message != "" && description == "" {
				description = body.Message
			}
			agentName := trigger.Agent.Name
			if agentName == "" {
				agentName = "supervisor"
			}

			t := &domain.EngineTask{
				Title:       title,
				Description: description,
				AgentName:   agentName,
				Source:      domain.TaskSourceWebhook,
				SourceID:    trigger.ID,
				Status:      domain.EngineTaskStatusPending,
				Mode:        domain.TaskModeBackground,
			}

			if err := taskRepo.Create(req.Context(), t); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"` + err.Error() + `"}`))
				return
			}

			// §4.1: stamp last_fired_at after validation + task creation so the
			// admin UI reflects the most recent fire. Non-fatal on error — the
			// task row is what the operator really needs.
			if markErr := webhookTriggerRepo.MarkFired(req.Context(), trigger.ID); markErr != nil {
				slog.WarnContext(req.Context(), "mark webhook trigger fired failed", "trigger_id", trigger.ID, "error", markErr)
			}

			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(fmt.Sprintf(`{"task_id":"%s"}`, t.ID)))
		})

		// Serve Admin Dashboard SPA (static files) — internal only
		adminDir := "/usr/share/bytebrew/admin"
		if _, statErr := os.Stat(adminDir); statErr == nil {
			spaFS := http.Dir(adminDir)
			adminFileHandler := func(w http.ResponseWriter, req *http.Request) {
				filePath := strings.TrimPrefix(req.URL.Path, "/admin")
				if filePath == "" || filePath == "/" {
					filePath = "/index.html"
				}
				if _, err := os.Stat(filepath.Join(adminDir, filePath)); os.IsNotExist(err) {
					http.ServeFile(w, req, filepath.Join(adminDir, "index.html"))
					return
				}
				http.StripPrefix("/admin", http.FileServer(spaFS)).ServeHTTP(w, req)
			}
			adminRedirect := func(w http.ResponseWriter, req *http.Request) {
				http.Redirect(w, req, "/admin/", http.StatusMovedPermanently)
			}
			internalRouter.Get("/admin/*", adminFileHandler)
			internalRouter.Get("/admin", adminRedirect)
			slog.InfoContext(ctx, "Admin Dashboard served", "path", adminDir)
		} else {
			slog.InfoContext(ctx, "Admin Dashboard not found (optional)", "path", adminDir)
		}

		// Serve Web Client SPA (static files) — internal only
		webclientDir := "/usr/share/bytebrew/webclient"
		if _, statErr := os.Stat(webclientDir); statErr == nil {
			chatSpaFS := http.Dir(webclientDir)
			chatFileHandler := func(w http.ResponseWriter, req *http.Request) {
				filePath := strings.TrimPrefix(req.URL.Path, "/chat")
				if filePath == "" || filePath == "/" {
					filePath = "/index.html"
				}
				if _, err := os.Stat(filepath.Join(webclientDir, filePath)); os.IsNotExist(err) {
					http.ServeFile(w, req, filepath.Join(webclientDir, "index.html"))
					return
				}
				http.StripPrefix("/chat", http.FileServer(chatSpaFS)).ServeHTTP(w, req)
			}
			chatRedirect := func(w http.ResponseWriter, req *http.Request) {
				http.Redirect(w, req, "/chat/", http.StatusMovedPermanently)
			}
			internalRouter.Get("/chat/*", chatFileHandler)
			internalRouter.Get("/chat", chatRedirect)
			slog.InfoContext(ctx, "Web Client served", "path", webclientDir)
		}

		// Serve widget.js (static file) — external only (or both in single-port mode).
		// V2: the admin generates a <script src="…/widget.js" data-agent="…" …>
		// snippet client-side (docs/architecture/agent-first-runtime.md §4.3),
		// so only the static bundle is served here — no dynamic /widget/{id}.js
		// bootstrap endpoint and no server-side widget configuration.
		widgetPath := "/usr/share/bytebrew/widget/widget.js"
		if _, statErr := os.Stat(widgetPath); statErr == nil {
			widgetFileHandler := func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "application/javascript")
				w.Header().Set("Cache-Control", "public, max-age=3600")
				http.ServeFile(w, req, widgetPath)
			}
			r.Get("/widget.js", widgetFileHandler)
			slog.InfoContext(ctx, "Widget served", "path", widgetPath)
		}

		// NOTE: HTTP server start is deferred until after SessionProcessor is created,
		// so the chat endpoint can be wired with all required dependencies.
	}

	// Initialize gRPC server.
	// When HTTP REST API is active (bootstrap mode), gRPC uses a random port
	// to avoid port conflicts. CLI discovers the gRPC port via the port file.
	grpcUsesRandomPort := httpServer != nil
	if grpcUsesRandomPort {
		cfg.Server.Port = 0 // force random port for gRPC
	}
	grpcServer, err := initializeGRPCServer(cfg, loggerInstance, sc.LicenseInfo, sc.Managed || grpcUsesRandomPort)
	if err != nil {
		return fmt.Errorf("initialize gRPC server: %w", err)
	}

	// Create flow registry for managing active flows
	flowRegistry := flowregistry.NewInMemoryRegistry()

	// Create event store (PostgreSQL) for reliable event replay on reconnect
	eventStore, err := eventstore.New(pgDB)
	if err != nil {
		return fmt.Errorf("create event store: %w", err)
	}

	// Create session registry for server-streaming API and bridge
	sessionRegistry := flowregistry.NewSessionRegistry(eventStore)

	// Create FlowHandler with multi-agent support
	pingInterval := 2 * time.Second
	flowHandlerCfg := grpc.FlowHandlerConfig{
		AgentService: components.AgentService,
		PingInterval: pingInterval,
		FlowRegistry: flowRegistry,
		SessionRegistry: sessionRegistry,
	}
	if components.AgentService != nil {
		flowHandlerCfg.ToolCallHistoryCleaner = components.AgentService.GetToolCallHistoryReminder()
	}

	// Engine components are always available
	// Use AgentRegistry as FlowProvider (replaces legacy FlowManager for agent resolution)
	var flowProvider turnexecutor.FlowProvider = components.FlowManager
	if agentRegistry != nil {
		flowProvider = agentRegistry
	}
	// Resolve AgentModelResolver (nil-safe: factory handles nil gracefully)
	var agentModelResolver turnexecutorfactory.AgentModelResolver
	if agentRegistry != nil {
		agentModelResolver = agentRegistry
	}

	factory := turnexecutorfactory.New(
		components.Engine,
		flowProvider,
		components.AgentToolResolver,
		components.ModelSelector,
		components.AgentConfig,
		components.AgentPoolAdapter,
		func() []turnexecutor.ContextReminderProvider {
			if components.AgentService != nil {
				return components.AgentService.GetContextReminders()
			}
			return nil
		},
		components.ModelCache,
		agentModelResolver,
	)
	flowHandlerCfg.TurnExecutorFactory = factory

	// platformTriggerRepo is shared across platform services: completion hook (wired
	// inside the pgDB block below) and cron scheduler (wired after sessProcessor).
	// Declared at this scope so both consumers can see it.
	var platformTriggerRepo *configrepo.GORMTriggerRepository

	// Wire memory storage into factory for memory_recall/memory_store tools (US-001 Memory capability)
	if pgDB != nil {
		memStorage := persistence.NewMemoryStorage(pgDB)
		factory.SetMemory(memStorage, memStorage, 0) // maxEntries=0 means unlimited
		loggerInstance.InfoContext(ctx, "Memory storage wired into TurnExecutorFactory")

		// BUG-007: Wire schema resolver so memory/knowledge tools get SchemaID (UUID).
		factory.SetSchemaResolver(&agentSchemaIDResolver{db: pgDB})
		loggerInstance.InfoContext(ctx, "Schema resolver wired into TurnExecutorFactory")

		// Wire EngineTaskManager so agents use DB-backed tasks (visible in Admin)
		factory.SetEngineTaskManager(components.TaskManager)
		loggerInstance.InfoContext(ctx, "EngineTaskManager wired into TurnExecutorFactory")

		// Platform services: the cron scheduler wires later, once sessProcessor
		// exists so the executor can actually run the agent.
		//
		// V2 (§4.2): the on-complete webhook feature is removed. Task terminal
		// transitions do not fan out to external URLs — if that use case
		// returns it will be expressed as an MCP webhook tool call from the
		// agent. last_fired_at is driven by TriggerRepository.MarkFired from
		// cron / webhook / chat dispatchers.
		if taskRepo != nil {
			platformTriggerRepo = configrepo.NewGORMTriggerRepository(pgDB)
		}

		// US-003: Wire guardrail pipeline with per-agent config resolver
		factory.SetGuardrail(
			&guardrailCheckerAdapter{pipeline: guardrailPipeline},
			&guardrailConfigResolver{db: pgDB},
		)
		loggerInstance.InfoContext(ctx, "Guardrail pipeline wired into TurnExecutorFactory")

		// Wire per-agent capability config reader for memory max_entries
		if capReader != nil {
			factory.SetCapabilityConfigReader(capReader)
		}
	}

	// Create shared SessionProcessor
	sessProcessor := sessionprocessor.New(sessionRegistry, factory, eventStore)
	flowHandlerCfg.SessionProcessor = sessProcessor

	// Autonomous task executor: cron/webhook triggers create a task, the worker picks it
	// up, opens a session scoped to the trigger's schema, runs the agent, records the
	// final answer. Wiring lives here because the executor needs sessProcessor and the
	// session registry, which are built just above.
	if taskRepo != nil && platformTriggerRepo != nil {
		taskExecutor := taskrunner.NewTaskExecutor(
			components.TaskManager,
			sessionRegistry,
			sessProcessor,
			0, // 0 → DefaultTaskTimeout
		)
		taskWorker := taskrunner.StartBackgroundWorker(taskExecutor, 4)
		if taskWorker != nil {
			defer taskWorker.Stop()
		}

		cronScheduler, cronErr := taskrunner.StartCronScheduler(ctx, platformTriggerRepo, components.TaskManager, taskWorker)
		if cronErr != nil {
			loggerInstance.WarnContext(ctx, "cron scheduler failed to start", "error", cronErr)
		} else if cronScheduler != nil {
			// Stop the scheduler when the server exits so in-flight ticks are not left hanging.
			defer cronScheduler.Stop()
		}
	}

	// Wire up agent pool if available (multi-agent mode)
	if components.AgentPool != nil && components.AgentPoolAdapter != nil {
		flowHandlerCfg.AgentPoolProxy = components.AgentPool
		flowHandlerCfg.AgentPoolAdapter = components.AgentPoolAdapter
		flowHandlerCfg.WorkManager = components.TaskManager
		flowHandlerCfg.SessionStorage = components.SessionStorage
		sessProcessor.SetAgentPoolRegistrar(components.AgentPool)
		// Re-wire AgentPool with AgentRegistry as FlowProvider (replaces legacy FlowManager)
		// so spawned agents can resolve flows from DB, not just YAML
		if agentRegistry != nil {
			components.AgentPool.SetEngine(
				components.Engine, agentRegistry,
				components.AgentToolResolver, components.ToolDepsProvider,
				components.ModelCache, agentRegistry,
			)
			components.AgentPool.SetModelResolver(agentRegistry, components.ModelCache)
		}
		loggerInstance.InfoContext(ctx, "Multi-agent mode enabled (Supervisor + Code Agents)")
	} else {
		loggerInstance.InfoContext(ctx, "Single-agent mode (no WorkStorage)")
	}

	flowHandler, err := grpc.NewFlowHandlerWithConfig(flowHandlerCfg)
	if err != nil {
		return fmt.Errorf("create flow handler: %w", err)
	}

	grpcServer.RegisterServices(flowHandler)

	// Wire chat endpoint and start HTTP server(s) now that SessionProcessor is ready.
	if httpServer != nil && agentRegistry != nil {
		chatService := &chatServiceHTTPAdapter{
			registry:    sessionRegistry,
			processor:   sessProcessor,
			agents:      agentRegistry,
			chatEnabled: components.AgentService != nil || components.ModelCache != nil,
		}
		var triggerChecker deliveryhttp.ChatTriggerChecker
		if pgDB != nil {
			chatTriggerRepo := configrepo.NewGORMTriggerRepository(pgDB)
			triggerChecker = &chatTriggerCheckerAdapter{repo: chatTriggerRepo}
			// §4.1: stamp last_fired_at on the first message of a chat session.
			chatService.triggers = chatTriggerRepo
		}
		chatHandler := deliveryhttp.NewChatHandler(chatService, triggerChecker, func() []string {
			return forwardHeadersStore.Load().([]string)
		})
		respondHandler := deliveryhttp.NewRespondHandler(sessionRegistry)

		// Register chat routes on external router (or single-port router).
		registerChatRoutes := func(router chi.Router) {
			router.Group(func(r chi.Router) {
				if httpAuthMW != nil {
					r.Use(httpAuthMW.Authenticate)
				}
				if configurableRL != nil {
					r.Use(configurableRL.Middleware)
				}
				r.Group(func(r chi.Router) {
					if httpAuthMW != nil {
						r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeChat))
					}
					r.Post("/api/v1/agents/{name}/chat", chatHandler.Chat)
					r.Post("/api/v1/sessions/{id}/respond", respondHandler.Respond)
				})
			})
		}

		// Chat API available on external port (or single-port)
		registerChatRoutes(httpServer.Router())
		// In two-port mode, also register on internal port (for /chat/ web client)
		if internalHTTPServer != nil {
			registerChatRoutes(internalHTTPServer.Router())
		}

		// Admin assistant — admin JWT required, no trigger gate.
		adminAssistantHandler := deliveryhttp.NewAdminAssistantHandler(chatService, func() []string {
			return forwardHeadersStore.Load().([]string)
		})
		registerAdminAssistantRoutes := func(router chi.Router) {
			router.Group(func(r chi.Router) {
				if httpAuthMW != nil {
					r.Use(httpAuthMW.Authenticate)
					r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeAgentsRead))
				}
				r.Post("/api/v1/admin/assistant/chat", adminAssistantHandler.Chat)
			})
		}
		registerAdminAssistantRoutes(httpServer.Router())
		if internalHTTPServer != nil {
			registerAdminAssistantRoutes(internalHTTPServer.Router())
		}

		// Agent list endpoint on external router (read-only, requires ScopeAgentsRead)
		if internalHTTPServer != nil {
			httpServer.Router().Group(func(r chi.Router) {
				if httpAuthMW != nil {
					r.Use(httpAuthMW.Authenticate)
					r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeAgentsRead))
				}
				r.Get("/api/v1/agents", deliveryhttp.NewAgentHandlerWithManager(
					&agentManagerHTTPAdapter{
						repo:       configrepo.NewGORMAgentRepository(pgDB),
						registry:   agentRegistry,
						db:         pgDB,
						schemaRepo: configrepo.NewGORMSchemaRepository(pgDB),
					}).List)
			})
		}

		// Start HTTP server(s)
		go func() {
			if err := httpServer.Start(); err != nil && err != http.ErrServerClosed {
				slog.Error("HTTP server error", "error", err)
			}
		}()
		if internalHTTPServer != nil {
			go func() {
				if err := internalHTTPServer.Start(); err != nil && err != http.ErrServerClosed {
					slog.Error("Internal HTTP server error", "error", err)
				}
			}()
			slog.InfoContext(ctx, "Two-port mode enabled",
				"external_port", httpPort, "internal_port", internalHTTPPort)
		} else {
			slog.InfoContext(ctx, "HTTP REST API server started", "port", httpPort)
		}
	}

	// Cron scheduler wiring lives in the platform-services block above
	// (taskrunner.StartCronScheduler). The legacy duplicate that used
	// cronTaskCreatorHTTPAdapter was removed — it created a second scheduler
	// that fired every trigger twice on boot.

	// Create WS connection handler for local CLI clients
	var agentCanceller ws.AgentCanceller
	if components.AgentPool != nil {
		agentCanceller = components.AgentPool
	}
	wsHandler := ws.NewConnectionHandler(sessionRegistry, sessProcessor, components.AgentService, agentCanceller, sc.LicenseInfo)

	// Create WS server (localhost only, random port)
	wsServer, err := ws.NewServer(wsHandler)
	if err != nil {
		return fmt.Errorf("create WS server: %w", err)
	}

	// Start WS server in goroutine
	go func() {
		if err := wsServer.Start(ctx); err != nil {
			slog.Error("WS server error", "error", err)
		}
	}()

	// Start gRPC server in goroutine
	serverErrChan := make(chan error, 1)
	go func() {
		if err := grpcServer.Start(ctx); err != nil {
			serverErrChan <- err
		}
	}()

	loggerInstance.InfoContext(ctx, "ByteBrew Server started successfully",
		"host", cfg.Server.Host,
		"grpc_port", grpcServer.ActualPort(),
		"ws_port", wsServer.Port(),
	)

	// Write port file for CLI discovery BEFORE emitting READY.
	portFileHost := cfg.Server.Host
	if portFileHost == "" || portFileHost == "0.0.0.0" {
		portFileHost = "127.0.0.1"
	}
	portWriter := portfile.NewWriter(dataDir)
	if err := portWriter.Write(portfile.PortInfo{
		PID:          os.Getpid(),
		Port:         grpcServer.ActualPort(),
		WsPort:       wsServer.Port(),
		HTTPPort:     httpPort,
		InternalPort: internalHTTPPort,
		Host:         portFileHost,
		StartedAt:    time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		slog.Warn("Failed to write port file", "error", err)
	} else {
		slog.Info("Port file written", "path", portWriter.Path())
	}

	// In managed mode, emit READY protocol AFTER port file is written.
	if sc.Managed {
		fmt.Printf("READY:%d\n", grpcServer.ActualPort())
		os.Stdout.Sync()
	}

	// Start memory retention cleanup goroutine (deletes expired entries based on per-agent config)
	if pgDB != nil {
		startMemoryRetentionCleanup(ctx, pgDB)
	}

	// Start bridge connectivity if enabled
	var bridgeCleanup func()
	if cfg.Bridge.Enabled && cfg.Bridge.URL != "" {
		cleanup, err := startBridge(ctx, cfg, dataDir, sessionRegistry, sessProcessor, wsHandler, loggerInstance, eventStore)
		if err != nil {
			slog.Error("Failed to start bridge connectivity", "error", err)
		} else {
			bridgeCleanup = cleanup
		}
	}

	// Wait for shutdown signal or server error
	select {
	case sig := <-sigChan:
		loggerInstance.InfoContext(ctx, "Received shutdown signal", "signal", sig)
		cancel()
	case err := <-serverErrChan:
		loggerInstance.ErrorContext(ctx, "Server error", "error", err)
		cancel()
	}

	loggerInstance.InfoContext(ctx, "Shutting down ByteBrew Server...")

	// Shutdown bridge first (stops accepting new messages)
	if bridgeCleanup != nil {
		bridgeCleanup()
	}

	// Cron scheduler is stopped via defer inside the platform-services block above.

	// Stop license watcher
	if sc.LicenseProvider != nil {
		sc.LicenseProvider.Stop()
		slog.Info("License watcher stopped")
	}

	// Close MCP client connections
	mcpRegistry.CloseAll()
	slog.Info("MCP clients closed")

	// Remove port file on shutdown
	if err := portWriter.Remove(); err != nil {
		slog.Warn("Failed to remove port file", "error", err)
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := wsServer.Shutdown(shutdownCtx); err != nil {
		slog.Warn("WS server shutdown error", "error", err)
	}

	if httpServer != nil {
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Warn("HTTP server shutdown error", "error", err)
		}
	}
	if internalHTTPServer != nil {
		if err := internalHTTPServer.Shutdown(shutdownCtx); err != nil {
			slog.Warn("Internal HTTP server shutdown error", "error", err)
		}
	}

	if err := grpcServer.Shutdown(shutdownCtx); err != nil {
		loggerInstance.ErrorContext(ctx, "Error during shutdown", "error", err)
	}

	loggerInstance.InfoContext(ctx, "ByteBrew Server stopped")
	return nil
}

// initializeGRPCServer creates the gRPC server, choosing between config-based
// listener and OS-assigned port based on managed mode.
func initializeGRPCServer(cfg *config.Config, log *logger.Logger, licenseInfo *domain.LicenseInfo, managed bool) (*grpc.Server, error) {
	if managed && cfg.Server.Port == 0 {
		listener, err := net.Listen("tcp4", "127.0.0.1:0")
		if err != nil {
			return nil, fmt.Errorf("listen on random port: %w", err)
		}
		return grpc.NewServerWithListener(listener, cfg.Server, log, licenseInfo), nil
	}

	server, err := grpc.NewServer(cfg.Server, log, licenseInfo)
	if err != nil {
		slog.Warn("Configured port busy, using random port",
			"port", cfg.Server.Port, "error", err)
		host := cfg.Server.Host
		if host == "" {
			host = "0.0.0.0"
		}
		listener, listenErr := net.Listen("tcp4", fmt.Sprintf("%s:0", host))
		if listenErr != nil {
			return nil, fmt.Errorf("listen on random port after fallback: %w", listenErr)
		}
		return grpc.NewServerWithListener(listener, cfg.Server, log, licenseInfo), nil
	}
	return server, nil
}

// UserDataDir returns the platform-specific user data directory for ByteBrew.
func UserDataDir() string {
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
		return filepath.Join(appData, "bytebrew")
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("failed to get user home directory: %v", err)
		}
		return filepath.Join(home, "Library", "Application Support", "bytebrew")
	default:
		xdgData := os.Getenv("XDG_DATA_HOME")
		if xdgData != "" {
			return filepath.Join(xdgData, "bytebrew")
		}
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("failed to get user home directory: %v", err)
		}
		return filepath.Join(home, ".local", "share", "bytebrew")
	}
}

// ensureManagedDirs creates the required subdirectories in the data directory.
func ensureManagedDirs(dataDir string) error {
	dirs := []string{
		filepath.Join(dataDir, "logs"),
		filepath.Join(dataDir, "data"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}
	return nil
}

// generateDefaultConfig writes a minimal config.yaml suitable for managed mode.
// If includeLicense is true, adds the default license public key section.
func generateDefaultConfig(path string, includeLicense bool) error {
	content := `# ByteBrew Server Config (auto-generated for managed mode)
server:
  host: "127.0.0.1"
  port: 0

database:
  host: localhost
  port: 5499
  user: postgres
  password: postgres
  database: bytebrew
  ssl_mode: disable

logging:
  level: "info"
  format: "text"
  output: "file"
  clear_on_startup: true

llm:
  default_provider: "ollama"
  ollama:
    model: "qwen2.5-coder:7b"
    base_url: "http://localhost:11434"
    timeout: 300s
`
	if includeLicense {
		content += `
license:
  public_key_hex: "5395bf9bb925ce56d86005104951984709670126f95a635e4e2ccf79ac58e395"
`
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// startBridge initializes and connects the Bridge relay stack for mobile device communication.
func startBridge(
	ctx context.Context,
	cfg *config.Config,
	dataDir string,
	sessionRegistry *flowregistry.SessionRegistry,
	processor *sessionprocessor.Processor,
	wsHandler *ws.ConnectionHandler,
	loggerInstance *logger.Logger,
	eventStore *eventstore.Store,
) (func(), error) {
	// Use shared PostgreSQL DB for bridge storage
	// Note: this function needs pgDB passed from caller
	// For now use inline GORM SQLite as fallback
	bridgeDBPath := filepath.Join(dataDir, "data", "bytebrew.db")
	bridgeDB, err := gorm.Open(sqlite.Open(bridgeDBPath), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open bridge db: %w", err)
	}

	identityStore := persistence.NewServerIdentityStore(bridgeDB)

	identity, err := identityStore.GetOrCreateIdentity()
	if err != nil {
		return nil, fmt.Errorf("get server identity: %w", err)
	}

	deviceStore := persistence.NewDeviceStore(bridgeDB)

	cryptoAdapter := bridge.NewDeviceCryptoAdapter()
	devices, err := deviceStore.List(ctx)
	if err != nil {
		slog.Warn("Failed to load existing devices for crypto", "error", err)
	} else {
		for _, d := range devices {
			if len(d.SharedSecret) > 0 {
				cryptoAdapter.AddDevice(d.ID, d.SharedSecret)
			}
		}
		if len(devices) > 0 {
			slog.Info("Loaded device crypto keys", "count", len(devices))
		}
	}

	hostName, _ := os.Hostname()
	if hostName == "" {
		hostName = "ByteBrew Server"
	}
	bridgeClient := bridge.NewBridgeClient(cfg.Bridge.URL, identity.ID, hostName, cfg.Bridge.AuthToken)

	messageRouter := bridge.NewMessageRouter(bridgeClient, cryptoAdapter)
	eventBroadcaster := bridge.NewEventBroadcaster(messageRouter, eventStore)
	sessionRegistry.SetEventHook(eventBroadcaster.BroadcastEvent)

	tokenStore := bridge.NewPairingTokenStore()
	pairingProvider := bridge.NewPairingProvider(tokenStore, identity, cfg.Bridge.URL)
	if wsHandler != nil {
		wsHandler.SetPairingProvider(pairingProvider)
	}

	deviceStoreAdapter := bridge.NewDeviceStoreAdapter(deviceStore)
	requestHandler := bridge.NewMobileRequestHandler(
		messageRouter,
		deviceStoreAdapter,
		tokenStore,
		cryptoAdapter,
		eventBroadcaster,
		sessionRegistry,
		processor,
		identity,
		hostName,
	)

	if err := bridgeClient.Connect(ctx); err != nil {
		// GORM handles connection pooling
		return nil, fmt.Errorf("connect to bridge: %w", err)
	}

	messageRouter.Start()
	requestHandler.Start()

	loggerInstance.InfoContext(ctx, "Bridge connectivity enabled",
		"url", cfg.Bridge.URL,
		"server_id", identity.ID,
	)

	cleanup := func() {
		slog.Info("Shutting down bridge connectivity")
		requestHandler.Stop()
		messageRouter.Stop()
		bridgeClient.Disconnect()
		// GORM handles connection pooling
		slog.Info("Bridge connectivity stopped")
	}

	return cleanup, nil
}

// connectMCPServers connects to MCP servers and registers them in the registry.
func connectMCPServers(ctx context.Context, mcpServers []models.MCPServerModel, registry *mcp.ClientRegistry) {
	for _, srv := range mcpServers {
		var forwardHeaders []string
		if srv.ForwardHeaders != "" {
			_ = json.Unmarshal([]byte(srv.ForwardHeaders), &forwardHeaders)
		}

		var transport mcp.Transport
		switch srv.Type {
		case "stdio":
			var args []string
			if srv.Args != "" {
				_ = json.Unmarshal([]byte(srv.Args), &args)
			}
			transport = mcp.NewStdioTransport(srv.Command, args, nil, forwardHeaders)
		case "http":
			transport = mcp.NewHTTPTransport(srv.URL, forwardHeaders)
		case "sse":
			transport = mcp.NewSSETransport(srv.URL, forwardHeaders)
		case "streamable-http":
			transport = mcp.NewStreamableHTTPTransport(srv.URL, forwardHeaders)
		default:
			slog.Warn("unknown MCP server type, skipping", "name", srv.Name, "type", srv.Type)
			continue
		}

		client := mcp.NewClient(srv.Name, transport)
		connectCtx, connectCancel := context.WithTimeout(ctx, 10*time.Second)
		if err := client.Connect(connectCtx); err != nil {
			slog.Warn("MCP server unavailable, skipping", "name", srv.Name, "error", err)
			connectCancel()
			continue
		}
		connectCancel()

		tools := client.ListTools()
		slog.Info("MCP server connected", "name", srv.Name, "tools", len(tools))
		registry.Register(srv.Name, client)
	}
}

// startMemoryRetentionCleanup launches a background goroutine that periodically
// deletes expired memory entries based on per-agent retention_days config.
func startMemoryRetentionCleanup(ctx context.Context, db *gorm.DB) {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		slog.InfoContext(ctx, "Memory retention cleanup goroutine started (every 1h)")
		runMemoryRetentionCleanup(ctx, db) // run once on startup
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				slog.Info("Memory retention cleanup goroutine stopped")
				return
			case <-ticker.C:
				runMemoryRetentionCleanup(ctx, db)
			}
		}
	}()
}

// runMemoryRetentionCleanup iterates all memory capabilities and deletes expired entries.
func runMemoryRetentionCleanup(ctx context.Context, db *gorm.DB) {
	var caps []models.CapabilityModel
	if err := db.WithContext(ctx).Where("type = ? AND enabled = ?", "memory", true).Find(&caps).Error; err != nil {
		slog.WarnContext(ctx, "memory retention cleanup: failed to list capabilities", "error", err)
		return
	}

	memStorage := persistence.NewMemoryStorage(db)
	totalDeleted := int64(0)

	for _, cap := range caps {
		if cap.Config == "" {
			continue
		}
		var config map[string]interface{}
		if err := json.Unmarshal([]byte(cap.Config), &config); err != nil {
			continue
		}

		unlimitedRetention, _ := config["unlimited_retention"].(bool)
		if unlimitedRetention {
			continue
		}

		retentionDays := 0
		if rd, ok := config["retention_days"].(float64); ok {
			retentionDays = int(rd)
		}
		if retentionDays <= 0 {
			continue
		}

		// V2: derive schema_ids for this agent via agent_relations
		// (docs/architecture/agent-first-runtime.md §2.1 — the
		// `schema_agents` join table no longer exists).
		var agentName string
		if err := db.WithContext(ctx).
			Raw("SELECT name FROM agents WHERE id = ?", cap.AgentID).
			Scan(&agentName).Error; err != nil || agentName == "" {
			slog.WarnContext(ctx, "memory retention cleanup: failed to resolve agent name",
				"agent_id", cap.AgentID, "error", err)
			continue
		}
		var schemaIDs []string
		if err := db.WithContext(ctx).
			Raw(`SELECT DISTINCT schema_id FROM agent_relations
				WHERE source_agent_name = ? OR target_agent_name = ?`, agentName, agentName).
			Scan(&schemaIDs).Error; err != nil {
			slog.WarnContext(ctx, "memory retention cleanup: failed to get schemas",
				"agent", agentName, "error", err)
			continue
		}

		for _, schemaID := range schemaIDs {
			deleted, err := memStorage.CleanupExpiredBySchema(ctx, schemaID, retentionDays)
			if err != nil {
				slog.WarnContext(ctx, "memory retention cleanup failed",
					"schema_id", schemaID, "retention_days", retentionDays, "error", err)
				continue
			}
			totalDeleted += deleted
		}
	}

	if totalDeleted > 0 {
		slog.InfoContext(ctx, "memory retention cleanup completed", "total_deleted", totalDeleted)
	}
}

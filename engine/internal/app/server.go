// Package app provides common server setup shared between CE and server (legacy) entry points.
package app

import (
	"context"
	"encoding/json"
	"fmt"
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
	googlegrpc "google.golang.org/grpc"

	"github.com/syntheticinc/bytebrew/engine/internal/delivery/grpc"
	deliveryhttp "github.com/syntheticinc/bytebrew/engine/internal/delivery/http"
	"github.com/syntheticinc/bytebrew/engine/internal/delivery/ws"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/embedded"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/agentregistry"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/agents/callbacks"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/taskrunner"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/turnexecutorfactory"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/versioncheck"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/audit"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/auth"
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

	"github.com/syntheticinc/bytebrew/engine/internal/service/resilience"
	"github.com/syntheticinc/bytebrew/engine/internal/service/sessionprocessor"
	"github.com/syntheticinc/bytebrew/engine/internal/service/turnexecutor"
	"github.com/syntheticinc/bytebrew/engine/pkg/config"
	"github.com/syntheticinc/bytebrew/engine/pkg/logger"
	pluginpkg "github.com/syntheticinc/bytebrew/engine/pkg/plugin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// ServerConfig holds parameters for Run.
type ServerConfig struct {
	// ConfigPath is the path to the config file (resolved by the caller).
	ConfigPath string

	// ConfigExplicit is true when --config was explicitly provided on the command line.
	ConfigExplicit bool

	// Port overrides the config port (0 = use config or random).
	Port int

	// Managed enables managed subprocess mode (random port, READY protocol).
	Managed bool

	// Plugin is the runtime extension point. nil defaults to pluginpkg.Noop{}
	// — a silent pass-through that adds nothing to the server.
	Plugin pluginpkg.Plugin

	// RequireTenant enforces presence of a non-empty tenant_id after auth.
	// CE defaults to false (single-tenant). Multi-tenant setups set it true.
	RequireTenant bool

	// Version, Commit, Date are build-time metadata.
	Version string
	Commit  string
	Date    string
}

// Run starts the ByteBrew server with the given configuration.
// This is the common entry point shared by CE and server (legacy) binaries.
func Run(sc ServerConfig) error {
	if sc.Plugin == nil {
		sc.Plugin = pluginpkg.Noop{}
	}

	// Wire the agent-step observer hook so plugins (EE metering, etc.) are
	// notified after every runtime step. The callbacks package uses a process-
	// global callback because the StepCounter lives deep in the agent
	// infrastructure; plumbing a dependency through four constructor layers
	// for a single observer hook would be disproportionate.
	plugin := sc.Plugin
	callbacks.SetStepCallback(func(ctx context.Context) error {
		return plugin.OnAgentStep(ctx, domain.TenantIDFromContext(ctx), pluginpkg.StepsLimitFromContext(ctx))
	})

	// Always resolve data dir (needed for port file discovery)
	dataDir, err := UserDataDir()
	if err != nil {
		return fmt.Errorf("resolve user data directory: %w", err)
	}
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
				if err := generateDefaultConfig(managedConfigPath); err != nil {
					return fmt.Errorf("generate default config: %w", err)
				}
				slog.InfoContext(context.Background(), "Generated default config", "path", managedConfigPath)
			}
			configPath = managedConfigPath
		}

		// Generate default prompts.yaml if missing (from embedded)
		managedPromptsPath := filepath.Join(dataDir, "prompts.yaml")
		if _, err := os.Stat(managedPromptsPath); os.IsNotExist(err) {
			if err := os.WriteFile(managedPromptsPath, embedded.DefaultPrompts, 0644); err != nil {
				return fmt.Errorf("write default prompts: %w", err)
			}
			slog.InfoContext(context.Background(), "Generated default prompts", "path", managedPromptsPath)
		}

		// Generate default flows.yaml if missing (from embedded)
		managedFlowsPath := filepath.Join(dataDir, "flows.yaml")
		if _, err := os.Stat(managedFlowsPath); os.IsNotExist(err) {
			if err := os.WriteFile(managedFlowsPath, embedded.DefaultFlows, 0644); err != nil {
				return fmt.Errorf("write default flows: %w", err)
			}
			slog.InfoContext(context.Background(), "Generated default flows", "path", managedFlowsPath)
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
		slog.InfoContext(context.Background(), "No config file found, using defaults (configure via environment variables or Admin Dashboard)", "path", configPath)
		cfg = config.DefaultConfig()
	} else {
		var loadErr error
		cfg, loadErr = config.Load(configPath)
		if loadErr != nil {
			return fmt.Errorf("load config: %w", loadErr)
		}
		slog.InfoContext(context.Background(), "Config loaded", "default_provider", cfg.LLM.DefaultProvider, "ollama_model", cfg.LLM.Ollama.Model)
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
			slog.WarnContext(context.Background(), "failed to remove stale port file", "error", err)
		} else {
			slog.InfoContext(context.Background(), "Removed stale port file", "pid", existingInfo.PID)
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
			slog.WarnContext(context.Background(), "failed to clear logs directory", "error", err)
		} else if removedCount > 0 {
			slog.InfoContext(context.Background(), "Cleared old log files", "count", removedCount, "dir", logsDir)
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
		slog.InfoContext(context.Background(), "pprof server started", "addr", pprofAddr)
		if err := http.ListenAndServe(pprofAddr, nil); err != nil {
			slog.ErrorContext(context.Background(), "pprof server failed", "error", err)
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
	var registryMgr *agentregistry.Manager
	var pgDB *gorm.DB
	var taskRepo *configrepo.GORMTaskRepository
	var apiTokenRepo *configrepo.GORMAPITokenRepository
	bootstrapCfg, bootstrapErr := config.LoadBootstrap(configPath)
	if bootstrapErr != nil {
		slog.InfoContext(context.Background(), "No bootstrap database config, running in legacy mode", "reason", bootstrapErr.Error())
	} else {
		var pgErr error
		pgDB, pgErr = gorm.Open(postgres.Open(bootstrapCfg.Database.URL), &gorm.Config{
			Logger: gormlogger.Default.LogMode(gormlogger.Silent),
		})
		if pgErr != nil {
			return fmt.Errorf("connect to PostgreSQL: %w", pgErr)
		}

		agentRepo := configrepo.NewGORMAgentRepository(pgDB)
		taskRepo = configrepo.NewGORMTaskRepository(pgDB)
		apiTokenRepo = configrepo.NewGORMAPITokenRepository(pgDB)
		capRepoForRegistry := configrepo.NewGORMCapabilityRepository(pgDB)
		registryMgr = agentregistry.NewManagerWithCapabilities(agentRepo, capRepoForRegistry, sc.RequireTenant)
		if loadErr := registryMgr.Init(ctx); loadErr != nil {
			return fmt.Errorf("load agents from database: %w", loadErr)
		}
		if !sc.RequireTenant {
			agentRegistry = registryMgr.Single()
			agentCount := agentRegistry.Count()
			if agentCount > 0 {
				slog.InfoContext(ctx, "Loaded agents from database", "count", agentCount, "agents", agentRegistry.List())
			} else {
				slog.InfoContext(ctx, "No agents configured in database")
			}
		} else {
			slog.InfoContext(ctx, "Multi-tenant mode: agent registries loaded per-tenant on first request")
		}

		// Wire the tenant seeder so plugins (EE Cloud provisioning) can populate
		// newly-created tenants with default data via engine repositories rather
		// than reimplementing schema/agent creation. CE's Noop plugin ignores
		// the seeder, so this is safe to wire unconditionally.
		sc.Plugin.SetTenantSeeder(&engineTenantSeeder{
			schemaRepo: configrepo.NewGORMSchemaRepository(pgDB),
		})

		// Wire the schema counter so EE quota middleware can enforce
		// SchemasLimit without making an internal HTTP sub-request (the old
		// sub-request design hard-coded the loopback port and silently
		// failed open whenever the engine bound a non-default port). CE's
		// Noop plugin ignores the counter — safe to wire unconditionally.
		schemaCounterRepo := configrepo.NewGORMSchemaRepository(pgDB)
		sc.Plugin.SetSchemaCounter(pluginpkg.SchemaCounterFunc(
			func(ctx context.Context, tenantID string) (int, error) {
				if tenantID == "" {
					return 0, nil
				}
				// Scope ctx to the plugin-supplied tenant so the repository
				// applies the same tenant filter it would for an authenticated
				// HTTP request.
				scoped := domain.WithTenantID(ctx, tenantID)
				recs, err := schemaCounterRepo.List(scoped)
				if err != nil {
					return 0, fmt.Errorf("count schemas: %w", err)
				}
				return len(recs), nil
			},
		))
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
		Config: *cfg,
		DB:     pgDB,
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
		// V2 Commit Group G (§5.8): per-end-user BYOK config seeds into
		// the `settings` table (jsonb) once on first boot. Admin UI edits
		// supersede this on subsequent boots.
		seedBYOKConfig(ctx, pgDB, cfg.BYOK)
	}

	if pgDB != nil {
		mcpServerRepo := configrepo.NewGORMMCPServerRepository(pgDB)
		mcpServers, mcpErr := mcpServerRepo.List(ctx)
		if mcpErr != nil {
			slog.WarnContext(context.Background(), "failed to load MCP servers from database", "error", mcpErr)
		} else {
			connectMCPServers(ctx, mcpServers, mcpRegistry, sc.Plugin.TransportPolicy())
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
				MCPServerRepo:  newAdminMCPServerRepoAdapter(configrepo.NewGORMMCPServerRepository(pgDB)),
				ModelRepo:      newAdminModelRepoAdapter(configrepo.NewGORMLLMProviderRepository(pgDB)),
				AgentRelationRepo: newAdminAgentRelationRepoAdapter(configrepo.NewGORMAgentRelationRepository(pgDB), configrepo.NewGORMAgentRepository(pgDB)),
				SessionRepo:    newAdminSessionRepoAdapter(configrepo.NewGORMSessionRepository(pgDB)),
				CapabilityRepo: newAdminCapabilityRepoAdapter(configrepo.NewGORMCapabilityRepository(pgDB)),
				Reloader: func() {
					if registryMgr != nil {
						registryMgr.InvalidateAll()
					}
				},
				TransportPolicy: sc.Plugin.TransportPolicy(),
			})
			slog.InfoContext(ctx, "admin tools registered into builtin store")
		}

		// Reload registry so the seeded builder-assistant is available at runtime.
		if registryMgr != nil {
			registryMgr.InvalidateAll()
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
	var poolRunner *poolBasedRunner
	if components.AgentPoolAdapter != nil && agentRegistry != nil {
		agentLifecycleReader = newAgentRegistryLifecycleAdapter(agentRegistry)
		poolRunner = &poolBasedRunner{pool: components.AgentPoolAdapter}
		lifecycleManager = lifecycle.NewManager(poolRunner)
		lifecycleManager.SetUUIDResolver(agentRegistry)
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
	heartbeatMonitor := resilience.NewHeartbeatMonitor(resilience.DefaultHeartbeatConfig(), heartbeatStuckCallback)
	heartbeatMonitor.Start(ctx)
	slog.InfoContext(ctx, "Heartbeat monitor started")

	// Resilience: DeadLetterQueue — tracks timed-out tasks (AC-RESIL-07/08)
	deadLetterQueue := resilience.NewDeadLetterQueue(resilience.DefaultDeadLetterConfig(), func(t resilience.TrackedTask, elapsed time.Duration) {
		slog.WarnContext(ctx, "task timed out, moved to dead letter",
			"task_id", t.TaskID, "agent_id", t.AgentID, "elapsed", elapsed)
	})

	// Wire per-agent capability config reader (memory max_entries, knowledge top_k)
	var capReader *capabilityConfigReader
	if pgDB != nil {
		capReader = &capabilityConfigReader{db: pgDB}
		if components.AgentToolResolver != nil {
			components.AgentToolResolver.SetCapabilityConfigReader(capReader)
			slog.InfoContext(ctx, "Capability config reader wired into AgentToolResolver")
		}
	}

	// HTTP REST API server — starts only when bootstrap config is available.
	// Supports two modes:
	//   Single-port (default): all routes on one port (backward compatible)
	//   Two-port: external (data plane) + internal (control plane)
	var httpServer *deliveryhttp.Server         // main server (single-port) or external (two-port)
	var internalHTTPServer *deliveryhttp.Server  // nil in single-port mode
	var httpPort int
	var internalHTTPPort int
	var httpAuthMW *deliveryhttp.AuthMiddleware
	var byokMW *deliveryhttp.BYOKMiddleware
	if bootstrapCfg != nil {
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

		// Security headers — applied globally before any route so every response
		// carries nosniff/frame-ancestors/CSP/referrer-policy. Widget routes
		// (which must be embeddable) install their own handler later with a
		// per-tenant frame-ancestors allowlist (key widget_embed_origins via settings table).
		r.Use(deliveryhttp.SecurityHeadersMiddleware)
		if internalHTTPServer != nil {
			internalRouter.Use(deliveryhttp.SecurityHeadersMiddleware)
		}

		// Extra HTTP middleware contributed by the plugin (e.g. EdDSA JWT verifier,
		// entitlements). Must be registered before any routes — chi panics otherwise.
		for _, mw := range sc.Plugin.HTTPMiddleware() {
			r.Use(mw)
			if internalHTTPServer != nil {
				internalRouter.Use(mw)
			}
		}

		// Auth — EdDSA verifier in all modes.
		//
		// Local mode: engine generates its own Ed25519 keypair on first boot,
		// signs short-lived admin sessions via POST /auth/local-session.
		// External mode (Cloud): engine loads the issuer's public key; token
		// issuance is owned by the landing service.
		// The plugin may override the default verifier entirely (EE) — the
		// middleware uses whatever it gets as long as the interface matches.
		var jwtVerifier pluginpkg.JWTVerifier
		var localSessionPrivKey []byte // non-nil in local mode, enables /auth/local-session route below
		if pluginVerifier := sc.Plugin.JWTVerifier(); pluginVerifier != nil {
			jwtVerifier = pluginVerifier
		} else {
			switch bootstrapCfg.Security.AuthMode {
			case config.AuthModeLocal:
				kp, err := auth.LoadOrGenerateKeypair(bootstrapCfg.Security.JWTKeysDir)
				if err != nil {
					return fmt.Errorf("load/generate local jwt keypair: %w", err)
				}
				verifier, err := auth.NewEdDSAVerifier(kp.Public)
				if err != nil {
					return fmt.Errorf("build local EdDSA verifier: %w", err)
				}
				jwtVerifier = verifier
				localSessionPrivKey = kp.Private
			case config.AuthModeExternal:
				pub, err := auth.LoadPublicKey(bootstrapCfg.Security.JWTPublicKeyPath)
				if err != nil {
					return fmt.Errorf("load external jwt public key: %w", err)
				}
				verifier, err := auth.NewEdDSAVerifier(pub)
				if err != nil {
					return fmt.Errorf("build external EdDSA verifier: %w", err)
				}
				jwtVerifier = verifier
			default:
				return fmt.Errorf("invalid auth_mode %q (expected %q or %q)",
					bootstrapCfg.Security.AuthMode, config.AuthModeLocal, config.AuthModeExternal)
			}
		}
		authMW := deliveryhttp.NewAuthMiddlewareWithVerifier(jwtVerifier, &tokenRepoHTTPAdapter{repo: apiTokenRepo})
		httpAuthMW = authMW

		// V2 §5.8: per-end-user BYOK middleware. Reads the live config from
		// `settings` (admin UI updates take effect on the next request via
		// SetConfig) and falls back to the YAML bootstrap when the table is
		// empty. Mounted after auth on chat / agent endpoints below.
		byokCfg := loadBYOKConfig(ctx, pgDB, cfg.BYOK)
		byokMW = deliveryhttp.NewBYOKMiddleware(deliveryhttp.BYOKConfig{
			Enabled:          byokCfg.Enabled,
			AllowedProviders: byokCfg.AllowedProviders,
		})

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

		// Local admin session issuer (public) — only wired in local auth mode.
		// Signs Ed25519 admin sessions with the local keypair generated at
		// boot. External/Cloud mode never exposes this route; token issuance
		// is owned by the landing service.
		var localSessionHandler *deliveryhttp.LocalSessionHandler
		if localSessionPrivKey != nil {
			localSessionHandler = deliveryhttp.NewLocalSessionHandler(localSessionPrivKey, time.Hour)
		}

		if internalHTTPServer != nil {
			// Two-port mode: register public routes on internal router too
			internalRouter.Get("/api/v1/health", healthHandler.ServeHTTP)
			internalRouter.Get("/api/v1/models/registry", registryHandler.List)
			internalRouter.Get("/api/v1/models/registry/providers", registryHandler.ListProviders)
			if localSessionHandler != nil {
				internalRouter.Post("/api/v1/auth/local-session", localSessionHandler.Issue)
			}
		}
		// Single-port or external: model registry + local session on main router
		r.Get("/api/v1/models/registry", registryHandler.List)
		r.Get("/api/v1/models/registry/providers", registryHandler.ListProviders)
		if localSessionHandler != nil {
			r.Post("/api/v1/auth/local-session", localSessionHandler.Issue)
		}

		// TenantMiddleware enforces presence of tenant_id after auth.
		// In CE mode RequireTenant is false, so requests without tenant_id
		// pass through; multi-tenant setups enable RequireTenant to reject
		// unscoped requests with 403.
		tenantExtractor := deliveryhttp.NewJWTTenantExtractor("tenant_id")
		tenantMW := deliveryhttp.NewTenantMiddleware(tenantExtractor, sc.RequireTenant)

		// Protected management routes — on internalRouter (= r in single-port mode)
		internalRouter.Group(func(r chi.Router) {
			r.Use(authMW.Authenticate)
			r.Use(tenantMW.Handler)
			r.Use(deliveryhttp.AuditMiddleware(&auditHTTPAdapter{logger: auditLogger}))

			// Schema repo (created early because agent manager needs it for used_in_schemas)
			schemaRepo := configrepo.NewGORMSchemaRepository(pgDB)

			// Agents
			agentRepo := configrepo.NewGORMAgentRepository(pgDB)
			// kbRepo is used by PatchAgent/CreateAgent to apply
			// knowledge_base_ids changes to the knowledge_base_agents M2M
			// table. Without this the request body field was silently
			// accepted and discarded (Bug 7).
			agentKBRepo := configrepo.NewGORMKnowledgeBaseRepository(pgDB)
			agentManager := &agentManagerHTTPAdapter{repo: agentRepo, registry: agentRegistry, registryMgr: registryMgr, db: pgDB, schemaRepo: schemaRepo, kbRepo: agentKBRepo}
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
				r.Patch("/api/v1/agents/{name}", agentHandler.Patch)
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
			capHandler := deliveryhttp.NewCapabilityHandler(&capabilityServiceHTTPAdapter{repo: capRepo, registryMgr: registryMgr})
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
				r.Patch("/api/v1/models/{name}", modelHandler.Patch)
				r.Delete("/api/v1/models/{name}", modelHandler.Delete)
				r.Post("/api/v1/models/{name}/verify", modelHandler.Verify)
			})

			// Tasks
			taskHandler := deliveryhttp.NewTaskHandler(&taskServiceHTTPAdapter{
				repo:          taskRepo,
				manager:       components.TaskManager,
				sessionReader: configrepo.NewGORMSessionRepository(pgDB),
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
				&configReloaderHTTPAdapter{registry: agentRegistry, mcpRegistry: mcpRegistry, db: pgDB, forwardHeadersStore: &forwardHeadersStore, transportPolicy: sc.Plugin.TransportPolicy()},
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
				kbRepo := configrepo.NewGORMKnowledgeBaseRepository(pgDB)
				knowledgeHandler := deliveryhttp.NewKnowledgeHandler(
					&knowledgeStatsHTTPAdapter{repo: knowledgeRepo, kbRepo: kbRepo},
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
				knowledgeHandler.SetFileLister(&knowledgeFileListerHTTPAdapter{svc: uploadSvc, kbRepo: kbRepo})

				// Knowledge Bases (many-to-many) handler
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
					r.Patch("/api/v1/knowledge-bases/{id}", kbHandler.PatchKB)
					r.Delete("/api/v1/knowledge-bases/{id}", kbHandler.Delete)
					r.Post("/api/v1/knowledge-bases/{id}/agents/{agent_name}", kbHandler.LinkAgent)
					r.Delete("/api/v1/knowledge-bases/{id}/agents/{agent_name}", kbHandler.UnlinkAgent)
					r.Post("/api/v1/knowledge-bases/{id}/files", kbHandler.UploadFile)
					r.Delete("/api/v1/knowledge-bases/{id}/files/{file_id}", kbHandler.DeleteFile)
					r.Post("/api/v1/knowledge-bases/{id}/files/{file_id}/reindex", kbHandler.ReindexFile)
				})
			}

			auditRepo := configrepo.NewGORMAuditRepository(pgDB)
			auditHandler := deliveryhttp.NewAuditHandler(&auditServiceHTTPAdapter{repo: auditRepo})
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireAdminSession)
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
			mcpHandler := deliveryhttp.NewMCPHandler(&mcpServiceHTTPAdapter{repo: mcpServerRepo}, sc.Plugin.TransportPolicy())
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeMCPRead))
				r.Get("/api/v1/mcp-servers", mcpHandler.List)
			})
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeMCPWrite))
				r.Post("/api/v1/mcp-servers", mcpHandler.Create)
				r.Put("/api/v1/mcp-servers/{name}", mcpHandler.Update)
				r.Patch("/api/v1/mcp-servers/{name}", mcpHandler.Patch)
				r.Delete("/api/v1/mcp-servers/{name}", mcpHandler.Delete)
			})

			// Schemas (with agent_relations). Chat access on a schema is
			// controlled by schemas.chat_enabled; edge graph lives in
			// agent_relations (source→target delegation).
			agentRelationRepo := configrepo.NewGORMAgentRelationRepository(pgDB)
			schemaHandler := deliveryhttp.NewSchemaHandler(
				&schemaServiceHTTPAdapter{repo: schemaRepo, db: pgDB},
				&agentRelationServiceHTTPAdapter{repo: agentRelationRepo, agentRepo: agentRepo, schemaRepo: schemaRepo, db: pgDB},
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
				r.Patch("/api/v1/schemas/{id}", schemaHandler.PatchSchema)
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
			settingHandler := deliveryhttp.NewSettingHandler(&settingServiceHTTPAdapter{
				repo:         settingRepo,
				byokMW:       byokMW,
				db:           pgDB,
				byokFallback: cfg.BYOK,
			})
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
			messageRepo := configrepo.NewGORMEventRepository(pgDB)
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
			usageHandler := deliveryhttp.NewUsageHandler(pgDB)
			r.Get("/api/v1/usage", usageHandler.GetUsage)

			// Tool Call Log — per-tool-call observability (OSS Phase 4).
			// OSS users rely on this to debug agent behavior: which tools were called,
			// with what args, how long they took, and whether they failed.
			toolCallRepoOSS := configrepo.NewToolCallEventRepository(pgDB)
			toolCallLogHandlerOSS := deliveryhttp.NewToolCallLogHandler(&toolCallLogHTTPAdapter{repo: toolCallRepoOSS})
			r.Get("/api/v1/audit/tool-calls", toolCallLogHandlerOSS.List)

			// Resilience admin endpoints (AC-RESIL-08: dead letters visible, circuit breaker management)
			resilienceHandler := deliveryhttp.NewResilienceHandler(
				&circuitBreakerQuerierHTTPAdapter{registry: cbRegistry},
				&deadLetterQuerierHTTPAdapter{queue: deadLetterQueue},
				&heartbeatQuerierHTTPAdapter{monitor: heartbeatMonitor},
			)
			r.Group(func(r chi.Router) {
				r.Use(deliveryhttp.RequireAdminSession)
				// Resilience Observability page (admin UI). Read-only.
				r.Get("/api/v1/resilience/circuit-breakers", resilienceHandler.ListCircuitBreakers)
				r.Post("/api/v1/resilience/circuit-breakers/{name}/reset", resilienceHandler.ResetCircuitBreaker)
				r.Get("/api/v1/resilience/dead-letter", resilienceHandler.ListDeadLetters)
				r.Get("/api/v1/resilience/stuck-agents", resilienceHandler.ListStuckAgents)
				r.Get("/api/v1/resilience/heartbeats", resilienceHandler.ListHeartbeats)
			})

			// Capability injector is wired into AgentToolResolver above (US-001).
			// capRepo is also used here for capability CRUD HTTP handlers.
		})

		// Extra HTTP routes contributed by the plugin (metrics, rate-limit
		// usage, etc.). Noop plugin registers nothing.
		sc.Plugin.RegisterHTTP(r, internalRouter)

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
			widgetAdapter := &widgetEmbedOriginsAdapter{repo: configrepo.NewGORMSettingRepository(pgDB)}
			r.Group(func(r chi.Router) {
				// Widget is publicly embeddable — optional auth so that if a
				// caller presents a valid JWT, tenant context is populated and
				// the widget CSP lookup returns the tenant's configured
				// widget_embed_origins. Anonymous callers get frame-ancestors
				// 'none' (safe default: blocks embedding until configured).
				if httpAuthMW != nil {
					r.Use(httpAuthMW.AuthenticateOptional)
				}
				r.Use(deliveryhttp.WidgetSecurityHeadersMiddleware(widgetAdapter))
				r.Get("/widget.js", widgetFileHandler)
			})
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
	// Tenant-aware gRPC interceptors: extract tenant_id from authorization
	// metadata and inject into context for downstream handlers.
	var jwtVerifierForGRPC pluginpkg.JWTVerifier
	if httpAuthMW != nil {
		jwtVerifierForGRPC = httpAuthMW.JWTVerifier()
	}
	extraGRPCOpts := []googlegrpc.ServerOption{
		googlegrpc.ChainUnaryInterceptor(grpc.TenantUnaryInterceptor(jwtVerifierForGRPC, sc.RequireTenant)),
		googlegrpc.ChainStreamInterceptor(grpc.TenantStreamInterceptor(jwtVerifierForGRPC, sc.RequireTenant)),
	}
	extraGRPCOpts = append(extraGRPCOpts, sc.Plugin.GRPCServerOptions()...)
	grpcServer, err := initializeGRPCServer(cfg, loggerInstance, extraGRPCOpts, sc.Managed || grpcUsesRandomPort)
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
	// Use AgentRegistry as FlowProvider (replaces legacy FlowManager for agent resolution).
	// In multi-tenant mode agentRegistry is nil and we must dispatch per-request via the
	// Manager, otherwise the static FlowManager has no agents (flows.yaml is empty).
	var flowProvider turnexecutor.FlowProvider = components.FlowManager
	var tenantAwareProvider *agentregistry.TenantAwareFlowProvider
	if agentRegistry != nil {
		flowProvider = agentRegistry
	} else if registryMgr != nil {
		tenantAwareProvider = agentregistry.NewTenantAwareFlowProvider(registryMgr)
		flowProvider = tenantAwareProvider
	}
	// Resolve AgentModelResolver (nil-safe: factory handles nil gracefully)
	var agentModelResolver turnexecutorfactory.AgentModelResolver
	if agentRegistry != nil {
		agentModelResolver = agentRegistry
	} else if tenantAwareProvider != nil {
		agentModelResolver = tenantAwareProvider
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

	// Wire agent UUID resolver so engine execution context uses uuid FK, not agent name.
	if agentRegistry != nil {
		factory.SetAgentUUIDResolver(agentRegistry)
		loggerInstance.InfoContext(ctx, "AgentUUIDResolver wired into TurnExecutorFactory")
	} else if tenantAwareProvider != nil {
		factory.SetAgentUUIDResolver(tenantAwareProvider)
		loggerInstance.InfoContext(ctx, "Tenant-aware AgentUUIDResolver wired into TurnExecutorFactory")
	}

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

		// V2: triggers-driven task fan-out (cron/webhook) removed. The background
		// task worker still runs so agents that spawn sub-tasks through the unified
		// task manager continue to be picked up; just no scheduler on top of it.

		// Wire per-agent capability config reader for memory max_entries
		if capReader != nil {
			factory.SetCapabilityConfigReader(capReader)
		}
	}

	// Create shared SessionProcessor
	sessProcessor := sessionprocessor.New(sessionRegistry, factory, eventStore)
	flowHandlerCfg.SessionProcessor = sessProcessor

	// Wire TurnExecutorFactory into poolBasedRunner so chat agents delegated via
	// lifecycle.Manager use the SSE path instead of the code-agent pool path.
	if poolRunner != nil {
		poolRunner.SetChatFactory(factory)
		slog.InfoContext(ctx, "TurnExecutorFactory wired into poolBasedRunner for chat agent delegation")
	}

	// Background task worker: picks up tasks created by agents (e.g. via
	// spawn_agent) and runs them in parallel through the session processor.
	// V2: cron/webhook trigger scheduler on top of this is deferred to V3.
	if taskRepo != nil {
		taskExecutor := taskrunner.NewTaskExecutor(
			components.TaskManager,
			sessionRegistry,
			sessProcessor,
			0, // 0 → DefaultTaskTimeout
		)
		if taskWorker := taskrunner.StartBackgroundWorker(taskExecutor, 4); taskWorker != nil {
			defer taskWorker.Stop()
		}
	}

	// Wire up agent pool if available (multi-agent mode)
	if components.AgentPool != nil && components.AgentPoolAdapter != nil {
		flowHandlerCfg.AgentPoolProxy = components.AgentPool
		flowHandlerCfg.AgentPoolAdapter = components.AgentPoolAdapter
		flowHandlerCfg.WorkManager = components.TaskManager
		// SessionStorage removed (V2 Group N: runtime_sessions table dropped).
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
	// In multi-tenant mode agentRegistry is nil (per-tenant registries loaded on demand),
	// but we still register the routes so auth middleware can reject unauthenticated requests.
	if httpServer != nil && (agentRegistry != nil || registryMgr != nil) {
		chatService := &chatServiceHTTPAdapter{
			registry:    sessionRegistry,
			processor:   sessProcessor,
			agents:      agentRegistry,
			registryMgr: registryMgr,
			chatEnabled: components.AgentService != nil || components.ModelCache != nil,
		}
		var schemaRepoForChat *configrepo.GORMSchemaRepository
		if pgDB != nil {
			schemaRepoForChat = configrepo.NewGORMSchemaRepository(pgDB)
			chatService.schemas = schemaRepoForChat
			chatService.sessions = configrepo.NewGORMSessionRepository(pgDB)
		}
		chatHandler := deliveryhttp.NewChatHandler(chatService, func() []string {
			return forwardHeadersStore.Load().([]string)
		})
		respondHandler := deliveryhttp.NewRespondHandler(sessionRegistry)

		// Register chat routes on external router (or single-port router).
		registerChatRoutes := func(router chi.Router) {
			router.Group(func(r chi.Router) {
				if httpAuthMW != nil {
					r.Use(httpAuthMW.Authenticate)
				}
				// V2 §5.8: BYOK runs AFTER auth so unauthenticated traffic
				// never reaches the header-parsing path; the LLM factory
				// reads ContextKeyBYOK* from the request context to decide
				// between tenant-configured and user-supplied credentials.
				if byokMW != nil {
					r.Use(byokMW.InjectBYOK)
				}
				r.Group(func(r chi.Router) {
					if httpAuthMW != nil {
						r.Use(deliveryhttp.RequireScope(deliveryhttp.ScopeChat))
					}
					r.Post("/api/v1/schemas/{id}/chat", chatHandler.Chat)
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

		// Admin assistant — admin JWT required, chats against the seeded
		// builder-schema; the schema resolver runs per-request so a late seed
		// is picked up without a restart.
		builderSchemaResolver := func(ctx context.Context) (string, error) {
			if pgDB == nil {
				return "", fmt.Errorf("no db")
			}
			var id string
			if err := pgDB.WithContext(ctx).Raw("SELECT id FROM schemas WHERE name = ? LIMIT 1", builderSchemaName).Scan(&id).Error; err != nil {
				return "", err
			}
			return id, nil
		}
		adminAssistantHandler := deliveryhttp.NewAdminAssistantHandler(chatService, builderSchemaResolver, func() []string {
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
						repo:        configrepo.NewGORMAgentRepository(pgDB),
						registry:    agentRegistry,
						registryMgr: registryMgr,
						db:          pgDB,
						schemaRepo:  configrepo.NewGORMSchemaRepository(pgDB),
						kbRepo:      configrepo.NewGORMKnowledgeBaseRepository(pgDB),
					}).List)
			})
		}

	}

	// Start HTTP server(s) — independent of agentRegistry (multi-tenant mode has no singleton).
	if httpServer != nil {
		go func() {
			if err := httpServer.Start(); err != nil && err != http.ErrServerClosed {
				slog.ErrorContext(context.Background(), "HTTP server error", "error", err)
			}
		}()
		if internalHTTPServer != nil {
			go func() {
				if err := internalHTTPServer.Start(); err != nil && err != http.ErrServerClosed {
					slog.ErrorContext(context.Background(), "Internal HTTP server error", "error", err)
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
	wsHandler := ws.NewConnectionHandler(sessionRegistry, sessProcessor, components.AgentService, agentCanceller, sc.Plugin)

	// Create WS server (localhost only, random port)
	wsServer, err := ws.NewServer(wsHandler)
	if err != nil {
		return fmt.Errorf("create WS server: %w", err)
	}

	// Start WS server in goroutine
	go func() {
		if err := wsServer.Start(ctx); err != nil {
			slog.ErrorContext(context.Background(), "WS server error", "error", err)
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
		slog.WarnContext(context.Background(), "Failed to write port file", "error", err)
	} else {
		slog.InfoContext(context.Background(), "Port file written", "path", portWriter.Path())
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

	// Cron scheduler is stopped via defer inside the platform-services block above.

	// Stop plugin resources (license watcher, etc.) — no-op in CE.
	sc.Plugin.Stop()
	slog.InfoContext(context.Background(), "plugin stopped")

	// Close MCP client connections
	mcpRegistry.CloseAll()
	slog.InfoContext(context.Background(), "MCP clients closed")

	// Remove port file on shutdown
	if err := portWriter.Remove(); err != nil {
		slog.WarnContext(context.Background(), "Failed to remove port file", "error", err)
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := wsServer.Shutdown(shutdownCtx); err != nil {
		slog.WarnContext(context.Background(), "WS server shutdown error", "error", err)
	}

	if httpServer != nil {
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.WarnContext(context.Background(), "HTTP server shutdown error", "error", err)
		}
	}
	if internalHTTPServer != nil {
		if err := internalHTTPServer.Shutdown(shutdownCtx); err != nil {
			slog.WarnContext(context.Background(), "Internal HTTP server shutdown error", "error", err)
		}
	}

	if err := grpcServer.Shutdown(shutdownCtx); err != nil {
		loggerInstance.ErrorContext(ctx, "Error during shutdown", "error", err)
	}

	loggerInstance.InfoContext(ctx, "ByteBrew Server stopped")
	return nil
}

// initializeGRPCServer creates the gRPC server, choosing between config-based
// listener and OS-assigned port based on managed mode. extraOpts are appended
// to the CE option chain (used by EE to inject license interceptors).
func initializeGRPCServer(cfg *config.Config, log *logger.Logger, extraOpts []googlegrpc.ServerOption, managed bool) (*grpc.Server, error) {
	if managed && cfg.Server.Port == 0 {
		listener, err := net.Listen("tcp4", "127.0.0.1:0")
		if err != nil {
			return nil, fmt.Errorf("listen on random port: %w", err)
		}
		return grpc.NewServerWithListener(listener, cfg.Server, log, extraOpts), nil
	}

	server, err := grpc.NewServer(cfg.Server, log, extraOpts)
	if err != nil {
		slog.WarnContext(context.Background(), "Configured port busy, using random port",
			"port", cfg.Server.Port, "error", err)
		host := cfg.Server.Host
		if host == "" {
			host = "0.0.0.0"
		}
		listener, listenErr := net.Listen("tcp4", fmt.Sprintf("%s:0", host))
		if listenErr != nil {
			return nil, fmt.Errorf("listen on random port after fallback: %w", listenErr)
		}
		return grpc.NewServerWithListener(listener, cfg.Server, log, extraOpts), nil
	}
	return server, nil
}

// UserDataDir returns the platform-specific user data directory for ByteBrew.
func UserDataDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
		return filepath.Join(appData, "bytebrew"), nil
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get user home directory: %w", err)
		}
		return filepath.Join(home, "Library", "Application Support", "bytebrew"), nil
	default:
		xdgData := os.Getenv("XDG_DATA_HOME")
		if xdgData != "" {
			return filepath.Join(xdgData, "bytebrew"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get user home directory: %w", err)
		}
		return filepath.Join(home, ".local", "share", "bytebrew"), nil
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
func generateDefaultConfig(path string) error {
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
	return os.WriteFile(path, []byte(content), 0644)
}

// connectMCPServers connects to MCP servers and registers them in the registry.
// policy is consulted before opening stdio transports: Cloud deployments block
// stdio to prevent host code execution; CE allows all transports.
func connectMCPServers(ctx context.Context, mcpServers []models.MCPServerModel, registry *mcp.ClientRegistry, policy mcpcatalog.TransportPolicy) {
	for _, srv := range mcpServers {
		var forwardHeaders []string
		if srv.ForwardHeaders != nil && *srv.ForwardHeaders != "" {
			_ = json.Unmarshal([]byte(*srv.ForwardHeaders), &forwardHeaders)
		}

		var transport mcp.Transport
		switch srv.Type {
		case "stdio":
			if err := policy.IsAllowed("stdio"); err != nil {
				slog.WarnContext(context.Background(), "MCP stdio transport blocked by policy", "name", srv.Name, "reason", err.Error())
				continue
			}
			var args []string
			if srv.Args != nil && *srv.Args != "" {
				_ = json.Unmarshal([]byte(*srv.Args), &args)
			}
			transport = mcp.NewStdioTransport(srv.Command, args, nil, forwardHeaders)
		case "http":
			transport = mcp.NewHTTPTransport(srv.URL, forwardHeaders)
		case "sse":
			transport = mcp.NewSSETransport(srv.URL, forwardHeaders)
		case "streamable-http":
			transport = mcp.NewStreamableHTTPTransport(srv.URL, forwardHeaders)
		default:
			slog.WarnContext(context.Background(), "unknown MCP server type, skipping", "name", srv.Name, "type", srv.Type)
			continue
		}

		client := mcp.NewClient(srv.Name, transport)
		connectCtx, connectCancel := context.WithTimeout(ctx, 10*time.Second)
		if err := client.Connect(connectCtx); err != nil {
			slog.WarnContext(context.Background(), "MCP server unavailable, skipping", "name", srv.Name, "error", err)
			connectCancel()
			continue
		}
		connectCancel()

		tools := client.ListTools()
		slog.InfoContext(context.Background(), "MCP server connected", "name", srv.Name, "tools", len(tools))
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
				slog.InfoContext(context.Background(), "Memory retention cleanup goroutine stopped")
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
		var agentID string
		if err := db.WithContext(ctx).
			Raw("SELECT id FROM agents WHERE name = ?", agentName).
			Scan(&agentID).Error; err != nil || agentID == "" {
			slog.WarnContext(ctx, "memory retention cleanup: failed to resolve agent id",
				"agent_name", agentName, "error", err)
			continue
		}
		var schemaIDs []string
		if err := db.WithContext(ctx).
			Raw(`SELECT DISTINCT schema_id FROM agent_relations
				WHERE source_agent_id = ? OR target_agent_id = ?`, agentID, agentID).
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

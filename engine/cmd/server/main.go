package main

import (
	"context"
	"flag"
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
	"syscall"
	"time"

	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/syntheticinc/bytebrew/engine/internal/delivery/grpc"
	deliveryhttp "github.com/syntheticinc/bytebrew/engine/internal/delivery/http"
	"github.com/syntheticinc/bytebrew/engine/internal/delivery/ws"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/audit"
	mcpinfra "github.com/syntheticinc/bytebrew/engine/internal/infrastructure/mcp"
	"github.com/syntheticinc/bytebrew/engine/internal/service/task"
	"github.com/syntheticinc/bytebrew/engine/internal/service/turn_executor"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/embedded"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/agent_registry"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/kit"
	"github.com/syntheticinc/bytebrew/engine/internal/kits/developer"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/bridge"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/flow_registry"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/config_repo"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/indexing"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/knowledge"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/portfile"
	"github.com/syntheticinc/bytebrew/engine/internal/service/eventstore"
	"github.com/syntheticinc/bytebrew/engine/internal/service/session_processor"
	"github.com/syntheticinc/bytebrew/engine/pkg/config"
	"github.com/syntheticinc/bytebrew/engine/pkg/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Build info is set via ldflags at build time.
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "Path to config file")
	showVersion := flag.Bool("version", false, "Print version and exit")
	managed := flag.Bool("managed", false, "Run as managed subprocess (random port, READY protocol)")
	portFlag := flag.Int("port", 0, "Override port (0 = random, only with --managed)")
	bridgeFlag := flag.String("bridge", "", "Bridge WebSocket URL (e.g., wss://bridge.bytebrew.ai)")
	flag.Parse()

	// --version: print and exit (no config needed)
	if *showVersion {
		fmt.Printf("bytebrew-srv %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	// Determine if --config was explicitly provided
	configExplicit := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "config" {
			configExplicit = true
		}
	})

	// Always resolve data dir (needed for port file discovery)
	dataDir := userDataDir()
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// In managed mode, create additional subdirs and override paths
	if *managed {
		if err := ensureManagedDirs(dataDir); err != nil {
			log.Fatalf("Failed to create managed directories: %v", err)
		}

		// If --config was not explicitly provided, use config from data dir
		if !configExplicit {
			managedConfigPath := filepath.Join(dataDir, "config.yaml")
			if _, err := os.Stat(managedConfigPath); os.IsNotExist(err) {
				if err := generateDefaultConfig(managedConfigPath); err != nil {
					log.Fatalf("Failed to generate default config: %v", err)
				}
				log.Printf("Generated default config at %s", managedConfigPath)
			}
			*configPath = managedConfigPath
		}

		// Generate default prompts.yaml if missing (from embedded)
		managedPromptsPath := filepath.Join(dataDir, "prompts.yaml")
		if _, err := os.Stat(managedPromptsPath); os.IsNotExist(err) {
			if err := os.WriteFile(managedPromptsPath, embedded.DefaultPrompts, 0644); err != nil {
				log.Fatalf("Failed to write default prompts: %v", err)
			}
			log.Printf("Generated default prompts at %s", managedPromptsPath)
		}

		// Generate default flows.yaml if missing (from embedded)
		managedFlowsPath := filepath.Join(dataDir, "flows.yaml")
		if _, err := os.Stat(managedFlowsPath); os.IsNotExist(err) {
			if err := os.WriteFile(managedFlowsPath, embedded.DefaultFlows, 0644); err != nil {
				log.Fatalf("Failed to write default flows: %v", err)
			}
			log.Printf("Generated default flows at %s", managedFlowsPath)
		}
	}

	// Get working directory for config path resolution
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	// Resolve config path relative to working directory
	if !filepath.IsAbs(*configPath) {
		*configPath = filepath.Join(wd, *configPath)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Config loaded: default_provider=%s, ollama_model=%s", cfg.LLM.DefaultProvider, cfg.LLM.Ollama.Model)

	// Override bridge config from --bridge flag
	if *bridgeFlag != "" {
		cfg.Bridge.URL = *bridgeFlag
		cfg.Bridge.Enabled = true
	}

	// Check for already running server BEFORE touching log files.
	// If log file is locked by the running server, logger.New will fail
	// with an unhelpful error. Give the user a clear message instead.
	portReader := portfile.NewReader(dataDir)
	existingInfo, _ := portReader.Read()
	if existingInfo != nil {
		if portfile.IsProcessAlive(existingInfo.PID) {
			log.Fatalf("Server already running (PID %d, port %d). Kill it first or use a different config.",
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
	if *managed {
		cfg.Logging.FilePath = filepath.Join(dataDir, "logs", "server.log")
		cfg.Server.Port = *portFlag
	}

	// Clear old logs if configured
	if cfg.Logging.ClearOnStartup {
		logsDir := filepath.Dir(cfg.Logging.FilePath)
		if logsDir == "" || logsDir == "." {
			logsDir = "logs" // default logs directory
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
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Set default slog logger to use our configured logger
	slog.SetDefault(loggerInstance.Logger)

	// Start pprof HTTP server for diagnostics (heap, goroutines, CPU profiling)
	go func() {
		pprofAddr := "localhost:6060"
		slog.Info("pprof server started", "addr", pprofAddr)
		if err := http.ListenAndServe(pprofAddr, nil); err != nil {
			slog.Error("pprof server failed", "error", err)
		}
	}()

	ctx := context.Background()
	loggerInstance.InfoContext(ctx, "Starting ByteBrew Server",
		"version", version,
		"commit", commit,
		"built", date,
		"config", *configPath,
	)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// NEW (Phase 1): Try loading bootstrap config for PostgreSQL database connection.
	// If database.url is present, connect to PostgreSQL, run migrations, and load AgentRegistry.
	// If not present (legacy config), skip — everything works as before.
	var agentRegistry *agent_registry.AgentRegistry
	var pgDB *gorm.DB
	var agentRepo *config_repo.GORMAgentRepository
	var taskRepo *config_repo.GORMTaskRepository
	var apiTokenRepo *config_repo.GORMAPITokenRepository
	var knowledgeRepo *config_repo.GORMKnowledgeRepository
	var knowledgeIndexer *knowledge.Indexer
	var bootstrapCfg *config.BootstrapConfig
	bootstrapCfg, bootstrapErr := config.LoadBootstrap(*configPath)
	if bootstrapErr != nil {
		slog.Info("No bootstrap database config, running in legacy mode", "reason", bootstrapErr.Error())
	} else {
		var pgErr error
		pgDB, pgErr = gorm.Open(postgres.Open(bootstrapCfg.Database.URL), &gorm.Config{
			Logger: gormlogger.Default.LogMode(gormlogger.Silent),
		})
		if pgErr != nil {
			slog.Error("Failed to connect to PostgreSQL", "error", pgErr)
			os.Exit(1)
		}

		if migrateErr := models.AutoMigrate(pgDB); migrateErr != nil {
			slog.Error("Failed to run database migrations", "error", migrateErr)
			os.Exit(1)
		}

		agentRepo = config_repo.NewGORMAgentRepository(pgDB)
		taskRepo = config_repo.NewGORMTaskRepository(pgDB)
		apiTokenRepo = config_repo.NewGORMAPITokenRepository(pgDB)
		knowledgeRepo = config_repo.NewGORMKnowledgeRepository(pgDB)

		// Knowledge indexer (requires Ollama embeddings)
		ollamaURL := "http://localhost:11434"
		if cfg != nil {
			if u := cfg.LLM.Ollama.BaseURL; u != "" {
				ollamaURL = u
			}
		}
		embClient := indexing.NewEmbeddingsClient(ollamaURL, "nomic-embed-text", 768)
		knowledgeIndexer = knowledge.NewIndexer(embClient, knowledgeRepo, slog.Default())

		// Wire knowledge deps into agent tool resolver (for auto-injection)
		// This will be set after engineComponents are created
		_ = embClient // used below when setting resolver knowledge
		agentRegistry = agent_registry.New(agentRepo)
		if loadErr := agentRegistry.Load(ctx); loadErr != nil {
			slog.Error("Failed to load agents from database", "error", loadErr)
			os.Exit(1)
		}

		agentCount := agentRegistry.Count()
		if agentCount > 0 {
			slog.InfoContext(ctx, "Loaded agents from database", "count", agentCount, "agents", agentRegistry.List())
		} else {
			slog.InfoContext(ctx, "No agents configured in database")
		}
	}

	// Validate license (server/legacy mode always validates)
	licenseInfo := infrastructure.ValidateLicense(cfg.License)
	slog.Info("license status",
		"tier", licenseInfo.Tier,
		"status", licenseInfo.Status,
		"expires", licenseInfo.ExpiresAt,
	)

	// Create infrastructure components (AgentService + WorkManager + AgentPool)
	components, err := infrastructure.NewInfraComponents(infrastructure.InfraComponentsConfig{
		Config:      *cfg,
		LicenseInfo: licenseInfo,
		DB:          pgDB,
	})
	if err != nil {
		log.Fatalf("Failed to create infrastructure components: %v", err)
	}

	// Wire knowledge search into agent tool resolver
	if knowledgeRepo != nil && components != nil && components.AgentToolResolver != nil {
		ollamaURL2 := "http://localhost:11434"
		if cfg != nil {
			if u := cfg.LLM.Ollama.BaseURL; u != "" {
				ollamaURL2 = u
			}
		}
		embClient2 := indexing.NewEmbeddingsClient(ollamaURL2, "nomic-embed-text", 768)
		components.AgentToolResolver.SetKnowledge(knowledgeRepo, embClient2)
		slog.InfoContext(ctx, "knowledge search wired into agent tool resolver")
	}

	// Create KitRegistry and register known kits.
	kitRegistry := kit.NewRegistry()
	kitRegistry.Register(developer.New())
	slog.InfoContext(ctx, "Kit registry initialized", "kits", kitRegistry.List())

	// HTTP REST API server — starts when bootstrap config is available (PostgreSQL connected).
	var chatAdapter *chatServiceAdapter
	if agentRegistry != nil && bootstrapCfg != nil {
		httpPort := bootstrapCfg.Engine.Port
		if httpPort == 0 {
			httpPort = 8443
		}
		httpSrv := deliveryhttp.NewServer(httpPort)
		r := httpSrv.Router()

		jwtSecret := bootstrapCfg.Security.AdminPassword
		authMW := deliveryhttp.NewAuthMiddleware(jwtSecret, &tokenRepoAdapter{repo: apiTokenRepo})

		auditLogger := audit.NewLogger(pgDB)

		// Public endpoints
		healthHandler := deliveryhttp.NewHealthHandler(version, &agentCounterAdapter{registry: agentRegistry})
		if pgDB != nil {
			sqlDB, _ := pgDB.DB()
			if sqlDB != nil {
				healthHandler.SetDBPinger(&dbPingerAdapter{db: sqlDB})
			}
		}
		r.Get("/api/v1/health", healthHandler.ServeHTTP)

		authHandler := deliveryhttp.NewAuthHandler(bootstrapCfg.Security.AdminUser, bootstrapCfg.Security.AdminPassword, jwtSecret)
		// Rate limit auth endpoints: 10 requests per minute per IP
		authRateLimiter := deliveryhttp.NewRateLimiter(10, time.Minute)
		r.With(authRateLimiter.Middleware).Post("/api/v1/auth/login", authHandler.Login)

		// Protected endpoints
		// Chat endpoint (SSE streaming — requires auth)
		// ChatService adapter gets sessionRegistry/sessProcessor wired later (after they're created)
		chatAdapter = &chatServiceAdapter{}
		chatHandler := deliveryhttp.NewChatHandler(chatAdapter)
		r.Group(func(r chi.Router) {
			r.Use(authMW.Authenticate)
			r.Post("/api/v1/agents/{name}/chat", chatHandler.Chat)
		})

		r.Group(func(r chi.Router) {
			r.Use(authMW.Authenticate)
			r.Use(deliveryhttp.AuditMiddleware(&auditLoggerAdapter{logger: auditLogger}))

			agentHandler := deliveryhttp.NewAgentHandlerWithManager(&agentManagerAdapter{
				agentListerAdapter: agentListerAdapter{registry: agentRegistry},
				repo:               agentRepo,
				registry:           agentRegistry,
			})
			r.Mount("/api/v1/agents", agentHandler.Routes())

			taskHandler := deliveryhttp.NewTaskHandler(&taskServiceAdapter{repo: taskRepo})
			r.Post("/api/v1/tasks", taskHandler.Create)
			r.Get("/api/v1/tasks", taskHandler.List)
			r.Get("/api/v1/tasks/{id}", taskHandler.Get)
			r.Delete("/api/v1/tasks/{id}", taskHandler.Cancel)
			r.Post("/api/v1/tasks/{id}/input", taskHandler.ProvideInput)

			configHandler := deliveryhttp.NewConfigHandler(&configReloaderAdapter{registry: agentRegistry}, &configImportExportAdapter{agentRepo: agentRepo, db: pgDB})
			r.Post("/api/v1/config/reload", configHandler.Reload)
			r.Post("/api/v1/config/import", configHandler.Import)
			r.Get("/api/v1/config/export", configHandler.Export)

			tokenHandler := deliveryhttp.NewTokenHandler(&tokenRepoAdapter{repo: apiTokenRepo})
			r.Post("/api/v1/auth/tokens", tokenHandler.CreateToken)
			r.Get("/api/v1/auth/tokens", tokenHandler.ListTokens)

			// Models (LLM Providers)
			llmProviderRepo := config_repo.NewGORMLLMProviderRepository(pgDB)
			modelHandler := deliveryhttp.NewModelHandler(&modelServiceAdapter{repo: llmProviderRepo})
			r.Mount("/api/v1/models", modelHandler.Routes())

			// MCP Servers
			mcpServerRepo := config_repo.NewGORMMCPServerRepository(pgDB)
			mcpHandler := deliveryhttp.NewMCPHandler(&mcpServiceAdapter{repo: mcpServerRepo})
			r.Mount("/api/v1/mcp-servers", mcpHandler.Routes())

			// Triggers
			triggerRepo := config_repo.NewGORMTriggerRepository(pgDB)
			triggerHandler := deliveryhttp.NewTriggerHandler(&triggerServiceAdapter{repo: triggerRepo})
			r.Mount("/api/v1/triggers", triggerHandler.Routes())

			// Settings
			settingRepo := config_repo.NewGORMSettingRepository(pgDB)
			settingHandler := deliveryhttp.NewSettingHandler(&settingServiceAdapter{repo: settingRepo})
			r.Mount("/api/v1/settings", settingHandler.Routes())

			// Sessions
			sessionRepo := config_repo.NewGORMSessionRepository(pgDB)
			messageRepo := config_repo.NewGORMMessageRepository(pgDB)
			sessionHandler := deliveryhttp.NewSessionHandler(&sessionServiceAdapter{repo: sessionRepo, messageRepo: messageRepo})
			sessionHandler.SetMessageService(&messageServiceAdapter{repo: messageRepo})
			r.Mount("/api/v1/sessions", sessionHandler.Routes())

			r.Delete("/api/v1/auth/tokens/{id}", tokenHandler.DeleteToken)

			// Tool metadata (security zones for admin UI)
			toolMetaHandler := deliveryhttp.NewToolMetadataHandler(&toolMetadataAdapter{})
			r.Get("/api/v1/tools/metadata", toolMetaHandler.List)

			// Knowledge endpoints
			if knowledgeRepo != nil {
				knowledgeStatsAdapter := &knowledgeStatsAdapter{repo: knowledgeRepo}
				var knowledgeReindexAdapter deliveryhttp.KnowledgeReindexer
				if knowledgeIndexer != nil && agentRegistry != nil {
					knowledgeReindexAdapter = &knowledgeReindexerAdapter{
						indexer:  knowledgeIndexer,
						registry: agentRegistry,
					}
				}
				knowledgeHandler := deliveryhttp.NewKnowledgeHandler(knowledgeStatsAdapter, knowledgeReindexAdapter)
				r.Get("/api/v1/agents/{name}/knowledge/status", knowledgeHandler.Status)
				r.Post("/api/v1/agents/{name}/knowledge/reindex", knowledgeHandler.Reindex)
			}

			// Audit log
			if pgDB != nil {
				auditRepo := config_repo.NewGORMAuditRepository(pgDB)
				auditServiceAdapt := &auditServiceHTTPAdapter{repo: auditRepo}
				auditHandler := deliveryhttp.NewAuditHandler(auditServiceAdapt)
				r.Get("/api/v1/audit", auditHandler.List)
			}
		})

		// Webhook route (public, no auth — triggered by external services)
		r.Post("/api/v1/webhooks/{path}", func(w http.ResponseWriter, req *http.Request) {
			webhookPath := chi.URLParam(req, "path")
			w.Header().Set("Content-Type", "application/json")

			var body struct {
				Title       string `json:"title"`
				Description string `json:"description"`
				Message     string `json:"message"`
			}
			// Best-effort parse; empty body is fine.
			_ = json.NewDecoder(req.Body).Decode(&body)

			t := &domain.EngineTask{
				Title:     "Webhook: " + webhookPath,
				AgentName: "supervisor",
				Source:    domain.TaskSourceWebhook,
				SourceID:  webhookPath,
				Status:    domain.EngineTaskStatusPending,
				Mode:      domain.TaskModeBackground,
			}
			if body.Title != "" {
				t.Title = body.Title
			}
			if body.Description != "" {
				t.Description = body.Description
			}
			if body.Message != "" && t.Description == "" {
				t.Description = body.Message
			}

			if err := taskRepo.Create(req.Context(), t); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"` + err.Error() + `"}`))
				return
			}
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(fmt.Sprintf(`{"task_id":%d}`, t.ID)))
		})

		go func() {
			if err := httpSrv.Start(); err != nil && err != http.ErrServerClosed {
				slog.Error("HTTP server error", "error", err)
			}
		}()
		slog.InfoContext(ctx, "HTTP REST API server started", "port", httpPort)
	}
	_ = kitRegistry // available for Kit resolution in AgentToolResolver

	// CronScheduler: load triggers from DB and start
	var cronScheduler *task.CronScheduler
	if taskRepo != nil {
		cronScheduler = task.NewCronScheduler(&cronTaskCreatorAdapter{repo: taskRepo})
		// Load triggers from DB
		triggers, trigErr := loadTriggersFromDB(pgDB)
		if trigErr == nil {
			for _, t := range triggers {
				if t.Type == "cron" && t.Schedule != "" {
					if err := cronScheduler.AddTrigger(t.Schedule, t.Title, t.Description, t.AgentName, fmt.Sprintf("trigger-%d", t.ID)); err != nil {
						slog.Warn("Failed to add cron trigger", "id", t.ID, "error", err)
					}
				}
			}
		}
		cronScheduler.Start()
		slog.InfoContext(ctx, "Cron scheduler started", "triggers", len(triggers))
	}

	// MCP Client: connect to configured MCP servers
	var mcpClients []*mcpinfra.Client
	if pgDB != nil {
		mcpServers, mcpErr := loadMCPServersFromDB(pgDB)
		if mcpErr != nil {
			slog.Warn("Failed to load MCP servers from DB", "error", mcpErr)
		} else {
			for _, srv := range mcpServers {
				var transport mcpinfra.Transport
				switch srv.Type {
				case "stdio":
					var args []string
					if srv.Args != "" {
						_ = json.Unmarshal([]byte(srv.Args), &args)
					}
					transport = mcpinfra.NewStdioTransport(srv.Command, args, nil)
				case "http":
					transport = mcpinfra.NewHTTPTransport(srv.URL)
				case "sse":
					transport = mcpinfra.NewSSETransport(srv.URL)
				default:
					slog.Warn("Unknown MCP server type", "name", srv.Name, "type", srv.Type)
					continue
				}
				client := mcpinfra.NewClient(srv.Name, transport)
				connectCtx, connectCancel := context.WithTimeout(ctx, 10*time.Second)
				if err := client.Connect(connectCtx); err != nil {
					slog.Warn("MCP server unavailable, skipping", "name", srv.Name, "error", err)
					connectCancel()
					continue
				}
				connectCancel()
				tools := client.ListTools()
				slog.InfoContext(ctx, "MCP server connected", "name", srv.Name, "tools", len(tools))
				mcpClients = append(mcpClients, client)
			}
		}
	}
	_ = mcpClients // available for tool resolution

	// Initialize gRPC server
	grpcServer, err := initializeGRPCServer(cfg, loggerInstance, components.LicenseInfo, *managed)
	if err != nil {
		log.Fatalf("Failed to initialize gRPC server: %v", err)
	}

	// Create flow registry for managing active flows
	flowRegistry := flow_registry.NewInMemoryRegistry()

	// Create event store (PostgreSQL) for reliable event replay on reconnect
	eventStore, err := eventstore.New(pgDB)
	if err != nil {
		log.Fatalf("Failed to create event store: %v", err)
	}

	// Create session registry for server-streaming API and bridge
	sessionRegistry := flow_registry.NewSessionRegistry(eventStore)

	// Create FlowHandler with multi-agent support
	pingInterval := 2 * time.Second
	flowHandlerCfg := grpc.FlowHandlerConfig{
		AgentService:           components.AgentService,
		ToolCallHistoryCleaner: components.AgentService.GetToolCallHistoryReminder(),
		PingInterval:           pingInterval,
		FlowRegistry:           flowRegistry,
		SessionRegistry:        sessionRegistry,
	}

	// Engine components are always available (server fails to start otherwise)
	// AgentRegistry is the FlowProvider (DB agents). PostgreSQL required.
	if agentRegistry == nil {
		log.Fatalf("AgentRegistry is nil — PostgreSQL database required. Set database.url in config.")
	}
	flowProvider := turn_executor.FlowProvider(agentRegistry)
	slog.InfoContext(ctx, "Using AgentRegistry as FlowProvider", "agents", agentRegistry.Count())

	factory := infrastructure.NewEngineTurnExecutorFactory(
		components.Engine,
		flowProvider,
		components.AgentToolResolver,
		components.ModelSelector,
		components.AgentConfig,
		components.WorkManager,     // taskManager (может быть nil)
		components.WorkManager,     // subtaskManager (может быть nil)
		components.AgentPoolAdapter, // agentPool (может быть nil)
		components.WebSearchTool,
		components.WebFetchTool,
		components.AgentService.GetContextReminders,
		components.ModelCache,
		agentRegistry,
	)
	flowHandlerCfg.TurnExecutorFactory = factory

	// Create shared SessionProcessor for server-streaming message processing.
	// Used by both gRPC SubscribeSession and bridge MobileRequestHandler.
	sessProcessor := session_processor.New(sessionRegistry, factory, eventStore)
	flowHandlerCfg.SessionProcessor = sessProcessor

	// Wire ChatService adapter to real session processor (for REST API SSE chat)
	if chatAdapter != nil {
		chatAdapter.sessionRegistry = sessionRegistry
		chatAdapter.sessProcessor = sessProcessor
		chatAdapter.sessionRepo = config_repo.NewGORMSessionRepository(pgDB)
		chatAdapter.messageRepo = config_repo.NewGORMMessageRepository(pgDB)
		slog.Info("ChatService REST wired to SessionProcessor")
	}

	// Wire up agent pool if available (multi-agent mode)
	if components.AgentPool != nil && components.AgentPoolAdapter != nil {
		flowHandlerCfg.AgentPoolProxy = components.AgentPool
		flowHandlerCfg.AgentPoolAdapter = components.AgentPoolAdapter
		flowHandlerCfg.WorkManager = components.WorkManager
		flowHandlerCfg.SessionStorage = components.SessionStorage
		// Register AgentPool as lifecycle event registrar for SessionProcessor.
		// This ensures agent_spawned/agent_completed events reach WS/mobile clients.
		sessProcessor.SetAgentPoolRegistrar(components.AgentPool)
		loggerInstance.InfoContext(ctx, "Multi-agent mode enabled (Supervisor + Code Agents)")
	} else {
		loggerInstance.InfoContext(ctx, "Single-agent mode (no WorkStorage)")
	}

	flowHandler, err := grpc.NewFlowHandlerWithConfig(flowHandlerCfg)
	if err != nil {
		log.Fatalf("Failed to create flow handler: %v", err)
	}

	grpcServer.RegisterServices(flowHandler)

	// Create WS connection handler for local CLI clients
	// AgentCanceller is nil-safe — handleCancelSession checks before calling.
	// Must cast explicitly to avoid non-nil interface wrapping a nil pointer.
	var agentCanceller ws.AgentCanceller
	if components.AgentPool != nil {
		agentCanceller = components.AgentPool
	}
	wsHandler := ws.NewConnectionHandler(sessionRegistry, sessProcessor, components.AgentService, agentCanceller, components.LicenseInfo)

	// Wire agent lister for WS list_agents command
	if agentRegistry != nil {
		wsHandler.SetAgentLister(&wsAgentListerAdapter{registry: agentRegistry})
	}

	// Create WS server (localhost only, random port)
	wsServer, err := ws.NewServer(wsHandler)
	if err != nil {
		log.Fatalf("Failed to create WS server: %v", err)
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
	// CLI reads READY and immediately tries to read the port file for ws_port.
	portFileHost := cfg.Server.Host
	if portFileHost == "" || portFileHost == "0.0.0.0" {
		portFileHost = "127.0.0.1"
	}
	portWriter := portfile.NewWriter(dataDir)
	if err := portWriter.Write(portfile.PortInfo{
		PID:       os.Getpid(),
		Port:      grpcServer.ActualPort(),
		WsPort:    wsServer.Port(),
		Host:      portFileHost,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		slog.Warn("Failed to write port file", "error", err)
	} else {
		slog.Info("Port file written", "path", portWriter.Path())
	}

	// In managed mode, emit READY protocol AFTER port file is written.
	if *managed {
		fmt.Printf("READY:%d\n", grpcServer.ActualPort())
		os.Stdout.Sync()
	}

	// Start bridge connectivity if enabled
	var bridgeCleanup func()
	if cfg.Bridge.Enabled && cfg.Bridge.URL != "" {
		cleanup, err := startBridge(ctx, cfg, dataDir, sessionRegistry, sessProcessor, wsHandler, loggerInstance, eventStore, pgDB)
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

	// Stop cron scheduler
	if cronScheduler != nil {
		cronScheduler.Stop()
		slog.Info("Cron scheduler stopped")
	}

	// Close MCP clients
	for _, client := range mcpClients {
		client.Close()
	}
	if len(mcpClients) > 0 {
		slog.Info("MCP clients closed", "count", len(mcpClients))
	}

	// Shutdown bridge first (stops accepting new messages)
	if bridgeCleanup != nil {
		bridgeCleanup()
	}

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

	if err := grpcServer.Shutdown(shutdownCtx); err != nil {
		loggerInstance.ErrorContext(ctx, "Error during shutdown", "error", err)
	}

	loggerInstance.InfoContext(ctx, "ByteBrew Server stopped")
}

// initializeGRPCServer creates the gRPC server, choosing between config-based
// listener and OS-assigned port based on managed mode.
// If the configured port is busy, falls back to a random OS-assigned port.
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
		// Port busy — fallback to random port (use tcp4 to avoid IPv6 issues with gRPC clients)
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

// userDataDir returns the platform-specific user data directory for ByteBrew.
func userDataDir() string {
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
	default: // linux and others
		// Respect XDG_DATA_HOME if set
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

// startBridge initializes and connects the Bridge relay stack for mobile device communication.
// Returns a cleanup function for graceful shutdown.
func startBridge(
	ctx context.Context,
	cfg *config.Config,
	dataDir string,
	sessionRegistry *flow_registry.SessionRegistry,
	processor *session_processor.Processor,
	wsHandler *ws.ConnectionHandler,
	loggerInstance *logger.Logger,
	eventStore *eventstore.Store,
	bridgeDB *gorm.DB,
) (func(), error) {
	db := bridgeDB
	if db == nil {
		return nil, fmt.Errorf("database required for bridge")
	}

	// Get or create persistent server identity (server_id + X25519 keypair)
	identityStore := persistence.NewServerIdentityStore(bridgeDB)

	identity, err := identityStore.GetOrCreateIdentity()
	if err != nil {
		return nil, fmt.Errorf("get server identity: %w", err)
	}

	// Create device store for paired mobile devices
	deviceStore := persistence.NewDeviceStore(db)

	// Create crypto adapter and load existing device secrets
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

	// Create bridge client — use machine hostname as server name so mobile
	// users can distinguish between multiple connected machines.
	hostName, _ := os.Hostname()
	if hostName == "" {
		hostName = "ByteBrew Server"
	}
	bridgeClient := bridge.NewBridgeClient(cfg.Bridge.URL, identity.ID, hostName, cfg.Bridge.AuthToken)

	// Create message router (handles E2E encryption transparently)
	messageRouter := bridge.NewMessageRouter(bridgeClient, cryptoAdapter)

	// Create event broadcaster (serializes session events for mobile clients)
	eventBroadcaster := bridge.NewEventBroadcaster(messageRouter, eventStore)

	// Wire event hook so SessionRegistry broadcasts events to mobile clients
	sessionRegistry.SetEventHook(eventBroadcaster.BroadcastEvent)

	// Create pairing token store (in-memory, ephemeral)
	tokenStore := bridge.NewPairingTokenStore()

	// Create pairing data provider and wire it to WS handler for CLI access
	pairingProvider := bridge.NewPairingProvider(tokenStore, identity, cfg.Bridge.URL)
	if wsHandler != nil {
		wsHandler.SetPairingProvider(pairingProvider)
	}

	// Create device store adapter (bridges context-less interface with context-aware store)
	deviceStoreAdapter := bridge.NewDeviceStoreAdapter(deviceStore)

	// Create request handler (routes incoming mobile messages)
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

	// Connect to bridge relay
	if err := bridgeClient.Connect(ctx); err != nil {
		// GORM handles connection pooling
		return nil, fmt.Errorf("connect to bridge: %w", err)
	}

	// Start message routing and request handling
	messageRouter.Start()
	requestHandler.Start()

	loggerInstance.InfoContext(ctx, "Bridge connectivity enabled",
		"url", cfg.Bridge.URL,
		"server_id", identity.ID,
	)

	// Return cleanup function for graceful shutdown
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

license:
  public_key_hex: "5395bf9bb925ce56d86005104951984709670126f95a635e4e2ccf79ac58e395"

llm:
  default_provider: "ollama"
  ollama:
    model: "qwen2.5-coder:7b"
    base_url: "http://localhost:11434"
    timeout: 300s
`
	return os.WriteFile(path, []byte(content), 0644)
}

// generateDefaultPrompts writes a minimal prompts.yaml suitable for managed mode.
func generateDefaultPrompts(path string) error {
	content := `# ByteBrew Prompts (auto-generated for managed mode)
prompts:
  system_prompt: |
    You are a coding assistant that helps users understand and work with code.

    **Language:** Match the user's language. Russian question = Russian answer.

    **BE EFFICIENT — minimize tool calls.**
    Every tool call takes time. Do NOT read the same file twice. Fewer calls = faster response.

    **ACTION over description.**
    When the user asks to modify/refactor/fix code — DO IT immediately.
    Read the file, make the changes, done.

    **Approach:**
    - Be concise. Reference files and line numbers when relevant.
    - Answer questions directly — don't just describe what you found.
    - Use grep_search and glob to explore unfamiliar code.

    **How you work:**
    - You are an AI coding assistant with access to file system tools.
    - You can read, write, and search code files.
    - You can execute shell commands.
`
	return os.WriteFile(path, []byte(content), 0644)
}


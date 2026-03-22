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
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/syntheticinc/bytebrew/engine/internal/delivery/grpc"
	deliveryhttp "github.com/syntheticinc/bytebrew/engine/internal/delivery/http"
	"github.com/syntheticinc/bytebrew/engine/internal/delivery/ws"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/embedded"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/agent_registry"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/audit"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/bridge"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/flow_registry"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/indexing"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/knowledge"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/kit"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/config_repo"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/portfile"
	"github.com/syntheticinc/bytebrew/engine/internal/kits/developer"
	"github.com/syntheticinc/bytebrew/engine/internal/service/eventstore"
	"github.com/syntheticinc/bytebrew/engine/internal/service/session_processor"
	"github.com/syntheticinc/bytebrew/engine/internal/service/task"
	"github.com/syntheticinc/bytebrew/engine/internal/service/turn_executor"
	"github.com/syntheticinc/bytebrew/engine/pkg/config"
	"github.com/syntheticinc/bytebrew/engine/pkg/logger"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

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
	var agentRegistry *agent_registry.AgentRegistry
	var pgDB *gorm.DB
	var taskRepo *config_repo.GORMTaskRepository
	var apiTokenRepo *config_repo.GORMAPITokenRepository
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

		agentRepo := config_repo.NewGORMAgentRepository(pgDB)
		taskRepo = config_repo.NewGORMTaskRepository(pgDB)
		apiTokenRepo = config_repo.NewGORMAPITokenRepository(pgDB)
		agentRegistry = agent_registry.New(agentRepo)
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
	components, err := infrastructure.NewInfraComponents(infrastructure.InfraComponentsConfig{
		Config:      *cfg,
		LicenseInfo: sc.LicenseInfo,
		DB:          pgDB,
	})
	if err != nil {
		return fmt.Errorf("create infrastructure components: %w", err)
	}

	// Create KitRegistry and register known kits.
	kitRegistry := kit.NewRegistry()
	kitRegistry.Register(developer.New())
	slog.InfoContext(ctx, "Kit registry initialized", "kits", kitRegistry.List())

	// Knowledge indexing infrastructure (created before HTTP so endpoints can use it)
	var knowledgeRepo *config_repo.GORMKnowledgeRepository
	var knowledgeIndexer *knowledge.Indexer
	var embeddingsClient *indexing.EmbeddingsClient
	if pgDB != nil {
		knowledgeRepo = config_repo.NewGORMKnowledgeRepository(pgDB)
		embeddingsClient = indexing.NewEmbeddingsClient(
			indexing.DefaultOllamaURL,
			indexing.DefaultEmbedModel,
			indexing.DefaultDimension,
		)
		knowledgeIndexer = knowledge.NewIndexer(embeddingsClient, knowledgeRepo, slog.Default())

		// Background indexing for agents with KnowledgePath on startup
		if agentRegistry != nil {
			for _, name := range agentRegistry.List() {
				agent, err := agentRegistry.Get(name)
				if err != nil || agent.Record.KnowledgePath == "" {
					continue
				}
				agentName := name
				folderPath := agent.Record.KnowledgePath
				go func() {
					bgCtx := context.Background()
					slog.InfoContext(bgCtx, "starting background knowledge indexing",
						"agent", agentName, "path", folderPath)
					if err := knowledgeIndexer.IndexFolder(bgCtx, agentName, folderPath); err != nil {
						slog.ErrorContext(bgCtx, "background knowledge indexing failed",
							"agent", agentName, "error", err)
					}
				}()
			}
		}
	}

	// HTTP REST API server (Phase 5) — starts only when bootstrap config is available.
	var httpServer *deliveryhttp.Server
	var httpPort int
	var httpAuthMW *deliveryhttp.AuthMiddleware
	if agentRegistry != nil && bootstrapCfg != nil {
		httpPort = bootstrapCfg.Engine.Port
		if httpPort == 0 {
			httpPort = 8443
		}
		httpServer = deliveryhttp.NewServer(httpPort)
		r := httpServer.Router()

		// Auth
		jwtSecret := bootstrapCfg.Security.AdminPassword
		authMW := deliveryhttp.NewAuthMiddleware(jwtSecret, &tokenRepoHTTPAdapter{repo: apiTokenRepo})
		httpAuthMW = authMW

		// Audit logger
		auditLogger := audit.NewLogger(pgDB)

		// Health (public)
		healthHandler := deliveryhttp.NewHealthHandler(sc.Version, &agentCounterHTTPAdapter{registry: agentRegistry})
		r.Get("/api/v1/health", healthHandler.ServeHTTP)

		// Auth login (public)
		authHandler := deliveryhttp.NewAuthHandler(
			bootstrapCfg.Security.AdminUser,
			bootstrapCfg.Security.AdminPassword,
			jwtSecret,
		)
		r.Post("/api/v1/auth/login", authHandler.Login)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(authMW.Authenticate)
			r.Use(deliveryhttp.AuditMiddleware(&auditHTTPAdapter{logger: auditLogger}))

			// Agents (full CRUD)
			agentRepo := config_repo.NewGORMAgentRepository(pgDB)
			agentManager := &agentManagerHTTPAdapter{repo: agentRepo, registry: agentRegistry, db: pgDB}
			agentHandler := deliveryhttp.NewAgentHandlerWithManager(agentManager)
			r.Get("/api/v1/agents", agentHandler.List)
			r.Get("/api/v1/agents/{name}", agentHandler.Get)
			r.Post("/api/v1/agents", agentHandler.Create)
			r.Put("/api/v1/agents/{name}", agentHandler.Update)
			r.Delete("/api/v1/agents/{name}", agentHandler.Delete)

			// Models (full CRUD)
			llmProviderRepo := config_repo.NewGORMLLMProviderRepository(pgDB)
			modelService := &modelServiceHTTPAdapter{repo: llmProviderRepo, modelCache: components.ModelCache}
			modelHandler := deliveryhttp.NewModelHandler(modelService)
			r.Mount("/api/v1/models", modelHandler.Routes())

			// Tasks
			taskHandler := deliveryhttp.NewTaskHandler(&taskServiceHTTPAdapter{repo: taskRepo})
			r.Post("/api/v1/tasks", taskHandler.Create)
			r.Get("/api/v1/tasks", taskHandler.List)
			r.Get("/api/v1/tasks/{id}", taskHandler.Get)
			r.Delete("/api/v1/tasks/{id}", taskHandler.Cancel)
			r.Post("/api/v1/tasks/{id}/input", taskHandler.ProvideInput)

			// Config
			configHandler := deliveryhttp.NewConfigHandler(
				&configReloaderHTTPAdapter{registry: agentRegistry},
				&configImportExportHTTPAdapter{db: pgDB},
			)
			r.Post("/api/v1/config/reload", configHandler.Reload)
			r.Post("/api/v1/config/import", configHandler.Import)
			r.Get("/api/v1/config/export", configHandler.Export)

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
				r.Get("/api/v1/agents/{name}/knowledge/status", knowledgeHandler.Status)
				r.Post("/api/v1/agents/{name}/knowledge/reindex", knowledgeHandler.Reindex)
			}

			// Audit logs
			auditRepo := config_repo.NewGORMAuditRepository(pgDB)
			auditHandler := deliveryhttp.NewAuditHandler(&auditServiceHTTPAdapter{repo: auditRepo})
			r.Get("/api/v1/audit", auditHandler.List)

			// API Tokens
			tokenHandler := deliveryhttp.NewTokenHandler(&tokenRepoHTTPAdapter{repo: apiTokenRepo})
			r.Post("/api/v1/auth/tokens", tokenHandler.CreateToken)
			r.Get("/api/v1/auth/tokens", tokenHandler.ListTokens)
			r.Delete("/api/v1/auth/tokens/{id}", tokenHandler.DeleteToken)

			// MCP Servers
			mcpServerRepo := config_repo.NewGORMMCPServerRepository(pgDB)
			mcpHandler := deliveryhttp.NewMCPHandler(&mcpServiceHTTPAdapter{repo: mcpServerRepo})
			r.Mount("/api/v1/mcp-servers", mcpHandler.Routes())

			// Triggers
			triggerRepo := config_repo.NewGORMTriggerRepository(pgDB)
			triggerHandler := deliveryhttp.NewTriggerHandler(&triggerServiceHTTPAdapter{repo: triggerRepo})
			r.Mount("/api/v1/triggers", triggerHandler.Routes())

			// Settings
			settingRepo := config_repo.NewGORMSettingRepository(pgDB)
			settingHandler := deliveryhttp.NewSettingHandler(&settingServiceHTTPAdapter{repo: settingRepo})
			r.Mount("/api/v1/settings", settingHandler.Routes())

			// Sessions
			sessionRepo := config_repo.NewGORMSessionRepository(pgDB)
			messageRepo := config_repo.NewGORMMessageRepository(pgDB)
			sessionHandler := deliveryhttp.NewSessionHandler(&sessionServiceHTTPAdapter{repo: sessionRepo, messageRepo: messageRepo})
			sessionHandler.SetMessageService(&messageServiceHTTPAdapter{repo: messageRepo})
			r.Mount("/api/v1/sessions", sessionHandler.Routes())

			// Tool metadata (security zones for admin UI)
			toolMetaHandler := deliveryhttp.NewToolMetadataHandler(&toolMetadataHTTPAdapter{})
			r.Get("/api/v1/tools/metadata", toolMetaHandler.List)
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

		// Serve Admin Dashboard SPA (static files)
		adminDir := "/usr/share/bytebrew/admin"
		if _, statErr := os.Stat(adminDir); statErr == nil {
			spaFS := http.Dir(adminDir)
			r.Get("/admin/*", func(w http.ResponseWriter, req *http.Request) {
				// Strip /admin prefix for file lookup
				filePath := strings.TrimPrefix(req.URL.Path, "/admin")
				if filePath == "" || filePath == "/" {
					filePath = "/index.html"
				}
				// Try serving the file; if not found, serve index.html (SPA routing)
				if _, err := os.Stat(filepath.Join(adminDir, filePath)); os.IsNotExist(err) {
					http.ServeFile(w, req, filepath.Join(adminDir, "index.html"))
					return
				}
				http.StripPrefix("/admin", http.FileServer(spaFS)).ServeHTTP(w, req)
			})
			r.Get("/admin", func(w http.ResponseWriter, req *http.Request) {
				http.Redirect(w, req, "/admin/", http.StatusMovedPermanently)
			})
			slog.InfoContext(ctx, "Admin Dashboard served", "path", adminDir)
		} else {
			slog.InfoContext(ctx, "Admin Dashboard not found (optional)", "path", adminDir)
		}

		// NOTE: HTTP server start is deferred until after SessionProcessor is created,
		// so the chat endpoint can be wired with all required dependencies.
	}
	_ = kitRegistry // available for Kit resolution in AgentToolResolver

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
	flowRegistry := flow_registry.NewInMemoryRegistry()

	// Create event store (PostgreSQL) for reliable event replay on reconnect
	eventStore, err := eventstore.New(pgDB)
	if err != nil {
		return fmt.Errorf("create event store: %w", err)
	}

	// Create session registry for server-streaming API and bridge
	sessionRegistry := flow_registry.NewSessionRegistry(eventStore)

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
	var flowProvider turn_executor.FlowProvider = components.FlowManager
	if agentRegistry != nil {
		flowProvider = agentRegistry
	}
	// Resolve AgentModelResolver (nil-safe: factory handles nil gracefully)
	var agentModelResolver infrastructure.AgentModelResolver
	if agentRegistry != nil {
		agentModelResolver = agentRegistry
	}

	factory := infrastructure.NewEngineTurnExecutorFactory(
		components.Engine,
		flowProvider,
		components.AgentToolResolver,
		components.ModelSelector,
		components.AgentConfig,
		components.WorkManager,
		components.WorkManager,
		components.AgentPoolAdapter,
		components.WebSearchTool,
		components.WebFetchTool,
		func() []turn_executor.ContextReminderProvider {
			if components.AgentService != nil {
				return components.AgentService.GetContextReminders()
			}
			return nil
		},
		components.ModelCache,
		agentModelResolver,
	)
	flowHandlerCfg.TurnExecutorFactory = factory

	// Create shared SessionProcessor
	sessProcessor := session_processor.New(sessionRegistry, factory, eventStore)
	flowHandlerCfg.SessionProcessor = sessProcessor

	// Wire up agent pool if available (multi-agent mode)
	if components.AgentPool != nil && components.AgentPoolAdapter != nil {
		flowHandlerCfg.AgentPoolProxy = components.AgentPool
		flowHandlerCfg.AgentPoolAdapter = components.AgentPoolAdapter
		flowHandlerCfg.WorkManager = components.WorkManager
		flowHandlerCfg.SessionStorage = components.SessionStorage
		sessProcessor.SetAgentPoolRegistrar(components.AgentPool)
		loggerInstance.InfoContext(ctx, "Multi-agent mode enabled (Supervisor + Code Agents)")
	} else {
		loggerInstance.InfoContext(ctx, "Single-agent mode (no WorkStorage)")
	}

	flowHandler, err := grpc.NewFlowHandlerWithConfig(flowHandlerCfg)
	if err != nil {
		return fmt.Errorf("create flow handler: %w", err)
	}

	grpcServer.RegisterServices(flowHandler)

	// Wire chat endpoint and start HTTP server now that SessionProcessor is ready.
	if httpServer != nil && agentRegistry != nil {
		chatService := &chatServiceHTTPAdapter{
			registry:    sessionRegistry,
			processor:   sessProcessor,
			agents:      agentRegistry,
			chatEnabled: components.AgentService != nil || components.ModelCache != nil,
		}
		chatHandler := deliveryhttp.NewChatHandler(chatService)
		respondHandler := deliveryhttp.NewRespondHandler(sessionRegistry)

		httpRouter := httpServer.Router()
		httpRouter.Group(func(r chi.Router) {
			if httpAuthMW != nil {
				r.Use(httpAuthMW.Authenticate)
			}
			r.Post("/api/v1/agents/{name}/chat", chatHandler.Chat)
			r.Post("/api/v1/sessions/{id}/respond", respondHandler.Respond)
		})

		go func() {
			if err := httpServer.Start(); err != nil && err != http.ErrServerClosed {
				slog.Error("HTTP server error", "error", err)
			}
		}()
		slog.InfoContext(ctx, "HTTP REST API server started", "port", httpPort)
	}

	// CronScheduler: load triggers from DB and start
	var cronScheduler *task.CronScheduler
	if taskRepo != nil {
		cronScheduler = task.NewCronScheduler(&cronTaskCreatorHTTPAdapter{repo: taskRepo})
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
	if sc.Managed {
		fmt.Printf("READY:%d\n", grpcServer.ActualPort())
		os.Stdout.Sync()
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

	// Stop cron scheduler
	if cronScheduler != nil {
		cronScheduler.Stop()
		slog.Info("Cron scheduler stopped")
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
	sessionRegistry *flow_registry.SessionRegistry,
	processor *session_processor.Processor,
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

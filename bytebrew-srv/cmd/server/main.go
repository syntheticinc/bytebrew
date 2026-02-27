package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/delivery/grpc"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/flow_registry"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/config"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/logger"
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

	// In managed mode, resolve data dir and override paths
	var dataDir string
	if *managed {
		dataDir = userDataDir()
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

	// Create infrastructure components (AgentService + WorkManager + AgentPool)
	components, err := infrastructure.NewInfraComponents(*cfg)
	if err != nil {
		log.Fatalf("Failed to create infrastructure components: %v", err)
	}

	// Initialize gRPC server
	grpcServer, err := initializeGRPCServer(cfg, loggerInstance, components.LicenseInfo, *managed)
	if err != nil {
		log.Fatalf("Failed to initialize gRPC server: %v", err)
	}

	// Create flow registry for managing active flows
	flowRegistry := flow_registry.NewInMemoryRegistry()

	// Create FlowHandler with multi-agent support
	pingInterval := 2 * time.Second
	flowHandlerCfg := grpc.FlowHandlerConfig{
		AgentService: components.AgentService,
		PingInterval: pingInterval,
		FlowRegistry: flowRegistry,
	}

	// Engine components are always available (server fails to start otherwise)
	factory := infrastructure.NewEngineTurnExecutorFactory(
		components.Engine,
		components.FlowManager,
		components.ToolResolver,
		components.ModelSelector,
		components.AgentConfig,
		components.WorkManager,     // taskManager (может быть nil)
		components.WorkManager,     // subtaskManager (может быть nil)
		components.AgentPoolAdapter, // agentPool (может быть nil)
		components.WebSearchTool,
		components.WebFetchTool,
		components.AgentService.GetContextReminders,
	)
	flowHandlerCfg.TurnExecutorFactory = factory

	// Wire up agent pool if available (multi-agent mode)
	if components.AgentPool != nil && components.AgentPoolAdapter != nil {
		flowHandlerCfg.AgentPoolProxy = components.AgentPool
		flowHandlerCfg.AgentPoolAdapter = components.AgentPoolAdapter
		flowHandlerCfg.WorkManager = components.WorkManager
		flowHandlerCfg.SessionStorage = components.SessionStorage
		loggerInstance.InfoContext(ctx, "Multi-agent mode enabled (Supervisor + Code Agents)")
	} else {
		loggerInstance.InfoContext(ctx, "Single-agent mode (no WorkStorage)")
	}

	flowHandler, err := grpc.NewFlowHandlerWithConfig(flowHandlerCfg)
	if err != nil {
		log.Fatalf("Failed to create flow handler: %v", err)
	}
	grpcServer.RegisterServices(flowHandler, nil, nil)

	// In managed mode, emit READY protocol before starting
	if *managed {
		fmt.Printf("READY:%d\n", grpcServer.ActualPort())
		os.Stdout.Sync()
	}

	// Start gRPC server in goroutine
	serverErrChan := make(chan error, 1)
	go func() {
		if err := grpcServer.Start(ctx); err != nil {
			serverErrChan <- err
		}
	}()

	loggerInstance.InfoContext(ctx, "ByteBrew Server started successfully",
		"host", cfg.Server.Host,
		"port", grpcServer.ActualPort(),
	)

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

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := grpcServer.Shutdown(shutdownCtx); err != nil {
		loggerInstance.ErrorContext(ctx, "Error during shutdown", "error", err)
	}

	loggerInstance.InfoContext(ctx, "ByteBrew Server stopped")
}

// initializeGRPCServer creates the gRPC server, choosing between config-based
// listener and OS-assigned port based on managed mode.
func initializeGRPCServer(cfg *config.Config, log *logger.Logger, licenseInfo *domain.LicenseInfo, managed bool) (*grpc.Server, error) {
	if managed && cfg.Server.Port == 0 {
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			return nil, fmt.Errorf("listen on random port: %w", err)
		}
		return grpc.NewServerWithListener(listener, log, licenseInfo), nil
	}
	return grpc.NewServer(cfg.Server, log, licenseInfo)
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

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

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/delivery/grpc"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/flow_registry"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/portfile"
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
		AgentService:           components.AgentService,
		ToolCallHistoryCleaner: components.AgentService.GetToolCallHistoryReminder(),
		PingInterval:           pingInterval,
		FlowRegistry:           flowRegistry,
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

	// Write port file for CLI discovery.
	// Always write 127.0.0.1 — clients connect via loopback, not 0.0.0.0.
	portFileHost := cfg.Server.Host
	if portFileHost == "" || portFileHost == "0.0.0.0" {
		portFileHost = "127.0.0.1"
	}
	portWriter := portfile.NewWriter(dataDir)
	if err := portWriter.Write(portfile.PortInfo{
		PID:       os.Getpid(),
		Port:      grpcServer.ActualPort(),
		Host:      portFileHost,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		slog.Warn("Failed to write port file", "error", err)
	} else {
		slog.Info("Port file written", "path", portWriter.Path())
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

	// Remove port file on shutdown
	if err := portWriter.Remove(); err != nil {
		slog.Warn("Failed to remove port file", "error", err)
	}

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


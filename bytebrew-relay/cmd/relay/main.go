package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	delivery "github.com/syntheticinc/bytebrew/bytebrew-relay/internal/delivery/http"
	"github.com/syntheticinc/bytebrew/bytebrew-relay/internal/infrastructure/cache"
	"github.com/syntheticinc/bytebrew/bytebrew-relay/internal/infrastructure/cloudapi"
	"github.com/syntheticinc/bytebrew/bytebrew-relay/internal/infrastructure/crypto"
	"github.com/syntheticinc/bytebrew/bytebrew-relay/internal/usecase/refresh"
	"github.com/syntheticinc/bytebrew/bytebrew-relay/internal/usecase/sessions"
	"github.com/syntheticinc/bytebrew/bytebrew-relay/internal/usecase/validate"
	"github.com/syntheticinc/bytebrew/bytebrew-relay/pkg/config"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// Determine config path
	configPath := os.Getenv("RELAY_CONFIG")
	if configPath == "" {
		configPath = "config.yaml"
	}

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Configure logging
	setupLogging(cfg.Logging.Level)

	slog.Info("starting bytebrew-relay",
		"port", cfg.Server.Port,
		"cloud_api", cfg.CloudAPI.BaseURL,
	)

	// Create infrastructure
	apiClient := cloudapi.New(cfg.CloudAPI.BaseURL, cfg.CloudAPI.AuthToken)
	licenseCache := cache.New(cfg.Cache.PersistPath, cfg.Cache.TTL, cfg.Cache.GracePeriod)

	// Load cache from disk
	if err := licenseCache.LoadFromDisk(); err != nil {
		slog.Warn("failed to load cache from disk", "error", err)
	}

	// Create adapters
	validateAdapter := cloudapi.NewValidateAdapter(apiClient)
	refreshAdapter := cloudapi.NewRefreshAdapter(apiClient)
	jwtHasher := crypto.NewJWTHasher()

	// Create usecases
	validateUC := validate.New(validateAdapter, licenseCache, jwtHasher)
	refreshUC := refresh.New(licenseCache, refreshAdapter, jwtHasher)
	sessionsUC := sessions.New(cfg.Sessions.HeartbeatTimeout)

	// Create HTTP handler and router
	handler := delivery.New(validateUC, sessionsUC, licenseCache)
	router := delivery.NewRouter(handler)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start background license refresh
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go backgroundRefresh(ctx, refreshUC, cfg.Cache.TTL)
	go sessionCleanup(ctx, sessionsUC, cfg.Sessions.CleanupInterval)

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		slog.Info("HTTP server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("listen: %w", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		slog.Info("shutdown signal received", "signal", sig)
	case err := <-errCh:
		return err
	}

	// Graceful shutdown
	cancel() // stop background goroutines

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	// Persist cache to disk
	if err := licenseCache.PersistToDisk(); err != nil {
		slog.Warn("failed to persist cache", "error", err)
	}

	slog.Info("bytebrew-relay stopped")
	return nil
}

// backgroundRefresh periodically refreshes all cached licenses via Cloud API.
func backgroundRefresh(ctx context.Context, refreshUC *refresh.Usecase, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := refreshUC.RefreshAll(ctx); err != nil {
				slog.WarnContext(ctx, "background refresh failed", "error", err)
			}
		}
	}
}

// sessionCleanup periodically removes expired sessions.
func sessionCleanup(ctx context.Context, s *sessions.Usecase, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.CleanExpired()
		}
	}
}

func setupLogging(level string) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	slog.SetDefault(slog.New(handler))
}

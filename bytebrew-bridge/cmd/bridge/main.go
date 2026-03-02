package main

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	bridgev1 "github.com/syntheticinc/bytebrew/bytebrew-bridge/api/proto/gen"
	"github.com/syntheticinc/bytebrew/bytebrew-bridge/internal/config"
	"github.com/syntheticinc/bytebrew/bytebrew-bridge/internal/relay"
	"github.com/syntheticinc/bytebrew/bytebrew-bridge/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	if err := run(); err != nil {
		slog.Error("bridge server failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	pool := relay.NewConnectionPool(cfg.AuthToken)
	srv := server.NewBridgeServer(pool)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		return fmt.Errorf("listen on port %d: %w", cfg.Port, err)
	}

	var opts []grpc.ServerOption
	if cfg.TLSCert != "" && cfg.TLSKey != "" {
		creds, err := credentials.NewServerTLSFromFile(cfg.TLSCert, cfg.TLSKey)
		if err != nil {
			return fmt.Errorf("load TLS credentials: %w", err)
		}
		opts = append(opts, grpc.Creds(creds))
		slog.Info("TLS enabled", "cert", cfg.TLSCert)
	}

	grpcServer := grpc.NewServer(opts...)
	bridgev1.RegisterBridgeServiceServer(grpcServer, srv)

	// Graceful shutdown on SIGINT/SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		slog.Info("received signal, shutting down", "signal", sig)
		grpcServer.GracefulStop()
	}()

	slog.Info("bridge server starting", "port", cfg.Port)

	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("serve: %w", err)
	}

	return nil
}

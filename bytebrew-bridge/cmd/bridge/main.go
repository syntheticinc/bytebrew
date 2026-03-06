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

	"github.com/coder/websocket"
	"github.com/syntheticinc/bytebrew/bytebrew-bridge/internal/config"
	"github.com/syntheticinc/bytebrew/bytebrew-bridge/internal/relay"
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

	wsRelay := relay.NewWsRelay(cfg.AuthToken)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /register", handleRegister(wsRelay))
	mux.HandleFunc("GET /connect", handleConnect(wsRelay))
	mux.HandleFunc("GET /health", handleHealth)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: mux,
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		slog.Info("received signal, shutting down", "signal", sig)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("http server shutdown failed", "error", err)
		}
	}()

	slog.Info("bridge server starting", "port", cfg.Port)

	var err error
	if cfg.TLSCert != "" && cfg.TLSKey != "" {
		slog.Info("TLS enabled", "cert", cfg.TLSCert)
		err = httpServer.ListenAndServeTLS(cfg.TLSCert, cfg.TLSKey)
	} else {
		err = httpServer.ListenAndServe()
	}

	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("serve http: %w", err)
	}

	return nil
}

func handleRegister(wsRelay *relay.WsRelay) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			slog.ErrorContext(r.Context(), "websocket accept failed",
				"error", err, "remote", r.RemoteAddr)
			return
		}
		defer func() { _ = conn.CloseNow() }()

		if err := wsRelay.HandleRegister(r.Context(), conn); err != nil {
			slog.ErrorContext(r.Context(), "register handler failed",
				"error", err, "remote", r.RemoteAddr)
			_ = conn.Close(websocket.StatusInternalError, err.Error())
			return
		}

		_ = conn.Close(websocket.StatusNormalClosure, "")
	}
}

func handleConnect(wsRelay *relay.WsRelay) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serverID := r.URL.Query().Get("server_id")
		deviceID := r.URL.Query().Get("device_id")

		if serverID == "" {
			http.Error(w, "server_id query parameter is required", http.StatusBadRequest)
			return
		}
		if deviceID == "" {
			http.Error(w, "device_id query parameter is required", http.StatusBadRequest)
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			slog.ErrorContext(r.Context(), "websocket accept failed",
				"error", err, "remote", r.RemoteAddr)
			return
		}
		defer func() { _ = conn.CloseNow() }()

		if err := wsRelay.HandleConnect(r.Context(), conn, serverID, deviceID); err != nil {
			slog.ErrorContext(r.Context(), "connect handler failed",
				"error", err, "remote", r.RemoteAddr,
				"server_id", serverID, "device_id", deviceID)
			_ = conn.Close(websocket.StatusInternalError, err.Error())
			return
		}

		_ = conn.Close(websocket.StatusNormalClosure, "")
	}
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

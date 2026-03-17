package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/coder/websocket"
	"github.com/syntheticinc/bytebrew/bytebrew/bridge/internal/config"
	"github.com/syntheticinc/bytebrew/bytebrew/bridge/internal/relay"
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
	mux.HandleFunc("GET /lookup", handleLookup(wsRelay))
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

// rateLimitEntry tracks request counts for a single IP.
type rateLimitEntry struct {
	count    int
	windowAt time.Time
}

// handleLookup handles GET /lookup?code=XXXXXX.
// Returns {server_id, server_public_key} for the given short code.
// Rate-limited to 10 requests per IP per minute to prevent brute force.
func handleLookup(wsRelay *relay.WsRelay) http.HandlerFunc {
	const maxPerMinute = 10

	var mu sync.Mutex
	limits := map[string]*rateLimitEntry{}

	return func(w http.ResponseWriter, r *http.Request) {
		// Rate limit by IP
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		if ip == "" {
			ip = r.RemoteAddr
		}

		now := time.Now()
		mu.Lock()
		entry, ok := limits[ip]
		if !ok || now.Sub(entry.windowAt) > time.Minute {
			limits[ip] = &rateLimitEntry{count: 1, windowAt: now}
		} else {
			entry.count++
		}
		count := limits[ip].count
		mu.Unlock()

		if count > maxPerMinute {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		code := r.URL.Query().Get("code")
		if len(code) != 6 {
			http.Error(w, "code must be 6 digits", http.StatusBadRequest)
			return
		}

		serverID, serverPublicKey, ok := wsRelay.LookupShortCode(code)
		if !ok {
			http.Error(w, "code not found or expired", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"server_id":         serverID,
			"server_public_key": serverPublicKey,
		})

		slog.InfoContext(r.Context(), "short code lookup",
			"code", code, "server_id", serverID, "ip", ip)
	}
}

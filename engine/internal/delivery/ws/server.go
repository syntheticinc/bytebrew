package ws

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
)

// Server listens for WebSocket connections from local clients (CLI).
// Binds to 127.0.0.1 only for security (no remote access).
type Server struct {
	httpServer *http.Server
	listener   net.Listener
	port       int
}

// NewServer creates a WS server bound to 127.0.0.1 with a random port.
func NewServer(handler *ConnectionHandler) (*Server, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler.ServeHTTP)

	return &Server{
		listener: listener,
		port:     listener.Addr().(*net.TCPAddr).Port,
		httpServer: &http.Server{
			Handler: mux,
			// No HTTP-level timeouts: WebSocket connections are long-lived.
			// Keepalive is handled at WS level (ping/pong).
		},
	}, nil
}

// Start begins serving WebSocket connections. Blocks until the server is shut down.
func (s *Server) Start(ctx context.Context) error {
	slog.InfoContext(ctx, "WS server started", "port", s.port)
	if err := s.httpServer.Serve(s.listener); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("ws serve: %w", err)
	}
	return nil
}

// Port returns the actual port the server is listening on.
func (s *Server) Port() int {
	return s.port
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

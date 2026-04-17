package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// Server is the HTTP server that hosts the REST API.
type Server struct {
	router     chi.Router
	httpServer *http.Server
	port       int
}

// NewServer creates a new HTTP server with standard middleware and permissive CORS.
func NewServer(port int) *Server {
	return NewServerWithCORS(port, nil)
}

// NewServerWithCORS creates a new HTTP server with standard middleware and configurable CORS.
// If allowedOrigins is nil or empty, all origins are allowed (wildcard).
func NewServerWithCORS(port int, allowedOrigins []string) *Server {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	origins := []string{"*"}
	if len(allowedOrigins) > 0 {
		origins = allowedOrigins
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-BYOK-Provider", "X-BYOK-API-Key", "X-BYOK-Model", "X-BYOK-Base-URL"},
		ExposedHeaders:   []string{"Link", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset", "Retry-After"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	return &Server{
		router: r,
		port:   port,
	}
}

// Router returns the chi router for registering routes.
func (s *Server) Router() chi.Router { return s.router }

// Start begins listening and serving HTTP requests. Blocks until shutdown.
func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	slog.Info("HTTP server starting", "port", s.port)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// widgetPathPrefix is the path under which the embeddable widget bundle is
// served. Requests under this prefix must accept cross-origin access from any
// host (that's the whole point — customers embed the script on their sites),
// regardless of the admin API's CORS allowlist.
const widgetPathPrefix = "/widget/"

// Server is the HTTP server that hosts the REST API.
type Server struct {
	router     chi.Router
	httpServer *http.Server
	port       int
}

// NewServer creates a new HTTP server with standard middleware and same-origin CORS policy.
// Use NewServerWithCORS to explicitly allow additional origins.
func NewServer(port int) *Server {
	return NewServerWithCORS(port, nil)
}

// NewServerWithCORS creates a new HTTP server with standard middleware and configurable CORS.
// If allowedOrigins is nil or empty, only same-origin requests are allowed (no wildcard).
func NewServerWithCORS(port int, allowedOrigins []string) *Server {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Widget bundle is publicly embeddable — it must accept cross-origin GET
	// (and preflight) from any host so customers can <script src="…/widget.js">
	// on their own domains. The regular admin-API CORS policy (same-origin or
	// the configured allowlist) deliberately does NOT cover it; without this
	// split the preflight returns 200 but without Access-Control-Allow-Origin,
	// which browsers treat as a CORS failure.
	widgetCORS := cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "HEAD", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		ExposedHeaders:   []string{"Content-Length", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           86400,
	})

	// Default is same-origin only — no wildcard fallback. The go-chi/cors
	// library treats an empty AllowedOrigins as "*"; explicitly deny all
	// origins via AllowOriginFunc to neutralize that. Same-origin requests
	// don't carry a CORS Origin header, so they pass through regardless.
	var apiCORS func(http.Handler) http.Handler
	if len(allowedOrigins) > 0 {
		apiCORS = cors.Handler(cors.Options{
			AllowedOrigins:   allowedOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-BYOK-Provider", "X-BYOK-API-Key", "X-BYOK-Model", "X-BYOK-Base-URL"},
			ExposedHeaders:   []string{"Link", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset", "Retry-After"},
			AllowCredentials: true,
			MaxAge:           300,
		})
	} else {
		apiCORS = cors.Handler(cors.Options{
			AllowOriginFunc: func(_ *http.Request, _ string) bool { return false },
			AllowedMethods:  []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders:  []string{"Accept", "Authorization", "Content-Type", "X-BYOK-Provider", "X-BYOK-API-Key", "X-BYOK-Model", "X-BYOK-Base-URL"},
			MaxAge:          300,
		})
	}

	// Dispatch CORS by path prefix: widget routes get the permissive policy,
	// everything else gets the admin-API policy.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if strings.HasPrefix(req.URL.Path, widgetPathPrefix) {
				widgetCORS(next).ServeHTTP(w, req)
				return
			}
			apiCORS(next).ServeHTTP(w, req)
		})
	})

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
	slog.InfoContext(context.Background(), "HTTP server starting", "port", s.port)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

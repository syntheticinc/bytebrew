package http

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates a new HTTP router with all relay endpoints registered.
func NewRouter(handler *RelayHandler) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	r.Route("/relay/v1", func(r chi.Router) {
		r.Post("/validate", handler.Validate)
		r.Post("/heartbeat", handler.Heartbeat)
		r.Post("/release", handler.Release)
		r.Get("/status", handler.Status)
	})

	return r
}

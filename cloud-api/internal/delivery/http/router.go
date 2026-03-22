package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/delivery/http/middleware"
)

// RouterConfig holds all dependencies for building the router.
type RouterConfig struct {
	AuthHandler    *AuthHandler
	LicenseHandler *LicenseHandler
	BillingHandler *BillingHandler  // nil if Stripe not configured
	WebhookHandler *WebhookHandler  // nil if Stripe not configured
	UsageHandler   *UsageHandler    // nil if not configured
	ProxyHandler   *ProxyHandler    // nil if DeepInfra not configured
	TeamHandler    *TeamHandler     // nil if not configured
	AccountHandler *AccountHandler  // nil if not configured
	TokenVerifier  middleware.TokenVerifier
	CORSOrigins    []string
}

// NewRouter creates the Chi router with all routes and middleware.
func NewRouter(cfg RouterConfig) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(maxBodySize(1 << 20)) // 1 MB
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(httprate.LimitByIP(100, time.Minute))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			slog.Error("failed to write health response", "error", err)
		}
	})

	// Webhook routes (outside /api/v1 — Stripe sends to /webhooks/stripe)
	if cfg.WebhookHandler != nil {
		r.Post("/webhooks/stripe", cfg.WebhookHandler.HandleStripe)
	}

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Public routes
		r.Post("/auth/register", cfg.AuthHandler.Register)
		r.Post("/auth/login", cfg.AuthHandler.Login)
		r.Post("/auth/refresh", cfg.AuthHandler.RefreshToken)

		// Account public routes (no auth required)
		if cfg.AccountHandler != nil {
			r.Post("/auth/forgot-password", cfg.AccountHandler.ForgotPassword)
			r.Post("/auth/reset-password", cfg.AccountHandler.ResetPassword)
		}

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuth(cfg.TokenVerifier))
			r.Post("/license/activate", cfg.LicenseHandler.Activate)
			r.Post("/license/refresh", cfg.LicenseHandler.Refresh)
			r.Get("/license/status", cfg.LicenseHandler.Status)
			r.Get("/license/download", cfg.LicenseHandler.Download)

			// Account protected routes (optional)
			if cfg.AccountHandler != nil {
				r.Post("/auth/change-password", cfg.AccountHandler.ChangePassword)
				r.Delete("/users/me", cfg.AccountHandler.DeleteAccount)
			}

			// Usage route (optional)
			if cfg.UsageHandler != nil {
				r.Get("/subscription/usage", cfg.UsageHandler.GetUsage)
			}

			// LLM proxy (optional — only when DeepInfra is configured)
			if cfg.ProxyHandler != nil {
				r.Route("/proxy", func(r chi.Router) {
					r.Use(maxBodySize(10 << 20)) // 10MB for LLM proxy requests
					r.Post("/llm", cfg.ProxyHandler.HandleProxy)
				})
			}

			// Billing routes (optional — only when Stripe is configured)
			if cfg.BillingHandler != nil {
				r.Post("/billing/checkout", cfg.BillingHandler.Checkout)
				r.Post("/billing/portal", cfg.BillingHandler.Portal)
			}

			// Team routes (optional)
			if cfg.TeamHandler != nil {
				r.Post("/teams", cfg.TeamHandler.CreateTeam)
				r.Post("/teams/invite", cfg.TeamHandler.InviteMember)
				r.Post("/teams/accept", cfg.TeamHandler.AcceptInvite)
				r.Delete("/teams/members/{userID}", cfg.TeamHandler.RemoveMember)
				r.Get("/teams/members", cfg.TeamHandler.ListMembers)
			}
		})
	})

	return r
}

// maxBodySize limits the request body size.
func maxBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

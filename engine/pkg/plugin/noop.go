package plugin

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
)

// Noop is the default CE Plugin: it adds nothing to the server.
//
// All extension points return zero values, so the server uses its built-in
// defaults (HMAC JWT, no extra middleware, no extra routes, no extra gRPC
// options, and no session admission rule).
type Noop struct{}

// JWTVerifier returns nil — the server uses the default HMAC verifier.
func (Noop) JWTVerifier() JWTVerifier { return nil }

// HTTPMiddleware returns no extra middleware.
func (Noop) HTTPMiddleware() []func(http.Handler) http.Handler { return nil }

// RegisterHTTP mounts no extra routes.
func (Noop) RegisterHTTP(chi.Router, chi.Router) {}

// GRPCServerOptions returns no extra gRPC options.
func (Noop) GRPCServerOptions() []grpc.ServerOption { return nil }

// CheckSessionAllowed always allows the session.
func (Noop) CheckSessionAllowed(context.Context) string { return "" }

// OnAgentStep is a no-op. CE has no billing/metering surface.
func (Noop) OnAgentStep(context.Context, string, int) error { return nil }

// SetTenantSeeder is a no-op. CE has no provisioning endpoint, so there is
// nothing to wire the seeder into.
func (Noop) SetTenantSeeder(TenantSeeder) {}

// Stop is a no-op.
func (Noop) Stop() {}

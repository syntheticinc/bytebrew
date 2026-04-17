// Package ee defines the plug-in point for Enterprise Edition features.
//
// CE (Community Edition) builds leave Extension nil and skip all EE behavior.
// A separate private EE module implements this interface and sets it on the
// server configuration before calling server.Run.
package ee

import (
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
)

// Extension is the plug-in point for Enterprise Edition features.
// Nil in CE mode — all EE functionality is silently skipped.
type Extension interface {
	// RegisterHTTP mounts EE-only HTTP routes (metrics, rate-limit usage, etc.).
	// mainRouter is the external router, internalRouter the management router
	// (they are the same router in single-port mode).
	RegisterHTTP(mainRouter chi.Router, internalRouter chi.Router)

	// GRPCServerOptions returns extra gRPC server options (interceptors) to
	// append to the CE chain.
	GRPCServerOptions() []grpc.ServerOption

	// CheckSessionAllowed reports whether a new WS session is allowed to
	// start. Returns "" to allow, or a non-empty reason string to reject.
	CheckSessionAllowed() string

	// Stop releases any background resources held by the extension
	// (watchers, tickers, etc.).
	Stop()
}

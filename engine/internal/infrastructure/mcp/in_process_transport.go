package mcp

import (
	"context"
	"fmt"
)

// Handler is a function that processes MCP JSON-RPC requests and returns responses.
// Used by InProcessTransport to route requests to an in-process handler
// instead of over the network.
type Handler func(ctx context.Context, req *Request) (*Response, error)

// InProcessTransport implements Transport by routing requests to an in-process handler.
// This is a pure transport with zero domain/usecase dependencies.
type InProcessTransport struct {
	handler Handler
}

// NewInProcessTransport creates a new InProcessTransport with the given handler.
// Returns an error if handler is nil.
func NewInProcessTransport(handler Handler) (*InProcessTransport, error) {
	if handler == nil {
		return nil, fmt.Errorf("mcp: InProcessTransport handler must not be nil")
	}
	return &InProcessTransport{handler: handler}, nil
}

// Start is a no-op for in-process transport (no network connection to establish).
func (t *InProcessTransport) Start(_ context.Context) error {
	return nil
}

// Send routes the request to the in-process handler.
func (t *InProcessTransport) Send(ctx context.Context, req *Request) (*Response, error) {
	if req == nil {
		return nil, fmt.Errorf("request must not be nil")
	}
	return t.handler(ctx, req)
}

// Notify is a no-op for in-process transport (notifications are fire-and-forget).
func (t *InProcessTransport) Notify(_ context.Context, _ *Request) {}

// Close is a no-op for in-process transport (no resources to release).
func (t *InProcessTransport) Close() error {
	return nil
}

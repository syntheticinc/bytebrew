package tools

import (
	"context"
	"errors"
)

// AskUserHandler answers an ask_user questionnaire without going through a
// gRPC client. It is used by the SSE path (web chat, REST API), where the
// engine itself runs the interactive loop.
type AskUserHandler func(ctx context.Context, sessionID, questionsJSON string) (string, error)

// InProcessProxy is a minimal ClientOperationsProxy implementation for flows
// that do not have a gRPC bidirectional stream (SSE/web chat). Only ask_user
// is supported — file/shell/LSP tools were parked (see
// bytebrew-archive/engine/internal/infrastructure/tools).
type InProcessProxy struct {
	askHandler       AskUserHandler
	confirmRequester ConfirmationRequester
}

// InProcessProxyOption configures an InProcessProxy.
type InProcessProxyOption func(*InProcessProxy)

// WithAskUserHandler wires the engine-side ask_user handler.
func WithAskUserHandler(h AskUserHandler) InProcessProxyOption {
	return func(p *InProcessProxy) { p.askHandler = h }
}

// WithConfirmRequester wires the confirm_before requester (exposed to
// engine_adapter via ConfirmRequester()).
func WithConfirmRequester(r ConfirmationRequester) InProcessProxyOption {
	return func(p *InProcessProxy) { p.confirmRequester = r }
}

// NewInProcessProxy builds a proxy for the SSE path.
func NewInProcessProxy(opts ...InProcessProxyOption) *InProcessProxy {
	p := &InProcessProxy{}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// AskUserQuestionnaire delegates to the configured handler. If none is set,
// the call returns an error — agents should not receive ask_user in such
// runtimes.
func (p *InProcessProxy) AskUserQuestionnaire(ctx context.Context, sessionID, questionsJSON string) (string, error) {
	if p.askHandler == nil {
		return "", errors.New("ask_user handler is not configured for this session")
	}
	return p.askHandler(ctx, sessionID, questionsJSON)
}

// ConfirmRequester exposes the injected requester. engine_adapter performs a
// type assertion on this method to wire confirm_before wrappers.
func (p *InProcessProxy) ConfirmRequester() ConfirmationRequester {
	return p.confirmRequester
}

// Dispose is a no-op retained for API compatibility with the prior
// LocalClientOperationsProxy — SSE callers defer proxy.Dispose().
func (p *InProcessProxy) Dispose() {}

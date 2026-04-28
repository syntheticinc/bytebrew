package tools

// InProcessProxy is a minimal ClientOperationsProxy implementation for flows
// that do not have a gRPC bidirectional stream (SSE/web chat). It carries
// only the confirm_before requester now; the legacy ask_user handler was
// removed alongside the ask_user tool (replaced by show_structured_output in
// form mode, which is non-blocking and emits a SESSION_EVENT_STRUCTURED_OUTPUT
// directly).
type InProcessProxy struct {
	confirmRequester ConfirmationRequester
}

// InProcessProxyOption configures an InProcessProxy.
type InProcessProxyOption func(*InProcessProxy)

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

// ConfirmRequester exposes the injected requester. engine_adapter performs a
// type assertion on this method to wire confirm_before wrappers.
func (p *InProcessProxy) ConfirmRequester() ConfirmationRequester {
	return p.confirmRequester
}

// Dispose is a no-op retained for API compatibility with the prior
// LocalClientOperationsProxy — SSE callers defer proxy.Dispose().
func (p *InProcessProxy) Dispose() {}

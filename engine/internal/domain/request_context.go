package domain

import (
	"context"
	"strings"
)

// RequestContext holds forwarded headers from the original HTTP request.
// Used for MCP header forwarding and audit log enrichment.
type RequestContext struct {
	Headers map[string]string // header name → value
}

// Get returns header value by name (case-insensitive lookup).
func (rc *RequestContext) Get(name string) string {
	if rc == nil || rc.Headers == nil {
		return ""
	}
	lower := strings.ToLower(name)
	for k, v := range rc.Headers {
		if strings.ToLower(k) == lower {
			return v
		}
	}
	return ""
}

type requestContextKey struct{}

// WithRequestContext stores RequestContext in Go context.
func WithRequestContext(ctx context.Context, rc *RequestContext) context.Context {
	return context.WithValue(ctx, requestContextKey{}, rc)
}

// GetRequestContext retrieves RequestContext from Go context (nil if not set).
func GetRequestContext(ctx context.Context) *RequestContext {
	rc, _ := ctx.Value(requestContextKey{}).(*RequestContext)
	return rc
}

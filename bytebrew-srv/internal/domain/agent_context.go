package domain

import "context"

type contextKey string

const agentIDKey contextKey = "agentID"

// WithAgentID adds agentID to context
func WithAgentID(ctx context.Context, agentID string) context.Context {
	return context.WithValue(ctx, agentIDKey, agentID)
}

// AgentIDFromContext retrieves agentID from context. Returns empty string if not found.
func AgentIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(agentIDKey).(string); ok {
		return id
	}
	return ""
}

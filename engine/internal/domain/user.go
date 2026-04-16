package domain

import (
	"context"
	"time"
)

// User represents a lazily-created identity record.
// No password storage — auth is external (JWT sub / guest UUID / admin-set).
type User struct {
	ID          string
	TenantID    string
	ExternalID  string // JWT sub / guest:uuid / admin-set
	Email       string
	DisplayName string
	Disabled    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// --- User context key ---

type userIDCtxKey struct{}

// WithUserID returns a context with the resolved user UUID set.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDCtxKey{}, userID)
}

// UserIDFromContext extracts the resolved user UUID from context.
// Returns empty string if not set.
func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userIDCtxKey{}).(string)
	return v
}

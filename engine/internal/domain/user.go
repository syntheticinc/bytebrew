package domain

import (
	"context"
	"time"
)

// User represents a system/admin user record.
// Auth is DB-backed (username + bcrypt password_hash).
// End-users are external (identified by user_sub on sessions/memories), NOT in this table.
type User struct {
	ID           string
	TenantID     string
	Username     string
	PasswordHash string
	Role         string // "admin" | "system"
	Disabled     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
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

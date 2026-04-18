package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// UserResolver looks up users by primary key.
// Admin/system users are pre-created via the `ce admin` CLI — no lazy creation.
type UserResolver interface {
	ResolveByID(ctx context.Context, id string) (userID string, err error)
}

// UserResolveMiddleware resolves the authenticated actor to a users table row.
// Must be mounted AFTER AuthMiddleware so ContextKeyActorID is set.
// Stores the resolved user UUID in the context via domain.WithUserID.
//
// For JWT auth, ContextKeyActorID holds the user UUID (claims.Subject = user.ID).
// For API token auth, ContextKeyActorID holds the token name (not a UUID) — no match.
func UserResolveMiddleware(resolver UserResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actorID, _ := r.Context().Value(ContextKeyActorID).(string)
			if actorID == "" {
				// No authenticated actor — skip resolution (public endpoints).
				next.ServeHTTP(w, r)
				return
			}

			userID, err := resolver.ResolveByID(r.Context(), actorID)
			if err != nil {
				slog.ErrorContext(r.Context(), "failed to resolve user", "actor_id", actorID, "error", err)
				// Non-fatal: don't block the request, just skip user resolution.
				next.ServeHTTP(w, r)
				return
			}
			if userID == "" {
				// Not a known user row (e.g. API token, or JWT sub is not a user UUID).
				next.ServeHTTP(w, r)
				return
			}

			ctx := domain.WithUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

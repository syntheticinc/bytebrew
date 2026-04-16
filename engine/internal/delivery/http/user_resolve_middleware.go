package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// UserResolver lazily creates/looks up users by tenant + external ID.
type UserResolver interface {
	GetOrCreate(ctx context.Context, tenantID, externalID string) (userID string, err error)
}

// UserResolveMiddleware resolves the authenticated actor to a users table row.
// Must be mounted AFTER AuthMiddleware so ContextKeyActorID is set.
// Stores the resolved user UUID in the context via domain.WithUserID.
func UserResolveMiddleware(resolver UserResolver) func(http.Handler) http.Handler {
	// Default tenant for CE (single-tenant).
	const defaultTenantID = "00000000-0000-0000-0000-000000000001"

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actorID, _ := r.Context().Value(ContextKeyActorID).(string)
			if actorID == "" {
				// No authenticated actor — skip resolution (public endpoints).
				next.ServeHTTP(w, r)
				return
			}

			tenantID := domain.TenantIDFromContext(r.Context())
			if tenantID == "" {
				tenantID = defaultTenantID
			}

			userID, err := resolver.GetOrCreate(r.Context(), tenantID, actorID)
			if err != nil {
				slog.ErrorContext(r.Context(), "failed to resolve user", "external_id", actorID, "error", err)
				// Non-fatal: don't block the request, just skip user resolution.
				next.ServeHTTP(w, r)
				return
			}

			ctx := domain.WithUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

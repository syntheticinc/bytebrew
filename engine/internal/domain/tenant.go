package domain

import "context"

// CETenantID is the fixed tenant UUID used in Community Edition (single-tenant) mode.
// All tenant-scoped tables default to this value so CE works without multi-tenancy.
const CETenantID = "00000000-0000-0000-0000-000000000001"

// --- Tenant context key ---

type tenantCtxKey struct{}

// WithTenantID returns a context with tenant_id set.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantCtxKey{}, tenantID)
}

// TenantIDFromContext extracts tenant_id from context. Returns empty string if not set.
func TenantIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(tenantCtxKey{}).(string)
	return v
}

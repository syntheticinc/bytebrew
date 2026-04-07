package domain

import (
	"context"
	"fmt"
	"time"
)

// Tenant represents a customer workspace in Cloud mode.
// Uses CloudPlan from billing.go for plan type.
type Tenant struct {
	ID        string
	Email     string
	Plan      CloudPlan
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewTenant creates a new Tenant with validation.
func NewTenant(email string, plan CloudPlan) (*Tenant, error) {
	t := &Tenant{
		Email:     email,
		Plan:      plan,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := t.Validate(); err != nil {
		return nil, err
	}
	return t, nil
}

// Validate validates the Tenant.
func (t *Tenant) Validate() error {
	if t.Email == "" {
		return fmt.Errorf("tenant email is required")
	}
	if !t.Plan.IsValid() {
		return fmt.Errorf("invalid plan: %s", t.Plan)
	}
	return nil
}

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

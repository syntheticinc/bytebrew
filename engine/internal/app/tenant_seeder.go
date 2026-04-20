package app

import (
	"context"
	"fmt"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/configrepo"
)

// engineTenantSeeder is the concrete plugin.TenantSeeder the engine wires into
// the plugin at startup. It uses the engine's real repositories so provisioning
// goes through the same code path as normal user-driven schema creation —
// tenant scoping, validation, timestamps, etc. remain consistent.
//
// Kept deliberately minimal: a single default schema named "My Workspace".
// Creating a default entry agent is a follow-up (requires a sane default model
// to be already present for the tenant, which is not guaranteed at
// provisioning time).
type engineTenantSeeder struct {
	schemaRepo *configrepo.GORMSchemaRepository
}

// SeedTenant satisfies plugin.TenantSeeder. Runs under a context scoped to the
// new tenant so repo-level tenant stamping picks the right tenant_id.
func (s *engineTenantSeeder) SeedTenant(ctx context.Context, tenantID, plan string) error {
	if tenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}
	if s.schemaRepo == nil {
		return fmt.Errorf("schema repository not configured")
	}

	// Scope the context to the new tenant so the repository stamps
	// tenant_id=<new> on inserted rows.
	ctx = domain.WithTenantID(ctx, tenantID)

	record := &configrepo.SchemaRecord{
		Name:        "My Workspace",
		Description: "Default workspace created on signup",
		ChatEnabled: false,
	}
	if err := s.schemaRepo.Create(ctx, record); err != nil {
		return fmt.Errorf("seed default schema: %w", err)
	}
	return nil
}

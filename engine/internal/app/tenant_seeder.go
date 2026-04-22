package app

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/configrepo"
)

// engineTenantSeeder is the concrete plugin.TenantSeeder the engine wires into
// the plugin at startup. It uses the engine's real repositories so provisioning
// goes through the same code path as normal user-driven schema creation —
// tenant scoping, validation, timestamps, etc. remain consistent.
//
// Seeds per-tenant:
//   1. "My Workspace" default schema (chat disabled until the user configures it)
//   2. builder-assistant system agent (editable by the user — deleting and
//      re-seeding via POST /admin/builder-assistant/restore is supported).
//      Model assignment is deferred: if no models exist yet the agent is
//      seeded without a model, and modelServiceHTTPAdapter.CreateModel picks
//      up the first user-created model to back-fill it. This matches the
//      dogfooding story: the AI Builder runs on the same engine the user
//      configures, so every tenant gets its own editable copy.
type engineTenantSeeder struct {
	schemaRepo *configrepo.GORMSchemaRepository
	db         *gorm.DB
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
	if s.db == nil {
		return fmt.Errorf("db handle not configured")
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

	// Seed the AI Builder agent for this tenant. seedBuilderAssistant is
	// tolerant of missing models (leaves ModelName empty) and idempotent
	// (updates if already present).
	seedBuilderAssistant(ctx, s.db)
	return nil
}

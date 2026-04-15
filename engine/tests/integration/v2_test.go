//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/configrepo"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
)

// newSchemaTestDB creates an in-memory SQLite database with the V2 schema +
// agent_relations tables wired up. The test focuses on the derivation
// contract introduced by Group F: schema membership comes from
// `agent_relations`, not a separate `schema_agents` join table
// (docs/architecture/agent-first-runtime.md §2.1).
func newSchemaTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Discard,
	})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.AgentModel{},
		&models.SchemaModel{},
		&models.AgentRelationModel{},
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	})

	return db
}

// TestV2_SchemaMembership_DerivedFromRelations verifies that
// GORMSchemaRepository.ListAgents derives membership from agent_relations
// (V2 §2.1 — no schema_agents join table).
func TestV2_SchemaMembership_DerivedFromRelations(t *testing.T) {
	db := newSchemaTestDB(t)
	ctx := context.Background()

	schemaRepo := configrepo.NewGORMSchemaRepository(db)
	relRepo := configrepo.NewGORMAgentRelationRepository(db)

	schema := &configrepo.SchemaRecord{Name: "support"}
	require.NoError(t, schemaRepo.Create(ctx, schema))

	// No relations → no derived members.
	members, err := schemaRepo.ListAgents(ctx, schema.ID)
	require.NoError(t, err)
	assert.Empty(t, members, "schema with no relations has no derived membership")

	// Adding a delegation makes both endpoints implicit members.
	require.NoError(t, relRepo.Create(ctx, &configrepo.AgentRelationRecord{
		SchemaID:        schema.ID,
		SourceAgentName: "triage",
		TargetAgentName: "faq",
	}))
	require.NoError(t, relRepo.Create(ctx, &configrepo.AgentRelationRecord{
		SchemaID:        schema.ID,
		SourceAgentName: "triage",
		TargetAgentName: "billing",
	}))

	members, err = schemaRepo.ListAgents(ctx, schema.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"triage", "faq", "billing"}, members)
}

// TestV2_ListSchemasForAgent_DerivedFromRelations verifies the inverse
// derivation: which schemas reference a given agent.
func TestV2_ListSchemasForAgent_DerivedFromRelations(t *testing.T) {
	db := newSchemaTestDB(t)
	ctx := context.Background()

	schemaRepo := configrepo.NewGORMSchemaRepository(db)
	relRepo := configrepo.NewGORMAgentRelationRepository(db)

	supportSchema := &configrepo.SchemaRecord{Name: "support"}
	require.NoError(t, schemaRepo.Create(ctx, supportSchema))
	salesSchema := &configrepo.SchemaRecord{Name: "sales"}
	require.NoError(t, schemaRepo.Create(ctx, salesSchema))

	// faq used in support only.
	require.NoError(t, relRepo.Create(ctx, &configrepo.AgentRelationRecord{
		SchemaID:        supportSchema.ID,
		SourceAgentName: "triage",
		TargetAgentName: "faq",
	}))
	// closer used in sales only.
	require.NoError(t, relRepo.Create(ctx, &configrepo.AgentRelationRecord{
		SchemaID:        salesSchema.ID,
		SourceAgentName: "lead",
		TargetAgentName: "closer",
	}))
	// triage appears in both schemas.
	require.NoError(t, relRepo.Create(ctx, &configrepo.AgentRelationRecord{
		SchemaID:        salesSchema.ID,
		SourceAgentName: "triage",
		TargetAgentName: "lead",
	}))

	faqSchemas, err := schemaRepo.ListSchemasForAgent(ctx, "faq")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"support"}, faqSchemas)

	triageSchemas, err := schemaRepo.ListSchemasForAgent(ctx, "triage")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"support", "sales"}, triageSchemas)

	unknown, err := schemaRepo.ListSchemasForAgent(ctx, "ghost")
	require.NoError(t, err)
	assert.Empty(t, unknown)
}

// TestV2_DeleteSchema_CascadesAgentRelations verifies the schema delete
// transaction also drops agent_relations bound to the schema.
func TestV2_DeleteSchema_CascadesAgentRelations(t *testing.T) {
	db := newSchemaTestDB(t)
	ctx := context.Background()

	schemaRepo := configrepo.NewGORMSchemaRepository(db)
	relRepo := configrepo.NewGORMAgentRelationRepository(db)

	schema := &configrepo.SchemaRecord{Name: "to-delete"}
	require.NoError(t, schemaRepo.Create(ctx, schema))

	require.NoError(t, relRepo.Create(ctx, &configrepo.AgentRelationRecord{
		SchemaID:        schema.ID,
		SourceAgentName: "a",
		TargetAgentName: "b",
	}))

	require.NoError(t, schemaRepo.Delete(ctx, schema.ID))

	rels, err := relRepo.List(ctx, schema.ID)
	require.NoError(t, err)
	assert.Empty(t, rels, "agent_relations should cascade with schema delete")
}

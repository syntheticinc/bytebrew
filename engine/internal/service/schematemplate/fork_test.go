package schematemplate

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
)

// forkTestTablesDDL provisions the SQLite-compatible tables the fork
// service touches. The `id` columns are TEXT — GORM's
// `default:gen_random_uuid()` is a Postgres-only default, so we fill ids
// manually via stubInsertModel hooks where necessary.
const forkTestTablesDDL = `
CREATE TABLE schemas (
    id             TEXT PRIMARY KEY,
    name           TEXT NOT NULL UNIQUE,
    description    TEXT,
    is_system      INTEGER NOT NULL DEFAULT 0,
    tenant_id      TEXT NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    entry_agent_id TEXT,
    created_at     DATETIME,
    updated_at     DATETIME
);
CREATE TABLE agents (
    id                 TEXT PRIMARY KEY,
    name               TEXT NOT NULL UNIQUE,
    model_id           TEXT,
    system_prompt      TEXT NOT NULL,
    lifecycle          TEXT NOT NULL DEFAULT 'persistent',
    tool_execution     TEXT NOT NULL DEFAULT 'sequential',
    max_steps          INTEGER NOT NULL DEFAULT 0,
    max_context_size   INTEGER NOT NULL DEFAULT 16000,
    max_turn_duration  INTEGER NOT NULL DEFAULT 120,
    temperature        REAL,
    top_p              REAL,
    max_tokens         INTEGER,
    stop_sequences     TEXT,
    confirm_before     TEXT,
    is_system          INTEGER NOT NULL DEFAULT 0,
    tenant_id          TEXT NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    created_at         DATETIME,
    updated_at         DATETIME
);
CREATE TABLE agent_relations (
    id                TEXT PRIMARY KEY,
    schema_id         TEXT NOT NULL,
    source_agent_name TEXT NOT NULL,
    target_agent_name TEXT NOT NULL,
    config            TEXT,
    tenant_id         TEXT NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    created_at        DATETIME,
    updated_at        DATETIME
);
CREATE TABLE triggers (
    id            TEXT PRIMARY KEY,
    type          TEXT NOT NULL,
    title         TEXT NOT NULL,
    agent_id      TEXT,
    schema_id     TEXT,
    description   TEXT,
    enabled       INTEGER NOT NULL DEFAULT 1,
    config        TEXT NOT NULL DEFAULT '{}',
    tenant_id     TEXT NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    last_fired_at DATETIME,
    created_at    DATETIME,
    updated_at    DATETIME
);
CREATE TABLE capabilities (
    id          TEXT PRIMARY KEY,
    agent_id    TEXT NOT NULL,
    type        TEXT NOT NULL,
    config      TEXT,
    enabled     INTEGER NOT NULL DEFAULT 1,
    tenant_id   TEXT NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    created_at  DATETIME,
    updated_at  DATETIME
);
`

// idAssigner is a GORM "before create" callback that fills empty string PKs
// with a fresh uuid so SQLite can distinguish rows (SQLite treats empty
// TEXT PK values as valid but distinct rows on the same value collide).
func idAssigner(tx *gorm.DB) {
	if tx.Statement == nil || tx.Statement.Dest == nil {
		return
	}
	switch v := tx.Statement.Dest.(type) {
	case *models.SchemaModel:
		if v.ID == "" {
			v.ID = uuid.NewString()
		}
	case *models.AgentModel:
		if v.ID == "" {
			v.ID = uuid.NewString()
		}
	case *models.AgentRelationModel:
		if v.ID == "" {
			v.ID = uuid.NewString()
		}
	case *models.TriggerModel:
		if v.ID == "" {
			v.ID = uuid.NewString()
		}
	case *models.CapabilityModel:
		if v.ID == "" {
			v.ID = uuid.NewString()
		}
	}
}

func setupForkDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	// Split DDL on ";\n" — gorm's Exec rejects multi-statement SQL on
	// SQLite even though the driver supports it batch-style.
	for _, stmt := range splitDDL(forkTestTablesDDL) {
		require.NoError(t, db.Exec(stmt).Error, "ddl: %s", stmt)
	}

	err = db.Callback().Create().Before("gorm:create").Register("test:id_assigner", idAssigner)
	require.NoError(t, err)

	return db
}

func splitDDL(ddl string) []string {
	var out []string
	cur := ""
	for _, line := range splitLines(ddl) {
		cur += line + "\n"
		if len(line) > 0 && line[len(line)-1] == ';' {
			out = append(out, cur)
			cur = ""
		}
	}
	return out
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

// stubTemplateReader is an in-memory TemplateReader for fork tests.
type stubTemplateReader struct {
	templates map[string]domain.SchemaTemplate
}

func newStubReader(t domain.SchemaTemplate) *stubTemplateReader {
	return &stubTemplateReader{
		templates: map[string]domain.SchemaTemplate{t.Name: t},
	}
}

func (s *stubTemplateReader) GetByName(_ context.Context, name string) (*domain.SchemaTemplate, error) {
	if t, ok := s.templates[name]; ok {
		return &t, nil
	}
	return nil, nil
}

// supportFixture builds a two-agent support template: triage delegates to
// resolver, one chat trigger, memory + knowledge capabilities.
func supportFixture() domain.SchemaTemplate {
	return domain.SchemaTemplate{
		Name:        "customer-support-basic",
		Display:     "Customer Support (Basic)",
		Description: "Triage → resolver",
		Category:    domain.SchemaTemplateCategorySupport,
		Version:     "1.0",
		Definition: domain.SchemaTemplateDefinition{
			EntryAgentName: "triage",
			Agents: []domain.SchemaTemplateAgent{
				{
					Name:         "triage",
					SystemPrompt: "You triage.",
					Capabilities: []domain.SchemaTemplateCapability{
						{Type: "memory"},
					},
				},
				{
					Name:         "resolver",
					SystemPrompt: "You resolve.",
					Capabilities: []domain.SchemaTemplateCapability{
						{Type: "knowledge"},
					},
				},
			},
			Relations: []domain.SchemaTemplateRelation{
				{Source: "triage", Target: "resolver"},
			},
			Triggers: []domain.SchemaTemplateTrigger{
				{Type: "chat", Title: "Main Chat", Enabled: true},
			},
		},
	}
}

// TestFork_HappyPath verifies logical names resolve to freshly-minted
// rows in every aggregate table and the catalog row stays untouched.
func TestFork_HappyPath(t *testing.T) {
	db := setupForkDB(t)
	tmpl := supportFixture()
	reader := newStubReader(tmpl)
	svc := NewForkService(db, reader)
	ctx := context.Background()

	forked, err := svc.Fork(ctx, "tenant-a", tmpl.Name, "acme-support")
	require.NoError(t, err)
	require.NotNil(t, forked)
	assert.Equal(t, "acme-support", forked.SchemaName)
	assert.NotEmpty(t, forked.SchemaID)
	require.Len(t, forked.AgentIDs, 2)
	assert.NotEmpty(t, forked.AgentIDs["triage"])
	assert.NotEmpty(t, forked.AgentIDs["resolver"])
	assert.NotEqual(t, forked.AgentIDs["triage"], forked.AgentIDs["resolver"])

	// 1. Schema row exists with the requested name.
	var schemaCount int64
	require.NoError(t, db.Model(&models.SchemaModel{}).Where("name = ?", "acme-support").Count(&schemaCount).Error)
	assert.Equal(t, int64(1), schemaCount)

	// 2. Agents exist with namespaced names.
	var agentNames []string
	require.NoError(t, db.Model(&models.AgentModel{}).Pluck("name", &agentNames).Error)
	assert.Contains(t, agentNames, "acme-support__triage")
	assert.Contains(t, agentNames, "acme-support__resolver")

	// 3. One delegation relation with the namespaced names.
	var relCount int64
	require.NoError(t, db.Model(&models.AgentRelationModel{}).
		Where("schema_id = ?", forked.SchemaID).
		Count(&relCount).Error)
	assert.Equal(t, int64(1), relCount)
	var rel models.AgentRelationModel
	require.NoError(t, db.Where("schema_id = ?", forked.SchemaID).First(&rel).Error)
	assert.Equal(t, "acme-support__triage", rel.SourceAgentName)
	assert.Equal(t, "acme-support__resolver", rel.TargetAgentName)

	// 4. One chat trigger pointing at the entry agent + new schema.
	var triggers []models.TriggerModel
	require.NoError(t, db.Where("schema_id = ?", forked.SchemaID).Find(&triggers).Error)
	require.Len(t, triggers, 1)
	assert.Equal(t, "chat", triggers[0].Type)
	assert.Equal(t, "Main Chat", triggers[0].Title)
	require.NotNil(t, triggers[0].AgentID)
	assert.Equal(t, forked.AgentIDs["triage"], *triggers[0].AgentID)

	// 5. Capabilities attached per-agent.
	var triageCapCount, resolverCapCount int64
	require.NoError(t, db.Model(&models.CapabilityModel{}).
		Where("agent_id = ? AND type = ?", forked.AgentIDs["triage"], "memory").
		Count(&triageCapCount).Error)
	assert.Equal(t, int64(1), triageCapCount)
	require.NoError(t, db.Model(&models.CapabilityModel{}).
		Where("agent_id = ? AND type = ?", forked.AgentIDs["resolver"], "knowledge").
		Count(&resolverCapCount).Error)
	assert.Equal(t, int64(1), resolverCapCount)
}

// TestFork_DuplicateSchemaName returns ErrSchemaNameTaken on a second
// fork that targets the same schema name.
func TestFork_DuplicateSchemaName(t *testing.T) {
	db := setupForkDB(t)
	tmpl := supportFixture()
	svc := NewForkService(db, newStubReader(tmpl))
	ctx := context.Background()

	_, err := svc.Fork(ctx, "", tmpl.Name, "acme-support")
	require.NoError(t, err)

	_, err = svc.Fork(ctx, "", tmpl.Name, "acme-support")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSchemaNameTaken)

	// Only one fork should have landed — nothing half-built from the
	// second attempt.
	var schemaCount, agentCount int64
	require.NoError(t, db.Model(&models.SchemaModel{}).Count(&schemaCount).Error)
	assert.Equal(t, int64(1), schemaCount)
	require.NoError(t, db.Model(&models.AgentModel{}).Count(&agentCount).Error)
	assert.Equal(t, int64(2), agentCount)
}

// TestFork_TemplateNotFound returns ErrTemplateNotFound when the catalog
// is empty.
func TestFork_TemplateNotFound(t *testing.T) {
	db := setupForkDB(t)
	reader := &stubTemplateReader{templates: map[string]domain.SchemaTemplate{}}
	svc := NewForkService(db, reader)

	_, err := svc.Fork(context.Background(), "", "missing", "foo")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrTemplateNotFound))
}

// TestFork_InvalidTemplate catches a dangling relation endpoint before the
// transaction opens — the fork must not write partial rows.
func TestFork_InvalidTemplate(t *testing.T) {
	db := setupForkDB(t)
	bad := supportFixture()
	// Mutate the fixture so the relation references an agent that does
	// not exist in the agents list.
	bad.Definition.Relations = []domain.SchemaTemplateRelation{
		{Source: "triage", Target: "ghost"},
	}
	svc := NewForkService(db, newStubReader(bad))

	_, err := svc.Fork(context.Background(), "", bad.Name, "acme-support")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidTemplate))

	// Nothing was written — validation ran before the tx.
	var schemaCount, agentCount int64
	require.NoError(t, db.Model(&models.SchemaModel{}).Count(&schemaCount).Error)
	assert.Equal(t, int64(0), schemaCount)
	require.NoError(t, db.Model(&models.AgentModel{}).Count(&agentCount).Error)
	assert.Equal(t, int64(0), agentCount)
}

// TestFork_EmptySchemaName rejects the empty new-schema name early.
func TestFork_EmptySchemaName(t *testing.T) {
	db := setupForkDB(t)
	svc := NewForkService(db, newStubReader(supportFixture()))

	_, err := svc.Fork(context.Background(), "", "customer-support-basic", "   ")
	require.Error(t, err)
}

// TestFork_TransactionalRollback verifies that a failure mid-transaction
// rolls the entire operation back — no partial rows remain. Simulated by
// pre-creating an agent with the namespaced name of the second template
// agent; the first agent creation succeeds, the second fails on the
// unique-name constraint, and the outer Transaction rolls the schema +
// first agent + its capabilities back.
func TestFork_TransactionalRollback(t *testing.T) {
	db := setupForkDB(t)
	tmpl := supportFixture()

	// Pre-insert an agent that will collide with "acme-support__resolver".
	require.NoError(t, db.Create(&models.AgentModel{
		ID:              uuid.NewString(),
		Name:            "acme-support__resolver",
		SystemPrompt:    "pre-existing",
		Lifecycle:       "persistent",
		ToolExecution:   "sequential",
		MaxContextSize:  16000,
		MaxTurnDuration: 120,
	}).Error)

	svc := NewForkService(db, newStubReader(tmpl))

	_, err := svc.Fork(context.Background(), "", tmpl.Name, "acme-support")
	require.Error(t, err)

	// Schema row must not exist.
	var schemaCount int64
	require.NoError(t, db.Model(&models.SchemaModel{}).Where("name = ?", "acme-support").Count(&schemaCount).Error)
	assert.Equal(t, int64(0), schemaCount, "schema row must be rolled back")

	// Triage agent must not exist — only the pre-existing resolver.
	var triageCount int64
	require.NoError(t, db.Model(&models.AgentModel{}).
		Where("name = ?", "acme-support__triage").
		Count(&triageCount).Error)
	assert.Equal(t, int64(0), triageCount, "triage agent must be rolled back")

	// No relations, triggers, or capabilities should have leaked.
	var relCount, trgCount, capCount int64
	require.NoError(t, db.Model(&models.AgentRelationModel{}).Count(&relCount).Error)
	assert.Equal(t, int64(0), relCount)
	require.NoError(t, db.Model(&models.TriggerModel{}).Count(&trgCount).Error)
	assert.Equal(t, int64(0), trgCount)
	require.NoError(t, db.Model(&models.CapabilityModel{}).Count(&capCount).Error)
	assert.Equal(t, int64(0), capCount)
}

// TestFork_IndependentFromCatalog covers the §2.2 invariant: the forked
// rows have no FK back to the catalog. Mutating the template after a fork
// must not affect the forked rows.
func TestFork_IndependentFromCatalog(t *testing.T) {
	db := setupForkDB(t)
	reader := newStubReader(supportFixture())
	svc := NewForkService(db, reader)
	ctx := context.Background()

	forked, err := svc.Fork(ctx, "", "customer-support-basic", "acme-support")
	require.NoError(t, err)

	// Mutate the in-memory template after the fork.
	reader.templates["customer-support-basic"] = domain.SchemaTemplate{
		Name: "customer-support-basic",
		Definition: domain.SchemaTemplateDefinition{
			EntryAgentName: "nobody",
		},
	}

	// Forked rows are unaffected.
	var agent models.AgentModel
	require.NoError(t, db.Where("id = ?", forked.AgentIDs["triage"]).First(&agent).Error)
	assert.Equal(t, "acme-support__triage", agent.Name)
	assert.Equal(t, "You triage.", agent.SystemPrompt)
}

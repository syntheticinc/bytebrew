package app

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
)

// setupSeedTestDB creates an in-memory SQLite DB shaped like V2 with the
// subset of tables seedBuilderChatTrigger reaches: agents + triggers.
func setupSeedTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	// AgentModel carries more columns than seedBuilderChatTrigger reads, but
	// GORM Create() still writes all of them — mirror the superset so the
	// insert does not fail with "no such column".
	require.NoError(t, db.Exec(`
CREATE TABLE agents (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	model_id TEXT,
	system_prompt TEXT,
	lifecycle TEXT,
	tool_execution TEXT,
	max_steps INTEGER,
	max_context_size INTEGER,
	max_turn_duration INTEGER,
	temperature REAL,
	top_p REAL,
	max_tokens INTEGER,
	stop_sequences TEXT,
	confirm_before TEXT,
	is_system BOOLEAN NOT NULL DEFAULT 0,
	tenant_id TEXT NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
	created_at DATETIME,
	updated_at DATETIME
)`).Error)

	// V2 (§4.1): type-specific config lives in the `config` jsonb column.
	// SQLite has no jsonb; TEXT round-trips through TriggerConfig.Value/Scan.
	// No schedule / webhook_path / on_complete_* columns exist.
	require.NoError(t, db.Exec(`
CREATE TABLE triggers (
	id TEXT PRIMARY KEY,
	type TEXT NOT NULL,
	title TEXT NOT NULL,
	schema_id TEXT,
	description TEXT,
	enabled INTEGER NOT NULL DEFAULT 1,
	config TEXT NOT NULL DEFAULT '{}',
	tenant_id TEXT NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
	last_fired_at DATETIME,
	created_at DATETIME,
	updated_at DATETIME
)`).Error)

	return db
}

// TestSeedBuilderChatTrigger_V2Shape verifies that the fresh-install seed
// produces a builder-schema chat trigger with a valid V2 `config` jsonb
// payload and makes no reference to the removed schedule / webhook_path /
// on_complete_* columns.
func TestSeedBuilderChatTrigger_V2Shape(t *testing.T) {
	db := setupSeedTestDB(t)

	// Seed a builder-assistant agent so the trigger can resolve an owner.
	now := time.Now()
	agent := &models.AgentModel{
		ID:        uuid.NewString(),
		Name:      builderAssistantName,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, db.Create(agent).Error)

	schemaID := uuid.NewString()
	seedBuilderChatTrigger(context.Background(), db, schemaID)

	// Trigger must exist for (schema, chat).
	var triggers []models.TriggerModel
	require.NoError(t, db.Where(
		"schema_id = ? AND type = ?",
		schemaID, models.TriggerTypeChat,
	).Find(&triggers).Error)
	require.Len(t, triggers, 1, "seedBuilderChatTrigger must create exactly one chat trigger")
	tr := triggers[0]

	// V2 shape contract: jsonb config exists, empty for chat triggers.
	assert.Empty(t, tr.Config.Schedule, "chat trigger must not carry a cron schedule")
	assert.Empty(t, tr.Config.WebhookPath, "chat trigger must not carry a webhook_path")
	assert.True(t, tr.Enabled, "seeded chat trigger must be enabled")

	// Idempotency: second call must not add another row.
	seedBuilderChatTrigger(context.Background(), db, schemaID)
	var count int64
	require.NoError(t, db.Model(&models.TriggerModel{}).
		Where("schema_id = ? AND type = ?", schemaID, models.TriggerTypeChat).
		Count(&count).Error)
	assert.Equal(t, int64(1), count, "seedBuilderChatTrigger must be idempotent")
}

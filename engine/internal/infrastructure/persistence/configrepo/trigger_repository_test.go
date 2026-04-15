package configrepo

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

// setupTriggerTestDB creates an in-memory SQLite DB with the V2 triggers
// table schema. SQLite has no jsonb — TEXT round-trips cleanly through the
// TriggerConfig Scan/Value implementations.
func setupTriggerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	const ddl = `
CREATE TABLE triggers (
	id TEXT PRIMARY KEY,
	type TEXT NOT NULL,
	title TEXT NOT NULL,
	agent_id TEXT,
	schema_id TEXT,
	description TEXT,
	enabled INTEGER NOT NULL DEFAULT 1,
	config TEXT NOT NULL DEFAULT '{}',
	last_fired_at DATETIME,
	created_at DATETIME,
	updated_at DATETIME
)`
	require.NoError(t, db.Exec(ddl).Error)
	return db
}

// insertTrigger inserts a minimal trigger row and returns its id.
func insertTrigger(t *testing.T, db *gorm.DB, cfg models.TriggerConfig) string {
	t.Helper()
	id := uuid.NewString()
	now := time.Now()
	require.NoError(t, db.Create(&models.TriggerModel{
		ID:        id,
		Type:      models.TriggerTypeCron,
		Title:     "Test trigger",
		Enabled:   true,
		Config:    cfg,
		CreatedAt: now,
		UpdatedAt: now,
	}).Error)
	return id
}

// TestMarkFired_StampsLastFiredAt verifies the happy path — after MarkFired
// returns, last_fired_at is non-nil and within a narrow window around the
// call.
func TestMarkFired_StampsLastFiredAt(t *testing.T) {
	db := setupTriggerTestDB(t)
	repo := NewGORMTriggerRepository(db)

	id := insertTrigger(t, db, models.TriggerConfig{Schedule: "0 * * * *"})

	before := time.Now().UTC().Add(-time.Second)
	require.NoError(t, repo.MarkFired(context.Background(), id))
	after := time.Now().UTC().Add(time.Second)

	got, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, got.LastFiredAt, "last_fired_at must be set after MarkFired")
	ts := got.LastFiredAt.UTC()
	assert.True(t, !ts.Before(before) && !ts.After(after),
		"last_fired_at %s must fall in [%s, %s]", ts, before, after)
}

// TestMarkFired_Idempotent verifies that calling MarkFired repeatedly keeps
// moving the timestamp forward — no unique constraint or stale-write fight.
func TestMarkFired_Idempotent(t *testing.T) {
	db := setupTriggerTestDB(t)
	repo := NewGORMTriggerRepository(db)

	id := insertTrigger(t, db, models.TriggerConfig{Schedule: "0 * * * *"})

	require.NoError(t, repo.MarkFired(context.Background(), id))
	first, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, first.LastFiredAt)

	// Sleep a little so the second timestamp can be distinguished from the
	// first on systems with millisecond clock granularity.
	time.Sleep(20 * time.Millisecond)

	require.NoError(t, repo.MarkFired(context.Background(), id))
	second, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, second.LastFiredAt)

	assert.True(t, !second.LastFiredAt.Before(*first.LastFiredAt),
		"second MarkFired must not roll the timestamp backwards (first=%s second=%s)",
		first.LastFiredAt, second.LastFiredAt)
}

// TestMarkFired_UnknownIDReturnsError surfaces the "trigger deleted mid-fire"
// case so callers can distinguish a stale id from a successful stamp.
func TestMarkFired_UnknownIDReturnsError(t *testing.T) {
	db := setupTriggerTestDB(t)
	repo := NewGORMTriggerRepository(db)

	err := repo.MarkFired(context.Background(), uuid.NewString())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "trigger not found")
}

// TestFindByWebhookPath_MatchesInsideConfig verifies that the V2 jsonb
// lookup returns the enabled webhook trigger whose config.webhook_path
// equals the requested path — flat `webhook_path` column is gone.
func TestFindByWebhookPath_MatchesInsideConfig(t *testing.T) {
	db := setupTriggerTestDB(t)
	repo := NewGORMTriggerRepository(db)

	hit := uuid.NewString()
	miss := uuid.NewString()
	now := time.Now()
	// Target webhook trigger — enabled, path matches.
	require.NoError(t, db.Create(&models.TriggerModel{
		ID:        hit,
		Type:      models.TriggerTypeWebhook,
		Title:     "Support webhook",
		Enabled:   true,
		Config:    models.TriggerConfig{WebhookPath: "/hooks/support"},
		CreatedAt: now,
		UpdatedAt: now,
	}).Error)
	// Same path, disabled → must not match.
	require.NoError(t, db.Create(&models.TriggerModel{
		ID:        miss,
		Type:      models.TriggerTypeWebhook,
		Title:     "Old webhook",
		Enabled:   false,
		Config:    models.TriggerConfig{WebhookPath: "/hooks/support"},
		CreatedAt: now,
		UpdatedAt: now,
	}).Error)

	got, err := repo.FindByWebhookPath(context.Background(), "/hooks/support")
	require.NoError(t, err)
	require.NotNil(t, got, "enabled webhook trigger must match")
	assert.Equal(t, hit, got.ID)

	// Unknown path → nil, no error.
	got, err = repo.FindByWebhookPath(context.Background(), "/hooks/unknown")
	require.NoError(t, err)
	assert.Nil(t, got)
}

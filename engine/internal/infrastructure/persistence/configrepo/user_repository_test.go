package configrepo

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupUserTestDB creates an in-memory SQLite DB with the users table.
// PostgreSQL-specific defaults (gen_random_uuid) are replaced with SQLite-compatible equivalents.
func setupUserTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:                                   logger.Discard,
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	const ddl = `
CREATE TABLE users (
	id           TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4)) || '-' || hex(randomblob(2)) || '-4' || substr(hex(randomblob(2)),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(hex(randomblob(2)),2) || '-' || hex(randomblob(6)))),
	tenant_id    TEXT NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
	external_id  TEXT NOT NULL,
	email        TEXT,
	display_name TEXT,
	disabled     INTEGER NOT NULL DEFAULT 0,
	created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(tenant_id, external_id)
)`
	require.NoError(t, db.Exec(ddl).Error)
	return db
}

const testTenantID = "00000000-0000-0000-0000-000000000001"

func TestGetOrCreate_CreatesNewUser(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewGORMUserRepository(db)
	ctx := context.Background()

	user, err := repo.GetOrCreate(ctx, testTenantID, "jwt-sub-123")
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.NotEmpty(t, user.ID)
	assert.Equal(t, testTenantID, user.TenantID)
	assert.Equal(t, "jwt-sub-123", user.ExternalID)
	assert.False(t, user.Disabled)
}

func TestGetOrCreate_Idempotent(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewGORMUserRepository(db)
	ctx := context.Background()

	user1, err := repo.GetOrCreate(ctx, testTenantID, "jwt-sub-456")
	require.NoError(t, err)

	user2, err := repo.GetOrCreate(ctx, testTenantID, "jwt-sub-456")
	require.NoError(t, err)

	assert.Equal(t, user1.ID, user2.ID, "same external_id must return same user row")
}

func TestGetOrCreate_DifferentExternalID(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewGORMUserRepository(db)
	ctx := context.Background()

	user1, err := repo.GetOrCreate(ctx, testTenantID, "user-a")
	require.NoError(t, err)

	user2, err := repo.GetOrCreate(ctx, testTenantID, "user-b")
	require.NoError(t, err)

	assert.NotEqual(t, user1.ID, user2.ID, "different external_id must create different rows")
}

func TestGetOrCreate_DifferentTenant(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewGORMUserRepository(db)
	ctx := context.Background()

	tenant2 := "00000000-0000-0000-0000-000000000002"

	user1, err := repo.GetOrCreate(ctx, testTenantID, "same-ext")
	require.NoError(t, err)

	user2, err := repo.GetOrCreate(ctx, tenant2, "same-ext")
	require.NoError(t, err)

	assert.NotEqual(t, user1.ID, user2.ID, "same external_id in different tenants must create different rows")
}

func TestGetByExternalID_Found(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewGORMUserRepository(db)
	ctx := context.Background()

	created, err := repo.GetOrCreate(ctx, testTenantID, "lookup-test")
	require.NoError(t, err)

	found, err := repo.GetByExternalID(ctx, testTenantID, "lookup-test")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
}

func TestGetByExternalID_NotFound(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewGORMUserRepository(db)
	ctx := context.Background()

	found, err := repo.GetByExternalID(ctx, testTenantID, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestGetByID_Found(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewGORMUserRepository(db)
	ctx := context.Background()

	created, err := repo.GetOrCreate(ctx, testTenantID, "id-test")
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, created.ExternalID, found.ExternalID)
}

func TestGetByID_NotFound(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewGORMUserRepository(db)
	ctx := context.Background()

	found, err := repo.GetByID(ctx, "00000000-0000-0000-0000-000000000099")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestUpdate_User(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewGORMUserRepository(db)
	ctx := context.Background()

	user, err := repo.GetOrCreate(ctx, testTenantID, "update-test")
	require.NoError(t, err)

	email := "test@example.com"
	displayName := "Test User"
	user.Email = &email
	user.DisplayName = &displayName

	err = repo.Update(ctx, user)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	require.NotNil(t, found.Email)
	assert.Equal(t, "test@example.com", *found.Email)
	require.NotNil(t, found.DisplayName)
	assert.Equal(t, "Test User", *found.DisplayName)
}

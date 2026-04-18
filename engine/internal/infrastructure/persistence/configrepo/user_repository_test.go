package configrepo

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
)

// setupUserTestDB creates an in-memory SQLite DB with a "users" table
// whose shape mirrors the Liquibase-managed PostgreSQL schema.
func setupUserTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:                                   logger.Discard,
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	const ddl = `
CREATE TABLE users (
	id            TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4)) || '-' || hex(randomblob(2)) || '-4' || substr(hex(randomblob(2)),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(hex(randomblob(2)),2) || '-' || hex(randomblob(6)))),
	tenant_id     TEXT NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
	username      TEXT NOT NULL,
	password_hash TEXT NOT NULL,
	role          TEXT NOT NULL DEFAULT 'admin',
	disabled      INTEGER NOT NULL DEFAULT 0,
	created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(tenant_id, username)
)`
	require.NoError(t, db.Exec(ddl).Error)
	return db
}

const testTenantID = "00000000-0000-0000-0000-000000000001"

func seedUser(t *testing.T, db *gorm.DB, tenantID, username, role string) *models.UserModel {
	t.Helper()
	u := &models.UserModel{
		TenantID:     tenantID,
		Username:     username,
		PasswordHash: "hash-not-checked-here",
		Role:         role,
	}
	require.NoError(t, db.Create(u).Error)
	return u
}

func TestGetByID_Found(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewGORMUserRepository(db)
	ctx := context.Background()

	created := seedUser(t, db, testTenantID, "alice", "admin")

	found, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, created.Username, found.Username)
	assert.Equal(t, "admin", found.Role)
}

func TestGetByID_NotFound(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewGORMUserRepository(db)
	ctx := context.Background()

	found, err := repo.GetByID(ctx, "00000000-0000-0000-0000-000000000099")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestGetByUsername_Found(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewGORMUserRepository(db)
	ctx := context.Background()

	created := seedUser(t, db, testTenantID, "bob", "admin")

	found, err := repo.GetByUsername(ctx, testTenantID, "bob")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
}

func TestGetByUsername_NotFound(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewGORMUserRepository(db)
	ctx := context.Background()

	found, err := repo.GetByUsername(ctx, testTenantID, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestGetByUsername_DifferentTenant(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewGORMUserRepository(db)
	ctx := context.Background()

	tenant2 := "00000000-0000-0000-0000-000000000002"
	seedUser(t, db, testTenantID, "shared-name", "admin")

	found, err := repo.GetByUsername(ctx, tenant2, "shared-name")
	require.NoError(t, err)
	assert.Nil(t, found, "username is scoped to tenant_id")
}

func TestUpdate_User(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewGORMUserRepository(db)
	ctx := context.Background()

	user := seedUser(t, db, testTenantID, "update-me", "admin")
	user.Disabled = true
	user.PasswordHash = "new-hash"

	require.NoError(t, repo.Update(ctx, user))

	found, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.True(t, found.Disabled)
	assert.Equal(t, "new-hash", found.PasswordHash)
}

func TestTableName_Users(t *testing.T) {
	assert.Equal(t, "users", models.UserModel{}.TableName())
}

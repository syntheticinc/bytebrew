package persistence

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"golang.org/x/crypto/curve25519"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestGormDB(t *testing.T, dsn string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.RuntimeConfigKV{}))
	return db
}

func TestServerIdentityStore_GetOrCreateIdentity(t *testing.T) {
	db := setupTestGormDB(t, ":memory:")
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})

	store := NewServerIdentityStore(db)

	// First call creates a new identity
	identity, err := store.GetOrCreateIdentity()
	require.NoError(t, err)
	require.NotNil(t, identity)
	assert.NotEmpty(t, identity.ID)
	assert.Len(t, identity.PublicKey, curve25519.PointSize)
	assert.Len(t, identity.PrivateKey, curve25519.ScalarSize)

	// Second call returns the same identity
	identity2, err := store.GetOrCreateIdentity()
	require.NoError(t, err)
	assert.Equal(t, identity.ID, identity2.ID)
	assert.Equal(t, identity.PublicKey, identity2.PublicKey)
	assert.Equal(t, identity.PrivateKey, identity2.PrivateKey)
}

func TestServerIdentityStore_PersistsAcrossInstances(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"

	// Create identity with first instance
	db1 := setupTestGormDB(t, dbPath)
	store1 := NewServerIdentityStore(db1)
	identity1, err := store1.GetOrCreateIdentity()
	require.NoError(t, err)
	sqlDB1, _ := db1.DB()
	sqlDB1.Close()

	// Load with second instance
	db2 := setupTestGormDB(t, dbPath)
	t.Cleanup(func() {
		sqlDB2, _ := db2.DB()
		sqlDB2.Close()
	})
	store2 := NewServerIdentityStore(db2)
	identity2, err := store2.GetOrCreateIdentity()
	require.NoError(t, err)

	assert.Equal(t, identity1.ID, identity2.ID)
	assert.Equal(t, identity1.PublicKey, identity2.PublicKey)
	assert.Equal(t, identity1.PrivateKey, identity2.PrivateKey)
}

// TC-P-01: Stable server_id — create identity, close DB, reopen, verify same server_id
func TestServerIdentityStore_StableServerID(t *testing.T) {
	dbPath := t.TempDir() + "/tc-p01.db"

	// Phase 1: create identity
	db1 := setupTestGormDB(t, dbPath)
	store1 := NewServerIdentityStore(db1)
	identity1, err := store1.GetOrCreateIdentity()
	require.NoError(t, err)
	require.NotEmpty(t, identity1.ID)
	sqlDB1, _ := db1.DB()
	sqlDB1.Close()

	// Phase 2: reopen and verify same server_id
	db2 := setupTestGormDB(t, dbPath)
	t.Cleanup(func() {
		sqlDB2, _ := db2.DB()
		sqlDB2.Close()
	})
	store2 := NewServerIdentityStore(db2)
	identity2, err := store2.GetOrCreateIdentity()
	require.NoError(t, err)

	assert.Equal(t, identity1.ID, identity2.ID, "server_id must be stable across DB restarts")
}

// TC-P-02: Stable keypair — create identity, close DB, reopen, verify same public+private key
func TestServerIdentityStore_StableKeypair(t *testing.T) {
	dbPath := t.TempDir() + "/tc-p02.db"

	// Phase 1: create identity
	db1 := setupTestGormDB(t, dbPath)
	store1 := NewServerIdentityStore(db1)
	identity1, err := store1.GetOrCreateIdentity()
	require.NoError(t, err)
	sqlDB1, _ := db1.DB()
	sqlDB1.Close()

	// Phase 2: reopen and verify same keypair
	db2 := setupTestGormDB(t, dbPath)
	t.Cleanup(func() {
		sqlDB2, _ := db2.DB()
		sqlDB2.Close()
	})
	store2 := NewServerIdentityStore(db2)
	identity2, err := store2.GetOrCreateIdentity()
	require.NoError(t, err)

	assert.Equal(t, identity1.PublicKey, identity2.PublicKey, "public key must be stable across DB restarts")
	assert.Equal(t, identity1.PrivateKey, identity2.PrivateKey, "private key must be stable across DB restarts")

	// Verify the loaded keypair is still mathematically valid
	expectedPub, err := curve25519.X25519(identity2.PrivateKey, curve25519.Basepoint)
	require.NoError(t, err)
	assert.Equal(t, expectedPub, identity2.PublicKey, "loaded public key must match derived from loaded private key")
}

func TestServerIdentityStore_ValidKeyPair(t *testing.T) {
	db := setupTestGormDB(t, ":memory:")
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})

	store := NewServerIdentityStore(db)

	identity, err := store.GetOrCreateIdentity()
	require.NoError(t, err)

	// Verify the public key is derived from the private key
	expectedPub, err := curve25519.X25519(identity.PrivateKey, curve25519.Basepoint)
	require.NoError(t, err)
	assert.Equal(t, expectedPub, identity.PublicKey)
}

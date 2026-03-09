package persistence

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
)

func newTestDeviceStore(t *testing.T) *SQLiteDeviceStore {
	t.Helper()
	db, err := NewWorkDB(t.TempDir() + "/test.db")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	store, err := NewSQLiteDeviceStore(db)
	require.NoError(t, err)
	return store
}

func testDevice(id, name, token string) *domain.MobileDevice {
	now := time.Now()
	return &domain.MobileDevice{
		ID:           id,
		Name:         name,
		DeviceToken:  token,
		PublicKey:    []byte("pub-key-" + id),
		SharedSecret: []byte("secret-" + id),
		PairedAt:     now,
		LastSeenAt:   now,
	}
}

func TestSQLiteDeviceStore_AddAndGetByID(t *testing.T) {
	store := newTestDeviceStore(t)
	ctx := context.Background()
	dev := testDevice("d1", "iPhone", "tok-1")

	require.NoError(t, store.Add(ctx, dev))

	got, err := store.GetByID(ctx, "d1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "d1", got.ID)
	assert.Equal(t, "iPhone", got.Name)
	assert.Equal(t, "tok-1", got.DeviceToken)
	assert.Equal(t, []byte("pub-key-d1"), got.PublicKey)
	assert.Equal(t, []byte("secret-d1"), got.SharedSecret)
}

func TestSQLiteDeviceStore_GetByID_NotFound(t *testing.T) {
	store := newTestDeviceStore(t)
	ctx := context.Background()

	got, err := store.GetByID(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestSQLiteDeviceStore_GetByToken(t *testing.T) {
	store := newTestDeviceStore(t)
	ctx := context.Background()
	require.NoError(t, store.Add(ctx, testDevice("d1", "iPhone", "tok-1")))

	got, err := store.GetByToken(ctx, "tok-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "d1", got.ID)
}

func TestSQLiteDeviceStore_List(t *testing.T) {
	store := newTestDeviceStore(t)
	ctx := context.Background()
	require.NoError(t, store.Add(ctx, testDevice("d1", "iPhone", "tok-1")))
	require.NoError(t, store.Add(ctx, testDevice("d2", "Pixel", "tok-2")))

	devices, err := store.List(ctx)
	require.NoError(t, err)
	assert.Len(t, devices, 2)
}

func TestSQLiteDeviceStore_Remove(t *testing.T) {
	store := newTestDeviceStore(t)
	ctx := context.Background()
	require.NoError(t, store.Add(ctx, testDevice("d1", "iPhone", "tok-1")))

	require.NoError(t, store.Remove(ctx, "d1"))

	got, err := store.GetByID(ctx, "d1")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestSQLiteDeviceStore_Remove_NotFound(t *testing.T) {
	store := newTestDeviceStore(t)
	ctx := context.Background()

	err := store.Remove(ctx, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "device not found")
}

func TestSQLiteDeviceStore_UpdateLastSeen(t *testing.T) {
	store := newTestDeviceStore(t)
	ctx := context.Background()
	require.NoError(t, store.Add(ctx, testDevice("d1", "iPhone", "tok-1")))

	newTime := time.Date(2030, 6, 15, 12, 0, 0, 0, time.UTC)
	require.NoError(t, store.UpdateLastSeen(ctx, "d1", newTime))

	got, err := store.GetByID(ctx, "d1")
	require.NoError(t, err)
	assert.Equal(t, newTime.Unix(), got.LastSeenAt.Unix())
}

func TestSQLiteDeviceStore_UniqueDeviceToken(t *testing.T) {
	store := newTestDeviceStore(t)
	ctx := context.Background()
	require.NoError(t, store.Add(ctx, testDevice("d1", "iPhone", "tok-same")))

	err := store.Add(ctx, testDevice("d2", "Pixel", "tok-same"))
	require.Error(t, err)
}

func TestSQLiteDeviceStore_AddInvalidDevice(t *testing.T) {
	store := newTestDeviceStore(t)
	ctx := context.Background()

	err := store.Add(ctx, &domain.MobileDevice{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validate device")
}

// --- Restart (persistence) tests ---
// These tests verify data survives a DB close + reopen cycle.

// TC-P-03: Device survives restart — add device, close DB, reopen, device found by ID
func TestSQLiteDeviceStore_DeviceSurvivesRestart(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "tc-p03.db")

	// Phase 1: add device
	store1, db1 := openDeviceStore(t, dbPath)
	dev := testDevice("d-persist", "Galaxy S24", "tok-persist")
	require.NoError(t, store1.Add(ctx, dev))
	require.NoError(t, db1.Close())

	// Phase 2: reopen and verify device is found by ID
	store2, db2 := openDeviceStore(t, dbPath)
	defer db2.Close()

	got, err := store2.GetByID(ctx, "d-persist")
	require.NoError(t, err)
	require.NotNil(t, got, "device must survive DB restart")
	assert.Equal(t, "d-persist", got.ID)
	assert.Equal(t, "Galaxy S24", got.Name)
	assert.Equal(t, "tok-persist", got.DeviceToken)
}

func openDeviceStore(t *testing.T, dbPath string) (*SQLiteDeviceStore, *sql.DB) {
	t.Helper()
	db, err := NewWorkDB(dbPath)
	require.NoError(t, err)
	store, err := NewSQLiteDeviceStore(db)
	require.NoError(t, err)
	return store, db
}

// TC-P-04: E2E after restart — add device with shared_secret, close DB, reopen, shared_secret same
func TestSQLiteDeviceStore_SharedSecretSurvivesRestart(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "restart.db")

	// Phase 1: add device with SharedSecret
	store1, db1 := openDeviceStore(t, dbPath)
	secret := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x01, 0x02, 0x03, 0x04,
		0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C,
		0x0D, 0x0E, 0x0F, 0x10, 0x11, 0x12, 0x13, 0x14,
		0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C}
	dev := testDevice("d-secret", "SecretPhone", "tok-secret")
	dev.SharedSecret = secret
	require.NoError(t, store1.Add(ctx, dev))
	require.NoError(t, db1.Close())

	// Phase 2: reopen DB and verify SharedSecret byte-for-byte
	store2, db2 := openDeviceStore(t, dbPath)
	defer db2.Close()

	got, err := store2.GetByID(ctx, "d-secret")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, secret, got.SharedSecret, "SharedSecret should survive restart byte-for-byte")
	assert.Equal(t, "SecretPhone", got.Name)
}

// TC-P-05: Multi-device restart — add 2+ devices, close DB, reopen, all found
func TestSQLiteDeviceStore_MultiDeviceRestart(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "multi.db")

	// Phase 1: add 3 devices
	store1, db1 := openDeviceStore(t, dbPath)
	require.NoError(t, store1.Add(ctx, testDevice("d1", "iPhone", "tok-1")))
	require.NoError(t, store1.Add(ctx, testDevice("d2", "Pixel", "tok-2")))
	require.NoError(t, store1.Add(ctx, testDevice("d3", "Galaxy", "tok-3")))
	require.NoError(t, db1.Close())

	// Phase 2: reopen and verify all 3 exist
	store2, db2 := openDeviceStore(t, dbPath)
	defer db2.Close()

	devices, err := store2.List(ctx)
	require.NoError(t, err)
	assert.Len(t, devices, 3)

	// Verify each device is retrievable
	for _, id := range []string{"d1", "d2", "d3"} {
		got, err := store2.GetByID(ctx, id)
		require.NoError(t, err)
		require.NotNil(t, got, "device %s should exist after restart", id)
	}
}

// TC-P-06: Revoked stays revoked — add device, remove, close DB, reopen, device not found
func TestSQLiteDeviceStore_RemoveSurvivesRestart(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "remove.db")

	// Phase 1: add device, then remove it
	store1, db1 := openDeviceStore(t, dbPath)
	require.NoError(t, store1.Add(ctx, testDevice("d-rm", "Removed", "tok-rm")))
	require.NoError(t, store1.Remove(ctx, "d-rm"))
	require.NoError(t, db1.Close())

	// Phase 2: reopen and verify the device is gone
	store2, db2 := openDeviceStore(t, dbPath)
	defer db2.Close()

	got, err := store2.GetByID(ctx, "d-rm")
	require.NoError(t, err)
	assert.Nil(t, got, "removed device should not exist after restart")

	// List should also be empty
	devices, err := store2.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, devices)
}

package mobile

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/delivery/grpc"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/usecase/pair_device"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Compile-time interface compliance checks
var _ pair_device.DeviceStore = (*SQLiteDeviceStore)(nil)
var _ grpc.DeviceAuthenticator = (*SQLiteDeviceStore)(nil)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Discard,
	})
	require.NoError(t, err, "open test db")
	return db
}

func newTestDevice(t *testing.T, id, name, token string) *domain.MobileDevice {
	t.Helper()
	device, err := domain.NewMobileDevice(id, name, token)
	require.NoError(t, err)
	device.PublicKey = []byte("pub-key-" + id)
	device.SharedSecret = []byte("secret-" + id)
	return device
}

func TestSQLiteDeviceStore_SaveAndGetDevice(t *testing.T) {
	db := setupTestDB(t)
	store, err := NewSQLiteDeviceStore(db)
	require.NoError(t, err)

	ctx := context.Background()
	device := newTestDevice(t, "dev-1", "iPhone 15", "token-abc")

	err = store.SaveDevice(ctx, device)
	require.NoError(t, err)

	got, err := store.GetDevice(ctx, "dev-1")
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, "dev-1", got.ID)
	assert.Equal(t, "iPhone 15", got.Name)
	assert.Equal(t, "token-abc", got.DeviceToken)
	assert.WithinDuration(t, device.PairedAt, got.PairedAt, time.Second)
	assert.WithinDuration(t, device.LastSeenAt, got.LastSeenAt, time.Second)
	assert.Equal(t, []byte("pub-key-dev-1"), got.PublicKey)
	assert.Equal(t, []byte("secret-dev-1"), got.SharedSecret)
}

func TestSQLiteDeviceStore_GetDeviceByToken(t *testing.T) {
	db := setupTestDB(t)
	store, err := NewSQLiteDeviceStore(db)
	require.NoError(t, err)

	ctx := context.Background()
	device := newTestDevice(t, "dev-1", "iPhone 15", "auth-token-123")

	err = store.SaveDevice(ctx, device)
	require.NoError(t, err)

	got, err := store.GetDeviceByToken(ctx, "auth-token-123")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "dev-1", got.ID)
	assert.Equal(t, "iPhone 15", got.Name)
}

func TestSQLiteDeviceStore_GetDevice_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store, err := NewSQLiteDeviceStore(db)
	require.NoError(t, err)

	ctx := context.Background()

	got, err := store.GetDevice(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestSQLiteDeviceStore_GetDeviceByToken_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store, err := NewSQLiteDeviceStore(db)
	require.NoError(t, err)

	ctx := context.Background()

	got, err := store.GetDeviceByToken(ctx, "nonexistent-token")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestSQLiteDeviceStore_ListDevices(t *testing.T) {
	db := setupTestDB(t)
	store, err := NewSQLiteDeviceStore(db)
	require.NoError(t, err)

	ctx := context.Background()

	// Empty store
	devices, err := store.ListDevices(ctx)
	require.NoError(t, err)
	assert.Empty(t, devices)

	// Add two devices
	dev1 := newTestDevice(t, "dev-1", "iPhone 15", "token-1")
	dev2 := newTestDevice(t, "dev-2", "Pixel 8", "token-2")

	require.NoError(t, store.SaveDevice(ctx, dev1))
	require.NoError(t, store.SaveDevice(ctx, dev2))

	devices, err = store.ListDevices(ctx)
	require.NoError(t, err)
	assert.Len(t, devices, 2)

	// Verify all devices are present (order may vary)
	ids := map[string]bool{}
	for _, d := range devices {
		ids[d.ID] = true
	}
	assert.True(t, ids["dev-1"])
	assert.True(t, ids["dev-2"])
}

func TestSQLiteDeviceStore_DeleteDevice(t *testing.T) {
	db := setupTestDB(t)
	store, err := NewSQLiteDeviceStore(db)
	require.NoError(t, err)

	ctx := context.Background()
	device := newTestDevice(t, "dev-1", "iPhone 15", "token-abc")

	require.NoError(t, store.SaveDevice(ctx, device))

	err = store.DeleteDevice(ctx, "dev-1")
	require.NoError(t, err)

	// Verify gone by ID
	got, err := store.GetDevice(ctx, "dev-1")
	require.NoError(t, err)
	assert.Nil(t, got)

	// Verify gone by token
	got, err = store.GetDeviceByToken(ctx, "token-abc")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestSQLiteDeviceStore_DeleteDevice_Idempotent(t *testing.T) {
	db := setupTestDB(t)
	store, err := NewSQLiteDeviceStore(db)
	require.NoError(t, err)

	ctx := context.Background()

	// Deleting a nonexistent device should not error
	err = store.DeleteDevice(ctx, "nonexistent")
	require.NoError(t, err)
}

func TestSQLiteDeviceStore_SaveDevice_Update(t *testing.T) {
	db := setupTestDB(t)
	store, err := NewSQLiteDeviceStore(db)
	require.NoError(t, err)

	ctx := context.Background()
	device := newTestDevice(t, "dev-1", "iPhone 15", "token-abc")

	require.NoError(t, store.SaveDevice(ctx, device))

	// Update name
	device.Name = "iPhone 16 Pro"
	require.NoError(t, store.SaveDevice(ctx, device))

	got, err := store.GetDevice(ctx, "dev-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "iPhone 16 Pro", got.Name)

	// Verify only one device exists
	devices, err := store.ListDevices(ctx)
	require.NoError(t, err)
	assert.Len(t, devices, 1)
}

func TestSQLiteDeviceStore_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	ctx := context.Background()

	// Phase 1: create store, save device, close DB
	db1, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: gormlogger.Discard,
	})
	require.NoError(t, err)

	store1, err := NewSQLiteDeviceStore(db1)
	require.NoError(t, err)

	device := newTestDevice(t, "persist-1", "Persistent Phone", "persist-token")
	require.NoError(t, store1.SaveDevice(ctx, device))

	sqlDB1, err := db1.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB1.Close())

	// Phase 2: reopen DB, verify data survived
	db2, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: gormlogger.Discard,
	})
	require.NoError(t, err)

	store2, err := NewSQLiteDeviceStore(db2)
	require.NoError(t, err)

	got, err := store2.GetDevice(ctx, "persist-1")
	require.NoError(t, err)
	require.NotNil(t, got, "device should survive DB close/reopen")

	assert.Equal(t, "persist-1", got.ID)
	assert.Equal(t, "Persistent Phone", got.Name)
	assert.Equal(t, "persist-token", got.DeviceToken)
	assert.Equal(t, []byte("pub-key-persist-1"), got.PublicKey)
	assert.Equal(t, []byte("secret-persist-1"), got.SharedSecret)

	// Also verify token lookup works after reopen
	gotByToken, err := store2.GetDeviceByToken(ctx, "persist-token")
	require.NoError(t, err)
	require.NotNil(t, gotByToken)
	assert.Equal(t, "persist-1", gotByToken.ID)

	sqlDB2, err := db2.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB2.Close())
}

func TestSQLiteDeviceStore_NilDB(t *testing.T) {
	store, err := NewSQLiteDeviceStore(nil)
	require.Error(t, err)
	assert.Nil(t, store)
	assert.Contains(t, err.Error(), "db is required")
}

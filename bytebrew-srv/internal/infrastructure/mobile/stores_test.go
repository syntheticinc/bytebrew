package mobile

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/usecase/pair_device"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/errors"
)

// Compile-time interface compliance checks
var _ pair_device.PairingTokenStore = (*InMemoryPairingTokenStore)(nil)
var _ pair_device.DeviceStore = (*InMemoryDeviceStore)(nil)

// --- PairingTokenStore tests ---

func TestPairingTokenStore_SaveAndGetByFullToken(t *testing.T) {
	store := NewInMemoryPairingTokenStore()
	ctx := context.Background()

	token := &domain.PairingToken{
		Token:     "abc123def456",
		ShortCode: "123456",
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	err := store.SaveToken(ctx, token)
	require.NoError(t, err)

	got, err := store.GetToken(ctx, "abc123def456")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "abc123def456", got.Token)
	assert.Equal(t, "123456", got.ShortCode)
}

func TestPairingTokenStore_GetByShortCode(t *testing.T) {
	store := NewInMemoryPairingTokenStore()
	ctx := context.Background()

	token := &domain.PairingToken{
		Token:     "fulltoken123",
		ShortCode: "654321",
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	err := store.SaveToken(ctx, token)
	require.NoError(t, err)

	got, err := store.GetToken(ctx, "654321")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "fulltoken123", got.Token)
}

func TestPairingTokenStore_GetNotFound(t *testing.T) {
	store := NewInMemoryPairingTokenStore()
	ctx := context.Background()

	got, err := store.GetToken(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestPairingTokenStore_Delete(t *testing.T) {
	store := NewInMemoryPairingTokenStore()
	ctx := context.Background()

	token := &domain.PairingToken{
		Token:     "todelete123456789012345678901234567890123456789012345678901234",
		ShortCode: "111111",
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	err := store.SaveToken(ctx, token)
	require.NoError(t, err)

	err = store.DeleteToken(ctx, token.Token)
	require.NoError(t, err)

	// Both full token and short code should be gone
	got, err := store.GetToken(ctx, token.Token)
	require.NoError(t, err)
	assert.Nil(t, got)

	got, err = store.GetToken(ctx, "111111")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestPairingTokenStore_DeleteIdempotent(t *testing.T) {
	store := NewInMemoryPairingTokenStore()
	ctx := context.Background()

	err := store.DeleteToken(ctx, "nonexistent12345678901234567890123456789012345678901234567890123")
	require.NoError(t, err)
}

func TestPairingTokenStore_OverwriteExisting(t *testing.T) {
	store := NewInMemoryPairingTokenStore()
	ctx := context.Background()

	token := &domain.PairingToken{
		Token:     "sametoken",
		ShortCode: "000000",
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	err := store.SaveToken(ctx, token)
	require.NoError(t, err)

	// Update: mark as used
	token.Used = true
	err = store.SaveToken(ctx, token)
	require.NoError(t, err)

	got, err := store.GetToken(ctx, "sametoken")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, got.Used)
}

// --- UseToken tests ---

func TestPairingTokenStore_UseToken_Success(t *testing.T) {
	store := NewInMemoryPairingTokenStore()
	ctx := context.Background()

	token := &domain.PairingToken{
		Token:     "use-token-full",
		ShortCode: "999888",
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	require.NoError(t, store.SaveToken(ctx, token))

	got, err := store.UseToken(ctx, "use-token-full")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "use-token-full", got.Token)
	assert.True(t, got.Used, "token should be marked as used")
}

func TestPairingTokenStore_UseToken_ByShortCode(t *testing.T) {
	store := NewInMemoryPairingTokenStore()
	ctx := context.Background()

	token := &domain.PairingToken{
		Token:     "use-token-short",
		ShortCode: "777666",
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	require.NoError(t, store.SaveToken(ctx, token))

	got, err := store.UseToken(ctx, "777666")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "use-token-short", got.Token)
	assert.True(t, got.Used)
}

func TestPairingTokenStore_UseToken_NotFound(t *testing.T) {
	store := NewInMemoryPairingTokenStore()
	ctx := context.Background()

	got, err := store.UseToken(ctx, "nonexistent")
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, errors.CodeNotFound))
}

func TestPairingTokenStore_UseToken_Expired(t *testing.T) {
	store := NewInMemoryPairingTokenStore()
	ctx := context.Background()

	token := &domain.PairingToken{
		Token:     "expired-token",
		ShortCode: "111222",
		ExpiresAt: time.Now().Add(-1 * time.Minute), // already expired
	}
	require.NoError(t, store.SaveToken(ctx, token))

	got, err := store.UseToken(ctx, "expired-token")
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, errors.CodeInvalidInput))
}

func TestPairingTokenStore_UseToken_AlreadyUsed(t *testing.T) {
	store := NewInMemoryPairingTokenStore()
	ctx := context.Background()

	token := &domain.PairingToken{
		Token:     "already-used-token",
		ShortCode: "333444",
		ExpiresAt: time.Now().Add(5 * time.Minute),
		Used:      true,
	}
	require.NoError(t, store.SaveToken(ctx, token))

	got, err := store.UseToken(ctx, "already-used-token")
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, errors.CodeInvalidInput))
}

func TestPairingTokenStore_UseToken_Concurrent(t *testing.T) {
	store := NewInMemoryPairingTokenStore()
	ctx := context.Background()

	token := &domain.PairingToken{
		Token:     "concurrent-token",
		ShortCode: "555666",
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	require.NoError(t, store.SaveToken(ctx, token))

	const goroutines = 50
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		successes int
		failures  int
	)

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, err := store.UseToken(ctx, "concurrent-token")
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				successes++
			} else {
				failures++
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, 1, successes, "exactly one goroutine should succeed")
	assert.Equal(t, goroutines-1, failures, "all other goroutines should fail")
}

// --- DeviceStore tests ---

func TestDeviceStore_SaveAndGet(t *testing.T) {
	store := NewInMemoryDeviceStore()
	ctx := context.Background()

	device, err := domain.NewMobileDevice("dev-1", "iPhone 15", "token-abc")
	require.NoError(t, err)

	err = store.SaveDevice(ctx, device)
	require.NoError(t, err)

	got, err := store.GetDevice(ctx, "dev-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "dev-1", got.ID)
	assert.Equal(t, "iPhone 15", got.Name)
}

func TestDeviceStore_GetNotFound(t *testing.T) {
	store := NewInMemoryDeviceStore()
	ctx := context.Background()

	got, err := store.GetDevice(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestDeviceStore_GetByToken(t *testing.T) {
	store := NewInMemoryDeviceStore()
	ctx := context.Background()

	device, err := domain.NewMobileDevice("dev-1", "iPhone 15", "auth-token-123")
	require.NoError(t, err)

	err = store.SaveDevice(ctx, device)
	require.NoError(t, err)

	got, err := store.GetDeviceByToken(ctx, "auth-token-123")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "dev-1", got.ID)
}

func TestDeviceStore_GetByTokenNotFound(t *testing.T) {
	store := NewInMemoryDeviceStore()
	ctx := context.Background()

	got, err := store.GetDeviceByToken(ctx, "nonexistent-token")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestDeviceStore_ListDevices(t *testing.T) {
	store := NewInMemoryDeviceStore()
	ctx := context.Background()

	// Empty store
	devices, err := store.ListDevices(ctx)
	require.NoError(t, err)
	assert.Empty(t, devices)

	// Add two devices
	dev1, _ := domain.NewMobileDevice("dev-1", "iPhone 15", "token-1")
	dev2, _ := domain.NewMobileDevice("dev-2", "Pixel 8", "token-2")

	require.NoError(t, store.SaveDevice(ctx, dev1))
	require.NoError(t, store.SaveDevice(ctx, dev2))

	devices, err = store.ListDevices(ctx)
	require.NoError(t, err)
	assert.Len(t, devices, 2)
}

func TestDeviceStore_Delete(t *testing.T) {
	store := NewInMemoryDeviceStore()
	ctx := context.Background()

	device, _ := domain.NewMobileDevice("dev-1", "iPhone 15", "token-abc")
	require.NoError(t, store.SaveDevice(ctx, device))

	err := store.DeleteDevice(ctx, "dev-1")
	require.NoError(t, err)

	// Both ID and token lookups should return nil
	got, err := store.GetDevice(ctx, "dev-1")
	require.NoError(t, err)
	assert.Nil(t, got)

	got, err = store.GetDeviceByToken(ctx, "token-abc")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestDeviceStore_DeleteIdempotent(t *testing.T) {
	store := NewInMemoryDeviceStore()
	ctx := context.Background()

	err := store.DeleteDevice(ctx, "nonexistent")
	require.NoError(t, err)
}

func TestDeviceStore_SaveUpdatesTokenMapping(t *testing.T) {
	store := NewInMemoryDeviceStore()
	ctx := context.Background()

	// Save device with token-1
	device, _ := domain.NewMobileDevice("dev-1", "iPhone 15", "token-old")
	require.NoError(t, store.SaveDevice(ctx, device))

	// Update device with new token
	device.DeviceToken = "token-new"
	require.NoError(t, store.SaveDevice(ctx, device))

	// Old token should not resolve
	got, err := store.GetDeviceByToken(ctx, "token-old")
	require.NoError(t, err)
	assert.Nil(t, got)

	// New token should resolve
	got, err = store.GetDeviceByToken(ctx, "token-new")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "dev-1", got.ID)
}

package bridge

import (
	"context"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence"
)

// DeviceStoreAdapter adapts persistence.SQLiteDeviceStore to the bridge.DeviceStore interface.
// The bridge layer operates without request-scoped context, so we use context.Background().
type DeviceStoreAdapter struct {
	store *persistence.SQLiteDeviceStore
}

// NewDeviceStoreAdapter creates a new adapter wrapping the given SQLiteDeviceStore.
func NewDeviceStoreAdapter(store *persistence.SQLiteDeviceStore) *DeviceStoreAdapter {
	return &DeviceStoreAdapter{store: store}
}

func (a *DeviceStoreAdapter) GetByID(id string) (*domain.MobileDevice, error) {
	return a.store.GetByID(context.Background(), id)
}

func (a *DeviceStoreAdapter) GetByToken(token string) (*domain.MobileDevice, error) {
	return a.store.GetByToken(context.Background(), token)
}

func (a *DeviceStoreAdapter) Add(device *domain.MobileDevice) error {
	return a.store.Add(context.Background(), device)
}

func (a *DeviceStoreAdapter) List() ([]*domain.MobileDevice, error) {
	return a.store.List(context.Background())
}

func (a *DeviceStoreAdapter) UpdateLastSeen(id string) error {
	return a.store.UpdateLastSeen(context.Background(), id, time.Now())
}

func (a *DeviceStoreAdapter) Remove(id string) error {
	return a.store.Remove(context.Background(), id)
}

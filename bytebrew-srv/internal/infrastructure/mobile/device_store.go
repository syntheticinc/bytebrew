package mobile

import (
	"context"
	"log/slog"
	"sync"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
)

// InMemoryDeviceStore is a thread-safe in-memory implementation of pair_device.DeviceStore.
type InMemoryDeviceStore struct {
	mu      sync.RWMutex
	devices map[string]*domain.MobileDevice // deviceID -> MobileDevice
	byToken map[string]string               // deviceToken -> deviceID (for auth lookup)
}

// NewInMemoryDeviceStore creates a new InMemoryDeviceStore.
func NewInMemoryDeviceStore() *InMemoryDeviceStore {
	return &InMemoryDeviceStore{
		devices: make(map[string]*domain.MobileDevice),
		byToken: make(map[string]string),
	}
}

// SaveDevice stores or updates a mobile device. A copy of the device is stored
// to prevent external mutations from corrupting the store's index.
func (s *InMemoryDeviceStore) SaveDevice(_ context.Context, device *domain.MobileDevice) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If device already exists, clean up old token mapping
	if existing, ok := s.devices[device.ID]; ok {
		delete(s.byToken, existing.DeviceToken)
	}

	// Store a shallow copy to protect against external pointer mutation
	copied := *device
	s.devices[device.ID] = &copied
	s.byToken[device.DeviceToken] = device.ID

	slog.Debug("device saved", "device_id", device.ID, "device_name", device.Name)
	return nil
}

// GetDevice returns a device by its ID, or nil if not found.
func (s *InMemoryDeviceStore) GetDevice(_ context.Context, deviceID string) (*domain.MobileDevice, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	device, ok := s.devices[deviceID]
	if !ok {
		return nil, nil
	}
	return device, nil
}

// GetDeviceByToken returns a device by its auth token, or nil if not found.
func (s *InMemoryDeviceStore) GetDeviceByToken(_ context.Context, deviceToken string) (*domain.MobileDevice, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deviceID, ok := s.byToken[deviceToken]
	if !ok {
		return nil, nil
	}

	device, ok := s.devices[deviceID]
	if !ok {
		return nil, nil
	}
	return device, nil
}

// ListDevices returns all stored devices.
func (s *InMemoryDeviceStore) ListDevices(_ context.Context) ([]*domain.MobileDevice, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	devices := make([]*domain.MobileDevice, 0, len(s.devices))
	for _, d := range s.devices {
		devices = append(devices, d)
	}
	return devices, nil
}

// DeleteDevice removes a device by its ID.
func (s *InMemoryDeviceStore) DeleteDevice(_ context.Context, deviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	device, ok := s.devices[deviceID]
	if !ok {
		return nil
	}

	delete(s.byToken, device.DeviceToken)
	delete(s.devices, deviceID)

	slog.Debug("device deleted", "device_id", deviceID)
	return nil
}

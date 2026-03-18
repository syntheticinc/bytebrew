package persistence

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// DeviceStore implements device persistence using GORM (PostgreSQL).
type DeviceStore struct {
	db *gorm.DB
}

// NewDeviceStore creates a new device store.
func NewDeviceStore(db *gorm.DB) *DeviceStore {
	slog.Info("device store initialized (PostgreSQL)")
	return &DeviceStore{db: db}
}

// Add persists a new paired device.
func (s *DeviceStore) Add(ctx context.Context, device *domain.MobileDevice) error {
	if err := device.Validate(); err != nil {
		return fmt.Errorf("validate device: %w", err)
	}
	m := deviceToModel(device)
	if err := s.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("insert device: %w", err)
	}
	slog.InfoContext(ctx, "device added", "device_id", device.ID, "name", device.Name)
	return nil
}

// GetByID retrieves a device by its ID.
func (s *DeviceStore) GetByID(ctx context.Context, id string) (*domain.MobileDevice, error) {
	var m models.RuntimeDeviceModel
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}
	return modelToDevice(&m), nil
}

// GetByToken retrieves a device by its device token.
func (s *DeviceStore) GetByToken(ctx context.Context, deviceToken string) (*domain.MobileDevice, error) {
	var m models.RuntimeDeviceModel
	err := s.db.WithContext(ctx).Where("device_token = ?", deviceToken).First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get device by token: %w", err)
	}
	return modelToDevice(&m), nil
}

// List returns all paired devices.
func (s *DeviceStore) List(ctx context.Context) ([]*domain.MobileDevice, error) {
	var ms []models.RuntimeDeviceModel
	err := s.db.WithContext(ctx).
		Order("paired_at DESC").
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("query devices: %w", err)
	}
	return modelsToDevices(ms), nil
}

// Remove deletes a device by its ID.
func (s *DeviceStore) Remove(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Where("id = ?", id).Delete(&models.RuntimeDeviceModel{})
	if result.Error != nil {
		return fmt.Errorf("delete device: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("device not found: %s", id)
	}
	slog.InfoContext(ctx, "device removed", "device_id", id)
	return nil
}

// UpdateLastSeen updates the last_seen_at timestamp for a device.
func (s *DeviceStore) UpdateLastSeen(ctx context.Context, id string, lastSeen time.Time) error {
	result := s.db.WithContext(ctx).
		Model(&models.RuntimeDeviceModel{}).
		Where("id = ?", id).
		Update("last_seen_at", lastSeen)
	if result.Error != nil {
		return fmt.Errorf("update last_seen: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("device not found: %s", id)
	}
	return nil
}

// Close is a no-op because the shared DB is owned by the caller.
func (s *DeviceStore) Close() error {
	return nil
}

func deviceToModel(device *domain.MobileDevice) models.RuntimeDeviceModel {
	return models.RuntimeDeviceModel{
		ID:           device.ID,
		Name:         device.Name,
		DeviceToken:  device.DeviceToken,
		PublicKey:    device.PublicKey,
		SharedSecret: device.SharedSecret,
		PairedAt:     device.PairedAt,
		LastSeenAt:   device.LastSeenAt,
	}
}

func modelToDevice(m *models.RuntimeDeviceModel) *domain.MobileDevice {
	return &domain.MobileDevice{
		ID:           m.ID,
		Name:         m.Name,
		DeviceToken:  m.DeviceToken,
		PublicKey:    m.PublicKey,
		SharedSecret: m.SharedSecret,
		PairedAt:     m.PairedAt,
		LastSeenAt:   m.LastSeenAt,
	}
}

func modelsToDevices(ms []models.RuntimeDeviceModel) []*domain.MobileDevice {
	devices := make([]*domain.MobileDevice, 0, len(ms))
	for i := range ms {
		devices = append(devices, modelToDevice(&ms[i]))
	}
	return devices
}

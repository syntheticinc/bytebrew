package mobile

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"gorm.io/gorm"
)

// deviceModel is the GORM model for mobile devices.
type deviceModel struct {
	ID           string    `gorm:"primaryKey"`
	Name         string    `gorm:"not null"`
	DeviceToken  string    `gorm:"uniqueIndex;not null"`
	PairedAt     time.Time `gorm:"not null"`
	LastSeenAt   time.Time `gorm:"not null"`
	PublicKey    []byte
	SharedSecret []byte
}

func (deviceModel) TableName() string {
	return "mobile_devices"
}

// SQLiteDeviceStore persists mobile devices in SQLite via GORM.
// Implements pair_device.DeviceStore and mobile_handler.DeviceAuthenticator.
type SQLiteDeviceStore struct {
	db *gorm.DB
}

// NewSQLiteDeviceStore creates a new SQLiteDeviceStore and auto-migrates the schema.
func NewSQLiteDeviceStore(db *gorm.DB) (*SQLiteDeviceStore, error) {
	if db == nil {
		return nil, fmt.Errorf("db is required")
	}

	if err := db.AutoMigrate(&deviceModel{}); err != nil {
		return nil, fmt.Errorf("auto-migrate mobile_devices: %w", err)
	}

	return &SQLiteDeviceStore{db: db}, nil
}

// SaveDevice stores or updates a mobile device (upsert by ID).
func (s *SQLiteDeviceStore) SaveDevice(_ context.Context, device *domain.MobileDevice) error {
	model := toModel(device)
	result := s.db.Save(&model)
	if result.Error != nil {
		return fmt.Errorf("save device: %w", result.Error)
	}
	return nil
}

// GetDevice returns a device by its ID, or nil if not found.
func (s *SQLiteDeviceStore) GetDevice(_ context.Context, deviceID string) (*domain.MobileDevice, error) {
	var model deviceModel
	result := s.db.First(&model, "id = ?", deviceID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get device: %w", result.Error)
	}
	return toDomain(&model), nil
}

// GetDeviceByToken returns a device by its auth token, or nil if not found.
func (s *SQLiteDeviceStore) GetDeviceByToken(_ context.Context, deviceToken string) (*domain.MobileDevice, error) {
	var model deviceModel
	result := s.db.First(&model, "device_token = ?", deviceToken)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get device by token: %w", result.Error)
	}
	return toDomain(&model), nil
}

// ListDevices returns all stored devices.
func (s *SQLiteDeviceStore) ListDevices(_ context.Context) ([]*domain.MobileDevice, error) {
	var models []deviceModel
	result := s.db.Find(&models)
	if result.Error != nil {
		return nil, fmt.Errorf("list devices: %w", result.Error)
	}

	devices := make([]*domain.MobileDevice, len(models))
	for i := range models {
		devices[i] = toDomain(&models[i])
	}
	return devices, nil
}

// DeleteDevice removes a device by its ID.
func (s *SQLiteDeviceStore) DeleteDevice(_ context.Context, deviceID string) error {
	result := s.db.Delete(&deviceModel{}, "id = ?", deviceID)
	if result.Error != nil {
		return fmt.Errorf("delete device: %w", result.Error)
	}
	return nil
}

// toModel converts a domain MobileDevice to a GORM model.
func toModel(d *domain.MobileDevice) deviceModel {
	return deviceModel{
		ID:           d.ID,
		Name:         d.Name,
		DeviceToken:  d.DeviceToken,
		PairedAt:     d.PairedAt,
		LastSeenAt:   d.LastSeenAt,
		PublicKey:     d.PublicKey,
		SharedSecret: d.SharedSecret,
	}
}

// toDomain converts a GORM model to a domain MobileDevice.
func toDomain(m *deviceModel) *domain.MobileDevice {
	return &domain.MobileDevice{
		ID:           m.ID,
		Name:         m.Name,
		DeviceToken:  m.DeviceToken,
		PairedAt:     m.PairedAt,
		LastSeenAt:   m.LastSeenAt,
		PublicKey:     m.PublicKey,
		SharedSecret: m.SharedSecret,
	}
}

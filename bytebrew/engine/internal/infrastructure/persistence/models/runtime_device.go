package models

import "time"

// RuntimeDeviceModel maps to the "runtime_paired_devices" table.
// Stores domain.MobileDevice data for paired mobile devices.
type RuntimeDeviceModel struct {
	ID           string    `gorm:"primaryKey;type:varchar(36)"`
	Name         string    `gorm:"type:varchar(255);not null"`
	DeviceToken  string    `gorm:"type:varchar(255);not null;uniqueIndex"`
	PublicKey    []byte    `gorm:"type:bytea"`
	SharedSecret []byte    `gorm:"type:bytea"`
	PairedAt     time.Time `gorm:"not null"`
	LastSeenAt   time.Time `gorm:"not null"`
}

func (RuntimeDeviceModel) TableName() string { return "runtime_paired_devices" }

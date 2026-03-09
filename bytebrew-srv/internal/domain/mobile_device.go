package domain

import (
	"fmt"
	"time"
)

// MobileDevice represents a paired mobile device with its cryptographic identity
type MobileDevice struct {
	ID           string
	Name         string
	DeviceToken  string
	PublicKey    []byte
	SharedSecret []byte
	PairedAt     time.Time
	LastSeenAt   time.Time
}

// Validate checks that required fields are present
func (d *MobileDevice) Validate() error {
	if d.ID == "" {
		return fmt.Errorf("device id is required")
	}
	if d.Name == "" {
		return fmt.Errorf("device name is required")
	}
	if d.DeviceToken == "" {
		return fmt.Errorf("device token is required")
	}
	return nil
}

// UpdateLastSeen sets LastSeenAt to the current time
func (d *MobileDevice) UpdateLastSeen() {
	d.LastSeenAt = time.Now()
}

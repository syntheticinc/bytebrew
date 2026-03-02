package domain

import (
	"fmt"
	"time"
)

// PairingNotification holds the outcome of a successful pairing.
// Used to notify CLI that a mobile device consumed the pairing token.
type PairingNotification struct {
	DeviceName string
	DeviceID   string
}

// MobileDevice represents a paired mobile device
type MobileDevice struct {
	ID           string
	Name         string // Human-readable name ("iPhone 15")
	DeviceToken  string // Long-lived auth token
	PairedAt     time.Time
	LastSeenAt   time.Time
	PublicKey    []byte // Mobile device X25519 public key
	SharedSecret []byte // ECDH shared secret (derived from server private key + mobile public key)
}

// NewMobileDevice creates a new MobileDevice with validation
func NewMobileDevice(id, name, deviceToken string) (*MobileDevice, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if deviceToken == "" {
		return nil, fmt.Errorf("device_token is required")
	}

	now := time.Now()
	return &MobileDevice{
		ID:          id,
		Name:        name,
		DeviceToken: deviceToken,
		PairedAt:    now,
		LastSeenAt:  now,
	}, nil
}

// Validate checks that the MobileDevice has all required fields
func (d *MobileDevice) Validate() error {
	if d.ID == "" {
		return fmt.Errorf("id is required")
	}
	if d.Name == "" {
		return fmt.Errorf("name is required")
	}
	if d.DeviceToken == "" {
		return fmt.Errorf("device_token is required")
	}
	if d.PairedAt.IsZero() {
		return fmt.Errorf("paired_at is required")
	}
	return nil
}

// UpdateLastSeen updates the last seen timestamp to now
func (d *MobileDevice) UpdateLastSeen() {
	d.LastSeenAt = time.Now()
}

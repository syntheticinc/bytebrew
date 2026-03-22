package domain

import (
	"time"
)

// PairingTokenExpiry is the default lifetime for a pairing token
const PairingTokenExpiry = 15 * time.Minute

// PairingToken holds the state for a single mobile pairing attempt
type PairingToken struct {
	Token            string
	ShortCode        string // 6-digit code for manual entry
	ExpiresAt        time.Time
	Used             bool
	ServerPublicKey  []byte // X25519 public key (ephemeral per pairing)
	ServerPrivateKey []byte // X25519 private key
}

// IsExpired returns true if the token has passed its expiration time
func (t *PairingToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsValid returns true if the token can still be used for pairing
func (t *PairingToken) IsValid() bool {
	return !t.IsExpired() && !t.Used
}

// MarkUsed marks the token as consumed
func (t *PairingToken) MarkUsed() {
	t.Used = true
}

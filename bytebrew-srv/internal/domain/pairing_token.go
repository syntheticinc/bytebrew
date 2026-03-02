package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"
)

const (
	// pairingTokenBytes is the number of random bytes for the full token (256-bit)
	pairingTokenBytes = 32
	// pairingTokenExpiry is how long a pairing token remains valid
	pairingTokenExpiry = 5 * time.Minute
	// shortCodeMax is the upper bound for generating a 6-digit code
	shortCodeMax = 1000000
)

// PairingToken represents a temporary token for mobile device pairing
type PairingToken struct {
	Token           string // Full token (256-bit, hex encoded)
	ShortCode       string // 6-digit code for manual entry
	ExpiresAt       time.Time
	Used            bool
	ServerID        string // Unique server identifier
	ServerPublicKey []byte // X25519 public key (sent to mobile)
	ServerPrivateKey []byte // X25519 private key (kept until pairing completes)
}

// NewPairingToken creates a new PairingToken with a random token, 6-digit short code, and 5 min expiry
func NewPairingToken(serverID string) (*PairingToken, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server_id is required")
	}

	tokenBytes := make([]byte, pairingTokenBytes)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	codeNum, err := rand.Int(rand.Reader, big.NewInt(shortCodeMax))
	if err != nil {
		return nil, fmt.Errorf("generate short code: %w", err)
	}
	shortCode := fmt.Sprintf("%06d", codeNum.Int64())

	return &PairingToken{
		Token:     token,
		ShortCode: shortCode,
		ExpiresAt: time.Now().Add(pairingTokenExpiry),
		Used:      false,
		ServerID:  serverID,
	}, nil
}

// IsExpired returns true if the token has expired
func (t *PairingToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsValid returns true if the token is not expired and not used
func (t *PairingToken) IsValid() bool {
	return !t.IsExpired() && !t.Used
}

// MarkUsed marks the token as used
func (t *PairingToken) MarkUsed() {
	t.Used = true
}

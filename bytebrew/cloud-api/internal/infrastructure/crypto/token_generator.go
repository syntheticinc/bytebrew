package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// SecureTokenGenerator generates cryptographically secure tokens.
type SecureTokenGenerator struct{}

// NewSecureTokenGenerator creates a new SecureTokenGenerator.
func NewSecureTokenGenerator() *SecureTokenGenerator {
	return &SecureTokenGenerator{}
}

// Generate returns a cryptographically secure random 32-byte hex string.
func (g *SecureTokenGenerator) Generate() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

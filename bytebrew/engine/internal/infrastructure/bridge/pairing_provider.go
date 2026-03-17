package bridge

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence"
	"golang.org/x/crypto/curve25519"
)

// PairingProvider generates pairing data for QR codes (token + server identity).
type PairingProvider struct {
	tokenStore     *PairingTokenStore
	serverIdentity *persistence.ServerIdentity
	bridgeURL      string
}

// NewPairingProvider creates a new PairingProvider.
func NewPairingProvider(
	tokenStore *PairingTokenStore,
	identity *persistence.ServerIdentity,
	bridgeURL string,
) *PairingProvider {
	return &PairingProvider{
		tokenStore:     tokenStore,
		serverIdentity: identity,
		bridgeURL:      bridgeURL,
	}
}

// GeneratePairingData creates a new pairing token and returns data for QR rendering.
func (p *PairingProvider) GeneratePairingData() (map[string]interface{}, error) {
	token, err := p.generateToken()
	if err != nil {
		return nil, fmt.Errorf("generate pairing token: %w", err)
	}

	p.tokenStore.Add(token)

	qrData := map[string]string{
		"server_id":         p.serverIdentity.ID,
		"server_public_key": base64.StdEncoding.EncodeToString(token.ServerPublicKey),
		"bridge_url":        p.bridgeURL,
		"token":             token.Token,
	}

	qrJSON, err := json.Marshal(qrData)
	if err != nil {
		return nil, fmt.Errorf("marshal qr data: %w", err)
	}

	return map[string]interface{}{
		"qr_data":            string(qrJSON),
		"short_code":         token.ShortCode,
		"expires_in_seconds": int(domain.PairingTokenExpiry.Seconds()),
	}, nil
}

// generateToken creates a new PairingToken with ephemeral X25519 keypair and short code.
func (p *PairingProvider) generateToken() (*domain.PairingToken, error) {
	// Generate ephemeral X25519 keypair for this pairing session
	privateKey := make([]byte, curve25519.ScalarSize)
	if _, err := rand.Read(privateKey); err != nil {
		return nil, fmt.Errorf("generate private key: %w", err)
	}

	publicKey, err := curve25519.X25519(privateKey, curve25519.Basepoint)
	if err != nil {
		return nil, fmt.Errorf("compute public key: %w", err)
	}

	// Generate random token string
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	// Generate 6-digit short code using crypto/rand
	shortCodeNum, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return nil, fmt.Errorf("generate short code: %w", err)
	}

	return &domain.PairingToken{
		Token:           base64.URLEncoding.EncodeToString(tokenBytes),
		ShortCode:       fmt.Sprintf("%06d", shortCodeNum.Int64()),
		ExpiresAt:       time.Now().Add(domain.PairingTokenExpiry),
		ServerPublicKey: publicKey,
		ServerPrivateKey: privateKey,
	}, nil
}

package crypto

import (
	"crypto/ed25519"
	"fmt"
	"time"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

// LicenseClaims are the JWT claims for license tokens.
type LicenseClaims struct {
	Email               string                `json:"email"`
	Tier                string                `json:"tier"`
	GraceUntil          *jwt.NumericDate      `json:"grace_until,omitempty"`
	Features            licenseFeaturesClaims `json:"features"`
	ProxyStepsRemaining int                   `json:"proxy_steps_remaining"`
	ProxyStepsLimit     int                   `json:"proxy_steps_limit"`
	BYOKEnabled         bool                  `json:"byok_enabled"`
	MaxSeats            int                   `json:"max_seats,omitempty"`
	jwt.RegisteredClaims
}

// LicenseSigner signs license JWTs with Ed25519.
type LicenseSigner struct {
	privateKey ed25519.PrivateKey
}

// NewLicenseSigner creates a new LicenseSigner from a private key.
func NewLicenseSigner(privateKey ed25519.PrivateKey) *LicenseSigner {
	return &LicenseSigner{privateKey: privateKey}
}

// SignLicense creates an EdDSA-signed license JWT from LicenseInfo.
func (s *LicenseSigner) SignLicense(info domain.LicenseInfo) (string, error) {
	now := time.Now()
	claims := LicenseClaims{
		Email:               info.Email,
		Tier:                string(info.Tier),
		GraceUntil:          jwt.NewNumericDate(info.GraceUntil),
		Features:            featuresFromDomain(info.Features),
		ProxyStepsRemaining: info.ProxyStepsRemaining,
		ProxyStepsLimit:     info.ProxyStepsLimit,
		BYOKEnabled:         info.BYOKEnabled,
		MaxSeats:            info.MaxSeats,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   info.UserID,
			ExpiresAt: jwt.NewNumericDate(info.ExpiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "bytebrew-cloud-api",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	return token.SignedString(s.privateKey)
}

// DomainFeatures converts the JWT features back to a domain entity.
func (c *LicenseClaims) DomainFeatures() domain.LicenseFeatures {
	return c.Features.toDomain()
}

// VerifyLicense verifies an EdDSA license JWT and returns its claims.
func VerifyLicense(publicKey ed25519.PublicKey, tokenString string) (*LicenseClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &LicenseClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return publicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse license token: %w", err)
	}
	claims, ok := token.Claims.(*LicenseClaims)
	if !ok {
		return nil, fmt.Errorf("invalid license token claims")
	}
	return claims, nil
}

package license

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

// licenseClaims represents the JWT claims structure, compatible with bytebrew-cloud-api LicenseClaims.
type licenseClaims struct {
	Email               string           `json:"email"`
	Tier                string           `json:"tier"`
	GraceUntil          *jwt.NumericDate `json:"grace_until,omitempty"`
	Features            featuresJSON     `json:"features"`
	ProxyStepsRemaining int              `json:"proxy_steps_remaining"`
	ProxyStepsLimit     int              `json:"proxy_steps_limit"`
	BYOKEnabled         bool             `json:"byok_enabled"`
	MaxSeats            int              `json:"max_seats,omitempty"`
	jwt.RegisteredClaims
}

// featuresJSON maps the JSON structure of license features.
type featuresJSON struct {
	FullAutonomy     bool `json:"full_autonomy"`
	ParallelAgents   int  `json:"parallel_agents"`
	ExploreCodebase  bool `json:"explore_codebase"`
	TraceSymbol      bool `json:"trace_symbol"`
	CodebaseIndexing bool `json:"codebase_indexing"`
}

// LicenseValidator validates Ed25519 license JWTs offline.
type LicenseValidator struct {
	publicKey ed25519.PublicKey
	nowFunc   func() time.Time // for testing; defaults to time.Now
}

// New creates a LicenseValidator from a hex-encoded Ed25519 public key.
func New(publicKeyHex string) (*LicenseValidator, error) {
	keyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return nil, fmt.Errorf("decode public key hex: %w", err)
	}
	if len(keyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: got %d, want %d", len(keyBytes), ed25519.PublicKeySize)
	}
	return &LicenseValidator{
		publicKey: ed25519.PublicKey(keyBytes),
		nowFunc:   time.Now,
	}, nil
}

// now returns the current time, using the override if set.
func (v *LicenseValidator) now() time.Time {
	if v.nowFunc != nil {
		return v.nowFunc()
	}
	return time.Now()
}

// Validate reads a license JWT from the given path and determines the license status.
// It always returns a valid LicenseInfo; on any error it falls back to BlockedLicense.
//
// Status logic:
//   - file missing      -> LicenseBlocked
//   - invalid signature -> LicenseBlocked (reject silently, log warning)
//   - now < exp         -> LicenseActive
//   - exp < now < grace -> LicenseGrace (with warning log)
//   - now > grace       -> LicenseBlocked
func (v *LicenseValidator) Validate(licensePath string) *domain.LicenseInfo {
	data, err := os.ReadFile(licensePath)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Info("no license file found, running as Blocked", "path", licensePath)
		} else {
			slog.Warn("failed to read license file, running as Blocked", "path", licensePath, "error", err)
		}
		return domain.BlockedLicense()
	}

	token, err := jwt.ParseWithClaims(string(data), &licenseClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return v.publicKey, nil
	}, jwt.WithoutClaimsValidation()) // We validate expiry manually for grace period logic.
	if err != nil {
		slog.Warn("invalid license token, running as Blocked", "error", err)
		return domain.BlockedLicense()
	}

	claims, ok := token.Claims.(*licenseClaims)
	if !ok {
		slog.Warn("invalid license claims type, running as Blocked")
		return domain.BlockedLicense()
	}

	info := v.buildLicenseInfo(claims)
	return v.determineStatus(info)
}

// buildLicenseInfo converts parsed JWT claims into a domain LicenseInfo.
func (v *LicenseValidator) buildLicenseInfo(claims *licenseClaims) *domain.LicenseInfo {
	info := &domain.LicenseInfo{
		UserID: claims.Subject,
		Email:  claims.Email,
		Tier:   domain.LicenseTier(claims.Tier),
		Features: domain.LicenseFeatures{
			FullAutonomy:     claims.Features.FullAutonomy,
			ParallelAgents:   claims.Features.ParallelAgents,
			ExploreCodebase:  claims.Features.ExploreCodebase,
			TraceSymbol:      claims.Features.TraceSymbol,
			CodebaseIndexing: claims.Features.CodebaseIndexing,
		},
		ProxyStepsRemaining: claims.ProxyStepsRemaining,
		ProxyStepsLimit:     claims.ProxyStepsLimit,
		BYOKEnabled:         claims.BYOKEnabled,
		MaxSeats:            claims.MaxSeats,
	}

	if claims.ExpiresAt != nil {
		info.ExpiresAt = claims.ExpiresAt.Time
	}
	if claims.GraceUntil != nil {
		info.GraceUntil = claims.GraceUntil.Time
	}

	return info
}

// determineStatus sets the Status field based on expiry and grace period.
func (v *LicenseValidator) determineStatus(info *domain.LicenseInfo) *domain.LicenseInfo {
	now := v.now()

	if info.ExpiresAt.IsZero() || now.Before(info.ExpiresAt) {
		info.Status = domain.LicenseActive
		return info
	}

	if !info.GraceUntil.IsZero() && now.Before(info.GraceUntil) {
		info.Status = domain.LicenseGrace
		slog.Warn("license expired but in grace period",
			"expires_at", info.ExpiresAt,
			"grace_until", info.GraceUntil,
		)
		return info
	}

	slog.Warn("license expired, blocking access",
		"expires_at", info.ExpiresAt,
		"grace_until", info.GraceUntil,
	)
	return domain.BlockedLicense()
}

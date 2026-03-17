package license

import (
	"crypto/ed25519"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTestKeyPair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	return pub, priv
}

func signTestLicense(t *testing.T, priv ed25519.PrivateKey, claims licenseClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	tokenStr, err := token.SignedString(priv)
	require.NoError(t, err)
	return tokenStr
}

func writeLicenseFile(t *testing.T, dir, tokenStr string) string {
	t.Helper()
	path := filepath.Join(dir, "license.jwt")
	err := os.WriteFile(path, []byte(tokenStr), 0644)
	require.NoError(t, err)
	return path
}

func newValidator(t *testing.T, pub ed25519.PublicKey) *LicenseValidator {
	t.Helper()
	pubHex := hex.EncodeToString(pub)
	v, err := New(pubHex)
	require.NoError(t, err)
	return v
}

func TestNew_InvalidPublicKey(t *testing.T) {
	tests := []struct {
		name      string
		keyHex    string
		wantError string
	}{
		{
			name:      "not hex",
			keyHex:    "zzzz",
			wantError: "decode public key hex",
		},
		{
			name:      "wrong size",
			keyHex:    hex.EncodeToString([]byte("tooshort")),
			wantError: "invalid public key size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.keyHex)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
		})
	}
}

func TestValidate_ValidJWT_Active(t *testing.T) {
	pub, priv := generateTestKeyPair(t)
	v := newValidator(t, pub)

	claims := licenseClaims{
		Email: "user@example.com",
		Tier:  "personal",
		Features: featuresJSON{
			FullAutonomy:     true,
			ParallelAgents:   -1,
			ExploreCodebase:  true,
			TraceSymbol:      true,
			CodebaseIndexing: true,
		},
		ProxyStepsRemaining: 250,
		ProxyStepsLimit:     300,
		BYOKEnabled:         true,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			Issuer:    "bytebrew-cloud-api",
		},
	}

	tokenStr := signTestLicense(t, priv, claims)
	path := writeLicenseFile(t, t.TempDir(), tokenStr)

	info := v.Validate(path)

	assert.Equal(t, domain.LicenseActive, info.Status)
	assert.Equal(t, domain.TierPersonal, info.Tier)
	assert.Equal(t, "user-123", info.UserID)
	assert.Equal(t, "user@example.com", info.Email)
	assert.True(t, info.Features.FullAutonomy)
	assert.Equal(t, -1, info.Features.ParallelAgents)
	assert.True(t, info.Features.ExploreCodebase)
	assert.True(t, info.Features.TraceSymbol)
	assert.True(t, info.Features.CodebaseIndexing)
	assert.Equal(t, 250, info.ProxyStepsRemaining)
	assert.Equal(t, 300, info.ProxyStepsLimit)
	assert.True(t, info.BYOKEnabled)
}

func TestValidate_ValidJWT_NoExpiry_Active(t *testing.T) {
	pub, priv := generateTestKeyPair(t)
	v := newValidator(t, pub)

	claims := licenseClaims{
		Email: "forever@example.com",
		Tier:  "teams",
		Features: featuresJSON{
			FullAutonomy:   true,
			ParallelAgents: -1,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "user-999",
			Issuer:  "bytebrew-cloud-api",
			// No ExpiresAt — should be treated as active
		},
	}

	tokenStr := signTestLicense(t, priv, claims)
	path := writeLicenseFile(t, t.TempDir(), tokenStr)

	info := v.Validate(path)

	assert.Equal(t, domain.LicenseActive, info.Status)
	assert.Equal(t, domain.TierTeams, info.Tier)
	assert.True(t, info.ExpiresAt.IsZero(), "ExpiresAt should be zero for no-expiry token")
}

func TestValidate_Expired_PastGrace_Blocked(t *testing.T) {
	pub, priv := generateTestKeyPair(t)
	v := newValidator(t, pub)

	claims := licenseClaims{
		Email:      "expired@example.com",
		Tier:       "trial",
		GraceUntil: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // grace also past
		Features: featuresJSON{
			FullAutonomy:   true,
			ParallelAgents: -1,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-456",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-48 * time.Hour)), // expired 2 days ago
			Issuer:    "bytebrew-cloud-api",
		},
	}

	tokenStr := signTestLicense(t, priv, claims)
	path := writeLicenseFile(t, t.TempDir(), tokenStr)

	info := v.Validate(path)

	// Should fall back to Blocked
	assert.Equal(t, domain.LicenseBlocked, info.Status)
	assert.Equal(t, domain.LicenseTier(""), info.Tier)
	assert.False(t, info.Features.FullAutonomy)
	assert.Equal(t, 0, info.Features.ParallelAgents)
}

func TestValidate_Expired_InGrace_Grace(t *testing.T) {
	pub, priv := generateTestKeyPair(t)
	v := newValidator(t, pub)

	claims := licenseClaims{
		Email:      "grace@example.com",
		Tier:       "trial",
		GraceUntil: jwt.NewNumericDate(time.Now().Add(72 * time.Hour)), // grace valid for 3 more days
		Features: featuresJSON{
			FullAutonomy:   true,
			ParallelAgents: -1,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-789",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // expired 1 hour ago
			Issuer:    "bytebrew-cloud-api",
		},
	}

	tokenStr := signTestLicense(t, priv, claims)
	path := writeLicenseFile(t, t.TempDir(), tokenStr)

	info := v.Validate(path)

	assert.Equal(t, domain.LicenseGrace, info.Status)
	assert.Equal(t, domain.TierTrial, info.Tier)
	assert.Equal(t, "user-789", info.UserID)
	assert.True(t, info.Features.FullAutonomy)
	assert.Equal(t, -1, info.Features.ParallelAgents)
}

func TestValidate_TamperedJWT_WrongKey_Blocked(t *testing.T) {
	pub, _ := generateTestKeyPair(t)   // key A (for validator)
	_, privB := generateTestKeyPair(t) // key B (for signing)
	v := newValidator(t, pub)

	claims := licenseClaims{
		Email: "tampered@example.com",
		Tier:  "personal",
		Features: featuresJSON{
			FullAutonomy:   true,
			ParallelAgents: -1,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-evil",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			Issuer:    "bytebrew-cloud-api",
		},
	}

	// Sign with key B, validate with key A
	tokenStr := signTestLicense(t, privB, claims)
	path := writeLicenseFile(t, t.TempDir(), tokenStr)

	info := v.Validate(path)

	assert.Equal(t, domain.LicenseBlocked, info.Status)
	assert.Equal(t, domain.LicenseTier(""), info.Tier)
	assert.False(t, info.Features.FullAutonomy)
}

func TestValidate_MissingFile_Blocked(t *testing.T) {
	pub, _ := generateTestKeyPair(t)
	v := newValidator(t, pub)

	path := filepath.Join(t.TempDir(), "nonexistent", "license.jwt")

	info := v.Validate(path)

	assert.Equal(t, domain.LicenseBlocked, info.Status)
	assert.Equal(t, domain.LicenseTier(""), info.Tier)
	assert.Equal(t, 0, info.Features.ParallelAgents)
	assert.False(t, info.Features.FullAutonomy)
}

func TestValidate_WrongAlgorithm_HMAC_Blocked(t *testing.T) {
	pub, _ := generateTestKeyPair(t)
	v := newValidator(t, pub)

	// Sign with HMAC (HS256) instead of EdDSA
	claims := licenseClaims{
		Email: "hmac@example.com",
		Tier:  "personal",
		Features: featuresJSON{
			FullAutonomy:   true,
			ParallelAgents: -1,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-hmac",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			Issuer:    "bytebrew-cloud-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte("some-secret-key-for-hmac-test!!"))
	require.NoError(t, err)

	path := writeLicenseFile(t, t.TempDir(), tokenStr)

	info := v.Validate(path)

	assert.Equal(t, domain.LicenseBlocked, info.Status)
	assert.Equal(t, domain.LicenseTier(""), info.Tier)
}

func TestValidate_CorruptedFileContent_Blocked(t *testing.T) {
	pub, _ := generateTestKeyPair(t)
	v := newValidator(t, pub)

	path := filepath.Join(t.TempDir(), "license.jwt")
	err := os.WriteFile(path, []byte("this-is-not-a-jwt"), 0644)
	require.NoError(t, err)

	info := v.Validate(path)

	assert.Equal(t, domain.LicenseBlocked, info.Status)
	assert.Equal(t, domain.LicenseTier(""), info.Tier)
}

func TestValidate_Expired_NoGrace_Blocked(t *testing.T) {
	pub, priv := generateTestKeyPair(t)
	v := newValidator(t, pub)

	claims := licenseClaims{
		Email: "nograce@example.com",
		Tier:  "trial",
		// No GraceUntil
		Features: featuresJSON{
			FullAutonomy:   true,
			ParallelAgents: -1,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-nograce",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-24 * time.Hour)), // expired yesterday
			Issuer:    "bytebrew-cloud-api",
		},
	}

	tokenStr := signTestLicense(t, priv, claims)
	path := writeLicenseFile(t, t.TempDir(), tokenStr)

	info := v.Validate(path)

	assert.Equal(t, domain.LicenseBlocked, info.Status)
	assert.Equal(t, domain.LicenseTier(""), info.Tier)
}

func TestValidate_AllFeaturesFalse(t *testing.T) {
	pub, priv := generateTestKeyPair(t)
	v := newValidator(t, pub)

	claims := licenseClaims{
		Email: "basic@example.com",
		Tier:  "trial",
		Features: featuresJSON{
			FullAutonomy:     false,
			ParallelAgents:   2,
			ExploreCodebase:  false,
			TraceSymbol:      false,
			CodebaseIndexing: false,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-basic",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			Issuer:    "bytebrew-cloud-api",
		},
	}

	tokenStr := signTestLicense(t, priv, claims)
	path := writeLicenseFile(t, t.TempDir(), tokenStr)

	info := v.Validate(path)

	assert.Equal(t, domain.LicenseActive, info.Status)
	assert.Equal(t, domain.TierTrial, info.Tier)
	assert.False(t, info.Features.FullAutonomy)
	assert.Equal(t, 2, info.Features.ParallelAgents)
	assert.False(t, info.Features.ExploreCodebase)
	assert.False(t, info.Features.TraceSymbol)
	assert.False(t, info.Features.CodebaseIndexing)
}

func TestValidate_ProxyClaims_Parsed(t *testing.T) {
	pub, priv := generateTestKeyPair(t)
	v := newValidator(t, pub)

	claims := licenseClaims{
		Email: "proxy@example.com",
		Tier:  "personal",
		Features: featuresJSON{
			FullAutonomy:   true,
			ParallelAgents: -1,
		},
		ProxyStepsRemaining: 150,
		ProxyStepsLimit:     300,
		BYOKEnabled:         false,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-proxy",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			Issuer:    "bytebrew-cloud-api",
		},
	}

	tokenStr := signTestLicense(t, priv, claims)
	path := writeLicenseFile(t, t.TempDir(), tokenStr)

	info := v.Validate(path)

	assert.Equal(t, domain.LicenseActive, info.Status)
	assert.Equal(t, 150, info.ProxyStepsRemaining)
	assert.Equal(t, 300, info.ProxyStepsLimit)
	assert.False(t, info.BYOKEnabled)
}

func TestValidate_BYOKEnabled(t *testing.T) {
	pub, priv := generateTestKeyPair(t)
	v := newValidator(t, pub)

	claims := licenseClaims{
		Email: "byok@example.com",
		Tier:  "teams",
		Features: featuresJSON{
			FullAutonomy:   true,
			ParallelAgents: -1,
		},
		BYOKEnabled: true,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-byok",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			Issuer:    "bytebrew-cloud-api",
		},
	}

	tokenStr := signTestLicense(t, priv, claims)
	path := writeLicenseFile(t, t.TempDir(), tokenStr)

	info := v.Validate(path)

	assert.Equal(t, domain.LicenseActive, info.Status)
	assert.True(t, info.BYOKEnabled)
	assert.Equal(t, domain.TierTeams, info.Tier)
}

package license

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLicenseWatcher_InitialLoad_Active(t *testing.T) {
	pub, priv := generateTestKeyPair(t)
	v := newValidator(t, pub)

	claims := licenseClaims{
		Email: "watcher@example.com",
		Tier:  "personal",
		Features: featuresJSON{
			FullAutonomy:   true,
			ParallelAgents: -1,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-w1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			Issuer:    "bytebrew-cloud-api",
		},
	}

	tokenStr := signTestLicense(t, priv, claims)
	path := writeLicenseFile(t, t.TempDir(), tokenStr)

	w := NewLicenseWatcher(v, path, time.Hour) // long interval, we only test initial load
	defer w.Stop()

	info := w.Current()
	require.NotNil(t, info)
	assert.Equal(t, domain.LicenseActive, info.Status)
	assert.Equal(t, domain.TierPersonal, info.Tier)
	assert.Equal(t, "user-w1", info.UserID)
}

func TestLicenseWatcher_MissingFile_ReturnsNil(t *testing.T) {
	pub, _ := generateTestKeyPair(t)
	v := newValidator(t, pub)

	path := filepath.Join(t.TempDir(), "nonexistent", "license.jwt")

	w := NewLicenseWatcher(v, path, time.Hour)
	defer w.Stop()

	assert.Nil(t, w.Current())
}

func TestLicenseWatcher_FileDeleted_BecomesNil(t *testing.T) {
	pub, priv := generateTestKeyPair(t)
	v := newValidator(t, pub)

	claims := licenseClaims{
		Email: "delete@example.com",
		Tier:  "personal",
		Features: featuresJSON{
			FullAutonomy:   true,
			ParallelAgents: -1,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-del",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			Issuer:    "bytebrew-cloud-api",
		},
	}

	dir := t.TempDir()
	tokenStr := signTestLicense(t, priv, claims)
	path := writeLicenseFile(t, dir, tokenStr)

	w := NewLicenseWatcher(v, path, time.Hour)
	defer w.Stop()

	// Initially active.
	require.NotNil(t, w.Current())
	assert.Equal(t, domain.LicenseActive, w.Current().Status)

	// Remove license file.
	require.NoError(t, os.Remove(path))

	// Manually trigger refresh.
	w.refresh()

	assert.Nil(t, w.Current())
}

func TestLicenseWatcher_FileChanged_UpdatesLicense(t *testing.T) {
	pub, priv := generateTestKeyPair(t)
	v := newValidator(t, pub)

	dir := t.TempDir()

	// Start with active license.
	activeClaims := licenseClaims{
		Email: "update@example.com",
		Tier:  "personal",
		Features: featuresJSON{
			FullAutonomy:   true,
			ParallelAgents: -1,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-upd",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			Issuer:    "bytebrew-cloud-api",
		},
	}

	tokenStr := signTestLicense(t, priv, activeClaims)
	path := writeLicenseFile(t, dir, tokenStr)

	w := NewLicenseWatcher(v, path, time.Hour)
	defer w.Stop()

	assert.Equal(t, domain.LicenseActive, w.Current().Status)
	assert.Equal(t, domain.TierPersonal, w.Current().Tier)

	// Overwrite with teams license.
	teamsClaims := licenseClaims{
		Email: "update@example.com",
		Tier:  "teams",
		Features: featuresJSON{
			FullAutonomy:   true,
			ParallelAgents: -1,
		},
		MaxSeats: 10,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-upd",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(48 * time.Hour)),
			Issuer:    "bytebrew-cloud-api",
		},
	}

	newTokenStr := signTestLicense(t, priv, teamsClaims)
	require.NoError(t, os.WriteFile(path, []byte(newTokenStr), 0644))

	w.refresh()

	info := w.Current()
	require.NotNil(t, info)
	assert.Equal(t, domain.LicenseActive, info.Status)
	assert.Equal(t, domain.TierTeams, info.Tier)
	assert.Equal(t, 10, info.MaxSeats)
}

func TestLicenseWatcher_Pointer_SharedWithMiddleware(t *testing.T) {
	pub, priv := generateTestKeyPair(t)
	v := newValidator(t, pub)

	claims := licenseClaims{
		Email: "ptr@example.com",
		Tier:  "personal",
		Features: featuresJSON{
			FullAutonomy:   true,
			ParallelAgents: -1,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-ptr",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			Issuer:    "bytebrew-cloud-api",
		},
	}

	tokenStr := signTestLicense(t, priv, claims)
	path := writeLicenseFile(t, t.TempDir(), tokenStr)

	w := NewLicenseWatcher(v, path, time.Hour)
	defer w.Stop()

	ptr := w.Pointer()
	info := ptr.Load()
	require.NotNil(t, info)
	assert.Equal(t, domain.LicenseActive, info.Status)

	// Verify pointer is the same atomic that Current() reads from.
	assert.Equal(t, w.Current(), ptr.Load())
}

func TestLicenseWatcher_ExpiredLicense_Grace(t *testing.T) {
	pub, priv := generateTestKeyPair(t)
	v := newValidator(t, pub)

	claims := licenseClaims{
		Email:      "grace@example.com",
		Tier:       "trial",
		GraceUntil: jwt.NewNumericDate(time.Now().Add(72 * time.Hour)),
		Features: featuresJSON{
			FullAutonomy:   true,
			ParallelAgents: -1,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-grace",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			Issuer:    "bytebrew-cloud-api",
		},
	}

	tokenStr := signTestLicense(t, priv, claims)
	path := writeLicenseFile(t, t.TempDir(), tokenStr)

	w := NewLicenseWatcher(v, path, time.Hour)
	defer w.Stop()

	info := w.Current()
	require.NotNil(t, info)
	assert.Equal(t, domain.LicenseGrace, info.Status)
}

func TestLicenseWatcher_StopPreventsRefresh(t *testing.T) {
	pub, priv := generateTestKeyPair(t)
	v := newValidator(t, pub)

	claims := licenseClaims{
		Email: "stop@example.com",
		Tier:  "personal",
		Features: featuresJSON{
			FullAutonomy:   true,
			ParallelAgents: -1,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-stop",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			Issuer:    "bytebrew-cloud-api",
		},
	}

	tokenStr := signTestLicense(t, priv, claims)
	path := writeLicenseFile(t, t.TempDir(), tokenStr)

	w := NewLicenseWatcher(v, path, 10*time.Millisecond)
	w.Start()

	// Allow at least one tick.
	time.Sleep(50 * time.Millisecond)

	// Stop should not panic and goroutine should exit.
	w.Stop()
}

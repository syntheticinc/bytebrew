package http

import (
	"net/http"
	"sync/atomic"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

const (
	eeUpgradeURL      = "https://bytebrew.ai/billing"
	eeLicenseRequired  = "Enterprise Edition license required"
	eeLicenseExpired   = "Enterprise license expired"
	eeLicenseWarning   = "License expires soon, renew at https://bytebrew.ai/billing"
)

// EEMiddleware gates HTTP endpoints behind an Enterprise Edition license.
// It reads the current license atomically from a shared pointer, allowing
// a background watcher to update the license without locks.
type EEMiddleware struct {
	license *atomic.Pointer[domain.LicenseInfo]
}

// NewEEMiddleware creates a new EEMiddleware.
func NewEEMiddleware(license *atomic.Pointer[domain.LicenseInfo]) *EEMiddleware {
	return &EEMiddleware{license: license}
}

// RequireEE returns middleware that rejects requests without a valid EE license.
func (m *EEMiddleware) RequireEE(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := m.license.Load()

		if info == nil {
			writeJSON(w, http.StatusForbidden, map[string]string{
				"error":       eeLicenseRequired,
				"upgrade_url": eeUpgradeURL,
			})
			return
		}

		if info.Status == domain.LicenseBlocked {
			writeJSON(w, http.StatusForbidden, map[string]string{
				"error":       eeLicenseExpired,
				"upgrade_url": eeUpgradeURL,
			})
			return
		}

		if info.Status == domain.LicenseGrace {
			w.Header().Set("X-License-Warning", eeLicenseWarning)
		}

		next.ServeHTTP(w, r)
	})
}

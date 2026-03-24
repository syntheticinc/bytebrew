package license

import (
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// LicenseWatcher periodically re-validates the license file and updates
// an atomic pointer so consumers (HTTP middleware, gRPC interceptors) can
// read the current license without locks.
type LicenseWatcher struct {
	validator   *LicenseValidator
	licensePath string
	current     atomic.Pointer[domain.LicenseInfo]
	interval    time.Duration
	stopCh      chan struct{}
}

// NewLicenseWatcher creates a watcher that re-reads licensePath every interval.
// It performs an initial validation immediately so Current() is usable right away.
func NewLicenseWatcher(validator *LicenseValidator, licensePath string, interval time.Duration) *LicenseWatcher {
	w := &LicenseWatcher{
		validator:   validator,
		licensePath: licensePath,
		interval:    interval,
		stopCh:      make(chan struct{}),
	}

	w.refresh()
	return w
}

// Current returns the latest validated license, or nil if the license file
// is missing (CE mode).
func (w *LicenseWatcher) Current() *domain.LicenseInfo {
	return w.current.Load()
}

// Pointer returns the underlying atomic pointer for direct use by middleware.
func (w *LicenseWatcher) Pointer() *atomic.Pointer[domain.LicenseInfo] {
	return &w.current
}

// Start launches the background refresh goroutine.
func (w *LicenseWatcher) Start() {
	go w.loop()
}

// Stop signals the background goroutine to exit.
func (w *LicenseWatcher) Stop() {
	close(w.stopCh)
}

func (w *LicenseWatcher) loop() {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.refresh()
		}
	}
}

// refresh re-reads the license file and updates the atomic pointer.
// If the file no longer exists, the pointer is set to nil (CE mode).
func (w *LicenseWatcher) refresh() {
	if _, err := os.Stat(w.licensePath); os.IsNotExist(err) {
		prev := w.current.Load()
		if prev != nil {
			slog.Info("License file removed, switching to CE mode", "path", w.licensePath)
		}
		w.current.Store(nil)
		return
	}

	info := w.validator.Validate(w.licensePath)

	prev := w.current.Load()
	w.current.Store(info)

	w.logTransition(prev, info)
}

// logTransition logs meaningful status changes between old and new license states.
func (w *LicenseWatcher) logTransition(prev, next *domain.LicenseInfo) {
	prevStatus := statusOf(prev)
	nextStatus := statusOf(next)

	if prevStatus == nextStatus {
		return
	}

	slog.Info("License status changed",
		"previous", string(prevStatus),
		"current", string(nextStatus),
		"path", w.licensePath,
	)
}

// statusOf returns the effective status string for logging.
// nil license is represented as "ce" (Community Edition).
func statusOf(info *domain.LicenseInfo) domain.LicenseStatus {
	if info == nil {
		return "ce"
	}
	return info.Status
}

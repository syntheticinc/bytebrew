package portfile

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// AcquireLock checks for an existing port file in dataDir and either:
//   - returns an error if another live process owns the file (port already in use), OR
//   - removes a stale port file left behind by a crashed process.
//
// The dataDir is expected to exist. AcquireLock does not create it.
//
// On Windows the recorded PID may belong to a recycled process slot;
// IsProcessAlive uses OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION) to
// confirm the process is genuinely alive.
func AcquireLock(dataDir string) error {
	reader := NewReader(dataDir)
	existing, _ := reader.Read()
	if existing == nil {
		return nil
	}

	// Skip check if the recorded PID is our own process (Docker restart scenario)
	if existing.PID != os.Getpid() && IsProcessAlive(existing.PID) {
		return fmt.Errorf("server already running (PID %d, http_port %d). Kill it first or use a different config",
			existing.PID, existing.HTTPPort)
	}

	// Stale port file from a crashed/killed server — clean up.
	stalePortFile := filepath.Join(dataDir, fileName)
	if err := os.Remove(stalePortFile); err != nil && !os.IsNotExist(err) {
		slog.WarnContext(context.Background(), "failed to remove stale port file", "error", err)
		return nil
	}
	slog.InfoContext(context.Background(), "Removed stale port file", "pid", existing.PID)
	return nil
}

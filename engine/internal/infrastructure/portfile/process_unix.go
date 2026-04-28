//go:build !windows

package portfile

import (
	"os"
	"syscall"
)

// IsProcessAlive reports whether a process with the given PID is alive.
// Used to detect a stale port file.
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Signal 0 checks for process existence without sending an actual signal.
	// nil — process exists and is accessible.
	// EPERM — process exists but we lack permission (still alive).
	err = proc.Signal(syscall.Signal(0))
	return err == nil || err == syscall.EPERM
}

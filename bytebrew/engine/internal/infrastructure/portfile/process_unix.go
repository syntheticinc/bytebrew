//go:build !windows

package portfile

import (
	"os"
	"syscall"
)

// IsProcessAlive проверяет жив ли процесс по PID.
// Используется для определения stale port file.
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Signal 0 проверяет существование процесса без отправки сигнала.
	// nil — процесс существует и доступен.
	// EPERM — процесс существует, но нет прав (всё равно жив).
	err = proc.Signal(syscall.Signal(0))
	return err == nil || err == syscall.EPERM
}

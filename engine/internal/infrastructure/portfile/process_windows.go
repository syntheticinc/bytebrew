//go:build windows

package portfile

import (
	"syscall"
	"unsafe"
)

// IsProcessAlive reports whether a process with the given PID is alive.
// Used to detect a stale port file.
//
// On Windows, OpenProcess alone is not enough — the handle can be valid
// for a process that has already exited (if held by an IDE, debugger, etc.).
// We additionally check GetExitCodeProcess: STILL_ACTIVE (259) means the
// process is actually still running.
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	const processQueryLimitedInformation = 0x1000
	const stillActive = 259

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	openProcess := kernel32.NewProc("OpenProcess")
	closeHandle := kernel32.NewProc("CloseHandle")
	getExitCodeProcess := kernel32.NewProc("GetExitCodeProcess")

	handle, _, _ := openProcess.Call(
		processQueryLimitedInformation,
		0,
		uintptr(pid),
	)
	if handle == 0 {
		return false
	}
	defer closeHandle.Call(handle)

	var exitCode uint32
	ret, _, _ := getExitCodeProcess.Call(handle, uintptr(unsafe.Pointer(&exitCode)))
	if ret == 0 {
		// GetExitCodeProcess failed — assume alive to be safe
		return true
	}

	return exitCode == stillActive
}

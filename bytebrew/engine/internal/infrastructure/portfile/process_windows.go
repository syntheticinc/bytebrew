//go:build windows

package portfile

import (
	"syscall"
	"unsafe"
)

// IsProcessAlive проверяет жив ли процесс по PID.
// Используется для определения stale port file.
//
// На Windows просто OpenProcess недостаточно — handle может быть валидным
// для уже завершённого процесса (если его держит IDE, debugger и т.д.).
// Поэтому дополнительно проверяем GetExitCodeProcess: STILL_ACTIVE (259)
// означает что процесс действительно работает.
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

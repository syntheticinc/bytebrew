package shell

import (
	"os/exec"
	"runtime"
)

// detectShell returns the path to a suitable shell binary.
// On Unix, it tries bash then sh. On Windows, it looks for bash (Git Bash) in PATH.
func detectShell() string {
	if runtime.GOOS == "windows" {
		// Git Bash or WSL bash
		if path, err := exec.LookPath("bash"); err == nil {
			return path
		}
		return "bash"
	}

	// Unix: try common bash paths, fall back to sh
	candidates := []string{"/usr/bin/bash", "/bin/bash", "/bin/sh"}
	for _, c := range candidates {
		if _, err := exec.LookPath(c); err == nil {
			return c
		}
	}
	return "/bin/sh"
}

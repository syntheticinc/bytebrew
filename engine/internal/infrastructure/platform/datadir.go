// Package platform centralises OS-specific path resolution. This is the SOLE
// place in the engine that may read APPDATA / USERPROFILE / XDG_DATA_HOME —
// these are platform conventions, not application configuration, so they
// stay outside `pkg/config`. Callers receive resolved string paths; no env
// access leaks into business code.
//
// See `.claude/rules/code-review.md` for the enforced env-vars policy.
package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// UserDataDir returns the platform-appropriate user data directory for ByteBrew:
//   - Windows: %APPDATA%/bytebrew (or %USERPROFILE%/AppData/Roaming/bytebrew when APPDATA is empty)
//   - macOS:   ~/Library/Application Support/bytebrew
//   - Linux:   ${XDG_DATA_HOME:-~/.local/share}/bytebrew
func UserDataDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
		return filepath.Join(appData, "bytebrew"), nil
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get user home directory: %w", err)
		}
		return filepath.Join(home, "Library", "Application Support", "bytebrew"), nil
	default:
		xdgData := os.Getenv("XDG_DATA_HOME")
		if xdgData != "" {
			return filepath.Join(xdgData, "bytebrew"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get user home directory: %w", err)
		}
		return filepath.Join(home, ".local", "share", "bytebrew"), nil
	}
}

// LSPBinDir returns the platform-appropriate directory for LSP server binaries
// auto-installed by the engine. It nests under UserDataDir so a single env
// override reroutes both data and binaries:
//   - Windows: %APPDATA%/bytebrew/bin
//   - macOS:   ~/Library/Application Support/bytebrew/bin
//   - Linux:   ${XDG_DATA_HOME:-~/.local/share}/bytebrew/bin
func LSPBinDir() (string, error) {
	dataDir, err := UserDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "bin"), nil
}

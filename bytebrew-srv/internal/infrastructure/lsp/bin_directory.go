package lsp

import (
	"os"
	"path/filepath"
	"runtime"
)

// ManagedBinDir returns the path to the managed binary directory for auto-installed LSP servers.
//   - Windows: %APPDATA%/bytebrew/bin/
//   - macOS:   ~/Library/Application Support/bytebrew/bin/
//   - Linux:   ${XDG_DATA_HOME:-~/.local/share}/bytebrew/bin/
func ManagedBinDir() string {
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			home, _ := os.UserHomeDir()
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "bytebrew", "bin")

	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "bytebrew", "bin")

	default: // linux and others
		dataHome := os.Getenv("XDG_DATA_HOME")
		if dataHome == "" {
			home, _ := os.UserHomeDir()
			dataHome = filepath.Join(home, ".local", "share")
		}
		return filepath.Join(dataHome, "bytebrew", "bin")
	}
}

// EnsureBinDir creates the managed binary directory if it doesn't exist.
func EnsureBinDir() (string, error) {
	dir := ManagedBinDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

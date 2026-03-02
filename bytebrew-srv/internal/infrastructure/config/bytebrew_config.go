package config

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
)

type bytebrewConfig struct {
	BridgeURL string `json:"bridge_url"`
}

// ReadBridgeURL reads bridge_url from ~/.bytebrew/config.json.
// Returns empty string if file doesn't exist or can't be parsed.
func ReadBridgeURL() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	configPath := filepath.Join(home, ".bytebrew", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}

	var cfg bytebrewConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		slog.Warn("failed to parse bytebrew config", "path", configPath, "error", err)
		return ""
	}

	return cfg.BridgeURL
}

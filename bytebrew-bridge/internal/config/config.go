package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
)

// Config holds bridge server configuration loaded from environment variables.
type Config struct {
	Port      int
	TLSCert   string
	TLSKey    string
	AuthToken string
}

// Load reads configuration from environment variables with sensible defaults.
// BRIDGE_PORT (default 8443), BRIDGE_TLS_CERT, BRIDGE_TLS_KEY, BRIDGE_AUTH_TOKEN.
func Load() *Config {
	cfg := &Config{
		Port:      8443,
		TLSCert:   os.Getenv("BRIDGE_TLS_CERT"),
		TLSKey:    os.Getenv("BRIDGE_TLS_KEY"),
		AuthToken: os.Getenv("BRIDGE_AUTH_TOKEN"),
	}

	if portStr := os.Getenv("BRIDGE_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil && p > 0 {
			cfg.Port = p
		}
	}

	return cfg
}

// Validate checks that the configuration values are valid.
func (c *Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port %d: must be between 1 and 65535", c.Port)
	}

	hasCert := c.TLSCert != ""
	hasKey := c.TLSKey != ""
	if hasCert != hasKey {
		return fmt.Errorf("both BRIDGE_TLS_CERT and BRIDGE_TLS_KEY must be set, or neither")
	}

	if c.AuthToken == "" {
		slog.Warn("no bridge auth token configured, all connections will be accepted")
	}

	return nil
}

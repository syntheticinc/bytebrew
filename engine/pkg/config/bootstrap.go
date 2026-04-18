package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// BootstrapConfig is the minimal config loaded from YAML before connecting to the database.
// All runtime configuration (agents, models, etc.) lives in the database.
type BootstrapConfig struct {
	Engine   EngineBootstrap   `mapstructure:"engine"`
	Database BootstrapDatabase `mapstructure:"database"`
	Security BootstrapSecurity `mapstructure:"security"`
	Logging  BootstrapLogging  `mapstructure:"logging"`
}

// EngineBootstrap holds the minimal engine settings needed at startup.
type EngineBootstrap struct {
	Host         string   `mapstructure:"host"`
	Port         int      `mapstructure:"port"`           // External/data plane port (default 8443)
	InternalPort int      `mapstructure:"internal_port"`  // Control plane port (default 0 = single-port mode)
	CORSOrigins  []string `mapstructure:"cors_origins"`   // Allowed CORS origins for external port (empty = allow all)
	DataDir      string   `mapstructure:"data_dir"`
}

// BootstrapDatabase holds the database connection settings.
type BootstrapDatabase struct {
	URL string `mapstructure:"url"` // PostgreSQL connection string
}

// BootstrapLogging holds logging settings loaded at startup.
type BootstrapLogging struct {
	Level string `mapstructure:"level"`
}

// BootstrapSecurity holds security settings loaded at startup.
// Admin credentials are NO LONGER here — admin/system users live in the
// `users` table and are created via the `ce admin` CLI subcommand.
type BootstrapSecurity struct {
	// JWTSecret is the HMAC key used to sign/verify admin JWT tokens.
	// Typically supplied via the JWT_SECRET environment variable.
	JWTSecret string `mapstructure:"jwt_secret"`
}

// LoadBootstrap loads the bootstrap config from a YAML file.
// If the config file is not found, falls back to environment variables:
//   - DATABASE_URL — PostgreSQL connection string (required)
//   - JWT_SECRET — HMAC signing key for admin JWTs (required if no users authenticate)
//   - ENGINE_HOST — listen host (optional, default: "0.0.0.0")
//   - ENGINE_PORT — listen port (optional, default: 8443)
//
// Environment variable placeholders (${VAR}) in YAML string fields are also expanded.
func LoadBootstrap(path string) (*BootstrapConfig, error) {
	if path == "" {
		return nil, fmt.Errorf("config path is required")
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		// Config file not found — try environment variables
		return loadBootstrapFromEnv()
	}

	var cfg BootstrapConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal bootstrap config: %w", err)
	}

	expandBootstrapEnvVars(&cfg)
	applyBootstrapEnvOverrides(&cfg)

	if err := validateBootstrap(&cfg); err != nil {
		return nil, fmt.Errorf("validate bootstrap config: %w", err)
	}

	// Resolve DataDir relative to config file directory if not absolute
	if cfg.Engine.DataDir != "" && !filepath.IsAbs(cfg.Engine.DataDir) {
		configDir := filepath.Dir(path)
		cfg.Engine.DataDir = filepath.Join(configDir, cfg.Engine.DataDir)
	}

	return &cfg, nil
}

// loadBootstrapFromEnv constructs BootstrapConfig from environment variables.
// This enables zero-config Docker deployments where everything is passed via env.
func loadBootstrapFromEnv() (*BootstrapConfig, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("no config file found and DATABASE_URL environment variable is not set")
	}

	cfg := DefaultBootstrapConfig()
	cfg.Database.URL = dbURL

	if host := os.Getenv("ENGINE_HOST"); host != "" {
		cfg.Engine.Host = host
	} else {
		cfg.Engine.Host = "0.0.0.0"
	}

	if portStr := os.Getenv("ENGINE_PORT"); portStr != "" {
		port := 8443
		fmt.Sscanf(portStr, "%d", &port)
		cfg.Engine.Port = port
	} else {
		cfg.Engine.Port = 8443
	}

	if portStr := os.Getenv("BYTEBREW_INTERNAL_PORT"); portStr != "" {
		var port int
		fmt.Sscanf(portStr, "%d", &port)
		cfg.Engine.InternalPort = port
	}

	if origins := os.Getenv("BYTEBREW_CORS_ORIGINS"); origins != "" {
		cfg.Engine.CORSOrigins = splitAndTrim(origins, ",")
	}

	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		cfg.Security.JWTSecret = secret
	}

	if err := validateBootstrap(cfg); err != nil {
		return nil, fmt.Errorf("validate env-based config: %w", err)
	}

	return cfg, nil
}

// applyBootstrapEnvOverrides applies BYTEBREW_* environment variable overrides
// on top of YAML-loaded config. This allows env vars to override YAML settings
// (e.g., BYTEBREW_INTERNAL_PORT=8444 enables two-port mode regardless of YAML).
func applyBootstrapEnvOverrides(cfg *BootstrapConfig) {
	if portStr := os.Getenv("BYTEBREW_INTERNAL_PORT"); portStr != "" {
		var port int
		fmt.Sscanf(portStr, "%d", &port)
		cfg.Engine.InternalPort = port
	}
	if origins := os.Getenv("BYTEBREW_CORS_ORIGINS"); origins != "" {
		cfg.Engine.CORSOrigins = splitAndTrim(origins, ",")
	}
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		cfg.Security.JWTSecret = secret
	}
}

// expandBootstrapEnvVars expands ${VAR} placeholders in all string fields of BootstrapConfig.
func expandBootstrapEnvVars(cfg *BootstrapConfig) {
	cfg.Engine.Host = expandEnvVars(cfg.Engine.Host)
	cfg.Engine.DataDir = expandEnvVars(cfg.Engine.DataDir)
	cfg.Database.URL = expandEnvVars(cfg.Database.URL)
	cfg.Security.JWTSecret = expandEnvVars(cfg.Security.JWTSecret)
	cfg.Logging.Level = expandEnvVars(cfg.Logging.Level)
}

// validateBootstrap checks that required bootstrap fields are present.
func validateBootstrap(cfg *BootstrapConfig) error {
	if cfg.Database.URL == "" {
		return fmt.Errorf("database.url is required")
	}
	if cfg.Engine.Port < 0 || cfg.Engine.Port > 65535 {
		return fmt.Errorf("invalid engine port: %d", cfg.Engine.Port)
	}
	if cfg.Engine.InternalPort < 0 || cfg.Engine.InternalPort > 65535 {
		return fmt.Errorf("invalid internal port: %d", cfg.Engine.InternalPort)
	}
	if cfg.Engine.InternalPort > 0 && cfg.Engine.InternalPort == cfg.Engine.Port {
		return fmt.Errorf("internal_port (%d) must differ from port (%d)", cfg.Engine.InternalPort, cfg.Engine.Port)
	}
	return nil
}

// DefaultBootstrapConfig returns sensible defaults for BootstrapConfig.
func DefaultBootstrapConfig() *BootstrapConfig {
	return &BootstrapConfig{
		Engine: EngineBootstrap{
			Host: "127.0.0.1",
			Port: 8080,
		},
		Logging: BootstrapLogging{
			Level: "info",
		},
	}
}

// DSN returns the database connection string.
// It returns the URL directly since BootstrapDatabase uses a connection string.
func (d *BootstrapDatabase) DSN() string {
	return d.URL
}

// DataDirOrDefault returns the configured data directory or a platform-appropriate default.
func (e *EngineBootstrap) DataDirOrDefault() string {
	if e.DataDir != "" {
		return e.DataDir
	}

	// Default to user config dir + bytebrew
	dir, err := os.UserConfigDir()
	if err != nil {
		return "./data"
	}
	return filepath.Join(dir, "bytebrew")
}

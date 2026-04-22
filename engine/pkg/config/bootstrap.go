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

// AuthMode values.
const (
	// AuthModeLocal: CE single-node — engine signs its own Ed25519 keypair on
	// first boot, issues short-lived admin sessions via POST /auth/local-session,
	// `sub` is the synthetic `local-admin`, `tenant_id` is empty.
	AuthModeLocal = "local"
	// AuthModeExternal: Cloud / hosted — tokens are signed by an external issuer
	// (landing service). Engine loads only the public key; there is no
	// /auth/local-session route.
	AuthModeExternal = "external"
)

// BootstrapSecurity holds auth-related settings loaded at startup.
//
// There is no shared HMAC secret any more (Wave 1+7). All JWTs are EdDSA.
// In local mode the engine generates its keypair automatically and
// persists it under JWTKeysDir. In external mode JWTPublicKeyPath points at
// the issuer's public key.
type BootstrapSecurity struct {
	// AuthMode selects local vs external JWT issuance.
	// Accepts "local" or "external"; defaults to "local" when empty.
	AuthMode string `mapstructure:"auth_mode"`

	// JWTKeysDir is the directory where the local Ed25519 keypair is stored.
	// Used only when AuthMode == "local". Defaults to <data_dir>/keys.
	JWTKeysDir string `mapstructure:"jwt_keys_dir"`

	// JWTPublicKeyPath is the path to the Ed25519 public key of the external
	// issuer. Required when AuthMode == "external".
	JWTPublicKeyPath string `mapstructure:"jwt_public_key_path"`
}

// LoadBootstrap loads the bootstrap config from a YAML file.
// If the config file is not found, falls back to environment variables:
//   - DATABASE_URL — PostgreSQL connection string (required)
//   - BYTEBREW_AUTH_MODE — "local" (default) or "external"
//   - BYTEBREW_JWT_KEYS_DIR — directory for local-mode Ed25519 keypair
//   - BYTEBREW_JWT_PUBLIC_KEY_PATH — path to external issuer public key
//   - ENGINE_HOST / ENGINE_PORT — listen host/port (optional)
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
	applySecurityDefaults(&cfg)

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

	applyBootstrapEnvOverrides(cfg)
	applySecurityDefaults(cfg)

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
	if mode := os.Getenv("BYTEBREW_AUTH_MODE"); mode != "" {
		cfg.Security.AuthMode = mode
	}
	if dir := os.Getenv("BYTEBREW_JWT_KEYS_DIR"); dir != "" {
		cfg.Security.JWTKeysDir = dir
	}
	if path := os.Getenv("BYTEBREW_JWT_PUBLIC_KEY_PATH"); path != "" {
		cfg.Security.JWTPublicKeyPath = path
	}
}

// applySecurityDefaults fills missing auth settings with sensible defaults
// after env overrides. Called after loadBootstrapFromEnv / LoadBootstrap so
// YAML-provided keys win over defaults.
func applySecurityDefaults(cfg *BootstrapConfig) {
	if cfg.Security.AuthMode == "" {
		cfg.Security.AuthMode = AuthModeLocal
	}
	if cfg.Security.AuthMode == AuthModeLocal && cfg.Security.JWTKeysDir == "" {
		cfg.Security.JWTKeysDir = filepath.Join(cfg.Engine.DataDirOrDefault(), "keys")
	}
}

// expandBootstrapEnvVars expands ${VAR} placeholders in all string fields of BootstrapConfig.
func expandBootstrapEnvVars(cfg *BootstrapConfig) {
	cfg.Engine.Host = expandEnvVars(cfg.Engine.Host)
	cfg.Engine.DataDir = expandEnvVars(cfg.Engine.DataDir)
	cfg.Database.URL = expandEnvVars(cfg.Database.URL)
	cfg.Security.AuthMode = expandEnvVars(cfg.Security.AuthMode)
	cfg.Security.JWTKeysDir = expandEnvVars(cfg.Security.JWTKeysDir)
	cfg.Security.JWTPublicKeyPath = expandEnvVars(cfg.Security.JWTPublicKeyPath)
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
	switch cfg.Security.AuthMode {
	case AuthModeLocal:
		if cfg.Security.JWTKeysDir == "" {
			return fmt.Errorf("security.jwt_keys_dir is required when auth_mode=local")
		}
	case AuthModeExternal:
		if cfg.Security.JWTPublicKeyPath == "" {
			return fmt.Errorf("security.jwt_public_key_path is required when auth_mode=external")
		}
	default:
		return fmt.Errorf("invalid auth_mode %q (expected %q or %q)",
			cfg.Security.AuthMode, AuthModeLocal, AuthModeExternal)
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
		Security: BootstrapSecurity{
			AuthMode: AuthModeLocal,
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

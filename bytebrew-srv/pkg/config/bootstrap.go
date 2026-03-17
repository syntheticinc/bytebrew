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
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
	DataDir string `mapstructure:"data_dir"`
}

// BootstrapDatabase holds the database connection settings.
type BootstrapDatabase struct {
	URL string `mapstructure:"url"` // PostgreSQL connection string
}

// BootstrapLogging holds logging settings loaded at startup.
type BootstrapLogging struct {
	Level string `mapstructure:"level"`
}

// BootstrapSecurity holds admin credentials for initial setup.
type BootstrapSecurity struct {
	AdminUser     string `mapstructure:"admin_user"`
	AdminPassword string `mapstructure:"admin_password"`
}

// LoadBootstrap loads only the bootstrap config from a YAML file.
// Environment variable placeholders (${VAR}) in string fields are expanded.
func LoadBootstrap(path string) (*BootstrapConfig, error) {
	if path == "" {
		return nil, fmt.Errorf("config path is required")
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg BootstrapConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal bootstrap config: %w", err)
	}

	expandBootstrapEnvVars(&cfg)

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

// expandBootstrapEnvVars expands ${VAR} placeholders in all string fields of BootstrapConfig.
func expandBootstrapEnvVars(cfg *BootstrapConfig) {
	cfg.Engine.Host = expandEnvVars(cfg.Engine.Host)
	cfg.Engine.DataDir = expandEnvVars(cfg.Engine.DataDir)
	cfg.Database.URL = expandEnvVars(cfg.Database.URL)
	cfg.Security.AdminUser = expandEnvVars(cfg.Security.AdminUser)
	cfg.Security.AdminPassword = expandEnvVars(cfg.Security.AdminPassword)
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

package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the relay service.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	CloudAPI CloudAPIConfig `yaml:"cloud_api"`
	Cache    CacheConfig    `yaml:"cache"`
	Sessions SessionsConfig `yaml:"sessions"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port int `yaml:"port"`
}

// CloudAPIConfig holds Cloud API client configuration.
type CloudAPIConfig struct {
	BaseURL   string `yaml:"base_url"`
	AuthToken string `yaml:"auth_token"`
}

// CacheConfig holds license cache configuration.
type CacheConfig struct {
	TTL         time.Duration `yaml:"ttl"`
	GracePeriod time.Duration `yaml:"grace_period"`
	PersistPath string        `yaml:"persist_path"`
}

// SessionsConfig holds session management configuration.
type SessionsConfig struct {
	HeartbeatTimeout time.Duration `yaml:"heartbeat_timeout"`
	CleanupInterval  time.Duration `yaml:"cleanup_interval"`
}

// LoggingConfig holds logging configuration.
type LoggingConfig struct {
	Level string `yaml:"level"`
}

// Load reads configuration from the given YAML file path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.setDefaults()

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) setDefaults() {
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Cache.TTL == 0 {
		c.Cache.TTL = 5 * time.Minute
	}
	if c.Cache.GracePeriod == 0 {
		c.Cache.GracePeriod = 30 * time.Minute
	}
	if c.Sessions.HeartbeatTimeout == 0 {
		c.Sessions.HeartbeatTimeout = 5 * time.Minute
	}
	if c.Sessions.CleanupInterval == 0 {
		c.Sessions.CleanupInterval = 1 * time.Minute
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
}

func (c *Config) validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	if c.CloudAPI.BaseURL == "" {
		return fmt.Errorf("cloud_api.base_url is required")
	}
	return nil
}

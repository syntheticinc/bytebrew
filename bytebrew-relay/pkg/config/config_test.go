package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// --- Helpers ---

// writeTempConfig creates a temporary YAML config file and returns its path.
func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

// --- Load tests ---

func TestLoad_FullConfig(t *testing.T) {
	yaml := `
server:
  port: 9090
cloud_api:
  base_url: https://api.example.com
  auth_token: secret-token
cache:
  ttl: 10m
  grace_period: 1h
  persist_path: /tmp/cache.json
sessions:
  heartbeat_timeout: 3m
  cleanup_interval: 30s
logging:
  level: debug
`
	cfg, err := Load(writeTempConfig(t, yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Fatalf("expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.CloudAPI.BaseURL != "https://api.example.com" {
		t.Fatalf("expected base_url https://api.example.com, got %s", cfg.CloudAPI.BaseURL)
	}
	if cfg.CloudAPI.AuthToken != "secret-token" {
		t.Fatalf("expected auth_token secret-token, got %s", cfg.CloudAPI.AuthToken)
	}
	if cfg.Cache.TTL != 10*time.Minute {
		t.Fatalf("expected cache TTL 10m, got %v", cfg.Cache.TTL)
	}
	if cfg.Cache.GracePeriod != time.Hour {
		t.Fatalf("expected grace period 1h, got %v", cfg.Cache.GracePeriod)
	}
	if cfg.Cache.PersistPath != "/tmp/cache.json" {
		t.Fatalf("expected persist_path /tmp/cache.json, got %s", cfg.Cache.PersistPath)
	}
	if cfg.Sessions.HeartbeatTimeout != 3*time.Minute {
		t.Fatalf("expected heartbeat timeout 3m, got %v", cfg.Sessions.HeartbeatTimeout)
	}
	if cfg.Sessions.CleanupInterval != 30*time.Second {
		t.Fatalf("expected cleanup interval 30s, got %v", cfg.Sessions.CleanupInterval)
	}
	if cfg.Logging.Level != "debug" {
		t.Fatalf("expected logging level debug, got %s", cfg.Logging.Level)
	}
}

func TestLoad_Defaults(t *testing.T) {
	yaml := `
cloud_api:
  base_url: https://api.example.com
`
	cfg, err := Load(writeTempConfig(t, yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Fatalf("expected default port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Cache.TTL != 5*time.Minute {
		t.Fatalf("expected default cache TTL 5m, got %v", cfg.Cache.TTL)
	}
	if cfg.Cache.GracePeriod != 30*time.Minute {
		t.Fatalf("expected default grace period 30m, got %v", cfg.Cache.GracePeriod)
	}
	if cfg.Sessions.HeartbeatTimeout != 5*time.Minute {
		t.Fatalf("expected default heartbeat timeout 5m, got %v", cfg.Sessions.HeartbeatTimeout)
	}
	if cfg.Sessions.CleanupInterval != 1*time.Minute {
		t.Fatalf("expected default cleanup interval 1m, got %v", cfg.Sessions.CleanupInterval)
	}
	if cfg.Logging.Level != "info" {
		t.Fatalf("expected default logging level info, got %s", cfg.Logging.Level)
	}
}

func TestLoad_Validation_MissingBaseURL(t *testing.T) {
	yaml := `
server:
  port: 8080
`
	_, err := Load(writeTempConfig(t, yaml))
	if err == nil {
		t.Fatal("expected validation error for missing base_url")
	}
}

func TestLoad_Validation_InvalidPort(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			"port too high",
			`
server:
  port: 99999
cloud_api:
  base_url: https://api.example.com
`,
		},
		{
			"negative port",
			`
server:
  port: -1
cloud_api:
  base_url: https://api.example.com
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(writeTempConfig(t, tt.yaml))
			if err == nil {
				t.Fatal("expected validation error for invalid port")
			}
		})
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	yaml := `
server:
  port: [invalid yaml
  not_closed: {
`
	_, err := Load(writeTempConfig(t, yaml))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

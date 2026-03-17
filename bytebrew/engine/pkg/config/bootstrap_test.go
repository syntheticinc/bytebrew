package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadBootstrap(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		env     map[string]string
		check   func(t *testing.T, cfg *BootstrapConfig)
		wantErr string
	}{
		{
			name: "valid minimal config",
			yaml: `
database:
  url: "postgresql://localhost:5432/bytebrew"
`,
			check: func(t *testing.T, cfg *BootstrapConfig) {
				assert.Equal(t, "postgresql://localhost:5432/bytebrew", cfg.Database.URL)
				assert.Equal(t, 0, cfg.Engine.Port)
				assert.Equal(t, "", cfg.Engine.Host)
			},
		},
		{
			name: "full config",
			yaml: `
engine:
  host: "0.0.0.0"
  port: 9090
  data_dir: "./data"
database:
  url: "postgresql://admin:pass@db.host:5432/bytebrew?sslmode=require"
security:
  admin_user: "admin"
  admin_password: "s3cret"
logging:
  level: "debug"
`,
			check: func(t *testing.T, cfg *BootstrapConfig) {
				assert.Equal(t, "0.0.0.0", cfg.Engine.Host)
				assert.Equal(t, 9090, cfg.Engine.Port)
				assert.Equal(t, "admin", cfg.Security.AdminUser)
				assert.Equal(t, "s3cret", cfg.Security.AdminPassword)
				assert.Equal(t, "debug", cfg.Logging.Level)
				assert.Contains(t, cfg.Database.URL, "sslmode=require")
			},
		},
		{
			name: "env var expansion",
			yaml: `
database:
  url: "postgresql://${TEST_DB_USER}:${TEST_DB_PASS}@localhost:5432/bytebrew"
security:
  admin_password: "${TEST_ADMIN_PASS}"
`,
			env: map[string]string{
				"TEST_DB_USER":    "pguser",
				"TEST_DB_PASS":    "pgpass",
				"TEST_ADMIN_PASS": "admin123",
			},
			check: func(t *testing.T, cfg *BootstrapConfig) {
				assert.Equal(t, "postgresql://pguser:pgpass@localhost:5432/bytebrew", cfg.Database.URL)
				assert.Equal(t, "admin123", cfg.Security.AdminPassword)
			},
		},
		{
			name: "missing database url",
			yaml: `
engine:
  port: 8080
`,
			wantErr: "database.url is required",
		},
		{
			name: "invalid port",
			yaml: `
engine:
  port: 70000
database:
  url: "postgresql://localhost/db"
`,
			wantErr: "invalid engine port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			require.NoError(t, os.WriteFile(configPath, []byte(tt.yaml), 0644))

			cfg, err := LoadBootstrap(configPath)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)
			tt.check(t, cfg)
		})
	}
}

func TestLoadBootstrap_EmptyPath(t *testing.T) {
	_, err := LoadBootstrap("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config path is required")
}

func TestLoadBootstrap_NonexistentFile(t *testing.T) {
	_, err := LoadBootstrap("/nonexistent/config.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read config")
}

func TestLoadBootstrap_RelativeDataDir(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	yaml := `
engine:
  data_dir: "./mydata"
database:
  url: "postgresql://localhost/db"
`
	require.NoError(t, os.WriteFile(configPath, []byte(yaml), 0644))

	cfg, err := LoadBootstrap(configPath)
	require.NoError(t, err)

	// data_dir should be resolved relative to config file directory
	expected := filepath.Join(tmpDir, "mydata")
	assert.Equal(t, expected, cfg.Engine.DataDir)
}

func TestBootstrapDatabase_DSN(t *testing.T) {
	db := &BootstrapDatabase{URL: "postgresql://localhost:5432/test"}
	assert.Equal(t, "postgresql://localhost:5432/test", db.DSN())
}

func TestEngineBootstrap_DataDirOrDefault(t *testing.T) {
	t.Run("configured", func(t *testing.T) {
		e := &EngineBootstrap{DataDir: "/custom/data"}
		assert.Equal(t, "/custom/data", e.DataDirOrDefault())
	})

	t.Run("default", func(t *testing.T) {
		e := &EngineBootstrap{}
		dir := e.DataDirOrDefault()
		assert.NotEmpty(t, dir)
	})
}

func TestDefaultBootstrapConfig(t *testing.T) {
	cfg := DefaultBootstrapConfig()
	assert.Equal(t, "127.0.0.1", cfg.Engine.Host)
	assert.Equal(t, 8080, cfg.Engine.Port)
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Empty(t, cfg.Database.URL)
}

package config

import (
	"fmt"
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
  jwt_secret: "s3cret-jwt-key"
logging:
  level: "debug"
`,
			check: func(t *testing.T, cfg *BootstrapConfig) {
				assert.Equal(t, "0.0.0.0", cfg.Engine.Host)
				assert.Equal(t, 9090, cfg.Engine.Port)
				assert.Equal(t, "s3cret-jwt-key", cfg.Security.JWTSecret)
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
  jwt_secret: "${TEST_JWT_SECRET}"
`,
			env: map[string]string{
				"TEST_DB_USER":    "pguser",
				"TEST_DB_PASS":    "pgpass",
				"TEST_JWT_SECRET": "jwt-xyz",
			},
			check: func(t *testing.T, cfg *BootstrapConfig) {
				assert.Equal(t, "postgresql://pguser:pgpass@localhost:5432/bytebrew", cfg.Database.URL)
				assert.Equal(t, "jwt-xyz", cfg.Security.JWTSecret)
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
	assert.Contains(t, err.Error(), "no config file found")
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

func TestBootstrapValidation_InternalPort(t *testing.T) {
	baseYAML := func(port, internalPort int) string {
		return fmt.Sprintf(`
engine:
  port: %d
  internal_port: %d
database:
  url: "postgresql://localhost/db"
`, port, internalPort)
	}

	tests := []struct {
		name         string
		port         int
		internalPort int
		wantErr      string
	}{
		{
			name:         "internal port same as port",
			port:         8443,
			internalPort: 8443,
			wantErr:      "internal_port (8443) must differ from port (8443)",
		},
		{
			name:         "internal port valid",
			port:         8443,
			internalPort: 8444,
		},
		{
			name:         "internal port zero is single-port mode",
			port:         8443,
			internalPort: 0,
		},
		{
			name:         "internal port negative",
			port:         8443,
			internalPort: -1,
			wantErr:      "invalid internal port",
		},
		{
			name:         "internal port too large",
			port:         8443,
			internalPort: 70000,
			wantErr:      "invalid internal port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			require.NoError(t, os.WriteFile(configPath, []byte(baseYAML(tt.port, tt.internalPort)), 0644))

			cfg, err := LoadBootstrap(configPath)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)
			assert.Equal(t, tt.internalPort, cfg.Engine.InternalPort)
		})
	}
}

func TestBootstrapEnvOverrides_InternalPort(t *testing.T) {
	t.Setenv("BYTEBREW_INTERNAL_PORT", "8444")

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	yaml := `
engine:
  port: 8443
database:
  url: "postgresql://localhost/db"
`
	require.NoError(t, os.WriteFile(configPath, []byte(yaml), 0644))

	cfg, err := LoadBootstrap(configPath)
	require.NoError(t, err)
	assert.Equal(t, 8444, cfg.Engine.InternalPort)
}

func TestBootstrapEnvOverrides_CORSOrigins(t *testing.T) {
	t.Setenv("BYTEBREW_CORS_ORIGINS", "https://example.com, https://app.example.com")

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	yaml := `
engine:
  port: 8443
database:
  url: "postgresql://localhost/db"
`
	require.NoError(t, os.WriteFile(configPath, []byte(yaml), 0644))

	cfg, err := LoadBootstrap(configPath)
	require.NoError(t, err)
	require.Len(t, cfg.Engine.CORSOrigins, 2)
	assert.Equal(t, "https://example.com", cfg.Engine.CORSOrigins[0])
	assert.Equal(t, "https://app.example.com", cfg.Engine.CORSOrigins[1])
}

func TestBootstrapEnvOverrides_InternalPortConflict(t *testing.T) {
	// Env override sets internal_port = port, should fail validation.
	t.Setenv("BYTEBREW_INTERNAL_PORT", "8443")

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	yaml := `
engine:
  port: 8443
database:
  url: "postgresql://localhost/db"
`
	require.NoError(t, os.WriteFile(configPath, []byte(yaml), 0644))

	_, err := LoadBootstrap(configPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must differ from port")
}

func TestLoadBootstrapFromEnv_InternalPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://localhost/db")
	t.Setenv("BYTEBREW_INTERNAL_PORT", "9090")

	cfg, err := LoadBootstrap("/nonexistent/config.yaml")
	require.NoError(t, err)
	assert.Equal(t, 9090, cfg.Engine.InternalPort)
}

func TestLoadBootstrapFromEnv_CORSOrigins(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://localhost/db")
	t.Setenv("BYTEBREW_CORS_ORIGINS", "https://a.com,https://b.com, https://c.com ")

	cfg, err := LoadBootstrap("/nonexistent/config.yaml")
	require.NoError(t, err)
	require.Len(t, cfg.Engine.CORSOrigins, 3)
	assert.Equal(t, "https://a.com", cfg.Engine.CORSOrigins[0])
	assert.Equal(t, "https://b.com", cfg.Engine.CORSOrigins[1])
	assert.Equal(t, "https://c.com", cfg.Engine.CORSOrigins[2])
}

func TestBootstrapValidation_TwoPortYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	yaml := `
engine:
  host: "0.0.0.0"
  port: 8443
  internal_port: 8444
  cors_origins:
    - "https://example.com"
    - "https://app.example.com"
database:
  url: "postgresql://localhost/db"
`
	require.NoError(t, os.WriteFile(configPath, []byte(yaml), 0644))

	cfg, err := LoadBootstrap(configPath)
	require.NoError(t, err)
	assert.Equal(t, 8443, cfg.Engine.Port)
	assert.Equal(t, 8444, cfg.Engine.InternalPort)
	require.Len(t, cfg.Engine.CORSOrigins, 2)
	assert.Equal(t, "https://example.com", cfg.Engine.CORSOrigins[0])
	assert.Equal(t, "https://app.example.com", cfg.Engine.CORSOrigins[1])
}

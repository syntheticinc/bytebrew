package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
server:
  host: "0.0.0.0"
  port: 50051
  grpc:
    max_recv_msg_size: 104857600
    max_send_msg_size: 104857600
    keepalive:
      time: 60s
      timeout: 20s
    connection_timeout: 120s

database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  database: vector_srv
  ssl_mode: disable
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m

llm:
  default_provider: ollama
  ollama:
    base_url: http://localhost:11434
    model: qwen2.5-coder:7b
    timeout: 300s

logging:
  level: info
  format: json
  output: stdout

observability:
  enable_metrics: true
  enable_tracing: true
  otlp:
    endpoint: localhost:4317
    insecure: true
  metrics:
    port: 9090
    path: /metrics

security:
  api_key: test-key
  enable_auth: false

agent:
  max_steps: 10
  max_context_size: 16000
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Create prompts.yaml
	promptsPath := filepath.Join(tmpDir, "prompts.yaml")
	promptsContent := `
prompts:
  system_prompt: |
    You are a helpful AI assistant.
  urgency_warning: |
    **CRITICAL: You have only %d steps remaining!**
`
	if err := os.WriteFile(promptsPath, []byte(promptsContent), 0644); err != nil {
		t.Fatalf("Failed to write prompts file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Test server config
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host = %v, want 0.0.0.0", cfg.Server.Host)
	}
	if cfg.Server.Port != 50051 {
		t.Errorf("Server.Port = %v, want 50051", cfg.Server.Port)
	}

	// Test database config
	if cfg.Database.Host != "localhost" {
		t.Errorf("Database.Host = %v, want localhost", cfg.Database.Host)
	}
	if cfg.Database.Port != 5432 {
		t.Errorf("Database.Port = %v, want 5432", cfg.Database.Port)
	}

	// Test LLM config
	if cfg.LLM.DefaultProvider != "ollama" {
		t.Errorf("LLM.DefaultProvider = %v, want ollama", cfg.LLM.DefaultProvider)
	}
	if cfg.LLM.Ollama.BaseURL != "http://localhost:11434" {
		t.Errorf("LLM.Ollama.BaseURL = %v, want http://localhost:11434", cfg.LLM.Ollama.BaseURL)
	}

	// Test logging config
	if cfg.Logging.Level != "info" {
		t.Errorf("Logging.Level = %v, want info", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "json" {
		t.Errorf("Logging.Format = %v, want json", cfg.Logging.Format)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				Server: ServerConfig{
					Port: 50051,
				},
				Database: DatabaseConfig{
					Host: "localhost",
				},
				LLM: LLMConfig{
					DefaultProvider: "ollama",
				},
				Agent: AgentConfig{
					Prompts: &PromptsConfig{
						SystemPrompt: "You are a helpful AI assistant.",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid port - zero for random",
			cfg: &Config{
				Server: ServerConfig{
					Port: 0,
				},
				Database: DatabaseConfig{
					Host: "localhost",
				},
				LLM: LLMConfig{
					DefaultProvider: "ollama",
				},
				Agent: AgentConfig{
					Prompts: &PromptsConfig{
						SystemPrompt: "You are a helpful AI assistant.",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid port - negative",
			cfg: &Config{
				Server: ServerConfig{
					Port: -1,
				},
				Database: DatabaseConfig{
					Host: "localhost",
				},
				LLM: LLMConfig{
					DefaultProvider: "ollama",
				},
				Agent: AgentConfig{
					Prompts: &PromptsConfig{
						SystemPrompt: "You are a helpful AI assistant.",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid port - too high",
			cfg: &Config{
				Server: ServerConfig{
					Port: 70000,
				},
				Database: DatabaseConfig{
					Host: "localhost",
				},
				LLM: LLMConfig{
					DefaultProvider: "ollama",
				},
				Agent: AgentConfig{
					Prompts: &PromptsConfig{
						SystemPrompt: "You are a helpful AI assistant.",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing database host",
			cfg: &Config{
				Server: ServerConfig{
					Port: 50051,
				},
				Database: DatabaseConfig{
					Host: "",
				},
				LLM: LLMConfig{
					DefaultProvider: "ollama",
				},
				Agent: AgentConfig{
					Prompts: &PromptsConfig{
						SystemPrompt: "You are a helpful AI assistant.",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing default provider",
			cfg: &Config{
				Server: ServerConfig{
					Port: 50051,
				},
				Database: DatabaseConfig{
					Host: "localhost",
				},
				LLM: LLMConfig{
					DefaultProvider: "",
				},
				Agent: AgentConfig{
					Prompts: &PromptsConfig{
						SystemPrompt: "You are a helpful AI assistant.",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDurationParsing(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
server:
  host: "0.0.0.0"
  port: 50051
  grpc:
    max_recv_msg_size: 104857600
    max_send_msg_size: 104857600
    keepalive:
      time: 60s
      timeout: 20s
    connection_timeout: 2m

database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  database: vector_srv
  ssl_mode: disable
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m

llm:
  default_provider: ollama
  ollama:
    base_url: http://localhost:11434
    model: qwen2.5-coder:7b
    timeout: 5m

logging:
  level: info
  format: json
  output: stdout

observability:
  enable_metrics: true
  enable_tracing: true

security:
  api_key: test-key
  enable_auth: false

agent:
  max_steps: 10
  max_context_size: 16000
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Create prompts.yaml
	promptsPath := filepath.Join(tmpDir, "prompts.yaml")
	promptsContent := `
prompts:
  system_prompt: |
    You are a helpful AI assistant.
  urgency_warning: |
    **CRITICAL: You have only %d steps remaining!**
`
	if err := os.WriteFile(promptsPath, []byte(promptsContent), 0644); err != nil {
		t.Fatalf("Failed to write prompts file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Test duration parsing
	if cfg.Server.GRPC.Keepalive.Time != 60*time.Second {
		t.Errorf("Keepalive.Time = %v, want 60s", cfg.Server.GRPC.Keepalive.Time)
	}
	if cfg.Server.GRPC.Keepalive.Timeout != 20*time.Second {
		t.Errorf("Keepalive.Timeout = %v, want 20s", cfg.Server.GRPC.Keepalive.Timeout)
	}
	if cfg.Server.GRPC.ConnectionTimeout != 2*time.Minute {
		t.Errorf("ConnectionTimeout = %v, want 2m", cfg.Server.GRPC.ConnectionTimeout)
	}
	if cfg.Database.ConnMaxLifetime != 5*time.Minute {
		t.Errorf("ConnMaxLifetime = %v, want 5m", cfg.Database.ConnMaxLifetime)
	}
	if cfg.LLM.Ollama.Timeout != 5*time.Minute {
		t.Errorf("Ollama.Timeout = %v, want 5m", cfg.LLM.Ollama.Timeout)
	}
}

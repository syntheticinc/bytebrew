package config

import (
	"testing"
)

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Port:      8443,
		TLSCert:   "cert.pem",
		TLSKey:    "key.pem",
		AuthToken: "secret",
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error for valid config, got %v", err)
	}
}

func TestValidate_InvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"zero port", 0},
		{"negative port", -1},
		{"port too high", 65536},
		{"port way too high", 100000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Port: tt.port}
			err := cfg.Validate()
			if err == nil {
				t.Errorf("expected error for port %d, got nil", tt.port)
			}
		})
	}
}

func TestValidate_PartialTLS(t *testing.T) {
	tests := []struct {
		name    string
		cert    string
		key     string
		wantErr bool
	}{
		{"cert only", "cert.pem", "", true},
		{"key only", "", "key.pem", true},
		{"both set", "cert.pem", "key.pem", false},
		{"neither set", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Port:    8443,
				TLSCert: tt.cert,
				TLSKey:  tt.key,
			}

			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestValidate_DefaultPort(t *testing.T) {
	cfg := Load()
	if cfg.Port != 8443 {
		t.Logf("port is %d (BRIDGE_PORT env may be set)", cfg.Port)
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("default config should be valid, got %v", err)
	}
}

func TestValidate_BoundaryPorts(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"port 1", 1, false},
		{"port 65535", 65535, false},
		{"port 80", 80, false},
		{"port 443", 443, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Port: tt.port}
			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("expected error for port %d, got nil", tt.port)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error for port %d, got %v", tt.port, err)
			}
		})
	}
}

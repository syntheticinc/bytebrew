package config

import "testing"

func TestModelRoutingConfig_RouteModel(t *testing.T) {
	cfg := ModelRoutingConfig{
		DefaultModel: "zai-org/GLM-5",
		RoleOverrides: map[string]string{
			"reviewer": "zai-org/GLM-4.7",
			"tester":   "zai-org/GLM-4.7",
		},
	}

	tests := []struct {
		name string
		role string
		want string
	}{
		{"supervisor uses default", "supervisor", "zai-org/GLM-5"},
		{"coder uses default", "coder", "zai-org/GLM-5"},
		{"reviewer uses override", "reviewer", "zai-org/GLM-4.7"},
		{"tester uses override", "tester", "zai-org/GLM-4.7"},
		{"unknown role uses default", "unknown", "zai-org/GLM-5"},
		{"empty role uses default", "", "zai-org/GLM-5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.RouteModel(tt.role)
			if got != tt.want {
				t.Errorf("RouteModel(%q) = %q, want %q", tt.role, got, tt.want)
			}
		})
	}
}

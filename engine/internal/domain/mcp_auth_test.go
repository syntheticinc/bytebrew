package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPAuthType_IsValid(t *testing.T) {
	tests := []struct {
		authType MCPAuthType
		valid    bool
	}{
		{MCPAuthNone, true},
		{MCPAuthAPIKey, true},
		{MCPAuthForwardHeaders, true},
		{MCPAuthOAuth2, true},
		{MCPAuthServiceAccount, true},
		{MCPAuthType("unknown"), false},
		{MCPAuthType(""), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.authType), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.authType.IsValid())
		})
	}
}

func TestMCPAuthConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  MCPAuthConfig
		wantErr bool
	}{
		{"none", MCPAuthConfig{Type: MCPAuthNone}, false},
		{"api_key with env", MCPAuthConfig{Type: MCPAuthAPIKey, KeyEnv: "TAVILY_KEY"}, false},
		{"api_key without env", MCPAuthConfig{Type: MCPAuthAPIKey}, true},
		{"forward_headers", MCPAuthConfig{Type: MCPAuthForwardHeaders}, false},
		{"oauth2 with client_id", MCPAuthConfig{Type: MCPAuthOAuth2, ClientID: "abc"}, false},
		{"oauth2 without client_id", MCPAuthConfig{Type: MCPAuthOAuth2}, true},
		{"service_account with env", MCPAuthConfig{Type: MCPAuthServiceAccount, TokenEnv: "SA_TOKEN"}, false},
		{"service_account without env", MCPAuthConfig{Type: MCPAuthServiceAccount}, true},
		{"invalid type", MCPAuthConfig{Type: "bogus"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

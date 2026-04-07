package domain

import "fmt"

// MCPAuthType represents the authentication type for an MCP server.
type MCPAuthType string

const (
	MCPAuthNone           MCPAuthType = "none"
	MCPAuthAPIKey         MCPAuthType = "api_key"
	MCPAuthForwardHeaders MCPAuthType = "forward_headers"
	MCPAuthOAuth2         MCPAuthType = "oauth2"
	MCPAuthServiceAccount MCPAuthType = "service_account"
)

// IsValid returns true if the auth type is recognized.
func (t MCPAuthType) IsValid() bool {
	switch t {
	case MCPAuthNone, MCPAuthAPIKey, MCPAuthForwardHeaders, MCPAuthOAuth2, MCPAuthServiceAccount:
		return true
	}
	return false
}

// MCPAuthConfig holds authentication configuration for an MCP server.
type MCPAuthConfig struct {
	Type       MCPAuthType // required
	KeyEnv     string      // env var name for api_key
	TokenEnv   string      // env var name for service_account token
	ClientID   string      // oauth2 client ID
	TokenStore string      // oauth2 token storage ("encrypted")
}

// Validate validates the auth config.
func (c *MCPAuthConfig) Validate() error {
	if !c.Type.IsValid() {
		return fmt.Errorf("invalid MCP auth type: %s", c.Type)
	}
	switch c.Type {
	case MCPAuthAPIKey:
		if c.KeyEnv == "" {
			return fmt.Errorf("api_key auth requires key_env")
		}
	case MCPAuthServiceAccount:
		if c.TokenEnv == "" {
			return fmt.Errorf("service_account auth requires token_env")
		}
	case MCPAuthOAuth2:
		if c.ClientID == "" {
			return fmt.Errorf("oauth2 auth requires client_id")
		}
	}
	return nil
}

// MCPCatalogCategory represents a category for MCP servers in the catalog.
type MCPCatalogCategory string

const (
	MCPCategorySearch        MCPCatalogCategory = "search"
	MCPCategoryData          MCPCatalogCategory = "data"
	MCPCategoryCommunication MCPCatalogCategory = "communication"
	MCPCategoryDevTools      MCPCatalogCategory = "dev-tools"
	MCPCategoryProductivity  MCPCatalogCategory = "productivity"
	MCPCategoryPayments      MCPCatalogCategory = "payments"
	MCPCategoryGeneric       MCPCatalogCategory = "generic"
)

// MCPCatalogEnvVar describes an environment variable needed by a catalog server.
type MCPCatalogEnvVar struct {
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Required    bool   `yaml:"required" json:"required"`
	Secret      bool   `yaml:"secret,omitempty" json:"secret,omitempty"`
}

// MCPCatalogPackage describes a deployment option for a catalog server.
type MCPCatalogPackage struct {
	Type        string             `yaml:"type" json:"type"` // stdio, remote, docker
	Transport   string             `yaml:"transport,omitempty" json:"transport,omitempty"`
	Command     string             `yaml:"command,omitempty" json:"command,omitempty"`
	Args        []string           `yaml:"args,omitempty" json:"args,omitempty"`
	Image       string             `yaml:"image,omitempty" json:"image,omitempty"`
	URLTemplate string             `yaml:"url_template,omitempty" json:"url_template,omitempty"`
	EnvVars     []MCPCatalogEnvVar `yaml:"env_vars,omitempty" json:"env_vars,omitempty"`
}

// MCPCatalogTool describes a tool provided by a catalog server.
type MCPCatalogTool struct {
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
}

// MCPCatalogEntry represents a single server entry in the MCP catalog.
type MCPCatalogEntry struct {
	Name          string             `yaml:"name" json:"name"`
	Display       string             `yaml:"display" json:"display"`
	Description   string             `yaml:"description" json:"description"`
	Category      MCPCatalogCategory `yaml:"category" json:"category"`
	Verified      bool               `yaml:"verified" json:"verified"`
	Packages      []MCPCatalogPackage `yaml:"packages" json:"packages"`
	ProvidedTools []MCPCatalogTool   `yaml:"provided_tools,omitempty" json:"provided_tools,omitempty"`
}

// MCPCatalog represents the full MCP catalog loaded from YAML.
type MCPCatalog struct {
	CatalogVersion string            `yaml:"catalog_version" json:"catalog_version"`
	Servers        []MCPCatalogEntry `yaml:"servers" json:"servers"`
}

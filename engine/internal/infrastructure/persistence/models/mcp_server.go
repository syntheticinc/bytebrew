package models

import "time"

// MCPServerModel maps to the "mcp_servers" table.
type MCPServerModel struct {
	ID             uint      `gorm:"primaryKey"`
	Name           string    `gorm:"uniqueIndex;not null"`
	Type           string    `gorm:"type:varchar(20);not null"`
	Command        string    `gorm:"type:varchar(500)"`
	Args           string    `gorm:"type:text"`
	URL            string    `gorm:"type:varchar(500)"`
	EnvVars        string    `gorm:"type:text"`
	ForwardHeaders string    `gorm:"type:text"`              // JSON array of HTTP header names to forward
	IsWellKnown    bool      `gorm:"not null;default:false"`
	AuthType       string    `gorm:"type:varchar(30);not null;default:none"` // none, api_key, forward_headers, oauth2, service_account (AC-AUTH-01)
	AuthKeyEnv     string    `gorm:"type:varchar(255)"`                      // env var for api_key
	AuthTokenEnv   string    `gorm:"type:varchar(255)"`                      // env var for service_account/oauth2 token
	AuthClientID   string    `gorm:"type:varchar(255)"`                      // oauth2 client ID
	CatalogName    string    `gorm:"type:varchar(255)"`                      // name from mcp-catalog.yaml if installed from catalog
	CreatedAt      time.Time `gorm:"autoCreateTime"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`

	Runtime *MCPServerRuntimeModel `gorm:"foreignKey:MCPServerID"`
}

func (MCPServerModel) TableName() string { return "mcp_servers" }

// MCPServerRuntimeModel maps to the "mcp_server_runtime" table (1:1 with mcp_servers).
type MCPServerRuntimeModel struct {
	MCPServerID   uint       `gorm:"primaryKey"`
	Status        string     `gorm:"type:varchar(20);not null;default:disconnected"`
	StatusMessage string     `gorm:"type:varchar(500)"`
	ToolsCount    int        `gorm:"not null;default:0"`
	ConnectedAt   *time.Time
	UpdatedAt     time.Time  `gorm:"autoUpdateTime"`

	MCPServer MCPServerModel `gorm:"foreignKey:MCPServerID"`
}

func (MCPServerRuntimeModel) TableName() string { return "mcp_server_runtime" }

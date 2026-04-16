package models

// AgentMCPServer maps to the "agent_mcp_servers" join table (composite PK).
type AgentMCPServer struct {
	AgentID     string `gorm:"primaryKey;type:uuid"`
	MCPServerID string `gorm:"primaryKey;type:uuid"`
	TenantID    string `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000001'" json:"tenant_id"`

	Agent     AgentModel   `gorm:"foreignKey:AgentID"`
	MCPServer MCPServerModel `gorm:"foreignKey:MCPServerID"`
}

func (AgentMCPServer) TableName() string { return "agent_mcp_servers" }

package models

// AgentMCPServer maps to the "agent_mcp_servers" join table (composite PK).
type AgentMCPServer struct {
	AgentID     string `gorm:"primaryKey;type:uuid"`
	MCPServerID string `gorm:"primaryKey;type:uuid"`

	Agent     AgentModel   `gorm:"foreignKey:AgentID"`
	MCPServer MCPServerModel `gorm:"foreignKey:MCPServerID"`
}

func (AgentMCPServer) TableName() string { return "agent_mcp_servers" }

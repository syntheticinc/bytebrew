package models

// AgentMCPServer maps to the "agent_mcp_servers" join table (composite PK).
type AgentMCPServer struct {
	AgentID     uint `gorm:"primaryKey"`
	MCPServerID uint `gorm:"primaryKey"`

	Agent     AgentModel   `gorm:"foreignKey:AgentID"`
	MCPServer MCPServerModel `gorm:"foreignKey:MCPServerID"`
}

func (AgentMCPServer) TableName() string { return "agent_mcp_servers" }

package models

// AgentToolModel maps to the "agent_tools" table.
type AgentToolModel struct {
	ID        string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	AgentID   string `gorm:"type:uuid;not null;uniqueIndex:idx_agent_tool_type_name"`
	ToolType  string `gorm:"type:varchar(20);not null;uniqueIndex:idx_agent_tool_type_name"`
	ToolName  string `gorm:"type:varchar(255);not null;uniqueIndex:idx_agent_tool_type_name"`
	Config    string `gorm:"type:text"`
	SortOrder int    `gorm:"not null;default:0"`
	TenantID  string `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000001'" json:"tenant_id"`

	Agent AgentModel `gorm:"foreignKey:AgentID"`
}

func (AgentToolModel) TableName() string { return "agent_tools" }

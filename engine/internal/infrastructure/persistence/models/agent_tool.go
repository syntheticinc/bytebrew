package models

// AgentToolModel maps to the "agent_tools" table.
type AgentToolModel struct {
	ID        uint   `gorm:"primaryKey"`
	AgentID   uint   `gorm:"not null;uniqueIndex:idx_agent_tool_type_name"`
	ToolType  string `gorm:"type:varchar(20);not null;uniqueIndex:idx_agent_tool_type_name"`
	ToolName  string `gorm:"type:varchar(255);not null;uniqueIndex:idx_agent_tool_type_name"`
	Config    string `gorm:"type:text"`
	SortOrder int    `gorm:"not null;default:0"`

	Agent AgentModel `gorm:"foreignKey:AgentID"`
}

func (AgentToolModel) TableName() string { return "agent_tools" }

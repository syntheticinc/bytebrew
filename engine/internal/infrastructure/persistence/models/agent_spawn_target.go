package models

// AgentSpawnTarget maps to the "agent_spawn_targets" table.
type AgentSpawnTarget struct {
	ID            string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	AgentID       string `gorm:"type:uuid;not null;uniqueIndex:idx_agent_spawn_pair"`
	TargetAgentID string `gorm:"type:uuid;not null;uniqueIndex:idx_agent_spawn_pair"`

	Agent       AgentModel `gorm:"foreignKey:AgentID"`
	TargetAgent AgentModel `gorm:"foreignKey:TargetAgentID"`
}

func (AgentSpawnTarget) TableName() string { return "agent_spawn_targets" }

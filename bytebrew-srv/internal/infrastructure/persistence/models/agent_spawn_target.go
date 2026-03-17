package models

// AgentSpawnTarget maps to the "agent_spawn_targets" table.
type AgentSpawnTarget struct {
	ID            uint `gorm:"primaryKey"`
	AgentID       uint `gorm:"not null;uniqueIndex:idx_agent_spawn_pair"`
	TargetAgentID uint `gorm:"not null;uniqueIndex:idx_agent_spawn_pair"`

	Agent       AgentModel `gorm:"foreignKey:AgentID"`
	TargetAgent AgentModel `gorm:"foreignKey:TargetAgentID"`
}

func (AgentSpawnTarget) TableName() string { return "agent_spawn_targets" }

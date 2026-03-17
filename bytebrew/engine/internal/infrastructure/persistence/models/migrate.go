package models

import "gorm.io/gorm"

// AutoMigrate registers all 16 engine tables and runs GORM auto-migration.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		// Config tables (11)
		&AgentModel{},
		&AgentToolModel{},
		&AgentSpawnTarget{},
		&AgentEscalation{},
		&AgentEscalationTrigger{},
		&LLMProviderModel{},
		&MCPServerModel{},
		&MCPServerRuntimeModel{},
		&AgentMCPServer{},
		&TriggerModel{},
		&SettingModel{},

		// Runtime tables (5)
		&SessionModel{},
		&TaskModel{},
		&SessionEventModel{},
		&APITokenModel{},
		&AuditLogModel{},
	)
}

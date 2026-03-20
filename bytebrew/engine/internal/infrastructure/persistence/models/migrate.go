package models

import "gorm.io/gorm"

// AutoMigrate registers all engine tables and runs GORM auto-migration.
func AutoMigrate(db *gorm.DB) error {
	// Ensure pgvector extension exists (required for Knowledge/RAG vector search).
	// Silently ignored if extension is not available (non-pgvector PostgreSQL).
	db.Exec("CREATE EXTENSION IF NOT EXISTS vector")

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

		// Dashboard runtime tables (5)
		&SessionModel{},
		&TaskModel{},
		&SessionEventModel{},
		&APITokenModel{},
		&AuditLogModel{},

		// Agent runtime tables (9)
		&RuntimeSessionModel{},
		&RuntimeTaskModel{},
		&RuntimeSubtaskModel{},
		&RuntimeAgentRunModel{},
		&RuntimeDeviceModel{},
		&RuntimeConfigKV{},
		&RuntimeSessionEventModel{},
		&RuntimeMessageModel{},
		&RuntimeAgentContextModel{},

		// Knowledge / RAG tables (2)
		&KnowledgeDocument{},
		&KnowledgeChunk{},
	)
}

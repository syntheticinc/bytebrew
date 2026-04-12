package models

import (
	"fmt"

	"gorm.io/gorm"
)

// AutoMigrate registers all engine tables and runs GORM auto-migration.
func AutoMigrate(db *gorm.DB) error {
	// Ensure pgvector extension exists (required for Knowledge/RAG vector search).
	// Silently ignored if extension is not available (non-pgvector PostgreSQL).
	db.Exec("CREATE EXTENSION IF NOT EXISTS vector")

	// WP-4: Migrate existing knowledge_chunks.embedding from vector(768) to variable-dimension vector.
	// Safe to run repeatedly — no-op if column already has correct type or table doesn't exist.
	db.Exec("ALTER TABLE knowledge_chunks ALTER COLUMN embedding TYPE vector USING embedding::vector")

	if err := db.AutoMigrate(
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
		&RuntimeEventModel{},
		&RuntimeAgentContextModel{},

		// Knowledge / RAG tables (2)
		&KnowledgeDocument{},
		&KnowledgeChunk{},

		// Schema / flow tables (4)
		&SchemaModel{},
		&SchemaAgentModel{},
		&GateModel{},
		&EdgeModel{},

		// Capability table (1)
		&CapabilityModel{},

		// Memory table (1)
		&MemoryModel{},

		// Tenant + Widget tables (2)
		&TenantModel{},
		&WidgetModel{},
	); err != nil {
		return err
	}

	// Partial unique index: enforce uniqueness on webhook_path only when non-empty.
	// This allows multiple cron triggers with empty webhook_path without conflicts.
	// DROP the old full unique index first (ignore error if it doesn't exist).
	db.Exec("DROP INDEX IF EXISTS idx_triggers_webhook_path")
	if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_triggers_webhook_path_nonempty
		ON triggers (webhook_path) WHERE webhook_path != ''`).Error; err != nil {
		return fmt.Errorf("create partial unique index on webhook_path: %w", err)
	}

	return nil
}

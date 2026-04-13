package models

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
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

		// Dashboard runtime tables (4)
		&SessionModel{},
		&SessionEventModel{},
		&APITokenModel{},
		&AuditLogModel{},

		// Agent runtime tables (7)
		&RuntimeSessionModel{},
		&RuntimeAgentRunModel{},
		&RuntimeDeviceModel{},
		&RuntimeConfigKV{},
		&RuntimeSessionEventModel{},
		&RuntimeEventModel{},
		&RuntimeAgentContextModel{},

		// Knowledge / RAG tables (4)
		&KnowledgeBase{},
		&KnowledgeBaseAgent{},
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

	// Migrate legacy knowledge documents (agent_name-scoped) to knowledge_base-scoped.
	// Creates a KnowledgeBase per unique (tenant_id, agent_name) and links documents + chunks.
	if err := migrateKnowledgeToKB(db); err != nil {
		return fmt.Errorf("migrate knowledge to KB: %w", err)
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

// migrateKnowledgeToKB migrates legacy agent_name-scoped knowledge documents to KB-scoped.
// Idempotent: skips documents that already have a knowledge_base_id set.
func migrateKnowledgeToKB(db *gorm.DB) error {
	// Find legacy documents: have agent_name but no knowledge_base_id.
	type legacyPair struct {
		TenantID  string
		AgentName string
	}
	var pairs []legacyPair
	if err := db.Raw(`SELECT DISTINCT tenant_id, agent_name FROM knowledge_documents
		WHERE agent_name != '' AND (knowledge_base_id IS NULL OR knowledge_base_id = '')`).
		Scan(&pairs).Error; err != nil {
		return nil // table may not exist yet on first run — safe to skip
	}

	if len(pairs) == 0 {
		return nil
	}

	slog.Info("[Migration] migrating legacy knowledge documents to knowledge bases", "pairs", len(pairs))

	for _, p := range pairs {
		kbID := uuid.New().String()
		now := time.Now()

		// Create KB named after agent
		kb := KnowledgeBase{
			ID:        kbID,
			TenantID:  p.TenantID,
			Name:      p.AgentName,
			CreatedAt: now,
			UpdatedAt: now,
		}

		// Try to resolve embedding_model_id from agent's capability config.
		var embModelID *string
		var agentID string
		db.Raw("SELECT id FROM agents WHERE name = ?", p.AgentName).Scan(&agentID)
		if agentID != "" {
			var config string
			db.Raw("SELECT config FROM capabilities WHERE agent_id = ? AND type = 'knowledge'", agentID).Scan(&config)
			if config != "" {
				// Simple JSON extraction — avoid importing encoding/json in models package.
				// Look for "embedding_model_id":"<uuid>"
				if idx := findJSONString(config, "embedding_model_id"); idx != "" {
					embModelID = &idx
				}
			}
		}
		kb.EmbeddingModelID = embModelID

		if err := db.Create(&kb).Error; err != nil {
			slog.Warn("[Migration] failed to create KB for legacy pair, skipping",
				"tenant", p.TenantID, "agent", p.AgentName, "error", err)
			continue
		}

		// Link agent if found
		if agentID != "" {
			db.Exec("INSERT INTO knowledge_base_agents (knowledge_base_id, agent_name) VALUES (?, ?) ON CONFLICT DO NOTHING",
				kbID, p.AgentName)
		}

		// Update documents
		db.Exec("UPDATE knowledge_documents SET knowledge_base_id = ? WHERE tenant_id = ? AND agent_name = ? AND (knowledge_base_id IS NULL OR knowledge_base_id = '')",
			kbID, p.TenantID, p.AgentName)

		// Update chunks
		db.Exec("UPDATE knowledge_chunks SET knowledge_base_id = ? WHERE tenant_id = ? AND agent_name = ? AND (knowledge_base_id IS NULL OR knowledge_base_id = '')",
			kbID, p.TenantID, p.AgentName)

		slog.Info("[Migration] migrated knowledge to KB",
			"kb_id", kbID, "tenant", p.TenantID, "agent", p.AgentName)
	}

	return nil
}

// findJSONString extracts a string value from a JSON object by key (simple, no dependency).
func findJSONString(jsonStr, key string) string {
	needle := `"` + key + `":"`
	idx := 0
	for {
		pos := indexOf(jsonStr[idx:], needle)
		if pos < 0 {
			return ""
		}
		idx += pos + len(needle)
		end := indexOf(jsonStr[idx:], `"`)
		if end < 0 {
			return ""
		}
		return jsonStr[idx : idx+end]
	}
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

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

	// V2 Tasks unification: drop legacy V1 tables that are superseded by the
	// unified EngineTask/TaskModel. No data migration is needed — V1 Task / Subtask
	// entities were session-scoped and already expired by the time V2 ships.
	// Safe to run repeatedly: DROP IF EXISTS is a no-op when the table is missing.
	if err := db.Migrator().DropTable("runtime_tasks", "runtime_subtasks"); err != nil {
		// DropTable returns an error if the underlying DB refuses the DROP (e.g. permission).
		// We never want to block startup on a dev DB that does not have these tables,
		// so log and continue — the AutoMigrate below is what really matters.
		slog.Warn("[Migration] dropping legacy V1 task tables failed (may already be absent)", "error", err)
	}

	// V2 Gate removal: gates are out of V2 (see docs/architecture/agent-first-runtime.md §3).
	// Drop legacy `gates` table if present. Clean-schema policy: pure DDL, no data
	// preservation — fresh install is the supported path. Edge-type removal and the
	// edges→agent_relations rename are handled in a separate commit group.
	if err := db.Migrator().DropTable("gates"); err != nil {
		slog.Warn("[Migration] dropping legacy gates table failed (may already be absent)", "error", err)
	}

	// V2 Escalation removal: Escalation capability is out of V2 (see §5.9).
	// Drop legacy agent_escalation_triggers (child) then agent_escalation (parent).
	// Order matters because of the FK from triggers → escalation.
	if err := db.Migrator().DropTable("agent_escalation_triggers", "agent_escalation"); err != nil {
		slog.Warn("[Migration] dropping legacy escalation tables failed (may already be absent)", "error", err)
	}

	// V2 Kit + knowledge_path removal: Kit was a never-implemented slot, superseded
	// by V2 Capabilities + MCP Catalog. knowledge_path is superseded by capability
	// Knowledge + knowledge_base_agents M2M. See §5.* and Commit Group J.
	// Defense-in-depth alongside Liquibase 011 — idempotent DropColumn.
	if db.Migrator().HasTable("agents") {
		if db.Migrator().HasColumn("agents", "kit") {
			if err := db.Migrator().DropColumn("agents", "kit"); err != nil {
				slog.Warn("[Migration] dropping legacy agents.kit column failed (may already be absent)", "error", err)
			}
		}
		if db.Migrator().HasColumn("agents", "knowledge_path") {
			if err := db.Migrator().DropColumn("agents", "knowledge_path"); err != nil {
				slog.Warn("[Migration] dropping legacy agents.knowledge_path column failed (may already be absent)", "error", err)
			}
		}
	}

	// V2 edges → agent_relations rename + drop type column (Commit Group A.1).
	// Defense-in-depth alongside Liquibase 012 — idempotent rename + DropColumn.
	// Target schema models a single implicit DELEGATION type — no per-row type
	// column. See docs/architecture/agent-first-runtime.md §3.1.
	if db.Migrator().HasTable("edges") && !db.Migrator().HasTable("agent_relations") {
		if err := db.Migrator().RenameTable("edges", "agent_relations"); err != nil {
			slog.Warn("[Migration] renaming legacy edges table to agent_relations failed", "error", err)
		}
	}
	if db.Migrator().HasTable("agent_relations") && db.Migrator().HasColumn("agent_relations", "type") {
		if err := db.Migrator().DropColumn("agent_relations", "type"); err != nil {
			slog.Warn("[Migration] dropping legacy agent_relations.type column failed (may already be absent)", "error", err)
		}
	}

	// V2 schema_agents removal (Commit Group F).
	// Defense-in-depth alongside Liquibase 013 — idempotent DropTable.
	// Schema membership is derived from `agent_relations` (entry agent +
	// relation source/target). See docs/architecture/agent-first-runtime.md
	// §2.1.
	if err := db.Migrator().DropTable("schema_agents"); err != nil {
		slog.Warn("[Migration] dropping legacy schema_agents table failed (may already be absent)", "error", err)
	}

	// V2 triggers cleanup (Commit Group D).
	// Defense-in-depth alongside Liquibase 014 — idempotent DropColumn.
	// Type-specific fields (schedule, webhook_path) collapse into the
	// `config` jsonb column. The on_complete webhook feature is removed
	// entirely. See docs/architecture/agent-first-runtime.md §4.1 / §4.2.
	if db.Migrator().HasTable("triggers") {
		for _, col := range []string{"schedule", "webhook_path", "on_complete_url", "on_complete_headers"} {
			if db.Migrator().HasColumn("triggers", col) {
				if err := db.Migrator().DropColumn("triggers", col); err != nil {
					slog.Warn("[Migration] dropping legacy triggers column failed (may already be absent)", "column", col, "error", err)
				}
			}
		}
	}

	// V2 widgets removal (Commit Group E).
	// Defense-in-depth alongside Liquibase 015 — idempotent DropTable.
	// A chat widget is a client (same class as web-client / mobile / CLI),
	// not a domain entity — there is no server-side widget configuration to
	// persist. The admin UI becomes a pure snippet generator. See
	// docs/architecture/agent-first-runtime.md §4.3.
	if err := db.Migrator().DropTable("widgets"); err != nil {
		slog.Warn("[Migration] dropping legacy widgets table failed (may already be absent)", "error", err)
	}

	// V2 MCP catalog split + runtime cleanup (Commit Group C, §5.5/§5.6).
	// Defense-in-depth alongside Liquibase 016 — idempotent DropTable +
	// DropColumn. The `mcp_server_runtime` table is gone (status is answered
	// live via MCP client ping/ListTools, not persisted). `is_well_known` and
	// `catalog_name` are gone because the catalog lives in its own
	// `mcp_catalog` table and install-from-catalog is a copy operation with
	// no link back.
	if err := db.Migrator().DropTable("mcp_server_runtime"); err != nil {
		slog.Warn("[Migration] dropping legacy mcp_server_runtime table failed (may already be absent)", "error", err)
	}
	if db.Migrator().HasTable("mcp_servers") {
		for _, col := range []string{"is_well_known", "catalog_name"} {
			if db.Migrator().HasColumn("mcp_servers", col) {
				if err := db.Migrator().DropColumn("mcp_servers", col); err != nil {
					slog.Warn("[Migration] dropping legacy mcp_servers column failed (may already be absent)", "column", col, "error", err)
				}
			}
		}
	}

	// V2 runtime_events → messages rename (Commit Group M.1).
	// Defense-in-depth alongside Liquibase 019 — idempotent rename.
	// See docs/database/target-schema.dbml: Table messages.
	if db.Migrator().HasTable("runtime_events") && !db.Migrator().HasTable("messages") {
		if err := db.Migrator().RenameTable("runtime_events", "messages"); err != nil {
			slog.Warn("[Migration] renaming legacy runtime_events table to messages failed", "error", err)
		}
	}

	// V2 runtime_session_events → session_event_log rename (Commit Group M.2).
	// Defense-in-depth alongside Liquibase 020 — idempotent rename.
	// See docs/database/target-schema.dbml: Table session_event_log.
	if db.Migrator().HasTable("runtime_session_events") && !db.Migrator().HasTable("session_event_log") {
		if err := db.Migrator().RenameTable("runtime_session_events", "session_event_log"); err != nil {
			slog.Warn("[Migration] renaming legacy runtime_session_events table to session_event_log failed", "error", err)
		}
	}

	// V2 settings final shape (Commit Group G, §5.8).
	// Defense-in-depth alongside Liquibase 018 — idempotent DropColumn for the
	// legacy `scope` column. The composite PK (tenant_id, key) and the jsonb
	// `value` column are installed by Liquibase 018 itself; GORM AutoMigrate
	// matches the final shape declared on SettingModel.
	if db.Migrator().HasTable("settings") && db.Migrator().HasColumn("settings", "scope") {
		if err := db.Migrator().DropColumn("settings", "scope"); err != nil {
			slog.Warn("[Migration] dropping legacy settings.scope column failed (may already be absent)", "error", err)
		}
	}

	if err := db.AutoMigrate(
		// Config tables (9)
		&AgentModel{},
		&AgentToolModel{},
		&AgentSpawnTarget{},
		&LLMProviderModel{},
		&MCPServerModel{},
		&MCPCatalogModel{},
		&AgentMCPServer{},
		&TriggerModel{},
		&SettingModel{},

		// Dashboard runtime tables (5)
		&SessionModel{},
		&TaskModel{},
		&SessionEventModel{},
		&APITokenModel{},
		&AuditLogModel{},

		// Agent runtime tables
		&RuntimeSessionModel{},
		&RuntimeAgentRunModel{},
		&RuntimeDeviceModel{},
		&RuntimeConfigKV{},
		&SessionEventLogModel{},
		&MessageModel{},
		&RuntimeAgentContextModel{},

		// Knowledge / RAG tables (4)
		&KnowledgeBase{},
		&KnowledgeBaseAgent{},
		&KnowledgeDocument{},
		&KnowledgeChunk{},

		// Schema / flow tables (2)
		&SchemaModel{},
		&AgentRelationModel{},

		// Schema template catalog (1) — V2 Commit Group L (§2.2).
		&SchemaTemplateModel{},

		// Capability table (1)
		&CapabilityModel{},

		// Memory table (1)
		&MemoryModel{},

		// Tenant table (1)
		&TenantModel{},
	); err != nil {
		return err
	}

	// Migrate legacy knowledge documents (agent_name-scoped) to knowledge_base-scoped.
	// Creates a KnowledgeBase per unique (tenant_id, agent_name) and links documents + chunks.
	if err := migrateKnowledgeToKB(db); err != nil {
		return fmt.Errorf("migrate knowledge to KB: %w", err)
	}

	// V2 triggers cleanup (Commit Group D): the legacy partial unique index
	// on `triggers.webhook_path` is obsolete — the column itself was dropped.
	// Drop the index if a pre-V2 DB still has it. Any future uniqueness on
	// webhook_path must be expressed against `config->>'webhook_path'`.
	db.Exec("DROP INDEX IF EXISTS idx_triggers_webhook_path")
	db.Exec("DROP INDEX IF EXISTS idx_triggers_webhook_path_nonempty")

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

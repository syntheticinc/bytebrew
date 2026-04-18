package models

import (
	"log/slog"

	"gorm.io/gorm"
)

// AutoMigrate registers all engine tables and runs GORM auto-migration.
//
// V2 fresh-start policy: the project is not in production yet, so this file
// produces the target shape directly — no multi-step ALTERs, no data
// preservation. `target-schema.dbml` is the authoritative spec.
//
// The legacy cleanup routines (drop-old-tables, rename-old-columns, backfill)
// that accumulated during earlier V1→V2 work have been removed. To upgrade a
// dev database: `docker compose ... down -v` and let this run on an empty DB.
func AutoMigrate(db *gorm.DB) error {
	// pgvector extension is required for Knowledge/RAG embeddings.
	db.Exec("CREATE EXTENSION IF NOT EXISTS vector")

	// Defense-in-depth: if a dev DB has the legacy `triggers` table left over
	// from pre-schema-alignment work, drop it. V2 replaced triggers with the
	// schemas.chat_enabled flag.
	if err := db.Migrator().DropTable("triggers"); err != nil {
		slog.Warn("[Migration] dropping legacy triggers table failed (may already be absent)", "error", err)
	}

	if err := db.AutoMigrate(
		// Identity
		&UserModel{},

		// Config tables
		&AgentModel{},
		&AgentToolModel{},
		&LLMProviderModel{},
		&MCPServerModel{},
		&MCPCatalogModel{},
		&AgentMCPServer{},
		&SettingModel{},
		&CapabilityModel{},

		// Schemas / flow
		&SchemaModel{},
		&AgentRelationModel{},
		&SchemaTemplateModel{},

		// Runtime (user-facing)
		&SessionModel{},
		&MessageModel{},
		&SessionEventLogModel{},
		&TaskModel{},
		&MemoryModel{},
		&AgentRunModel{},
		&AgentContextSnapshotModel{},

		// Knowledge / RAG
		&KnowledgeBase{},
		&KnowledgeBaseAgent{},
		&KnowledgeDocument{},
		&KnowledgeChunk{},

		// Observability / auth
		&AuditLogModel{},
		&APITokenModel{},
	); err != nil {
		return err
	}

	return applyRawConstraints(db)
}

// applyRawConstraints installs DB-level constraints and partial indexes that
// GORM struct tags cannot express: partial unique indexes, CHECK constraints
// on enum columns, and vector-type coercion for knowledge embeddings.
//
// Each statement is idempotent or uses IF EXISTS/NOT EXISTS so that repeated
// AutoMigrate calls are safe.
func applyRawConstraints(db *gorm.DB) error {
	statements := []string{
		// Agent context snapshots: at most one ACTIVE snapshot per (session, agent).
		// GORM's uniqueIndex tag produces a full unique index on SQLite (test env),
		// but in Postgres we want a partial index so compacted/expired rows can
		// coexist with the active one. Replace the full unique with the partial.
		`DROP INDEX IF EXISTS idx_ctx_snapshots_session_agent`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_ctx_snapshots_active_unique
		 ON agent_context_snapshots (session_id, agent_id)
		 WHERE status = 'active'`,

		// Knowledge chunk embeddings: variable-dim vector (was fixed dim pre-V2).
		`ALTER TABLE knowledge_chunks ALTER COLUMN embedding TYPE vector USING embedding::vector`,

		// CHECK constraints — enum fields. Use DO blocks so they're idempotent.
		dropCheck("agents", "chk_agents_lifecycle"),
		`ALTER TABLE agents ADD CONSTRAINT chk_agents_lifecycle
		 CHECK (lifecycle IN ('persistent','spawn'))`,

		dropCheck("agents", "chk_agents_tool_execution"),
		`ALTER TABLE agents ADD CONSTRAINT chk_agents_tool_execution
		 CHECK (tool_execution IN ('sequential','parallel'))`,

		dropCheck("sessions", "chk_sessions_status"),
		`ALTER TABLE sessions ADD CONSTRAINT chk_sessions_status
		 CHECK (status IN ('active','completed','expired','failed'))`,

		dropCheck("tasks", "chk_tasks_status"),
		`ALTER TABLE tasks ADD CONSTRAINT chk_tasks_status
		 CHECK (status IN ('pending','in_progress','completed','failed','needs_input','cancelled'))`,

		dropCheck("tasks", "chk_tasks_mode"),
		`ALTER TABLE tasks ADD CONSTRAINT chk_tasks_mode
		 CHECK (mode IN ('interactive','background'))`,

		dropCheck("capabilities", "chk_capabilities_type"),
		`ALTER TABLE capabilities ADD CONSTRAINT chk_capabilities_type
		 CHECK (type IN ('memory','knowledge'))`,

		dropCheck("messages", "chk_messages_event_type"),
		`ALTER TABLE messages ADD CONSTRAINT chk_messages_event_type
		 CHECK (event_type IN ('user_message','assistant_message','tool_call','tool_result','reasoning','system'))`,

		dropCheck("agent_runs", "chk_agent_runs_status"),
		`ALTER TABLE agent_runs ADD CONSTRAINT chk_agent_runs_status
		 CHECK (status IN ('running','completed','failed','cancelled'))`,

		dropCheck("agent_context_snapshots", "chk_ctx_status"),
		`ALTER TABLE agent_context_snapshots ADD CONSTRAINT chk_ctx_status
		 CHECK (status IN ('active','suspended','completed','interrupted'))`,

		dropCheck("audit_logs", "chk_audit_actor_type"),
		`ALTER TABLE audit_logs ADD CONSTRAINT chk_audit_actor_type
		 CHECK (actor_type IN ('admin','api_token','system'))`,

		dropCheck("audit_logs", "chk_audit_actor_one_of"),
		`ALTER TABLE audit_logs ADD CONSTRAINT chk_audit_actor_one_of
		 CHECK ((actor_user_id IS NOT NULL) OR (actor_sub IS NOT NULL))`,

		dropCheck("agent_runs", "chk_agent_runs_completion_order"),
		`ALTER TABLE agent_runs ADD CONSTRAINT chk_agent_runs_completion_order
		 CHECK (completed_at IS NULL OR completed_at >= started_at)`,

		dropCheck("users", "chk_users_role"),
		`ALTER TABLE users ADD CONSTRAINT chk_users_role
		 CHECK (role IN ('admin','system'))`,
	}

	for _, stmt := range statements {
		if err := db.Exec(stmt).Error; err != nil {
			// CHECK constraint attach will fail if a row already violates it.
			// Log and continue — this is a fresh-start setup, not a migration.
			slog.Warn("[Migration] raw constraint statement failed", "stmt", stmt, "error", err)
		}
	}
	return nil
}

// dropCheck returns an idempotent statement that removes a named CHECK
// constraint if present, so that the following ADD CONSTRAINT can re-install it.
func dropCheck(table, name string) string {
	return `ALTER TABLE ` + table + ` DROP CONSTRAINT IF EXISTS ` + name
}

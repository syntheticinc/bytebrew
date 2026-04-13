-- Migration: Task v2 columns (unified EngineTask system)
-- Date: 2026-04-13
-- Target: PostgreSQL (GORM-managed tasks table)
--
-- CONTEXT:
--   The engine normally handles schema changes via GORM AutoMigrate at startup,
--   which adds missing columns idempotently. This SQL is a manual fallback for
--   operators who prefer explicit migrations or are running in a locked-down
--   environment where AutoMigrate is disabled.
--
-- SAFE TO RUN:
--   All statements use IF NOT EXISTS, so repeated execution is a no-op.
--   No data is destroyed. New columns are added with sensible defaults so
--   existing rows remain valid.
--
-- ROLLBACK:
--   See the commented ROLLBACK section at the end. Note that dropping columns
--   destroys the corresponding data.

BEGIN;

-- Priority (0=normal, 1=high, 2=critical). Default 0 for existing rows.
ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS priority INTEGER NOT NULL DEFAULT 0;

-- Acceptance criteria — JSON array serialized as text for portability.
-- NULL / empty string both mean "no criteria".
ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS acceptance_criteria TEXT;

-- Blocked-by dependency list — JSON array of task IDs serialized as text.
ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS blocked_by TEXT;

-- Assigned agent runtime id (set when AgentPool spawns a code-agent against the task).
ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS assigned_agent_id VARCHAR(100);

-- Approval timestamp (draft → approved transition).
ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS approved_at TIMESTAMP;

-- Explicit update timestamp (GORM's autoUpdateTime).
ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();

-- Supporting indexes for common queries.

-- Priority + created_at composite supports GetReadySubtasks ordering.
CREATE INDEX IF NOT EXISTS idx_tasks_priority_created
    ON tasks (priority DESC, created_at ASC);

-- Assigned agent lookup (GetByAgentID is called each turn).
CREATE INDEX IF NOT EXISTS idx_tasks_assigned_agent
    ON tasks (assigned_agent_id)
    WHERE assigned_agent_id IS NOT NULL;

COMMIT;

-- ROLLBACK (manual, destructive — removes data).
-- BEGIN;
-- DROP INDEX IF EXISTS idx_tasks_assigned_agent;
-- DROP INDEX IF EXISTS idx_tasks_priority_created;
-- ALTER TABLE tasks DROP COLUMN IF EXISTS updated_at;
-- ALTER TABLE tasks DROP COLUMN IF EXISTS approved_at;
-- ALTER TABLE tasks DROP COLUMN IF EXISTS assigned_agent_id;
-- ALTER TABLE tasks DROP COLUMN IF EXISTS blocked_by;
-- ALTER TABLE tasks DROP COLUMN IF EXISTS acceptance_criteria;
-- ALTER TABLE tasks DROP COLUMN IF EXISTS priority;
-- COMMIT;

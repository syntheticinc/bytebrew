-- Migration: FlowType purge — agent_id replaces flow_type (Group Q.4)
-- Date: 2026-04-16
-- Target: PostgreSQL
--
-- CONTEXT:
--   FlowType was a Code Agent concept (supervisor/coder enum). V2 agent-first
--   model identifies agents by uuid. agent_runs and agent_context_snapshots
--   now use agent_id uuid instead of flow_type string. subtask_id -> task_id uuid.
--
-- SAFE TO RUN:
--   All statements use IF NOT EXISTS / IF EXISTS, repeated execution is a no-op.
--
-- ROLLBACK:
--   See the commented ROLLBACK section at the end.

BEGIN;

-- agent_runs: add agent_id uuid, add task_id uuid, drop flow_type, drop subtask_id
ALTER TABLE agent_runs ADD COLUMN IF NOT EXISTS agent_id uuid;
ALTER TABLE agent_runs ADD COLUMN IF NOT EXISTS task_id uuid;
ALTER TABLE agent_runs DROP COLUMN IF EXISTS flow_type;
ALTER TABLE agent_runs DROP COLUMN IF EXISTS subtask_id;

-- agent_context_snapshots: drop flow_type, change agent_id to uuid
ALTER TABLE agent_context_snapshots DROP COLUMN IF EXISTS flow_type;
ALTER TABLE agent_context_snapshots ALTER COLUMN agent_id TYPE uuid USING agent_id::uuid;

COMMIT;

-- ROLLBACK (manual, destructive — removes data).
-- BEGIN;
-- ALTER TABLE agent_runs ADD COLUMN IF NOT EXISTS flow_type VARCHAR(50) NOT NULL DEFAULT 'coder';
-- ALTER TABLE agent_runs ADD COLUMN IF NOT EXISTS subtask_id VARCHAR(36);
-- ALTER TABLE agent_runs DROP COLUMN IF EXISTS agent_id;
-- ALTER TABLE agent_runs DROP COLUMN IF EXISTS task_id;
-- ALTER TABLE agent_context_snapshots ADD COLUMN IF NOT EXISTS flow_type VARCHAR(50) NOT NULL DEFAULT '';
-- ALTER TABLE agent_context_snapshots ALTER COLUMN agent_id TYPE VARCHAR(100);
-- COMMIT;

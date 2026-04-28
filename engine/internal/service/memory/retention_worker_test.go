package memory

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// capabilitiesTableDDL is a SQLite-compatible analogue of the production
// `capabilities` table. We only include the columns the worker reads
// (id/agent_id/type/config/enabled).
const capabilitiesTableDDL = `CREATE TABLE IF NOT EXISTS capabilities (
    id         TEXT PRIMARY KEY,
    agent_id   TEXT NOT NULL,
    type       TEXT NOT NULL,
    config     TEXT NOT NULL DEFAULT '',
    enabled    INTEGER NOT NULL DEFAULT 1,
    tenant_id  TEXT NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);`

// agentsAndRelationsDDL satisfies the worker's raw queries so the skip
// branches can be exercised without migrating the full memory schema.
const agentsAndRelationsDDL = `CREATE TABLE IF NOT EXISTS agents (
    id   TEXT PRIMARY KEY,
    name TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS agent_relations (
    schema_id        TEXT NOT NULL,
    source_agent_id  TEXT,
    target_agent_id  TEXT
);`

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, db.Exec(capabilitiesTableDDL).Error)
	for _, stmt := range splitDDL(agentsAndRelationsDDL) {
		require.NoError(t, db.Exec(stmt).Error)
	}
	return db
}

// splitDDL splits a multi-statement DDL string on ";\n" so each CREATE TABLE
// is executed separately (gorm.Exec only runs the first statement).
func splitDDL(ddl string) []string {
	var out []string
	current := ""
	for _, r := range ddl {
		if r == ';' {
			s := trimSpace(current)
			if s != "" {
				out = append(out, s+";")
			}
			current = ""
			continue
		}
		current += string(r)
	}
	s := trimSpace(current)
	if s != "" {
		out = append(out, s)
	}
	return out
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\n' || s[0] == '\t' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 {
		last := s[len(s)-1]
		if last == ' ' || last == '\n' || last == '\t' || last == '\r' {
			s = s[:len(s)-1]
			continue
		}
		break
	}
	return s
}

// insertCap is a tiny helper to seed a capability row without needing the
// full repository plumbing.
func insertCap(t *testing.T, db *gorm.DB, agentID, capType, configJSON string, enabled bool) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO capabilities (id, agent_id, type, config, enabled) VALUES (?, ?, ?, ?, ?)`,
		"cap-"+agentID, agentID, capType, configJSON, boolToInt(enabled),
	).Error)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// TestRunOnce_SkipsUnlimitedRetention verifies that capabilities flagged
// with `unlimited_retention=true` are not subject to cleanup.
func TestRunOnce_SkipsUnlimitedRetention(t *testing.T) {
	db := setupTestDB(t)
	insertCap(t, db, "agent-unlimited", "memory", `{"unlimited_retention": true, "retention_days": 1}`, true)

	w := NewRetentionWorker(db)
	deleted := w.runOnce(context.Background())

	require.EqualValues(t, 0, deleted, "no rows should be cleaned for unlimited_retention=true")
}

// TestRunOnce_SkipsZeroRetentionDays verifies that capabilities with
// retention_days <= 0 are not subject to cleanup (treated as unlimited).
func TestRunOnce_SkipsZeroRetentionDays(t *testing.T) {
	db := setupTestDB(t)
	insertCap(t, db, "agent-zero", "memory", `{"retention_days": 0}`, true)
	insertCap(t, db, "agent-negative", "memory", `{"retention_days": -5}`, true)

	w := NewRetentionWorker(db)
	deleted := w.runOnce(context.Background())

	require.EqualValues(t, 0, deleted, "no rows should be cleaned for retention_days <= 0")
}

// TestRunOnce_SkipsDisabledCapability verifies that disabled capabilities
// are filtered by the initial WHERE clause.
func TestRunOnce_SkipsDisabledCapability(t *testing.T) {
	db := setupTestDB(t)
	insertCap(t, db, "agent-disabled", "memory", `{"retention_days": 7}`, false)

	w := NewRetentionWorker(db)
	deleted := w.runOnce(context.Background())

	require.EqualValues(t, 0, deleted, "disabled capabilities should not be processed")
}

// TestRunOnce_SkipsEmptyConfig verifies that capabilities with an empty
// config field are skipped (no JSON parse attempt).
func TestRunOnce_SkipsEmptyConfig(t *testing.T) {
	db := setupTestDB(t)
	insertCap(t, db, "agent-empty", "memory", "", true)

	w := NewRetentionWorker(db)
	deleted := w.runOnce(context.Background())

	require.EqualValues(t, 0, deleted, "empty config should be skipped without errors")
}

//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TC-MIGR-01: Expected tables are present after Liquibase apply.
func TestMIGR01_TablesExist(t *testing.T) {
	requireSuite(t)
	require.NotNil(t, testDB, "testDB must be initialised")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Smoke: SELECT on each expected table returns without error. A missing
	// table surfaces a useful "relation does not exist" message via GORM.
	for _, tbl := range []string{"agents", "schemas", "sessions", "users", "audit_logs"} {
		var count int64
		err := testDB.WithContext(ctx).Raw(
			`SELECT COUNT(*) FROM ` + `"` + ensureTableName(tbl) + `"`,
		).Scan(&count).Error
		assert.NoError(t, err, "SELECT COUNT(*) FROM %s failed", tbl)
	}
}

// TC-MIGR-02: The seeded admin user has tenant_id = CE default tenant.
func TestMIGR02_AdminTenantDefault(t *testing.T) {
	requireSuite(t)
	require.NotNil(t, testDB)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var count int64
	err := testDB.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM users WHERE tenant_id = ?::uuid`, ceTenantID,
	).Scan(&count).Error
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(1),
		"admin user seeded with tenant_id=%s should exist", ceTenantID)
}

// TC-MIGR-03: Public schema has at least a reasonable number of tables —
// ballparks "migrations actually ran" without pinning the exact count.
func TestMIGR03_SchemaSanity(t *testing.T) {
	requireSuite(t)
	require.NotNil(t, testDB)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var count int64
	err := testDB.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'`,
	).Scan(&count).Error
	require.NoError(t, err)
	assert.Greater(t, count, int64(10),
		"public schema should have >10 tables after migrations; got %d", count)
}

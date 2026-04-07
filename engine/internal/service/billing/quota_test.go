package billing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

type mockUsageReader struct {
	usage *domain.TenantUsage
}

func (m *mockUsageReader) GetUsage(ctx context.Context, tenantID string) (*domain.TenantUsage, error) {
	return m.usage, nil
}

type mockPlanReader struct {
	plan domain.CloudPlan
}

func (m *mockPlanReader) GetPlan(ctx context.Context, tenantID string) (domain.CloudPlan, error) {
	return m.plan, nil
}

func TestQuotaEnforcer_APICall_Allowed(t *testing.T) {
	enforcer := NewQuotaEnforcer(
		&mockUsageReader{usage: &domain.TenantUsage{APICalls: 500}},
		&mockPlanReader{plan: domain.PlanFree},
		"/upgrade",
	)

	result, err := enforcer.CheckAPICall(context.Background(), "tenant-1")
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, "api_calls", result.Resource)
	assert.Equal(t, int64(500), result.Used)
	assert.Equal(t, int64(1000), result.Limit)
}

func TestQuotaEnforcer_APICall_Blocked(t *testing.T) {
	// AC-PRICE-02: limit reached → blocked with upgrade URL
	enforcer := NewQuotaEnforcer(
		&mockUsageReader{usage: &domain.TenantUsage{APICalls: 1000}},
		&mockPlanReader{plan: domain.PlanFree},
		"/upgrade",
	)

	result, err := enforcer.CheckAPICall(context.Background(), "tenant-1")
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, "/upgrade", result.UpgradeURL)
	assert.Equal(t, "blocked", result.WarningLevel())
}

func TestQuotaEnforcer_Schema_FreePlan(t *testing.T) {
	// AC-PRICE-01: Free plan = 1 schema
	enforcer := NewQuotaEnforcer(
		&mockUsageReader{usage: &domain.TenantUsage{Schemas: 1}},
		&mockPlanReader{plan: domain.PlanFree},
		"/upgrade",
	)

	result, err := enforcer.CheckSchema(context.Background(), "tenant-1")
	require.NoError(t, err)
	assert.False(t, result.Allowed)
}

func TestQuotaEnforcer_Schema_ProPlan(t *testing.T) {
	enforcer := NewQuotaEnforcer(
		&mockUsageReader{usage: &domain.TenantUsage{Schemas: 3}},
		&mockPlanReader{plan: domain.PlanPro},
		"/upgrade",
	)

	result, err := enforcer.CheckSchema(context.Background(), "tenant-1")
	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestQuotaEnforcer_Storage(t *testing.T) {
	enforcer := NewQuotaEnforcer(
		&mockUsageReader{usage: &domain.TenantUsage{StorageBytes: 90 * 1024 * 1024}},
		&mockPlanReader{plan: domain.PlanFree},
		"/upgrade",
	)

	// Adding 20MB would exceed 100MB limit
	result, err := enforcer.CheckStorage(context.Background(), "tenant-1", 20*1024*1024)
	require.NoError(t, err)
	assert.False(t, result.Allowed)

	// Adding 5MB is fine
	result, err = enforcer.CheckStorage(context.Background(), "tenant-1", 5*1024*1024)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestQuotaEnforcer_UnlimitedPlan(t *testing.T) {
	// Business plan: unlimited schemas
	enforcer := NewQuotaEnforcer(
		&mockUsageReader{usage: &domain.TenantUsage{Schemas: 100}},
		&mockPlanReader{plan: domain.PlanBusiness},
		"/upgrade",
	)

	result, err := enforcer.CheckSchema(context.Background(), "tenant-1")
	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestQuotaEnforcer_UsageSummary(t *testing.T) {
	// AC-PRICE-08: usage dashboard
	enforcer := NewQuotaEnforcer(
		&mockUsageReader{usage: &domain.TenantUsage{
			APICalls:     850,
			StorageBytes: 50 * 1024 * 1024,
			Schemas:      1,
		}},
		&mockPlanReader{plan: domain.PlanFree},
		"/upgrade",
	)

	summary, err := enforcer.GetUsageSummary(context.Background(), "tenant-1")
	require.NoError(t, err)
	assert.Equal(t, domain.PlanFree, summary.Plan)
	assert.Len(t, summary.Metrics, 3)

	// API calls at 85% → warning
	apiMetric := summary.Metrics[0]
	assert.Equal(t, "api_calls", apiMetric.Resource)
	assert.Equal(t, "warning", apiMetric.Warning)
}

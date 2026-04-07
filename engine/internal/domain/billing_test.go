package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloudPlan_IsValid(t *testing.T) {
	assert.True(t, PlanFree.IsValid())
	assert.True(t, PlanPro.IsValid())
	assert.True(t, PlanBusiness.IsValid())
	assert.True(t, PlanEnterprise.IsValid())
	assert.False(t, CloudPlan("unknown").IsValid())
}

func TestCloudPlan_StripeProductID(t *testing.T) {
	assert.Equal(t, "bytebrew_cloud_free", PlanFree.StripeProductID())
	assert.Equal(t, "bytebrew_cloud_pro", PlanPro.StripeProductID())
	assert.Equal(t, "bytebrew_cloud_business", PlanBusiness.StripeProductID())
}

func TestGetPlanLimits_Free(t *testing.T) {
	// AC-PRICE-01: Free limits
	limits := GetPlanLimits(PlanFree)
	assert.Equal(t, 1, limits.MaxSchemas)
	assert.Equal(t, 10, limits.MaxAgentsPerSchema)
	assert.Equal(t, 1000, limits.MaxAPICalls)
	assert.Equal(t, int64(100*1024*1024), limits.MaxStorageBytes)
	assert.False(t, limits.ForwardHeaders)
	assert.False(t, limits.AllMCP) // verified only
	assert.Equal(t, 100, limits.DefaultModelReqs)
}

func TestGetPlanLimits_Pro(t *testing.T) {
	limits := GetPlanLimits(PlanPro)
	assert.Equal(t, 5, limits.MaxSchemas)
	assert.Equal(t, 0, limits.MaxAgentsPerSchema) // unlimited
	assert.Equal(t, 50000, limits.MaxAPICalls)
	assert.True(t, limits.ForwardHeaders)
	assert.True(t, limits.AllMCP)
}

func TestGetPlanLimits_Business(t *testing.T) {
	limits := GetPlanLimits(PlanBusiness)
	assert.Equal(t, 0, limits.MaxSchemas) // unlimited
	assert.Equal(t, 500000, limits.MaxAPICalls)
	assert.True(t, limits.OPSMode)
}

func TestQuotaCheckResult_WarningLevel(t *testing.T) {
	tests := []struct {
		pct   float64
		level string
	}{
		{50, "ok"},
		{79, "ok"},
		{80, "warning"},
		{94, "warning"},
		{95, "critical"},
		{99, "critical"},
		{100, "blocked"},
		{150, "blocked"},
	}
	for _, tt := range tests {
		r := &QuotaCheckResult{PercentUsed: tt.pct}
		assert.Equal(t, tt.level, r.WarningLevel(), "pct=%.0f", tt.pct)
	}
}

func TestNewTrial(t *testing.T) {
	// AC-PRICE-04: 14-day Pro trial
	trial := NewTrial()
	assert.True(t, trial.Active)
	assert.False(t, trial.IsExpired())
	assert.WithinDuration(t, time.Now().Add(14*24*time.Hour), trial.EndDate, time.Minute)
	require.NoError(t, trial.Validate())
}

func TestTrialInfo_IsExpired(t *testing.T) {
	// AC-PRICE-05: expired trial
	trial := &TrialInfo{
		Active:    true,
		StartDate: time.Now().Add(-15 * 24 * time.Hour),
		EndDate:   time.Now().Add(-1 * 24 * time.Hour),
	}
	assert.True(t, trial.IsExpired())
}

package billing

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// UsageReader reads current tenant usage.
type UsageReader interface {
	GetUsage(ctx context.Context, tenantID string) (*domain.TenantUsage, error)
}

// PlanReader reads the current plan for a tenant.
type PlanReader interface {
	GetPlan(ctx context.Context, tenantID string) (domain.CloudPlan, error)
}

// QuotaEnforcer checks resource quotas before operations (AC-PRICE-01, AC-PRICE-02).
type QuotaEnforcer struct {
	usage      UsageReader
	plans      PlanReader
	upgradeURL string
}

// NewQuotaEnforcer creates a new quota enforcer.
func NewQuotaEnforcer(usage UsageReader, plans PlanReader, upgradeURL string) *QuotaEnforcer {
	return &QuotaEnforcer{
		usage:      usage,
		plans:      plans,
		upgradeURL: upgradeURL,
	}
}

// CheckAPICall checks if a tenant can make an API call.
func (q *QuotaEnforcer) CheckAPICall(ctx context.Context, tenantID string) (*domain.QuotaCheckResult, error) {
	_, usage, limits, err := q.loadContext(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if limits.MaxAPICalls == 0 {
		return &domain.QuotaCheckResult{Allowed: true, Resource: "api_calls"}, nil
	}

	pct := float64(usage.APICalls) / float64(limits.MaxAPICalls) * 100
	result := &domain.QuotaCheckResult{
		Resource:    "api_calls",
		Used:        int64(usage.APICalls),
		Limit:       int64(limits.MaxAPICalls),
		PercentUsed: pct,
		UpgradeURL:  q.upgradeURL,
		Allowed:     usage.APICalls < limits.MaxAPICalls,
	}

	if !result.Allowed {
		slog.WarnContext(ctx, "[Quota] API call limit reached",
			"tenant", tenantID, "used", usage.APICalls, "limit", limits.MaxAPICalls)
	}
	return result, nil
}

// CheckSchema checks if a tenant can create another schema.
func (q *QuotaEnforcer) CheckSchema(ctx context.Context, tenantID string) (*domain.QuotaCheckResult, error) {
	_, usage, limits, err := q.loadContext(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if limits.MaxSchemas == 0 {
		return &domain.QuotaCheckResult{Allowed: true, Resource: "schemas"}, nil
	}

	pct := float64(usage.Schemas) / float64(limits.MaxSchemas) * 100
	return &domain.QuotaCheckResult{
		Resource:    "schemas",
		Used:        int64(usage.Schemas),
		Limit:       int64(limits.MaxSchemas),
		PercentUsed: pct,
		UpgradeURL:  q.upgradeURL,
		Allowed:     usage.Schemas < limits.MaxSchemas,
	}, nil
}

// CheckStorage checks if a tenant has storage capacity.
func (q *QuotaEnforcer) CheckStorage(ctx context.Context, tenantID string, additionalBytes int64) (*domain.QuotaCheckResult, error) {
	_, usage, limits, err := q.loadContext(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if limits.MaxStorageBytes == 0 {
		return &domain.QuotaCheckResult{Allowed: true, Resource: "storage"}, nil
	}

	totalAfter := usage.StorageBytes + additionalBytes
	pct := float64(totalAfter) / float64(limits.MaxStorageBytes) * 100
	return &domain.QuotaCheckResult{
		Resource:    "storage",
		Used:        totalAfter,
		Limit:       limits.MaxStorageBytes,
		PercentUsed: pct,
		UpgradeURL:  q.upgradeURL,
		Allowed:     totalAfter <= limits.MaxStorageBytes,
	}, nil
}

// GetUsageSummary returns a full usage summary for the usage dashboard (AC-PRICE-08).
func (q *QuotaEnforcer) GetUsageSummary(ctx context.Context, tenantID string) (*UsageSummary, error) {
	plan, usage, limits, err := q.loadContext(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return &UsageSummary{
		Plan:   plan,
		Limits: limits,
		Usage:  *usage,
		Metrics: []UsageMetric{
			q.buildMetric("api_calls", int64(usage.APICalls), int64(limits.MaxAPICalls)),
			q.buildMetric("storage", usage.StorageBytes, limits.MaxStorageBytes),
			q.buildMetric("schemas", int64(usage.Schemas), int64(limits.MaxSchemas)),
		},
	}, nil
}

func (q *QuotaEnforcer) loadContext(ctx context.Context, tenantID string) (domain.CloudPlan, *domain.TenantUsage, domain.PlanLimits, error) {
	plan, err := q.plans.GetPlan(ctx, tenantID)
	if err != nil {
		return "", nil, domain.PlanLimits{}, fmt.Errorf("get plan: %w", err)
	}

	usage, err := q.usage.GetUsage(ctx, tenantID)
	if err != nil {
		return "", nil, domain.PlanLimits{}, fmt.Errorf("get usage: %w", err)
	}

	limits := domain.GetPlanLimits(plan)
	return plan, usage, limits, nil
}

func (q *QuotaEnforcer) buildMetric(resource string, used, limit int64) UsageMetric {
	pct := float64(0)
	if limit > 0 {
		pct = float64(used) / float64(limit) * 100
	}
	warning := "ok"
	switch {
	case limit > 0 && pct >= 100:
		warning = "blocked"
	case limit > 0 && pct >= 95:
		warning = "critical"
	case limit > 0 && pct >= 80:
		warning = "warning"
	}
	return UsageMetric{
		Resource:    resource,
		Used:        used,
		Limit:       limit,
		PercentUsed: pct,
		Warning:     warning,
	}
}

// UsageSummary is the response for GET /api/v1/usage.
type UsageSummary struct {
	Plan    domain.CloudPlan   `json:"plan"`
	Limits  domain.PlanLimits  `json:"limits"`
	Usage   domain.TenantUsage `json:"usage"`
	Metrics []UsageMetric      `json:"metrics"`
}

// UsageMetric represents a single usage metric with warning level.
type UsageMetric struct {
	Resource    string  `json:"resource"`
	Used        int64   `json:"used"`
	Limit       int64   `json:"limit"`
	PercentUsed float64 `json:"percent_used"`
	Warning     string  `json:"warning"` // ok, warning, critical, blocked
}

package domain

import (
	"fmt"
	"time"
)

// CloudPlan represents a Cloud subscription plan.
type CloudPlan string

const (
	PlanFree       CloudPlan = "free"
	PlanPro        CloudPlan = "pro"
	PlanBusiness   CloudPlan = "business"
	PlanEnterprise CloudPlan = "enterprise"
)

// IsValid returns true if the plan is recognized.
func (p CloudPlan) IsValid() bool {
	switch p {
	case PlanFree, PlanPro, PlanBusiness, PlanEnterprise:
		return true
	}
	return false
}

// StripeProductID returns the Stripe product identifier for this plan.
func (p CloudPlan) StripeProductID() string {
	switch p {
	case PlanFree:
		return "bytebrew_cloud_free"
	case PlanPro:
		return "bytebrew_cloud_pro"
	case PlanBusiness:
		return "bytebrew_cloud_business"
	default:
		return ""
	}
}

// PlanLimits defines the resource limits for a Cloud plan.
type PlanLimits struct {
	MaxSchemas        int   // 0 = unlimited
	MaxAgentsPerSchema int   // 0 = unlimited
	MaxAPICalls       int   // per month, 0 = unlimited
	MaxStorageBytes   int64 // 0 = unlimited
	MaxWidgets        int   // 0 = unlimited
	MaxTeamMembers    int   // 0 = unlimited
	ForwardHeaders    bool
	OPSMode           bool
	AllMCP            bool  // false = verified only
	DefaultModelReqs  int   // GLM 4.7 free requests/month
}

// GetPlanLimits returns the limits for a given plan (AC-PRICE-01).
func GetPlanLimits(plan CloudPlan) PlanLimits {
	switch plan {
	case PlanFree:
		return PlanLimits{
			MaxSchemas:        1,
			MaxAgentsPerSchema: 10,
			MaxAPICalls:       1000,
			MaxStorageBytes:   100 * 1024 * 1024, // 100 MB
			MaxWidgets:        1,
			MaxTeamMembers:    1,
			ForwardHeaders:    false,
			OPSMode:           false,
			AllMCP:            false,
			DefaultModelReqs:  100,
		}
	case PlanPro:
		return PlanLimits{
			MaxSchemas:        5,
			MaxAgentsPerSchema: 0, // unlimited
			MaxAPICalls:       50000,
			MaxStorageBytes:   5 * 1024 * 1024 * 1024, // 5 GB
			MaxWidgets:        3,
			MaxTeamMembers:    3,
			ForwardHeaders:    true,
			OPSMode:           false,
			AllMCP:            true,
			DefaultModelReqs:  100,
		}
	case PlanBusiness:
		return PlanLimits{
			MaxSchemas:        0, // unlimited
			MaxAgentsPerSchema: 0,
			MaxAPICalls:       500000,
			MaxStorageBytes:   50 * 1024 * 1024 * 1024, // 50 GB
			MaxWidgets:        0,
			MaxTeamMembers:    10,
			ForwardHeaders:    true,
			OPSMode:           true,
			AllMCP:            true,
			DefaultModelReqs:  100,
		}
	case PlanEnterprise:
		return PlanLimits{
			MaxSchemas:        0,
			MaxAgentsPerSchema: 0,
			MaxAPICalls:       0, // custom
			MaxStorageBytes:   0,
			MaxWidgets:        0,
			MaxTeamMembers:    0,
			ForwardHeaders:    true,
			OPSMode:           true,
			AllMCP:            true,
			DefaultModelReqs:  100,
		}
	default:
		return GetPlanLimits(PlanFree)
	}
}

// TenantUsage tracks current resource usage for a tenant.
type TenantUsage struct {
	TenantID     string
	APICalls     int   // current month
	StorageBytes int64
	Schemas      int
	BillingStart time.Time // start of current billing cycle
}

// QuotaCheckResult holds the result of a quota check.
type QuotaCheckResult struct {
	Allowed     bool
	Resource    string  // "api_calls", "schemas", "storage", etc.
	Used        int64
	Limit       int64
	PercentUsed float64
	UpgradeURL  string
}

// WarningLevel returns the warning level based on percentage used.
func (r *QuotaCheckResult) WarningLevel() string {
	switch {
	case r.PercentUsed >= 100:
		return "blocked"
	case r.PercentUsed >= 95:
		return "critical"
	case r.PercentUsed >= 80:
		return "warning"
	default:
		return "ok"
	}
}

// TrialInfo tracks trial status for a tenant.
type TrialInfo struct {
	Active    bool
	StartDate time.Time
	EndDate   time.Time // 14 days from start
}

// NewTrial creates a new 14-day Pro trial (AC-PRICE-04).
func NewTrial() *TrialInfo {
	now := time.Now()
	return &TrialInfo{
		Active:    true,
		StartDate: now,
		EndDate:   now.Add(14 * 24 * time.Hour),
	}
}

// IsExpired returns true if the trial has expired (AC-PRICE-05).
func (t *TrialInfo) IsExpired() bool {
	return time.Now().After(t.EndDate)
}

// Validate validates the TrialInfo.
func (t *TrialInfo) Validate() error {
	if t.StartDate.IsZero() {
		return fmt.Errorf("trial start_date is required")
	}
	if t.EndDate.IsZero() {
		return fmt.Errorf("trial end_date is required")
	}
	if t.EndDate.Before(t.StartDate) {
		return fmt.Errorf("trial end_date must be after start_date")
	}
	return nil
}

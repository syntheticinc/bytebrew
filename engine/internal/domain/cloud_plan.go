package domain

// CloudPlan represents a subscription plan tier for Cloud deployments.
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

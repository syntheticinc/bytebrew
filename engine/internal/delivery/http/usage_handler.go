package http

import (
	"net/http"
	"time"
)

type usageMetric struct {
	Name  string  `json:"name"`
	Label string  `json:"label"`
	Used  float64 `json:"used"`
	Limit float64 `json:"limit"`
	Unit  string  `json:"unit"`
}

type usageResponse struct {
	Plan              string        `json:"plan"`
	BillingCycleStart string        `json:"billing_cycle_start"`
	BillingCycleEnd   string        `json:"billing_cycle_end"`
	Metrics           []usageMetric `json:"metrics"`
	StripePortalURL   string        `json:"stripe_portal_url,omitempty"`
}

// UsageHandler serves GET /api/v1/usage with quota/billing usage data.
type UsageHandler struct{}

func NewUsageHandler() *UsageHandler {
	return &UsageHandler{}
}

// GetUsage handles GET /api/v1/usage.
// In CE mode (no billing), returns unlimited Community Edition values.
func (h *UsageHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	cycleStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	cycleEnd := cycleStart.AddDate(0, 1, 0)

	writeJSON(w, http.StatusOK, usageResponse{
		Plan:              "Community Edition",
		BillingCycleStart: cycleStart.Format(time.RFC3339),
		BillingCycleEnd:   cycleEnd.Format(time.RFC3339),
		Metrics: []usageMetric{
			{Name: "agents", Label: "Agents", Used: 0, Limit: -1, Unit: ""},
			{Name: "schemas", Label: "Schemas", Used: 0, Limit: -1, Unit: ""},
			{Name: "sessions", Label: "Sessions", Used: 0, Limit: -1, Unit: ""},
		},
	})
}

package http

import (
	"log/slog"
	"net/http"
	"time"

	"gorm.io/gorm"
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
type UsageHandler struct {
	db *gorm.DB
}

func NewUsageHandler(db *gorm.DB) *UsageHandler {
	return &UsageHandler{db: db}
}

// GetUsage handles GET /api/v1/usage.
// In CE mode (no billing), returns unlimited Community Edition values with real counters.
func (h *UsageHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := time.Now().UTC()
	cycleStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	cycleEnd := cycleStart.AddDate(0, 1, 0)

	var agentCount, schemaCount int64
	if err := h.db.WithContext(ctx).Raw("SELECT COUNT(*) FROM agents").Scan(&agentCount).Error; err != nil {
		slog.ErrorContext(ctx, "usage: failed to count agents", "error", err)
	}
	if err := h.db.WithContext(ctx).Raw("SELECT COUNT(*) FROM schemas").Scan(&schemaCount).Error; err != nil {
		slog.ErrorContext(ctx, "usage: failed to count schemas", "error", err)
	}

	var sessionCount int64
	if err := h.db.WithContext(ctx).Raw(
		"SELECT COUNT(DISTINCT session_id) FROM messages WHERE created_at >= ?", cycleStart,
	).Scan(&sessionCount).Error; err != nil {
		slog.ErrorContext(ctx, "usage: failed to count sessions", "error", err)
	}

	writeJSON(w, http.StatusOK, usageResponse{
		Plan:              "Community Edition",
		BillingCycleStart: cycleStart.Format(time.RFC3339),
		BillingCycleEnd:   cycleEnd.Format(time.RFC3339),
		Metrics: []usageMetric{
			{Name: "agents", Label: "Agents", Used: float64(agentCount), Limit: -1, Unit: ""},
			{Name: "schemas", Label: "Schemas", Used: float64(schemaCount), Limit: -1, Unit: ""},
			{Name: "sessions", Label: "Sessions", Used: float64(sessionCount), Limit: -1, Unit: ""},
		},
	})
}

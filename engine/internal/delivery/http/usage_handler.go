package http

import (
	"net/http"
)

// UsageHandler serves GET /api/v1/usage with quota/billing usage data.
type UsageHandler struct{}

// NewUsageHandler creates a new UsageHandler.
func NewUsageHandler() *UsageHandler {
	return &UsageHandler{}
}

// usageResponse represents usage/quota data for the frontend.
type usageResponse struct {
	Plan          string `json:"plan"`
	MessagesUsed  int    `json:"messages_used"`
	MessagesLimit int    `json:"messages_limit"`
	Unlimited     bool   `json:"unlimited"`
}

// GetUsage handles GET /api/v1/usage.
// In CE mode (no billing), returns unlimited values.
func (h *UsageHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, usageResponse{
		Plan:          "community",
		MessagesUsed:  0,
		MessagesLimit: 0,
		Unlimited:     true,
	})
}

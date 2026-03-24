package http

import (
	"net/http"
	"time"
)

// RateLimitUsageProvider provides rate limit usage data.
type RateLimitUsageProvider interface {
	Rules() []RateLimitRule
	Usage(ruleName, keyValue string, now time.Time) (used int, limit int, window time.Duration, resetAt time.Time, tierName string, found bool)
}

// RateLimitUsageHandler serves GET /api/v1/rate-limits/usage.
type RateLimitUsageHandler struct {
	provider RateLimitUsageProvider
}

// NewRateLimitUsageHandler creates a new RateLimitUsageHandler.
func NewRateLimitUsageHandler(provider RateLimitUsageProvider) *RateLimitUsageHandler {
	return &RateLimitUsageHandler{provider: provider}
}

type rateLimitUsageResponse struct {
	Rule     string `json:"rule"`
	Key      string `json:"key"`
	Tier     string `json:"tier"`
	Used     int    `json:"used"`
	Limit    int    `json:"limit"`
	Window   string `json:"window"`
	ResetsAt string `json:"resets_at"`
}

// Usage handles GET /api/v1/rate-limits/usage?key_header=X-Org-Id&key_value=org-123
func (h *RateLimitUsageHandler) Usage(w http.ResponseWriter, r *http.Request) {
	keyHeader := r.URL.Query().Get("key_header")
	keyValue := r.URL.Query().Get("key_value")

	if keyHeader == "" || keyValue == "" {
		writeJSONError(w, http.StatusBadRequest, "key_header and key_value query parameters are required")
		return
	}

	// Find rule that matches the key_header
	var matchedRuleName string
	for _, rule := range h.provider.Rules() {
		if rule.KeyHeader == keyHeader {
			matchedRuleName = rule.Name
			break
		}
	}
	if matchedRuleName == "" {
		writeJSONError(w, http.StatusNotFound, "no rate limit rule found for header: "+keyHeader)
		return
	}

	now := time.Now()
	used, limit, windowDur, resetAt, tierName, found := h.provider.Usage(matchedRuleName, keyValue, now)
	if !found {
		writeJSONError(w, http.StatusNotFound, "rate limit rule not found: "+matchedRuleName)
		return
	}

	resp := rateLimitUsageResponse{
		Rule:     matchedRuleName,
		Key:      keyValue,
		Tier:     tierName,
		Used:     used,
		Limit:    limit,
		Window:   windowDur.String(),
		ResetsAt: resetAt.Format(time.RFC3339),
	}
	writeJSON(w, http.StatusOK, resp)
}

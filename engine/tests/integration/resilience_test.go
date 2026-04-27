//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// resilienceEndpoints is the canonical list of read endpoints. Only
// circuit-breakers is registered in V2 — dead-letter / heartbeats /
// stuck-agents were removed.
var resilienceEndpoints = []string{
	"/api/v1/resilience/circuit-breakers",
}

// TC-RES-01: Circuit breakers list → 200 JSON.
func TestRES01_CircuitBreakers(t *testing.T) {
	requireSuite(t)

	resp := do(t, http.MethodGet, "/api/v1/resilience/circuit-breakers", nil, adminToken)
	body := readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "body=%s", body)
	if len(body) > 0 {
		first := body[0]
		assert.True(t, first == '[' || first == '{',
			"circuit-breakers response should be JSON: %s", body)
	}
}

// TC-RES-05: Reset an unknown circuit breaker → 200 (idempotent) or 404.
func TestRES05_ResetBreaker(t *testing.T) {
	requireSuite(t)

	resp := do(t, http.MethodPost, "/api/v1/resilience/circuit-breakers/nonexistent/reset",
		nil, adminToken)
	_ = readBody(t, resp)
	assertStatusAny(t, resp,
		http.StatusOK, http.StatusNoContent, http.StatusNotFound, http.StatusAccepted)
}

// TC-RES-06: All resilience endpoints require auth.
func TestRES06_RequireAuth(t *testing.T) {
	requireSuite(t)

	for _, path := range resilienceEndpoints {
		resp := do(t, http.MethodGet, path, nil, "")
		_ = readBody(t, resp)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
			"%s without token must be 401", path)
	}
}

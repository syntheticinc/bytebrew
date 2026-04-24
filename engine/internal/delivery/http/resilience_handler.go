package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// CircuitBreakerState is the wire format for a single circuit breaker.
type CircuitBreakerState struct {
	Name         string     `json:"name"`
	State        string     `json:"state"` // closed | open | half_open
	FailureCount int        `json:"failure_count"`
	LastFailure  *time.Time `json:"last_failure,omitempty"`
}

// CircuitBreakerQuerier provides read access to circuit breaker states.
type CircuitBreakerQuerier interface {
	Snapshots() []CircuitBreakerState
	Reset(name string) bool
}

// ResilienceHandler serves admin resilience endpoints.
//
// Heartbeat monitoring and a dead-letter queue were planned (AC-RESIL-01..04,
// 07..08) but deferred — the producers were never wired to the runtime, so the
// former endpoints returned only empty lists. They have been removed; the
// circuit breaker path is the only resilience surface actually backed by live
// data (AC-RESIL-05/06/09..12).
type ResilienceHandler struct {
	circuitBreakers CircuitBreakerQuerier
}

// NewResilienceHandler creates a ResilienceHandler.
// cb may be nil — endpoints will return empty lists / 404 accordingly.
func NewResilienceHandler(cb CircuitBreakerQuerier) *ResilienceHandler {
	return &ResilienceHandler{circuitBreakers: cb}
}

// ListCircuitBreakers handles GET /api/v1/resilience/circuit-breakers.
func (h *ResilienceHandler) ListCircuitBreakers(w http.ResponseWriter, r *http.Request) {
	if h.circuitBreakers == nil {
		writeJSON(w, http.StatusOK, map[string]any{"breakers": []CircuitBreakerState{}})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"breakers": h.circuitBreakers.Snapshots()})
}

// ResetCircuitBreaker handles POST /api/v1/resilience/circuit-breakers/{name}/reset.
func (h *ResilienceHandler) ResetCircuitBreaker(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "circuit breaker name required")
		return
	}

	if h.circuitBreakers == nil {
		writeJSONError(w, http.StatusNotFound, "circuit breaker not found: "+name)
		return
	}

	if !h.circuitBreakers.Reset(name) {
		writeJSONError(w, http.StatusNotFound, "circuit breaker not found: "+name)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "reset", "name": name})
}

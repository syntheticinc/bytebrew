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

// DeadLetterEntry is the wire format for a dead-letter task.
type DeadLetterEntry struct {
	TaskID    string    `json:"task_id"`
	AgentID   string    `json:"agent_id"`
	StartedAt time.Time `json:"started_at"`
	Status    string    `json:"status"`
}

// HeartbeatEntry is the wire format for a monitored agent's heartbeat.
type HeartbeatEntry struct {
	AgentID       string    `json:"agent_id"`
	AgentType     string    `json:"agent_type"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	CurrentStep   string    `json:"current_step,omitempty"`
}

// CircuitBreakerQuerier provides read access to circuit breaker states.
type CircuitBreakerQuerier interface {
	Snapshots() []CircuitBreakerState
	Reset(name string) bool
}

// DeadLetterQuerier provides read access to dead-letter tasks.
type DeadLetterQuerier interface {
	DeadLetters() []DeadLetterEntry
}

// HeartbeatQuerier provides read access to heartbeat snapshots.
type HeartbeatQuerier interface {
	Snapshots() []HeartbeatEntry
}

// ResilienceHandler serves admin resilience endpoints.
type ResilienceHandler struct {
	circuitBreakers CircuitBreakerQuerier
	deadLetters     DeadLetterQuerier
	heartbeats      HeartbeatQuerier
}

// NewResilienceHandler creates a ResilienceHandler.
// Any querier may be nil — the corresponding endpoint will return an empty list.
func NewResilienceHandler(cb CircuitBreakerQuerier, dl DeadLetterQuerier, hb HeartbeatQuerier) *ResilienceHandler {
	return &ResilienceHandler{
		circuitBreakers: cb,
		deadLetters:     dl,
		heartbeats:      hb,
	}
}

// ListCircuitBreakers handles GET /api/v1/admin/resilience/circuit-breakers.
func (h *ResilienceHandler) ListCircuitBreakers(w http.ResponseWriter, r *http.Request) {
	if h.circuitBreakers == nil {
		writeJSON(w, http.StatusOK, []CircuitBreakerState{})
		return
	}

	writeJSON(w, http.StatusOK, h.circuitBreakers.Snapshots())
}

// ResetCircuitBreaker handles POST /api/v1/admin/resilience/circuit-breakers/{name}/reset.
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

// ListDeadLetters handles GET /api/v1/admin/resilience/dead-letters.
func (h *ResilienceHandler) ListDeadLetters(w http.ResponseWriter, r *http.Request) {
	if h.deadLetters == nil {
		writeJSON(w, http.StatusOK, []DeadLetterEntry{})
		return
	}

	writeJSON(w, http.StatusOK, h.deadLetters.DeadLetters())
}

// ListHeartbeats handles GET /api/v1/admin/resilience/heartbeats.
func (h *ResilienceHandler) ListHeartbeats(w http.ResponseWriter, r *http.Request) {
	if h.heartbeats == nil {
		writeJSON(w, http.StatusOK, []HeartbeatEntry{})
		return
	}

	writeJSON(w, http.StatusOK, h.heartbeats.Snapshots())
}

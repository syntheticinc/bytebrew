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
//
// The extended fields (Reason, MovedAt, ElapsedMs, LastError, AgentName) are
// populated by the resilience service when a task is moved to dead-letter
// status. They may be empty for entries that have not yet been fully
// annotated (e.g. while the task is still running), so consumers must treat
// them as optional.
type DeadLetterEntry struct {
	TaskID    string    `json:"task_id"`
	AgentID   string    `json:"agent_id"`
	AgentName string    `json:"agent_name,omitempty"`
	StartedAt time.Time `json:"started_at"`
	MovedAt   time.Time `json:"moved_at,omitempty"`
	Status    string    `json:"status"`
	Reason    string    `json:"reason,omitempty"`
	ElapsedMs int64     `json:"elapsed_ms,omitempty"`
	LastError string    `json:"last_error,omitempty"`
}

// HeartbeatEntry is the wire format for a monitored agent's heartbeat.
type HeartbeatEntry struct {
	AgentID       string    `json:"agent_id"`
	AgentType     string    `json:"agent_type"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	CurrentStep   string    `json:"current_step,omitempty"`
}

// StuckAgentEntry is the wire format for an agent that has missed its
// heartbeat for longer than the stuck threshold.
type StuckAgentEntry struct {
	AgentID       string    `json:"agent_id"`
	AgentName     string    `json:"agent_name,omitempty"`
	AgentType     string    `json:"agent_type"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	ElapsedMs     int64     `json:"elapsed_ms"`
	Status        string    `json:"status"`
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

// HeartbeatQuerier provides read access to heartbeat snapshots and the
// filtered subset of agents believed to be stuck (missed 2 × interval).
type HeartbeatQuerier interface {
	Snapshots() []HeartbeatEntry
	StuckSnapshots() []StuckAgentEntry
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

// ListDeadLetters handles GET /api/v1/resilience/dead-letter.
func (h *ResilienceHandler) ListDeadLetters(w http.ResponseWriter, r *http.Request) {
	if h.deadLetters == nil {
		writeJSON(w, http.StatusOK, map[string]any{"tasks": []DeadLetterEntry{}})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"tasks": h.deadLetters.DeadLetters()})
}

// ListHeartbeats handles GET /api/v1/resilience/heartbeats — returns ALL
// monitored agents (intended for debugging / admin dashboards). The user-
// facing observability page uses ListStuckAgents instead.
func (h *ResilienceHandler) ListHeartbeats(w http.ResponseWriter, r *http.Request) {
	if h.heartbeats == nil {
		writeJSON(w, http.StatusOK, []HeartbeatEntry{})
		return
	}

	writeJSON(w, http.StatusOK, h.heartbeats.Snapshots())
}

// ListStuckAgents handles GET /api/v1/resilience/stuck-agents — returns only
// agents whose last heartbeat is older than the stuck threshold.
func (h *ResilienceHandler) ListStuckAgents(w http.ResponseWriter, r *http.Request) {
	if h.heartbeats == nil {
		writeJSON(w, http.StatusOK, map[string]any{"agents": []StuckAgentEntry{}})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"agents": h.heartbeats.StuckSnapshots()})
}

package app

import (
	"time"

	deliveryhttp "github.com/syntheticinc/bytebrew/engine/internal/delivery/http"
	"github.com/syntheticinc/bytebrew/engine/internal/service/resilience"
)

// circuitBreakerQuerierHTTPAdapter adapts CircuitBreakerRegistry to the
// deliveryhttp.CircuitBreakerQuerier interface (consumer-side).
type circuitBreakerQuerierHTTPAdapter struct {
	registry *resilience.CircuitBreakerRegistry
}

func (a *circuitBreakerQuerierHTTPAdapter) Snapshots() []deliveryhttp.CircuitBreakerState {
	raw := a.registry.Snapshots()
	out := make([]deliveryhttp.CircuitBreakerState, len(raw))
	for i, s := range raw {
		var lastFailure *time.Time
		if !s.LastFailure.IsZero() {
			t := s.LastFailure
			lastFailure = &t
		}
		out[i] = deliveryhttp.CircuitBreakerState{
			Name:         s.Name,
			State:        string(s.State),
			FailureCount: s.FailureCount,
			LastFailure:  lastFailure,
		}
	}
	return out
}

func (a *circuitBreakerQuerierHTTPAdapter) Reset(name string) bool {
	return a.registry.Reset(name)
}

// deadLetterQuerierHTTPAdapter adapts DeadLetterQueue to the
// deliveryhttp.DeadLetterQuerier interface (consumer-side).
type deadLetterQuerierHTTPAdapter struct {
	queue *resilience.DeadLetterQueue
}

func (a *deadLetterQuerierHTTPAdapter) DeadLetters() []deliveryhttp.DeadLetterEntry {
	raw := a.queue.DeadLetters()
	out := make([]deliveryhttp.DeadLetterEntry, len(raw))
	for i, t := range raw {
		out[i] = deliveryhttp.DeadLetterEntry{
			TaskID:    t.TaskID,
			AgentID:   t.AgentID,
			StartedAt: t.StartedAt,
			Status:    string(t.Status),
		}
	}
	return out
}

// heartbeatQuerierHTTPAdapter adapts HeartbeatMonitor to the
// deliveryhttp.HeartbeatQuerier interface (consumer-side).
type heartbeatQuerierHTTPAdapter struct {
	monitor *resilience.HeartbeatMonitor
}

func (a *heartbeatQuerierHTTPAdapter) Snapshots() []deliveryhttp.HeartbeatEntry {
	raw := a.monitor.Snapshots()
	out := make([]deliveryhttp.HeartbeatEntry, len(raw))
	for i, s := range raw {
		out[i] = deliveryhttp.HeartbeatEntry{
			AgentID:       s.AgentID,
			AgentType:     string(s.AgentType),
			LastHeartbeat: s.LastHeartbeat,
			CurrentStep:   s.CurrentStep,
		}
	}
	return out
}

// Compile-time interface checks.
var (
	_ deliveryhttp.CircuitBreakerQuerier = (*circuitBreakerQuerierHTTPAdapter)(nil)
	_ deliveryhttp.DeadLetterQuerier     = (*deadLetterQuerierHTTPAdapter)(nil)
	_ deliveryhttp.HeartbeatQuerier      = (*heartbeatQuerierHTTPAdapter)(nil)
)

// stubHeartbeatCallback is a default StuckCallback that logs via slog.
func stubHeartbeatCallback(agentID string, agentType resilience.AgentType, lastHeartbeat time.Time) {
	// Logging is already done inside HeartbeatMonitor.CheckStuck().
	// This callback is reserved for future recovery actions (respawn, escalate).
}

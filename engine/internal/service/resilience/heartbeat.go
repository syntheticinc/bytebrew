package resilience

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// HeartbeatConfig holds configuration for the heartbeat monitor.
type HeartbeatConfig struct {
	Interval time.Duration // heartbeat interval (default 15s)
	// Stuck = 2 × Interval without heartbeat
}

// DefaultHeartbeatConfig returns default heartbeat configuration.
func DefaultHeartbeatConfig() HeartbeatConfig {
	return HeartbeatConfig{
		Interval: 15 * time.Second,
	}
}

// AgentType represents whether an agent is spawn or persistent.
type AgentType string

const (
	AgentTypeSpawn      AgentType = "spawn"
	AgentTypePersistent AgentType = "persistent"
)

// StuckAction is the action to take when an agent is stuck.
type StuckAction string

const (
	StuckActionRespawn  StuckAction = "respawn"           // spawn: kill + re-spawn
	StuckActionKill     StuckAction = "force_kill"        // persistent: force kill after grace
	StuckActionEscalate StuckAction = "escalate_to_parent" // persistent: escalate
)

// HeartbeatEvent represents a heartbeat received from an agent.
type HeartbeatEvent struct {
	AgentID     string
	Timestamp   time.Time
	CurrentStep string
}

// StuckCallback is called when an agent is detected as stuck.
type StuckCallback func(agentID string, agentType AgentType, lastHeartbeat time.Time)

// agentEntry tracks heartbeat state for a single agent.
type agentEntry struct {
	agentType     AgentType
	lastHeartbeat time.Time
	currentStep   string
}

// HeartbeatMonitor tracks agent heartbeats and detects stuck agents.
// AC-RESIL-01: agents send heartbeats at interval
// AC-RESIL-02: parent notified when agent stuck (2× miss)
type HeartbeatMonitor struct {
	mu       sync.RWMutex
	agents   map[string]*agentEntry
	config   HeartbeatConfig
	callback StuckCallback
	cancel   context.CancelFunc
}

// NewHeartbeatMonitor creates a new heartbeat monitor.
func NewHeartbeatMonitor(config HeartbeatConfig, callback StuckCallback) *HeartbeatMonitor {
	return &HeartbeatMonitor{
		agents:   make(map[string]*agentEntry),
		config:   config,
		callback: callback,
	}
}

// Register registers an agent for heartbeat monitoring.
func (m *HeartbeatMonitor) Register(agentID string, agentType AgentType) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.agents[agentID] = &agentEntry{
		agentType:     agentType,
		lastHeartbeat: time.Now(),
	}
	slog.Info("[Heartbeat] agent registered", "agent_id", agentID, "type", agentType)
}

// Unregister removes an agent from monitoring.
func (m *HeartbeatMonitor) Unregister(agentID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.agents, agentID)
}

// RecordHeartbeat records a heartbeat from an agent (AC-RESIL-01).
func (m *HeartbeatMonitor) RecordHeartbeat(event HeartbeatEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.agents[event.AgentID]
	if !ok {
		return
	}
	entry.lastHeartbeat = event.Timestamp
	entry.currentStep = event.CurrentStep
}

// CheckStuck checks all agents for stuck state and calls the callback.
// An agent is stuck if no heartbeat received for 2× interval (AC-RESIL-02).
func (m *HeartbeatMonitor) CheckStuck() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stuckTimeout := 2 * m.config.Interval
	now := time.Now()
	var stuckAgents []string

	for agentID, entry := range m.agents {
		if now.Sub(entry.lastHeartbeat) > stuckTimeout {
			stuckAgents = append(stuckAgents, agentID)
			slog.Warn("[Heartbeat] agent stuck",
				"agent_id", agentID, "type", entry.agentType,
				"last_heartbeat", entry.lastHeartbeat,
				"elapsed", now.Sub(entry.lastHeartbeat))

			if m.callback != nil {
				m.callback(agentID, entry.agentType, entry.lastHeartbeat)
			}
		}
	}
	return stuckAgents
}

// Start begins periodic stuck checking.
func (m *HeartbeatMonitor) Start(ctx context.Context) {
	ctx, m.cancel = context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(m.config.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.CheckStuck()
			}
		}
	}()
}

// Stop stops the heartbeat monitor.
func (m *HeartbeatMonitor) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
}

// GetLastHeartbeat returns the last heartbeat time for an agent.
func (m *HeartbeatMonitor) GetLastHeartbeat(agentID string) (time.Time, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.agents[agentID]
	if !ok {
		return time.Time{}, false
	}
	return entry.lastHeartbeat, true
}

// AgentCount returns the number of monitored agents.
func (m *HeartbeatMonitor) AgentCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.agents)
}

// HeartbeatSnapshot is a point-in-time view of an agent's heartbeat state.
type HeartbeatSnapshot struct {
	AgentID       string    `json:"agent_id"`
	AgentType     AgentType `json:"agent_type"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	CurrentStep   string    `json:"current_step,omitempty"`
}

// Snapshots returns a snapshot of all monitored agents.
func (m *HeartbeatMonitor) Snapshots() []HeartbeatSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]HeartbeatSnapshot, 0, len(m.agents))
	for agentID, entry := range m.agents {
		result = append(result, HeartbeatSnapshot{
			AgentID:       agentID,
			AgentType:     entry.agentType,
			LastHeartbeat: entry.lastHeartbeat,
			CurrentStep:   entry.currentStep,
		})
	}
	return result
}

// StuckSnapshot is a point-in-time view of an agent believed to be stuck.
// Returned by StuckSnapshots — the caller gets elapsed time already computed
// so the UI does not need to know the stuck threshold.
type StuckSnapshot struct {
	AgentID       string    `json:"agent_id"`
	AgentType     AgentType `json:"agent_type"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	ElapsedMs     int64     `json:"elapsed_ms"`
	Status        string    `json:"status"`
	CurrentStep   string    `json:"current_step,omitempty"`
}

// StuckSnapshots returns snapshots of agents whose last heartbeat is older
// than the stuck threshold (2× Interval). An agent is reported at most
// once per call. Unlike CheckStuck, this is a pure read — it does not
// invoke the stuck callback.
func (m *HeartbeatMonitor) StuckSnapshots() []StuckSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stuckTimeout := 2 * m.config.Interval
	now := time.Now()
	out := make([]StuckSnapshot, 0)
	for agentID, entry := range m.agents {
		elapsed := now.Sub(entry.lastHeartbeat)
		if elapsed <= stuckTimeout {
			continue
		}
		out = append(out, StuckSnapshot{
			AgentID:       agentID,
			AgentType:     entry.agentType,
			LastHeartbeat: entry.lastHeartbeat,
			ElapsedMs:     elapsed.Milliseconds(),
			Status:        "stuck",
			CurrentStep:   entry.currentStep,
		})
	}
	return out
}

package resilience

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHeartbeatMonitor_RegisterAndHeartbeat(t *testing.T) {
	monitor := NewHeartbeatMonitor(HeartbeatConfig{Interval: 50 * time.Millisecond}, nil)

	monitor.Register("agent-1", AgentTypeSpawn)
	assert.Equal(t, 1, monitor.AgentCount())

	// Record heartbeat
	monitor.RecordHeartbeat(HeartbeatEvent{
		AgentID:     "agent-1",
		Timestamp:   time.Now(),
		CurrentStep: "step-3",
	})

	last, ok := monitor.GetLastHeartbeat("agent-1")
	assert.True(t, ok)
	assert.WithinDuration(t, time.Now(), last, time.Second)

	// Unknown agent
	_, ok = monitor.GetLastHeartbeat("unknown")
	assert.False(t, ok)
}

func TestHeartbeatMonitor_DetectsStuck(t *testing.T) {
	// AC-RESIL-02: parent gets event when agent stuck (2× miss)
	var mu sync.Mutex
	var stuckCalls []string

	callback := func(agentID string, agentType AgentType, lastHB time.Time) {
		mu.Lock()
		stuckCalls = append(stuckCalls, agentID)
		mu.Unlock()
	}

	monitor := NewHeartbeatMonitor(HeartbeatConfig{Interval: 30 * time.Millisecond}, callback)
	monitor.Register("agent-1", AgentTypeSpawn)

	// Fresh agent — not stuck
	stuck := monitor.CheckStuck()
	assert.Len(t, stuck, 0)

	// Wait for 2× interval
	time.Sleep(70 * time.Millisecond)

	stuck = monitor.CheckStuck()
	assert.Len(t, stuck, 1)
	assert.Equal(t, "agent-1", stuck[0])

	mu.Lock()
	assert.Len(t, stuckCalls, 1)
	mu.Unlock()
}

func TestHeartbeatMonitor_HeartbeatPreventsStuck(t *testing.T) {
	monitor := NewHeartbeatMonitor(HeartbeatConfig{Interval: 50 * time.Millisecond}, nil)
	monitor.Register("agent-1", AgentTypeSpawn)

	// Send heartbeat before stuck timeout
	time.Sleep(30 * time.Millisecond)
	monitor.RecordHeartbeat(HeartbeatEvent{
		AgentID:   "agent-1",
		Timestamp: time.Now(),
	})

	time.Sleep(30 * time.Millisecond)
	stuck := monitor.CheckStuck()
	assert.Len(t, stuck, 0)
}

func TestHeartbeatMonitor_Unregister(t *testing.T) {
	monitor := NewHeartbeatMonitor(DefaultHeartbeatConfig(), nil)
	monitor.Register("agent-1", AgentTypeSpawn)
	assert.Equal(t, 1, monitor.AgentCount())

	monitor.Unregister("agent-1")
	assert.Equal(t, 0, monitor.AgentCount())
}

func TestHeartbeatMonitor_MultipleAgents(t *testing.T) {
	monitor := NewHeartbeatMonitor(HeartbeatConfig{Interval: 30 * time.Millisecond}, nil)
	monitor.Register("agent-1", AgentTypeSpawn)
	monitor.Register("agent-2", AgentTypePersistent)
	assert.Equal(t, 2, monitor.AgentCount())

	// Only agent-1 heartbeats
	time.Sleep(70 * time.Millisecond)
	monitor.RecordHeartbeat(HeartbeatEvent{AgentID: "agent-1", Timestamp: time.Now()})

	stuck := monitor.CheckStuck()
	assert.Len(t, stuck, 1)
	assert.Equal(t, "agent-2", stuck[0])
}

package agent

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/service/orchestrator"
	"github.com/syntheticinc/bytebrew/engine/pkg/config"
)

// TestWaitForAllSessionAgents_AllComplete verifies that WaitForAllSessionAgents
// returns AllDone when all agents complete normally
func TestWaitForAllSessionAgents_AllComplete(t *testing.T) {
	ctx := context.Background()
	mgr := newMockSubtaskManager()
	mgr.addSubtask("subtask-1", &domain.Subtask{ID: "subtask-1", Title: "Test 1", Status: domain.SubtaskStatusPending})
	mgr.addSubtask("subtask-2", &domain.Subtask{ID: "subtask-2", Title: "Test 2", Status: domain.SubtaskStatusPending})

	pool := NewAgentPool(AgentPoolConfig{
		SubtaskManager: mgr,
		AgentConfig:    &config.AgentConfig{},
	})

	// Create 2 blocking agents directly (without Spawn)
	agent1 := "agent-1"
	agent2 := "agent-2"
	pool.mu.Lock()
	pool.agents[agent1] = &RunningAgent{
		ID:            agent1,
		SubtaskID:     "subtask-1",
		SessionID:     "session-1",
		Status:        "running",
		StartedAt:     time.Now(),
		completionCh:  make(chan struct{}),
		blockingSpawn: true,
		Cancel:        func() {},
	}
	pool.agents[agent2] = &RunningAgent{
		ID:            agent2,
		SubtaskID:     "subtask-2",
		SessionID:     "session-1",
		Status:        "running",
		StartedAt:     time.Now(),
		completionCh:  make(chan struct{}),
		blockingSpawn: true,
		Cancel:        func() {},
	}
	pool.mu.Unlock()

	// Wait in goroutine
	done := make(chan WaitResult, 1)
	go func() {
		result, err := pool.WaitForAllSessionAgents(ctx, "session-1")
		if err != nil {
			t.Errorf("wait failed: %v", err)
		}
		done <- result
	}()

	// Give goroutine time to start waiting
	time.Sleep(50 * time.Millisecond)

	// Complete both agents
	pool.mu.Lock()
	pool.agents[agent1].Status = "completed"
	pool.agents[agent1].Result = "result 1"
	a1 := pool.agents[agent1]
	pool.mu.Unlock()
	a1.signalCompletion()

	pool.mu.Lock()
	pool.agents[agent2].Status = "completed"
	pool.agents[agent2].Result = "result 2"
	a2 := pool.agents[agent2]
	pool.mu.Unlock()
	a2.signalCompletion()

	// Wait should return AllDone
	select {
	case result := <-done:
		if !result.AllDone {
			t.Errorf("expected AllDone=true, got false")
		}
		if result.Interrupted {
			t.Errorf("expected Interrupted=false, got true")
		}
		if len(result.Results) != 2 {
			t.Errorf("expected 2 results, got %d", len(result.Results))
		}
		if r, ok := result.Results[agent1]; ok {
			if r.Result != "result 1" {
				t.Errorf("agent1 result mismatch: got %q", r.Result)
			}
		} else {
			t.Errorf("agent1 result not found")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("wait timed out")
	}
}

// TestWaitForAllSessionAgents_Interrupt verifies that NotifyUserMessage
// interrupts the wait
func TestWaitForAllSessionAgents_Interrupt(t *testing.T) {
	ctx := context.Background()
	mgr := newMockSubtaskManager()
	mgr.addSubtask("subtask-1", &domain.Subtask{ID: "subtask-1", Title: "Test 1", Status: domain.SubtaskStatusPending})
	mgr.addSubtask("subtask-2", &domain.Subtask{ID: "subtask-2", Title: "Test 2", Status: domain.SubtaskStatusPending})

	pool := NewAgentPool(AgentPoolConfig{
		SubtaskManager: mgr,
		AgentConfig:    &config.AgentConfig{},
	})

	// Create 2 blocking agents directly
	agent1 := "agent-1"
	agent2 := "agent-2"
	pool.mu.Lock()
	pool.agents[agent1] = &RunningAgent{
		ID:            agent1,
		SubtaskID:     "subtask-1",
		SessionID:     "session-1",
		Status:        "running",
		StartedAt:     time.Now(),
		completionCh:  make(chan struct{}),
		blockingSpawn: true,
		Cancel:        func() {},
	}
	pool.agents[agent2] = &RunningAgent{
		ID:            agent2,
		SubtaskID:     "subtask-2",
		SessionID:     "session-1",
		Status:        "running",
		StartedAt:     time.Now(),
		completionCh:  make(chan struct{}),
		blockingSpawn: true,
		Cancel:        func() {},
	}
	pool.mu.Unlock()

	// Wait in goroutine
	done := make(chan WaitResult, 1)
	go func() {
		result, err := pool.WaitForAllSessionAgents(ctx, "session-1")
		if err != nil {
			t.Errorf("wait failed: %v", err)
		}
		done <- result
	}()

	// Give goroutine time to start waiting
	time.Sleep(50 * time.Millisecond)

	// Complete one agent
	pool.mu.Lock()
	pool.agents[agent1].Status = "completed"
	pool.agents[agent1].Result = "result 1"
	a1 := pool.agents[agent1]
	pool.mu.Unlock()
	a1.signalCompletion()

	// Send interrupt (while agent2 still running)
	pool.NotifyUserMessage("session-1", "stop and fix bug")

	// Wait should return Interrupted
	select {
	case result := <-done:
		if result.AllDone {
			t.Errorf("expected AllDone=false, got true")
		}
		if !result.Interrupted {
			t.Errorf("expected Interrupted=true, got false")
		}
		if !result.IsInterruptResponder {
			t.Errorf("expected IsInterruptResponder=true (only one waiter)")
		}
		if result.UserMessage != "stop and fix bug" {
			t.Errorf("user message mismatch: got %q", result.UserMessage)
		}
		if len(result.StillRunning) == 0 {
			t.Errorf("expected still running agents")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("wait timed out")
	}
}

// TestWaitForAllSessionAgents_ParallelInterrupt_SingleResponder verifies
// that only ONE parallel waiter gets IsInterruptResponder=true
func TestWaitForAllSessionAgents_ParallelInterrupt_SingleResponder(t *testing.T) {
	ctx := context.Background()
	mgr := newMockSubtaskManager()
	mgr.addSubtask("subtask-1", &domain.Subtask{ID: "subtask-1", Title: "Test 1", Status: domain.SubtaskStatusPending})

	pool := NewAgentPool(AgentPoolConfig{
		SubtaskManager: mgr,
		AgentConfig:    &config.AgentConfig{},
	})

	// Create 1 blocking agent directly
	agent1 := "agent-1"
	pool.mu.Lock()
	pool.agents[agent1] = &RunningAgent{
		ID:            agent1,
		SubtaskID:     "subtask-1",
		SessionID:     "session-1",
		Status:        "running",
		StartedAt:     time.Now(),
		completionCh:  make(chan struct{}),
		blockingSpawn: true,
		Cancel:        func() {},
	}
	pool.mu.Unlock()

	// Start 3 parallel waiters
	var wg sync.WaitGroup
	results := make([]WaitResult, 3)
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			result, err := pool.WaitForAllSessionAgents(ctx, "session-1")
			if err != nil {
				t.Errorf("wait failed: %v", err)
			}
			results[idx] = result
		}(i)
	}

	// Give goroutines time to start waiting
	time.Sleep(100 * time.Millisecond)

	// Send interrupt
	pool.NotifyUserMessage("session-1", "urgent message")

	// Wait for all to finish
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Count responders
		responderCount := 0
		pausedCount := 0
		for i, r := range results {
			if !r.Interrupted {
				t.Errorf("waiter %d: expected Interrupted=true", i)
			}
			if r.IsInterruptResponder {
				responderCount++
			} else {
				pausedCount++
			}
		}
		if responderCount != 1 {
			t.Errorf("expected exactly 1 responder, got %d", responderCount)
		}
		if pausedCount != 2 {
			t.Errorf("expected 2 paused, got %d", pausedCount)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("parallel wait timed out")
	}
}

// TestWaitForAllSessionAgents_ContextCancelled verifies that context cancellation
// causes wait to return error
func TestWaitForAllSessionAgents_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	mgr := newMockSubtaskManager()
	mgr.addSubtask("subtask-1", &domain.Subtask{ID: "subtask-1", Title: "Test 1", Status: domain.SubtaskStatusPending})

	pool := NewAgentPool(AgentPoolConfig{
		SubtaskManager: mgr,
		AgentConfig:    &config.AgentConfig{},
	})

	// Create blocking agent directly
	agent1 := "agent-1"
	pool.mu.Lock()
	pool.agents[agent1] = &RunningAgent{
		ID:            agent1,
		SubtaskID:     "subtask-1",
		SessionID:     "session-1",
		Status:        "running",
		StartedAt:     time.Now(),
		completionCh:  make(chan struct{}),
		blockingSpawn: true,
		Cancel:        func() {},
	}
	pool.mu.Unlock()

	// Wait in goroutine with cancellable context
	done := make(chan error, 1)
	go func() {
		_, err := pool.WaitForAllSessionAgents(ctx, "session-1")
		done <- err
	}()

	// Give goroutine time to start waiting
	time.Sleep(50 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait should return context.Canceled
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("wait did not return on context cancel")
	}
}

// TestHasBlockingWait_TrueWhenWaiting verifies HasBlockingWait returns true
// when a waiter is active
func TestHasBlockingWait_TrueWhenWaiting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensures WaitForAllSessionAgents goroutine exits via ctx.Done()

	mgr := newMockSubtaskManager()
	mgr.addSubtask("subtask-1", &domain.Subtask{ID: "subtask-1", Title: "Test", Status: domain.SubtaskStatusPending})

	pool := NewAgentPool(AgentPoolConfig{
		SubtaskManager: mgr,
		AgentConfig:    &config.AgentConfig{},
	})

	// Initially false
	if pool.HasBlockingWait("session-1") {
		t.Errorf("expected HasBlockingWait=false before wait")
	}

	// Create agent directly
	completionCh := make(chan struct{})
	agent1 := "agent-1"
	pool.mu.Lock()
	pool.agents[agent1] = &RunningAgent{
		ID:            agent1,
		SubtaskID:     "subtask-1",
		SessionID:     "session-1",
		Status:        "running",
		StartedAt:     time.Now(),
		completionCh:  completionCh,
		blockingSpawn: true,
		Cancel:        func() {},
	}
	pool.mu.Unlock()

	// Start wait in background
	waitDone := make(chan struct{})
	go func() {
		pool.WaitForAllSessionAgents(ctx, "session-1")
		close(waitDone)
	}()

	// Give goroutine time to start waiting
	time.Sleep(50 * time.Millisecond)

	// Should be true now
	if !pool.HasBlockingWait("session-1") {
		t.Errorf("expected HasBlockingWait=true during wait")
	}

	// Cleanup: cancel context so goroutine exits, then close completionCh
	// to unblock the inner goroutine in WaitForAllSessionAgents
	cancel()
	<-waitDone
	close(completionCh)
}

// TestHasBlockingWait_FalseWhenNotWaiting verifies HasBlockingWait returns false
// when no waiter is active
func TestHasBlockingWait_FalseWhenNotWaiting(t *testing.T) {
	pool := NewAgentPool(AgentPoolConfig{
		SubtaskManager: newMockSubtaskManager(),
		AgentConfig:    &config.AgentConfig{},
	})

	if pool.HasBlockingWait("session-1") {
		t.Errorf("expected HasBlockingWait=false for non-existent session")
	}
}

// TestSignalCompletion_DoubleClose verifies that signalCompletion can be called
// multiple times without panic (sync.Once protection)
func TestSignalCompletion_DoubleClose(t *testing.T) {
	agent := &RunningAgent{
		completionCh: make(chan struct{}),
	}

	// Call twice
	agent.signalCompletion()
	agent.signalCompletion()

	// Should not panic
	select {
	case <-agent.completionCh:
		// OK, channel closed
	default:
		t.Error("completion channel not closed")
	}
}

// TestBlockingSpawn_PublishesEventBus verifies that both blocking and non-blocking
// agents publish to EventBus so Orchestrator can track active work status.
func TestBlockingSpawn_PublishesEventBus(t *testing.T) {
	mgr := newMockSubtaskManager()
	mgr.addSubtask("subtask-1", &domain.Subtask{ID: "subtask-1", Title: "Test", Status: domain.SubtaskStatusPending})

	pool := NewAgentPool(AgentPoolConfig{
		SubtaskManager: mgr,
		AgentConfig:    &config.AgentConfig{},
	})

	// Use real SessionEventBus
	bus := orchestrator.NewSessionEventBus(64)
	pool.eventBus = bus

	// Create blocking agent directly
	agentID := "agent-1"
	pool.mu.Lock()
	pool.agents[agentID] = &RunningAgent{
		ID:            agentID,
		SubtaskID:     "subtask-1",
		SessionID:     "session-1",
		Status:        "running",
		StartedAt:     time.Now(),
		completionCh:  make(chan struct{}),
		blockingSpawn: true,
		Cancel:        func() {},
	}
	pool.mu.Unlock()

	// Mark completed
	pool.markCompleted(agentID, "subtask-1", "result")

	// Give async operations time to complete
	time.Sleep(50 * time.Millisecond)

	// EventBus SHOULD have events (blocking agents also publish for Orchestrator state tracking)
	select {
	case evt := <-bus.Events():
		if evt.Type != orchestrator.EventAgentCompleted {
			t.Errorf("expected EventAgentCompleted, got %v", evt.Type)
		}
	default:
		t.Error("expected EventBus event for blocking spawn completion")
	}

	// Now test non-blocking
	mgr.addSubtask("subtask-2", &domain.Subtask{ID: "subtask-2", Title: "Test 2", Status: domain.SubtaskStatusPending})
	agentID2 := "agent-2"
	pool.mu.Lock()
	pool.agents[agentID2] = &RunningAgent{
		ID:            agentID2,
		SubtaskID:     "subtask-2",
		SessionID:     "session-1",
		Status:        "running",
		StartedAt:     time.Now(),
		completionCh:  make(chan struct{}),
		blockingSpawn: false, // non-blocking
		Cancel:        func() {},
	}
	pool.mu.Unlock()

	pool.markCompleted(agentID2, "subtask-2", "result 2")

	// Give async operations time to complete
	time.Sleep(50 * time.Millisecond)

	// EventBus SHOULD have event (non-blocking spawn publishes)
	select {
	case <-bus.Events():
		// Good — got event
	default:
		t.Errorf("expected EventBus event for non-blocking spawn, got none")
	}
}

package persistence

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentRunStorage_SaveAndGetByID(t *testing.T) {
	ctx := context.Background()

	// Setup temp DB
	tmpDir, err := os.MkdirTemp("", "agent_run_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := NewWorkDB(dbPath)
	require.NoError(t, err)
	defer db.Close()

	storage, err := NewSQLiteAgentRunStorage(db)
	require.NoError(t, err)

	// Create agent run
	run, err := domain.NewAgentRun("agent-1", "subtask-1", "session-1", domain.FlowType("coder"))
	require.NoError(t, err)

	// Save
	err = storage.Save(ctx, run)
	require.NoError(t, err)

	// Get by ID
	retrieved, err := storage.GetByID(ctx, "agent-1")
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, run.ID, retrieved.ID)
	assert.Equal(t, run.SubtaskID, retrieved.SubtaskID)
	assert.Equal(t, run.SessionID, retrieved.SessionID)
	assert.Equal(t, run.FlowType, retrieved.FlowType)
	assert.Equal(t, domain.AgentRunRunning, retrieved.Status)
	assert.Nil(t, retrieved.CompletedAt)
}

func TestAgentRunStorage_Update(t *testing.T) {
	ctx := context.Background()

	tmpDir, err := os.MkdirTemp("", "agent_run_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := NewWorkDB(dbPath)
	require.NoError(t, err)
	defer db.Close()

	storage, err := NewSQLiteAgentRunStorage(db)
	require.NoError(t, err)

	run, err := domain.NewAgentRun("agent-1", "subtask-1", "session-1", domain.FlowType("coder"))
	require.NoError(t, err)

	err = storage.Save(ctx, run)
	require.NoError(t, err)

	// Complete the run
	run.Complete("success")

	// Update
	err = storage.Update(ctx, run)
	require.NoError(t, err)

	// Verify
	retrieved, err := storage.GetByID(ctx, "agent-1")
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, domain.AgentRunCompleted, retrieved.Status)
	assert.Equal(t, "success", retrieved.Result)
	assert.NotNil(t, retrieved.CompletedAt)
}

func TestAgentRunStorage_GetRunningBySession(t *testing.T) {
	ctx := context.Background()

	tmpDir, err := os.MkdirTemp("", "agent_run_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := NewWorkDB(dbPath)
	require.NoError(t, err)
	defer db.Close()

	storage, err := NewSQLiteAgentRunStorage(db)
	require.NoError(t, err)

	// Create 3 runs: 2 running, 1 completed
	run1, _ := domain.NewAgentRun("agent-1", "subtask-1", "session-1", domain.FlowType("coder"))
	run2, _ := domain.NewAgentRun("agent-2", "subtask-2", "session-1", domain.FlowType("coder"))
	run3, _ := domain.NewAgentRun("agent-3", "subtask-3", "session-1", domain.FlowType("coder"))

	run3.Complete("done")

	storage.Save(ctx, run1)
	storage.Save(ctx, run2)
	storage.Save(ctx, run3)

	// Get running
	running, err := storage.GetRunningBySession(ctx, "session-1")
	require.NoError(t, err)

	assert.Len(t, running, 2)
	assert.Contains(t, []string{running[0].ID, running[1].ID}, "agent-1")
	assert.Contains(t, []string{running[0].ID, running[1].ID}, "agent-2")
}

func TestAgentRunStorage_CountRunningBySession(t *testing.T) {
	ctx := context.Background()

	tmpDir, err := os.MkdirTemp("", "agent_run_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := NewWorkDB(dbPath)
	require.NoError(t, err)
	defer db.Close()

	storage, err := NewSQLiteAgentRunStorage(db)
	require.NoError(t, err)

	run1, _ := domain.NewAgentRun("agent-1", "subtask-1", "session-1", domain.FlowType("coder"))
	run2, _ := domain.NewAgentRun("agent-2", "subtask-2", "session-1", domain.FlowType("coder"))

	storage.Save(ctx, run1)
	storage.Save(ctx, run2)

	count, err := storage.CountRunningBySession(ctx, "session-1")
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Complete one
	run1.Complete("done")
	storage.Update(ctx, run1)

	count, err = storage.CountRunningBySession(ctx, "session-1")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestAgentRunStorage_CleanupOrphanedRuns(t *testing.T) {
	ctx := context.Background()

	tmpDir, err := os.MkdirTemp("", "agent_run_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := NewWorkDB(dbPath)
	require.NoError(t, err)
	defer db.Close()

	storage, err := NewSQLiteAgentRunStorage(db)
	require.NoError(t, err)

	// Create 3 running agents
	run1, _ := domain.NewAgentRun("agent-1", "subtask-1", "session-1", domain.FlowType("coder"))
	run2, _ := domain.NewAgentRun("agent-2", "subtask-2", "session-1", domain.FlowType("coder"))
	run3, _ := domain.NewAgentRun("agent-3", "subtask-3", "session-2", domain.FlowType("coder"))

	storage.Save(ctx, run1)
	storage.Save(ctx, run2)
	storage.Save(ctx, run3)

	// Cleanup (simulating server restart)
	cleaned, err := storage.CleanupOrphanedRuns(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(3), cleaned)

	// Verify all marked as stopped
	retrieved1, _ := storage.GetByID(ctx, "agent-1")
	assert.Equal(t, domain.AgentRunStopped, retrieved1.Status)
	assert.NotNil(t, retrieved1.CompletedAt)

	// Count running should be 0
	count, _ := storage.CountRunningBySession(ctx, "session-1")
	assert.Equal(t, 0, count)
}

func TestAgentRunStorage_GetByID_NotFound(t *testing.T) {
	ctx := context.Background()

	tmpDir, err := os.MkdirTemp("", "agent_run_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := NewWorkDB(dbPath)
	require.NoError(t, err)
	defer db.Close()

	storage, err := NewSQLiteAgentRunStorage(db)
	require.NoError(t, err)

	retrieved, err := storage.GetByID(ctx, "non-existent")
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestAgentRunStorage_FailedRun(t *testing.T) {
	ctx := context.Background()

	tmpDir, err := os.MkdirTemp("", "agent_run_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := NewWorkDB(dbPath)
	require.NoError(t, err)
	defer db.Close()

	storage, err := NewSQLiteAgentRunStorage(db)
	require.NoError(t, err)

	run, err := domain.NewAgentRun("agent-1", "subtask-1", "session-1", domain.FlowType("coder"))
	require.NoError(t, err)

	storage.Save(ctx, run)

	// Fail the run
	run.Fail("timeout error")
	storage.Update(ctx, run)

	// Verify
	retrieved, err := storage.GetByID(ctx, "agent-1")
	require.NoError(t, err)

	assert.Equal(t, domain.AgentRunFailed, retrieved.Status)
	assert.Equal(t, "timeout error", retrieved.Error)
	assert.NotNil(t, retrieved.CompletedAt)
}

func TestAgentRunStorage_GetBySessionID(t *testing.T) {
	ctx := context.Background()

	tmpDir, err := os.MkdirTemp("", "agent_run_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := NewWorkDB(dbPath)
	require.NoError(t, err)
	defer db.Close()

	storage, err := NewSQLiteAgentRunStorage(db)
	require.NoError(t, err)

	// Create runs in different sessions with distinct timestamps
	run1, _ := domain.NewAgentRun("agent-1", "subtask-1", "session-1", domain.FlowType("coder"))
	storage.Save(ctx, run1)

	// Wait for different second (SQLite timestamp resolution)
	time.Sleep(1100 * time.Millisecond)

	run2, _ := domain.NewAgentRun("agent-2", "subtask-2", "session-1", domain.FlowType("coder"))
	storage.Save(ctx, run2)

	run3, _ := domain.NewAgentRun("agent-3", "subtask-3", "session-2", domain.FlowType("coder"))
	storage.Save(ctx, run3)

	// Get by session
	runs, err := storage.GetBySessionID(ctx, "session-1")
	require.NoError(t, err)

	assert.Len(t, runs, 2)
	// Should be ordered by started_at DESC (most recent first)
	assert.Equal(t, "agent-2", runs[0].ID)
	assert.Equal(t, "agent-1", runs[1].ID)
}

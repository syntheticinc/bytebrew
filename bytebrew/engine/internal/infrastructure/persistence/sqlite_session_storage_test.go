package persistence

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestSessionDB(t *testing.T) (*sql.DB, func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_sessions.db")

	db, err := NewWorkDB(dbPath)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func TestSessionStorage_SaveAndGetByID(t *testing.T) {
	ctx := context.Background()
	db, cleanup := setupTestSessionDB(t)
	defer cleanup()

	storage, err := NewSQLiteSessionStorage(db)
	require.NoError(t, err)

	// Create and save session
	session, err := domain.NewSession("session-1", "project-a")
	require.NoError(t, err)

	err = storage.Save(ctx, session)
	require.NoError(t, err)

	// Retrieve session
	retrieved, err := storage.GetByID(ctx, "session-1")
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, session.ID, retrieved.ID)
	assert.Equal(t, session.ProjectKey, retrieved.ProjectKey)
	assert.Equal(t, domain.SessionActive, retrieved.Status)
	assert.WithinDuration(t, session.CreatedAt, retrieved.CreatedAt, time.Second)
	assert.WithinDuration(t, session.LastActivityAt, retrieved.LastActivityAt, time.Second)
}

func TestSessionStorage_Update(t *testing.T) {
	ctx := context.Background()
	db, cleanup := setupTestSessionDB(t)
	defer cleanup()

	storage, err := NewSQLiteSessionStorage(db)
	require.NoError(t, err)

	// Create and save session
	session, err := domain.NewSession("session-1", "project-a")
	require.NoError(t, err)

	err = storage.Save(ctx, session)
	require.NoError(t, err)

	// Update session status
	session.Suspend()
	err = storage.Update(ctx, session)
	require.NoError(t, err)

	// Retrieve and verify
	retrieved, err := storage.GetByID(ctx, "session-1")
	require.NoError(t, err)
	assert.Equal(t, domain.SessionSuspended, retrieved.Status)
}

func TestSessionStorage_GetLatestByProjectKey(t *testing.T) {
	ctx := context.Background()
	db, cleanup := setupTestSessionDB(t)
	defer cleanup()

	storage, err := NewSQLiteSessionStorage(db)
	require.NoError(t, err)

	// Create multiple sessions for same project with explicit timing
	now := time.Now()

	session1, _ := domain.NewSession("session-1", "project-a")
	session1.LastActivityAt = now
	storage.Save(ctx, session1)

	session2, _ := domain.NewSession("session-2", "project-a")
	session2.LastActivityAt = now.Add(1 * time.Second)
	storage.Save(ctx, session2)

	session3, _ := domain.NewSession("session-3", "project-a")
	session3.LastActivityAt = now.Add(2 * time.Second)
	storage.Save(ctx, session3)

	// Get latest
	latest, err := storage.GetLatestByProjectKey(ctx, "project-a")
	require.NoError(t, err)
	require.NotNil(t, latest)

	assert.Equal(t, "session-3", latest.ID)
}

func TestSessionStorage_SuspendActiveSessions(t *testing.T) {
	ctx := context.Background()
	db, cleanup := setupTestSessionDB(t)
	defer cleanup()

	storage, err := NewSQLiteSessionStorage(db)
	require.NoError(t, err)

	// Create multiple sessions
	session1, _ := domain.NewSession("session-1", "project-a")
	session2, _ := domain.NewSession("session-2", "project-b")
	session3, _ := domain.NewSession("session-3", "project-c")
	session3.Complete()

	storage.Save(ctx, session1)
	storage.Save(ctx, session2)
	storage.Save(ctx, session3)

	// Suspend active sessions
	count, err := storage.SuspendActiveSessions(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count) // session1 and session2

	// Verify
	s1, _ := storage.GetByID(ctx, "session-1")
	assert.Equal(t, domain.SessionSuspended, s1.Status)

	s2, _ := storage.GetByID(ctx, "session-2")
	assert.Equal(t, domain.SessionSuspended, s2.Status)

	s3, _ := storage.GetByID(ctx, "session-3")
	assert.Equal(t, domain.SessionCompleted, s3.Status) // unchanged
}

func TestSessionStorage_GetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	db, cleanup := setupTestSessionDB(t)
	defer cleanup()

	storage, err := NewSQLiteSessionStorage(db)
	require.NoError(t, err)

	// Try to get non-existent session
	session, err := storage.GetByID(ctx, "non-existent")
	require.NoError(t, err)
	assert.Nil(t, session)
}

func TestSessionStorage_ListByProjectKey(t *testing.T) {
	ctx := context.Background()
	db, cleanup := setupTestSessionDB(t)
	defer cleanup()

	storage, err := NewSQLiteSessionStorage(db)
	require.NoError(t, err)

	// Create sessions for different projects with explicit timing
	now := time.Now()

	sessionA1, _ := domain.NewSession("session-a1", "project-a")
	sessionA1.LastActivityAt = now
	storage.Save(ctx, sessionA1)

	sessionA2, _ := domain.NewSession("session-a2", "project-a")
	sessionA2.LastActivityAt = now.Add(1 * time.Second)
	storage.Save(ctx, sessionA2)

	sessionB1, _ := domain.NewSession("session-b1", "project-b")
	storage.Save(ctx, sessionB1)

	// List sessions for project-a
	sessions, err := storage.ListByProjectKey(ctx, "project-a")
	require.NoError(t, err)
	assert.Len(t, sessions, 2)

	// Should be ordered by last_activity_at DESC
	assert.Equal(t, "session-a2", sessions[0].ID)
	assert.Equal(t, "session-a1", sessions[1].ID)
}

package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
)

const createSessionsTableSQL = `
CREATE TABLE IF NOT EXISTS sessions (
	id TEXT PRIMARY KEY,
	project_key TEXT NOT NULL,
	status TEXT NOT NULL CHECK(status IN ('active','suspended','completed')),
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL,
	last_activity_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_project ON sessions(project_key);
CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
`

// SQLiteSessionStorage implements session persistence using SQLite
type SQLiteSessionStorage struct {
	db *sql.DB
}

// NewSQLiteSessionStorage creates a new session storage using the shared work DB.
// The caller is responsible for calling NewWorkDB first and passing the *sql.DB.
func NewSQLiteSessionStorage(db *sql.DB) (*SQLiteSessionStorage, error) {
	if _, err := db.Exec(createSessionsTableSQL); err != nil {
		return nil, fmt.Errorf("create sessions table: %w", err)
	}

	slog.Info("SQLite session storage initialized")
	return &SQLiteSessionStorage{db: db}, nil
}

// Save persists a new session
func (s *SQLiteSessionStorage) Save(ctx context.Context, session *domain.Session) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, project_key, status, created_at, updated_at, last_activity_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, session.ID, session.ProjectKey, string(session.Status),
		session.CreatedAt.Unix(), session.UpdatedAt.Unix(), session.LastActivityAt.Unix())

	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}

	slog.DebugContext(ctx, "session saved", "session_id", session.ID, "project_key", session.ProjectKey)
	return nil
}

// Update updates an existing session
func (s *SQLiteSessionStorage) Update(ctx context.Context, session *domain.Session) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE sessions
		SET project_key = ?, status = ?, updated_at = ?, last_activity_at = ?
		WHERE id = ?
	`, session.ProjectKey, string(session.Status),
		session.UpdatedAt.Unix(), session.LastActivityAt.Unix(), session.ID)

	if err != nil {
		return fmt.Errorf("update session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("session not found: %s", session.ID)
	}

	slog.DebugContext(ctx, "session updated", "session_id", session.ID, "status", session.Status)
	return nil
}

// GetByID retrieves a session by ID (returns nil if not found)
func (s *SQLiteSessionStorage) GetByID(ctx context.Context, id string) (*domain.Session, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, project_key, status, created_at, updated_at, last_activity_at
		FROM sessions WHERE id = ?
	`, id)

	return s.scanSession(row)
}

// GetLatestByProjectKey retrieves the most recent session for a project
func (s *SQLiteSessionStorage) GetLatestByProjectKey(ctx context.Context, projectKey string) (*domain.Session, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, project_key, status, created_at, updated_at, last_activity_at
		FROM sessions
		WHERE project_key = ?
		ORDER BY last_activity_at DESC
		LIMIT 1
	`, projectKey)

	return s.scanSession(row)
}

// ListByProjectKey retrieves all sessions for a project, ordered by recent activity
func (s *SQLiteSessionStorage) ListByProjectKey(ctx context.Context, projectKey string) ([]*domain.Session, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, project_key, status, created_at, updated_at, last_activity_at
		FROM sessions
		WHERE project_key = ?
		ORDER BY last_activity_at DESC
	`, projectKey)
	if err != nil {
		return nil, fmt.Errorf("query sessions by project: %w", err)
	}
	defer rows.Close()

	return s.scanSessions(rows)
}

// SuspendActiveSessions marks all 'active' sessions as 'suspended'.
// Called at server startup to handle crash recovery.
// Returns the number of sessions updated.
func (s *SQLiteSessionStorage) SuspendActiveSessions(ctx context.Context) (int64, error) {
	now := time.Now().Unix()
	result, err := s.db.ExecContext(ctx,
		`UPDATE sessions SET status = 'suspended', updated_at = ? WHERE status = 'active'`,
		now)
	if err != nil {
		return 0, fmt.Errorf("suspend active sessions: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}

	return affected, nil
}

// Close is a no-op because the shared DB is owned by the caller
func (s *SQLiteSessionStorage) Close() error {
	return nil
}

// scanSession scans a single session from a row (returns nil if not found)
func (s *SQLiteSessionStorage) scanSession(row *sql.Row) (*domain.Session, error) {
	var (
		id             string
		projectKey     string
		status         string
		createdAt      int64
		updatedAt      int64
		lastActivityAt int64
	)

	err := row.Scan(&id, &projectKey, &status, &createdAt, &updatedAt, &lastActivityAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan session: %w", err)
	}

	return s.buildSession(id, projectKey, status, createdAt, updatedAt, lastActivityAt)
}

// scanSessions scans multiple sessions from rows
func (s *SQLiteSessionStorage) scanSessions(rows *sql.Rows) ([]*domain.Session, error) {
	var sessions []*domain.Session

	for rows.Next() {
		var (
			id             string
			projectKey     string
			status         string
			createdAt      int64
			updatedAt      int64
			lastActivityAt int64
		)

		if err := rows.Scan(&id, &projectKey, &status, &createdAt, &updatedAt, &lastActivityAt); err != nil {
			return nil, fmt.Errorf("scan session row: %w", err)
		}

		session, err := s.buildSession(id, projectKey, status, createdAt, updatedAt, lastActivityAt)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate session rows: %w", err)
	}

	return sessions, nil
}

func (s *SQLiteSessionStorage) buildSession(id, projectKey, status string, createdAt, updatedAt, lastActivityAt int64) (*domain.Session, error) {
	session := &domain.Session{
		ID:             id,
		ProjectKey:     projectKey,
		Status:         domain.SessionStatus(status),
		CreatedAt:      time.Unix(createdAt, 0),
		UpdatedAt:      time.Unix(updatedAt, 0),
		LastActivityAt: time.Unix(lastActivityAt, 0),
	}

	return session, nil
}

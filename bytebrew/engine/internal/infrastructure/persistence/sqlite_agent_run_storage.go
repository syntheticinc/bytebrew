package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
)

const createAgentRunsTableSQL = `
CREATE TABLE IF NOT EXISTS agent_runs (
	id TEXT PRIMARY KEY,
	subtask_id TEXT NOT NULL,
	session_id TEXT NOT NULL,
	flow_type TEXT NOT NULL DEFAULT 'coder',
	status TEXT NOT NULL CHECK(status IN ('running','completed','failed','stopped')),
	result TEXT,
	error TEXT,
	started_at INTEGER NOT NULL,
	completed_at INTEGER
);

CREATE INDEX IF NOT EXISTS idx_agent_runs_session ON agent_runs(session_id);
CREATE INDEX IF NOT EXISTS idx_agent_runs_status ON agent_runs(session_id, status);
`

// SQLiteAgentRunStorage implements agent run persistence using SQLite
type SQLiteAgentRunStorage struct {
	db *sql.DB
}

// NewSQLiteAgentRunStorage creates a new agent run storage using the shared work DB.
// The caller is responsible for calling NewWorkDB first and passing the *sql.DB.
func NewSQLiteAgentRunStorage(db *sql.DB) (*SQLiteAgentRunStorage, error) {
	if _, err := db.Exec(createAgentRunsTableSQL); err != nil {
		return nil, fmt.Errorf("create agent_runs table: %w", err)
	}

	slog.Info("SQLite agent run storage initialized")
	return &SQLiteAgentRunStorage{db: db}, nil
}

// Save persists a new agent run
func (s *SQLiteAgentRunStorage) Save(ctx context.Context, run *domain.AgentRun) error {
	var completedAt *int64
	if run.CompletedAt != nil {
		v := run.CompletedAt.Unix()
		completedAt = &v
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO agent_runs (id, subtask_id, session_id, flow_type, status, result, error, started_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, run.ID, run.SubtaskID, run.SessionID, string(run.FlowType), string(run.Status),
		run.Result, run.Error, run.StartedAt.Unix(), completedAt)

	if err != nil {
		return fmt.Errorf("insert agent run: %w", err)
	}

	slog.DebugContext(ctx, "agent run saved", "agent_id", run.ID, "session_id", run.SessionID, "status", run.Status)
	return nil
}

// Update updates an existing agent run
func (s *SQLiteAgentRunStorage) Update(ctx context.Context, run *domain.AgentRun) error {
	var completedAt *int64
	if run.CompletedAt != nil {
		v := run.CompletedAt.Unix()
		completedAt = &v
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE agent_runs
		SET status = ?, result = ?, error = ?, completed_at = ?
		WHERE id = ?
	`, string(run.Status), run.Result, run.Error, completedAt, run.ID)

	if err != nil {
		return fmt.Errorf("update agent run: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("agent run not found: %s", run.ID)
	}

	slog.DebugContext(ctx, "agent run updated", "agent_id", run.ID, "status", run.Status)
	return nil
}

// GetByID retrieves an agent run by ID
func (s *SQLiteAgentRunStorage) GetByID(ctx context.Context, id string) (*domain.AgentRun, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, subtask_id, session_id, flow_type, status, result, error, started_at, completed_at
		FROM agent_runs WHERE id = ?
	`, id)

	return s.scanAgentRun(row)
}

// GetBySessionID retrieves all agent runs for a session
func (s *SQLiteAgentRunStorage) GetBySessionID(ctx context.Context, sessionID string) ([]*domain.AgentRun, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, subtask_id, session_id, flow_type, status, result, error, started_at, completed_at
		FROM agent_runs WHERE session_id = ?
		ORDER BY started_at DESC
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query agent runs by session: %w", err)
	}
	defer rows.Close()

	return s.scanAgentRuns(rows)
}

// GetRunningBySession retrieves all running agent runs for a session
func (s *SQLiteAgentRunStorage) GetRunningBySession(ctx context.Context, sessionID string) ([]*domain.AgentRun, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, subtask_id, session_id, flow_type, status, result, error, started_at, completed_at
		FROM agent_runs WHERE session_id = ? AND status = 'running'
		ORDER BY started_at ASC
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query running agent runs: %w", err)
	}
	defer rows.Close()

	return s.scanAgentRuns(rows)
}

// CountRunningBySession counts running agent runs for a session
func (s *SQLiteAgentRunStorage) CountRunningBySession(ctx context.Context, sessionID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM agent_runs WHERE session_id = ? AND status = 'running'
	`, sessionID).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("count running agent runs: %w", err)
	}

	return count, nil
}

// CleanupOrphanedRuns marks all 'running' agent_runs as 'stopped'.
// Called at server startup — after crash, these agents are dead.
func (s *SQLiteAgentRunStorage) CleanupOrphanedRuns(ctx context.Context) (int64, error) {
	now := time.Now().Unix()
	result, err := s.db.ExecContext(ctx,
		`UPDATE agent_runs SET status = 'stopped', completed_at = ? WHERE status = 'running'`,
		now)
	if err != nil {
		return 0, fmt.Errorf("cleanup orphaned runs: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}

	return affected, nil
}

// Close is a no-op because the shared DB is owned by the caller
func (s *SQLiteAgentRunStorage) Close() error {
	return nil
}

// scanAgentRun scans a single agent run from a row
func (s *SQLiteAgentRunStorage) scanAgentRun(row *sql.Row) (*domain.AgentRun, error) {
	var (
		id          string
		subtaskID   string
		sessionID   string
		flowType    string
		status      string
		result      sql.NullString
		errMsg      sql.NullString
		startedAt   int64
		completedAt sql.NullInt64
	)

	err := row.Scan(&id, &subtaskID, &sessionID, &flowType, &status,
		&result, &errMsg, &startedAt, &completedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan agent run: %w", err)
	}

	return s.buildAgentRun(id, subtaskID, sessionID, flowType, status,
		result, errMsg, startedAt, completedAt)
}

// scanAgentRuns scans multiple agent runs from rows
func (s *SQLiteAgentRunStorage) scanAgentRuns(rows *sql.Rows) ([]*domain.AgentRun, error) {
	var runs []*domain.AgentRun

	for rows.Next() {
		var (
			id          string
			subtaskID   string
			sessionID   string
			flowType    string
			status      string
			result      sql.NullString
			errMsg      sql.NullString
			startedAt   int64
			completedAt sql.NullInt64
		)

		if err := rows.Scan(&id, &subtaskID, &sessionID, &flowType, &status,
			&result, &errMsg, &startedAt, &completedAt); err != nil {
			return nil, fmt.Errorf("scan agent run row: %w", err)
		}

		run, err := s.buildAgentRun(id, subtaskID, sessionID, flowType, status,
			result, errMsg, startedAt, completedAt)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agent run rows: %w", err)
	}

	return runs, nil
}

func (s *SQLiteAgentRunStorage) buildAgentRun(id, subtaskID, sessionID, flowType, status string,
	result, errMsg sql.NullString, startedAt int64, completedAt sql.NullInt64) (*domain.AgentRun, error) {

	run := &domain.AgentRun{
		ID:        id,
		SubtaskID: subtaskID,
		SessionID: sessionID,
		FlowType:  domain.FlowType(flowType),
		Status:    domain.AgentRunStatus(status),
		StartedAt: time.Unix(startedAt, 0),
	}

	if result.Valid {
		run.Result = result.String
	}

	if errMsg.Valid {
		run.Error = errMsg.String
	}

	if completedAt.Valid {
		t := time.Unix(completedAt.Int64, 0)
		run.CompletedAt = &t
	}

	return run, nil
}

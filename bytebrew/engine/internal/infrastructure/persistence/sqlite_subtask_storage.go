package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
)

const createSubtasksTableSQL = `
CREATE TABLE IF NOT EXISTS subtasks (
	id TEXT PRIMARY KEY,
	session_id TEXT NOT NULL,
	task_id TEXT NOT NULL REFERENCES tasks(id),
	title TEXT NOT NULL,
	description TEXT,
	status TEXT NOT NULL CHECK(status IN ('pending', 'in_progress', 'waiting_for_input', 'completed', 'failed', 'cancelled')),
	assigned_agent_id TEXT,
	blocked_by TEXT CHECK(blocked_by IS NULL OR json_valid(blocked_by)),
	files_involved TEXT CHECK(files_involved IS NULL OR json_valid(files_involved)),
	result TEXT,
	context TEXT CHECK(context IS NULL OR json_valid(context)),
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL,
	completed_at INTEGER
);

CREATE INDEX IF NOT EXISTS idx_subtasks_session ON subtasks(session_id);
CREATE INDEX IF NOT EXISTS idx_subtasks_task ON subtasks(task_id);
CREATE INDEX IF NOT EXISTS idx_subtasks_status ON subtasks(status);
CREATE INDEX IF NOT EXISTS idx_subtasks_agent ON subtasks(assigned_agent_id);
`

// SQLiteSubtaskStorage implements subtask persistence using SQLite
type SQLiteSubtaskStorage struct {
	db *sql.DB
}

// NewSQLiteSubtaskStorage creates a new subtask storage using the shared work DB.
// Tasks table must be created first (FK dependency).
func NewSQLiteSubtaskStorage(db *sql.DB) (*SQLiteSubtaskStorage, error) {
	if _, err := db.Exec(createSubtasksTableSQL); err != nil {
		return nil, fmt.Errorf("create subtasks table: %w", err)
	}

	slog.Info("SQLite subtask storage initialized")
	return &SQLiteSubtaskStorage{db: db}, nil
}

// Save persists a new subtask
func (s *SQLiteSubtaskStorage) Save(ctx context.Context, subtask *domain.Subtask) error {
	blockedByJSON, err := json.Marshal(subtask.BlockedBy)
	if err != nil {
		return fmt.Errorf("marshal blocked_by: %w", err)
	}

	filesJSON, err := json.Marshal(subtask.FilesInvolved)
	if err != nil {
		return fmt.Errorf("marshal files_involved: %w", err)
	}

	contextJSON, err := json.Marshal(subtask.Context)
	if err != nil {
		return fmt.Errorf("marshal context: %w", err)
	}

	var completedAt *int64
	if subtask.CompletedAt != nil {
		v := subtask.CompletedAt.Unix()
		completedAt = &v
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO subtasks (id, session_id, task_id, title, description, status,
		                      assigned_agent_id, blocked_by, files_involved, result, context,
		                      created_at, updated_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, subtask.ID, subtask.SessionID, subtask.TaskID, subtask.Title, subtask.Description,
		string(subtask.Status), subtask.AssignedAgentID,
		string(blockedByJSON), string(filesJSON), subtask.Result, string(contextJSON),
		subtask.CreatedAt.Unix(), subtask.UpdatedAt.Unix(), completedAt)

	if err != nil {
		if strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
			return fmt.Errorf("task %q does not exist. Create a task first using manage_tasks(action=create), then create subtasks for it", subtask.TaskID)
		}
		return fmt.Errorf("insert subtask: %w", err)
	}

	slog.DebugContext(ctx, "subtask saved", "subtask_id", subtask.ID, "task_id", subtask.TaskID)
	return nil
}

// Update updates an existing subtask
func (s *SQLiteSubtaskStorage) Update(ctx context.Context, subtask *domain.Subtask) error {
	blockedByJSON, err := json.Marshal(subtask.BlockedBy)
	if err != nil {
		return fmt.Errorf("marshal blocked_by: %w", err)
	}

	filesJSON, err := json.Marshal(subtask.FilesInvolved)
	if err != nil {
		return fmt.Errorf("marshal files_involved: %w", err)
	}

	contextJSON, err := json.Marshal(subtask.Context)
	if err != nil {
		return fmt.Errorf("marshal context: %w", err)
	}

	var completedAt *int64
	if subtask.CompletedAt != nil {
		v := subtask.CompletedAt.Unix()
		completedAt = &v
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE subtasks
		SET title = ?, description = ?, status = ?, assigned_agent_id = ?,
		    blocked_by = ?, files_involved = ?, result = ?, context = ?,
		    updated_at = ?, completed_at = ?
		WHERE id = ?
	`, subtask.Title, subtask.Description, string(subtask.Status), subtask.AssignedAgentID,
		string(blockedByJSON), string(filesJSON), subtask.Result, string(contextJSON),
		subtask.UpdatedAt.Unix(), completedAt, subtask.ID)

	if err != nil {
		return fmt.Errorf("update subtask: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("subtask not found: %s", subtask.ID)
	}

	slog.DebugContext(ctx, "subtask updated", "subtask_id", subtask.ID, "status", subtask.Status)
	return nil
}

// GetByID retrieves a subtask by ID
func (s *SQLiteSubtaskStorage) GetByID(ctx context.Context, id string) (*domain.Subtask, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, session_id, task_id, title, description, status,
		       assigned_agent_id, blocked_by, files_involved, result, context,
		       created_at, updated_at, completed_at
		FROM subtasks WHERE id = ?
	`, id)

	return s.scanSubtask(row)
}

// GetByTaskID retrieves all subtasks for a task
func (s *SQLiteSubtaskStorage) GetByTaskID(ctx context.Context, taskID string) ([]*domain.Subtask, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, session_id, task_id, title, description, status,
		       assigned_agent_id, blocked_by, files_involved, result, context,
		       created_at, updated_at, completed_at
		FROM subtasks WHERE task_id = ?
		ORDER BY created_at ASC
	`, taskID)
	if err != nil {
		return nil, fmt.Errorf("query subtasks by task: %w", err)
	}
	defer rows.Close()

	return s.scanSubtasks(rows)
}

// GetBySessionID retrieves all subtasks for a session
func (s *SQLiteSubtaskStorage) GetBySessionID(ctx context.Context, sessionID string) ([]*domain.Subtask, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, session_id, task_id, title, description, status,
		       assigned_agent_id, blocked_by, files_involved, result, context,
		       created_at, updated_at, completed_at
		FROM subtasks WHERE session_id = ?
		ORDER BY created_at ASC
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query subtasks by session: %w", err)
	}
	defer rows.Close()

	return s.scanSubtasks(rows)
}

// GetReadySubtasks returns subtasks that are pending and have no unfinished blockers
func (s *SQLiteSubtaskStorage) GetReadySubtasks(ctx context.Context, taskID string) ([]*domain.Subtask, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, session_id, task_id, title, description, status,
		       assigned_agent_id, blocked_by, files_involved, result, context,
		       created_at, updated_at, completed_at
		FROM subtasks
		WHERE task_id = ? AND status = 'pending'
		  AND NOT EXISTS (
		      SELECT 1 FROM json_each(subtasks.blocked_by) AS jb
		      JOIN subtasks AS bt ON bt.id = jb.value
		      WHERE bt.status != 'completed'
		  )
		ORDER BY created_at ASC
	`, taskID)
	if err != nil {
		return nil, fmt.Errorf("query ready subtasks: %w", err)
	}
	defer rows.Close()

	return s.scanSubtasks(rows)
}

// GetByAgentID retrieves the subtask assigned to a specific agent
func (s *SQLiteSubtaskStorage) GetByAgentID(ctx context.Context, agentID string) (*domain.Subtask, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, session_id, task_id, title, description, status,
		       assigned_agent_id, blocked_by, files_involved, result, context,
		       created_at, updated_at, completed_at
		FROM subtasks WHERE assigned_agent_id = ? AND status = 'in_progress'
		LIMIT 1
	`, agentID)

	return s.scanSubtask(row)
}

// Close is a no-op because the shared DB is owned by the caller
func (s *SQLiteSubtaskStorage) Close() error {
	return nil
}

// scanSubtask scans a single subtask from a row
func (s *SQLiteSubtaskStorage) scanSubtask(row *sql.Row) (*domain.Subtask, error) {
	var (
		id            string
		sessionID     string
		taskID        string
		title         string
		description   sql.NullString
		status        string
		agentID       sql.NullString
		blockedByJSON sql.NullString
		filesJSON     sql.NullString
		result        sql.NullString
		contextJSON   sql.NullString
		createdAt     int64
		updatedAt     int64
		completedAt   sql.NullInt64
	)

	err := row.Scan(&id, &sessionID, &taskID, &title, &description, &status,
		&agentID, &blockedByJSON, &filesJSON, &result, &contextJSON,
		&createdAt, &updatedAt, &completedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan subtask: %w", err)
	}

	return s.buildSubtask(id, sessionID, taskID, title, description, status, agentID,
		blockedByJSON, filesJSON, result, contextJSON, createdAt, updatedAt, completedAt)
}

// scanSubtasks scans multiple subtasks from rows
func (s *SQLiteSubtaskStorage) scanSubtasks(rows *sql.Rows) ([]*domain.Subtask, error) {
	var subtasks []*domain.Subtask

	for rows.Next() {
		var (
			id            string
			sessionID     string
			taskID        string
			title         string
			description   sql.NullString
			status        string
			agentID       sql.NullString
			blockedByJSON sql.NullString
			filesJSON     sql.NullString
			result        sql.NullString
			contextJSON   sql.NullString
			createdAt     int64
			updatedAt     int64
			completedAt   sql.NullInt64
		)

		if err := rows.Scan(&id, &sessionID, &taskID, &title, &description, &status,
			&agentID, &blockedByJSON, &filesJSON, &result, &contextJSON,
			&createdAt, &updatedAt, &completedAt); err != nil {
			return nil, fmt.Errorf("scan subtask row: %w", err)
		}

		subtask, err := s.buildSubtask(id, sessionID, taskID, title, description, status, agentID,
			blockedByJSON, filesJSON, result, contextJSON, createdAt, updatedAt, completedAt)
		if err != nil {
			return nil, err
		}
		subtasks = append(subtasks, subtask)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate subtask rows: %w", err)
	}

	return subtasks, nil
}

func (s *SQLiteSubtaskStorage) buildSubtask(id, sessionID, taskID, title string, description sql.NullString, status string, agentID, blockedByJSON, filesJSON, result, contextJSON sql.NullString, createdAt, updatedAt int64, completedAt sql.NullInt64) (*domain.Subtask, error) {
	subtask := &domain.Subtask{
		ID:        id,
		SessionID: sessionID,
		TaskID:    taskID,
		Title:     title,
		Status:    domain.SubtaskStatus(status),
		CreatedAt: time.Unix(createdAt, 0),
		UpdatedAt: time.Unix(updatedAt, 0),
	}

	if description.Valid {
		subtask.Description = description.String
	}

	if agentID.Valid {
		subtask.AssignedAgentID = agentID.String
	}

	if blockedByJSON.Valid && blockedByJSON.String != "" && blockedByJSON.String != "null" {
		if err := json.Unmarshal([]byte(blockedByJSON.String), &subtask.BlockedBy); err != nil {
			return nil, fmt.Errorf("unmarshal blocked_by: %w", err)
		}
	}

	if filesJSON.Valid && filesJSON.String != "" && filesJSON.String != "null" {
		if err := json.Unmarshal([]byte(filesJSON.String), &subtask.FilesInvolved); err != nil {
			return nil, fmt.Errorf("unmarshal files_involved: %w", err)
		}
	}

	if result.Valid {
		subtask.Result = result.String
	}

	if contextJSON.Valid && contextJSON.String != "" && contextJSON.String != "null" {
		subtask.Context = make(map[string]string)
		if err := json.Unmarshal([]byte(contextJSON.String), &subtask.Context); err != nil {
			return nil, fmt.Errorf("unmarshal context: %w", err)
		}
	}

	if completedAt.Valid {
		t := time.Unix(completedAt.Int64, 0)
		subtask.CompletedAt = &t
	}

	return subtask, nil
}

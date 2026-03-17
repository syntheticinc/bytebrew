package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
)

const createTasksTableSQL = `
CREATE TABLE IF NOT EXISTS tasks (
	id TEXT PRIMARY KEY,
	session_id TEXT NOT NULL,
	title TEXT NOT NULL,
	description TEXT,
	acceptance_criteria TEXT CHECK(acceptance_criteria IS NULL OR json_valid(acceptance_criteria)),
	status TEXT NOT NULL CHECK(status IN ('draft', 'approved', 'in_progress', 'completed', 'failed', 'cancelled')),
	priority INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL,
	approved_at INTEGER,
	completed_at INTEGER
);

CREATE INDEX IF NOT EXISTS idx_tasks_session ON tasks(session_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
`

// SQLiteTaskStorage implements task persistence using SQLite
type SQLiteTaskStorage struct {
	db *sql.DB
}

// NewSQLiteTaskStorage creates a new task storage using the shared work DB.
// The caller is responsible for calling NewWorkDB first and passing the *sql.DB.
func NewSQLiteTaskStorage(db *sql.DB) (*SQLiteTaskStorage, error) {
	if _, err := db.Exec(createTasksTableSQL); err != nil {
		return nil, fmt.Errorf("create tasks table: %w", err)
	}

	// Safe migration: add priority column if it doesn't exist (for existing DBs)
	var count int
	row := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('tasks') WHERE name='priority'")
	_ = row.Scan(&count)
	if count == 0 {
		_, _ = db.Exec("ALTER TABLE tasks ADD COLUMN priority INTEGER NOT NULL DEFAULT 0")
		slog.Info("migrated tasks table: added priority column")
	}

	slog.Info("SQLite task storage initialized")
	return &SQLiteTaskStorage{db: db}, nil
}

// Save persists a new task
func (s *SQLiteTaskStorage) Save(ctx context.Context, task *domain.Task) error {
	criteriaJSON, err := json.Marshal(task.AcceptanceCriteria)
	if err != nil {
		return fmt.Errorf("marshal acceptance_criteria: %w", err)
	}

	var approvedAt *int64
	if task.ApprovedAt != nil {
		v := task.ApprovedAt.Unix()
		approvedAt = &v
	}

	var completedAt *int64
	if task.CompletedAt != nil {
		v := task.CompletedAt.Unix()
		completedAt = &v
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO tasks (id, session_id, title, description, acceptance_criteria, status, priority, created_at, updated_at, approved_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, task.ID, task.SessionID, task.Title, task.Description,
		string(criteriaJSON), string(task.Status), task.Priority,
		task.CreatedAt.Unix(), task.UpdatedAt.Unix(),
		approvedAt, completedAt)

	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}

	slog.DebugContext(ctx, "task saved", "task_id", task.ID, "session_id", task.SessionID)
	return nil
}

// Update updates an existing task
func (s *SQLiteTaskStorage) Update(ctx context.Context, task *domain.Task) error {
	criteriaJSON, err := json.Marshal(task.AcceptanceCriteria)
	if err != nil {
		return fmt.Errorf("marshal acceptance_criteria: %w", err)
	}

	var approvedAt *int64
	if task.ApprovedAt != nil {
		v := task.ApprovedAt.Unix()
		approvedAt = &v
	}

	var completedAt *int64
	if task.CompletedAt != nil {
		v := task.CompletedAt.Unix()
		completedAt = &v
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE tasks
		SET title = ?, description = ?, acceptance_criteria = ?, status = ?,
		    priority = ?, updated_at = ?, approved_at = ?, completed_at = ?
		WHERE id = ?
	`, task.Title, task.Description, string(criteriaJSON), string(task.Status),
		task.Priority, task.UpdatedAt.Unix(), approvedAt, completedAt, task.ID)

	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("task not found: %s", task.ID)
	}

	slog.DebugContext(ctx, "task updated", "task_id", task.ID, "status", task.Status)
	return nil
}

// GetByID retrieves a task by ID
func (s *SQLiteTaskStorage) GetByID(ctx context.Context, id string) (*domain.Task, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, session_id, title, description, acceptance_criteria, status,
		       priority, created_at, updated_at, approved_at, completed_at
		FROM tasks WHERE id = ?
	`, id)

	return s.scanTask(row)
}

// GetBySessionID retrieves all tasks for a session
func (s *SQLiteTaskStorage) GetBySessionID(ctx context.Context, sessionID string) ([]*domain.Task, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, session_id, title, description, acceptance_criteria, status,
		       priority, created_at, updated_at, approved_at, completed_at
		FROM tasks WHERE session_id = ?
		ORDER BY created_at DESC
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query tasks by session: %w", err)
	}
	defer rows.Close()

	return s.scanTasks(rows)
}

// GetByStatus retrieves tasks with a specific status for a session
func (s *SQLiteTaskStorage) GetByStatus(ctx context.Context, sessionID string, status domain.TaskStatus) ([]*domain.Task, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, session_id, title, description, acceptance_criteria, status,
		       priority, created_at, updated_at, approved_at, completed_at
		FROM tasks WHERE session_id = ? AND status = ?
		ORDER BY created_at DESC
	`, sessionID, string(status))
	if err != nil {
		return nil, fmt.Errorf("query tasks by status: %w", err)
	}
	defer rows.Close()

	return s.scanTasks(rows)
}

// GetBySessionIDOrdered retrieves tasks for a session, ordered by priority (DESC) then created_at (ASC)
func (s *SQLiteTaskStorage) GetBySessionIDOrdered(ctx context.Context, sessionID string) ([]*domain.Task, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, session_id, title, description, acceptance_criteria, status,
		       priority, created_at, updated_at, approved_at, completed_at
		FROM tasks
		WHERE session_id = ?
		ORDER BY priority DESC, created_at ASC
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query tasks by session ordered: %w", err)
	}
	defer rows.Close()

	return s.scanTasks(rows)
}

// Close is a no-op because the shared DB is owned by the caller
func (s *SQLiteTaskStorage) Close() error {
	return nil
}

// scanTask scans a single task from a row
func (s *SQLiteTaskStorage) scanTask(row *sql.Row) (*domain.Task, error) {
	var (
		id          string
		sessionID   string
		title       string
		description sql.NullString
		criteria    sql.NullString
		status      string
		priority    int
		createdAt   int64
		updatedAt   int64
		approvedAt  sql.NullInt64
		completedAt sql.NullInt64
	)

	err := row.Scan(&id, &sessionID, &title, &description, &criteria, &status,
		&priority, &createdAt, &updatedAt, &approvedAt, &completedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan task: %w", err)
	}

	return s.buildTask(id, sessionID, title, description, criteria, status, priority, createdAt, updatedAt, approvedAt, completedAt)
}

// scanTasks scans multiple tasks from rows
func (s *SQLiteTaskStorage) scanTasks(rows *sql.Rows) ([]*domain.Task, error) {
	var tasks []*domain.Task

	for rows.Next() {
		var (
			id          string
			sessionID   string
			title       string
			description sql.NullString
			criteria    sql.NullString
			status      string
			priority    int
			createdAt   int64
			updatedAt   int64
			approvedAt  sql.NullInt64
			completedAt sql.NullInt64
		)

		if err := rows.Scan(&id, &sessionID, &title, &description, &criteria, &status,
			&priority, &createdAt, &updatedAt, &approvedAt, &completedAt); err != nil {
			return nil, fmt.Errorf("scan task row: %w", err)
		}

		task, err := s.buildTask(id, sessionID, title, description, criteria, status, priority, createdAt, updatedAt, approvedAt, completedAt)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task rows: %w", err)
	}

	return tasks, nil
}

func (s *SQLiteTaskStorage) buildTask(id, sessionID, title string, description, criteria sql.NullString, status string, priority int, createdAt, updatedAt int64, approvedAt, completedAt sql.NullInt64) (*domain.Task, error) {
	task := &domain.Task{
		ID:        id,
		SessionID: sessionID,
		Title:     title,
		Status:    domain.TaskStatus(status),
		Priority:  priority,
		CreatedAt: time.Unix(createdAt, 0),
		UpdatedAt: time.Unix(updatedAt, 0),
	}

	if description.Valid {
		task.Description = description.String
	}

	if criteria.Valid && criteria.String != "" {
		if err := json.Unmarshal([]byte(criteria.String), &task.AcceptanceCriteria); err != nil {
			return nil, fmt.Errorf("unmarshal acceptance_criteria: %w", err)
		}
	}

	if approvedAt.Valid {
		t := time.Unix(approvedAt.Int64, 0)
		task.ApprovedAt = &t
	}

	if completedAt.Valid {
		t := time.Unix(completedAt.Int64, 0)
		task.CompletedAt = &t
	}

	return task, nil
}

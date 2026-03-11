package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
)

const (
	createTableSQL = `
	CREATE TABLE IF NOT EXISTS plans (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		goal TEXT NOT NULL,
		steps TEXT NOT NULL,
		status TEXT NOT NULL,
		metadata TEXT,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_plans_session_id ON plans(session_id);
	CREATE INDEX IF NOT EXISTS idx_plans_status ON plans(status);
	CREATE INDEX IF NOT EXISTS idx_plans_created_at ON plans(created_at DESC);
	`
)

// SQLitePlanStorage implements PlanStorage using SQLite
type SQLitePlanStorage struct {
	db *sql.DB
}

// NewSQLitePlanStorage creates a new SQLite-based plan storage
func NewSQLitePlanStorage(dbPath string) (*SQLitePlanStorage, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory for database: %w", err)
	}

	// Open database with WAL mode for better concurrency
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// Create tables and indexes
	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	slog.Info("SQLite plan storage initialized", "db_path", dbPath)

	return &SQLitePlanStorage{db: db}, nil
}

// Save persists a new plan to the database
func (s *SQLitePlanStorage) Save(ctx context.Context, plan *domain.Plan) error {
	// Serialize steps to JSON
	stepsJSON, err := json.Marshal(plan.Steps)
	if err != nil {
		return fmt.Errorf("failed to marshal steps: %w", err)
	}

	// Serialize metadata to JSON
	metadataJSON := "{}"
	if plan.Metadata != nil && len(plan.Metadata) > 0 {
		metadataBytes, err := json.Marshal(plan.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = string(metadataBytes)
	}

	// Insert plan
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO plans (id, session_id, goal, steps, status, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, plan.ID, plan.SessionID, plan.Goal, string(stepsJSON), string(plan.Status),
		metadataJSON, plan.CreatedAt.Unix(), plan.UpdatedAt.Unix())

	if err != nil {
		return fmt.Errorf("failed to insert plan: %w", err)
	}

	slog.DebugContext(ctx, "plan saved to SQLite",
		"plan_id", plan.ID,
		"session_id", plan.SessionID)

	return nil
}

// GetBySessionID retrieves the active plan for a session
func (s *SQLitePlanStorage) GetBySessionID(ctx context.Context, sessionID string) (*domain.Plan, error) {
	var (
		id         string
		goal       string
		stepsJSON  string
		status     string
		metadata   string
		createdAt  int64
		updatedAt  int64
	)

	// Query for active plan
	err := s.db.QueryRowContext(ctx, `
		SELECT id, goal, steps, status, metadata, created_at, updated_at
		FROM plans
		WHERE session_id = ? AND status IN ('draft', 'active')
		ORDER BY created_at DESC
		LIMIT 1
	`, sessionID).Scan(&id, &goal, &stepsJSON, &status, &metadata, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no active plan found for session %s", sessionID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query plan: %w", err)
	}

	// Deserialize steps
	var steps []*domain.PlanStep
	if err := json.Unmarshal([]byte(stepsJSON), &steps); err != nil {
		return nil, fmt.Errorf("failed to unmarshal steps: %w", err)
	}

	// Deserialize metadata
	var metadataMap domain.StringMap
	if metadata != "" && metadata != "{}" {
		if err := json.Unmarshal([]byte(metadata), &metadataMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	} else {
		metadataMap = make(domain.StringMap)
	}

	plan := &domain.Plan{
		ID:        id,
		SessionID: sessionID,
		Goal:      goal,
		Steps:     steps,
		Status:    domain.PlanStatus(status),
		Metadata:  metadataMap,
		CreatedAt: time.Unix(createdAt, 0),
		UpdatedAt: time.Unix(updatedAt, 0),
	}

	slog.DebugContext(ctx, "plan loaded from SQLite",
		"plan_id", plan.ID,
		"session_id", sessionID,
		"status", plan.Status)

	return plan, nil
}

// Update updates an existing plan
func (s *SQLitePlanStorage) Update(ctx context.Context, plan *domain.Plan) error {
	// Serialize steps to JSON
	stepsJSON, err := json.Marshal(plan.Steps)
	if err != nil {
		return fmt.Errorf("failed to marshal steps: %w", err)
	}

	// Serialize metadata to JSON
	metadataJSON := "{}"
	if plan.Metadata != nil && len(plan.Metadata) > 0 {
		metadataBytes, err := json.Marshal(plan.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = string(metadataBytes)
	}

	// Update plan
	result, err := s.db.ExecContext(ctx, `
		UPDATE plans
		SET goal = ?, steps = ?, status = ?, metadata = ?, updated_at = ?
		WHERE id = ?
	`, plan.Goal, string(stepsJSON), string(plan.Status), metadataJSON,
		plan.UpdatedAt.Unix(), plan.ID)

	if err != nil {
		return fmt.Errorf("failed to update plan: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("plan not found: %s", plan.ID)
	}

	slog.DebugContext(ctx, "plan updated in SQLite",
		"plan_id", plan.ID,
		"session_id", plan.SessionID,
		"status", plan.Status)

	return nil
}

// DeleteOldPlans deletes plans older than the specified duration
func (s *SQLitePlanStorage) DeleteOldPlans(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-olderThan).Unix()

	result, err := s.db.ExecContext(ctx, `
		DELETE FROM plans
		WHERE created_at < ? AND status IN ('completed', 'abandoned')
	`, cutoffTime)

	if err != nil {
		return 0, fmt.Errorf("failed to delete old plans: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if deleted > 0 {
		slog.InfoContext(ctx, "deleted old plans from SQLite",
			"count", deleted,
			"older_than", olderThan)
	}

	return deleted, nil
}

// Close closes the database connection
func (s *SQLitePlanStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// GetStats returns statistics about stored plans
func (s *SQLitePlanStorage) GetStats(ctx context.Context) (map[string]int, error) {
	stats := make(map[string]int)

	// Count plans by status
	rows, err := s.db.QueryContext(ctx, `
		SELECT status, COUNT(*) as count
		FROM plans
		GROUP BY status
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}
	defer rows.Close()

	total := 0
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan stats row: %w", err)
		}
		stats[status] = count
		total += count
	}

	stats["total"] = total

	return stats, nil
}

package persistence

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// NewWorkDB creates a shared SQLite database for tasks and subtasks.
// Both SQLiteTaskStorage and SQLiteSubtaskStorage share this *sql.DB.
// Foreign keys are enabled via DSN.
func NewWorkDB(dbPath string) (*sql.DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create directory for work db: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?cache=shared&mode=rwc&_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open work db: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// Verify FK support
	var fkEnabled int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled); err != nil {
		db.Close()
		return nil, fmt.Errorf("check foreign_keys pragma: %w", err)
	}
	if fkEnabled != 1 {
		db.Close()
		return nil, fmt.Errorf("foreign_keys not enabled (got %d)", fkEnabled)
	}

	slog.Info("work DB initialized", "db_path", dbPath, "foreign_keys", fkEnabled)
	return db, nil
}

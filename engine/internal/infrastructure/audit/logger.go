package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
)

// Logger writes audit log entries to the database.
type Logger struct {
	db *gorm.DB
}

// NewLogger creates a new audit Logger.
func NewLogger(db *gorm.DB) *Logger {
	return &Logger{db: db}
}

// Entry represents an audit event to be recorded.
type Entry struct {
	Timestamp time.Time
	ActorType string
	ActorID   string
	Action    string
	Resource  string
	Details   map[string]interface{}
	SessionID string
	TaskID    *string
}

// Log persists an audit entry to the database.
func (l *Logger) Log(ctx context.Context, entry Entry) error {
	detailsJSON, err := json.Marshal(entry.Details)
	if err != nil {
		slog.ErrorContext(ctx, "marshal audit details failed", "error", err)
		detailsJSON = []byte("{}")
	}

	var sessionID *string
	if entry.SessionID != "" {
		sessionID = &entry.SessionID
	}

	ts := entry.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}

	model := models.AuditLogModel{
		Timestamp: ts,
		ActorType: entry.ActorType,
		ActorID:   entry.ActorID,
		Action:    entry.Action,
		Resource:  entry.Resource,
		Details:   string(detailsJSON),
		SessionID: sessionID,
		TaskID:    entry.TaskID,
	}

	if err := l.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}

	return nil
}

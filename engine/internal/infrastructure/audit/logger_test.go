package audit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Discard,
	})
	require.NoError(t, err)

	// Create table manually to avoid PostgreSQL-specific syntax in GORM tags.
	err = db.Exec(`CREATE TABLE audit_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME,
		actor_type VARCHAR(20) NOT NULL,
		actor_id VARCHAR(255),
		action VARCHAR(50) NOT NULL,
		resource VARCHAR(500),
		details TEXT,
		session_id VARCHAR(36),
		task_id TEXT
	)`).Error
	require.NoError(t, err)

	return db
}

func TestLogger_Log(t *testing.T) {
	db := setupTestDB(t)
	logger := NewLogger(db)

	ts := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)
	sessionID := "session-123"

	err := logger.Log(context.Background(), Entry{
		Timestamp: ts,
		ActorType: "admin",
		ActorID:   "user-1",
		Action:    "api_call",
		Resource:  "GET /api/v1/agents",
		Details: map[string]interface{}{
			"method":      "GET",
			"status_code": 200,
		},
		SessionID: sessionID,
	})
	require.NoError(t, err)

	var result models.AuditLogModel
	require.NoError(t, db.First(&result).Error)

	assert.Equal(t, "admin", result.ActorType)
	assert.Equal(t, "user-1", result.ActorID)
	assert.Equal(t, "api_call", result.Action)
	assert.Equal(t, "GET /api/v1/agents", result.Resource)
	assert.Contains(t, result.Details, `"method":"GET"`)
	assert.Contains(t, result.Details, `"status_code":200`)
	require.NotNil(t, result.SessionID)
	assert.Equal(t, "session-123", *result.SessionID)
	assert.Nil(t, result.TaskID)
}

func TestLogger_Log_EmptySessionID(t *testing.T) {
	db := setupTestDB(t)
	logger := NewLogger(db)

	err := logger.Log(context.Background(), Entry{
		ActorType: "system",
		Action:    "config_change",
	})
	require.NoError(t, err)

	var result models.AuditLogModel
	require.NoError(t, db.First(&result).Error)
	assert.Nil(t, result.SessionID)
}

func TestLogger_Log_WithTaskID(t *testing.T) {
	db := setupTestDB(t)
	logger := NewLogger(db)

	taskID := "task-uuid-42"
	err := logger.Log(context.Background(), Entry{
		ActorType: "api_token",
		ActorID:   "bot-token",
		Action:    "task_created",
		Resource:  "POST /api/v1/tasks",
		TaskID:    &taskID,
	})
	require.NoError(t, err)

	var result models.AuditLogModel
	require.NoError(t, db.First(&result).Error)
	require.NotNil(t, result.TaskID)
	assert.Equal(t, "task-uuid-42", *result.TaskID)
}

func TestLogger_Log_ZeroTimestamp(t *testing.T) {
	db := setupTestDB(t)
	logger := NewLogger(db)

	before := time.Now()
	err := logger.Log(context.Background(), Entry{
		ActorType: "system",
		Action:    "config_change",
	})
	require.NoError(t, err)

	var result models.AuditLogModel
	require.NoError(t, db.First(&result).Error)
	assert.False(t, result.Timestamp.Before(before))
}

func TestLogger_Log_NilDetails(t *testing.T) {
	db := setupTestDB(t)
	logger := NewLogger(db)

	err := logger.Log(context.Background(), Entry{
		ActorType: "admin",
		Action:    "api_call",
		Details:   nil,
	})
	require.NoError(t, err)

	var result models.AuditLogModel
	require.NoError(t, db.First(&result).Error)
	assert.Equal(t, "null", result.Details)
}

func TestLogger_Log_MultipleEntries(t *testing.T) {
	db := setupTestDB(t)
	logger := NewLogger(db)

	for i := 0; i < 5; i++ {
		require.NoError(t, logger.Log(context.Background(), Entry{
			ActorType: "admin",
			Action:    "api_call",
		}))
	}

	var count int64
	db.Model(&models.AuditLogModel{}).Count(&count)
	assert.Equal(t, int64(5), count)
}

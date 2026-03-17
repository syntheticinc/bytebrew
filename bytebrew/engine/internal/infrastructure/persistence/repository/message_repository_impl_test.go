package repository

import (
	"context"
	"testing"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupMessageTestDB creates in-memory SQLite DB for message tests
func setupMessageTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "failed to open in-memory SQLite")

	// Create table manually for SQLite compatibility
	err = db.Exec(`
		CREATE TABLE message (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			message_type TEXT NOT NULL,
			sender TEXT,
			agent_id TEXT,
			content TEXT NOT NULL,
			metadata TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	require.NoError(t, err, "failed to create table")

	return db
}

func TestMessageRepositoryImpl_CreateAndGetBySessionID(t *testing.T) {
	db := setupMessageTestDB(t)
	repo := NewMessageRepositoryImpl(db)
	ctx := context.Background()

	sessionID := uuid.New().String()
	baseTime := time.Now().Add(-1 * time.Hour)

	// Create 3 messages with explicit timestamps
	msg1, err := domain.NewMessage(sessionID, domain.MessageTypeUser, "user", "First message")
	require.NoError(t, err)
	msg1.AgentID = "supervisor"
	msg1.CreatedAt = baseTime

	msg2, err := domain.NewMessage(sessionID, domain.MessageTypeAgent, "assistant", "Second message")
	require.NoError(t, err)
	msg2.AgentID = "supervisor"
	msg2.CreatedAt = baseTime.Add(1 * time.Second)

	msg3, err := domain.NewMessage(sessionID, domain.MessageTypeUser, "user", "Third message")
	require.NoError(t, err)
	msg3.AgentID = "supervisor"
	msg3.CreatedAt = baseTime.Add(2 * time.Second)

	// Save messages
	err = repo.Create(ctx, msg1)
	require.NoError(t, err)

	err = repo.Create(ctx, msg2)
	require.NoError(t, err)

	err = repo.Create(ctx, msg3)
	require.NoError(t, err)

	// GetBySessionID should return all 3 messages in chronological order
	messages, err := repo.GetBySessionID(ctx, sessionID, 0, 0)
	require.NoError(t, err)
	require.Len(t, messages, 3, "should return 3 messages")

	// Verify chronological order (oldest first)
	assert.Equal(t, "First message", messages[0].Content)
	assert.Equal(t, "Second message", messages[1].Content)
	assert.Equal(t, "Third message", messages[2].Content)

	// Verify all fields
	for _, msg := range messages {
		assert.Equal(t, sessionID, msg.SessionID)
		assert.NotEmpty(t, msg.ID)
		assert.Equal(t, "supervisor", msg.AgentID)
	}
}

func TestMessageRepositoryImpl_GetBySessionAndAgent(t *testing.T) {
	db := setupMessageTestDB(t)
	repo := NewMessageRepositoryImpl(db)
	ctx := context.Background()

	sessionID := uuid.New().String()
	baseTime := time.Now().Add(-1 * time.Hour)

	// Create 3 messages: 2 for agent-1, 1 for agent-2 with explicit timestamps
	msg1, err := domain.NewMessage(sessionID, domain.MessageTypeUser, "user", "Message 1 for agent-1")
	require.NoError(t, err)
	msg1.AgentID = "agent-1"
	msg1.CreatedAt = baseTime

	msg2, err := domain.NewMessage(sessionID, domain.MessageTypeAgent, "assistant", "Message 2 for agent-2")
	require.NoError(t, err)
	msg2.AgentID = "agent-2"
	msg2.CreatedAt = baseTime.Add(1 * time.Second)

	msg3, err := domain.NewMessage(sessionID, domain.MessageTypeUser, "user", "Message 3 for agent-1")
	require.NoError(t, err)
	msg3.AgentID = "agent-1"
	msg3.CreatedAt = baseTime.Add(2 * time.Second)

	// Save messages
	err = repo.Create(ctx, msg1)
	require.NoError(t, err)

	err = repo.Create(ctx, msg2)
	require.NoError(t, err)

	err = repo.Create(ctx, msg3)
	require.NoError(t, err)

	// GetBySessionAndAgent for agent-1 should return 2 messages
	messages, err := repo.GetBySessionAndAgent(ctx, sessionID, "agent-1", 0, 0)
	require.NoError(t, err)
	require.Len(t, messages, 2, "should return 2 messages for agent-1")

	// Verify both messages belong to agent-1
	for _, msg := range messages {
		assert.Equal(t, "agent-1", msg.AgentID)
		assert.Contains(t, msg.Content, "agent-1")
	}

	// Verify chronological order
	assert.Equal(t, "Message 1 for agent-1", messages[0].Content)
	assert.Equal(t, "Message 3 for agent-1", messages[1].Content)

	// GetBySessionAndAgent for agent-2 should return 1 message
	messages2, err := repo.GetBySessionAndAgent(ctx, sessionID, "agent-2", 0, 0)
	require.NoError(t, err)
	require.Len(t, messages2, 1, "should return 1 message for agent-2")
	assert.Equal(t, "agent-2", messages2[0].AgentID)
	assert.Equal(t, "Message 2 for agent-2", messages2[0].Content)
}

func TestMessageRepositoryImpl_EmptySession(t *testing.T) {
	db := setupMessageTestDB(t)
	repo := NewMessageRepositoryImpl(db)
	ctx := context.Background()

	sessionID := uuid.New().String()

	// GetBySessionID for empty session should return empty slice, not error
	messages, err := repo.GetBySessionID(ctx, sessionID, 0, 0)
	require.NoError(t, err, "should not return error for empty session")
	assert.Empty(t, messages, "should return empty slice")

	// GetBySessionAndAgent for empty session should also return empty slice
	messages2, err := repo.GetBySessionAndAgent(ctx, sessionID, "agent-1", 0, 0)
	require.NoError(t, err, "should not return error for empty session")
	assert.Empty(t, messages2, "should return empty slice")
}

func TestMessageRepositoryImpl_LimitAndOffset(t *testing.T) {
	db := setupMessageTestDB(t)
	repo := NewMessageRepositoryImpl(db)
	ctx := context.Background()

	sessionID := uuid.New().String()

	// Create 5 messages
	for i := 1; i <= 5; i++ {
		msg, err := domain.NewMessage(sessionID, domain.MessageTypeUser, "user", "Message "+string(rune('0'+i)))
		require.NoError(t, err)
		msg.AgentID = "supervisor"
		err = repo.Create(ctx, msg)
		require.NoError(t, err)
	}

	// Test limit
	messages, err := repo.GetBySessionID(ctx, sessionID, 3, 0)
	require.NoError(t, err)
	assert.Len(t, messages, 3, "should return last 3 messages")

	// Test offset
	messages2, err := repo.GetBySessionID(ctx, sessionID, 0, 2)
	require.NoError(t, err)
	assert.Len(t, messages2, 3, "should return 3 messages after skipping 2")

	// Test limit + offset
	messages3, err := repo.GetBySessionID(ctx, sessionID, 2, 1)
	require.NoError(t, err)
	assert.Len(t, messages3, 2, "should return 2 messages after skipping 1")
}

func TestMessageRepositoryImpl_ToolCallsSerialization(t *testing.T) {
	db := setupMessageTestDB(t)
	repo := NewMessageRepositoryImpl(db)
	ctx := context.Background()

	sessionID := uuid.New().String()

	// Create message with tool calls
	toolCalls := []domain.ToolCallInfo{
		{
			ID:        "call-1",
			Name:      "read_file",
			Arguments: map[string]string{"path": "test.go"},
		},
		{
			ID:        "call-2",
			Name:      "search_code",
			Arguments: map[string]string{"query": "func main"},
		},
	}

	msg, err := domain.NewAssistantMessageWithToolCalls(sessionID, "Using tools", toolCalls)
	require.NoError(t, err)
	msg.AgentID = "supervisor"

	// Save message
	err = repo.Create(ctx, msg)
	require.NoError(t, err)

	// Load message
	messages, err := repo.GetBySessionID(ctx, sessionID, 0, 0)
	require.NoError(t, err)
	require.Len(t, messages, 1)

	loaded := messages[0]

	// Verify tool calls deserialized correctly
	require.Len(t, loaded.ToolCalls, 2)
	assert.Equal(t, "call-1", loaded.ToolCalls[0].ID)
	assert.Equal(t, "read_file", loaded.ToolCalls[0].Name)
	assert.Equal(t, "test.go", loaded.ToolCalls[0].Arguments["path"])

	assert.Equal(t, "call-2", loaded.ToolCalls[1].ID)
	assert.Equal(t, "search_code", loaded.ToolCalls[1].Name)
	assert.Equal(t, "func main", loaded.ToolCalls[1].Arguments["query"])
}

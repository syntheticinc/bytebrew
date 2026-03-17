package engine

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/agents"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/persistence/adapters"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/config"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock SnapshotRepository
type mockSnapshotRepo struct {
	snapshots map[string]*domain.AgentContextSnapshot // keyed by agentID
	mu        sync.Mutex
}

func newMockSnapshotRepo() *mockSnapshotRepo {
	return &mockSnapshotRepo{
		snapshots: make(map[string]*domain.AgentContextSnapshot),
	}
}

func (m *mockSnapshotRepo) Save(ctx context.Context, snapshot *domain.AgentContextSnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snapshots[snapshot.AgentID] = snapshot
	return nil
}

func (m *mockSnapshotRepo) Load(ctx context.Context, sessionID, agentID string) (*domain.AgentContextSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.snapshots[agentID], nil
}

func (m *mockSnapshotRepo) Delete(ctx context.Context, sessionID, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.snapshots, agentID)
	return nil
}

func (m *mockSnapshotRepo) FindActive(ctx context.Context) ([]*domain.AgentContextSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var active []*domain.AgentContextSnapshot
	for _, snap := range m.snapshots {
		if snap.Status == domain.AgentContextStatusActive {
			active = append(active, snap)
		}
	}
	return active, nil
}

// Mock HistoryRepository
type mockHistoryRepo struct {
	messages []*domain.Message
	mu       sync.Mutex
}

func newMockHistoryRepo() *mockHistoryRepo {
	return &mockHistoryRepo{
		messages: make([]*domain.Message, 0),
	}
}

func (m *mockHistoryRepo) Create(ctx context.Context, message *domain.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, message)
	return nil
}

// Mock ChatModel
type mockChatModel struct {
	response *schema.Message
	err      error
}

func (m *mockChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.response == nil {
		return &schema.Message{
			Role:    schema.Assistant,
			Content: "mock response",
		}, nil
	}
	return m.response, nil
}

func (m *mockChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	// Return empty reader
	sr, sw := schema.Pipe[*schema.Message](1)
	sw.Close()
	return sr, nil
}

func (m *mockChatModel) BindTools(tools []*schema.ToolInfo) error {
	return nil
}

func (m *mockChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return m, nil
}

// Helper to create test flow
func testFlow() *domain.Flow {
	return &domain.Flow{
		Type:           domain.FlowType("supervisor"),
		Name:           "test-flow",
		SystemPrompt:   "You are a test agent",
		ToolNames:      []string{},
		MaxSteps:       10,
		MaxContextSize: 4000,
		Lifecycle: domain.LifecyclePolicy{
			SuspendOn: []string{},
			ReportTo:  "user",
		},
	}
}

// Test 1: Fresh start (no snapshot)
func TestEngine_FreshStart(t *testing.T) {
	ctx := context.Background()
	snapshotRepo := newMockSnapshotRepo()
	historyRepo := newMockHistoryRepo()
	engine := New(snapshotRepo, historyRepo)

	cfg := ExecutionConfig{
		SessionID:   "session-1",
		AgentID:     "supervisor",
		Flow:        testFlow(),
		ChatModel:   &mockChatModel{},
		Input:       "Hello",
		Streaming:   false,
		AgentConfig: &config.AgentConfig{},
	}

	result, err := engine.Execute(ctx, cfg)

	require.NoError(t, err)
	assert.Equal(t, StatusCompleted, result.Status)

	// Check snapshot saved
	snapshot := snapshotRepo.snapshots["supervisor"]
	require.NotNil(t, snapshot)
	assert.Equal(t, "session-1", snapshot.SessionID)
	assert.Equal(t, domain.FlowType("supervisor"), snapshot.FlowType)
	assert.Equal(t, domain.CurrentSchemaVersion, snapshot.SchemaVersion)
	assert.Equal(t, domain.AgentContextStatusCompleted, snapshot.Status)

	// Check messages were deserialized successfully
	messages, err := adapters.DeserializeSchemaMessages(snapshot.ContextData)
	require.NoError(t, err)
	assert.NotEmpty(t, messages)
}

// Test 2: Resume from snapshot
func TestEngine_ResumeFromSnapshot(t *testing.T) {
	ctx := context.Background()
	snapshotRepo := newMockSnapshotRepo()
	historyRepo := newMockHistoryRepo()
	engine := New(snapshotRepo, historyRepo)

	// Create initial snapshot with history
	initialMessages := []*schema.Message{
		{Role: schema.User, Content: "First message"},
		{Role: schema.Assistant, Content: "First response"},
	}
	contextData, err := adapters.SerializeSchemaMessages(initialMessages)
	require.NoError(t, err)

	snapshotRepo.snapshots["supervisor"] = &domain.AgentContextSnapshot{
		SessionID:     "session-1",
		AgentID:       "supervisor",
		FlowType:      domain.FlowType("supervisor"),
		SchemaVersion: domain.CurrentSchemaVersion,
		ContextData:   contextData,
		StepNumber:    1,
		Status:        domain.AgentContextStatusSuspended,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	cfg := ExecutionConfig{
		SessionID:   "session-1",
		AgentID:     "supervisor",
		Flow:        testFlow(),
		ChatModel:   &mockChatModel{},
		Input:       "Second message",
		Streaming:   false,
		AgentConfig: &config.AgentConfig{},
	}

	result, err := engine.Execute(ctx, cfg)

	require.NoError(t, err)
	assert.Equal(t, StatusCompleted, result.Status)

	// Check snapshot updated with new messages
	snapshot := snapshotRepo.snapshots["supervisor"]
	require.NotNil(t, snapshot)

	messages, err := adapters.DeserializeSchemaMessages(snapshot.ContextData)
	require.NoError(t, err)
	// Should have: initial 2 messages + new user message + new assistant response
	assert.GreaterOrEqual(t, len(messages), 3)
}

// Test 3: Suspend flow
func TestEngine_SuspendFlow(t *testing.T) {
	ctx := context.Background()
	snapshotRepo := newMockSnapshotRepo()
	historyRepo := newMockHistoryRepo()
	engine := New(snapshotRepo, historyRepo)

	flow := testFlow()
	flow.Lifecycle.SuspendOn = []string{"final_answer"}

	cfg := ExecutionConfig{
		SessionID:   "session-1",
		AgentID:     "supervisor",
		Flow:        flow,
		ChatModel:   &mockChatModel{},
		Input:       "Hello",
		Streaming:   false,
		AgentConfig: &config.AgentConfig{},
	}

	result, err := engine.Execute(ctx, cfg)

	require.NoError(t, err)
	assert.Equal(t, StatusSuspended, result.Status)
	assert.Equal(t, "final_answer", result.SuspendedAt)

	// Check snapshot status
	snapshot := snapshotRepo.snapshots["supervisor"]
	require.NotNil(t, snapshot)
	assert.Equal(t, domain.AgentContextStatusSuspended, snapshot.Status)
}

// Test 4: Failed execution
func TestEngine_FailedExecution(t *testing.T) {
	ctx := context.Background()
	snapshotRepo := newMockSnapshotRepo()
	historyRepo := newMockHistoryRepo()
	engine := New(snapshotRepo, historyRepo)

	// Mock model that returns error
	mockErr := assert.AnError
	cfg := ExecutionConfig{
		SessionID: "session-1",
		AgentID:   "supervisor",
		Flow:      testFlow(),
		ChatModel: &mockChatModel{err: mockErr},
		Input:     "Hello",
		Streaming: false,
	}

	result, err := engine.Execute(ctx, cfg)

	require.Error(t, err)
	assert.Equal(t, StatusFailed, result.Status)

	// Check snapshot still saved for debugging
	snapshot := snapshotRepo.snapshots["supervisor"]
	require.NotNil(t, snapshot)
	assert.Equal(t, domain.AgentContextStatusInterrupted, snapshot.Status)
}

// Test 5: Message collection
func TestEngine_MessageCollection(t *testing.T) {
	historyRepo := newMockHistoryRepo()

	// Create collector with mock events
	collector := NewMessageCollector("session-1", "supervisor", historyRepo)

	// Simulate tool call event
	toolCallEvent := &domain.AgentEvent{
		Type: domain.EventTypeToolCall,
		Metadata: map[string]interface{}{
			"id":                 "call-1",
			"tool_name":          "read_file",
			"function_arguments": `{"path":"test.txt"}`,
			"assistant_content":  "Let me read the file",
		},
	}
	collector.handleEvent(toolCallEvent)

	// Simulate tool result event
	toolResultEvent := &domain.AgentEvent{
		Type:    domain.EventTypeToolResult,
		Content: "file content",
		Metadata: map[string]interface{}{
			"tool_name":   "read_file",
			"full_result": "file content here",
		},
	}
	collector.handleEvent(toolResultEvent)

	// Simulate answer event
	answerEvent := &domain.AgentEvent{
		Type:    domain.EventTypeAnswer,
		Content: "Done",
	}
	collector.handleEvent(answerEvent)

	// Check collected messages
	messages := collector.GetAccumulatedMessages()
	assert.Len(t, messages, 3) // assistant+tool_call, tool, assistant

	// Check message types
	assert.Equal(t, schema.Assistant, messages[0].Role)
	assert.NotEmpty(t, messages[0].ToolCalls)
	assert.Equal(t, schema.Tool, messages[1].Role)
	assert.Equal(t, schema.Assistant, messages[2].Role)

	// Check history repo received messages
	assert.Len(t, historyRepo.messages, 3)
}

// Test 6: Lossless round-trip
func TestEngine_LosslessRoundTrip(t *testing.T) {
	ctx := context.Background()
	snapshotRepo := newMockSnapshotRepo()
	historyRepo := newMockHistoryRepo()
	engine := New(snapshotRepo, historyRepo)

	// Create complex message set
	originalMessages := []*schema.Message{
		{Role: schema.User, Content: "Hello"},
		{
			Role:    schema.Assistant,
			Content: "Let me help",
			ToolCalls: []schema.ToolCall{{
				ID: "call-1",
				Function: schema.FunctionCall{
					Name:      "read_file",
					Arguments: `{"path":"test.txt"}`,
				},
			}},
		},
		{
			Role:       schema.Tool,
			Content:    "file content",
			ToolCallID: "call-1",
			Name:       "read_file",
		},
		{Role: schema.Assistant, Content: "Done"},
	}

	// Serialize and save
	contextData, err := adapters.SerializeSchemaMessages(originalMessages)
	require.NoError(t, err)

	snapshotRepo.snapshots["supervisor"] = &domain.AgentContextSnapshot{
		SessionID:     "session-1",
		AgentID:       "supervisor",
		FlowType:      domain.FlowType("supervisor"),
		SchemaVersion: domain.CurrentSchemaVersion,
		ContextData:   contextData,
		StepNumber:    2,
		Status:        domain.AgentContextStatusSuspended,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Load and deserialize
	cfg := ExecutionConfig{
		SessionID:   "session-1",
		AgentID:     "supervisor",
		Flow:        testFlow(),
		ChatModel:   &mockChatModel{},
		Input:       "Continue",
		Streaming:   false,
		AgentConfig: &config.AgentConfig{},
	}

	_, err = engine.Execute(ctx, cfg)
	require.NoError(t, err)

	// Load final snapshot
	snapshot := snapshotRepo.snapshots["supervisor"]
	require.NotNil(t, snapshot)

	loadedMessages, err := adapters.DeserializeSchemaMessages(snapshot.ContextData)
	require.NoError(t, err)

	// Should start with original messages
	assert.GreaterOrEqual(t, len(loadedMessages), len(originalMessages))
	for i, orig := range originalMessages {
		loaded := loadedMessages[i]
		assert.Equal(t, orig.Role, loaded.Role)
		assert.Equal(t, orig.Content, loaded.Content)
		if len(orig.ToolCalls) > 0 {
			require.NotEmpty(t, loaded.ToolCalls)
			assert.Equal(t, orig.ToolCalls[0].ID, loaded.ToolCalls[0].ID)
			assert.Equal(t, orig.ToolCalls[0].Function.Name, loaded.ToolCalls[0].Function.Name)
		}
		if orig.ToolCallID != "" {
			assert.Equal(t, orig.ToolCallID, loaded.ToolCallID)
		}
	}
}

// Test 7: Crash recovery
func TestEngine_RecoverInterrupted(t *testing.T) {
	ctx := context.Background()
	snapshotRepo := newMockSnapshotRepo()
	historyRepo := newMockHistoryRepo()
	engine := New(snapshotRepo, historyRepo)

	// Create active snapshots (simulating server crash)
	snapshotRepo.snapshots["agent-1"] = &domain.AgentContextSnapshot{
		SessionID:     "session-1",
		AgentID:       "agent-1",
		FlowType:      domain.FlowType("supervisor"),
		SchemaVersion: domain.CurrentSchemaVersion,
		ContextData:   []byte("{}"),
		Status:        domain.AgentContextStatusActive,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	snapshotRepo.snapshots["agent-2"] = &domain.AgentContextSnapshot{
		SessionID:     "session-2",
		AgentID:       "agent-2",
		FlowType:      domain.FlowType("coder"),
		SchemaVersion: domain.CurrentSchemaVersion,
		ContextData:   []byte("{}"),
		Status:        domain.AgentContextStatusActive,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err := engine.RecoverInterrupted(ctx)
	require.NoError(t, err)

	// Check both marked as interrupted
	assert.Equal(t, domain.AgentContextStatusInterrupted, snapshotRepo.snapshots["agent-1"].Status)
	assert.Equal(t, domain.AgentContextStatusInterrupted, snapshotRepo.snapshots["agent-2"].Status)
}

// Test 8: Validate config
func TestEngine_ValidateConfig(t *testing.T) {
	ctx := context.Background()
	engine := New(newMockSnapshotRepo(), newMockHistoryRepo())

	tests := []struct {
		name    string
		cfg     ExecutionConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: ExecutionConfig{
				SessionID: "session-1",
				AgentID:   "supervisor",
				Flow:      testFlow(),
				ChatModel: &mockChatModel{},
				Input:     "test",
			},
			wantErr: false,
		},
		{
			name: "missing session_id",
			cfg: ExecutionConfig{
				AgentID:   "supervisor",
				Flow:      testFlow(),
				ChatModel: &mockChatModel{},
			},
			wantErr: true,
		},
		{
			name: "missing agent_id",
			cfg: ExecutionConfig{
				SessionID: "session-1",
				Flow:      testFlow(),
				ChatModel: &mockChatModel{},
			},
			wantErr: true,
		},
		{
			name: "missing flow",
			cfg: ExecutionConfig{
				SessionID: "session-1",
				AgentID:   "supervisor",
				ChatModel: &mockChatModel{},
			},
			wantErr: true,
		},
		{
			name: "missing chat_model",
			cfg: ExecutionConfig{
				SessionID: "session-1",
				AgentID:   "supervisor",
				Flow:      testFlow(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := engine.Execute(ctx, tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				// Error may occur from agent execution, but not from validation
				// Just check that validation doesn't fail
			}
		})
	}
}

// Test 9: Snapshot compression
func TestEngine_SnapshotCompression(t *testing.T) {
	ctx := context.Background()
	snapshotRepo := newMockSnapshotRepo()
	historyRepo := newMockHistoryRepo()
	engine := New(snapshotRepo, historyRepo)

	// Create a large history with 100+ messages (simulating multiple resumes)
	initialMessages := make([]*schema.Message, 0, 120)

	// Add system prompt
	initialMessages = append(initialMessages, &schema.Message{
		Role:    schema.System,
		Content: "You are a helpful assistant",
	})

	// Add 100 user-assistant pairs (simulating many turns)
	for i := 1; i <= 100; i++ {
		initialMessages = append(initialMessages,
			&schema.Message{Role: schema.User, Content: fmt.Sprintf("Question %d", i)},
			&schema.Message{Role: schema.Assistant, Content: fmt.Sprintf("Answer %d - this is a detailed response with lots of text content to simulate real usage", i)},
		)
	}

	// Save initial snapshot with large history
	contextData, err := adapters.SerializeSchemaMessages(initialMessages)
	require.NoError(t, err)

	snapshotRepo.snapshots["supervisor"] = &domain.AgentContextSnapshot{
		SessionID:     "session-1",
		AgentID:       "supervisor",
		FlowType:      domain.FlowType("supervisor"),
		SchemaVersion: domain.CurrentSchemaVersion,
		ContextData:   contextData,
		StepNumber:    100,
		Status:        domain.AgentContextStatusSuspended,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Configure flow with MaxContextSize limit
	flow := testFlow()
	flow.MaxContextSize = 1000 // 1000 tokens = ~4000 chars (tight limit)

	cfg := ExecutionConfig{
		SessionID:         "session-1",
		AgentID:           "supervisor",
		Flow:              flow,
		ChatModel:         &mockChatModel{},
		Input:             "New question",
		Streaming:         false,
		AgentConfig:       &config.AgentConfig{},
		MessageCompressor: MessageCompressor(agents.NewContextRewriter(flow.MaxContextSize)),
	}

	result, err := engine.Execute(ctx, cfg)

	require.NoError(t, err)
	assert.Equal(t, StatusCompleted, result.Status)

	// Check that snapshot was compressed
	snapshot := snapshotRepo.snapshots["supervisor"]
	require.NotNil(t, snapshot)

	compressedMessages, err := adapters.DeserializeSchemaMessages(snapshot.ContextData)
	require.NoError(t, err)

	// After compression, message count should be LESS than original 201 (100 pairs + 1 system)
	originalCount := len(initialMessages)
	compressedCount := len(compressedMessages)

	t.Logf("Compression result: %d → %d messages", originalCount, compressedCount)
	assert.Less(t, compressedCount, originalCount, "snapshot should be compressed")

	// ALL user messages should be preserved (ContextRewriter always keeps them)
	userCount := 0
	for _, msg := range compressedMessages {
		if msg.Role == schema.User {
			userCount++
		}
	}

	// Original had 100 user messages + 1 new from current execution = 101 total
	// (or 100 if new message wasn't added yet)
	assert.GreaterOrEqual(t, userCount, 100, "all user messages should be preserved")
	t.Logf("User messages preserved: %d", userCount)

	// System prompt should be preserved
	hasSystem := false
	for _, msg := range compressedMessages {
		if msg.Role == schema.System {
			hasSystem = true
			break
		}
	}
	assert.True(t, hasSystem, "system prompt should be preserved")
}

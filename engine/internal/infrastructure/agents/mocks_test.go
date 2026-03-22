package agents

import (
	"context"
	"sync"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// mockChatModel is a mock implementation of model.ChatModel for testing
type mockChatModel struct {
	generateFunc       func(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error)
	streamFunc         func(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error)
	bindToolsFunc      func(tools []*schema.ToolInfo) error
	getTypeFunc        func() string
	isCallbacksEnabled bool
}

func (m *mockChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, input, opts...)
	}
	return &schema.Message{
		Role:    schema.Assistant,
		Content: "mock response",
	}, nil
}

func (m *mockChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	if m.streamFunc != nil {
		return m.streamFunc(ctx, input, opts...)
	}
	return nil, nil
}

func (m *mockChatModel) BindTools(tools []*schema.ToolInfo) error {
	if m.bindToolsFunc != nil {
		return m.bindToolsFunc(tools)
	}
	return nil
}

func (m *mockChatModel) GetType() string {
	if m.getTypeFunc != nil {
		return m.getTypeFunc()
	}
	return "mock"
}

func (m *mockChatModel) IsCallbacksEnabled() bool {
	return m.isCallbacksEnabled
}

// mockContextLogger is a mock for ContextLogger (without file system)
type mockContextLogger struct {
	mu                     sync.Mutex
	logContextCalled       int
	logContextSummaryCalled int
	loggedMessages         [][]*schema.Message
	loggedSteps            []int
}

func newMockContextLogger() *mockContextLogger {
	return &mockContextLogger{
		loggedMessages: make([][]*schema.Message, 0),
		loggedSteps:    make([]int, 0),
	}
}

func (m *mockContextLogger) LogContext(ctx context.Context, messages []*schema.Message, step int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logContextCalled++
	m.loggedMessages = append(m.loggedMessages, messages)
	m.loggedSteps = append(m.loggedSteps, step)
}

func (m *mockContextLogger) LogContextSummary(ctx context.Context, messages []*schema.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logContextSummaryCalled++
}

func (m *mockContextLogger) GetLogContextCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.logContextCalled
}

func (m *mockContextLogger) GetLoggedMessages() [][]*schema.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([][]*schema.Message, len(m.loggedMessages))
	copy(result, m.loggedMessages)
	return result
}

func (m *mockContextLogger) GetLoggedSteps() []int {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]int, len(m.loggedSteps))
	copy(result, m.loggedSteps)
	return result
}

// mockEventCallback collects events for testing
type mockEventCallback struct {
	mu     sync.Mutex
	events []*mockEvent
}

type mockEvent struct {
	eventType string
	step      int
	content   string
	metadata  map[string]interface{}
}

func newMockEventCallback() *mockEventCallback {
	return &mockEventCallback{
		events: make([]*mockEvent, 0),
	}
}

func (m *mockEventCallback) Callback(event interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Type assertion would depend on actual event type
	return nil
}

func (m *mockEventCallback) GetEvents() []*mockEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*mockEvent, len(m.events))
	copy(result, m.events)
	return result
}

// mockStepContentStore is a mock for StepContentStore interface
type mockStepContentStore struct {
	mu      sync.RWMutex
	content map[int]string
}

func newMockStepContentStore() *mockStepContentStore {
	return &mockStepContentStore{
		content: make(map[int]string),
	}
}

func (m *mockStepContentStore) Append(step int, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.content[step] += content
}

func (m *mockStepContentStore) Get(step int) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.content[step]
}

func (m *mockStepContentStore) GetAll() map[int]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[int]string, len(m.content))
	for k, v := range m.content {
		result[k] = v
	}
	return result
}

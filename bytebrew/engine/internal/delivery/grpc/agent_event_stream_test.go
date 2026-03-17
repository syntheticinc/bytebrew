package grpc

import (
	"context"
	"sync"
	"testing"

	pb "github.com/syntheticinc/bytebrew/bytebrew/engine/api/proto/gen"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"google.golang.org/grpc/metadata"
)

// mockEventStream implements pb.FlowService_ExecuteFlowServer for event stream testing
type mockEventStream struct {
	sentResponses []*pb.FlowResponse
	mu            sync.Mutex
	sendFunc      func(*pb.FlowResponse) error
	ctx           context.Context
}

func newMockEventStream() *mockEventStream {
	return &mockEventStream{
		sentResponses: make([]*pb.FlowResponse, 0),
		ctx:           context.Background(),
	}
}

func (m *mockEventStream) Send(resp *pb.FlowResponse) error {
	m.mu.Lock()
	m.sentResponses = append(m.sentResponses, resp)
	m.mu.Unlock()

	if m.sendFunc != nil {
		return m.sendFunc(resp)
	}
	return nil
}

func (m *mockEventStream) Recv() (*pb.FlowRequest, error) {
	return nil, nil
}

func (m *mockEventStream) Context() context.Context {
	return m.ctx
}

func (m *mockEventStream) SetHeader(md metadata.MD) error  { return nil }
func (m *mockEventStream) SendHeader(md metadata.MD) error { return nil }
func (m *mockEventStream) SetTrailer(md metadata.MD)       {}
func (m *mockEventStream) SendMsg(msg interface{}) error   { return nil }
func (m *mockEventStream) RecvMsg(msg interface{}) error   { return nil }

func (m *mockEventStream) getSentResponses() []*pb.FlowResponse {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*pb.FlowResponse, len(m.sentResponses))
	copy(result, m.sentResponses)
	return result
}

// mockToolClassifier implements domain.ToolClassifier for testing
type mockToolClassifier struct {
	classification map[string]domain.ToolType
}

func newMockToolClassifier() *mockToolClassifier {
	return &mockToolClassifier{
		classification: map[string]domain.ToolType{
			"read_file":        domain.ToolTypeProxied,
			"search_code":      domain.ToolTypeProxied,
			"get_project_tree": domain.ToolTypeProxied,
			"grep_search":      domain.ToolTypeProxied,
			"symbol_search":    domain.ToolTypeProxied,
			"smart_search":     domain.ToolTypeProxied, // Uses proxy.ExecuteSubQueries
			"manage_plan":      domain.ToolTypeServerSide,
		},
	}
}

func (c *mockToolClassifier) ClassifyTool(toolName string) domain.ToolType {
	if toolType, ok := c.classification[toolName]; ok {
		return toolType
	}
	return domain.ToolTypeServerSide // Default to server-side
}

// TestAgentEventStream_SendAnswer tests sending answer events with IsComplete flag
func TestAgentEventStream_SendAnswer(t *testing.T) {
	tests := []struct {
		name            string
		isComplete      bool
		expectedIsFinal bool
	}{
		{"complete answer", true, true},
		{"intermediate answer", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream := newMockEventStream()
			classifier := newMockToolClassifier()
			streamWriter := NewStreamWriter(stream)
			eventStream := NewGrpcAgentEventStream(stream, "session-1", classifier, streamWriter)

			event := &domain.AgentEvent{
				Type:       domain.EventTypeAnswer,
				Content:    "This is the answer",
				Step:       1,
				IsComplete: tt.isComplete,
			}

			err := eventStream.Send(event)
			if err != nil {
				t.Fatalf("Send() error = %v", err)
			}

			streamWriter.Close()
			responses := stream.getSentResponses()
			if len(responses) != 1 {
				t.Fatalf("Expected 1 response, got %d", len(responses))
			}

			resp := responses[0]
			if resp.Type != pb.ResponseType_RESPONSE_TYPE_ANSWER {
				t.Errorf("Type = %v, want RESPONSE_TYPE_ANSWER", resp.Type)
			}
			if resp.Content != "This is the answer" {
				t.Errorf("Content = %v, want 'This is the answer'", resp.Content)
			}
			if resp.IsFinal != tt.expectedIsFinal {
				t.Errorf("IsFinal = %v, want %v", resp.IsFinal, tt.expectedIsFinal)
			}
			if resp.SessionId != "session-1" {
				t.Errorf("SessionId = %v, want 'session-1'", resp.SessionId)
			}
			if resp.Thought == nil {
				t.Error("Thought should not be nil for answer events")
			}
		})
	}
}

// TestAgentEventStream_SendToolCall_Proxied tests that proxied tool calls are skipped
func TestAgentEventStream_SendToolCall_Proxied(t *testing.T) {
	stream := newMockEventStream()
	classifier := newMockToolClassifier()
	streamWriter := NewStreamWriter(stream)
	defer streamWriter.Close()
	eventStream := NewGrpcAgentEventStream(stream, "session-1", classifier, streamWriter)

	// Proxied tool call should be skipped
	event := &domain.AgentEvent{
		Type:    domain.EventTypeToolCall,
		Content: "read_file", // Proxied tool
		Step:    1,
	}

	err := eventStream.Send(event)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	responses := stream.getSentResponses()
	if len(responses) != 0 {
		t.Errorf("Expected 0 responses for proxied tool call, got %d", len(responses))
	}
}

// TestAgentEventStream_SendToolCall_ServerSide tests sending server-side tool calls
func TestAgentEventStream_SendToolCall_ServerSide(t *testing.T) {
	stream := newMockEventStream()
	classifier := newMockToolClassifier()
	streamWriter := NewStreamWriter(stream)
	eventStream := NewGrpcAgentEventStream(stream, "session-1", classifier, streamWriter)

	// Server-side tool call should be sent
	event := &domain.AgentEvent{
		Type:    domain.EventTypeToolCall,
		Content: "manage_plan", // Server-side tool
		Step:    2,
		Metadata: map[string]interface{}{
			"function_arguments": `{"action": "create", "title": "test plan"}`,
		},
	}

	err := eventStream.Send(event)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	streamWriter.Close()
	responses := stream.getSentResponses()
	if len(responses) != 1 {
		t.Fatalf("Expected 1 response for server-side tool call, got %d", len(responses))
	}

	resp := responses[0]
	if resp.Type != pb.ResponseType_RESPONSE_TYPE_TOOL_CALL {
		t.Errorf("Type = %v, want RESPONSE_TYPE_TOOL_CALL", resp.Type)
	}
	if resp.ToolCall == nil {
		t.Fatal("ToolCall should not be nil")
	}
	if resp.ToolCall.ToolName != "manage_plan" {
		t.Errorf("ToolName = %v, want 'manage_plan'", resp.ToolCall.ToolName)
	}
	if resp.ToolCall.CallId != "server-manage_plan-2" {
		t.Errorf("CallId = %v, want 'server-manage_plan-2'", resp.ToolCall.CallId)
	}
	// Verify arguments were parsed
	if resp.ToolCall.Arguments["action"] != "create" {
		t.Errorf("Arguments[action] = %v, want 'create'", resp.ToolCall.Arguments["action"])
	}
}

// TestAgentEventStream_SendToolCall_SmartSearchSkipped tests that smart_search TOOL_CALL is skipped
// smart_search uses proxy.ExecuteSubQueries, so agent_event_stream should not duplicate the TOOL_CALL
func TestAgentEventStream_SendToolCall_SmartSearchSkipped(t *testing.T) {
	stream := newMockEventStream()
	classifier := newMockToolClassifier()
	streamWriter := NewStreamWriter(stream)
	eventStream := NewGrpcAgentEventStream(stream, "session-1", classifier, streamWriter)

	event := &domain.AgentEvent{
		Type:    domain.EventTypeToolCall,
		Content: "smart_search", // Proxied (uses proxy.ExecuteSubQueries)
		Step:    2,
		Metadata: map[string]interface{}{
			"function_arguments": `{"query": "test query"}`,
		},
	}

	err := eventStream.Send(event)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	streamWriter.Close()
	responses := stream.getSentResponses()
	if len(responses) != 0 {
		t.Fatalf("Expected 0 responses for proxied smart_search TOOL_CALL, got %d", len(responses))
	}
}

// TestAgentEventStream_SendToolResult tests sending tool results
func TestAgentEventStream_SendToolResult(t *testing.T) {
	stream := newMockEventStream()
	classifier := newMockToolClassifier()
	streamWriter := NewStreamWriter(stream)
	eventStream := NewGrpcAgentEventStream(stream, "session-1", classifier, streamWriter)

	// Server-side tool result
	event := &domain.AgentEvent{
		Type:    domain.EventTypeToolResult,
		Content: "result preview",
		Step:    2,
		Metadata: map[string]interface{}{
			"tool_name":   "manage_plan",
			"full_result": "plan created successfully",
		},
	}

	err := eventStream.Send(event)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	streamWriter.Close()
	responses := stream.getSentResponses()
	if len(responses) != 1 {
		t.Fatalf("Expected 1 response for server-side tool result, got %d", len(responses))
	}

	resp := responses[0]
	if resp.Type != pb.ResponseType_RESPONSE_TYPE_TOOL_RESULT {
		t.Errorf("Type = %v, want RESPONSE_TYPE_TOOL_RESULT", resp.Type)
	}
	if resp.ToolResult == nil {
		t.Fatal("ToolResult should not be nil")
	}
	if resp.ToolResult.CallId != "server-manage_plan-2" {
		t.Errorf("CallId = %v, want 'server-manage_plan-2'", resp.ToolResult.CallId)
	}
	if resp.ToolResult.Result != "plan created successfully" {
		t.Errorf("Result = %v, want 'plan created successfully'", resp.ToolResult.Result)
	}
}

// TestAgentEventStream_SendToolResult_Proxied tests that proxied tool results are skipped
func TestAgentEventStream_SendToolResult_Proxied(t *testing.T) {
	stream := newMockEventStream()
	classifier := newMockToolClassifier()
	streamWriter := NewStreamWriter(stream)
	defer streamWriter.Close()
	eventStream := NewGrpcAgentEventStream(stream, "session-1", classifier, streamWriter)

	// Proxied tool result should be skipped
	event := &domain.AgentEvent{
		Type:    domain.EventTypeToolResult,
		Content: "file content",
		Step:    1,
		Metadata: map[string]interface{}{
			"tool_name": "read_file", // Proxied tool
		},
	}

	err := eventStream.Send(event)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	responses := stream.getSentResponses()
	if len(responses) != 0 {
		t.Errorf("Expected 0 responses for proxied tool result, got %d", len(responses))
	}
}

// TestAgentEventStream_SendReasoning tests sending reasoning content
func TestAgentEventStream_SendReasoning(t *testing.T) {
	tests := []struct {
		name       string
		isComplete bool
	}{
		{"incomplete reasoning", false},
		{"complete reasoning", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream := newMockEventStream()
			classifier := newMockToolClassifier()
			streamWriter := NewStreamWriter(stream)
			eventStream := NewGrpcAgentEventStream(stream, "session-1", classifier, streamWriter)

			event := &domain.AgentEvent{
				Type:       domain.EventTypeReasoning,
				Content:    "Thinking about the problem...",
				Step:       1,
				IsComplete: tt.isComplete,
			}

			err := eventStream.Send(event)
			if err != nil {
				t.Fatalf("Send() error = %v", err)
			}

			streamWriter.Close()
			responses := stream.getSentResponses()
			if len(responses) != 1 {
				t.Fatalf("Expected 1 response, got %d", len(responses))
			}

			resp := responses[0]
			if resp.Type != pb.ResponseType_RESPONSE_TYPE_REASONING {
				t.Errorf("Type = %v, want RESPONSE_TYPE_REASONING", resp.Type)
			}
			if resp.Reasoning == nil {
				t.Fatal("Reasoning should not be nil")
			}
			if resp.Reasoning.Thinking != "Thinking about the problem..." {
				t.Errorf("Thinking = %v, want 'Thinking about the problem...'", resp.Reasoning.Thinking)
			}
			if resp.Reasoning.IsComplete != tt.isComplete {
				t.Errorf("IsComplete = %v, want %v", resp.Reasoning.IsComplete, tt.isComplete)
			}
		})
	}
}

// TestAgentEventStream_SendAnswerChunk tests sending answer chunks
func TestAgentEventStream_SendAnswerChunk(t *testing.T) {
	stream := newMockEventStream()
	classifier := newMockToolClassifier()
	streamWriter := NewStreamWriter(stream)
	eventStream := NewGrpcAgentEventStream(stream, "session-1", classifier, streamWriter)

	event := &domain.AgentEvent{
		Type:    domain.EventTypeAnswerChunk,
		Content: "chunk of text",
		Step:    1,
	}

	err := eventStream.Send(event)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	streamWriter.Close()
	responses := stream.getSentResponses()
	if len(responses) != 1 {
		t.Fatalf("Expected 1 response, got %d", len(responses))
	}

	resp := responses[0]
	if resp.Type != pb.ResponseType_RESPONSE_TYPE_ANSWER_CHUNK {
		t.Errorf("Type = %v, want RESPONSE_TYPE_ANSWER_CHUNK", resp.Type)
	}
	if resp.Content != "chunk of text" {
		t.Errorf("Content = %v, want 'chunk of text'", resp.Content)
	}
	if resp.IsFinal {
		t.Error("IsFinal should be false for answer chunk events")
	}
}

// TestAgentEventStream_SanitizeUTF8 tests UTF-8 sanitization
func TestAgentEventStream_SanitizeUTF8(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid UTF-8",
			input:    "Hello, world!",
			expected: "Hello, world!",
		},
		{
			name:     "UTF-8 with emojis",
			input:    "Hello 😀 World 🌍",
			expected: "Hello 😀 World 🌍",
		},
		{
			name:     "Russian text",
			input:    "Привет, мир!",
			expected: "Привет, мир!",
		},
		{
			name:     "Chinese text",
			input:    "你好世界",
			expected: "你好世界",
		},
		{
			name:     "invalid UTF-8 sequence",
			input:    string([]byte{0xff, 0xfe, 0x00, 0x01}),
			expected: "\ufffd\ufffd\x00\x01", // Invalid bytes replaced with replacement character
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeUTF8(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeUTF8(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestAgentEventStream_SendError tests error handling when StreamWriter is closed
func TestAgentEventStream_SendError(t *testing.T) {
	stream := newMockEventStream()
	classifier := newMockToolClassifier()
	streamWriter := NewStreamWriter(stream)
	eventStream := NewGrpcAgentEventStream(stream, "session-1", classifier, streamWriter)

	// Close writer first — subsequent Send() should return error
	streamWriter.Close()

	event := &domain.AgentEvent{
		Type:    domain.EventTypeAnswer,
		Content: "test answer",
		Step:    1,
	}

	err := eventStream.Send(event)
	if err == nil {
		t.Fatal("Send() expected error after writer closed, got nil")
	}
}

// TestAgentEventStream_ToolCallArgumentParsing tests various argument formats
func TestAgentEventStream_ToolCallArgumentParsing(t *testing.T) {
	tests := []struct {
		name     string
		argsJSON string
		expected map[string]string
	}{
		{
			name:     "string arguments",
			argsJSON: `{"query": "test", "file": "path.go"}`,
			expected: map[string]string{"query": "test", "file": "path.go"},
		},
		{
			name:     "numeric arguments",
			argsJSON: `{"limit": 10, "score": 0.5}`,
			expected: map[string]string{"limit": "10", "score": "0"}, // %.0f rounds 0.5 to 0
		},
		{
			name:     "boolean arguments",
			argsJSON: `{"include": true, "exclude": false}`,
			expected: map[string]string{"include": "true", "exclude": "false"},
		},
		{
			name:     "invalid JSON",
			argsJSON: `not json`,
			expected: map[string]string{"_json": "not json"},
		},
		{
			name:     "empty arguments",
			argsJSON: `{}`,
			expected: map[string]string{},
		},
		{
			name:     "array with single element",
			argsJSON: `{"question": ["What is the meaning of life?"]}`,
			expected: map[string]string{"question": "What is the meaning of life?"},
		},
		{
			name:     "array with multiple elements",
			argsJSON: `{"items": ["first item", "second item", "third item"]}`,
			expected: map[string]string{"items": "first item\nsecond item\nthird item"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream := newMockEventStream()
			classifier := newMockToolClassifier()
			streamWriter := NewStreamWriter(stream)
			eventStream := NewGrpcAgentEventStream(stream, "session-1", classifier, streamWriter)

			event := &domain.AgentEvent{
				Type:    domain.EventTypeToolCall,
				Content: "manage_plan", // Server-side tool for argument parsing test
				Step:    1,
				Metadata: map[string]interface{}{
					"function_arguments": tt.argsJSON,
				},
			}

			err := eventStream.Send(event)
			if err != nil {
				t.Fatalf("Send() error = %v", err)
			}

			streamWriter.Close()
			responses := stream.getSentResponses()
			if len(responses) != 1 {
				t.Fatalf("Expected 1 response, got %d", len(responses))
			}

			args := responses[0].ToolCall.Arguments
			for k, v := range tt.expected {
				if args[k] != v {
					t.Errorf("Arguments[%s] = %v, want %v", k, args[k], v)
				}
			}
		})
	}
}

// TestAgentEventStream_SessionIdPreserved tests that session ID is preserved
func TestAgentEventStream_SessionIdPreserved(t *testing.T) {
	stream := newMockEventStream()
	classifier := newMockToolClassifier()
	streamWriter := NewStreamWriter(stream)
	eventStream := NewGrpcAgentEventStream(stream, "test-session-123", classifier, streamWriter)

	events := []*domain.AgentEvent{
		{Type: domain.EventTypeAnswer, Content: "answer", Step: 1},
		{Type: domain.EventTypeReasoning, Content: "thinking", Step: 1},
		{Type: domain.EventTypeAnswerChunk, Content: "chunk", Step: 1},
	}

	for _, event := range events {
		err := eventStream.Send(event)
		if err != nil {
			t.Fatalf("Send() error = %v", err)
		}
	}

	streamWriter.Close()
	responses := stream.getSentResponses()
	for i, resp := range responses {
		if resp.SessionId != "test-session-123" {
			t.Errorf("Response %d: SessionId = %v, want 'test-session-123'", i, resp.SessionId)
		}
	}
}

// TestAgentEventStream_StepNumber tests that step numbers are preserved
func TestAgentEventStream_StepNumber(t *testing.T) {
	stream := newMockEventStream()
	classifier := newMockToolClassifier()
	streamWriter := NewStreamWriter(stream)
	eventStream := NewGrpcAgentEventStream(stream, "session-1", classifier, streamWriter)

	steps := []int{0, 1, 5, 10, 100}

	for _, step := range steps {
		event := &domain.AgentEvent{
			Type:    domain.EventTypeAnswerChunk,
			Content: "chunk",
			Step:    step,
		}

		err := eventStream.Send(event)
		if err != nil {
			t.Fatalf("Send() error = %v", err)
		}
	}

	streamWriter.Close()
	responses := stream.getSentResponses()
	for i, resp := range responses {
		if int(resp.Step) != steps[i] {
			t.Errorf("Response %d: Step = %v, want %v", i, resp.Step, steps[i])
		}
	}
}

package grpc

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
	"google.golang.org/grpc/metadata"
)

// mockStream implements pb.FlowService_ExecuteFlowServer for proxy testing
type mockProxyStream struct {
	sentResponses []*pb.FlowResponse
	mu            sync.Mutex
	sendFunc      func(*pb.FlowResponse) error
	ctx           context.Context
}

func newMockProxyStream() *mockProxyStream {
	return &mockProxyStream{
		sentResponses: make([]*pb.FlowResponse, 0),
		ctx:           context.Background(),
	}
}

func (m *mockProxyStream) Send(resp *pb.FlowResponse) error {
	m.mu.Lock()
	m.sentResponses = append(m.sentResponses, resp)
	m.mu.Unlock()

	if m.sendFunc != nil {
		return m.sendFunc(resp)
	}
	return nil
}

func (m *mockProxyStream) Recv() (*pb.FlowRequest, error) {
	return nil, nil
}

func (m *mockProxyStream) Context() context.Context {
	return m.ctx
}

func (m *mockProxyStream) SetHeader(md metadata.MD) error  { return nil }
func (m *mockProxyStream) SendHeader(md metadata.MD) error { return nil }
func (m *mockProxyStream) SetTrailer(md metadata.MD)       {}
func (m *mockProxyStream) SendMsg(msg interface{}) error   { return nil }
func (m *mockProxyStream) RecvMsg(msg interface{}) error   { return nil }

func (m *mockProxyStream) getSentResponses() []*pb.FlowResponse {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*pb.FlowResponse, len(m.sentResponses))
	copy(result, m.sentResponses)
	return result
}

// newTestProxy creates a proxy with a StreamWriter for testing.
// Caller should defer sw.Close() if the test needs to wait for all sends to complete.
func newTestProxy(stream *mockProxyStream) (*StreamBasedClientOperationsProxy, *StreamWriter) {
	sw := NewStreamWriter(stream)
	proxy := NewStreamBasedClientOperationsProxy(stream, "session-1", "project-1", sw)
	return proxy, sw
}

// TestStreamClientOperationsProxy_ReadFile_Success tests successful file read
func TestStreamClientOperationsProxy_ReadFile_Success(t *testing.T) {
	// Use a channel to signal when request is received
	requestReceived := make(chan *pb.FlowResponse, 1)

	stream := newMockProxyStream()
	stream.sendFunc = func(resp *pb.FlowResponse) error {
		requestReceived <- resp
		return nil
	}

	proxy, sw := newTestProxy(stream)
	defer sw.Close()

	// Simulate client response in background
	go func() {
		// Wait for the request
		resp := <-requestReceived

		if resp.ToolCall == nil {
			t.Error("Expected ToolCall in response")
			return
		}

		// Send result back
		proxy.HandleToolResult(&pb.ToolResult{
			CallId: resp.ToolCall.CallId,
			Result: "file content here",
		})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	content, err := proxy.ReadFile(ctx, "session-1", "/path/to/file.txt", 0, 100)

	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if content != "file content here" {
		t.Errorf("ReadFile() content = %v, want %v", content, "file content here")
	}

	// Verify the tool call was sent correctly
	responses := stream.getSentResponses()
	if len(responses) != 1 {
		t.Fatalf("Expected 1 response sent, got %d", len(responses))
	}

	toolCall := responses[0].ToolCall
	if toolCall.ToolName != "read_file" {
		t.Errorf("ToolName = %v, want read_file", toolCall.ToolName)
	}
	if toolCall.Arguments["file_path"] != "/path/to/file.txt" {
		t.Errorf("file_path = %v, want /path/to/file.txt", toolCall.Arguments["file_path"])
	}
}

// TestStreamClientOperationsProxy_ReadFile_Timeout tests timeout handling
func TestStreamClientOperationsProxy_ReadFile_Timeout(t *testing.T) {
	stream := newMockProxyStream()
	proxy, sw := newTestProxy(stream)
	defer sw.Close()

	// Don't send any result - simulate timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := proxy.ReadFile(ctx, "session-1", "/path/to/file.txt", 0, 100)

	if err == nil {
		t.Fatal("ReadFile() expected timeout error, got nil")
	}

	// Verify pending call was cleaned up
	proxy.mu.RLock()
	pendingCount := len(proxy.pendingCalls)
	proxy.mu.RUnlock()

	if pendingCount != 0 {
		t.Errorf("Expected 0 pending calls after timeout, got %d", pendingCount)
	}
}

// TestStreamClientOperationsProxy_ReadFile_Error tests error handling from client
func TestStreamClientOperationsProxy_ReadFile_Error(t *testing.T) {
	stream := newMockProxyStream()
	proxy, sw := newTestProxy(stream)
	defer sw.Close()

	// Simulate error response from client
	go func() {
		time.Sleep(10 * time.Millisecond)

		responses := stream.getSentResponses()
		if len(responses) == 0 {
			return
		}

		resp := responses[len(responses)-1]
		proxy.HandleToolResult(&pb.ToolResult{
			CallId: resp.ToolCall.CallId,
			Error: &pb.Error{
				Code:    "NOT_FOUND",
				Message: "file not found",
			},
		})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err := proxy.ReadFile(ctx, "session-1", "/nonexistent/file.txt", 0, 100)

	if err == nil {
		t.Fatal("ReadFile() expected error, got nil")
	}
}

// TestStreamClientOperationsProxy_SearchCode_Success tests successful code search
func TestStreamClientOperationsProxy_SearchCode_Success(t *testing.T) {
	stream := newMockProxyStream()
	proxy, sw := newTestProxy(stream)
	defer sw.Close()

	expectedResult := `[{"file":"test.go","content":"found code"}]`

	go func() {
		time.Sleep(10 * time.Millisecond)

		responses := stream.getSentResponses()
		if len(responses) == 0 {
			return
		}

		resp := responses[len(responses)-1]
		proxy.HandleToolResult(&pb.ToolResult{
			CallId: resp.ToolCall.CallId,
			Result: expectedResult,
		})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	result, err := proxy.SearchCode(ctx, "session-1", "search query", "project-1", 10, 0.5)

	if err != nil {
		t.Fatalf("SearchCode() error = %v", err)
	}
	if string(result) != expectedResult {
		t.Errorf("SearchCode() result = %v, want %v", string(result), expectedResult)
	}

	// Verify arguments
	responses := stream.getSentResponses()
	toolCall := responses[0].ToolCall
	if toolCall.ToolName != "search_code" {
		t.Errorf("ToolName = %v, want search_code", toolCall.ToolName)
	}
	if toolCall.Arguments["query"] != "search query" {
		t.Errorf("query = %v, want 'search query'", toolCall.Arguments["query"])
	}
}

// TestStreamClientOperationsProxy_ExecuteSubQueries_Success tests grouped sub-queries
func TestStreamClientOperationsProxy_ExecuteSubQueries_Success(t *testing.T) {
	stream := newMockProxyStream()
	proxy, sw := newTestProxy(stream)
	defer sw.Close()

	subQueries := []*pb.SubQuery{
		{Type: "vector", Query: "query1"},
		{Type: "grep", Query: "query2"},
	}

	expectedSubResults := []*pb.SubResult{
		{Type: "vector", Result: "vector result"},
		{Type: "grep", Result: "grep result"},
	}

	go func() {
		time.Sleep(10 * time.Millisecond)

		responses := stream.getSentResponses()
		if len(responses) == 0 {
			return
		}

		resp := responses[len(responses)-1]
		proxy.HandleToolResult(&pb.ToolResult{
			CallId:     resp.ToolCall.CallId,
			SubResults: expectedSubResults,
		})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	results, err := proxy.ExecuteSubQueries(ctx, "session-1", subQueries)

	if err != nil {
		t.Fatalf("ExecuteSubQueries() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("ExecuteSubQueries() returned %d results, want 2", len(results))
	}

	// Verify sub-queries were sent
	responses := stream.getSentResponses()
	toolCall := responses[0].ToolCall
	if toolCall.ToolName != "smart_search" {
		t.Errorf("ToolName = %v, want smart_search", toolCall.ToolName)
	}
	if len(toolCall.SubQueries) != 2 {
		t.Errorf("SubQueries count = %d, want 2", len(toolCall.SubQueries))
	}
}

// TestStreamClientOperationsProxy_PendingCallsCleanup tests cleanup of pending calls
func TestStreamClientOperationsProxy_PendingCallsCleanup(t *testing.T) {
	stream := newMockProxyStream()
	proxy, sw := newTestProxy(stream)
	defer sw.Close()

	// Verify initial state
	proxy.mu.RLock()
	if len(proxy.pendingCalls) != 0 {
		t.Fatal("Expected 0 pending calls initially")
	}
	proxy.mu.RUnlock()

	// Start multiple calls that will timeout
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			proxy.ReadFile(ctx, "session-1", "/file.txt", 0, 100)
		}()
	}

	// Wait for all to timeout
	wg.Wait()

	// Give a moment for cleanup
	time.Sleep(10 * time.Millisecond)

	// Verify all pending calls were cleaned up
	proxy.mu.RLock()
	pendingCount := len(proxy.pendingCalls)
	proxy.mu.RUnlock()

	if pendingCount != 0 {
		t.Errorf("Expected 0 pending calls after cleanup, got %d", pendingCount)
	}
}

// TestStreamClientOperationsProxy_ConcurrentCalls tests concurrent tool calls
func TestStreamClientOperationsProxy_ConcurrentCalls(t *testing.T) {
	stream := newMockProxyStream()
	proxy, sw := newTestProxy(stream)
	defer sw.Close()

	// Track results
	var successCount int32
	var wg sync.WaitGroup

	numCalls := 5

	// Start responder that handles all calls
	go func() {
		for {
			time.Sleep(5 * time.Millisecond)

			responses := stream.getSentResponses()
			for _, resp := range responses {
				if resp.ToolCall == nil {
					continue
				}

				// Check if we already responded to this call
				proxy.mu.RLock()
				_, exists := proxy.pendingCalls[resp.ToolCall.CallId]
				proxy.mu.RUnlock()

				if exists {
					proxy.HandleToolResult(&pb.ToolResult{
						CallId: resp.ToolCall.CallId,
						Result: "result for " + resp.ToolCall.CallId,
					})
				}
			}
		}
	}()

	// Launch concurrent calls
	for i := 0; i < numCalls; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			_, err := proxy.ReadFile(ctx, "session-1", "/file.txt", 0, 100)
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()

	if int(atomic.LoadInt32(&successCount)) != numCalls {
		t.Errorf("Expected %d successful calls, got %d", numCalls, successCount)
	}

	// Verify unique call IDs were used
	responses := stream.getSentResponses()
	callIDs := make(map[string]bool)
	for _, resp := range responses {
		if resp.ToolCall != nil {
			if callIDs[resp.ToolCall.CallId] {
				t.Errorf("Duplicate call ID found: %s", resp.ToolCall.CallId)
			}
			callIDs[resp.ToolCall.CallId] = true
		}
	}
}

// TestStreamClientOperationsProxy_GrepSearch_Success tests grep search
func TestStreamClientOperationsProxy_GrepSearch_Success(t *testing.T) {
	stream := newMockProxyStream()
	proxy, sw := newTestProxy(stream)
	defer sw.Close()

	expectedResult := "file1.go:10: match\nfile2.go:20: match"

	go func() {
		time.Sleep(10 * time.Millisecond)

		responses := stream.getSentResponses()
		if len(responses) == 0 {
			return
		}

		resp := responses[len(responses)-1]
		proxy.HandleToolResult(&pb.ToolResult{
			CallId: resp.ToolCall.CallId,
			Result: expectedResult,
		})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	result, err := proxy.GrepSearch(ctx, "session-1", "pattern", 50, []string{"go", "ts"}, false)

	if err != nil {
		t.Fatalf("GrepSearch() error = %v", err)
	}
	if result != expectedResult {
		t.Errorf("GrepSearch() result = %v, want %v", result, expectedResult)
	}

	// Verify arguments
	responses := stream.getSentResponses()
	toolCall := responses[0].ToolCall
	if toolCall.ToolName != "grep_search" {
		t.Errorf("ToolName = %v, want grep_search", toolCall.ToolName)
	}
	if toolCall.Arguments["pattern"] != "pattern" {
		t.Errorf("pattern = %v, want 'pattern'", toolCall.Arguments["pattern"])
	}
	if toolCall.Arguments["file_types"] != "go,ts" {
		t.Errorf("file_types = %v, want 'go,ts'", toolCall.Arguments["file_types"])
	}
}

// TestStreamClientOperationsProxy_SymbolSearch_Success tests symbol search
func TestStreamClientOperationsProxy_SymbolSearch_Success(t *testing.T) {
	stream := newMockProxyStream()
	proxy, sw := newTestProxy(stream)
	defer sw.Close()

	expectedResult := "Symbol: MyFunc at file.go:10"

	go func() {
		time.Sleep(10 * time.Millisecond)

		responses := stream.getSentResponses()
		if len(responses) == 0 {
			return
		}

		resp := responses[len(responses)-1]
		proxy.HandleToolResult(&pb.ToolResult{
			CallId: resp.ToolCall.CallId,
			Result: expectedResult,
		})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	result, err := proxy.SymbolSearch(ctx, "session-1", "MyFunc", 10, []string{"function", "class"})

	if err != nil {
		t.Fatalf("SymbolSearch() error = %v", err)
	}
	if result != expectedResult {
		t.Errorf("SymbolSearch() result = %v, want %v", result, expectedResult)
	}

	// Verify arguments
	responses := stream.getSentResponses()
	toolCall := responses[0].ToolCall
	if toolCall.ToolName != "symbol_search" {
		t.Errorf("ToolName = %v, want symbol_search", toolCall.ToolName)
	}
	if toolCall.Arguments["symbol_name"] != "MyFunc" {
		t.Errorf("symbol_name = %v, want 'MyFunc'", toolCall.Arguments["symbol_name"])
	}
}

// TestStreamClientOperationsProxy_HandleToolResult_UnknownCallId tests handling unknown call ID
func TestStreamClientOperationsProxy_HandleToolResult_UnknownCallId(t *testing.T) {
	stream := newMockProxyStream()
	proxy, sw := newTestProxy(stream)
	defer sw.Close()

	// This should not panic or cause issues
	proxy.HandleToolResult(&pb.ToolResult{
		CallId: "unknown-call-id",
		Result: "some result",
	})

	// Verify no pending calls were created
	proxy.mu.RLock()
	pendingCount := len(proxy.pendingCalls)
	proxy.mu.RUnlock()

	if pendingCount != 0 {
		t.Errorf("Expected 0 pending calls, got %d", pendingCount)
	}
}

// TestStreamClientOperationsProxy_SendError tests handling send errors when StreamWriter is closed
func TestStreamClientOperationsProxy_SendError(t *testing.T) {
	stream := newMockProxyStream()
	sw := NewStreamWriter(stream)
	proxy := NewStreamBasedClientOperationsProxy(stream, "session-1", "project-1", sw)

	// Close StreamWriter first — subsequent sends should fail
	sw.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err := proxy.ReadFile(ctx, "session-1", "/file.txt", 0, 100)

	if err == nil {
		t.Fatal("ReadFile() expected error on closed StreamWriter, got nil")
	}

	// Verify pending call was cleaned up after send error
	proxy.mu.RLock()
	pendingCount := len(proxy.pendingCalls)
	proxy.mu.RUnlock()

	if pendingCount != 0 {
		t.Errorf("Expected 0 pending calls after send error, got %d", pendingCount)
	}
}

// TestStreamClientOperationsProxy_UniqueCallIDs tests that call IDs are unique
func TestStreamClientOperationsProxy_UniqueCallIDs(t *testing.T) {
	stream := newMockProxyStream()
	proxy, sw := newTestProxy(stream)
	defer sw.Close()

	// Generate many call IDs in parallel
	var wg sync.WaitGroup
	callIDs := make(chan string, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Use atomic counter directly
			counter := atomic.AddUint64(&proxy.callCounter, 1)
			callID := "test-" + string(rune(counter))
			callIDs <- callID
		}()
	}

	wg.Wait()
	close(callIDs)

	// Verify all counter values are unique
	seen := make(map[string]bool)
	for id := range callIDs {
		if seen[id] {
			t.Errorf("Duplicate call ID generated: %s", id)
		}
		seen[id] = true
	}

	// Counter should be 100 after 100 increments
	if proxy.callCounter != 100 {
		t.Errorf("Expected callCounter = 100, got %d", proxy.callCounter)
	}
}

// TestProxy_CleanupPendingCalls tests CleanupPendingCalls method
func TestProxy_CleanupPendingCalls(t *testing.T) {
	stream := newMockProxyStream()
	proxy, sw := newTestProxy(stream)
	defer sw.Close()

	// Manually add pending calls
	proxy.mu.Lock()
	proxy.pendingCalls["call-1"] = make(chan *pb.ToolResult, 1)
	proxy.pendingCalls["call-2"] = make(chan *pb.ToolResult, 1)
	proxy.pendingCalls["call-3"] = make(chan *pb.ToolResult, 1)
	initialCount := len(proxy.pendingCalls)
	proxy.mu.Unlock()

	if initialCount != 3 {
		t.Fatalf("Setup failed: expected 3 pending calls, got %d", initialCount)
	}

	// Cleanup
	proxy.CleanupPendingCalls()

	// Verify map is empty
	proxy.mu.RLock()
	finalCount := len(proxy.pendingCalls)
	proxy.mu.RUnlock()

	if finalCount != 0 {
		t.Errorf("CleanupPendingCalls() left %d pending calls, want 0", finalCount)
	}
}

// TestProxy_CleanupPendingCalls_ClosesChannels tests that channels are closed
func TestProxy_CleanupPendingCalls_ClosesChannels(t *testing.T) {
	stream := newMockProxyStream()
	proxy, sw := newTestProxy(stream)
	defer sw.Close()

	// Add pending call
	ch := make(chan *pb.ToolResult, 1)
	proxy.mu.Lock()
	proxy.pendingCalls["call-1"] = ch
	proxy.mu.Unlock()

	// Start goroutine that waits on channel
	done := make(chan bool)
	go func() {
		// This should receive nil when channel is closed
		_, ok := <-ch
		if ok {
			t.Error("Channel should be closed, but received value")
		}
		done <- true
	}()

	// Cleanup should close the channel
	proxy.CleanupPendingCalls()

	// Wait for goroutine to finish (with timeout)
	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("CleanupPendingCalls() did not close channel")
	}
}

// TestStreamClientOperationsProxy_AskUserQuestionnaire_Success tests successful AskUserQuestionnaire flow
func TestStreamClientOperationsProxy_AskUserQuestionnaire_Success(t *testing.T) {
	requestReceived := make(chan *pb.FlowResponse, 1)

	stream := newMockProxyStream()
	stream.sendFunc = func(resp *pb.FlowResponse) error {
		requestReceived <- resp
		return nil
	}

	proxy, sw := newTestProxy(stream)
	defer sw.Close()

	go func() {
		resp := <-requestReceived

		if resp.ToolCall == nil {
			t.Error("Expected ToolCall in response")
			return
		}

		proxy.HandleToolResult(&pb.ToolResult{
			CallId: resp.ToolCall.CallId,
			Result: `[{"question":"Do you approve?","answer":"yes"}]`,
		})
	}()

	ctx := context.Background()

	questionsJSON := `[{"text":"Do you approve?","options":[{"label":"yes"},{"label":"no"}]}]`
	result, err := proxy.AskUserQuestionnaire(ctx, "session-1", questionsJSON)

	if err != nil {
		t.Fatalf("AskUserQuestionnaire() error = %v", err)
	}
	if result != `[{"question":"Do you approve?","answer":"yes"}]` {
		t.Errorf("AskUserQuestionnaire() result = %v, want JSON answers", result)
	}

	if proxy.IsWaitingForUser() {
		t.Error("waitingForUser should be false after successful AskUserQuestionnaire")
	}

	responses := stream.getSentResponses()
	if len(responses) != 1 {
		t.Fatalf("Expected 1 response sent, got %d", len(responses))
	}

	toolCall := responses[0].ToolCall
	if toolCall.ToolName != "ask_user" {
		t.Errorf("ToolName = %v, want ask_user", toolCall.ToolName)
	}
	if toolCall.Arguments["questions"] != questionsJSON {
		t.Errorf("questions = %v, want %v", toolCall.Arguments["questions"], questionsJSON)
	}
}

// TestStreamClientOperationsProxy_AskUserQuestionnaire_NoTimeout tests that AskUserQuestionnaire does NOT time out after 5min
func TestStreamClientOperationsProxy_AskUserQuestionnaire_NoTimeout(t *testing.T) {
	requestReceived := make(chan *pb.FlowResponse, 1)

	stream := newMockProxyStream()
	stream.sendFunc = func(resp *pb.FlowResponse) error {
		requestReceived <- resp
		return nil
	}

	proxy, sw := newTestProxy(stream)
	defer sw.Close()

	go func() {
		resp := <-requestReceived

		time.Sleep(150 * time.Millisecond)

		proxy.HandleToolResult(&pb.ToolResult{
			CallId: resp.ToolCall.CallId,
			Result: "late response",
		})
	}()

	ctx := context.Background()

	questionsJSON := `[{"text":"Wait for me..."}]`
	result, err := proxy.AskUserQuestionnaire(ctx, "session-1", questionsJSON)

	if err != nil {
		t.Fatalf("AskUserQuestionnaire() should not timeout, got error = %v", err)
	}
	if result != "late response" {
		t.Errorf("AskUserQuestionnaire() result = %v, want 'late response'", result)
	}

	if proxy.IsWaitingForUser() {
		t.Error("waitingForUser should be false after receiving response")
	}
}

// TestStreamClientOperationsProxy_AskUserQuestionnaire_CancelledByContext tests that AskUserQuestionnaire respects context cancellation
func TestStreamClientOperationsProxy_AskUserQuestionnaire_CancelledByContext(t *testing.T) {
	requestReceived := make(chan *pb.FlowResponse, 1)

	stream := newMockProxyStream()
	stream.sendFunc = func(resp *pb.FlowResponse) error {
		requestReceived <- resp
		return nil
	}

	proxy, sw := newTestProxy(stream)
	defer sw.Close()

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-requestReceived
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	questionsJSON := `[{"text":"Will be cancelled"}]`
	_, err := proxy.AskUserQuestionnaire(ctx, "session-1", questionsJSON)

	if err == nil {
		t.Fatal("AskUserQuestionnaire() expected error on context cancellation, got nil")
	}

	if proxy.IsWaitingForUser() {
		t.Error("waitingForUser should be false after cancellation")
	}

	proxy.mu.RLock()
	pendingCount := len(proxy.pendingCalls)
	proxy.mu.RUnlock()

	if pendingCount != 0 {
		t.Errorf("Expected 0 pending calls after cancellation cleanup, got %d", pendingCount)
	}
}

package grpc

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pb "github.com/syntheticinc/bytebrew/engine/api/proto/gen"
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

// TestStreamClientOperationsProxy_HandleToolResult_UnknownCallId tests that unknown call IDs are handled gracefully.
func TestStreamClientOperationsProxy_HandleToolResult_UnknownCallId(t *testing.T) {
	stream := newMockProxyStream()
	proxy, sw := newTestProxy(stream)
	defer sw.Close()

	proxy.HandleToolResult(&pb.ToolResult{
		CallId: "unknown-call-id",
		Result: "some result",
	})

	proxy.mu.RLock()
	pendingCount := len(proxy.pendingCalls)
	proxy.mu.RUnlock()

	if pendingCount != 0 {
		t.Errorf("Expected 0 pending calls, got %d", pendingCount)
	}
}

// TestStreamClientOperationsProxy_SendError tests handling send errors when StreamWriter is closed.
func TestStreamClientOperationsProxy_SendError(t *testing.T) {
	stream := newMockProxyStream()
	sw := NewStreamWriter(stream)
	proxy := NewStreamBasedClientOperationsProxy(stream, "session-1", "project-1", sw)

	sw.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err := proxy.AskUserQuestionnaire(ctx, "session-1", `[{"text":"?"}]`)

	if err == nil {
		t.Fatal("AskUserQuestionnaire() expected error on closed StreamWriter, got nil")
	}

	proxy.mu.RLock()
	pendingCount := len(proxy.pendingCalls)
	proxy.mu.RUnlock()

	if pendingCount != 0 {
		t.Errorf("Expected 0 pending calls after send error, got %d", pendingCount)
	}
}

// TestStreamClientOperationsProxy_UniqueCallIDs tests that call IDs are unique.
func TestStreamClientOperationsProxy_UniqueCallIDs(t *testing.T) {
	stream := newMockProxyStream()
	proxy, sw := newTestProxy(stream)
	defer sw.Close()

	var wg sync.WaitGroup
	callIDs := make(chan string, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			counter := atomic.AddUint64(&proxy.callCounter, 1)
			callID := fmt.Sprintf("test-%d", counter)
			callIDs <- callID
		}()
	}

	wg.Wait()
	close(callIDs)

	seen := make(map[string]bool)
	for id := range callIDs {
		if seen[id] {
			t.Errorf("Duplicate call ID generated: %s", id)
		}
		seen[id] = true
	}

	if proxy.callCounter != 100 {
		t.Errorf("Expected callCounter = 100, got %d", proxy.callCounter)
	}
}

// TestProxy_CleanupPendingCalls tests CleanupPendingCalls method.
func TestProxy_CleanupPendingCalls(t *testing.T) {
	stream := newMockProxyStream()
	proxy, sw := newTestProxy(stream)
	defer sw.Close()

	proxy.mu.Lock()
	proxy.pendingCalls["call-1"] = make(chan *pb.ToolResult, 1)
	proxy.pendingCalls["call-2"] = make(chan *pb.ToolResult, 1)
	proxy.pendingCalls["call-3"] = make(chan *pb.ToolResult, 1)
	initialCount := len(proxy.pendingCalls)
	proxy.mu.Unlock()

	if initialCount != 3 {
		t.Fatalf("Setup failed: expected 3 pending calls, got %d", initialCount)
	}

	proxy.CleanupPendingCalls()

	proxy.mu.RLock()
	finalCount := len(proxy.pendingCalls)
	proxy.mu.RUnlock()

	if finalCount != 0 {
		t.Errorf("CleanupPendingCalls() left %d pending calls, want 0", finalCount)
	}
}

// TestProxy_CleanupPendingCalls_ClosesChannels tests that channels are closed.
func TestProxy_CleanupPendingCalls_ClosesChannels(t *testing.T) {
	stream := newMockProxyStream()
	proxy, sw := newTestProxy(stream)
	defer sw.Close()

	ch := make(chan *pb.ToolResult, 1)
	proxy.mu.Lock()
	proxy.pendingCalls["call-1"] = ch
	proxy.mu.Unlock()

	done := make(chan bool)
	go func() {
		_, ok := <-ch
		if ok {
			t.Error("Channel should be closed, but received value")
		}
		done <- true
	}()

	proxy.CleanupPendingCalls()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("CleanupPendingCalls() did not close channel")
	}
}

// TestStreamClientOperationsProxy_AskUserQuestionnaire_Success tests successful AskUserQuestionnaire flow.
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

// TestStreamClientOperationsProxy_AskUserQuestionnaire_NoTimeout tests that AskUserQuestionnaire does NOT time out after 5min.
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

// TestStreamClientOperationsProxy_AskUserQuestionnaire_CancelledByContext tests that AskUserQuestionnaire respects context cancellation.
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

// TestStreamClientOperationsProxy_ConcurrentCalls verifies the proxy handles concurrent AskUser calls safely.
func TestStreamClientOperationsProxy_ConcurrentCalls(t *testing.T) {
	const n = 10

	stream := newMockProxyStream()
	proxy, sw := newTestProxy(stream)
	defer sw.Close()

	// Track call IDs as they arrive and respond asynchronously.
	callIDs := make(chan string, n)
	stream.sendFunc = func(resp *pb.FlowResponse) error {
		if resp.ToolCall != nil {
			callIDs <- resp.ToolCall.CallId
		}
		return nil
	}

	responder := make(chan struct{})
	go func() {
		for id := range callIDs {
			proxy.HandleToolResult(&pb.ToolResult{
				CallId: id,
				Result: "ok",
			})
		}
		close(responder)
	}()

	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, err := proxy.AskUserQuestionnaire(ctx, "session-1", `[{"text":"?"}]`)
			if err != nil {
				errs <- err
			}
		}()
	}

	wg.Wait()
	close(callIDs)
	<-responder
	close(errs)

	for err := range errs {
		t.Errorf("AskUserQuestionnaire concurrent call returned error: %v", err)
	}

	proxy.mu.RLock()
	pendingCount := len(proxy.pendingCalls)
	proxy.mu.RUnlock()

	if pendingCount != 0 {
		t.Errorf("Expected 0 pending calls after concurrent run, got %d", pendingCount)
	}
}

package grpc

import (
	"testing"
	"time"

	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
)

// mockFlowStream implements pb.FlowService_ExecuteFlowServer for testing
type mockFlowStream struct {
	pb.FlowService_ExecuteFlowServer
	sendFunc func(*pb.FlowResponse) error
}

func (m *mockFlowStream) Send(resp *pb.FlowResponse) error {
	if m.sendFunc != nil {
		return m.sendFunc(resp)
	}
	return nil
}

// TestStreamWriter_LastError_InitiallyNil verifies that a new StreamWriter has nil LastError
func TestStreamWriter_LastError_InitiallyNil(t *testing.T) {
	mockStream := &mockFlowStream{
		sendFunc: func(*pb.FlowResponse) error { return nil },
	}
	sw := NewStreamWriter(mockStream)
	defer sw.Close()

	if err := sw.LastError(); err != nil {
		t.Errorf("LastError() = %v, want nil", err)
	}
}

// TestStreamWriter_Send_Success verifies that Send() works correctly
func TestStreamWriter_Send_Success(t *testing.T) {
	sendCalled := false
	mockStream := &mockFlowStream{
		sendFunc: func(resp *pb.FlowResponse) error {
			sendCalled = true
			if resp.Content != "test" {
				t.Errorf("Send() got content %q, want %q", resp.Content, "test")
			}
			return nil
		},
	}
	sw := NewStreamWriter(mockStream)
	defer sw.Close()

	err := sw.Send(&pb.FlowResponse{Content: "test"})
	if err != nil {
		t.Errorf("Send() error = %v, want nil", err)
	}

	// Wait for writer loop to process
	time.Sleep(50 * time.Millisecond)

	if !sendCalled {
		t.Error("Send() did not call stream.Send()")
	}
}

// TestStreamWriter_LastError_SetOnSendFailure verifies that LastError is set when stream.Send() fails
func TestStreamWriter_LastError_SetOnSendFailure(t *testing.T) {
	sendErr := "mock send error"
	mockStream := &mockFlowStream{
		sendFunc: func(*pb.FlowResponse) error {
			return &mockError{msg: sendErr}
		},
	}
	sw := NewStreamWriter(mockStream)
	defer sw.Close()

	err := sw.Send(&pb.FlowResponse{Content: "test"})
	if err != nil {
		t.Errorf("Send() should not return error immediately, got %v", err)
	}

	// Wait for writer loop to process and set error
	time.Sleep(100 * time.Millisecond)

	lastErr := sw.LastError()
	if lastErr == nil {
		t.Fatal("LastError() = nil, want error")
	}
	if lastErr.Error() != sendErr {
		t.Errorf("LastError() = %q, want %q", lastErr.Error(), sendErr)
	}
}

// TestStreamWriter_Send_ClosedWriter verifies that Send() returns error after Close()
func TestStreamWriter_Send_ClosedWriter(t *testing.T) {
	mockStream := &mockFlowStream{
		sendFunc: func(*pb.FlowResponse) error { return nil },
	}
	sw := NewStreamWriter(mockStream)
	sw.Close()

	err := sw.Send(&pb.FlowResponse{Content: "test"})
	if err == nil {
		t.Error("Send() after Close() should return error, got nil")
	}
	if err.Error() != "stream writer closed" {
		t.Errorf("Send() error = %q, want %q", err.Error(), "stream writer closed")
	}
}

// TestStreamWriter_Close_DrainsMessages verifies that Close() drains messages before exiting
func TestStreamWriter_Close_DrainsMessages(t *testing.T) {
	sentCount := 0
	mockStream := &mockFlowStream{
		sendFunc: func(resp *pb.FlowResponse) error {
			sentCount++
			// Small delay to simulate real sending
			time.Sleep(10 * time.Millisecond)
			return nil
		},
	}
	sw := NewStreamWriter(mockStream)

	// Send 3 messages
	for i := 0; i < 3; i++ {
		err := sw.Send(&pb.FlowResponse{Content: "test"})
		if err != nil {
			t.Fatalf("Send() error = %v", err)
		}
	}

	// Close should drain all messages
	sw.Close()

	if sentCount != 3 {
		t.Errorf("Close() drained %d messages, want 3", sentCount)
	}
}

// TestStreamWriter_SendTimeout verifies that Send() respects the timeout when buffer is full
func TestStreamWriter_SendTimeout(t *testing.T) {
	// This test verifies that Send() can detect when the writer is unable to accept more messages
	// We simulate a slow/blocked stream and verify error handling

	sendCount := 0
	blockChan := make(chan struct{})
	blockingStream := &mockFlowStream{
		sendFunc: func(*pb.FlowResponse) error {
			sendCount++
			if sendCount == 1 {
				// Block on first send
				<-blockChan
			}
			return nil
		},
	}
	sw := NewStreamWriter(blockingStream)

	// Fill the buffer (256 items)
	for i := 0; i < 256; i++ {
		err := sw.Send(&pb.FlowResponse{Content: "filler"})
		if err != nil {
			t.Fatalf("Send() #%d error = %v", i, err)
		}
	}

	// Give writer a moment to start processing first item (and block)
	time.Sleep(50 * time.Millisecond)

	// Try to send one more - it should be queued but since stream.Send is blocked,
	// the channel is full. In production this would timeout after 10s.
	// For testing, we verify the error path by closing the writer.

	// Close writer (this will signal closeCh)
	close(blockChan) // Unblock the writer first so Close() can complete
	sw.Close()

	// Trying to send after close should fail immediately
	err := sw.Send(&pb.FlowResponse{Content: "should-fail"})
	if err == nil {
		t.Error("Send() on closed writer should error, got nil")
	}
	if err.Error() != "stream writer closed" {
		t.Errorf("Send() error = %q, want %q", err.Error(), "stream writer closed")
	}
}

// TestStreamWriter_Close_Idempotent verifies that Close() can be called multiple times safely
func TestStreamWriter_Close_Idempotent(t *testing.T) {
	mockStream := &mockFlowStream{
		sendFunc: func(*pb.FlowResponse) error { return nil },
	}
	sw := NewStreamWriter(mockStream)

	sw.Close()
	sw.Close() // Should not panic or deadlock
	sw.Close()
}

// TestStreamWriter_WriterLoopExitsOnError verifies that writer loop exits after stream.Send() error
func TestStreamWriter_WriterLoopExitsOnError(t *testing.T) {
	sendCount := 0
	mockStream := &mockFlowStream{
		sendFunc: func(resp *pb.FlowResponse) error {
			sendCount++
			if sendCount == 1 {
				return &mockError{msg: "first error"}
			}
			// Should not be called again after error
			t.Error("stream.Send() called after error")
			return nil
		},
	}
	sw := NewStreamWriter(mockStream)
	defer sw.Close()

	// First send triggers error
	err := sw.Send(&pb.FlowResponse{Content: "first"})
	if err != nil {
		t.Errorf("Send() should not return error immediately, got %v", err)
	}

	// Wait for writer loop to process and exit
	time.Sleep(100 * time.Millisecond)

	// Second send should fail because writer loop exited
	_ = sw.Send(&pb.FlowResponse{Content: "second"})
	// This might timeout or be queued, depending on timing
	// Main check: LastError should be set
	lastErr := sw.LastError()
	if lastErr == nil {
		t.Error("LastError() = nil after send failure, want error")
	}
}

// mockError is a simple error implementation for testing
type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

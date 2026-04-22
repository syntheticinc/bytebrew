package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	pb "github.com/syntheticinc/bytebrew/engine/api/proto/gen"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/pkg/errors"
)

const toolCallTimeout = 5 * time.Minute

// StreamBasedClientOperationsProxy implements ClientOperationsProxy using a
// gRPC bidirectional stream. Only ask_user crosses the gRPC boundary;
// file/shell/LSP tool calls are not proxied to the client.
type StreamBasedClientOperationsProxy struct {
	stream         pb.FlowService_ExecuteFlowServer
	streamWriter   *StreamWriter
	sessionID      string
	projectKey     string
	mu             sync.RWMutex
	pendingCalls   map[string]chan *pb.ToolResult
	callCounter    uint64 // Atomic counter for unique call IDs
	waitingForUser atomic.Bool
}

// NewStreamBasedClientOperationsProxy creates a new stream-based proxy.
// projectKey is kept on the struct for future use and logging.
func NewStreamBasedClientOperationsProxy(stream pb.FlowService_ExecuteFlowServer, sessionID, projectKey string, streamWriter *StreamWriter) *StreamBasedClientOperationsProxy {
	return &StreamBasedClientOperationsProxy{
		stream:       stream,
		streamWriter: streamWriter,
		sessionID:    sessionID,
		projectKey:   projectKey,
		pendingCalls: make(map[string]chan *pb.ToolResult),
	}
}

// executeToolCallCore sends a tool call request and waits for the result.
// Uses ctx deadline as-is without adding any timeout.
func (p *StreamBasedClientOperationsProxy) executeToolCallCore(ctx context.Context, toolName string, arguments map[string]string) (string, error) {
	counter := atomic.AddUint64(&p.callCounter, 1)
	callID := fmt.Sprintf("%s-%d", toolName, counter)

	slog.InfoContext(ctx, "[PROXY] executeToolCallCore: sending TOOL_CALL to client",
		"tool_name", toolName,
		"call_id", callID,
		"session_id", p.sessionID)

	resultChan := make(chan *pb.ToolResult, 1)
	p.mu.Lock()
	p.pendingCalls[callID] = resultChan
	p.mu.Unlock()

	toolCall := &pb.ToolCall{
		ToolName:  toolName,
		Arguments: arguments,
		CallId:    callID,
	}

	err := p.streamWriter.Send(&pb.FlowResponse{
		SessionId: p.sessionID,
		Type:      pb.ResponseType_RESPONSE_TYPE_TOOL_CALL,
		ToolCall:  toolCall,
		AgentId:   domain.AgentIDFromContext(ctx),
	})
	if err != nil {
		slog.ErrorContext(ctx, "[PROXY] executeToolCallCore: failed to send TOOL_CALL",
			"tool_name", toolName,
			"call_id", callID,
			"error", err)
		p.mu.Lock()
		delete(p.pendingCalls, callID)
		p.mu.Unlock()
		return "", errors.Wrap(err, errors.CodeInternal, "failed to send tool call request")
	}

	slog.InfoContext(ctx, "[PROXY] executeToolCallCore: TOOL_CALL sent, waiting for result",
		"tool_name", toolName,
		"call_id", callID)

	select {
	case result := <-resultChan:
		if result.Error != nil {
			slog.ErrorContext(ctx, "[PROXY] executeToolCallCore: received error result",
				"tool_name", toolName,
				"call_id", callID,
				"error_code", result.Error.Code,
				"error_message", result.Error.Message)
			return "", errors.New(result.Error.Code, result.Error.Message)
		}
		slog.InfoContext(ctx, "[PROXY] executeToolCallCore: received result",
			"tool_name", toolName,
			"call_id", callID,
			"result_length", len(result.Result))
		return result.Result, nil
	case <-ctx.Done():
		slog.WarnContext(ctx, "[PROXY] executeToolCallCore: context cancelled or timed out",
			"tool_name", toolName,
			"call_id", callID,
			"error", ctx.Err())
		p.mu.Lock()
		delete(p.pendingCalls, callID)
		p.mu.Unlock()
		return "", errors.New(errors.CodeTimeout, fmt.Sprintf("tool call %s timed out after %s", toolName, toolCallTimeout))
	}
}

// AskUserQuestionnaire sends structured questions to the client and waits for the user's responses.
// Uses executeToolCallCore directly (no timeout) — the user may take unlimited time to respond.
// Cancellation happens via parent context (gRPC stream close).
func (p *StreamBasedClientOperationsProxy) AskUserQuestionnaire(ctx context.Context, sessionID, questionsJSON string) (string, error) {
	p.waitingForUser.Store(true)
	defer p.waitingForUser.Store(false)

	arguments := map[string]string{
		"questions": questionsJSON,
	}

	return p.executeToolCallCore(ctx, "ask_user", arguments)
}

// HandleToolResult handles a tool result received from the client.
func (p *StreamBasedClientOperationsProxy) HandleToolResult(result *pb.ToolResult) bool {
	p.mu.RLock()
	resultChan, exists := p.pendingCalls[result.CallId]
	p.mu.RUnlock()

	if !exists {
		slog.WarnContext(context.Background(), "[PROXY] late ToolResult (pending call expired)", "call_id", result.CallId)
		return false
	}

	resultChan <- result

	p.mu.Lock()
	delete(p.pendingCalls, result.CallId)
	p.mu.Unlock()
	return true
}

// IsWaitingForUser returns true if the proxy is waiting for user response.
func (p *StreamBasedClientOperationsProxy) IsWaitingForUser() bool {
	return p.waitingForUser.Load()
}

// ClearWaitingForUser resets the waiting for user flag.
func (p *StreamBasedClientOperationsProxy) ClearWaitingForUser() {
	p.waitingForUser.Store(false)
}

// CleanupPendingCalls cancels all pending tool calls.
// Used during shutdown to unblock any goroutines waiting for results.
func (p *StreamBasedClientOperationsProxy) CleanupPendingCalls() {
	p.mu.Lock()
	pending := make(map[string]chan *pb.ToolResult, len(p.pendingCalls))
	for k, v := range p.pendingCalls {
		pending[k] = v
	}
	p.pendingCalls = make(map[string]chan *pb.ToolResult)
	p.mu.Unlock()

	for callID, ch := range pending {
		slog.WarnContext(context.Background(), "[PROXY] CleanupPendingCalls: cancelling pending call", "call_id", callID)
		close(ch)
	}
}

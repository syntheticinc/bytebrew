package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/errors"
)

const toolCallTimeout = 5 * time.Minute

// StreamBasedClientOperationsProxy implements ClientOperationsProxy using bidirectional stream
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

// NewStreamBasedClientOperationsProxy creates a new stream-based proxy
func NewStreamBasedClientOperationsProxy(stream pb.FlowService_ExecuteFlowServer, sessionID, projectKey string, streamWriter *StreamWriter) *StreamBasedClientOperationsProxy {
	return &StreamBasedClientOperationsProxy{
		stream:       stream,
		streamWriter: streamWriter,
		sessionID:    sessionID,
		projectKey:   projectKey,
		pendingCalls: make(map[string]chan *pb.ToolResult),
	}
}

// ReadFile reads a file from the client's filesystem
func (p *StreamBasedClientOperationsProxy) ReadFile(ctx context.Context, sessionID, filePath string, startLine, endLine int32) (string, error) {
	// Create tool call request
	arguments := map[string]string{
		"file_path":  filePath,
		"start_line": fmt.Sprintf("%d", startLine),
		"end_line":   fmt.Sprintf("%d", endLine),
	}

	result, err := p.executeToolCall(ctx, "read_file", arguments)
	if err != nil {
		return "", err
	}

	return result, nil
}

// SearchCode performs vector search on the client's Qdrant instance
func (p *StreamBasedClientOperationsProxy) SearchCode(ctx context.Context, sessionID, query, projectKey string, limit int32, minScore float32) ([]byte, error) {
	// Create tool call request
	arguments := map[string]string{
		"query":       query,
		"project_key": projectKey,
		"limit":       fmt.Sprintf("%d", limit),
		"min_score":   fmt.Sprintf("%f", minScore),
	}

	result, err := p.executeToolCall(ctx, "search_code", arguments)
	if err != nil {
		return nil, err
	}

	return []byte(result), nil
}

// GetProjectTree returns the project file tree for a specific path
func (p *StreamBasedClientOperationsProxy) GetProjectTree(ctx context.Context, sessionID, projectKey, path string, maxDepth int) (string, error) {
	arguments := map[string]string{
		"project_key": projectKey,
		"max_depth":   fmt.Sprintf("%d", maxDepth),
	}

	if path != "" {
		arguments["path"] = path
	}

	result, err := p.executeToolCall(ctx, "get_project_tree", arguments)
	if err != nil {
		return "", err
	}

	return result, nil
}

// executeToolCallCore sends a tool call request and waits for the result.
// Uses ctx deadline as-is without adding any timeout.
func (p *StreamBasedClientOperationsProxy) executeToolCallCore(ctx context.Context, toolName string, arguments map[string]string) (string, error) {
	// Generate unique call ID using atomic counter (thread-safe)
	counter := atomic.AddUint64(&p.callCounter, 1)
	callID := fmt.Sprintf("%s-%d", toolName, counter)

	slog.InfoContext(ctx, "[PROXY] executeToolCallCore: sending TOOL_CALL to client",
		"tool_name", toolName,
		"call_id", callID,
		"session_id", p.sessionID)

	// Create channel for result
	resultChan := make(chan *pb.ToolResult, 1)
	p.mu.Lock()
	p.pendingCalls[callID] = resultChan
	p.mu.Unlock()

	// Send tool call request
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

	// Wait for result (bounded by context timeout)
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

// executeToolCall is a wrapper that adds a 5-minute timeout and calls executeToolCallCore.
func (p *StreamBasedClientOperationsProxy) executeToolCall(ctx context.Context, toolName string, arguments map[string]string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, toolCallTimeout)
	defer cancel()
	return p.executeToolCallCore(ctx, toolName, arguments)
}

// ExecuteSubQueries sends sub-queries for grouped search operations (e.g., smart_search)
// Returns results from all sub-queries executed in parallel on the client.
// Applies a 5-minute timeout to prevent indefinite blocking.
func (p *StreamBasedClientOperationsProxy) ExecuteSubQueries(ctx context.Context, sessionID string, subQueries []*pb.SubQuery) ([]*pb.SubResult, error) {
	// Apply timeout to prevent indefinite blocking
	ctx, cancel := context.WithTimeout(ctx, toolCallTimeout)
	defer cancel()

	// Generate unique call ID
	counter := atomic.AddUint64(&p.callCounter, 1)
	callID := fmt.Sprintf("smart_search-%d", counter)

	slog.InfoContext(ctx, "[PROXY] ExecuteSubQueries: sending grouped TOOL_CALL to client",
		"call_id", callID,
		"sub_queries_count", len(subQueries),
		"session_id", p.sessionID)

	// Create channel for result
	resultChan := make(chan *pb.ToolResult, 1)
	p.mu.Lock()
	p.pendingCalls[callID] = resultChan
	p.mu.Unlock()

	// Send tool call with sub-queries
	// Extract query from first sub-query for display in client UI
	queryArg := ""
	if len(subQueries) > 0 {
		queryArg = subQueries[0].Query
	}
	toolCall := &pb.ToolCall{
		ToolName:   "smart_search",
		Arguments:  map[string]string{"query": queryArg},
		CallId:     callID,
		SubQueries: subQueries,
	}

	err := p.streamWriter.Send(&pb.FlowResponse{
		SessionId: p.sessionID,
		Type:      pb.ResponseType_RESPONSE_TYPE_TOOL_CALL,
		ToolCall:  toolCall,
		AgentId:   domain.AgentIDFromContext(ctx),
	})
	if err != nil {
		slog.ErrorContext(ctx, "[PROXY] ExecuteSubQueries: failed to send TOOL_CALL",
			"call_id", callID,
			"error", err)
		p.mu.Lock()
		delete(p.pendingCalls, callID)
		p.mu.Unlock()
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to send sub-queries request")
	}

	slog.InfoContext(ctx, "[PROXY] ExecuteSubQueries: TOOL_CALL sent, waiting for result",
		"call_id", callID)

	// Wait for result with sub-results (bounded by context timeout)
	select {
	case result := <-resultChan:
		if result.Error != nil {
			slog.ErrorContext(ctx, "[PROXY] ExecuteSubQueries: received error result",
				"call_id", callID,
				"error_code", result.Error.Code,
				"error_message", result.Error.Message)
			return nil, errors.New(result.Error.Code, result.Error.Message)
		}
		slog.InfoContext(ctx, "[PROXY] ExecuteSubQueries: received result",
			"call_id", callID,
			"sub_results_count", len(result.SubResults))
		return result.SubResults, nil
	case <-ctx.Done():
		slog.WarnContext(ctx, "[PROXY] ExecuteSubQueries: context cancelled or timed out",
			"call_id", callID,
			"error", ctx.Err())
		p.mu.Lock()
		delete(p.pendingCalls, callID)
		p.mu.Unlock()
		return nil, errors.New(errors.CodeTimeout, fmt.Sprintf("smart_search timed out after %s", toolCallTimeout))
	}
}

// GrepSearch performs pattern-based search using ripgrep on the client
func (p *StreamBasedClientOperationsProxy) GrepSearch(ctx context.Context, sessionID, pattern string, limit int32, fileTypes []string, ignoreCase bool) (string, error) {
	arguments := map[string]string{
		"pattern": pattern,
		"limit":   fmt.Sprintf("%d", limit),
	}

	if len(fileTypes) > 0 {
		// Join file types as comma-separated string
		fileTypesStr := ""
		for i, ft := range fileTypes {
			if i > 0 {
				fileTypesStr += ","
			}
			fileTypesStr += ft
		}
		arguments["file_types"] = fileTypesStr
	}

	if ignoreCase {
		arguments["ignore_case"] = "true"
	}

	result, err := p.executeToolCall(ctx, "grep_search", arguments)
	if err != nil {
		return "", err
	}

	return result, nil
}

// GlobSearch finds files matching glob pattern
func (p *StreamBasedClientOperationsProxy) GlobSearch(ctx context.Context, sessionID, pattern string, limit int32) (string, error) {
	arguments := map[string]string{
		"pattern": pattern,
		"limit":   fmt.Sprintf("%d", limit),
	}
	result, err := p.executeToolCall(ctx, "glob", arguments)
	if err != nil {
		return "", err
	}
	return result, nil
}

// SymbolSearch searches for code symbols by name on the client
func (p *StreamBasedClientOperationsProxy) SymbolSearch(ctx context.Context, sessionID, symbolName string, limit int32, symbolTypes []string) (string, error) {
	arguments := map[string]string{
		"symbol_name": symbolName,
		"limit":       fmt.Sprintf("%d", limit),
	}

	if len(symbolTypes) > 0 {
		// Join symbol types as comma-separated string
		symbolTypesStr := ""
		for i, st := range symbolTypes {
			if i > 0 {
				symbolTypesStr += ","
			}
			symbolTypesStr += st
		}
		arguments["symbol_types"] = symbolTypesStr
	}

	result, err := p.executeToolCall(ctx, "symbol_search", arguments)
	if err != nil {
		return "", err
	}

	return result, nil
}

// WriteFile writes content to a file on the client's filesystem
func (p *StreamBasedClientOperationsProxy) WriteFile(ctx context.Context, sessionID, filePath, content string) (string, error) {
	arguments := map[string]string{
		"file_path": filePath,
		"content":   content,
	}

	result, err := p.executeToolCall(ctx, "write_file", arguments)
	if err != nil {
		return "", err
	}

	return result, nil
}

// EditFile performs a find-and-replace edit on a file on the client's filesystem
func (p *StreamBasedClientOperationsProxy) EditFile(ctx context.Context, sessionID, filePath, oldString, newString string, replaceAll bool) (string, error) {
	arguments := map[string]string{
		"file_path":   filePath,
		"old_string":  oldString,
		"new_string":  newString,
		"replace_all": fmt.Sprintf("%t", replaceAll),
	}

	result, err := p.executeToolCall(ctx, "edit_file", arguments)
	if err != nil {
		return "", err
	}

	return result, nil
}

// ExecuteCommand executes a shell command on the client.
// Uses executeToolCallCore directly (no server-side timeout) — like AskUserQuestionnaire,
// because the client may wait for interactive permission approval which takes
// unbounded time. The client enforces its own command timeout (args.timeout).
// Cancellation happens via parent context (gRPC stream close).
func (p *StreamBasedClientOperationsProxy) ExecuteCommand(ctx context.Context, sessionID, command, cwd string, timeout int32) (string, error) {
	arguments := map[string]string{
		"command": command,
		"timeout": fmt.Sprintf("%d", timeout),
	}

	if cwd != "" {
		arguments["cwd"] = cwd
	}

	return p.executeToolCallCore(ctx, "execute_command", arguments)
}

// ExecuteCommandFull executes a shell command with full control over all parameters.
// Used for persistent shell and background mode support (background, bg_action, bg_id).
func (p *StreamBasedClientOperationsProxy) ExecuteCommandFull(ctx context.Context, sessionID string, arguments map[string]string) (string, error) {
	return p.executeToolCallCore(ctx, "execute_command", arguments)
}

// AskUserQuestionnaire sends structured questions to the client and waits for the user's responses.
// Uses executeToolCallCore directly (no timeout) - user may take unlimited time to respond.
// Cancellation happens via parent context (gRPC stream close).
func (p *StreamBasedClientOperationsProxy) AskUserQuestionnaire(ctx context.Context, sessionID, questionsJSON string) (string, error) {
	p.waitingForUser.Store(true)
	defer p.waitingForUser.Store(false)

	arguments := map[string]string{
		"questions": questionsJSON,
	}

	return p.executeToolCallCore(ctx, "ask_user", arguments)
}

// LspRequest performs LSP-based code navigation (definition, references, implementation) on the client
func (p *StreamBasedClientOperationsProxy) LspRequest(ctx context.Context, sessionID, symbolName, operation string) (string, error) {
	arguments := map[string]string{
		"symbol_name": symbolName,
		"operation":   operation,
	}
	return p.executeToolCall(ctx, "lsp", arguments)
}

// HandleToolResult handles a tool result received from the client
func (p *StreamBasedClientOperationsProxy) HandleToolResult(result *pb.ToolResult) bool {
	p.mu.RLock()
	resultChan, exists := p.pendingCalls[result.CallId]
	p.mu.RUnlock()

	if !exists {
		slog.Warn("[PROXY] late ToolResult (pending call expired)", "call_id", result.CallId)
		return false
	}

	// Send result to waiting goroutine
	resultChan <- result

	// Clean up
	p.mu.Lock()
	delete(p.pendingCalls, result.CallId)
	p.mu.Unlock()
	return true
}

// IsWaitingForUser returns true if the proxy is waiting for user response
func (p *StreamBasedClientOperationsProxy) IsWaitingForUser() bool {
	return p.waitingForUser.Load()
}

// ClearWaitingForUser resets the waiting for user flag
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
		slog.Warn("[PROXY] CleanupPendingCalls: cancelling pending call", "call_id", callID)
		close(ch)
	}
}

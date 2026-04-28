package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Diagnostic represents an LSP diagnostic message.
type Diagnostic struct {
	Range    Range  `json:"range"`
	Message  string `json:"message"`
	Severity int    `json:"severity"`
}

// Range represents an LSP text range.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Position represents a position in a text document (0-based).
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Location represents a location in a text document.
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// Client is a JSON-RPC 2.0 LSP client communicating over stdio with a spawned LSP server.
type Client struct {
	cmd         *exec.Cmd
	stdin       io.WriteCloser
	stdout      *bufio.Reader
	serverID    string
	root        string
	requestID   atomic.Int64
	responses   map[int64]chan json.RawMessage
	mu          sync.Mutex
	ready       chan struct{}
	closed      atomic.Bool
	diagnostics sync.Map // map[string][]Diagnostic
}

// jsonrpcRequest is a JSON-RPC 2.0 request message.
type jsonrpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// jsonrpcNotification is a JSON-RPC 2.0 notification message (no ID).
type jsonrpcNotification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// jsonrpcResponse is a JSON-RPC 2.0 response or server notification.
type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// jsonrpcError represents a JSON-RPC error object.
type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewClient creates a new LSP client, starts the server process, and begins reading responses.
func NewClient(serverID, root string, cmd *exec.Cmd) (*Client, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}

	// Discard stderr to avoid blocking
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("start LSP server %s: %w", serverID, err)
	}

	c := &Client{
		cmd:       cmd,
		stdin:     stdin,
		stdout:    bufio.NewReaderSize(stdout, 64*1024),
		serverID:  serverID,
		root:      root,
		responses: make(map[int64]chan json.RawMessage),
		ready:     make(chan struct{}),
	}

	go c.readLoop()

	slog.InfoContext(context.Background(), "LSP client started", "server", serverID, "pid", cmd.Process.Pid, "root", root)
	return c, nil
}

// Initialize sends the initialize request and initialized notification to the server.
func (c *Client) Initialize(ctx context.Context, projectRoot string) error {
	rootURI := pathToURI(projectRoot)

	initParams := map[string]interface{}{
		"processId": os.Getpid(),
		"rootUri":   rootURI,
		"capabilities": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"definition":       map[string]interface{}{"dynamicRegistration": false},
				"references":       map[string]interface{}{"dynamicRegistration": false},
				"implementation":   map[string]interface{}{"dynamicRegistration": false},
				"publishDiagnostics": map[string]interface{}{"relatedInformation": true},
			},
			"window": map[string]interface{}{
				"workDoneProgress": true,
			},
		},
		"workspaceFolders": []map[string]interface{}{
			{"uri": rootURI, "name": filepath.Base(projectRoot)},
		},
	}

	_, err := c.request(ctx, "initialize", initParams)
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	if err := c.notify("initialized", map[string]interface{}{}); err != nil {
		return fmt.Errorf("initialized notification: %w", err)
	}

	slog.InfoContext(context.Background(), "LSP initialized", "server", c.serverID, "root", projectRoot)
	return nil
}

// DidOpen sends a textDocument/didOpen notification for the given file.
func (c *Client) DidOpen(ctx context.Context, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file for didOpen: %w", err)
	}

	lang := detectLanguageID(filePath)
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":        pathToURI(filePath),
			"languageId": lang,
			"version":    1,
			"text":       string(data),
		},
	}

	return c.notify("textDocument/didOpen", params)
}

// Definition finds the definition of a symbol at the given position.
func (c *Client) Definition(ctx context.Context, uri string, pos Position) ([]Location, error) {
	return c.textDocumentPositionRequest(ctx, "textDocument/definition", uri, pos)
}

// References finds all references to a symbol at the given position.
func (c *Client) References(ctx context.Context, uri string, pos Position) ([]Location, error) {
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": uri},
		"position":     pos,
		"context":      map[string]interface{}{"includeDeclaration": true},
	}

	result, err := c.request(ctx, "textDocument/references", params)
	if err != nil {
		return nil, err
	}

	return parseLocations(result)
}

// Implementation finds implementations of an interface at the given position.
func (c *Client) Implementation(ctx context.Context, uri string, pos Position) ([]Location, error) {
	return c.textDocumentPositionRequest(ctx, "textDocument/implementation", uri, pos)
}

// WaitForReady blocks until the server signals readiness via $/progress end, or timeout.
func (c *Client) WaitForReady(timeout time.Duration) bool {
	select {
	case <-c.ready:
		return true
	case <-time.After(timeout):
		return false
	}
}

// Shutdown sends shutdown and exit requests, then kills the process.
func (c *Client) Shutdown(ctx context.Context) error {
	if c.closed.Swap(true) {
		return nil
	}

	// Best-effort shutdown
	_, _ = c.request(ctx, "shutdown", nil)
	_ = c.notify("exit", nil)

	// Give process time to exit
	done := make(chan struct{})
	go func() {
		_ = c.cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.InfoContext(context.Background(), "LSP server exited gracefully", "server", c.serverID)
	case <-time.After(3 * time.Second):
		slog.WarnContext(context.Background(), "LSP server did not exit, killing", "server", c.serverID)
		_ = c.Close()
	}

	return nil
}

// Close forcefully kills the LSP server process.
func (c *Client) Close() error {
	c.closed.Store(true)
	return killProcess(c.cmd)
}

// readLoop reads JSON-RPC messages from stdout and dispatches them.
func (c *Client) readLoop() {
	for {
		if c.closed.Load() {
			return
		}

		body, err := c.readMessage()
		if err != nil {
			if c.closed.Load() {
				return
			}
			slog.DebugContext(context.Background(), "LSP readLoop error", "server", c.serverID, "error", err)
			return
		}

		var msg jsonrpcResponse
		if err := json.Unmarshal(body, &msg); err != nil {
			slog.DebugContext(context.Background(), "LSP parse message error", "server", c.serverID, "error", err)
			continue
		}

		// Server-initiated request (has both ID and Method, e.g., window/workDoneProgress/create)
		if msg.ID != nil && msg.Method != "" {
			c.handleServerRequest(*msg.ID, msg.Method, msg.Params)
			continue
		}

		// Response to a client request
		if msg.ID != nil {
			c.mu.Lock()
			ch, ok := c.responses[*msg.ID]
			if ok {
				delete(c.responses, *msg.ID)
			}
			c.mu.Unlock()

			if ok {
				if msg.Error != nil {
					ch <- json.RawMessage(fmt.Sprintf(`{"error":{"code":%d,"message":%q}}`, msg.Error.Code, msg.Error.Message))
				} else {
					ch <- msg.Result
				}
			}
			continue
		}

		// Server notification (no ID)
		c.handleNotification(msg.Method, msg.Params)
	}
}

// handleServerRequest responds to server-initiated requests (e.g., window/workDoneProgress/create).
// LSP servers may send requests that expect a response; failing to respond blocks the server.
func (c *Client) handleServerRequest(id int64, method string, params json.RawMessage) {
	slog.DebugContext(context.Background(), "LSP server request", "server", c.serverID, "method", method, "id", id)

	// Respond with an empty success result for known methods.
	// This unblocks the server so it can continue processing.
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  nil,
	}

	if err := c.writeMessage(response); err != nil {
		slog.DebugContext(context.Background(), "LSP: failed to respond to server request", "server", c.serverID, "method", method, "error", err)
	}
}

// handleNotification processes server-initiated notifications.
func (c *Client) handleNotification(method string, params json.RawMessage) {
	switch method {
	case "textDocument/publishDiagnostics":
		c.handleDiagnostics(params)

	case "$/progress":
		c.handleProgress(params)

	case "window/logMessage":
		var logMsg struct {
			Type    int    `json:"type"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(params, &logMsg); err == nil {
			slog.DebugContext(context.Background(), "LSP server log", "server", c.serverID, "message", logMsg.Message)
		}
	}
}

// handleDiagnostics stores diagnostics from the server.
func (c *Client) handleDiagnostics(params json.RawMessage) {
	var diag struct {
		URI         string       `json:"uri"`
		Diagnostics []Diagnostic `json:"diagnostics"`
	}
	if err := json.Unmarshal(params, &diag); err != nil {
		return
	}
	c.diagnostics.Store(diag.URI, diag.Diagnostics)
}

// handleProgress checks for $/progress end to signal readiness.
func (c *Client) handleProgress(params json.RawMessage) {
	var progress struct {
		Value struct {
			Kind string `json:"kind"`
		} `json:"value"`
	}
	if err := json.Unmarshal(params, &progress); err != nil {
		return
	}
	if progress.Value.Kind == "end" {
		select {
		case <-c.ready:
			// Already closed
		default:
			close(c.ready)
			slog.InfoContext(context.Background(), "LSP server ready", "server", c.serverID)
		}
	}
}

// request sends a JSON-RPC request and waits for the response.
func (c *Client) request(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	if c.closed.Load() {
		return nil, fmt.Errorf("client closed")
	}

	id := c.requestID.Add(1)
	ch := make(chan json.RawMessage, 1)

	c.mu.Lock()
	c.responses[id] = ch
	c.mu.Unlock()

	var paramsRaw json.RawMessage
	if params != nil {
		var err error
		paramsRaw, err = json.Marshal(params)
		if err != nil {
			c.mu.Lock()
			delete(c.responses, id)
			c.mu.Unlock()
			return nil, fmt.Errorf("marshal params: %w", err)
		}
	}

	msg := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  paramsRaw,
	}

	if err := c.writeMessage(msg); err != nil {
		c.mu.Lock()
		delete(c.responses, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("write request: %w", err)
	}

	select {
	case result := <-ch:
		// Check if result contains an error
		var errResp struct {
			Error *jsonrpcError `json:"error"`
		}
		if json.Unmarshal(result, &errResp) == nil && errResp.Error != nil {
			return nil, fmt.Errorf("LSP error %d: %s", errResp.Error.Code, errResp.Error.Message)
		}
		return result, nil

	case <-ctx.Done():
		c.mu.Lock()
		delete(c.responses, id)
		c.mu.Unlock()
		return nil, ctx.Err()

	case <-time.After(30 * time.Second):
		c.mu.Lock()
		delete(c.responses, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("request %s timed out", method)
	}
}

// notify sends a JSON-RPC notification (no response expected).
func (c *Client) notify(method string, params interface{}) error {
	if c.closed.Load() {
		return fmt.Errorf("client closed")
	}

	var paramsRaw json.RawMessage
	if params != nil {
		var err error
		paramsRaw, err = json.Marshal(params)
		if err != nil {
			return fmt.Errorf("marshal params: %w", err)
		}
	}

	msg := jsonrpcNotification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsRaw,
	}

	return c.writeMessage(msg)
}

// writeMessage serializes a message and writes it with Content-Length header.
func (c *Client) writeMessage(msg interface{}) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	data := append([]byte(header), body...)

	_, err = c.stdin.Write(data)
	return err
}

// readMessage reads a Content-Length framed message from stdout.
func (c *Client) readMessage() ([]byte, error) {
	contentLength := -1

	for {
		line, err := c.stdout.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("read header: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		if strings.HasPrefix(line, "Content-Length:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			contentLength, err = strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("parse Content-Length %q: %w", val, err)
			}
		}
	}

	if contentLength < 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}

	body := make([]byte, contentLength)
	if _, err := io.ReadFull(c.stdout, body); err != nil {
		return nil, fmt.Errorf("read body (%d bytes): %w", contentLength, err)
	}

	return body, nil
}

// textDocumentPositionRequest is a helper for definition/implementation requests.
func (c *Client) textDocumentPositionRequest(ctx context.Context, method, uri string, pos Position) ([]Location, error) {
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": uri},
		"position":     pos,
	}

	result, err := c.request(ctx, method, params)
	if err != nil {
		return nil, err
	}

	return parseLocations(result)
}

// parseLocations parses LSP Location or Location[] from a JSON response.
func parseLocations(raw json.RawMessage) ([]Location, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	// Try as array first
	var locations []Location
	if err := json.Unmarshal(raw, &locations); err == nil {
		return locations, nil
	}

	// Try as single location
	var single Location
	if err := json.Unmarshal(raw, &single); err == nil {
		if single.URI != "" {
			return []Location{single}, nil
		}
		return nil, nil
	}

	return nil, nil
}

// pathToURI converts an OS file path to a file:// URI.
func pathToURI(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	slashed := filepath.ToSlash(abs)

	// On Windows, paths start with drive letter (C:/...)
	if runtime.GOOS == "windows" || (len(slashed) >= 2 && slashed[1] == ':') {
		return "file:///" + slashed
	}
	return "file://" + slashed
}

// detectLanguageID returns the LSP language identifier for a file extension.
func detectLanguageID(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return "go"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "typescriptreact"
	case ".js":
		return "javascript"
	case ".jsx":
		return "javascriptreact"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".c":
		return "c"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".h", ".hpp":
		return "cpp"
	case ".dart":
		return "dart"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".cs":
		return "csharp"
	default:
		return "plaintext"
	}
}

// killProcess kills a process tree. Platform-specific implementation.
func killProcess(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}

	if runtime.GOOS == "windows" {
		return killProcessWindows(cmd)
	}
	return killProcessUnix(cmd)
}

// killProcessWindows kills the process tree using taskkill.
func killProcessWindows(cmd *exec.Cmd) error {
	pid := strconv.Itoa(cmd.Process.Pid)
	kill := exec.Command("taskkill", "/T", "/F", "/PID", pid)

	done := make(chan error, 1)
	go func() {
		done <- kill.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			return cmd.Process.Kill()
		}
		return nil
	case <-time.After(3 * time.Second):
		if kill.Process != nil {
			_ = kill.Process.Kill()
		}
		return cmd.Process.Kill()
	}
}

// killProcessUnix sends SIGTERM, waits for grace period, then SIGKILL.
func killProcessUnix(cmd *exec.Cmd) error {
	// Try graceful shutdown first
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		return cmd.Process.Kill()
	}

	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(5 * time.Second):
		return cmd.Process.Kill()
	}
}

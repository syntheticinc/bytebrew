package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// SSETransport connects to an MCP server via SSE (Server-Sent Events).
// Per MCP spec: GET /sse for server→client events, POST /message for client→server requests.
type SSETransport struct {
	baseURL        string
	sseClient      *http.Client // For SSE GET — no timeout (long-lived stream)
	postClient     *http.Client // For POST /message — with timeout
	messageURL     string       // discovered from SSE endpoint event
	forwardHeaders []string

	mu       sync.Mutex
	pending  map[interface{}]chan *Response
	cancel   context.CancelFunc
	closed   bool
}

// NewSSETransport creates a transport for MCP SSE servers.
// baseURL should be the SSE endpoint URL (e.g., "http://server:3001/sse").
func NewSSETransport(baseURL string, forwardHeaders ...[]string) *SSETransport {
	var fh []string
	if len(forwardHeaders) > 0 {
		fh = forwardHeaders[0]
	}
	return &SSETransport{
		baseURL:        baseURL,
		sseClient:      &http.Client{},                            // No timeout — SSE stream is persistent
		postClient:     &http.Client{Timeout: 30 * time.Second},   // Timeout for POST requests
		pending:        make(map[interface{}]chan *Response),
		forwardHeaders: fh,
	}
}

func (t *SSETransport) Start(_ context.Context) error {
	// Use background context for SSE connection lifecycle — it must outlive the
	// caller's context (e.g. connectCtx with 10s timeout). The SSE stream stays
	// open until Close() is called.
	sseCtx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel

	// Connect to SSE endpoint
	req, err := http.NewRequestWithContext(sseCtx, http.MethodGet, t.baseURL, nil)
	if err != nil {
		return fmt.Errorf("create SSE request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := t.sseClient.Do(req)
	if err != nil {
		return fmt.Errorf("connect to SSE: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return fmt.Errorf("SSE server returned %d", resp.StatusCode)
	}

	// Start reading SSE events in background
	go t.readSSE(sseCtx, resp.Body)

	// Wait briefly for endpoint event
	time.Sleep(100 * time.Millisecond)

	return nil
}

func (t *SSETransport) Send(ctx context.Context, req *Request) (*Response, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, fmt.Errorf("transport closed")
	}

	// Create response channel for this request ID
	ch := make(chan *Response, 1)
	t.pending[req.ID] = ch
	t.mu.Unlock()

	defer func() {
		t.mu.Lock()
		delete(t.pending, req.ID)
		t.mu.Unlock()
	}()

	// POST request to message endpoint
	msgURL := t.getMessageURL()
	if msgURL == "" {
		return nil, fmt.Errorf("message endpoint not discovered yet")
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, msgURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	t.applyForwardHeaders(ctx, httpReq)

	httpResp, err := t.postClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode >= 400 {
		return nil, fmt.Errorf("message endpoint returned %d", httpResp.StatusCode)
	}

	// Some MCP servers return the JSON-RPC response in the HTTP body
	// (not via SSE stream). Try to read it first.
	body, readErr := io.ReadAll(httpResp.Body)
	if readErr == nil && len(body) > 2 {
		var directResp Response
		if json.Unmarshal(body, &directResp) == nil && directResp.ID != nil {
			return &directResp, nil
		}
	}

	// Otherwise wait for response via SSE stream
	select {
	case resp := <-ch:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("timeout waiting for SSE response")
	}
}

func (t *SSETransport) Notify(ctx context.Context, req *Request) {
	msgURL := t.getMessageURL()
	if msgURL == "" {
		return
	}

	data, err := json.Marshal(req)
	if err != nil {
		return
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, msgURL, bytes.NewReader(data))
	if err != nil {
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	t.applyForwardHeaders(ctx, httpReq)

	resp, err := t.postClient.Do(httpReq)
	if err != nil {
		return
	}
	resp.Body.Close()
}

func (t *SSETransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.closed = true
	if t.cancel != nil {
		t.cancel()
	}

	// Close all pending channels
	for id, ch := range t.pending {
		close(ch)
		delete(t.pending, id)
	}

	return nil
}

func (t *SSETransport) getMessageURL() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.messageURL
}

func (t *SSETransport) setMessageURL(url string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// If relative URL, resolve against base
	if strings.HasPrefix(url, "/") {
		// Extract base from SSE URL (scheme + host)
		base := t.baseURL
		if idx := strings.Index(base, "://"); idx != -1 {
			rest := base[idx+3:]
			if slashIdx := strings.Index(rest, "/"); slashIdx != -1 {
				base = base[:idx+3+slashIdx]
			}
		}
		url = base + url
	}

	t.messageURL = url
}

// readSSE processes the SSE stream from the server.
func (t *SSETransport) readSSE(ctx context.Context, body io.ReadCloser) {
	defer body.Close()

	scanner := bufio.NewScanner(body)
	var eventType string

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()

		if line == "" {
			eventType = ""
			continue
		}

		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			t.handleSSEData(eventType, data)
		}
	}

	if err := scanner.Err(); err != nil {
		slog.Warn("SSE transport: stream error", "error", err)
	}
}

func (t *SSETransport) handleSSEData(eventType, data string) {
	switch eventType {
	case "endpoint":
		// Server announces its message endpoint
		t.setMessageURL(strings.TrimSpace(data))
		slog.Info("SSE transport: discovered message endpoint", "url", data)

	case "message":
		// JSON-RPC response
		var resp Response
		if err := json.Unmarshal([]byte(data), &resp); err != nil {
			slog.Warn("SSE transport: failed to parse response", "error", err)
			return
		}

		// Normalize response ID: JSON unmarshals numbers as float64,
		// but pending map keys are int64 from nextRequestID().
		normalizedID := normalizeID(resp.ID)

		t.mu.Lock()
		if ch, ok := t.pending[normalizedID]; ok {
			ch <- &resp
		}
		t.mu.Unlock()

	default:
		slog.Debug("SSE transport: unknown event", "type", eventType, "data", data[:min(len(data), 100)])
	}
}

// normalizeID converts JSON-unmarshalled float64 IDs back to int64 for map lookup.
// JSON numbers unmarshal as float64 in Go, but pending map keys are int64.
func normalizeID(id interface{}) interface{} {
	if f, ok := id.(float64); ok {
		return int64(f)
	}
	return id
}

// applyForwardHeaders copies configured headers from RequestContext to the HTTP request.
func (t *SSETransport) applyForwardHeaders(ctx context.Context, httpReq *http.Request) {
	if len(t.forwardHeaders) == 0 {
		return
	}
	rc := domain.GetRequestContext(ctx)
	if rc == nil {
		return
	}
	for _, headerName := range t.forwardHeaders {
		if val := rc.Get(headerName); val != "" {
			httpReq.Header.Set(headerName, val)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

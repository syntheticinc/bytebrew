package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sseServer is a test helper that simulates an MCP SSE server.
type sseServer struct {
	mu         sync.Mutex
	flusher    http.Flusher
	writer     http.ResponseWriter
	msgHandler func(w http.ResponseWriter, r *http.Request)
	sessionID  string
	httpServer *httptest.Server

	// sseConnections tracks how many SSE connections have been made
	sseConnections int
	connCh         chan struct{} // signaled on each new SSE connection
}

func newSSEServer() *sseServer {
	s := &sseServer{
		sessionID: "session-1",
		connCh:    make(chan struct{}, 10),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/sse", s.handleSSE)
	mux.HandleFunc("/message", s.handleMessage)

	s.httpServer = httptest.NewServer(mux)
	return s
}

func (s *sseServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	s.mu.Lock()
	s.writer = w
	s.flusher = flusher
	s.sseConnections++
	s.mu.Unlock()

	// Signal new connection
	select {
	case s.connCh <- struct{}{}:
	default:
	}

	// Send endpoint event
	fmt.Fprintf(w, "event: endpoint\ndata: %s/message?sessionId=%s\n\n", s.httpServer.URL, s.sessionID)
	flusher.Flush()

	// Keep connection open until client disconnects
	<-r.Context().Done()
}

func (s *sseServer) handleMessage(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	handler := s.msgHandler
	sid := s.sessionID
	s.mu.Unlock()

	// Check session ID
	querySessionID := r.URL.Query().Get("sessionId")
	if querySessionID != "" && querySessionID != sid {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error": "Invalid session ID: %s"}`, querySessionID)
		return
	}

	if handler != nil {
		handler(w, r)
		return
	}

	// Default: read request and respond inline
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	resp := Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  json.RawMessage(`{"ok": true}`),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *sseServer) setSessionID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionID = id
}

func (s *sseServer) close() {
	s.httpServer.Close()
}

func (s *sseServer) url() string {
	return s.httpServer.URL + "/sse"
}

func (s *sseServer) getConnectionCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sseConnections
}

func TestSSETransport_StartAndSend(t *testing.T) {
	srv := newSSEServer()
	defer srv.close()

	transport := NewSSETransport(srv.url())
	defer transport.Close()

	err := transport.Start(context.Background())
	require.NoError(t, err)

	// Verify endpoint was discovered
	msgURL := transport.getMessageURL()
	assert.Contains(t, msgURL, "/message")
	assert.Contains(t, msgURL, "sessionId=session-1")

	// Send a request
	resp, err := transport.Send(context.Background(), &Request{
		JSONRPC: "2.0",
		ID:      int64(1),
		Method:  "tools/list",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "2.0", resp.JSONRPC)
}

func TestSSETransport_EndpointReadyTimeout(t *testing.T) {
	// Server that never sends endpoint event
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher := w.(http.Flusher)
		flusher.Flush()

		// Just keep alive, never send endpoint event
		<-r.Context().Done()
	}))
	defer srv.Close()

	transport := NewSSETransport(srv.URL)
	defer transport.Close()

	err := transport.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout waiting for SSE endpoint event")
}

func TestSSETransport_SendErrorIncludesBody(t *testing.T) {
	srv := newSSEServer()
	defer srv.close()

	srv.mu.Lock()
	srv.msgHandler = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error": "Invalid session ID"}`)
	}
	srv.mu.Unlock()

	transport := NewSSETransport(srv.url())
	defer transport.Close()

	err := transport.Start(context.Background())
	require.NoError(t, err)

	// The Send will get a 400 with "session" in body -> reconnect attempt
	// After reconnect, the handler still returns 400, so we'll get the error
	_, err = transport.Send(context.Background(), &Request{
		JSONRPC: "2.0",
		ID:      int64(1),
		Method:  "test",
	})
	require.Error(t, err)
	// The error message should contain the response body
	assert.Contains(t, err.Error(), "Invalid session ID")
}

func TestSSETransport_ReconnectOnSessionError(t *testing.T) {
	srv := newSSEServer()
	defer srv.close()

	transport := NewSSETransport(srv.url())
	defer transport.Close()

	err := transport.Start(context.Background())
	require.NoError(t, err)

	// First call fails with session error, then server updates session
	callCount := 0
	srv.mu.Lock()
	srv.msgHandler = func(w http.ResponseWriter, r *http.Request) {
		srv.mu.Lock()
		callCount++
		count := callCount
		srv.mu.Unlock()

		if count == 1 {
			// First call: return session error
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "Invalid session ID")
			return
		}

		// After reconnect: return success
		var req Request
		json.NewDecoder(r.Body).Decode(&req)
		resp := Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  json.RawMessage(`{"reconnected": true}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
	srv.mu.Unlock()

	resp, err := transport.Send(context.Background(), &Request{
		JSONRPC: "2.0",
		ID:      int64(1),
		Method:  "test",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Contains(t, string(resp.Result), "reconnected")
}

func TestSSETransport_SendOnClosedTransport(t *testing.T) {
	srv := newSSEServer()
	defer srv.close()

	transport := NewSSETransport(srv.url())

	err := transport.Start(context.Background())
	require.NoError(t, err)

	transport.Close()

	_, err = transport.Send(context.Background(), &Request{
		JSONRPC: "2.0",
		ID:      int64(1),
		Method:  "test",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transport closed")
}

func TestSSETransport_NotifyIgnoresErrors(t *testing.T) {
	srv := newSSEServer()
	defer srv.close()

	transport := NewSSETransport(srv.url())
	defer transport.Close()

	err := transport.Start(context.Background())
	require.NoError(t, err)

	// Notify should not panic or return error even if server rejects
	transport.Notify(context.Background(), &Request{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	})
}

func TestIsSessionError(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want bool
	}{
		{"empty body", nil, false},
		{"session in body", []byte(`{"error": "Invalid session ID"}`), true},
		{"Session uppercase", []byte(`Invalid Session`), true},
		{"no session", []byte(`{"error": "internal error"}`), false},
		{"connection refused", []byte(`connection refused`), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSessionError(tt.body)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeID(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  interface{}
	}{
		{"float64 to int64", float64(42), int64(42)},
		{"string stays string", "abc", "abc"},
		{"int64 stays int64", int64(7), int64(7)},
		{"nil stays nil", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeID(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSSETransport_ReconnectOnStreamDrop(t *testing.T) {
	// This test verifies that when the SSE stream drops, the transport reconnects.
	// We use a server that closes the first SSE connection after a short delay.

	connCount := 0
	var connMu sync.Mutex
	connCh := make(chan struct{}, 10)

	// Track which connections are active
	type connState struct {
		cancel context.CancelFunc
	}
	var firstConn *connState

	mux := http.NewServeMux()
	mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		flusher := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		connMu.Lock()
		connCount++
		n := connCount
		connMu.Unlock()

		// Send endpoint event
		fmt.Fprintf(w, "event: endpoint\ndata: /message?session=%d\n\n", n)
		flusher.Flush()

		connCh <- struct{}{}

		if n == 1 {
			// First connection: store cancel so we can close it
			ctx, cancel := context.WithCancel(r.Context())
			connMu.Lock()
			firstConn = &connState{cancel: cancel}
			connMu.Unlock()
			<-ctx.Done()
			return
		}

		// Second connection: stay open
		<-r.Context().Done()
	})
	mux.HandleFunc("/message", func(w http.ResponseWriter, r *http.Request) {
		var req Request
		json.NewDecoder(r.Body).Decode(&req)
		resp := Response{JSONRPC: "2.0", ID: req.ID, Result: json.RawMessage(`{"ok":true}`)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	transport := NewSSETransport(srv.URL + "/sse")
	defer transport.Close()

	err := transport.Start(context.Background())
	require.NoError(t, err)

	// Wait for first connection
	<-connCh

	// Verify first endpoint is discovered
	url1 := transport.getMessageURL()
	assert.Contains(t, url1, "session=1")

	// Drop the first SSE connection by closing the response
	connMu.Lock()
	if firstConn != nil {
		firstConn.cancel()
	}
	connMu.Unlock()

	// Wait for reconnect (second connection)
	select {
	case <-connCh:
		// Second connection established
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for reconnect")
	}

	// Give a moment for endpoint discovery
	time.Sleep(100 * time.Millisecond)

	// Verify new endpoint
	url2 := transport.getMessageURL()
	assert.Contains(t, url2, "session=2")
}

func TestSSETransport_ConcurrentSend(t *testing.T) {
	srv := newSSEServer()
	defer srv.close()

	transport := NewSSETransport(srv.url())
	defer transport.Close()

	err := transport.Start(context.Background())
	require.NoError(t, err)

	// Send 10 concurrent requests
	var wg sync.WaitGroup
	errs := make([]error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = transport.Send(context.Background(), &Request{
				JSONRPC: "2.0",
				ID:      int64(idx + 1),
				Method:  "test",
			})
		}(i)
	}

	wg.Wait()

	for i, err := range errs {
		assert.NoError(t, err, "request %d failed", i)
	}
}

func TestSSETransport_SendContextCancellation(t *testing.T) {
	// Server that never responds
	srv := newSSEServer()
	defer srv.close()

	srv.mu.Lock()
	srv.msgHandler = func(w http.ResponseWriter, r *http.Request) {
		// Accepted but no response sent via HTTP body or SSE
		w.WriteHeader(http.StatusAccepted)
	}
	srv.mu.Unlock()

	transport := NewSSETransport(srv.url())
	defer transport.Close()

	err := transport.Start(context.Background())
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = transport.Send(ctx, &Request{
		JSONRPC: "2.0",
		ID:      int64(1),
		Method:  "test",
	})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "context deadline exceeded") ||
		strings.Contains(err.Error(), "context canceled"),
		"expected context error, got: %v", err)
}

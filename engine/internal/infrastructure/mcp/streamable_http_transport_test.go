package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// streamableServer is a test helper for Streamable HTTP MCP endpoints.
type streamableServer struct {
	mu          sync.Mutex
	respondMode string      // "json" or "sse"
	sessionID   string      // Mcp-Session-Id to send back
	lastHeaders http.Header // captured from last request
	handler     func(w http.ResponseWriter, r *http.Request)
	httpServer  *httptest.Server
}

func newStreamableServer(mode string) *streamableServer {
	s := &streamableServer{
		respondMode: mode,
		sessionID:   "test-session-42",
	}

	s.httpServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.lastHeaders = r.Header.Clone()
		h := s.handler
		mode := s.respondMode
		sid := s.sessionID
		s.mu.Unlock()

		if h != nil {
			h(w, r)
			return
		}

		body, _ := io.ReadAll(r.Body)
		var req Request
		_ = json.Unmarshal(body, &req)

		resp := Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  json.RawMessage(`{"content":[{"type":"text","text":"hello"}]}`),
		}

		if sid != "" {
			w.Header().Set("Mcp-Session-Id", sid)
		}

		switch mode {
		case "sse":
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			data, _ := json.Marshal(resp)
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
		}
	}))

	return s
}

func (s *streamableServer) close() {
	s.httpServer.Close()
}

func (s *streamableServer) getLastHeaders() http.Header {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastHeaders.Clone()
}

// --- Tests ---

func TestStreamableHTTP_JSONResponse(t *testing.T) {
	srv := newStreamableServer("json")
	defer srv.close()

	transport := NewStreamableHTTPTransport(srv.httpServer.URL)
	require.NoError(t, transport.Start(context.Background()))

	resp, err := transport.Send(context.Background(), &Request{
		JSONRPC: "2.0",
		ID:      int64(1),
		Method:  "tools/call",
		Params:  map[string]interface{}{"name": "test"},
	})

	require.NoError(t, err)
	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.NotNil(t, resp.Result)
	assert.Contains(t, string(resp.Result), "hello")
}

func TestStreamableHTTP_SSEResponse(t *testing.T) {
	srv := newStreamableServer("sse")
	defer srv.close()

	transport := NewStreamableHTTPTransport(srv.httpServer.URL)

	resp, err := transport.Send(context.Background(), &Request{
		JSONRPC: "2.0",
		ID:      int64(1),
		Method:  "tools/call",
	})

	require.NoError(t, err)
	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Contains(t, string(resp.Result), "hello")
}

func TestStreamableHTTP_SessionIDManagement(t *testing.T) {
	srv := newStreamableServer("json")
	defer srv.close()

	transport := NewStreamableHTTPTransport(srv.httpServer.URL)

	// First request — server sends session ID.
	_, err := transport.Send(context.Background(), &Request{
		JSONRPC: "2.0",
		ID:      int64(1),
		Method:  "initialize",
	})
	require.NoError(t, err)

	// Verify transport stored the session ID.
	transport.mu.RLock()
	assert.Equal(t, "test-session-42", transport.sessionID)
	transport.mu.RUnlock()

	// Second request — transport should echo session ID.
	_, err = transport.Send(context.Background(), &Request{
		JSONRPC: "2.0",
		ID:      int64(2),
		Method:  "tools/list",
	})
	require.NoError(t, err)

	headers := srv.getLastHeaders()
	assert.Equal(t, "test-session-42", headers.Get("Mcp-Session-Id"))
}

func TestStreamableHTTP_Notify(t *testing.T) {
	var called atomic.Int32
	srv := newStreamableServer("json")
	srv.mu.Lock()
	srv.handler = func(w http.ResponseWriter, r *http.Request) {
		called.Add(1)
		w.WriteHeader(http.StatusOK)
	}
	srv.mu.Unlock()
	defer srv.close()

	transport := NewStreamableHTTPTransport(srv.httpServer.URL)
	transport.Notify(context.Background(), &Request{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	})

	// Give the HTTP call a moment to complete.
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(1), called.Load())
}

func TestStreamableHTTP_ContextCancellation(t *testing.T) {
	srv := newStreamableServer("json")
	srv.mu.Lock()
	srv.handler = func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow server.
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}
	srv.mu.Unlock()
	defer srv.close()

	transport := NewStreamableHTTPTransport(srv.httpServer.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := transport.Send(ctx, &Request{
		JSONRPC: "2.0",
		ID:      int64(1),
		Method:  "tools/call",
	})

	assert.Error(t, err)
}

func TestStreamableHTTP_ServerError(t *testing.T) {
	srv := newStreamableServer("json")
	srv.mu.Lock()
	srv.handler = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}
	srv.mu.Unlock()
	defer srv.close()

	transport := NewStreamableHTTPTransport(srv.httpServer.URL)

	_, err := transport.Send(context.Background(), &Request{
		JSONRPC: "2.0",
		ID:      int64(1),
		Method:  "tools/call",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestStreamableHTTP_SSEMultipleEvents(t *testing.T) {
	srv := newStreamableServer("json")
	srv.mu.Lock()
	srv.handler = func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req Request
		_ = json.Unmarshal(body, &req)

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// First: a notification (no matching ID).
		notif := Response{
			JSONRPC: "2.0",
			Result:  json.RawMessage(`{"type":"progress","progress":50}`),
		}
		data, _ := json.Marshal(notif)
		fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)

		// Second: the actual response with matching ID.
		resp := Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  json.RawMessage(`{"content":[{"type":"text","text":"found it"}]}`),
		}
		data, _ = json.Marshal(resp)
		fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
	}
	srv.mu.Unlock()
	defer srv.close()

	transport := NewStreamableHTTPTransport(srv.httpServer.URL)

	resp, err := transport.Send(context.Background(), &Request{
		JSONRPC: "2.0",
		ID:      int64(7),
		Method:  "tools/call",
	})

	require.NoError(t, err)
	assert.Contains(t, string(resp.Result), "found it")
}

func TestStreamableHTTP_ConcurrentSend(t *testing.T) {
	srv := newStreamableServer("json")
	defer srv.close()

	transport := NewStreamableHTTPTransport(srv.httpServer.URL)

	var wg sync.WaitGroup
	errs := make([]error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := transport.Send(context.Background(), &Request{
				JSONRPC: "2.0",
				ID:      int64(idx + 1),
				Method:  "tools/call",
			})
			errs[idx] = err
		}(i)
	}

	wg.Wait()

	for i, err := range errs {
		assert.NoError(t, err, "request %d failed", i)
	}

	// Session ID should be set from any of the responses.
	transport.mu.RLock()
	assert.Equal(t, "test-session-42", transport.sessionID)
	transport.mu.RUnlock()
}

func TestStreamableHTTP_AcceptHeader(t *testing.T) {
	srv := newStreamableServer("json")
	defer srv.close()

	transport := NewStreamableHTTPTransport(srv.httpServer.URL)
	_, _ = transport.Send(context.Background(), &Request{
		JSONRPC: "2.0",
		ID:      int64(1),
		Method:  "initialize",
	})

	headers := srv.getLastHeaders()
	assert.Equal(t, "application/json, text/event-stream", headers.Get("Accept"))
	assert.Equal(t, "application/json", headers.Get("Content-Type"))
}

package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSSEWriter_SetsHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	sw, err := NewSSEWriter(rec)
	require.NoError(t, err)
	require.NotNil(t, sw)

	assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", rec.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", rec.Header().Get("Connection"))
	assert.Equal(t, "no", rec.Header().Get("X-Accel-Buffering"))
}

func TestNewSSEWriter_NoFlusher(t *testing.T) {
	// nonFlushWriter does not implement http.Flusher.
	sw, err := NewSSEWriter(&nonFlushWriter{})
	assert.Error(t, err)
	assert.Nil(t, sw)
	assert.Contains(t, err.Error(), "streaming not supported")
}

func TestSSEWriter_WriteEvent(t *testing.T) {
	rec := httptest.NewRecorder()
	sw, err := NewSSEWriter(rec)
	require.NoError(t, err)

	err = sw.WriteEvent("message", `{"text":"hello"}`)
	require.NoError(t, err)

	body := rec.Body.String()
	assert.Contains(t, body, "event: message\n")
	assert.Contains(t, body, `data: {"text":"hello"}`)
	assert.True(t, strings.HasSuffix(body, "\n\n"))
}

func TestSSEWriter_WriteComment(t *testing.T) {
	rec := httptest.NewRecorder()
	sw, err := NewSSEWriter(rec)
	require.NoError(t, err)

	sw.WriteComment("heartbeat")

	body := rec.Body.String()
	assert.Contains(t, body, ": heartbeat\n")
}

func TestSSEWriter_StartHeartbeat(t *testing.T) {
	rec := httptest.NewRecorder()
	sw, err := NewSSEWriter(rec)
	require.NoError(t, err)

	stop := sw.StartHeartbeat(10 * time.Millisecond)
	time.Sleep(50 * time.Millisecond)
	stop()

	body := rec.Body.String()
	assert.Contains(t, body, ": heartbeat\n")
}

// nonFlushWriter implements http.ResponseWriter but NOT http.Flusher.
type nonFlushWriter struct{}

func (n *nonFlushWriter) Header() http.Header        { return http.Header{} }
func (n *nonFlushWriter) Write(b []byte) (int, error) { return len(b), nil }
func (n *nonFlushWriter) WriteHeader(int)             {}

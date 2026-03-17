package http

import (
	"fmt"
	"net/http"
	"time"
)

// SSEWriter writes Server-Sent Events to an http.ResponseWriter.
type SSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

// NewSSEWriter creates an SSEWriter after setting SSE headers.
// Returns an error if the ResponseWriter does not support flushing.
func NewSSEWriter(w http.ResponseWriter) (*SSEWriter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher.Flush()

	return &SSEWriter{w: w, flusher: flusher}, nil
}

// WriteEvent writes a single SSE event with the given type and data.
func (s *SSEWriter) WriteEvent(eventType string, data string) error {
	_, err := fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", eventType, data)
	if err != nil {
		return fmt.Errorf("write SSE event: %w", err)
	}
	s.flusher.Flush()
	return nil
}

// WriteComment writes an SSE comment line (prefixed with ':').
func (s *SSEWriter) WriteComment(comment string) {
	fmt.Fprintf(s.w, ": %s\n\n", comment)
	s.flusher.Flush()
}

// StartHeartbeat sends comment heartbeats at the given interval.
// Returns a stop function that must be called to clean up the goroutine.
func (s *SSEWriter) StartHeartbeat(interval time.Duration) func() {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				s.WriteComment("heartbeat")
			}
		}
	}()
	return func() { close(done) }
}

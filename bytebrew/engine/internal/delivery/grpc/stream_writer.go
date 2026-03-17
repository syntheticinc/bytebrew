package grpc

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	pb "github.com/syntheticinc/bytebrew/bytebrew/engine/api/proto/gen"
)

const sendTimeout = 10 * time.Second

// StreamWriter provides thread-safe access to gRPC stream.Send().
// Multiple goroutines (Supervisor, Code Agents, proxy) can safely
// write to a single gRPC stream through this writer.
type StreamWriter struct {
	stream  pb.FlowService_ExecuteFlowServer
	writeCh chan *pb.FlowResponse
	closeCh chan struct{} // Signal to stop accepting new messages
	done    chan struct{} // Closed when writerLoop exits
	once    sync.Once
	lastErr error     // Last send error (set in writerLoop)
	errMu   sync.Mutex // Protects lastErr
}

// NewStreamWriter creates a thread-safe stream writer.
// Starts a background goroutine that serializes Send() calls.
func NewStreamWriter(stream pb.FlowService_ExecuteFlowServer) *StreamWriter {
	w := &StreamWriter{
		stream:  stream,
		writeCh: make(chan *pb.FlowResponse, 256),
		closeCh: make(chan struct{}),
		done:    make(chan struct{}),
	}

	go w.writerLoop()
	return w
}

// Send queues a response for sending. Thread-safe.
// Returns error if the writer is closed or if the send times out.
func (w *StreamWriter) Send(resp *pb.FlowResponse) error {
	// Fast path: check if already closed
	select {
	case <-w.closeCh:
		return fmt.Errorf("stream writer closed")
	default:
	}

	select {
	case w.writeCh <- resp:
		return nil
	case <-w.closeCh:
		return fmt.Errorf("stream writer closed")
	case <-time.After(sendTimeout):
		return fmt.Errorf("stream writer send timeout (%s)", sendTimeout)
	}
}

// LastError returns the last error from the writer loop (e.g. stream.Send failure).
func (w *StreamWriter) LastError() error {
	w.errMu.Lock()
	defer w.errMu.Unlock()
	return w.lastErr
}

func (w *StreamWriter) setLastError(err error) {
	w.errMu.Lock()
	w.lastErr = err
	w.errMu.Unlock()
}

// Close stops the writer goroutine and drains remaining messages.
func (w *StreamWriter) Close() {
	w.once.Do(func() {
		close(w.closeCh)
		<-w.done // Wait for writer goroutine to finish draining
	})
}

// writerLoop reads from the channel and calls stream.Send() sequentially.
func (w *StreamWriter) writerLoop() {
	defer close(w.done)

	for {
		select {
		case resp := <-w.writeCh:
			if err := w.stream.Send(resp); err != nil {
				slog.Error("[StreamWriter] failed to send", "error", err)
				w.setLastError(err)
				w.drainWriteChannel()
				return
			}
		case <-w.closeCh:
			w.drainAndSendRemaining()
			return
		}
	}
}

// drainWriteChannel discards remaining messages to unblock senders.
func (w *StreamWriter) drainWriteChannel() {
	for {
		select {
		case <-w.writeCh:
		default:
			return
		}
	}
}

// drainAndSendRemaining sends remaining messages before shutdown.
func (w *StreamWriter) drainAndSendRemaining() {
	for {
		select {
		case resp := <-w.writeCh:
			if err := w.stream.Send(resp); err != nil {
				w.setLastError(err)
				return
			}
		default:
			return
		}
	}
}

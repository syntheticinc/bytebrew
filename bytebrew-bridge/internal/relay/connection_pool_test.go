package relay

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	bridgev1 "github.com/syntheticinc/bytebrew/bytebrew-bridge/api/proto/gen"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// mockStream implements grpc.BidiStreamingServer[BridgeFrame, BridgeFrame] for testing.
type mockStream struct {
	ctx     context.Context
	recvCh  chan *bridgev1.BridgeFrame
	sentMu  sync.Mutex
	sent    []*bridgev1.BridgeFrame
	sendErr error
}

func newMockStream(ctx context.Context) *mockStream {
	return &mockStream{
		ctx:    ctx,
		recvCh: make(chan *bridgev1.BridgeFrame, 64),
	}
}

func (m *mockStream) Send(frame *bridgev1.BridgeFrame) error {
	m.sentMu.Lock()
	defer m.sentMu.Unlock()

	if m.sendErr != nil {
		return m.sendErr
	}
	m.sent = append(m.sent, frame)
	return nil
}

func (m *mockStream) Recv() (*bridgev1.BridgeFrame, error) {
	select {
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	case frame, ok := <-m.recvCh:
		if !ok {
			return nil, io.EOF
		}
		return frame, nil
	}
}

func (m *mockStream) getSent() []*bridgev1.BridgeFrame {
	m.sentMu.Lock()
	defer m.sentMu.Unlock()
	result := make([]*bridgev1.BridgeFrame, len(m.sent))
	copy(result, m.sent)
	return result
}

func (m *mockStream) SetHeader(metadata.MD) error  { return nil }
func (m *mockStream) SendHeader(metadata.MD) error  { return nil }
func (m *mockStream) SetTrailer(metadata.MD)         {}
func (m *mockStream) Context() context.Context       { return m.ctx }
func (m *mockStream) SendMsg(interface{}) error      { return nil }
func (m *mockStream) RecvMsg(interface{}) error      { return nil }

// enqueueAndClose sends frames to the stream then closes it (simulating EOF).
func (m *mockStream) enqueueAndClose(frames ...*bridgev1.BridgeFrame) {
	for _, f := range frames {
		m.recvCh <- f
	}
	close(m.recvCh)
}

func TestRegisterAndListOnline(t *testing.T) {
	pool := NewConnectionPool("")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream := newMockStream(ctx)

	// Send REGISTER frame but do NOT close yet — server must stay alive for ListOnline.
	stream.recvCh <- &bridgev1.BridgeFrame{
		ServerId: "srv-1",
		Payload:  []byte("My Server"),
		Type:     bridgev1.FrameType_FRAME_TYPE_REGISTER,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- pool.RegisterServer(stream)
	}()

	// Give RegisterServer time to process the handshake and register.
	time.Sleep(50 * time.Millisecond)

	servers := pool.ListOnline()
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	if servers[0].ServerId != "srv-1" {
		t.Errorf("expected server_id=srv-1, got %s", servers[0].ServerId)
	}
	if servers[0].ServerName != "My Server" {
		t.Errorf("expected server_name=My Server, got %s", servers[0].ServerName)
	}
	if servers[0].ConnectedSince <= 0 {
		t.Errorf("expected positive connected_since, got %d", servers[0].ConnectedSince)
	}

	// Now close the stream to trigger EOF and server removal.
	close(stream.recvCh)

	err := <-errCh
	if err != nil {
		t.Fatalf("RegisterServer returned unexpected error: %v", err)
	}

	// After EOF, server should be removed.
	servers = pool.ListOnline()
	if len(servers) != 0 {
		t.Errorf("expected 0 servers after disconnect, got %d", len(servers))
	}
}

func TestConnectToUnregisteredServer(t *testing.T) {
	pool := NewConnectionPool("")
	ctx := context.Background()

	stream := newMockStream(ctx)
	stream.enqueueAndClose(&bridgev1.BridgeFrame{
		ServerId: "nonexistent",
		DeviceId: "mobile-1",
		Type:     bridgev1.FrameType_FRAME_TYPE_CONNECT,
	})

	err := pool.ConnectMobile(stream)
	if err == nil {
		t.Fatal("expected error for unregistered server, got nil")
	}

	expectedMsg := "server nonexistent is not online"
	if err.Error() != expectedMsg {
		t.Errorf("expected error %q, got %q", expectedMsg, err.Error())
	}
}

func TestRemoveServer(t *testing.T) {
	pool := NewConnectionPool("")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream := newMockStream(ctx)

	// Manually add a server to the pool (bypassing handshake for this unit test).
	pool.mu.Lock()
	pool.servers["srv-remove"] = &serverConn{
		serverID:    "srv-remove",
		serverName:  "Test Server",
		stream:      stream,
		connectedAt: time.Now(),
		mobileChans: make(map[string]chan *bridgev1.BridgeFrame),
	}
	pool.mu.Unlock()

	servers := pool.ListOnline()
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}

	pool.RemoveServer("srv-remove")

	servers = pool.ListOnline()
	if len(servers) != 0 {
		t.Errorf("expected 0 servers after removal, got %d", len(servers))
	}

	// Removing again should not panic.
	pool.RemoveServer("srv-remove")
}

func TestConcurrentRegistrations(t *testing.T) {
	pool := NewConnectionPool("")
	const numServers = 20

	var wg sync.WaitGroup
	wg.Add(numServers)

	for i := 0; i < numServers; i++ {
		go func(idx int) {
			defer wg.Done()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			stream := newMockStream(ctx)
			serverID := "srv-" + time.Now().Format("150405") + "-" + string(rune('A'+idx))

			stream.enqueueAndClose(&bridgev1.BridgeFrame{
				ServerId: serverID,
				Payload:  []byte("Server " + string(rune('A'+idx))),
				Type:     bridgev1.FrameType_FRAME_TYPE_REGISTER,
			})

			_ = pool.RegisterServer(stream)
		}(i)
	}

	wg.Wait()

	// After all goroutines finish (all streams closed -> all servers removed),
	// the pool should be empty.
	servers := pool.ListOnline()
	if len(servers) != 0 {
		t.Errorf("expected 0 servers after all disconnected, got %d", len(servers))
	}
}

func TestRegisterServerInvalidFirstFrame(t *testing.T) {
	pool := NewConnectionPool("")
	ctx := context.Background()

	stream := newMockStream(ctx)
	stream.enqueueAndClose(&bridgev1.BridgeFrame{
		ServerId: "srv-1",
		Type:     bridgev1.FrameType_FRAME_TYPE_DATA,
	})

	err := pool.RegisterServer(stream)
	if err == nil {
		t.Fatal("expected error for non-REGISTER frame, got nil")
	}
}

func TestRegisterServerEmptyID(t *testing.T) {
	pool := NewConnectionPool("")
	ctx := context.Background()

	stream := newMockStream(ctx)
	stream.enqueueAndClose(&bridgev1.BridgeFrame{
		ServerId: "",
		Type:     bridgev1.FrameType_FRAME_TYPE_REGISTER,
	})

	err := pool.RegisterServer(stream)
	if err == nil {
		t.Fatal("expected error for empty server_id, got nil")
	}
}

func TestConnectMobileInvalidFirstFrame(t *testing.T) {
	pool := NewConnectionPool("")
	ctx := context.Background()

	stream := newMockStream(ctx)
	stream.enqueueAndClose(&bridgev1.BridgeFrame{
		ServerId: "srv-1",
		DeviceId: "mobile-1",
		Type:     bridgev1.FrameType_FRAME_TYPE_DATA,
	})

	err := pool.ConnectMobile(stream)
	if err == nil {
		t.Fatal("expected error for non-CONNECT frame, got nil")
	}
}

func TestConnectMobileEmptyDeviceID(t *testing.T) {
	pool := NewConnectionPool("")
	ctx := context.Background()

	stream := newMockStream(ctx)
	stream.enqueueAndClose(&bridgev1.BridgeFrame{
		ServerId: "srv-1",
		DeviceId: "",
		Type:     bridgev1.FrameType_FRAME_TYPE_CONNECT,
	})

	err := pool.ConnectMobile(stream)
	if err == nil {
		t.Fatal("expected error for empty device_id, got nil")
	}
}

func TestBidirectionalRelay(t *testing.T) {
	pool := NewConnectionPool("")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Setup: register a server that stays alive.
	serverStream := newMockStream(ctx)
	serverReady := make(chan struct{})
	serverDone := make(chan error, 1)

	go func() {
		// Server stream: send REGISTER, then keep alive to relay frames.
		// We cannot use enqueueAndClose because we need the server to stay alive.
		serverStream.recvCh <- &bridgev1.BridgeFrame{
			ServerId: "srv-relay",
			Payload:  []byte("Relay Server"),
			Type:     bridgev1.FrameType_FRAME_TYPE_REGISTER,
		}

		// Signal that the server is registered (wait a bit for processing).
		time.Sleep(50 * time.Millisecond)
		close(serverReady)

		// Block until context is cancelled (simulating long-lived connection).
		<-ctx.Done()
		close(serverStream.recvCh)

		// Drain the error from RegisterServer.
		serverDone <- nil
	}()

	// Start RegisterServer in background.
	go func() {
		err := pool.RegisterServer(serverStream)
		if err != nil {
			serverDone <- err
		}
	}()

	// Wait for server to be registered.
	<-serverReady

	// Setup: mobile connects.
	mobileStream := newMockStream(ctx)
	mobileDone := make(chan error, 1)

	go func() {
		// Send CONNECT frame, then a data frame, then close.
		mobileStream.recvCh <- &bridgev1.BridgeFrame{
			ServerId: "srv-relay",
			DeviceId: "mobile-relay",
			Type:     bridgev1.FrameType_FRAME_TYPE_CONNECT,
		}

		// Wait a moment for connection setup.
		time.Sleep(50 * time.Millisecond)

		// Send a data frame.
		mobileStream.recvCh <- &bridgev1.BridgeFrame{
			ServerId: "srv-relay",
			DeviceId: "mobile-relay",
			Type:     bridgev1.FrameType_FRAME_TYPE_DATA,
			Payload:  []byte("hello from mobile"),
		}

		// Wait for relay, then close.
		time.Sleep(50 * time.Millisecond)
		close(mobileStream.recvCh)
	}()

	go func() {
		mobileDone <- pool.ConnectMobile(mobileStream)
	}()

	// Wait for mobile relay to finish.
	select {
	case err := <-mobileDone:
		if err != nil {
			t.Errorf("ConnectMobile returned unexpected error: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("test timed out waiting for mobile relay")
	}

	// Verify server received the frames.
	serverSent := serverStream.getSent()
	if len(serverSent) < 2 {
		t.Fatalf("expected at least 2 frames sent to server (CONNECT + DATA), got %d", len(serverSent))
	}

	// First should be CONNECT.
	if serverSent[0].GetType() != bridgev1.FrameType_FRAME_TYPE_CONNECT {
		t.Errorf("expected first frame to be CONNECT, got %s", serverSent[0].GetType())
	}

	// Second should be DATA.
	if serverSent[1].GetType() != bridgev1.FrameType_FRAME_TYPE_DATA {
		t.Errorf("expected second frame to be DATA, got %s", serverSent[1].GetType())
	}
	if string(serverSent[1].GetPayload()) != "hello from mobile" {
		t.Errorf("expected payload 'hello from mobile', got %q", string(serverSent[1].GetPayload()))
	}

	// Third should be DISCONNECT (sent when mobile stream closes).
	if len(serverSent) >= 3 {
		if serverSent[2].GetType() != bridgev1.FrameType_FRAME_TYPE_DISCONNECT {
			t.Errorf("expected third frame to be DISCONNECT, got %s", serverSent[2].GetType())
		}
	}

	cancel()
}

func TestRemoveServerClosesMobileChannels(t *testing.T) {
	pool := NewConnectionPool("")
	ctx := context.Background()

	stream := newMockStream(ctx)

	// Add server with a mobile channel.
	mobileCh := make(chan *bridgev1.BridgeFrame, 8)
	pool.mu.Lock()
	pool.servers["srv-close"] = &serverConn{
		serverID:    "srv-close",
		serverName:  "Test",
		stream:      stream,
		connectedAt: time.Now(),
		mobileChans: map[string]chan *bridgev1.BridgeFrame{
			"mobile-1": mobileCh,
		},
	}
	pool.mu.Unlock()

	pool.RemoveServer("srv-close")

	// Channel should be closed.
	_, ok := <-mobileCh
	if ok {
		t.Error("expected mobile channel to be closed after RemoveServer")
	}
}

func TestRegisterServerInvalidAuthToken(t *testing.T) {
	pool := NewConnectionPool("secret123")
	ctx := context.Background()

	stream := newMockStream(ctx)
	stream.enqueueAndClose(&bridgev1.BridgeFrame{
		ServerId:  "srv-1",
		Payload:   []byte("My Server"),
		Type:      bridgev1.FrameType_FRAME_TYPE_REGISTER,
		AuthToken: "wrong-token",
	})

	err := pool.RegisterServer(stream)
	if err == nil {
		t.Fatal("expected error for invalid auth token, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.Unauthenticated {
		t.Errorf("expected Unauthenticated, got %s", st.Code())
	}
	if !strings.Contains(st.Message(), "invalid auth token") {
		t.Errorf("expected 'invalid auth token' in message, got %q", st.Message())
	}

	// Server should NOT be registered.
	servers := pool.ListOnline()
	if len(servers) != 0 {
		t.Errorf("expected 0 servers, got %d", len(servers))
	}
}

func TestRegisterServerValidAuthToken(t *testing.T) {
	pool := NewConnectionPool("secret123")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream := newMockStream(ctx)

	// Send REGISTER frame with valid token, keep stream open.
	stream.recvCh <- &bridgev1.BridgeFrame{
		ServerId:  "srv-auth",
		Payload:   []byte("Authenticated Server"),
		Type:      bridgev1.FrameType_FRAME_TYPE_REGISTER,
		AuthToken: "secret123",
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- pool.RegisterServer(stream)
	}()

	// Give RegisterServer time to process the handshake.
	time.Sleep(50 * time.Millisecond)

	servers := pool.ListOnline()
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	if servers[0].ServerId != "srv-auth" {
		t.Errorf("expected server_id=srv-auth, got %s", servers[0].ServerId)
	}

	// Close stream to clean up.
	close(stream.recvCh)

	err := <-errCh
	if err != nil {
		t.Fatalf("RegisterServer returned unexpected error: %v", err)
	}
}

func TestConnectMobileInvalidAuthToken(t *testing.T) {
	pool := NewConnectionPool("secret123")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Register a server with valid token first.
	serverStream := newMockStream(ctx)
	serverStream.recvCh <- &bridgev1.BridgeFrame{
		ServerId:  "srv-mobile-auth",
		Payload:   []byte("Server"),
		Type:      bridgev1.FrameType_FRAME_TYPE_REGISTER,
		AuthToken: "secret123",
	}

	go func() {
		_ = pool.RegisterServer(serverStream)
	}()

	// Wait for server registration.
	time.Sleep(50 * time.Millisecond)

	// Now try connecting mobile with invalid token.
	mobileStream := newMockStream(ctx)
	mobileStream.enqueueAndClose(&bridgev1.BridgeFrame{
		ServerId:  "srv-mobile-auth",
		DeviceId:  "mobile-1",
		Type:      bridgev1.FrameType_FRAME_TYPE_CONNECT,
		AuthToken: "wrong-token",
	})

	err := pool.ConnectMobile(mobileStream)
	if err == nil {
		t.Fatal("expected error for invalid auth token, got nil")
	}

	st2, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st2.Code() != codes.Unauthenticated {
		t.Errorf("expected Unauthenticated, got %s", st2.Code())
	}
	if !strings.Contains(st2.Message(), "invalid auth token") {
		t.Errorf("expected 'invalid auth token' in message, got %q", st2.Message())
	}

	// Clean up server stream.
	close(serverStream.recvCh)
}

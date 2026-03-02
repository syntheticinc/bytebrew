package relay

import (
	"context"
	"crypto/subtle"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	bridgev1 "github.com/syntheticinc/bytebrew/bytebrew-bridge/api/proto/gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// serverConn represents a registered bytebrew-srv connection.
type serverConn struct {
	serverID    string
	serverName  string
	stream      grpc.BidiStreamingServer[bridgev1.BridgeFrame, bridgev1.BridgeFrame]
	connectedAt time.Time

	mu          sync.Mutex
	// mobileChans maps deviceID to a channel for sending frames to that device's relay goroutine.
	mobileChans map[string]chan *bridgev1.BridgeFrame
}

// sendToMobile routes a frame from the server to the appropriate mobile device channel.
func (sc *serverConn) sendToMobile(frame *bridgev1.BridgeFrame) {
	sc.mu.Lock()
	ch, ok := sc.mobileChans[frame.GetDeviceId()]
	sc.mu.Unlock()

	if !ok {
		return
	}

	select {
	case ch <- frame:
	default:
		slog.Warn("mobile channel full, dropping frame",
			"server_id", sc.serverID,
			"device_id", frame.GetDeviceId(),
		)
	}
}

// addMobile registers a mobile device channel for receiving server frames.
func (sc *serverConn) addMobile(deviceID string) chan *bridgev1.BridgeFrame {
	ch := make(chan *bridgev1.BridgeFrame, 64)
	sc.mu.Lock()
	sc.mobileChans[deviceID] = ch
	sc.mu.Unlock()
	return ch
}

// removeMobile removes a mobile device channel and closes it.
func (sc *serverConn) removeMobile(deviceID string) {
	sc.mu.Lock()
	ch, ok := sc.mobileChans[deviceID]
	if ok {
		delete(sc.mobileChans, deviceID)
		close(ch)
	}
	sc.mu.Unlock()
}

// ConnectionPool manages registered server connections and mobile-to-server relay.
type ConnectionPool struct {
	mu        sync.RWMutex
	servers   map[string]*serverConn
	authToken string // empty = no auth required
}

// NewConnectionPool creates an empty connection pool.
// If authToken is non-empty, all incoming connections must present a matching token.
func NewConnectionPool(authToken string) *ConnectionPool {
	return &ConnectionPool{
		servers:   make(map[string]*serverConn),
		authToken: authToken,
	}
}

// RegisterServer handles a server registration stream. It blocks until the stream closes.
// The first frame must have type=REGISTER with server_id and server_name set.
// Subsequent frames from the server are routed to the appropriate mobile device.
func (p *ConnectionPool) RegisterServer(stream grpc.BidiStreamingServer[bridgev1.BridgeFrame, bridgev1.BridgeFrame]) error {
	sc, err := p.registerServerHandshake(stream)
	if err != nil {
		return err
	}
	defer p.removeServerConn(sc.serverID, sc)

	return p.routeServerFrames(sc)
}

// registerServerHandshake reads the REGISTER frame, validates auth, creates serverConn,
// and registers it in the pool. Returns the serverConn on success.
func (p *ConnectionPool) registerServerHandshake(stream grpc.BidiStreamingServer[bridgev1.BridgeFrame, bridgev1.BridgeFrame]) (*serverConn, error) {
	first, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("recv registration frame: %w", err)
	}

	if first.GetType() != bridgev1.FrameType_FRAME_TYPE_REGISTER {
		return nil, fmt.Errorf("expected REGISTER frame, got %s", first.GetType())
	}

	serverID := first.GetServerId()
	if serverID == "" {
		return nil, fmt.Errorf("server_id is required in REGISTER frame")
	}

	if p.authToken != "" && subtle.ConstantTimeCompare([]byte(first.GetAuthToken()), []byte(p.authToken)) != 1 {
		return nil, status.Errorf(codes.Unauthenticated, "invalid auth token for server %s", serverID)
	}

	sc := &serverConn{
		serverID:    serverID,
		serverName:  string(first.GetPayload()),
		stream:      stream,
		connectedAt: time.Now(),
		mobileChans: make(map[string]chan *bridgev1.BridgeFrame),
	}

	p.mu.Lock()
	if old, exists := p.servers[serverID]; exists {
		slog.Warn("server re-registering, replacing old connection", "server_id", serverID)
		old.mu.Lock()
		for devID, ch := range old.mobileChans {
			close(ch)
			delete(old.mobileChans, devID)
		}
		old.mu.Unlock()
	}
	p.servers[serverID] = sc
	p.mu.Unlock()

	slog.Info("server registered", "server_id", serverID, "server_name", sc.serverName)

	return sc, nil
}

// routeServerFrames reads frames from the server stream and routes them to mobile devices.
// Blocks until the stream closes or returns an error.
func (p *ConnectionPool) routeServerFrames(sc *serverConn) error {
	for {
		frame, err := sc.stream.Recv()
		if err != nil {
			if err == io.EOF {
				slog.Info("server disconnected gracefully", "server_id", sc.serverID)
				return nil
			}
			return fmt.Errorf("recv from server %s: %w", sc.serverID, err)
		}

		sc.sendToMobile(frame)
	}
}

// connectInfo holds the result of a successful mobile handshake.
type connectInfo struct {
	serverID   string
	deviceID   string
	sc         *serverConn
	fromServer chan *bridgev1.BridgeFrame
}

// ConnectMobile relays frames between a mobile device and a registered server.
// The first frame must have type=CONNECT with server_id and device_id set.
// It blocks until either side disconnects.
func (p *ConnectionPool) ConnectMobile(stream grpc.BidiStreamingServer[bridgev1.BridgeFrame, bridgev1.BridgeFrame]) error {
	info, err := p.connectMobileHandshake(stream)
	if err != nil {
		return err
	}
	defer info.sc.removeMobile(info.deviceID)

	relayCtx, relayCancel := context.WithCancel(stream.Context())
	defer relayCancel()

	errCh := make(chan error, 2)
	go p.relayMobileToServer(relayCtx, stream, info.sc, info.serverID, info.deviceID, errCh)
	go p.relayServerToMobile(relayCtx, stream, info.fromServer, info.deviceID, errCh)

	relayErr := <-errCh
	relayCancel()

	slog.Info("mobile disconnected", "server_id", info.serverID, "device_id", info.deviceID)

	return relayErr
}

// connectMobileHandshake reads the CONNECT frame, validates auth, finds the server,
// registers a mobile channel, and forwards the CONNECT frame to the server.
func (p *ConnectionPool) connectMobileHandshake(stream grpc.BidiStreamingServer[bridgev1.BridgeFrame, bridgev1.BridgeFrame]) (*connectInfo, error) {
	first, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("recv connect frame: %w", err)
	}

	if first.GetType() != bridgev1.FrameType_FRAME_TYPE_CONNECT {
		return nil, fmt.Errorf("expected CONNECT frame, got %s", first.GetType())
	}

	serverID := first.GetServerId()
	deviceID := first.GetDeviceId()

	if serverID == "" {
		return nil, fmt.Errorf("server_id is required in CONNECT frame")
	}
	if deviceID == "" {
		return nil, fmt.Errorf("device_id is required in CONNECT frame")
	}

	if p.authToken != "" && subtle.ConstantTimeCompare([]byte(first.GetAuthToken()), []byte(p.authToken)) != 1 {
		return nil, status.Errorf(codes.Unauthenticated, "invalid auth token for device %s", deviceID)
	}

	p.mu.RLock()
	sc, ok := p.servers[serverID]
	p.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("server %s is not online", serverID)
	}

	slog.Info("mobile connected", "server_id", serverID, "device_id", deviceID)

	fromServer := sc.addMobile(deviceID)

	sc.mu.Lock()
	sendErr := sc.stream.Send(first)
	sc.mu.Unlock()

	if sendErr != nil {
		sc.removeMobile(deviceID)
		return nil, fmt.Errorf("send connect frame to server %s: %w", serverID, sendErr)
	}

	return &connectInfo{
		serverID:   serverID,
		deviceID:   deviceID,
		sc:         sc,
		fromServer: fromServer,
	}, nil
}

// relayMobileToServer reads frames from the mobile stream and sends them to the server.
// Sends the result (nil or error) to errCh when done.
func (p *ConnectionPool) relayMobileToServer(
	_ context.Context,
	stream grpc.BidiStreamingServer[bridgev1.BridgeFrame, bridgev1.BridgeFrame],
	sc *serverConn,
	serverID, deviceID string,
	errCh chan<- error,
) {
	for {
		frame, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				sc.mu.Lock()
				sendErr := sc.stream.Send(&bridgev1.BridgeFrame{
					ServerId: serverID,
					DeviceId: deviceID,
					Type:     bridgev1.FrameType_FRAME_TYPE_DISCONNECT,
				})
				sc.mu.Unlock()
				if sendErr != nil {
					slog.Warn("failed to send disconnect frame", "error", sendErr, "device_id", deviceID)
				}
				errCh <- nil
				return
			}
			errCh <- fmt.Errorf("recv from mobile %s: %w", deviceID, err)
			return
		}

		frame.ServerId = serverID
		frame.DeviceId = deviceID

		sc.mu.Lock()
		err = sc.stream.Send(frame)
		sc.mu.Unlock()

		if err != nil {
			errCh <- fmt.Errorf("send to server %s: %w", serverID, err)
			return
		}
	}
}

// relayServerToMobile reads frames from the server channel and sends them to the mobile stream.
// Sends the result (nil or error) to errCh when done.
func (p *ConnectionPool) relayServerToMobile(
	relayCtx context.Context,
	stream grpc.BidiStreamingServer[bridgev1.BridgeFrame, bridgev1.BridgeFrame],
	fromServer <-chan *bridgev1.BridgeFrame,
	deviceID string,
	errCh chan<- error,
) {
	for {
		select {
		case frame, ok := <-fromServer:
			if !ok {
				errCh <- nil
				return
			}
			if err := stream.Send(frame); err != nil {
				errCh <- fmt.Errorf("send to mobile %s: %w", deviceID, err)
				return
			}
		case <-relayCtx.Done():
			errCh <- relayCtx.Err()
			return
		}
	}
}

// ListOnline returns all currently connected servers.
func (p *ConnectionPool) ListOnline() []*bridgev1.OnlineServer {
	p.mu.RLock()
	defer p.mu.RUnlock()

	servers := make([]*bridgev1.OnlineServer, 0, len(p.servers))
	for _, sc := range p.servers {
		servers = append(servers, &bridgev1.OnlineServer{
			ServerId:       sc.serverID,
			ServerName:     sc.serverName,
			ConnectedSince: sc.connectedAt.Unix(),
		})
	}

	return servers
}

// removeServerConn removes the server only if the current connection matches sc.
// This prevents a re-registered (newer) connection from being removed
// when an older connection's defer fires.
func (p *ConnectionPool) removeServerConn(serverID string, sc *serverConn) {
	p.mu.Lock()
	current, exists := p.servers[serverID]
	if !exists || current != sc {
		p.mu.Unlock()
		return
	}
	delete(p.servers, serverID)
	p.mu.Unlock()

	sc.mu.Lock()
	for devID, ch := range sc.mobileChans {
		close(ch)
		delete(sc.mobileChans, devID)
	}
	sc.mu.Unlock()

	slog.Info("server removed", "server_id", serverID)
}

// RemoveServer removes a server connection and closes all associated mobile channels.
func (p *ConnectionPool) RemoveServer(serverID string) {
	p.mu.Lock()
	sc, ok := p.servers[serverID]
	if !ok {
		p.mu.Unlock()
		return
	}
	delete(p.servers, serverID)
	p.mu.Unlock()

	sc.mu.Lock()
	for devID, ch := range sc.mobileChans {
		close(ch)
		delete(sc.mobileChans, devID)
	}
	sc.mu.Unlock()

	slog.Info("server removed", "server_id", serverID)
}

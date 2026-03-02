package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	bridgev1 "github.com/syntheticinc/bytebrew/bytebrew-bridge/api/proto/gen"
	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// BridgeConnector maintains an outbound gRPC connection to the bridge relay.
//
// This is an OUTBOUND connection from srv to bridge, which bypasses NAT.
// The bridge relays encrypted frames between mobile devices and this server.
type BridgeConnector struct {
	bridgeAddr string
	serverID   string
	serverName string
	authToken  string

	// gRPC connection to bridge.
	conn   *grpclib.ClientConn
	stream bridgev1.BridgeService_RegisterServerClient

	// Channel for incoming frames from mobile via bridge.
	incomingCh chan *bridgev1.BridgeFrame

	mu        sync.Mutex
	connected bool
	cancel    context.CancelFunc
}

// NewBridgeConnector creates a new connector that will connect to the bridge
// at the given address, identifying itself with the given server ID and name.
func NewBridgeConnector(bridgeAddr, serverID, serverName, authToken string) *BridgeConnector {
	return &BridgeConnector{
		bridgeAddr: bridgeAddr,
		serverID:   serverID,
		serverName: serverName,
		authToken:  authToken,
		incomingCh: make(chan *bridgev1.BridgeFrame, 256),
	}
}

// Connect establishes an outbound gRPC stream to the bridge.
//
// It sends a REGISTER frame, then enters a relay loop reading incoming
// frames. If the connection drops, it automatically reconnects with
// exponential backoff (1, 2, 4, 8, 16, max 30 seconds).
//
// This method blocks until the context is cancelled.
func (c *BridgeConnector) Connect(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	c.mu.Lock()
	c.cancel = cancel
	c.mu.Unlock()

	defer cancel()

	var attempt int
	for {
		err := c.connectOnce(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err != nil {
			slog.ErrorContext(ctx, "bridge connection lost",
				"error", err,
				"attempt", attempt,
			)
		}

		c.setConnected(false)

		delay := c.backoffDelay(attempt)
		attempt++

		slog.InfoContext(ctx, "bridge reconnecting",
			"delay", delay,
			"attempt", attempt,
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
}

// SendToMobile sends an encrypted frame to a mobile device via the bridge.
func (c *BridgeConnector) SendToMobile(deviceID string, payload []byte) error {
	c.mu.Lock()
	stream := c.stream
	connected := c.connected
	c.mu.Unlock()

	if !connected || stream == nil {
		return fmt.Errorf("bridge not connected")
	}

	frame := &bridgev1.BridgeFrame{
		ServerId: c.serverID,
		DeviceId: deviceID,
		Payload:  payload,
		Type:     bridgev1.FrameType_FRAME_TYPE_DATA,
	}

	if err := stream.Send(frame); err != nil {
		return fmt.Errorf("send frame to bridge: %w", err)
	}

	return nil
}

// IncomingFrames returns a read-only channel of frames arriving from
// mobile devices via the bridge.
func (c *BridgeConnector) IncomingFrames() <-chan *bridgev1.BridgeFrame {
	return c.incomingCh
}

// Close gracefully disconnects from the bridge.
func (c *BridgeConnector) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancel != nil {
		c.cancel()
	}

	if c.stream != nil {
		if err := c.stream.CloseSend(); err != nil {
			slog.Warn("failed to close bridge stream", "error", err)
		}
		c.stream = nil
	}

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return fmt.Errorf("close bridge connection: %w", err)
		}
		c.conn = nil
	}

	c.connected = false
	return nil
}

// IsConnected returns whether the connector currently has an active
// connection to the bridge.
func (c *BridgeConnector) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// connectOnce dials the bridge, sends a REGISTER frame, and reads
// incoming frames until the stream ends or an error occurs.
func (c *BridgeConnector) connectOnce(ctx context.Context) error {
	slog.InfoContext(ctx, "connecting to bridge",
		"addr", c.bridgeAddr,
		"server_id", c.serverID,
	)

	conn, err := grpclib.NewClient(
		c.bridgeAddr,
		grpclib.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("dial bridge: %w", err)
	}

	client := bridgev1.NewBridgeServiceClient(conn)
	stream, err := client.RegisterServer(ctx)
	if err != nil {
		conn.Close()
		return fmt.Errorf("open RegisterServer stream: %w", err)
	}

	// Send REGISTER frame to identify ourselves.
	registerFrame := &bridgev1.BridgeFrame{
		ServerId:  c.serverID,
		Payload:   []byte(c.serverName),
		Type:      bridgev1.FrameType_FRAME_TYPE_REGISTER,
		AuthToken: c.authToken,
	}
	if err := stream.Send(registerFrame); err != nil {
		conn.Close()
		return fmt.Errorf("send register frame: %w", err)
	}

	// Store connection state.
	c.mu.Lock()
	c.conn = conn
	c.stream = stream
	c.connected = true
	c.mu.Unlock()

	slog.InfoContext(ctx, "connected to bridge",
		"addr", c.bridgeAddr,
		"server_id", c.serverID,
	)

	// Read loop: forward incoming frames to the channel.
	return c.readLoop(ctx, stream)
}

// readLoop reads frames from the bridge stream and forwards them to
// incomingCh until the stream closes or the context is cancelled.
func (c *BridgeConnector) readLoop(ctx context.Context, stream bridgev1.BridgeService_RegisterServerClient) error {
	for {
		frame, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("recv from bridge: %w", err)
		}

		select {
		case c.incomingCh <- frame:
		case <-ctx.Done():
			return ctx.Err()
		default:
			slog.WarnContext(ctx, "bridge incoming channel full, dropping frame",
				"device_id", frame.GetDeviceId(),
				"type", frame.GetType().String(),
			)
		}
	}
}

// setConnected updates the connected flag under lock.
func (c *BridgeConnector) setConnected(v bool) {
	c.mu.Lock()
	c.connected = v
	c.mu.Unlock()
}

// backoffDelay returns the reconnect delay for the given attempt number.
// Uses exponential backoff: 1s, 2s, 4s, 8s, 16s, capped at 30s.
// The attempt is capped at 5 (1<<5 = 32s) to prevent overflow when attempt > 63.
func (c *BridgeConnector) backoffDelay(attempt int) time.Duration {
	if attempt > 5 {
		attempt = 5
	}
	delay := time.Duration(1<<uint(attempt)) * time.Second
	const maxDelay = 30 * time.Second
	if delay > maxDelay {
		delay = maxDelay
	}
	return delay
}

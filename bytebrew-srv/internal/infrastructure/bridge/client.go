package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	maxReconnectDelay = 30 * time.Second
	connectTimeout    = 10 * time.Second
)

// bridgeMessage represents a message exchanged with the Bridge relay.
type bridgeMessage struct {
	Type            string          `json:"type"`
	ServerID        string          `json:"server_id,omitempty"`
	ServerName      string          `json:"server_name,omitempty"`
	AuthToken       string          `json:"auth_token,omitempty"`
	DeviceID        string          `json:"device_id,omitempty"`
	Payload         json.RawMessage `json:"payload,omitempty"`
	Code            string          `json:"code,omitempty"`
	ServerPublicKey string          `json:"server_public_key,omitempty"`
}

// BridgeClient connects to a Bridge relay server via WebSocket
// and manages the connection lifecycle including automatic reconnection.
type BridgeClient struct {
	url        string
	serverID   string
	serverName string
	authToken  string

	conn *websocket.Conn

	dataHandler       func(deviceID string, payload json.RawMessage)
	deviceConnHandler func(deviceID string)
	deviceDiscHandler func(deviceID string)

	connected bool
	done      chan struct{}
	mu        sync.Mutex
}

// NewBridgeClient creates a new BridgeClient configured to connect to the given relay.
func NewBridgeClient(url, serverID, serverName, authToken string) *BridgeClient {
	return &BridgeClient{
		url:        url,
		serverID:   serverID,
		serverName: serverName,
		authToken:  authToken,
		done:       make(chan struct{}),
	}
}

// OnData sets the handler called when a data message is received from a device.
func (c *BridgeClient) OnData(handler func(deviceID string, payload json.RawMessage)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dataHandler = handler
}

// OnDeviceConnect sets the handler called when a device connects through the bridge.
func (c *BridgeClient) OnDeviceConnect(handler func(deviceID string)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deviceConnHandler = handler
}

// OnDeviceDisconnect sets the handler called when a device disconnects from the bridge.
func (c *BridgeClient) OnDeviceDisconnect(handler func(deviceID string)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deviceDiscHandler = handler
}

// IsConnected returns true if the client has an active connection to the bridge.
func (c *BridgeClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// Connect establishes a WebSocket connection to the bridge, sends the registration
// message, and waits for confirmation. After successful registration it starts
// a background read loop. Returns an error if the initial connection fails.
func (c *BridgeClient) Connect(ctx context.Context) error {
	if err := c.connectAndRegister(ctx); err != nil {
		return fmt.Errorf("initial connect: %w", err)
	}

	go c.readLoop()
	return nil
}

// Disconnect closes the connection and stops any reconnection attempts.
func (c *BridgeClient) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case <-c.done:
		// Already closed.
	default:
		close(c.done)
	}

	c.closeConnLocked()
}

// SendData sends a data message to a specific device through the bridge.
func (c *BridgeClient) SendData(deviceID string, payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	msg := bridgeMessage{
		Type:     "data",
		DeviceID: deviceID,
		Payload:  payloadBytes,
	}

	return c.writeJSON(msg)
}

// SendRegisterCode sends a pairing code and server public key to the bridge
// so that mobile devices can discover this server.
func (c *BridgeClient) SendRegisterCode(code, serverPublicKey string) error {
	msg := bridgeMessage{
		Type:            "register_code",
		Code:            code,
		ServerPublicKey: serverPublicKey,
	}

	return c.writeJSON(msg)
}

func (c *BridgeClient) connectAndRegister(ctx context.Context) error {
	connectCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	wsURL := c.url + "/register"
	slog.InfoContext(ctx, "connecting to bridge", "url", wsURL, "server_id", c.serverID)

	dialer := websocket.Dialer{
		HandshakeTimeout: connectTimeout,
	}

	conn, _, err := dialer.DialContext(connectCtx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial bridge: %w", err)
	}

	// Send registration message.
	regMsg := bridgeMessage{
		Type:       "register",
		ServerID:   c.serverID,
		ServerName: c.serverName,
		AuthToken:  c.authToken,
	}
	if err := conn.WriteJSON(regMsg); err != nil {
		conn.Close()
		return fmt.Errorf("send register: %w", err)
	}

	// Wait for registration confirmation.
	conn.SetReadDeadline(time.Now().Add(connectTimeout))
	var resp bridgeMessage
	if err := conn.ReadJSON(&resp); err != nil {
		conn.Close()
		return fmt.Errorf("read register response: %w", err)
	}
	conn.SetReadDeadline(time.Time{}) // Clear deadline.

	if resp.Type != "registered" {
		conn.Close()
		return fmt.Errorf("unexpected register response type: %q", resp.Type)
	}

	c.mu.Lock()
	c.conn = conn
	c.connected = true
	c.mu.Unlock()

	slog.InfoContext(ctx, "registered with bridge", "server_id", c.serverID)
	return nil
}

func (c *BridgeClient) readLoop() {
	for {
		select {
		case <-c.done:
			return
		default:
		}

		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()

		if conn == nil {
			return
		}

		var msg bridgeMessage
		if err := conn.ReadJSON(&msg); err != nil {
			select {
			case <-c.done:
				return
			default:
			}

			slog.Error("bridge read error, reconnecting", "error", err)
			c.mu.Lock()
			c.connected = false
			c.closeConnLocked()
			c.mu.Unlock()

			c.reconnect()
			continue
		}

		c.handleMessage(msg)
	}
}

func (c *BridgeClient) handleMessage(msg bridgeMessage) {
	c.mu.Lock()
	dataHandler := c.dataHandler
	connHandler := c.deviceConnHandler
	discHandler := c.deviceDiscHandler
	c.mu.Unlock()

	switch msg.Type {
	case "data":
		if dataHandler != nil {
			dataHandler(msg.DeviceID, msg.Payload)
		}
	case "device_connected":
		slog.Info("device connected via bridge", "device_id", msg.DeviceID)
		if connHandler != nil {
			connHandler(msg.DeviceID)
		}
	case "device_disconnected":
		slog.Info("device disconnected from bridge", "device_id", msg.DeviceID)
		if discHandler != nil {
			discHandler(msg.DeviceID)
		}
	default:
		slog.Warn("unknown bridge message type", "type", msg.Type)
	}
}

func (c *BridgeClient) reconnect() {
	delay := time.Second

	for {
		select {
		case <-c.done:
			return
		case <-time.After(delay):
		}

		slog.Info("attempting bridge reconnect", "delay", delay)

		ctx := context.Background()
		if err := c.connectAndRegister(ctx); err != nil {
			slog.Error("bridge reconnect failed", "error", err, "next_delay", delay*2)
			delay *= 2
			if delay > maxReconnectDelay {
				delay = maxReconnectDelay
			}
			continue
		}

		slog.Info("bridge reconnected successfully")
		return
	}
}

func (c *BridgeClient) writeJSON(msg bridgeMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("not connected to bridge")
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		return fmt.Errorf("write to bridge: %w", err)
	}

	return nil
}

// closeConnLocked closes the connection. Must be called with c.mu held.
func (c *BridgeClient) closeConnLocked() {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

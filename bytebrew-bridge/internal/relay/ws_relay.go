package relay

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Message represents a JSON message exchanged over WebSocket.
type Message struct {
	Type       string          `json:"type"`
	ServerID   string          `json:"server_id,omitempty"`
	ServerName string          `json:"server_name,omitempty"`
	AuthToken  string          `json:"auth_token,omitempty"`
	DeviceID   string          `json:"device_id,omitempty"`
	Payload    json.RawMessage `json:"payload,omitempty"`
}

// Message types.
const (
	MsgTypeRegister           = "register"
	MsgTypeRegistered         = "registered"
	MsgTypeData               = "data"
	MsgTypeDeviceConnected    = "device_connected"
	MsgTypeDeviceDisconnected = "device_disconnected"
	MsgTypeError              = "error"
)

// serverConn represents a registered CLI server connection.
type serverConn struct {
	serverID   string
	serverName string
	ws         *websocket.Conn
	writeMu    sync.Mutex
	devices    map[string]*deviceConn
	devicesMu  sync.Mutex
}

// writeJSON sends a JSON message to the CLI server (thread-safe).
func (sc *serverConn) writeJSON(ctx context.Context, msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	sc.writeMu.Lock()
	defer sc.writeMu.Unlock()

	return sc.ws.Write(ctx, websocket.MessageText, data)
}

// deviceConn represents a connected mobile device.
type deviceConn struct {
	ws      *websocket.Conn
	writeMu sync.Mutex
}

// writeJSON sends a JSON message to the mobile device (thread-safe).
func (dc *deviceConn) writeJSON(ctx context.Context, msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	dc.writeMu.Lock()
	defer dc.writeMu.Unlock()

	return dc.ws.Write(ctx, websocket.MessageText, data)
}

// WsRelay manages CLI server registrations and mobile connections.
// CLI registers via WS on /register, mobile connects via WS on /connect.
// All messages are JSON.
type WsRelay struct {
	mu        sync.RWMutex
	servers   map[string]*serverConn
	authToken string
}

// NewWsRelay creates a new WS relay.
// If authToken is non-empty, CLI servers must present a matching token on register.
func NewWsRelay(authToken string) *WsRelay {
	return &WsRelay{
		servers:   make(map[string]*serverConn),
		authToken: authToken,
	}
}

// maxMessageSize is the maximum allowed WebSocket message size (1 MB).
const maxMessageSize = 1 << 20

// HandleRegister handles a CLI server registration WebSocket connection.
// It blocks until the connection closes.
func (r *WsRelay) HandleRegister(ctx context.Context, ws *websocket.Conn) error {
	ws.SetReadLimit(maxMessageSize)

	sc, err := r.registerHandshake(ctx, ws)
	if err != nil {
		return err
	}
	defer r.removeServer(sc)

	return r.readServerMessages(ctx, sc)
}

// registerHandshake reads the register message, validates auth, and registers the server.
func (r *WsRelay) registerHandshake(ctx context.Context, ws *websocket.Conn) (*serverConn, error) {
	msg, err := readJSON(ctx, ws)
	if err != nil {
		return nil, fmt.Errorf("read register message: %w", err)
	}

	if msg.Type != MsgTypeRegister {
		return nil, fmt.Errorf("expected register message, got %q", msg.Type)
	}

	if msg.ServerID == "" {
		return nil, fmt.Errorf("server_id is required")
	}

	if r.authToken != "" {
		if subtle.ConstantTimeCompare([]byte(msg.AuthToken), []byte(r.authToken)) != 1 {
			return nil, fmt.Errorf("invalid auth token for server %s", msg.ServerID)
		}
	}

	sc := &serverConn{
		serverID:   msg.ServerID,
		serverName: msg.ServerName,
		ws:         ws,
		devices:    make(map[string]*deviceConn),
	}

	r.mu.Lock()
	if old, exists := r.servers[msg.ServerID]; exists {
		slog.WarnContext(ctx, "server re-registering, replacing old connection",
			"server_id", msg.ServerID)
		r.closeServerDevices(old)
	}
	r.servers[msg.ServerID] = sc
	r.mu.Unlock()

	slog.InfoContext(ctx, "server registered",
		"server_id", msg.ServerID, "server_name", msg.ServerName)

	if err := sc.writeJSON(ctx, &Message{Type: MsgTypeRegistered}); err != nil {
		r.mu.Lock()
		if r.servers[msg.ServerID] == sc {
			delete(r.servers, msg.ServerID)
		}
		r.mu.Unlock()
		return nil, fmt.Errorf("send registered confirmation: %w", err)
	}

	return sc, nil
}

// readServerMessages reads messages from a registered CLI server and routes them.
func (r *WsRelay) readServerMessages(ctx context.Context, sc *serverConn) error {
	for {
		msg, err := readJSON(ctx, sc.ws)
		if err != nil {
			if isNormalClose(err) {
				slog.InfoContext(ctx, "server disconnected gracefully",
					"server_id", sc.serverID)
				return nil
			}
			return fmt.Errorf("read from server %s: %w", sc.serverID, err)
		}

		if msg.Type != MsgTypeData {
			slog.WarnContext(ctx, "unexpected message type from server",
				"server_id", sc.serverID, "type", msg.Type)
			continue
		}

		r.routeToDevice(ctx, sc, msg)
	}
}

// routeToDevice forwards a data message from CLI to the target mobile device.
func (r *WsRelay) routeToDevice(ctx context.Context, sc *serverConn, msg *Message) {
	sc.devicesMu.Lock()
	dc, ok := sc.devices[msg.DeviceID]
	sc.devicesMu.Unlock()

	if !ok {
		slog.WarnContext(ctx, "device not found, dropping message",
			"server_id", sc.serverID, "device_id", msg.DeviceID)
		return
	}

	outMsg := &Message{
		Type:    MsgTypeData,
		Payload: msg.Payload,
	}

	if err := dc.writeJSON(ctx, outMsg); err != nil {
		slog.ErrorContext(ctx, "failed to send to device",
			"server_id", sc.serverID, "device_id", msg.DeviceID, "error", err)
	}
}

// HandleConnect handles a mobile device WebSocket connection.
// server_id comes from the query parameter. It blocks until the connection closes.
func (r *WsRelay) HandleConnect(ctx context.Context, ws *websocket.Conn, serverID, deviceID string) error {
	ws.SetReadLimit(maxMessageSize)

	if serverID == "" {
		return fmt.Errorf("server_id is required")
	}
	if deviceID == "" {
		return fmt.Errorf("device_id is required")
	}

	r.mu.RLock()
	sc, ok := r.servers[serverID]
	r.mu.RUnlock()

	if !ok {
		return fmt.Errorf("server %s is not online", serverID)
	}

	dc := &deviceConn{ws: ws}

	sc.devicesMu.Lock()
	sc.devices[deviceID] = dc
	sc.devicesMu.Unlock()

	defer r.disconnectDevice(ctx, sc, deviceID)

	// Notify CLI that device connected.
	if err := sc.writeJSON(ctx, &Message{
		Type:     MsgTypeDeviceConnected,
		DeviceID: deviceID,
	}); err != nil {
		slog.ErrorContext(ctx, "failed to notify CLI about device connect",
			"server_id", serverID, "device_id", deviceID, "error", err)
	}

	slog.InfoContext(ctx, "device connected",
		"server_id", serverID, "device_id", deviceID)

	return r.readDeviceMessages(ctx, sc, dc, deviceID)
}

// readDeviceMessages reads messages from a mobile device and forwards them to the CLI server.
func (r *WsRelay) readDeviceMessages(ctx context.Context, sc *serverConn, dc *deviceConn, deviceID string) error {
	for {
		msg, err := readJSON(ctx, dc.ws)
		if err != nil {
			if isNormalClose(err) {
				return nil
			}
			return fmt.Errorf("read from device %s: %w", deviceID, err)
		}

		outMsg := &Message{
			Type:     MsgTypeData,
			DeviceID: deviceID,
			Payload:  msg.Payload,
		}

		if err := sc.writeJSON(ctx, outMsg); err != nil {
			return fmt.Errorf("forward to server %s: %w", sc.serverID, err)
		}
	}
}

// disconnectDevice removes a device from the server and notifies the CLI.
func (r *WsRelay) disconnectDevice(ctx context.Context, sc *serverConn, deviceID string) {
	sc.devicesMu.Lock()
	delete(sc.devices, deviceID)
	sc.devicesMu.Unlock()

	if err := sc.writeJSON(ctx, &Message{
		Type:     MsgTypeDeviceDisconnected,
		DeviceID: deviceID,
	}); err != nil {
		slog.ErrorContext(ctx, "failed to notify CLI about device disconnect",
			"server_id", sc.serverID, "device_id", deviceID, "error", err)
	}

	slog.InfoContext(ctx, "device disconnected",
		"server_id", sc.serverID, "device_id", deviceID)
}

// removeServer removes the server from the pool and closes all device connections.
func (r *WsRelay) removeServer(sc *serverConn) {
	r.mu.Lock()
	current, exists := r.servers[sc.serverID]
	if exists && current == sc {
		delete(r.servers, sc.serverID)
	}
	r.mu.Unlock()

	r.closeServerDevices(sc)

	slog.Info("server removed", "server_id", sc.serverID)
}

// closeServerDevices closes all device connections for a server.
func (r *WsRelay) closeServerDevices(sc *serverConn) {
	sc.devicesMu.Lock()
	for devID, dc := range sc.devices {
		_ = dc.ws.Close(websocket.StatusGoingAway, "server disconnected")
		delete(sc.devices, devID)
	}
	sc.devicesMu.Unlock()
}

// OnlineServer contains public info about a registered server.
type OnlineServer struct {
	ServerID    string    `json:"server_id"`
	ServerName  string    `json:"server_name"`
	ConnectedAt time.Time `json:"connected_at"`
}

// ListOnline returns all currently registered servers.
func (r *WsRelay) ListOnline() []OnlineServer {
	r.mu.RLock()
	defer r.mu.RUnlock()

	servers := make([]OnlineServer, 0, len(r.servers))
	for _, sc := range r.servers {
		servers = append(servers, OnlineServer{
			ServerID:   sc.serverID,
			ServerName: sc.serverName,
		})
	}

	return servers
}

// readJSON reads and unmarshals a JSON message from a WebSocket connection.
func readJSON(ctx context.Context, ws *websocket.Conn) (*Message, error) {
	_, data, err := ws.Read(ctx)
	if err != nil {
		return nil, err
	}

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("unmarshal message: %w", err)
	}

	return &msg, nil
}

// isNormalClose checks if the error represents a normal WebSocket closure or EOF.
func isNormalClose(err error) bool {
	if err == io.EOF {
		return true
	}

	status := websocket.CloseStatus(err)
	return status == websocket.StatusNormalClosure || status == websocket.StatusGoingAway
}

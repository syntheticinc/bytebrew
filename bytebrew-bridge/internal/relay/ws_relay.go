package relay

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

// connIDCounter generates unique connection IDs for race-safe device disconnect.
var connIDCounter atomic.Uint64

// pingInterval is the interval between WebSocket keep-alive pings for CLI servers.
const pingInterval = 15 * time.Second

// deviceKeepAliveInterval is the interval for mobile device keepalive.
// More aggressive (5s) to survive MIUI/Android battery optimizations and
// carrier NAT timeouts that kill idle TCP connections after 30-60s.
const deviceKeepAliveInterval = 5 * time.Second

// shortCodeTTL is how long a registered short code lives before expiring.
const shortCodeTTL = 5 * time.Minute

// Message represents a JSON message exchanged over WebSocket.
type Message struct {
	Type            string          `json:"type"`
	ServerID        string          `json:"server_id,omitempty"`
	ServerName      string          `json:"server_name,omitempty"`
	AuthToken       string          `json:"auth_token,omitempty"`
	DeviceID        string          `json:"device_id,omitempty"`
	Code            string          `json:"code,omitempty"`
	ServerPublicKey string          `json:"server_public_key,omitempty"`
	Payload         json.RawMessage `json:"payload,omitempty"`
}

// Message types.
const (
	MsgTypeRegister           = "register"
	MsgTypeRegistered         = "registered"
	MsgTypeData               = "data"
	MsgTypePing               = "ping"
	MsgTypePong               = "pong"
	MsgTypeDeviceConnected    = "device_connected"
	MsgTypeDeviceDisconnected = "device_disconnected"
	MsgTypeRegisterCode       = "register_code"
)

// shortCodeEntry stores discovery info for a pending pairing short code.
type shortCodeEntry struct {
	serverID        string
	serverPublicKey string // base64-encoded X25519 public key
	expiresAt       time.Time
}

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
	id      string // unique connection ID to detect stale disconnects
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
	mu          sync.RWMutex
	servers     map[string]*serverConn
	authToken   string
	shortCodes  map[string]shortCodeEntry
	shortCodeMu sync.RWMutex
}

// NewWsRelay creates a new WS relay.
// If authToken is non-empty, CLI servers must present a matching token on register.
func NewWsRelay(authToken string) *WsRelay {
	r := &WsRelay{
		servers:    make(map[string]*serverConn),
		authToken:  authToken,
		shortCodes: make(map[string]shortCodeEntry),
	}
	go r.cleanExpiredCodes()
	return r
}

// cleanExpiredCodes periodically removes expired short codes.
func (r *WsRelay) cleanExpiredCodes() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		r.shortCodeMu.Lock()
		for code, entry := range r.shortCodes {
			if now.After(entry.expiresAt) {
				delete(r.shortCodes, code)
			}
		}
		r.shortCodeMu.Unlock()
	}
}

// LookupShortCode returns the server_id and server_public_key for a short code.
// Returns false if the code does not exist or has expired.
func (r *WsRelay) LookupShortCode(code string) (serverID, serverPublicKey string, ok bool) {
	r.shortCodeMu.RLock()
	entry, exists := r.shortCodes[code]
	r.shortCodeMu.RUnlock()
	if !exists || time.Now().After(entry.expiresAt) {
		return "", "", false
	}
	return entry.serverID, entry.serverPublicKey, true
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
// It also starts a keep-alive ping goroutine to detect dead connections.
func (r *WsRelay) readServerMessages(ctx context.Context, sc *serverConn) error {
	pingCtx, pingCancel := context.WithCancel(ctx)
	defer pingCancel()
	go keepAlive(pingCtx, sc.ws, "server", sc.serverID)
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

		switch msg.Type {
		case MsgTypeData:
			r.routeToDevice(ctx, sc, msg)
		case MsgTypeRegisterCode:
			r.handleRegisterCode(ctx, sc, msg)
		default:
			slog.WarnContext(ctx, "unexpected message type from server",
				"server_id", sc.serverID, "type", msg.Type)
		}
	}
}

// handleRegisterCode stores a short code for pairing discovery.
// Mobile can later call GET /lookup?code=XXXXXX to resolve it to a server_id.
func (r *WsRelay) handleRegisterCode(ctx context.Context, sc *serverConn, msg *Message) {
	if msg.Code == "" || msg.ServerPublicKey == "" {
		slog.WarnContext(ctx, "register_code missing fields", "server_id", sc.serverID)
		return
	}

	r.shortCodeMu.Lock()
	r.shortCodes[msg.Code] = shortCodeEntry{
		serverID:        sc.serverID,
		serverPublicKey: msg.ServerPublicKey,
		expiresAt:       time.Now().Add(shortCodeTTL),
	}
	r.shortCodeMu.Unlock()

	slog.InfoContext(ctx, "short code registered",
		"code", msg.Code, "server_id", sc.serverID)
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

	slog.InfoContext(ctx, "forwarding server→device",
		"server_id", sc.serverID, "device_id", msg.DeviceID,
		"payload_len", len(msg.Payload))

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

	connID := fmt.Sprintf("%d", connIDCounter.Add(1))
	dc := &deviceConn{id: connID, ws: ws}

	sc.devicesMu.Lock()
	if old, exists := sc.devices[deviceID]; exists {
		slog.InfoContext(ctx, "device reconnecting, replacing old connection",
			"server_id", serverID, "device_id", deviceID,
			"old_conn", old.id, "new_conn", connID)
		_ = old.ws.Close(websocket.StatusGoingAway, "replaced by new connection")
	}
	sc.devices[deviceID] = dc
	sc.devicesMu.Unlock()

	defer r.disconnectDevice(ctx, sc, deviceID, connID)

	// Send application-level pong messages to devices periodically to keep
	// the connection alive through Caddy reverse proxy and carrier NAT.
	// WS-level pings don't work through Caddy (it doesn't proxy control
	// frames). Instead, we send {"type":"pong"} which the mobile already
	// handles — it updates _lastDataAt, preventing stale detection.
	devicePingCtx, devicePingCancel := context.WithCancel(ctx)
	defer devicePingCancel()
	go keepAliveDevice(devicePingCtx, dc, deviceID)

	// Notify CLI that device connected.
	if err := sc.writeJSON(ctx, &Message{
		Type:     MsgTypeDeviceConnected,
		DeviceID: deviceID,
	}); err != nil {
		slog.ErrorContext(ctx, "failed to notify CLI about device connect",
			"server_id", serverID, "device_id", deviceID, "error", err)
	}

	slog.InfoContext(ctx, "device connected",
		"server_id", serverID, "device_id", deviceID, "conn_id", connID)

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

		// Handle application-level ping from device (keeps NAT alive).
		if msg.Type == MsgTypePing {
			_ = dc.writeJSON(ctx, &Message{Type: MsgTypePong})
			continue
		}

		if msg.Type != MsgTypeData {
			slog.WarnContext(ctx, "unexpected message type from device",
				"device_id", deviceID, "type", msg.Type)
			continue
		}

		slog.InfoContext(ctx, "forwarding device→server",
			"device_id", deviceID, "server_id", sc.serverID,
			"payload_len", len(msg.Payload))

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
// connID ensures only the current connection is removed (not a newer replacement).
func (r *WsRelay) disconnectDevice(ctx context.Context, sc *serverConn, deviceID, connID string) {
	sc.devicesMu.Lock()
	current, exists := sc.devices[deviceID]
	if !exists || current.id != connID {
		// Device was already replaced by a newer connection — do NOT remove it.
		sc.devicesMu.Unlock()
		slog.InfoContext(ctx, "stale disconnect ignored",
			"server_id", sc.serverID, "device_id", deviceID,
			"stale_conn", connID)
		return
	}
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
		"server_id", sc.serverID, "device_id", deviceID, "conn_id", connID)
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
	ServerID   string `json:"server_id"`
	ServerName string `json:"server_name"`
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

// keepAliveDevice sends periodic application-level {"type":"pong"} messages
// to mobile devices. This keeps the TCP connection alive through reverse
// proxies (Caddy) and carrier NAT. The mobile side treats any received data
// as proof of liveness (updates _lastDataAt).
func keepAliveDevice(ctx context.Context, dc *deviceConn, deviceID string) {
	slog.InfoContext(ctx, "device keepalive goroutine started",
		"device_id", deviceID, "interval", deviceKeepAliveInterval.String())

	ticker := time.NewTicker(deviceKeepAliveInterval)
	defer ticker.Stop()

	var count int
	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "device keepalive goroutine exiting (context done)",
				"device_id", deviceID, "total_sent", count)
			return
		case <-ticker.C:
			count++
			if err := dc.writeJSON(ctx, &Message{Type: MsgTypePong}); err != nil {
				slog.WarnContext(ctx, "device keepalive goroutine exiting (write failed)",
					"device_id", deviceID, "count", count, "error", err)
				return
			}
			slog.InfoContext(ctx, "device keepalive sent",
				"device_id", deviceID, "count", count)
		}
	}
}

// keepAlive sends periodic WebSocket pings to detect dead connections.
// It runs until ctx is cancelled.
func keepAlive(ctx context.Context, ws *websocket.Conn, connType, connID string) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			if err := ws.Ping(pingCtx); err != nil {
				cancel()
				slog.WarnContext(ctx, "ping failed, connection likely dead",
					"type", connType, "id", connID, "error", err)
				_ = ws.Close(websocket.StatusGoingAway, "ping timeout")
				return
			}
			cancel()
		}
	}
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

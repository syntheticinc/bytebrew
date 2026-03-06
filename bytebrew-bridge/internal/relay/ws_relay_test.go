package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// setupTestServer creates an httptest.Server with register and connect endpoints
// backed by the given WsRelay, and returns the server and its WS base URL.
func setupTestServer(t *testing.T, r *WsRelay) (*httptest.Server, string) {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /register", func(w http.ResponseWriter, req *http.Request) {
		conn, err := websocket.Accept(w, req, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			return
		}
		defer func() { _ = conn.CloseNow() }()

		if err := r.HandleRegister(req.Context(), conn); err != nil {
			_ = conn.Close(websocket.StatusInternalError, err.Error())
			return
		}
		_ = conn.Close(websocket.StatusNormalClosure, "")
	})
	mux.HandleFunc("GET /connect", func(w http.ResponseWriter, req *http.Request) {
		serverID := req.URL.Query().Get("server_id")
		deviceID := req.URL.Query().Get("device_id")
		if serverID == "" || deviceID == "" {
			http.Error(w, "missing params", http.StatusBadRequest)
			return
		}

		conn, err := websocket.Accept(w, req, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			return
		}
		defer func() { _ = conn.CloseNow() }()

		if err := r.HandleConnect(req.Context(), conn, serverID, deviceID); err != nil {
			_ = conn.Close(websocket.StatusInternalError, err.Error())
			return
		}
		_ = conn.Close(websocket.StatusNormalClosure, "")
	})

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	return ts, wsURL
}

// registerCLI dials the /register endpoint, sends the register message, and
// waits for the "registered" confirmation. Returns the WS conn.
func registerCLI(t *testing.T, ctx context.Context, wsURL, serverID, serverName, authToken string) *websocket.Conn {
	t.Helper()

	conn, _, err := websocket.Dial(ctx, wsURL+"/register", nil)
	if err != nil {
		t.Fatalf("dial register: %v", err)
	}
	conn.SetReadLimit(maxMessageSize)

	reg := Message{
		Type:       MsgTypeRegister,
		ServerID:   serverID,
		ServerName: serverName,
		AuthToken:  authToken,
	}
	writeMsg(t, ctx, conn, &reg)

	// Read "registered" confirmation.
	msg := readMsg(t, ctx, conn)
	if msg.Type != MsgTypeRegistered {
		t.Fatalf("expected registered, got %q", msg.Type)
	}

	return conn
}

// connectMobile dials the /connect endpoint and returns the WS conn.
func connectMobile(t *testing.T, ctx context.Context, wsURL, serverID, deviceID string) *websocket.Conn {
	t.Helper()

	url := fmt.Sprintf("%s/connect?server_id=%s&device_id=%s", wsURL, serverID, deviceID)
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("dial connect: %v", err)
	}
	conn.SetReadLimit(maxMessageSize)

	return conn
}

func writeMsg(t *testing.T, ctx context.Context, conn *websocket.Conn, msg *Message) {
	t.Helper()

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func readMsg(t *testing.T, ctx context.Context, conn *websocket.Conn) *Message {
	t.Helper()

	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	return &msg
}

func readMsgTimeout(ctx context.Context, conn *websocket.Conn, timeout time.Duration) (*Message, error) {
	tctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, data, err := conn.Read(tctx)
	if err != nil {
		return nil, err
	}

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	return &msg, nil
}

func TestRegisterAndListOnline(t *testing.T) {
	r := NewWsRelay("")
	_, wsURL := setupTestServer(t, r)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cli := registerCLI(t, ctx, wsURL, "srv-1", "My Server", "")
	defer cli.Close(websocket.StatusNormalClosure, "")

	// Verify server is listed.
	servers := r.ListOnline()
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	if servers[0].ServerID != "srv-1" {
		t.Errorf("expected server_id=srv-1, got %s", servers[0].ServerID)
	}
	if servers[0].ServerName != "My Server" {
		t.Errorf("expected server_name=My Server, got %s", servers[0].ServerName)
	}
}

func TestBidirectionalRelay(t *testing.T) {
	r := NewWsRelay("")
	_, wsURL := setupTestServer(t, r)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Register CLI server.
	cli := registerCLI(t, ctx, wsURL, "srv-relay", "Relay Server", "")
	defer cli.Close(websocket.StatusNormalClosure, "")

	// Connect mobile.
	mobile := connectMobile(t, ctx, wsURL, "srv-relay", "mobile-1")
	defer mobile.Close(websocket.StatusNormalClosure, "")

	// CLI should get device_connected.
	cliMsg := readMsg(t, ctx, cli)
	if cliMsg.Type != MsgTypeDeviceConnected {
		t.Fatalf("expected device_connected, got %q", cliMsg.Type)
	}
	if cliMsg.DeviceID != "mobile-1" {
		t.Errorf("expected device_id=mobile-1, got %s", cliMsg.DeviceID)
	}

	// Mobile sends data.
	payload := json.RawMessage(`{"text":"hello from mobile"}`)
	writeMsg(t, ctx, mobile, &Message{
		Type:    MsgTypeData,
		Payload: payload,
	})

	// CLI should receive data with device_id.
	cliMsg = readMsg(t, ctx, cli)
	if cliMsg.Type != MsgTypeData {
		t.Fatalf("expected data, got %q", cliMsg.Type)
	}
	if cliMsg.DeviceID != "mobile-1" {
		t.Errorf("expected device_id=mobile-1, got %s", cliMsg.DeviceID)
	}
	if string(cliMsg.Payload) != `{"text":"hello from mobile"}` {
		t.Errorf("expected payload {\"text\":\"hello from mobile\"}, got %s", string(cliMsg.Payload))
	}

	// CLI sends data back to mobile.
	writeMsg(t, ctx, cli, &Message{
		Type:     MsgTypeData,
		DeviceID: "mobile-1",
		Payload:  json.RawMessage(`{"text":"hello from CLI"}`),
	})

	// Mobile should receive data (without device_id).
	mobileMsg := readMsg(t, ctx, mobile)
	if mobileMsg.Type != MsgTypeData {
		t.Fatalf("expected data, got %q", mobileMsg.Type)
	}
	if string(mobileMsg.Payload) != `{"text":"hello from CLI"}` {
		t.Errorf("expected payload {\"text\":\"hello from CLI\"}, got %s", string(mobileMsg.Payload))
	}
}

func TestServerToMobileRelay(t *testing.T) {
	r := NewWsRelay("")
	_, wsURL := setupTestServer(t, r)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cli := registerCLI(t, ctx, wsURL, "srv-s2m", "S2M Server", "")
	defer cli.Close(websocket.StatusNormalClosure, "")

	mobile := connectMobile(t, ctx, wsURL, "srv-s2m", "mobile-s2m")
	defer mobile.Close(websocket.StatusNormalClosure, "")

	// Drain device_connected from CLI.
	readMsg(t, ctx, cli)

	// CLI sends data to mobile.
	writeMsg(t, ctx, cli, &Message{
		Type:     MsgTypeData,
		DeviceID: "mobile-s2m",
		Payload:  json.RawMessage(`{"event":"session_update"}`),
	})

	// Mobile receives data.
	mobileMsg := readMsg(t, ctx, mobile)
	if mobileMsg.Type != MsgTypeData {
		t.Fatalf("expected data, got %q", mobileMsg.Type)
	}
	if string(mobileMsg.Payload) != `{"event":"session_update"}` {
		t.Errorf("unexpected payload: %s", string(mobileMsg.Payload))
	}
}

func TestUnregisteredServer(t *testing.T) {
	r := NewWsRelay("")
	_, wsURL := setupTestServer(t, r)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/connect?server_id=nonexistent&device_id=mobile-1", wsURL)
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow()

	// Connection should be closed by relay with error.
	_, _, err = conn.Read(ctx)
	if err == nil {
		t.Fatal("expected error reading from closed connection, got nil")
	}
}

func TestInvalidAuthToken(t *testing.T) {
	r := NewWsRelay("secret123")
	_, wsURL := setupTestServer(t, r)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL+"/register", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow()

	// Send register with wrong token.
	writeMsg(t, ctx, conn, &Message{
		Type:       MsgTypeRegister,
		ServerID:   "srv-bad-auth",
		ServerName: "Bad Auth Server",
		AuthToken:  "wrong-token",
	})

	// Should receive close frame with error.
	_, _, err = conn.Read(ctx)
	if err == nil {
		t.Fatal("expected error for invalid auth token, got nil")
	}

	// Server should NOT be registered.
	if len(r.ListOnline()) != 0 {
		t.Error("expected 0 servers after invalid auth, got some")
	}
}

func TestValidAuthToken(t *testing.T) {
	r := NewWsRelay("secret123")
	_, wsURL := setupTestServer(t, r)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cli := registerCLI(t, ctx, wsURL, "srv-auth", "Auth Server", "secret123")
	defer cli.Close(websocket.StatusNormalClosure, "")

	servers := r.ListOnline()
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	if servers[0].ServerID != "srv-auth" {
		t.Errorf("expected server_id=srv-auth, got %s", servers[0].ServerID)
	}
}

func TestMobileDisconnect_CLIGetsDeviceDisconnected(t *testing.T) {
	r := NewWsRelay("")
	_, wsURL := setupTestServer(t, r)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cli := registerCLI(t, ctx, wsURL, "srv-disc", "Disconnect Server", "")
	defer cli.Close(websocket.StatusNormalClosure, "")

	mobile := connectMobile(t, ctx, wsURL, "srv-disc", "mobile-disc")

	// Read device_connected from CLI.
	cliMsg := readMsg(t, ctx, cli)
	if cliMsg.Type != MsgTypeDeviceConnected {
		t.Fatalf("expected device_connected, got %q", cliMsg.Type)
	}

	// Close mobile connection.
	mobile.Close(websocket.StatusNormalClosure, "")

	// CLI should receive device_disconnected.
	cliMsg = readMsg(t, ctx, cli)
	if cliMsg.Type != MsgTypeDeviceDisconnected {
		t.Fatalf("expected device_disconnected, got %q", cliMsg.Type)
	}
	if cliMsg.DeviceID != "mobile-disc" {
		t.Errorf("expected device_id=mobile-disc, got %s", cliMsg.DeviceID)
	}
}

func TestLargePayload(t *testing.T) {
	r := NewWsRelay("")
	_, wsURL := setupTestServer(t, r)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cli := registerCLI(t, ctx, wsURL, "srv-large", "Large Server", "")
	defer cli.Close(websocket.StatusNormalClosure, "")

	mobile := connectMobile(t, ctx, wsURL, "srv-large", "mobile-large")
	defer mobile.Close(websocket.StatusNormalClosure, "")

	// Drain device_connected.
	readMsg(t, ctx, cli)

	// Create a 100KB payload.
	largeData := strings.Repeat("A", 100*1024)
	payload, _ := json.Marshal(map[string]string{"data": largeData})

	writeMsg(t, ctx, mobile, &Message{
		Type:    MsgTypeData,
		Payload: payload,
	})

	// CLI should receive the full payload.
	cliMsg := readMsg(t, ctx, cli)
	if cliMsg.Type != MsgTypeData {
		t.Fatalf("expected data, got %q", cliMsg.Type)
	}

	var received map[string]string
	if err := json.Unmarshal(cliMsg.Payload, &received); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if len(received["data"]) != 100*1024 {
		t.Errorf("expected 100KB data, got %d bytes", len(received["data"]))
	}
}

func TestMultipleConcurrentMobiles(t *testing.T) {
	r := NewWsRelay("")
	_, wsURL := setupTestServer(t, r)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cli := registerCLI(t, ctx, wsURL, "srv-multi", "Multi Server", "")
	defer cli.Close(websocket.StatusNormalClosure, "")

	const numDevices = 5
	mobiles := make([]*websocket.Conn, numDevices)

	for i := 0; i < numDevices; i++ {
		deviceID := fmt.Sprintf("mobile-%d", i)
		mobiles[i] = connectMobile(t, ctx, wsURL, "srv-multi", deviceID)
		defer mobiles[i].Close(websocket.StatusNormalClosure, "")

		// Drain device_connected from CLI.
		msg := readMsg(t, ctx, cli)
		if msg.Type != MsgTypeDeviceConnected {
			t.Fatalf("device %d: expected device_connected, got %q", i, msg.Type)
		}
	}

	// Each mobile sends a message.
	var wg sync.WaitGroup
	for i := 0; i < numDevices; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			payload, _ := json.Marshal(map[string]int{"device": idx})
			writeMsg(t, ctx, mobiles[idx], &Message{
				Type:    MsgTypeData,
				Payload: payload,
			})
		}(i)
	}
	wg.Wait()

	// CLI should receive all messages (order not guaranteed).
	received := make(map[string]bool)
	for i := 0; i < numDevices; i++ {
		msg := readMsg(t, ctx, cli)
		if msg.Type != MsgTypeData {
			t.Fatalf("expected data, got %q", msg.Type)
		}
		received[msg.DeviceID] = true
	}

	for i := 0; i < numDevices; i++ {
		deviceID := fmt.Sprintf("mobile-%d", i)
		if !received[deviceID] {
			t.Errorf("did not receive message from %s", deviceID)
		}
	}
}

func TestServerDisconnect_ClosesDeviceConnections(t *testing.T) {
	r := NewWsRelay("")
	_, wsURL := setupTestServer(t, r)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cli := registerCLI(t, ctx, wsURL, "srv-srvdisc", "Srv Disconnect Server", "")

	mobile := connectMobile(t, ctx, wsURL, "srv-srvdisc", "mobile-srvdisc")
	defer mobile.CloseNow()

	// Drain device_connected.
	readMsg(t, ctx, cli)

	// Close CLI connection.
	cli.Close(websocket.StatusNormalClosure, "")

	// Mobile connection should be closed eventually.
	_, _, err := mobile.Read(ctx)
	if err == nil {
		t.Fatal("expected error reading from mobile after server disconnect, got nil")
	}

	// Wait a bit and verify server is removed.
	time.Sleep(100 * time.Millisecond)
	if len(r.ListOnline()) != 0 {
		t.Error("expected 0 servers after CLI disconnect")
	}
}

func TestServerReRegister_ReplacesOldConnection(t *testing.T) {
	r := NewWsRelay("")
	_, wsURL := setupTestServer(t, r)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Register first connection.
	cli1 := registerCLI(t, ctx, wsURL, "srv-rereg", "Server V1", "")

	if len(r.ListOnline()) != 1 {
		t.Fatalf("expected 1 server after first register, got %d", len(r.ListOnline()))
	}

	// Register second connection with same server_id.
	cli2 := registerCLI(t, ctx, wsURL, "srv-rereg", "Server V2", "")
	defer cli2.Close(websocket.StatusNormalClosure, "")

	// Should still have 1 server.
	servers := r.ListOnline()
	if len(servers) != 1 {
		t.Fatalf("expected 1 server after re-register, got %d", len(servers))
	}
	if servers[0].ServerName != "Server V2" {
		t.Errorf("expected server_name=Server V2, got %s", servers[0].ServerName)
	}

	// Old connection should be usable for reading (graceful handling).
	// The old CLI conn may get an error on next read since it was replaced.
	_ = cli1.CloseNow()
}

func TestConnectMissingServerID(t *testing.T) {
	r := NewWsRelay("")
	ts, _ := setupTestServer(t, r)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try connecting without server_id param — should get 400 from HTTP handler.
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	_, resp, err := websocket.Dial(ctx, wsURL+"/connect?device_id=d1", nil)
	if err == nil {
		t.Fatal("expected error for missing server_id")
	}
	if resp != nil && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestConnectMissingDeviceID(t *testing.T) {
	r := NewWsRelay("")
	ts, _ := setupTestServer(t, r)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	_, resp, err := websocket.Dial(ctx, wsURL+"/connect?server_id=srv-1", nil)
	if err == nil {
		t.Fatal("expected error for missing device_id")
	}
	if resp != nil && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestBase64PayloadRelay(t *testing.T) {
	r := NewWsRelay("")
	_, wsURL := setupTestServer(t, r)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cli := registerCLI(t, ctx, wsURL, "srv-b64", "B64 Server", "")
	defer cli.Close(websocket.StatusNormalClosure, "")

	mobile := connectMobile(t, ctx, wsURL, "srv-b64", "mobile-b64")
	defer mobile.Close(websocket.StatusNormalClosure, "")

	// Drain device_connected.
	readMsg(t, ctx, cli)

	// Mobile sends base64-encoded encrypted payload (as a JSON string).
	writeMsg(t, ctx, mobile, &Message{
		Type:    MsgTypeData,
		Payload: json.RawMessage(`"dGhpcyBpcyBlbmNyeXB0ZWQgZGF0YQ=="`),
	})

	// CLI should receive it as-is.
	cliMsg := readMsg(t, ctx, cli)
	if cliMsg.Type != MsgTypeData {
		t.Fatalf("expected data, got %q", cliMsg.Type)
	}
	if string(cliMsg.Payload) != `"dGhpcyBpcyBlbmNyeXB0ZWQgZGF0YQ=="` {
		t.Errorf("expected base64 payload, got %s", string(cliMsg.Payload))
	}
}

func TestCLISendsToNonexistentDevice(t *testing.T) {
	r := NewWsRelay("")
	_, wsURL := setupTestServer(t, r)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cli := registerCLI(t, ctx, wsURL, "srv-nodev", "No Device Server", "")
	defer cli.Close(websocket.StatusNormalClosure, "")

	// CLI sends data to a non-existent device — should not crash.
	writeMsg(t, ctx, cli, &Message{
		Type:     MsgTypeData,
		DeviceID: "ghost-device",
		Payload:  json.RawMessage(`{"test":true}`),
	})

	// Verify relay is still working — no panic, no disconnect.
	mobile := connectMobile(t, ctx, wsURL, "srv-nodev", "real-device")
	defer mobile.Close(websocket.StatusNormalClosure, "")

	cliMsg := readMsg(t, ctx, cli)
	if cliMsg.Type != MsgTypeDeviceConnected {
		t.Fatalf("expected device_connected after sending to ghost, got %q", cliMsg.Type)
	}
}

func TestEmptyServerIDInRegister(t *testing.T) {
	r := NewWsRelay("")
	_, wsURL := setupTestServer(t, r)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL+"/register", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow()

	writeMsg(t, ctx, conn, &Message{
		Type:       MsgTypeRegister,
		ServerID:   "",
		ServerName: "No ID",
	})

	// Should be closed with error.
	_, _, err = conn.Read(ctx)
	if err == nil {
		t.Fatal("expected error for empty server_id, got nil")
	}
}

func TestWrongMessageTypeOnRegister(t *testing.T) {
	r := NewWsRelay("")
	_, wsURL := setupTestServer(t, r)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL+"/register", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow()

	writeMsg(t, ctx, conn, &Message{
		Type:     MsgTypeData,
		ServerID: "srv-1",
	})

	// Should be closed with error.
	_, _, err = conn.Read(ctx)
	if err == nil {
		t.Fatal("expected error for wrong message type, got nil")
	}
}

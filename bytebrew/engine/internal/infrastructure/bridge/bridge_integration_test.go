//go:build bridge_integration

package bridge

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const bridgeHost = "bridge.bytebrew.ai:443"
const bridgeWSURL = "wss://bridge.bytebrew.ai"

func skipIfBridgeUnavailable(t *testing.T) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", bridgeHost, 5*time.Second)
	if err != nil {
		t.Skip("Bridge not reachable:", err)
	}
	conn.Close()
}

// bridgeAuthToken returns the auth token from BRIDGE_AUTH_TOKEN env var.
// Tests that require a real bridge connection are skipped if the token is not set.
func bridgeAuthToken(t *testing.T) string {
	t.Helper()
	token := os.Getenv("BRIDGE_AUTH_TOKEN")
	if token == "" {
		t.Skip("BRIDGE_AUTH_TOKEN not set")
	}
	return token
}

func newTestClient(t *testing.T) *BridgeClient {
	t.Helper()
	token := bridgeAuthToken(t)
	serverID := uuid.New().String()
	return NewBridgeClient(bridgeWSURL, serverID, "integration-test", token)
}

// TC-B-01: Connect to real Bridge, verify registration succeeds.
func TestBridge_ServerRegister(t *testing.T) {
	skipIfBridgeUnavailable(t)

	client := newTestClient(t)
	defer client.Disconnect()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	require.NoError(t, err, "Connect to bridge should succeed")
	assert.True(t, client.IsConnected(), "client should report connected after Connect")
}

// TC-B-16: Connect, force-close WS conn, verify automatic reconnect.
func TestBridge_Reconnect(t *testing.T) {
	skipIfBridgeUnavailable(t)

	client := newTestClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	require.NoError(t, err, "initial connect should succeed")
	assert.True(t, client.IsConnected())

	// Force-close the underlying WS connection to trigger reconnect logic.
	// We only call conn.Close() without setting conn=nil so that readLoop's
	// ReadJSON call returns an error and triggers the reconnect path.
	// Calling Disconnect() would close the done channel and stop reconnect.
	client.mu.Lock()
	if client.conn != nil {
		client.conn.Close()
	}
	client.mu.Unlock()

	// readLoop detects the closed connection and calls reconnect().
	// Exponential backoff starts at 1s, so reconnect should happen within ~3s.
	var reconnected bool
	deadline := time.After(15 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("context expired waiting for reconnect")
		case <-deadline:
			t.Fatal("timed out waiting for reconnect")
		case <-ticker.C:
			if client.IsConnected() {
				reconnected = true
			}
		}
		if reconnected {
			break
		}
	}

	assert.True(t, reconnected, "client should have reconnected automatically")

	// Clean shutdown after successful reconnect.
	client.Disconnect()

	// Verify the done channel is closed (Disconnect signals shutdown).
	select {
	case <-client.done:
		// Expected: done channel is closed.
	default:
		t.Fatal("done channel should be closed after Disconnect")
	}
}

// TC-B-02: Verify SendRegisterCode works on a connected client.
func TestBridge_SendRegisterCode(t *testing.T) {
	skipIfBridgeUnavailable(t)

	client := newTestClient(t)
	defer client.Disconnect()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	require.NoError(t, err)

	code := "123456"
	fakePublicKey := "dGVzdC1wdWJsaWMta2V5" // base64("test-public-key")
	err = client.SendRegisterCode(code, fakePublicKey)
	assert.NoError(t, err, "SendRegisterCode should succeed on connected client")
}

// TC-B-03: Verify SendData returns error when not connected.
func TestBridge_SendDataNotConnected(t *testing.T) {
	// This test does not require bridge connectivity.
	serverID := uuid.New().String()
	client := NewBridgeClient(bridgeWSURL, serverID, "test", "fake-token")

	err := client.SendData("device-1", map[string]string{"hello": "world"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

// TC-B-04: Verify OnData handler registration and client stability.
func TestBridge_OnDataCallback(t *testing.T) {
	skipIfBridgeUnavailable(t)

	client := newTestClient(t)
	defer client.Disconnect()

	var received bool
	var mu sync.Mutex

	client.OnData(func(deviceID string, payload json.RawMessage) {
		mu.Lock()
		received = true
		mu.Unlock()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	require.NoError(t, err)

	// Verify the client stays connected after setting OnData handler.
	// Full data relay testing would require a second (mobile) client.
	time.Sleep(1 * time.Second)
	assert.True(t, client.IsConnected(), "client should remain connected after setting OnData")

	_ = received
}

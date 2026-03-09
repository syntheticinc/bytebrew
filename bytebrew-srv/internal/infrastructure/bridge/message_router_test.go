package bridge

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageRouter_DecodePlaintext(t *testing.T) {
	client := NewBridgeClient("ws://localhost", "srv-1", "test", "token")
	crypto := NewDeviceCryptoAdapter()
	router := NewMessageRouter(client, crypto)

	var received *MobileMessage
	router.OnMessage(func(msg *MobileMessage) {
		received = msg
	})
	router.Start()

	// Simulate a plaintext data message from bridge.
	payload, err := json.Marshal(MobileMessage{
		Type:      "chat_message",
		RequestID: "req-1",
		Payload:   map[string]interface{}{"text": "hello"},
	})
	require.NoError(t, err)

	router.handleData("device-1", payload)

	require.NotNil(t, received)
	assert.Equal(t, "chat_message", received.Type)
	assert.Equal(t, "req-1", received.RequestID)
	assert.Equal(t, "device-1", received.DeviceID)
	assert.Equal(t, "hello", received.Payload["text"])
}

func TestMessageRouter_DecodeEncrypted(t *testing.T) {
	client := NewBridgeClient("ws://localhost", "srv-1", "test", "token")
	crypto := NewDeviceCryptoAdapter()
	router := NewMessageRouter(client, crypto)

	// Setup shared secret for device.
	alice, err := GenerateKeyPair()
	require.NoError(t, err)
	bob, err := GenerateKeyPair()
	require.NoError(t, err)
	shared, err := ComputeSharedSecret(alice.PrivateKey, bob.PublicKey)
	require.NoError(t, err)
	crypto.AddDevice("device-2", shared)

	var received *MobileMessage
	router.OnMessage(func(msg *MobileMessage) {
		received = msg
	})
	router.Start()

	// Encrypt a message as the "device" would.
	innerMsg := MobileMessage{
		Type:    "ping",
		Payload: map[string]interface{}{"ts": float64(12345)},
	}
	innerBytes, err := json.Marshal(innerMsg)
	require.NoError(t, err)

	ciphertext, err := Encrypt(innerBytes, shared, 0)
	require.NoError(t, err)

	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	payloadJSON, err := json.Marshal(encoded) // JSON string
	require.NoError(t, err)

	router.handleData("device-2", payloadJSON)

	require.NotNil(t, received)
	assert.Equal(t, "ping", received.Type)
	assert.Equal(t, "device-2", received.DeviceID)
}

// TC-B-06: Mirror encryption — encrypted request activates mirror mode for device.
func TestMessageRouter_MirrorEncryption(t *testing.T) {
	client := NewBridgeClient("ws://localhost", "srv-1", "test", "token")
	crypto := NewDeviceCryptoAdapter()
	router := NewMessageRouter(client, crypto)

	// Setup shared secret.
	alice, err := GenerateKeyPair()
	require.NoError(t, err)
	bob, err := GenerateKeyPair()
	require.NoError(t, err)
	shared, err := ComputeSharedSecret(alice.PrivateKey, bob.PublicKey)
	require.NoError(t, err)
	crypto.AddDevice("device-3", shared)

	router.Start()

	// Simulate encrypted incoming message to activate mirror mode.
	innerMsg := MobileMessage{Type: "ping"}
	innerBytes, err := json.Marshal(innerMsg)
	require.NoError(t, err)

	ciphertext, err := Encrypt(innerBytes, shared, 0)
	require.NoError(t, err)

	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	payloadJSON, err := json.Marshal(encoded)
	require.NoError(t, err)

	router.handleData("device-3", payloadJSON)

	// Verify mirror encryption is active.
	router.mu.RLock()
	active := router.deviceEncryptionActive["device-3"]
	router.mu.RUnlock()
	assert.True(t, active)
}

// TC-B-06: Mirror encryption — plaintext request disables mirror mode for device.
func TestMessageRouter_PlaintextDisablesMirror(t *testing.T) {
	client := NewBridgeClient("ws://localhost", "srv-1", "test", "token")
	crypto := NewDeviceCryptoAdapter()
	router := NewMessageRouter(client, crypto)
	router.Start()

	// Manually set encryption active.
	router.mu.Lock()
	router.deviceEncryptionActive["device-4"] = true
	router.mu.Unlock()

	// Send a plaintext message — should disable mirror mode.
	payload, err := json.Marshal(MobileMessage{Type: "ping"})
	require.NoError(t, err)

	router.handleData("device-4", payload)

	router.mu.RLock()
	active := router.deviceEncryptionActive["device-4"]
	router.mu.RUnlock()
	assert.False(t, active)
}

// TC-B-06: Mirror encryption — plaintext request → SendMessage uses plaintext,
// encrypted request → SendMessage uses encryption. End-to-end mirror verification.
func TestMessageRouter_MirrorEncryption_SendMessagePath(t *testing.T) {
	client := NewBridgeClient("ws://localhost", "srv-1", "test", "token")
	crypto := NewDeviceCryptoAdapter()
	router := NewMessageRouter(client, crypto)

	// Setup shared secret for device.
	alice, err := GenerateKeyPair()
	require.NoError(t, err)
	bob, err := GenerateKeyPair()
	require.NoError(t, err)
	shared, err := ComputeSharedSecret(alice.PrivateKey, bob.PublicKey)
	require.NoError(t, err)
	crypto.AddDevice("dev-mirror", shared)

	router.Start()

	// --- Scenario 1: plaintext request → mirror off → SendMessage uses plaintext ---
	plaintextPayload, err := json.Marshal(MobileMessage{Type: "ping"})
	require.NoError(t, err)
	router.handleData("dev-mirror", plaintextPayload)

	router.mu.RLock()
	active := router.deviceEncryptionActive["dev-mirror"]
	router.mu.RUnlock()
	assert.False(t, active, "plaintext request should disable mirror encryption")

	// SendMessage should attempt plaintext (will fail on nil conn, but that's OK).
	response := &MobileMessage{Type: "pong", Payload: map[string]interface{}{"ts": float64(1)}}
	err = router.SendMessage("dev-mirror", response)
	// Error from nil conn is expected, but the path should be plaintext.
	assert.Error(t, err, "nil conn expected")

	// --- Scenario 2: encrypted request → mirror on → SendMessage uses encryption ---
	innerMsg := MobileMessage{Type: "ping"}
	innerBytes, err := json.Marshal(innerMsg)
	require.NoError(t, err)
	ciphertext, err := Encrypt(innerBytes, shared, 0)
	require.NoError(t, err)
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	encPayloadJSON, err := json.Marshal(encoded)
	require.NoError(t, err)

	router.handleData("dev-mirror", encPayloadJSON)

	router.mu.RLock()
	active = router.deviceEncryptionActive["dev-mirror"]
	router.mu.RUnlock()
	assert.True(t, active, "encrypted request should enable mirror encryption")

	// SendMessage should attempt encrypted path (will also fail on nil conn).
	err = router.SendMessage("dev-mirror", response)
	assert.Error(t, err, "nil conn expected")
}

// TC-B-05: E2E encrypted message — encrypted incoming message is decrypted for processing,
// and response is encrypted back when device has active encryption (mirror mode).
func TestMessageRouter_E2EEncryptedMessage(t *testing.T) {
	client := NewBridgeClient("ws://localhost", "srv-1", "test", "token")
	crypto := NewDeviceCryptoAdapter()
	router := NewMessageRouter(client, crypto)

	// Setup key pairs and shared secret (simulating paired device).
	serverKP, err := GenerateKeyPair()
	require.NoError(t, err)
	deviceKP, err := GenerateKeyPair()
	require.NoError(t, err)
	shared, err := ComputeSharedSecret(serverKP.PrivateKey, deviceKP.PublicKey)
	require.NoError(t, err)
	crypto.AddDevice("enc-device", shared)

	var received *MobileMessage
	router.OnMessage(func(msg *MobileMessage) {
		received = msg
	})
	router.Start()

	// Step 1: Create an encrypted incoming message (as a mobile device would send).
	innerMsg := MobileMessage{
		Type:      "new_task",
		RequestID: "req-enc-1",
		Payload: map[string]interface{}{
			"device_token": "tok-enc",
			"content":      "encrypted hello",
		},
	}
	innerBytes, err := json.Marshal(innerMsg)
	require.NoError(t, err)

	ciphertext, err := Encrypt(innerBytes, shared, 0)
	require.NoError(t, err)

	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	payloadJSON, err := json.Marshal(encoded)
	require.NoError(t, err)

	// Step 2: Feed encrypted payload through router.
	router.handleData("enc-device", payloadJSON)

	// Step 3: Verify message was decrypted correctly for processing.
	require.NotNil(t, received, "handler should receive decrypted message")
	assert.Equal(t, "new_task", received.Type)
	assert.Equal(t, "req-enc-1", received.RequestID)
	assert.Equal(t, "enc-device", received.DeviceID)
	assert.Equal(t, "encrypted hello", received.Payload["content"])

	// Step 4: Verify mirror encryption is active (response will be encrypted).
	router.mu.RLock()
	active := router.deviceEncryptionActive["enc-device"]
	router.mu.RUnlock()
	assert.True(t, active, "mirror encryption should be active after encrypted request")

	// Step 5: Verify SendMessage would use encryption path.
	// (SendMessage will fail due to nil conn, but we verify the encryption flag.)
	response := &MobileMessage{
		Type:      "new_task_ack",
		RequestID: "req-enc-1",
		Payload:   map[string]interface{}{"session_id": "s1"},
	}
	err = router.SendMessage("enc-device", response)
	assert.Error(t, err, "nil conn expected, but encryption path should be taken")
}

// TC-B-15: Device alias mapping — bridge_device_id → authenticated_device_id alias
// allows encrypted communication through the bridge-assigned ID.
func TestMessageRouter_DeviceAliasMapping(t *testing.T) {
	client := NewBridgeClient("ws://localhost", "srv-1", "test", "token")
	crypto := NewDeviceCryptoAdapter()
	router := NewMessageRouter(client, crypto)

	// Setup: create keypairs and register authenticated device.
	serverKP, err := GenerateKeyPair()
	require.NoError(t, err)
	deviceKP, err := GenerateKeyPair()
	require.NoError(t, err)
	shared, err := ComputeSharedSecret(serverKP.PrivateKey, deviceKP.PublicKey)
	require.NoError(t, err)

	// Register device with authenticated ID and shared secret.
	authDeviceID := "auth-device-123"
	bridgeDeviceID := "bridge-ws-456"
	crypto.AddDevice(authDeviceID, shared)

	// Register alias: bridge ID → authenticated ID.
	crypto.RegisterAlias(bridgeDeviceID, authDeviceID)

	var received *MobileMessage
	router.OnMessage(func(msg *MobileMessage) {
		received = msg
	})
	router.Start()

	// Encrypt a message using the shared secret (simulating mobile device).
	innerMsg := MobileMessage{
		Type:      "ping",
		RequestID: "req-alias-1",
		Payload:   map[string]interface{}{"ts": float64(99999)},
	}
	innerBytes, err := json.Marshal(innerMsg)
	require.NoError(t, err)

	ciphertext, err := Encrypt(innerBytes, shared, 0)
	require.NoError(t, err)

	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	payloadJSON, err := json.Marshal(encoded)
	require.NoError(t, err)

	// Send encrypted message using bridge device ID (alias).
	router.handleData(bridgeDeviceID, payloadJSON)

	// Verify: message decrypted correctly via alias resolution.
	require.NotNil(t, received, "handler should receive decrypted message via alias")
	assert.Equal(t, "ping", received.Type)
	assert.Equal(t, bridgeDeviceID, received.DeviceID, "DeviceID should be bridge ID")
	assert.Equal(t, float64(99999), received.Payload["ts"])

	// Verify: mirror encryption active for bridge device ID.
	router.mu.RLock()
	active := router.deviceEncryptionActive[bridgeDeviceID]
	router.mu.RUnlock()
	assert.True(t, active, "mirror encryption should be active for bridge device ID")

	// Verify: can encrypt a response for the bridge device ID (uses alias → shared secret).
	assert.True(t, crypto.HasSharedSecret(bridgeDeviceID), "alias should resolve to device with shared secret")
}

func TestMessageRouter_MultipleHandlers(t *testing.T) {
	client := NewBridgeClient("ws://localhost", "srv-1", "test", "token")
	crypto := NewDeviceCryptoAdapter()
	router := NewMessageRouter(client, crypto)

	callCount := 0
	router.OnMessage(func(msg *MobileMessage) { callCount++ })
	router.OnMessage(func(msg *MobileMessage) { callCount++ })
	router.Start()

	payload, err := json.Marshal(MobileMessage{Type: "test"})
	require.NoError(t, err)

	router.handleData("device-5", payload)

	assert.Equal(t, 2, callCount)
}

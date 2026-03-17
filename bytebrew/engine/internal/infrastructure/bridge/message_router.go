package bridge

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
)

// MessageRouter routes messages between the BridgeClient and business logic,
// handling E2E encryption transparently. It uses mirror encryption: responses
// are encrypted only if the corresponding request was encrypted.
type MessageRouter struct {
	client *BridgeClient
	crypto *DeviceCryptoAdapter

	// Mirror encryption: tracks which devices are sending encrypted messages.
	deviceEncryptionActive map[string]bool

	messageHandlers []func(msg *MobileMessage)
	mu              sync.RWMutex
}

// NewMessageRouter creates a new MessageRouter.
func NewMessageRouter(client *BridgeClient, crypto *DeviceCryptoAdapter) *MessageRouter {
	return &MessageRouter{
		client:                 client,
		crypto:                 crypto,
		deviceEncryptionActive: make(map[string]bool),
	}
}

// OnMessage registers a handler that will be called for each incoming mobile message.
func (r *MessageRouter) OnMessage(handler func(msg *MobileMessage)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messageHandlers = append(r.messageHandlers, handler)
}

// Start subscribes to BridgeClient data events and begins routing messages.
func (r *MessageRouter) Start() {
	r.client.OnData(r.handleData)
}

// Stop is a no-op placeholder for cleanup if needed in the future.
func (r *MessageRouter) Stop() {
	// Currently no background goroutines to stop.
}

// SendMessage sends a message to a device, applying mirror encryption
// if the device previously sent an encrypted request.
func (r *MessageRouter) SendMessage(deviceID string, msg *MobileMessage) error {
	r.mu.RLock()
	encrypted := r.deviceEncryptionActive[deviceID]
	r.mu.RUnlock()

	if encrypted && r.crypto.HasSharedSecret(deviceID) {
		return r.sendEncrypted(deviceID, msg)
	}

	return r.sendPlaintext(deviceID, msg)
}

func (r *MessageRouter) handleData(deviceID string, payload json.RawMessage) {
	slog.Info("bridge data received", "device_id", deviceID, "payload_len", len(payload))
	msg, err := r.decodePayload(deviceID, payload)
	if err != nil {
		slog.Error("failed to decode bridge payload", "device_id", deviceID, "error", err)
		return
	}

	slog.Info("bridge message decoded", "device_id", deviceID, "type", msg.Type)
	msg.DeviceID = deviceID

	r.mu.RLock()
	handlers := make([]func(msg *MobileMessage), len(r.messageHandlers))
	copy(handlers, r.messageHandlers)
	r.mu.RUnlock()

	for _, handler := range handlers {
		handler(msg)
	}
}

func (r *MessageRouter) decodePayload(deviceID string, payload json.RawMessage) (*MobileMessage, error) {
	// First, try to parse as plaintext JSON MobileMessage.
	var msg MobileMessage
	if err := json.Unmarshal(payload, &msg); err == nil && msg.Type != "" {
		r.mu.Lock()
		r.deviceEncryptionActive[deviceID] = false
		r.mu.Unlock()
		return &msg, nil
	}

	// Try to parse as a base64-encoded encrypted string.
	var encoded string
	if err := json.Unmarshal(payload, &encoded); err != nil {
		return nil, fmt.Errorf("payload is neither JSON object nor string: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}

	plaintext, err := r.crypto.Decrypt(deviceID, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decrypt payload: %w", err)
	}

	var decryptedMsg MobileMessage
	if err := json.Unmarshal(plaintext, &decryptedMsg); err != nil {
		return nil, fmt.Errorf("unmarshal decrypted payload: %w", err)
	}

	r.mu.Lock()
	r.deviceEncryptionActive[deviceID] = true
	r.mu.Unlock()

	return &decryptedMsg, nil
}

func (r *MessageRouter) sendEncrypted(deviceID string, msg *MobileMessage) error {
	plaintext, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	ciphertext, err := r.crypto.Encrypt(deviceID, plaintext)
	if err != nil {
		return fmt.Errorf("encrypt message: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return r.client.SendData(deviceID, encoded)
}

func (r *MessageRouter) sendPlaintext(deviceID string, msg *MobileMessage) error {
	return r.client.SendData(deviceID, msg)
}

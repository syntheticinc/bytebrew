package bridge

import (
	"fmt"
	"sync"
)

// DeviceCryptoAdapter maps device IDs to shared secrets and provides
// per-device encryption/decryption. It supports alias mapping from
// bridge-assigned device IDs to authenticated device IDs.
type DeviceCryptoAdapter struct {
	secrets  map[string][]byte  // deviceID -> sharedSecret
	aliases  map[string]string  // bridgeDeviceID -> authenticatedDeviceID
	counters map[string]uint64  // deviceID -> encrypt counter
	mu       sync.RWMutex
}

// NewDeviceCryptoAdapter creates a new DeviceCryptoAdapter.
func NewDeviceCryptoAdapter() *DeviceCryptoAdapter {
	return &DeviceCryptoAdapter{
		secrets:  make(map[string][]byte),
		aliases:  make(map[string]string),
		counters: make(map[string]uint64),
	}
}

// AddDevice registers a device with its shared secret for E2E encryption.
func (a *DeviceCryptoAdapter) AddDevice(deviceID string, sharedSecret []byte) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.secrets[deviceID] = sharedSecret
	a.counters[deviceID] = 0
}

// RemoveDevice removes a device and its shared secret.
func (a *DeviceCryptoAdapter) RemoveDevice(deviceID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.secrets, deviceID)
	delete(a.counters, deviceID)

	// Remove any aliases pointing to this device.
	for alias, target := range a.aliases {
		if target == deviceID {
			delete(a.aliases, alias)
		}
	}
}

// HasSharedSecret returns true if a shared secret exists for the device
// (resolving aliases if needed).
func (a *DeviceCryptoAdapter) HasSharedSecret(deviceID string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	resolved := a.resolveAlias(deviceID)
	_, ok := a.secrets[resolved]
	return ok
}

// Encrypt encrypts plaintext for the given device using its shared secret.
// Resolves aliases and increments the per-device counter.
func (a *DeviceCryptoAdapter) Encrypt(deviceID string, plaintext []byte) ([]byte, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	resolved := a.resolveAlias(deviceID)

	secret, ok := a.secrets[resolved]
	if !ok {
		return nil, fmt.Errorf("no shared secret for device %q", deviceID)
	}

	counter := a.counters[resolved]
	a.counters[resolved] = counter + 1

	encrypted, err := Encrypt(plaintext, secret, counter)
	if err != nil {
		return nil, fmt.Errorf("encrypt for device %q: %w", deviceID, err)
	}

	return encrypted, nil
}

// Decrypt decrypts ciphertext from the given device using its shared secret.
// Resolves aliases before looking up the secret.
func (a *DeviceCryptoAdapter) Decrypt(deviceID string, ciphertext []byte) ([]byte, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	resolved := a.resolveAlias(deviceID)

	secret, ok := a.secrets[resolved]
	if !ok {
		return nil, fmt.Errorf("no shared secret for device %q", deviceID)
	}

	plaintext, err := Decrypt(ciphertext, secret)
	if err != nil {
		return nil, fmt.Errorf("decrypt from device %q: %w", deviceID, err)
	}

	return plaintext, nil
}

// RegisterAlias maps a bridge-assigned device ID to an authenticated device ID.
// IMPORTANT: pair_response must be sent BEFORE calling RegisterAlias,
// because the pair_response is sent in plaintext and the alias switches
// encryption to use the authenticated device's shared secret.
func (a *DeviceCryptoAdapter) RegisterAlias(bridgeDeviceID, authenticatedDeviceID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.aliases[bridgeDeviceID] = authenticatedDeviceID
}

// resolveAlias resolves a device ID through the alias map.
// Must be called with at least a read lock held.
func (a *DeviceCryptoAdapter) resolveAlias(deviceID string) string {
	if resolved, ok := a.aliases[deviceID]; ok {
		return resolved
	}
	return deviceID
}

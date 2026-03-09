package bridge

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAdapterWithDevice(t *testing.T) (*DeviceCryptoAdapter, []byte) {
	t.Helper()

	adapter := NewDeviceCryptoAdapter()

	alice, err := GenerateKeyPair()
	require.NoError(t, err)

	bob, err := GenerateKeyPair()
	require.NoError(t, err)

	shared, err := ComputeSharedSecret(alice.PrivateKey, bob.PublicKey)
	require.NoError(t, err)

	adapter.AddDevice("device-1", shared)
	return adapter, shared
}

func TestDeviceCryptoAdapter_AddAndHas(t *testing.T) {
	adapter, _ := setupAdapterWithDevice(t)

	assert.True(t, adapter.HasSharedSecret("device-1"))
	assert.False(t, adapter.HasSharedSecret("device-unknown"))
}

func TestDeviceCryptoAdapter_EncryptDecrypt(t *testing.T) {
	adapter, _ := setupAdapterWithDevice(t)

	plaintext := []byte(`{"type":"ping"}`)

	encrypted, err := adapter.Encrypt("device-1", plaintext)
	require.NoError(t, err)

	decrypted, err := adapter.Decrypt("device-1", encrypted)
	require.NoError(t, err)

	assert.Equal(t, plaintext, decrypted)
}

func TestDeviceCryptoAdapter_EncryptUnknownDevice(t *testing.T) {
	adapter := NewDeviceCryptoAdapter()

	_, err := adapter.Encrypt("unknown", []byte("test"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no shared secret")
}

func TestDeviceCryptoAdapter_DecryptUnknownDevice(t *testing.T) {
	adapter := NewDeviceCryptoAdapter()

	_, err := adapter.Decrypt("unknown", make([]byte, 100))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no shared secret")
}

func TestDeviceCryptoAdapter_RemoveDevice(t *testing.T) {
	adapter, _ := setupAdapterWithDevice(t)

	adapter.RemoveDevice("device-1")

	assert.False(t, adapter.HasSharedSecret("device-1"))
	_, err := adapter.Encrypt("device-1", []byte("test"))
	require.Error(t, err)
}

func TestDeviceCryptoAdapter_Alias(t *testing.T) {
	adapter, _ := setupAdapterWithDevice(t)

	// Register an alias: bridge-assigned ID → authenticated ID.
	adapter.RegisterAlias("bridge-xyz", "device-1")

	// Should be able to encrypt/decrypt using the alias.
	assert.True(t, adapter.HasSharedSecret("bridge-xyz"))

	plaintext := []byte(`{"type":"hello"}`)

	encrypted, err := adapter.Encrypt("bridge-xyz", plaintext)
	require.NoError(t, err)

	decrypted, err := adapter.Decrypt("bridge-xyz", encrypted)
	require.NoError(t, err)

	assert.Equal(t, plaintext, decrypted)
}

func TestDeviceCryptoAdapter_RemoveDeviceCleansAliases(t *testing.T) {
	adapter, _ := setupAdapterWithDevice(t)

	adapter.RegisterAlias("alias-1", "device-1")
	adapter.RegisterAlias("alias-2", "device-1")

	adapter.RemoveDevice("device-1")

	// Aliases should no longer resolve.
	assert.False(t, adapter.HasSharedSecret("alias-1"))
	assert.False(t, adapter.HasSharedSecret("alias-2"))
}

func TestDeviceCryptoAdapter_CounterIncrements(t *testing.T) {
	adapter, _ := setupAdapterWithDevice(t)

	plaintext := []byte("same message")

	enc1, err := adapter.Encrypt("device-1", plaintext)
	require.NoError(t, err)

	enc2, err := adapter.Encrypt("device-1", plaintext)
	require.NoError(t, err)

	// Different counters → different ciphertext.
	assert.NotEqual(t, enc1, enc2)

	// Both should decrypt correctly.
	dec1, err := adapter.Decrypt("device-1", enc1)
	require.NoError(t, err)
	assert.Equal(t, plaintext, dec1)

	dec2, err := adapter.Decrypt("device-1", enc2)
	require.NoError(t, err)
	assert.Equal(t, plaintext, dec2)
}

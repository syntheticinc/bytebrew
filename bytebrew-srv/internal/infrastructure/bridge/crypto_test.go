package bridge

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKeyPair(t *testing.T) {
	kp, err := GenerateKeyPair()
	require.NoError(t, err)
	assert.Len(t, kp.PublicKey, 32)
	assert.Len(t, kp.PrivateKey, 32)
	assert.NotEqual(t, kp.PublicKey, kp.PrivateKey)
}

func TestGenerateKeyPair_Unique(t *testing.T) {
	kp1, err := GenerateKeyPair()
	require.NoError(t, err)

	kp2, err := GenerateKeyPair()
	require.NoError(t, err)

	assert.NotEqual(t, kp1.PrivateKey, kp2.PrivateKey)
	assert.NotEqual(t, kp1.PublicKey, kp2.PublicKey)
}

func TestComputeSharedSecret(t *testing.T) {
	alice, err := GenerateKeyPair()
	require.NoError(t, err)

	bob, err := GenerateKeyPair()
	require.NoError(t, err)

	// Alice computes shared secret with Bob's public key.
	secretAlice, err := ComputeSharedSecret(alice.PrivateKey, bob.PublicKey)
	require.NoError(t, err)

	// Bob computes shared secret with Alice's public key.
	secretBob, err := ComputeSharedSecret(bob.PrivateKey, alice.PublicKey)
	require.NoError(t, err)

	// Both should derive the same shared secret.
	assert.Equal(t, secretAlice, secretBob)
	assert.Len(t, secretAlice, 32)
}

func TestComputeSharedSecret_InvalidKeyLength(t *testing.T) {
	tests := []struct {
		name       string
		privateKey []byte
		publicKey  []byte
	}{
		{"short private key", make([]byte, 16), make([]byte, 32)},
		{"short public key", make([]byte, 32), make([]byte, 16)},
		{"empty keys", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ComputeSharedSecret(tt.privateKey, tt.publicKey)
			require.Error(t, err)
		})
	}
}

func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	alice, err := GenerateKeyPair()
	require.NoError(t, err)

	bob, err := GenerateKeyPair()
	require.NoError(t, err)

	shared, err := ComputeSharedSecret(alice.PrivateKey, bob.PublicKey)
	require.NoError(t, err)

	plaintext := []byte("hello world, this is a secret message")

	encrypted, err := Encrypt(plaintext, shared, 0)
	require.NoError(t, err)

	// Encrypted should be larger than plaintext (nonce + tag overhead).
	assert.Greater(t, len(encrypted), len(plaintext))

	decrypted, err := Decrypt(encrypted, shared)
	require.NoError(t, err)

	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptDecrypt_DifferentCounters(t *testing.T) {
	kp, err := GenerateKeyPair()
	require.NoError(t, err)

	shared, err := ComputeSharedSecret(kp.PrivateKey, kp.PublicKey)
	require.NoError(t, err)

	plaintext := []byte("test message")

	enc0, err := Encrypt(plaintext, shared, 0)
	require.NoError(t, err)

	enc1, err := Encrypt(plaintext, shared, 1)
	require.NoError(t, err)

	// Different counters should produce different ciphertext.
	assert.NotEqual(t, enc0, enc1)

	// Both should decrypt successfully.
	dec0, err := Decrypt(enc0, shared)
	require.NoError(t, err)
	assert.Equal(t, plaintext, dec0)

	dec1, err := Decrypt(enc1, shared)
	require.NoError(t, err)
	assert.Equal(t, plaintext, dec1)
}

func TestDecrypt_WrongKey(t *testing.T) {
	alice, err := GenerateKeyPair()
	require.NoError(t, err)

	bob, err := GenerateKeyPair()
	require.NoError(t, err)

	shared, err := ComputeSharedSecret(alice.PrivateKey, bob.PublicKey)
	require.NoError(t, err)

	encrypted, err := Encrypt([]byte("secret"), shared, 0)
	require.NoError(t, err)

	// Generate a different key to decrypt with.
	wrongKey := make([]byte, 32)
	copy(wrongKey, shared)
	wrongKey[0] ^= 0xFF

	_, err = Decrypt(encrypted, wrongKey)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decrypt")
}

func TestDecrypt_TruncatedCiphertext(t *testing.T) {
	_, err := Decrypt([]byte("short"), make([]byte, 32))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too short")
}

func TestEncrypt_InvalidSecretLength(t *testing.T) {
	_, err := Encrypt([]byte("test"), make([]byte, 16), 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "shared secret must be")
}

func TestDecrypt_InvalidSecretLength(t *testing.T) {
	_, err := Decrypt(make([]byte, 100), make([]byte, 16))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "shared secret must be")
}

func TestEncryptDecrypt_EmptyPlaintext(t *testing.T) {
	kp, err := GenerateKeyPair()
	require.NoError(t, err)

	shared, err := ComputeSharedSecret(kp.PrivateKey, kp.PublicKey)
	require.NoError(t, err)

	encrypted, err := Encrypt([]byte{}, shared, 0)
	require.NoError(t, err)

	decrypted, err := Decrypt(encrypted, shared)
	require.NoError(t, err)

	assert.Empty(t, decrypted)
}

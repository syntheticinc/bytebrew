package mobile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKeypair(t *testing.T) {
	crypto := NewCryptoService()

	t.Run("returns valid keypair", func(t *testing.T) {
		pub, priv, err := crypto.GenerateKeypair()
		require.NoError(t, err)
		assert.Len(t, pub, 32, "public key must be 32 bytes")
		assert.Len(t, priv, 32, "private key must be 32 bytes")
	})

	t.Run("generates different keys each time", func(t *testing.T) {
		pub1, priv1, err := crypto.GenerateKeypair()
		require.NoError(t, err)

		pub2, priv2, err := crypto.GenerateKeypair()
		require.NoError(t, err)

		assert.NotEqual(t, pub1, pub2, "public keys should differ")
		assert.NotEqual(t, priv1, priv2, "private keys should differ")
	})

	t.Run("public and private keys are different", func(t *testing.T) {
		pub, priv, err := crypto.GenerateKeypair()
		require.NoError(t, err)
		assert.NotEqual(t, pub, priv, "public and private keys should differ")
	})
}

func TestKeyExchange(t *testing.T) {
	crypto := NewCryptoService()

	t.Run("both sides compute same shared secret", func(t *testing.T) {
		// Server keypair
		serverPub, serverPriv, err := crypto.GenerateKeypair()
		require.NoError(t, err)

		// Mobile keypair
		mobilePub, mobilePriv, err := crypto.GenerateKeypair()
		require.NoError(t, err)

		// Server computes shared secret using its private key + mobile's public key
		serverShared, err := crypto.ComputeSharedSecret(serverPriv, mobilePub)
		require.NoError(t, err)

		// Mobile computes shared secret using its private key + server's public key
		mobileShared, err := crypto.ComputeSharedSecret(mobilePriv, serverPub)
		require.NoError(t, err)

		assert.Equal(t, serverShared, mobileShared, "shared secrets must match")
		assert.Len(t, serverShared, 32, "shared secret must be 32 bytes")
	})

	t.Run("invalid private key length", func(t *testing.T) {
		pub, _, err := crypto.GenerateKeypair()
		require.NoError(t, err)

		_, err = crypto.ComputeSharedSecret([]byte("short"), pub)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "private key must be 32 bytes")
	})

	t.Run("invalid public key length", func(t *testing.T) {
		_, priv, err := crypto.GenerateKeypair()
		require.NoError(t, err)

		_, err = crypto.ComputeSharedSecret(priv, []byte("short"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "peer public key must be 32 bytes")
	})
}

func TestEncryptDecrypt(t *testing.T) {
	crypto := NewCryptoService()

	t.Run("round trip succeeds", func(t *testing.T) {
		// Generate shared secret via key exchange
		_, serverPriv, err := crypto.GenerateKeypair()
		require.NoError(t, err)
		mobilePub, _, err := crypto.GenerateKeypair()
		require.NoError(t, err)
		// Use a deterministic shared secret for test simplicity
		shared, err := crypto.ComputeSharedSecret(serverPriv, mobilePub)
		require.NoError(t, err)

		plaintext := []byte("hello, encrypted world!")
		var counter uint64 = 42

		encrypted, err := crypto.Encrypt(shared, plaintext, counter)
		require.NoError(t, err)
		assert.NotEqual(t, plaintext, encrypted[:len(plaintext)], "ciphertext should differ from plaintext")

		decrypted, gotCounter, err := crypto.Decrypt(shared, encrypted)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
		assert.Equal(t, counter, gotCounter)
	})

	t.Run("empty plaintext", func(t *testing.T) {
		pub, priv, err := crypto.GenerateKeypair()
		require.NoError(t, err)
		shared, err := crypto.ComputeSharedSecret(priv, pub)
		require.NoError(t, err)

		encrypted, err := crypto.Encrypt(shared, []byte{}, 0)
		require.NoError(t, err)

		decrypted, counter, err := crypto.Decrypt(shared, encrypted)
		require.NoError(t, err)
		assert.Empty(t, decrypted)
		assert.Equal(t, uint64(0), counter)
	})

	t.Run("counter preserved", func(t *testing.T) {
		pub, priv, err := crypto.GenerateKeypair()
		require.NoError(t, err)
		shared, err := crypto.ComputeSharedSecret(priv, pub)
		require.NoError(t, err)

		var maxCounter uint64 = 18446744073709551615 // max uint64
		encrypted, err := crypto.Encrypt(shared, []byte("test"), maxCounter)
		require.NoError(t, err)

		_, gotCounter, err := crypto.Decrypt(shared, encrypted)
		require.NoError(t, err)
		assert.Equal(t, maxCounter, gotCounter)
	})
}

func TestEncryptDecryptDifferentKeys(t *testing.T) {
	crypto := NewCryptoService()

	// Two different shared secrets
	pub1, priv1, err := crypto.GenerateKeypair()
	require.NoError(t, err)
	shared1, err := crypto.ComputeSharedSecret(priv1, pub1)
	require.NoError(t, err)

	pub2, priv2, err := crypto.GenerateKeypair()
	require.NoError(t, err)
	shared2, err := crypto.ComputeSharedSecret(priv2, pub2)
	require.NoError(t, err)

	plaintext := []byte("secret message")
	encrypted, err := crypto.Encrypt(shared1, plaintext, 1)
	require.NoError(t, err)

	// Attempt to decrypt with wrong key
	_, _, err = crypto.Decrypt(shared2, encrypted)
	require.Error(t, err, "decryption with wrong key should fail")
	assert.Contains(t, err.Error(), "decrypt")
}

func TestDecryptTamperedData(t *testing.T) {
	crypto := NewCryptoService()

	pub, priv, err := crypto.GenerateKeypair()
	require.NoError(t, err)
	shared, err := crypto.ComputeSharedSecret(priv, pub)
	require.NoError(t, err)

	plaintext := []byte("important data")
	encrypted, err := crypto.Encrypt(shared, plaintext, 1)
	require.NoError(t, err)

	t.Run("flipped byte in ciphertext", func(t *testing.T) {
		tampered := make([]byte, len(encrypted))
		copy(tampered, encrypted)
		// Flip a byte in the ciphertext portion (after the 24-byte nonce)
		tampered[25] ^= 0xff

		_, _, err := crypto.Decrypt(shared, tampered)
		require.Error(t, err, "decryption of tampered data should fail")
	})

	t.Run("truncated data", func(t *testing.T) {
		_, _, err := crypto.Decrypt(shared, encrypted[:10])
		require.Error(t, err, "decryption of truncated data should fail")
		assert.Contains(t, err.Error(), "ciphertext too short")
	})

	t.Run("empty data", func(t *testing.T) {
		_, _, err := crypto.Decrypt(shared, []byte{})
		require.Error(t, err, "decryption of empty data should fail")
	})
}

package bridge

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
)

// KeyPair holds an X25519 key pair for Diffie-Hellman key exchange.
type KeyPair struct {
	PublicKey  []byte // 32 bytes
	PrivateKey []byte // 32 bytes
}

// GenerateKeyPair creates a new X25519 key pair from a random private key.
func GenerateKeyPair() (*KeyPair, error) {
	privateKey := make([]byte, curve25519.ScalarSize)
	if _, err := rand.Read(privateKey); err != nil {
		return nil, fmt.Errorf("generate random private key: %w", err)
	}

	publicKey, err := curve25519.X25519(privateKey, curve25519.Basepoint)
	if err != nil {
		return nil, fmt.Errorf("compute public key: %w", err)
	}

	return &KeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}, nil
}

// ComputeSharedSecret performs X25519 Diffie-Hellman to derive a shared secret.
func ComputeSharedSecret(privateKey, peerPublicKey []byte) ([]byte, error) {
	if len(privateKey) != curve25519.ScalarSize {
		return nil, fmt.Errorf("private key must be %d bytes, got %d", curve25519.ScalarSize, len(privateKey))
	}
	if len(peerPublicKey) != curve25519.PointSize {
		return nil, fmt.Errorf("peer public key must be %d bytes, got %d", curve25519.PointSize, len(peerPublicKey))
	}

	shared, err := curve25519.X25519(privateKey, peerPublicKey)
	if err != nil {
		return nil, fmt.Errorf("compute shared secret: %w", err)
	}

	return shared, nil
}

// Encrypt encrypts plaintext using XChaCha20-Poly1305 with the given shared secret and counter.
// Nonce format: 16 random bytes + 8 bytes counter (little-endian) = 24 bytes.
// Output format: nonce(24) || ciphertext || tag(16).
func Encrypt(plaintext, sharedSecret []byte, counter uint64) ([]byte, error) {
	if len(sharedSecret) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("shared secret must be %d bytes, got %d", chacha20poly1305.KeySize, len(sharedSecret))
	}

	aead, err := chacha20poly1305.NewX(sharedSecret)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	nonce := make([]byte, chacha20poly1305.NonceSizeX) // 24 bytes
	if _, err := rand.Read(nonce[:16]); err != nil {
		return nil, fmt.Errorf("generate random nonce: %w", err)
	}
	binary.LittleEndian.PutUint64(nonce[16:], counter)

	sealed := aead.Seal(nonce, nonce, plaintext, nil)
	return sealed, nil
}

// Decrypt decrypts data encrypted with Encrypt.
// Expects input format: nonce(24) || ciphertext || tag(16).
func Decrypt(sealed, sharedSecret []byte) ([]byte, error) {
	if len(sharedSecret) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("shared secret must be %d bytes, got %d", chacha20poly1305.KeySize, len(sharedSecret))
	}

	aead, err := chacha20poly1305.NewX(sharedSecret)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	if len(sealed) < chacha20poly1305.NonceSizeX+aead.Overhead() {
		return nil, fmt.Errorf("ciphertext too short: %d bytes", len(sealed))
	}

	nonce := sealed[:chacha20poly1305.NonceSizeX]
	ciphertext := sealed[chacha20poly1305.NonceSizeX:]

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil
}

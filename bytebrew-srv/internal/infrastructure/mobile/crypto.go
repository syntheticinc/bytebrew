package mobile

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
)

const (
	// x25519KeySize is the size of X25519 keys in bytes.
	x25519KeySize = 32
	// randomNoncePrefix is the number of random bytes in the XChaCha20-Poly1305 nonce.
	randomNoncePrefix = 16
)

// CryptoService provides X25519 key exchange and XChaCha20-Poly1305 encryption.
type CryptoService struct{}

// NewCryptoService creates a new CryptoService.
func NewCryptoService() *CryptoService {
	return &CryptoService{}
}

// GenerateKeypair generates an X25519 keypair.
// Returns (publicKey, privateKey) each 32 bytes.
func (s *CryptoService) GenerateKeypair() (publicKey, privateKey []byte, err error) {
	privateKey = make([]byte, x25519KeySize)
	if _, err := rand.Read(privateKey); err != nil {
		return nil, nil, fmt.Errorf("generate private key: %w", err)
	}

	publicKey, err = curve25519.X25519(privateKey, curve25519.Basepoint)
	if err != nil {
		return nil, nil, fmt.Errorf("derive public key: %w", err)
	}

	return publicKey, privateKey, nil
}

// ComputeSharedSecret performs X25519 ECDH key exchange.
// Returns a 32-byte shared secret.
func (s *CryptoService) ComputeSharedSecret(privateKey, peerPublicKey []byte) ([]byte, error) {
	if len(privateKey) != x25519KeySize {
		return nil, fmt.Errorf("private key must be %d bytes, got %d", x25519KeySize, len(privateKey))
	}
	if len(peerPublicKey) != x25519KeySize {
		return nil, fmt.Errorf("peer public key must be %d bytes, got %d", x25519KeySize, len(peerPublicKey))
	}

	shared, err := curve25519.X25519(privateKey, peerPublicKey)
	if err != nil {
		return nil, fmt.Errorf("compute shared secret: %w", err)
	}

	return shared, nil
}

// Encrypt encrypts plaintext using XChaCha20-Poly1305.
// Nonce layout: 16 random bytes + 8 bytes counter (little-endian).
// Output format: nonce(24) || ciphertext+tag.
func (s *CryptoService) Encrypt(sharedSecret, plaintext []byte, counter uint64) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(sharedSecret)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	// Build nonce: 16 random bytes + 8 bytes counter
	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := rand.Read(nonce[:randomNoncePrefix]); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	binary.LittleEndian.PutUint64(nonce[randomNoncePrefix:], counter)

	// Encrypt: nonce || ciphertext+tag
	ciphertext := aead.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, nil
}

// Decrypt decrypts data encrypted with XChaCha20-Poly1305.
// Input format: nonce(24) || ciphertext+tag.
// Returns plaintext and the counter extracted from the nonce.
func (s *CryptoService) Decrypt(sharedSecret, data []byte) (plaintext []byte, counter uint64, err error) {
	aead, err := chacha20poly1305.NewX(sharedSecret)
	if err != nil {
		return nil, 0, fmt.Errorf("create cipher: %w", err)
	}

	nonceSize := chacha20poly1305.NonceSizeX
	if len(data) < nonceSize+aead.Overhead() {
		return nil, 0, fmt.Errorf("ciphertext too short: need at least %d bytes, got %d", nonceSize+aead.Overhead(), len(data))
	}

	nonce := data[:nonceSize]
	ciphertext := data[nonceSize:]

	plaintext, err = aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("decrypt: %w", err)
	}

	counter = binary.LittleEndian.Uint64(nonce[randomNoncePrefix:])

	return plaintext, counter, nil
}

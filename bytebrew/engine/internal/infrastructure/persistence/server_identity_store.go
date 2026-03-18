package persistence

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/models"
	"golang.org/x/crypto/curve25519"
	"gorm.io/gorm"
)

// ServerIdentity holds the stable server identity (ID + X25519 keypair).
type ServerIdentity struct {
	ID         string
	PublicKey  []byte
	PrivateKey []byte
}

// ServerIdentityStore manages persistent server identity in PostgreSQL (GORM).
type ServerIdentityStore struct {
	db *gorm.DB
}

// NewServerIdentityStore creates a new identity store.
func NewServerIdentityStore(db *gorm.DB) *ServerIdentityStore {
	slog.Info("server identity store initialized (PostgreSQL)")
	return &ServerIdentityStore{db: db}
}

// GetOrCreateIdentity loads the existing identity or generates a new one.
func (s *ServerIdentityStore) GetOrCreateIdentity() (*ServerIdentity, error) {
	identity, err := s.loadIdentity()
	if err != nil {
		return nil, fmt.Errorf("load identity: %w", err)
	}
	if identity != nil {
		slog.Info("loaded existing server identity", "server_id", identity.ID)
		return identity, nil
	}

	identity, err = s.generateAndSave()
	if err != nil {
		return nil, fmt.Errorf("generate identity: %w", err)
	}

	slog.Info("generated new server identity", "server_id", identity.ID)
	return identity, nil
}

func (s *ServerIdentityStore) loadIdentity() (*ServerIdentity, error) {
	serverID, err := s.getConfig("server_id")
	if err != nil {
		return nil, fmt.Errorf("get server_id: %w", err)
	}
	if serverID == "" {
		return nil, nil
	}

	pubKeyHex, err := s.getConfig("server_public_key")
	if err != nil {
		return nil, fmt.Errorf("get server_public_key: %w", err)
	}

	privKeyHex, err := s.getConfig("server_private_key")
	if err != nil {
		return nil, fmt.Errorf("get server_private_key: %w", err)
	}

	if pubKeyHex == "" || privKeyHex == "" {
		return nil, nil
	}

	pubKey, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}

	privKey, err := hex.DecodeString(privKeyHex)
	if err != nil {
		return nil, fmt.Errorf("decode private key: %w", err)
	}

	return &ServerIdentity{
		ID:         serverID,
		PublicKey:  pubKey,
		PrivateKey: privKey,
	}, nil
}

func (s *ServerIdentityStore) generateAndSave() (*ServerIdentity, error) {
	serverID := uuid.New().String()

	privateKey := make([]byte, curve25519.ScalarSize)
	if _, err := rand.Read(privateKey); err != nil {
		return nil, fmt.Errorf("generate private key: %w", err)
	}

	publicKey, err := curve25519.X25519(privateKey, curve25519.Basepoint)
	if err != nil {
		return nil, fmt.Errorf("compute public key: %w", err)
	}

	// Use a transaction to save all config entries atomically
	err = s.db.Transaction(func(tx *gorm.DB) error {
		for _, kv := range []struct{ key, value string }{
			{"server_id", serverID},
			{"server_public_key", hex.EncodeToString(publicKey)},
			{"server_private_key", hex.EncodeToString(privateKey)},
		} {
			m := models.RuntimeConfigKV{Key: kv.key, Value: kv.value}
			if upsertErr := tx.Save(&m).Error; upsertErr != nil {
				return fmt.Errorf("save config %s: %w", kv.key, upsertErr)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("commit identity: %w", err)
	}

	return &ServerIdentity{
		ID:         serverID,
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}, nil
}

func (s *ServerIdentityStore) getConfig(key string) (string, error) {
	var m models.RuntimeConfigKV
	err := s.db.Where("key = ?", key).First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("query config %s: %w", key, err)
	}
	return m.Value, nil
}

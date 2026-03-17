package persistence

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"golang.org/x/crypto/curve25519"
)

const createConfigTableSQL = `
CREATE TABLE IF NOT EXISTS config (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
`

// ServerIdentity holds the stable server identity (ID + X25519 keypair)
type ServerIdentity struct {
	ID         string
	PublicKey  []byte
	PrivateKey []byte
}

// ServerIdentityStore manages persistent server identity in SQLite
type ServerIdentityStore struct {
	db *sql.DB
}

// NewServerIdentityStore creates a new identity store using the shared DB.
func NewServerIdentityStore(db *sql.DB) (*ServerIdentityStore, error) {
	if _, err := db.Exec(createConfigTableSQL); err != nil {
		return nil, fmt.Errorf("create config table: %w", err)
	}

	slog.Info("server identity store initialized")
	return &ServerIdentityStore{db: db}, nil
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

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, kv := range []struct{ key, value string }{
		{"server_id", serverID},
		{"server_public_key", hex.EncodeToString(publicKey)},
		{"server_private_key", hex.EncodeToString(privateKey)},
	} {
		if _, err := tx.Exec(`INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)`, kv.key, kv.value); err != nil {
			return nil, fmt.Errorf("save config %s: %w", kv.key, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit identity: %w", err)
	}

	return &ServerIdentity{
		ID:         serverID,
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}, nil
}

func (s *ServerIdentityStore) getConfig(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM config WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("query config %s: %w", key, err)
	}
	return value, nil
}

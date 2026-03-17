package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
)

const createDevicesTableSQL = `
CREATE TABLE IF NOT EXISTS paired_devices (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	device_token TEXT NOT NULL UNIQUE,
	public_key BLOB,
	shared_secret BLOB,
	paired_at INTEGER NOT NULL,
	last_seen_at INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_paired_devices_token ON paired_devices(device_token);
`

// SQLiteDeviceStore implements device persistence using SQLite
type SQLiteDeviceStore struct {
	db *sql.DB
}

// NewSQLiteDeviceStore creates a new device store using the shared DB.
// The caller is responsible for calling NewWorkDB first and passing the *sql.DB.
func NewSQLiteDeviceStore(db *sql.DB) (*SQLiteDeviceStore, error) {
	if _, err := db.Exec(createDevicesTableSQL); err != nil {
		return nil, fmt.Errorf("create paired_devices table: %w", err)
	}

	slog.Info("SQLite device store initialized")
	return &SQLiteDeviceStore{db: db}, nil
}

// Add persists a new paired device
func (s *SQLiteDeviceStore) Add(ctx context.Context, device *domain.MobileDevice) error {
	if err := device.Validate(); err != nil {
		return fmt.Errorf("validate device: %w", err)
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO paired_devices (id, name, device_token, public_key, shared_secret, paired_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, device.ID, device.Name, device.DeviceToken,
		device.PublicKey, device.SharedSecret,
		device.PairedAt.Unix(), device.LastSeenAt.Unix())

	if err != nil {
		return fmt.Errorf("insert device: %w", err)
	}

	slog.InfoContext(ctx, "device added", "device_id", device.ID, "name", device.Name)
	return nil
}

// GetByID retrieves a device by its ID
func (s *SQLiteDeviceStore) GetByID(ctx context.Context, id string) (*domain.MobileDevice, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, device_token, public_key, shared_secret, paired_at, last_seen_at
		FROM paired_devices WHERE id = ?
	`, id)

	return s.scanDevice(row)
}

// GetByToken retrieves a device by its device token
func (s *SQLiteDeviceStore) GetByToken(ctx context.Context, deviceToken string) (*domain.MobileDevice, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, device_token, public_key, shared_secret, paired_at, last_seen_at
		FROM paired_devices WHERE device_token = ?
	`, deviceToken)

	return s.scanDevice(row)
}

// List returns all paired devices
func (s *SQLiteDeviceStore) List(ctx context.Context) ([]*domain.MobileDevice, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, device_token, public_key, shared_secret, paired_at, last_seen_at
		FROM paired_devices
		ORDER BY paired_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query devices: %w", err)
	}
	defer rows.Close()

	return s.scanDevices(rows)
}

// Remove deletes a device by its ID
func (s *SQLiteDeviceStore) Remove(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM paired_devices WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("device not found: %s", id)
	}

	slog.InfoContext(ctx, "device removed", "device_id", id)
	return nil
}

// UpdateLastSeen updates the last_seen_at timestamp for a device
func (s *SQLiteDeviceStore) UpdateLastSeen(ctx context.Context, id string, lastSeen time.Time) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE paired_devices SET last_seen_at = ? WHERE id = ?
	`, lastSeen.Unix(), id)
	if err != nil {
		return fmt.Errorf("update last_seen: %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("device not found: %s", id)
	}

	return nil
}

// Close is a no-op because the shared DB is owned by the caller
func (s *SQLiteDeviceStore) Close() error {
	return nil
}

func (s *SQLiteDeviceStore) scanDevice(row *sql.Row) (*domain.MobileDevice, error) {
	var (
		id           string
		name         string
		deviceToken  string
		publicKey    []byte
		sharedSecret []byte
		pairedAt     int64
		lastSeenAt   int64
	)

	err := row.Scan(&id, &name, &deviceToken, &publicKey, &sharedSecret, &pairedAt, &lastSeenAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan device: %w", err)
	}

	return &domain.MobileDevice{
		ID:           id,
		Name:         name,
		DeviceToken:  deviceToken,
		PublicKey:    publicKey,
		SharedSecret: sharedSecret,
		PairedAt:     time.Unix(pairedAt, 0),
		LastSeenAt:   time.Unix(lastSeenAt, 0),
	}, nil
}

func (s *SQLiteDeviceStore) scanDevices(rows *sql.Rows) ([]*domain.MobileDevice, error) {
	var devices []*domain.MobileDevice

	for rows.Next() {
		var (
			id           string
			name         string
			deviceToken  string
			publicKey    []byte
			sharedSecret []byte
			pairedAt     int64
			lastSeenAt   int64
		)

		if err := rows.Scan(&id, &name, &deviceToken, &publicKey, &sharedSecret, &pairedAt, &lastSeenAt); err != nil {
			return nil, fmt.Errorf("scan device row: %w", err)
		}

		devices = append(devices, &domain.MobileDevice{
			ID:           id,
			Name:         name,
			DeviceToken:  deviceToken,
			PublicKey:    publicKey,
			SharedSecret: sharedSecret,
			PairedAt:     time.Unix(pairedAt, 0),
			LastSeenAt:   time.Unix(lastSeenAt, 0),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate device rows: %w", err)
	}

	return devices, nil
}

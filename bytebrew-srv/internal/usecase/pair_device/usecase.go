package pair_device

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"

	"github.com/google/uuid"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/errors"
)

// deviceTokenBytes is the number of random bytes for the device auth token (256-bit)
const deviceTokenBytes = 32

// PairingTokenStore defines interface for pairing token persistence
type PairingTokenStore interface {
	SaveToken(ctx context.Context, token *domain.PairingToken) error
	GetToken(ctx context.Context, tokenOrCode string) (*domain.PairingToken, error)
	UseToken(ctx context.Context, tokenOrCode string) (*domain.PairingToken, error)
	DeleteToken(ctx context.Context, token string) error
}

// DeviceStore defines interface for mobile device persistence
type DeviceStore interface {
	SaveDevice(ctx context.Context, device *domain.MobileDevice) error
	GetDevice(ctx context.Context, deviceID string) (*domain.MobileDevice, error)
	ListDevices(ctx context.Context) ([]*domain.MobileDevice, error)
	DeleteDevice(ctx context.Context, deviceID string) error
}

// CryptoProvider defines interface for cryptographic operations (X25519 + XChaCha20-Poly1305)
type CryptoProvider interface {
	GenerateKeypair() (publicKey, privateKey []byte, err error)
	ComputeSharedSecret(privateKey, peerPublicKey []byte) ([]byte, error)
}

// Usecase handles mobile device pairing
type Usecase struct {
	tokenStore  PairingTokenStore
	deviceStore DeviceStore
	crypto      CryptoProvider
	serverName  string
	serverID    string
}

// New creates a new Pair Device use case.
// crypto is optional: if nil, pairing works without E2E encryption (LAN mode).
func New(tokenStore PairingTokenStore, deviceStore DeviceStore, crypto CryptoProvider, serverName, serverID string) (*Usecase, error) {
	if tokenStore == nil {
		return nil, errors.New(errors.CodeInvalidInput, "pairing token store is required")
	}
	if deviceStore == nil {
		return nil, errors.New(errors.CodeInvalidInput, "device store is required")
	}
	if serverName == "" {
		return nil, errors.New(errors.CodeInvalidInput, "server name is required")
	}
	if serverID == "" {
		return nil, errors.New(errors.CodeInvalidInput, "server id is required")
	}

	return &Usecase{
		tokenStore:  tokenStore,
		deviceStore: deviceStore,
		crypto:      crypto,
		serverName:  serverName,
		serverID:    serverID,
	}, nil
}

// GeneratePairingToken creates a new pairing token and saves it to the store.
// If crypto is configured, generates an X25519 keypair and stores it in the token.
func (u *Usecase) GeneratePairingToken(ctx context.Context, serverID string) (*domain.PairingToken, error) {
	slog.InfoContext(ctx, "generating pairing token", "server_id", serverID)

	if serverID == "" {
		return nil, errors.New(errors.CodeInvalidInput, "server_id is required")
	}

	token, err := domain.NewPairingToken(serverID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create pairing token", "error", err)
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to create pairing token")
	}

	// Generate X25519 keypair for ECDH key exchange (if crypto is available)
	if u.crypto != nil {
		pubKey, privKey, err := u.crypto.GenerateKeypair()
		if err != nil {
			slog.ErrorContext(ctx, "failed to generate keypair", "error", err)
			return nil, errors.Wrap(err, errors.CodeInternal, "failed to generate keypair")
		}
		token.ServerPublicKey = pubKey
		token.ServerPrivateKey = privKey
	}

	if err := u.tokenStore.SaveToken(ctx, token); err != nil {
		slog.ErrorContext(ctx, "failed to save pairing token", "error", err)
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to save pairing token")
	}

	slog.InfoContext(ctx, "pairing token generated successfully",
		"short_code", token.ShortCode,
		"has_crypto", u.crypto != nil,
	)

	return token, nil
}

// PairOutput represents the result of a successful pairing
type PairOutput struct {
	DeviceID        string
	DeviceToken     string
	ServerName      string
	ServerPublicKey []byte // X25519 public key (empty if crypto not configured)
	Token           string // Full pairing token that was consumed (for waiter notification)
}

// PairInput contains the input parameters for device pairing
type PairInput struct {
	TokenOrCode     string
	DeviceName      string
	MobilePublicKey []byte // X25519 public key from mobile (optional for backward compat)
}

// Pair validates the pairing token, performs ECDH key exchange if keys are present,
// creates a new device, and returns credentials.
func (u *Usecase) Pair(ctx context.Context, input PairInput) (*PairOutput, error) {
	slog.InfoContext(ctx, "pairing device", "device_name", input.DeviceName)

	if input.TokenOrCode == "" {
		return nil, errors.New(errors.CodeInvalidInput, "token or code is required")
	}
	if input.DeviceName == "" {
		return nil, errors.New(errors.CodeInvalidInput, "device name is required")
	}

	// Atomically find, validate, and mark the pairing token as used
	pairingToken, err := u.tokenStore.UseToken(ctx, input.TokenOrCode)
	if err != nil {
		slog.ErrorContext(ctx, "failed to use pairing token", "error", err)
		return nil, err
	}

	// Generate device credentials
	deviceID := uuid.New().String()
	tokenBuf := make([]byte, deviceTokenBytes)
	if _, err := rand.Read(tokenBuf); err != nil {
		slog.ErrorContext(ctx, "failed to generate device token", "error", err)
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to generate device token")
	}
	deviceToken := hex.EncodeToString(tokenBuf)

	// Create the device
	device, err := domain.NewMobileDevice(deviceID, input.DeviceName, deviceToken)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create mobile device entity", "error", err)
		return nil, errors.Wrap(err, errors.CodeInvalidInput, "invalid device data")
	}

	// Perform ECDH key exchange if mobile sent a public key and server has a private key
	var serverPublicKey []byte
	if len(input.MobilePublicKey) > 0 && len(pairingToken.ServerPrivateKey) > 0 && u.crypto != nil {
		sharedSecret, err := u.crypto.ComputeSharedSecret(pairingToken.ServerPrivateKey, input.MobilePublicKey)
		if err != nil {
			slog.ErrorContext(ctx, "failed to compute shared secret", "error", err)
			return nil, errors.Wrap(err, errors.CodeInternal, "failed to compute shared secret")
		}
		device.PublicKey = input.MobilePublicKey
		device.SharedSecret = sharedSecret
		serverPublicKey = pairingToken.ServerPublicKey

		slog.InfoContext(ctx, "ECDH key exchange completed", "device_id", deviceID)
	}

	if err := u.deviceStore.SaveDevice(ctx, device); err != nil {
		slog.ErrorContext(ctx, "failed to save mobile device", "error", err)
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to save mobile device")
	}

	// Clean up the used token (includes private key)
	if err := u.tokenStore.DeleteToken(ctx, pairingToken.Token); err != nil {
		slog.ErrorContext(ctx, "failed to delete used pairing token", "error", err)
		// Non-critical: device is already paired, just log the error
	}

	slog.InfoContext(ctx, "device paired successfully", "device_id", deviceID, "device_name", input.DeviceName)

	return &PairOutput{
		DeviceID:        deviceID,
		DeviceToken:     deviceToken,
		ServerName:      u.serverName,
		ServerPublicKey: serverPublicKey,
		Token:           pairingToken.Token,
	}, nil
}

// ListDevices returns all paired mobile devices
func (u *Usecase) ListDevices(ctx context.Context) ([]*domain.MobileDevice, error) {
	slog.InfoContext(ctx, "listing paired devices")

	devices, err := u.deviceStore.ListDevices(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list devices", "error", err)
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to list devices")
	}

	return devices, nil
}

// RevokeDevice removes a paired device
func (u *Usecase) RevokeDevice(ctx context.Context, deviceID string) error {
	slog.InfoContext(ctx, "revoking device", "device_id", deviceID)

	if deviceID == "" {
		return errors.New(errors.CodeInvalidInput, "device_id is required")
	}

	// Check that the device exists
	device, err := u.deviceStore.GetDevice(ctx, deviceID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get device", "error", err, "device_id", deviceID)
		return errors.Wrap(err, errors.CodeInternal, "failed to get device")
	}
	if device == nil {
		return errors.New(errors.CodeNotFound, "device not found")
	}

	if err := u.deviceStore.DeleteDevice(ctx, deviceID); err != nil {
		slog.ErrorContext(ctx, "failed to delete device", "error", err, "device_id", deviceID)
		return errors.Wrap(err, errors.CodeInternal, "failed to delete device")
	}

	slog.InfoContext(ctx, "device revoked successfully", "device_id", deviceID)

	return nil
}

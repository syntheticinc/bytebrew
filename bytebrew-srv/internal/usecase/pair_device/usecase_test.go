package pair_device

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/errors"
)

// --- Mocks ---

type mockTokenStore struct {
	tokens     map[string]*domain.PairingToken
	err        error
	useTokenFn func(tokenOrCode string) (*domain.PairingToken, error)
}

func newMockTokenStore() *mockTokenStore {
	return &mockTokenStore{
		tokens: make(map[string]*domain.PairingToken),
	}
}

func (m *mockTokenStore) SaveToken(_ context.Context, token *domain.PairingToken) error {
	if m.err != nil {
		return m.err
	}
	m.tokens[token.Token] = token
	m.tokens[token.ShortCode] = token
	return nil
}

func (m *mockTokenStore) GetToken(_ context.Context, tokenOrCode string) (*domain.PairingToken, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.tokens[tokenOrCode], nil
}

func (m *mockTokenStore) UseToken(_ context.Context, tokenOrCode string) (*domain.PairingToken, error) {
	if m.useTokenFn != nil {
		return m.useTokenFn(tokenOrCode)
	}
	if m.err != nil {
		return nil, m.err
	}
	t := m.tokens[tokenOrCode]
	if t == nil {
		return nil, errors.New(errors.CodeNotFound, "pairing token not found")
	}
	if !t.IsValid() {
		return nil, errors.New(errors.CodeInvalidInput, "pairing token is expired or already used")
	}
	t.MarkUsed()
	return t, nil
}

func (m *mockTokenStore) DeleteToken(_ context.Context, token string) error {
	if m.err != nil {
		return m.err
	}
	delete(m.tokens, token)
	return nil
}

type mockDeviceStore struct {
	devices map[string]*domain.MobileDevice
	err     error
}

func newMockDeviceStore() *mockDeviceStore {
	return &mockDeviceStore{
		devices: make(map[string]*domain.MobileDevice),
	}
}

func (m *mockDeviceStore) SaveDevice(_ context.Context, device *domain.MobileDevice) error {
	if m.err != nil {
		return m.err
	}
	m.devices[device.ID] = device
	return nil
}

func (m *mockDeviceStore) GetDevice(_ context.Context, deviceID string) (*domain.MobileDevice, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.devices[deviceID], nil
}

func (m *mockDeviceStore) ListDevices(_ context.Context) ([]*domain.MobileDevice, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := make([]*domain.MobileDevice, 0, len(m.devices))
	for _, d := range m.devices {
		result = append(result, d)
	}
	return result, nil
}

func (m *mockDeviceStore) DeleteDevice(_ context.Context, deviceID string) error {
	if m.err != nil {
		return m.err
	}
	delete(m.devices, deviceID)
	return nil
}

type mockCryptoProvider struct {
	publicKey  []byte
	privateKey []byte
	shared     []byte
	err        error
}

func newMockCryptoProvider() *mockCryptoProvider {
	return &mockCryptoProvider{
		publicKey:  make([]byte, 32),
		privateKey: make([]byte, 32),
		shared:     make([]byte, 32),
	}
}

func (m *mockCryptoProvider) GenerateKeypair() (publicKey, privateKey []byte, err error) {
	if m.err != nil {
		return nil, nil, m.err
	}
	return m.publicKey, m.privateKey, nil
}

func (m *mockCryptoProvider) ComputeSharedSecret(privateKey, peerPublicKey []byte) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.shared, nil
}

// --- Constructor Tests ---

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		tokenStore  PairingTokenStore
		deviceStore DeviceStore
		crypto      CryptoProvider
		serverName  string
		serverID    string
		wantErr     bool
	}{
		{
			name:        "valid inputs with crypto",
			tokenStore:  newMockTokenStore(),
			deviceStore: newMockDeviceStore(),
			crypto:      newMockCryptoProvider(),
			serverName:  "test-server",
			serverID:    "server-1",
			wantErr:     false,
		},
		{
			name:        "valid inputs without crypto",
			tokenStore:  newMockTokenStore(),
			deviceStore: newMockDeviceStore(),
			crypto:      nil,
			serverName:  "test-server",
			serverID:    "server-1",
			wantErr:     false,
		},
		{
			name:        "nil token store",
			tokenStore:  nil,
			deviceStore: newMockDeviceStore(),
			crypto:      nil,
			serverName:  "test-server",
			serverID:    "server-1",
			wantErr:     true,
		},
		{
			name:        "nil device store",
			tokenStore:  newMockTokenStore(),
			deviceStore: nil,
			crypto:      nil,
			serverName:  "test-server",
			serverID:    "server-1",
			wantErr:     true,
		},
		{
			name:        "empty server name",
			tokenStore:  newMockTokenStore(),
			deviceStore: newMockDeviceStore(),
			crypto:      nil,
			serverName:  "",
			serverID:    "server-1",
			wantErr:     true,
		},
		{
			name:        "empty server id",
			tokenStore:  newMockTokenStore(),
			deviceStore: newMockDeviceStore(),
			crypto:      nil,
			serverName:  "test-server",
			serverID:    "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, err := New(tt.tokenStore, tt.deviceStore, tt.crypto, tt.serverName, tt.serverID)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, uc)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, uc)
		})
	}
}

// --- GeneratePairingToken Tests ---

func TestGeneratePairingToken(t *testing.T) {
	tests := []struct {
		name     string
		serverID string
		storeErr error
		wantErr  bool
	}{
		{
			name:     "success",
			serverID: "server-1",
			wantErr:  false,
		},
		{
			name:     "empty server id",
			serverID: "",
			wantErr:  true,
		},
		{
			name:     "store save error",
			serverID: "server-1",
			storeErr: assert.AnError,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenStore := newMockTokenStore()
			tokenStore.err = tt.storeErr
			uc, err := New(tokenStore, newMockDeviceStore(), newMockCryptoProvider(), "test-server", "server-1")
			require.NoError(t, err)

			token, err := uc.GeneratePairingToken(context.Background(), tt.serverID)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, token)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, token.Token)
			assert.NotEmpty(t, token.ShortCode)
			assert.Len(t, token.ShortCode, 6)
			assert.Equal(t, tt.serverID, token.ServerID)
			assert.False(t, token.Used)
			assert.True(t, token.IsValid())
		})
	}
}

func TestGeneratePairingToken_WithCrypto(t *testing.T) {
	ctx := context.Background()
	crypto := newMockCryptoProvider()
	crypto.publicKey = []byte("server-public-key-32-bytes-xxxxx")
	crypto.privateKey = []byte("server-private-key-32-bytes-xxxx")

	uc, err := New(newMockTokenStore(), newMockDeviceStore(), crypto, "test-server", "server-1")
	require.NoError(t, err)

	token, err := uc.GeneratePairingToken(ctx, "server-1")
	require.NoError(t, err)
	assert.Equal(t, crypto.publicKey, token.ServerPublicKey)
	assert.Equal(t, crypto.privateKey, token.ServerPrivateKey)
}

func TestGeneratePairingToken_WithoutCrypto(t *testing.T) {
	ctx := context.Background()

	uc, err := New(newMockTokenStore(), newMockDeviceStore(), nil, "test-server", "server-1")
	require.NoError(t, err)

	token, err := uc.GeneratePairingToken(ctx, "server-1")
	require.NoError(t, err)
	assert.Nil(t, token.ServerPublicKey)
	assert.Nil(t, token.ServerPrivateKey)
}

// --- Pair Tests ---

func TestPair(t *testing.T) {
	ctx := context.Background()

	t.Run("success with token", func(t *testing.T) {
		tokenStore := newMockTokenStore()
		deviceStore := newMockDeviceStore()
		uc, err := New(tokenStore, deviceStore, nil, "My Server", "server-1")
		require.NoError(t, err)

		// Generate a token first
		token, err := uc.GeneratePairingToken(ctx, "server-1")
		require.NoError(t, err)

		// Pair with the full token
		output, err := uc.Pair(ctx, PairInput{
			TokenOrCode: token.Token,
			DeviceName:  "iPhone 15",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, output.DeviceID)
		assert.NotEmpty(t, output.DeviceToken)
		assert.Len(t, output.DeviceToken, 64) // 32 bytes hex
		assert.Equal(t, "My Server", output.ServerName)

		// Device should be saved
		assert.Len(t, deviceStore.devices, 1)
	})

	t.Run("success with short code", func(t *testing.T) {
		tokenStore := newMockTokenStore()
		deviceStore := newMockDeviceStore()
		uc, err := New(tokenStore, deviceStore, nil, "My Server", "server-1")
		require.NoError(t, err)

		token, err := uc.GeneratePairingToken(ctx, "server-1")
		require.NoError(t, err)

		output, err := uc.Pair(ctx, PairInput{
			TokenOrCode: token.ShortCode,
			DeviceName:  "Pixel 8",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, output.DeviceID)
		assert.Equal(t, "My Server", output.ServerName)
	})

	t.Run("empty token", func(t *testing.T) {
		uc, err := New(newMockTokenStore(), newMockDeviceStore(), nil, "s", "s")
		require.NoError(t, err)

		_, err = uc.Pair(ctx, PairInput{TokenOrCode: "", DeviceName: "iPhone"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, errors.CodeInvalidInput))
	})

	t.Run("empty device name", func(t *testing.T) {
		uc, err := New(newMockTokenStore(), newMockDeviceStore(), nil, "s", "s")
		require.NoError(t, err)

		_, err = uc.Pair(ctx, PairInput{TokenOrCode: "some-token", DeviceName: ""})
		require.Error(t, err)
		assert.True(t, errors.Is(err, errors.CodeInvalidInput))
	})

	t.Run("token not found", func(t *testing.T) {
		uc, err := New(newMockTokenStore(), newMockDeviceStore(), nil, "s", "s")
		require.NoError(t, err)

		_, err = uc.Pair(ctx, PairInput{TokenOrCode: "nonexistent-token", DeviceName: "iPhone"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, errors.CodeNotFound))
	})

	t.Run("expired token", func(t *testing.T) {
		tokenStore := newMockTokenStore()
		uc, err := New(tokenStore, newMockDeviceStore(), nil, "s", "s")
		require.NoError(t, err)

		// Create an already-used token
		token, err := domain.NewPairingToken("server-1")
		require.NoError(t, err)
		token.MarkUsed()
		err = tokenStore.SaveToken(ctx, token)
		require.NoError(t, err)

		_, err = uc.Pair(ctx, PairInput{TokenOrCode: token.Token, DeviceName: "iPhone"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, errors.CodeInvalidInput))
	})
}

func TestPair_WithECDH(t *testing.T) {
	ctx := context.Background()

	crypto := newMockCryptoProvider()
	crypto.publicKey = []byte("server-public-key-32-bytes-xxxxx")
	crypto.privateKey = []byte("server-private-key-32-bytes-xxxx")
	crypto.shared = []byte("shared-secret-32-bytes-xxxxxxxxx")

	tokenStore := newMockTokenStore()
	deviceStore := newMockDeviceStore()
	uc, err := New(tokenStore, deviceStore, crypto, "My Server", "server-1")
	require.NoError(t, err)

	// Generate token (will have keypair)
	token, err := uc.GeneratePairingToken(ctx, "server-1")
	require.NoError(t, err)
	assert.Equal(t, crypto.publicKey, token.ServerPublicKey)

	// Pair with mobile public key
	mobilePublicKey := []byte("mobile-public-key-32-bytes-xxxxx")
	output, err := uc.Pair(ctx, PairInput{
		TokenOrCode:     token.Token,
		DeviceName:      "iPhone 15",
		MobilePublicKey: mobilePublicKey,
	})
	require.NoError(t, err)
	assert.Equal(t, crypto.publicKey, output.ServerPublicKey)

	// Verify device has crypto fields
	for _, device := range deviceStore.devices {
		assert.Equal(t, mobilePublicKey, device.PublicKey)
		assert.Equal(t, crypto.shared, device.SharedSecret)
	}
}

func TestPair_WithoutMobilePublicKey(t *testing.T) {
	ctx := context.Background()

	crypto := newMockCryptoProvider()
	crypto.publicKey = []byte("server-public-key-32-bytes-xxxxx")
	crypto.privateKey = []byte("server-private-key-32-bytes-xxxx")

	tokenStore := newMockTokenStore()
	deviceStore := newMockDeviceStore()
	uc, err := New(tokenStore, deviceStore, crypto, "My Server", "server-1")
	require.NoError(t, err)

	token, err := uc.GeneratePairingToken(ctx, "server-1")
	require.NoError(t, err)

	// Pair WITHOUT mobile public key (backward compat)
	output, err := uc.Pair(ctx, PairInput{
		TokenOrCode: token.Token,
		DeviceName:  "Old Device",
	})
	require.NoError(t, err)
	assert.Nil(t, output.ServerPublicKey, "no server public key when mobile doesn't send one")

	// Device should NOT have crypto fields
	for _, device := range deviceStore.devices {
		assert.Nil(t, device.PublicKey)
		assert.Nil(t, device.SharedSecret)
	}
}

// --- ListDevices Tests ---

func TestListDevices(t *testing.T) {
	ctx := context.Background()

	t.Run("empty list", func(t *testing.T) {
		uc, err := New(newMockTokenStore(), newMockDeviceStore(), nil, "s", "s")
		require.NoError(t, err)

		devices, err := uc.ListDevices(ctx)
		require.NoError(t, err)
		assert.Empty(t, devices)
	})

	t.Run("with devices", func(t *testing.T) {
		deviceStore := newMockDeviceStore()
		dev, _ := domain.NewMobileDevice("d1", "iPhone", "token1")
		deviceStore.devices["d1"] = dev

		uc, err := New(newMockTokenStore(), deviceStore, nil, "s", "s")
		require.NoError(t, err)

		devices, err := uc.ListDevices(ctx)
		require.NoError(t, err)
		assert.Len(t, devices, 1)
		assert.Equal(t, "d1", devices[0].ID)
	})

	t.Run("store error", func(t *testing.T) {
		deviceStore := newMockDeviceStore()
		deviceStore.err = assert.AnError

		uc, err := New(newMockTokenStore(), deviceStore, nil, "s", "s")
		require.NoError(t, err)

		_, err = uc.ListDevices(ctx)
		require.Error(t, err)
	})
}

// --- RevokeDevice Tests ---

func TestRevokeDevice(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		deviceStore := newMockDeviceStore()
		dev, _ := domain.NewMobileDevice("d1", "iPhone", "token1")
		deviceStore.devices["d1"] = dev

		uc, err := New(newMockTokenStore(), deviceStore, nil, "s", "s")
		require.NoError(t, err)

		err = uc.RevokeDevice(ctx, "d1")
		require.NoError(t, err)
		assert.Empty(t, deviceStore.devices)
	})

	t.Run("empty device id", func(t *testing.T) {
		uc, err := New(newMockTokenStore(), newMockDeviceStore(), nil, "s", "s")
		require.NoError(t, err)

		err = uc.RevokeDevice(ctx, "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, errors.CodeInvalidInput))
	})

	t.Run("device not found", func(t *testing.T) {
		uc, err := New(newMockTokenStore(), newMockDeviceStore(), nil, "s", "s")
		require.NoError(t, err)

		err = uc.RevokeDevice(ctx, "nonexistent")
		require.Error(t, err)
		assert.True(t, errors.Is(err, errors.CodeNotFound))
	})
}

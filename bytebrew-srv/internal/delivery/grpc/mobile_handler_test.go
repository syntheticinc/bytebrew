package grpc

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/usecase/list_mobile_sessions"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/usecase/pair_device"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// --- Mocks ---

type mockDeviceAuthenticator struct {
	device *domain.MobileDevice
	err    error
}

func (m *mockDeviceAuthenticator) GetDeviceByToken(_ context.Context, _ string) (*domain.MobileDevice, error) {
	return m.device, m.err
}

type mockPairDeviceUsecase struct {
	generateToken *domain.PairingToken
	generateErr   error
	pairOutput    *pair_device.PairOutput
	pairErr       error
	devices       []*domain.MobileDevice
	listErr       error
	revokeErr     error
}

func (m *mockPairDeviceUsecase) GeneratePairingToken(_ context.Context, _ string) (*domain.PairingToken, error) {
	return m.generateToken, m.generateErr
}

func (m *mockPairDeviceUsecase) Pair(_ context.Context, _ pair_device.PairInput) (*pair_device.PairOutput, error) {
	return m.pairOutput, m.pairErr
}

func (m *mockPairDeviceUsecase) ListDevices(_ context.Context) ([]*domain.MobileDevice, error) {
	return m.devices, m.listErr
}

func (m *mockPairDeviceUsecase) RevokeDevice(_ context.Context, _ string) error {
	return m.revokeErr
}

type mockListMobileSessionsUsecase struct {
	sessions []list_mobile_sessions.MobileSession
	err      error
}

func (m *mockListMobileSessionsUsecase) Execute(_ context.Context) ([]list_mobile_sessions.MobileSession, error) {
	return m.sessions, m.err
}

type mockMobileCommandUsecase struct {
	sendNewTaskErr    error
	sendAskUserErr    error
	cancelSessionErr  error
}

func (m *mockMobileCommandUsecase) SendNewTask(_ context.Context, _, _ string) error {
	return m.sendNewTaskErr
}

func (m *mockMobileCommandUsecase) SendAskUserReply(_ context.Context, _, _, _ string) error {
	return m.sendAskUserErr
}

func (m *mockMobileCommandUsecase) CancelSession(_ context.Context, _ string) error {
	return m.cancelSessionErr
}

type mockMobileEventSubscriber struct {
	ch           chan *pb.SessionEvent
	unsubscribed bool
	missedEvents []*pb.SessionEvent
}

func (m *mockMobileEventSubscriber) Subscribe(_, _ string) (<-chan *pb.SessionEvent, func()) {
	return m.ch, func() { m.unsubscribed = true }
}

func (m *mockMobileEventSubscriber) GetMissedEvents(_, _ string) []*pb.SessionEvent {
	return m.missedEvents
}

type mockRateLimiter struct {
	err error
}

func (m *mockRateLimiter) Allow(_ string) error {
	return m.err
}

// --- Helpers ---

func localhostPeerCtx() context.Context {
	return peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345},
	})
}

func ipv6LocalhostPeerCtx() context.Context {
	return peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP("::1"), Port: 12345},
	})
}

func remotePeerCtx(ip string) context.Context {
	return peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP(ip), Port: 12345},
	})
}

func withDeviceToken(ctx context.Context, token string) context.Context {
	return metadata.NewIncomingContext(ctx, metadata.Pairs("device-token", token))
}

func newTestHandler(t *testing.T, opts ...func(*MobileHandlerConfig)) *MobileHandler {
	t.Helper()

	cfg := MobileHandlerConfig{
		PairDevice:    &mockPairDeviceUsecase{},
		ListSessions:  &mockListMobileSessionsUsecase{},
		MobileCommand: &mockMobileCommandUsecase{},
		EventSub:      &mockMobileEventSubscriber{ch: make(chan *pb.SessionEvent)},
		DeviceAuth:    &mockDeviceAuthenticator{},
		PairLimiter:   &mockRateLimiter{},
		TokenLimiter:  &mockRateLimiter{},
		ServerName:    "test-server",
		ServerID:      "test-id",
		ServerPort:    60401,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	h, err := NewMobileHandler(cfg)
	require.NoError(t, err)
	return h
}

// --- Auth tests ---

func TestAuthenticateDevice_Localhost_NoToken(t *testing.T) {
	h := newTestHandler(t)
	ctx := localhostPeerCtx()
	// No metadata at all

	device, err := h.authenticateDevice(ctx)
	require.NoError(t, err)
	assert.Equal(t, localhostAdminDevice, device)
	assert.Equal(t, "localhost-admin", device.ID)
	assert.Equal(t, "CLI (localhost)", device.Name)
}

func TestAuthenticateDevice_Localhost_WithToken(t *testing.T) {
	expectedDevice := &domain.MobileDevice{
		ID:       "device-123",
		Name:     "iPhone",
		PairedAt: time.Now(),
	}
	h := newTestHandler(t, func(cfg *MobileHandlerConfig) {
		cfg.DeviceAuth = &mockDeviceAuthenticator{device: expectedDevice}
	})
	ctx := withDeviceToken(localhostPeerCtx(), "valid-token")

	device, err := h.authenticateDevice(ctx)
	require.NoError(t, err)
	assert.Equal(t, expectedDevice, device)
}

func TestAuthenticateDevice_Remote_MissingToken(t *testing.T) {
	h := newTestHandler(t)
	ctx := remotePeerCtx("192.168.1.100")
	// No metadata

	_, err := h.authenticateDevice(ctx)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "missing metadata")
}

func TestAuthenticateDevice_Remote_InvalidToken(t *testing.T) {
	h := newTestHandler(t, func(cfg *MobileHandlerConfig) {
		cfg.DeviceAuth = &mockDeviceAuthenticator{err: fmt.Errorf("token expired")}
	})
	ctx := withDeviceToken(remotePeerCtx("192.168.1.100"), "bad-token")

	_, err := h.authenticateDevice(ctx)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Equal(t, "invalid device token", st.Message())
	// Internal error must NOT leak
	assert.NotContains(t, st.Message(), "token expired")
}

func TestAuthenticateDevice_Remote_ValidToken(t *testing.T) {
	expectedDevice := &domain.MobileDevice{
		ID:       "device-456",
		Name:     "Android",
		PairedAt: time.Now(),
	}
	h := newTestHandler(t, func(cfg *MobileHandlerConfig) {
		cfg.DeviceAuth = &mockDeviceAuthenticator{device: expectedDevice}
	})
	ctx := withDeviceToken(remotePeerCtx("192.168.1.100"), "valid-token")

	device, err := h.authenticateDevice(ctx)
	require.NoError(t, err)
	assert.Equal(t, expectedDevice, device)
}

func TestAuthenticateDevice_IPv6_Localhost(t *testing.T) {
	h := newTestHandler(t)
	ctx := ipv6LocalhostPeerCtx()

	device, err := h.authenticateDevice(ctx)
	require.NoError(t, err)
	assert.Equal(t, localhostAdminDevice, device)
}

func TestAuthenticateDevice_Remote_NilDevice(t *testing.T) {
	h := newTestHandler(t, func(cfg *MobileHandlerConfig) {
		cfg.DeviceAuth = &mockDeviceAuthenticator{device: nil, err: nil}
	})
	ctx := withDeviceToken(remotePeerCtx("10.0.0.1"), "some-token")

	_, err := h.authenticateDevice(ctx)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Equal(t, "invalid device token", st.Message())
}

// --- Rate limiting tests ---

func TestGeneratePairingToken_RateLimited(t *testing.T) {
	h := newTestHandler(t, func(cfg *MobileHandlerConfig) {
		cfg.TokenLimiter = &mockRateLimiter{err: fmt.Errorf("rate limit exceeded")}
	})
	ctx := localhostPeerCtx()

	_, err := h.GeneratePairingToken(ctx, &pb.GeneratePairingTokenRequest{})
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.ResourceExhausted, st.Code())
	assert.Contains(t, st.Message(), "too many requests")
}

func TestPair_RateLimited(t *testing.T) {
	h := newTestHandler(t, func(cfg *MobileHandlerConfig) {
		cfg.PairLimiter = &mockRateLimiter{err: fmt.Errorf("rate limit exceeded")}
	})
	ctx := localhostPeerCtx()

	_, err := h.Pair(ctx, &pb.PairRequest{
		PairingToken: "some-token",
		DeviceName:   "test",
	})
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.ResourceExhausted, st.Code())
	assert.Contains(t, st.Message(), "too many requests")
}

// --- Error sanitization tests ---

func TestSendCommand_ErrorSanitized(t *testing.T) {
	internalErrMsg := "database connection refused at 10.0.0.5:5432"
	h := newTestHandler(t, func(cfg *MobileHandlerConfig) {
		cfg.MobileCommand = &mockMobileCommandUsecase{
			sendNewTaskErr: fmt.Errorf("%s", internalErrMsg),
		}
	})
	// Localhost without device-token metadata => authenticated as localhost admin
	ctx := localhostPeerCtx()

	resp, err := h.SendCommand(ctx, &pb.SendCommandRequest{
		SessionId: "session-1",
		Command:   &pb.SendCommandRequest_NewTask{NewTask: "do something"},
	})
	require.NoError(t, err) // SendCommand returns response, not gRPC error
	assert.False(t, resp.Success)
	assert.Equal(t, "command failed", resp.ErrorMessage)
	// Internal error details must NOT leak to the client
	assert.NotContains(t, resp.ErrorMessage, internalErrMsg)
}

// --- Happy path tests ---

func TestPing(t *testing.T) {
	h := newTestHandler(t)
	ctx := context.Background()

	resp, err := h.Ping(ctx, &pb.MobilePingRequest{})
	require.NoError(t, err)
	assert.Equal(t, "test-server", resp.ServerName)
	assert.Equal(t, "test-id", resp.ServerId)
	assert.Greater(t, resp.Timestamp, int64(0))
}

func TestListDevices_Success(t *testing.T) {
	now := time.Now()
	devices := []*domain.MobileDevice{
		{ID: "d1", Name: "iPhone 15", PairedAt: now, LastSeenAt: now},
		{ID: "d2", Name: "Pixel 9", PairedAt: now, LastSeenAt: now},
	}
	h := newTestHandler(t, func(cfg *MobileHandlerConfig) {
		cfg.PairDevice = &mockPairDeviceUsecase{devices: devices}
	})
	ctx := localhostPeerCtx()

	resp, err := h.ListDevices(ctx, &pb.ListDevicesRequest{})
	require.NoError(t, err)
	assert.Len(t, resp.Devices, 2)
	assert.Equal(t, "d1", resp.Devices[0].DeviceId)
	assert.Equal(t, "iPhone 15", resp.Devices[0].DeviceName)
	assert.Equal(t, "d2", resp.Devices[1].DeviceId)
}

func TestListDevices_Unauthenticated(t *testing.T) {
	h := newTestHandler(t)
	ctx := remotePeerCtx("192.168.1.100")

	_, err := h.ListDevices(ctx, &pb.ListDevicesRequest{})
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestGeneratePairingToken_Success(t *testing.T) {
	token := &domain.PairingToken{
		Token:           "full-token",
		ShortCode:       "123456",
		ExpiresAt:       time.Now().Add(5 * time.Minute),
		ServerPublicKey: []byte("pubkey"),
	}
	h := newTestHandler(t, func(cfg *MobileHandlerConfig) {
		cfg.PairDevice = &mockPairDeviceUsecase{generateToken: token}
	})
	ctx := localhostPeerCtx()

	resp, err := h.GeneratePairingToken(ctx, &pb.GeneratePairingTokenRequest{})
	require.NoError(t, err)
	assert.Equal(t, "123456", resp.ShortCode)
	assert.Equal(t, "full-token", resp.Token)
	assert.Equal(t, "test-server", resp.ServerName)
	assert.Equal(t, "test-id", resp.ServerId)
	assert.Equal(t, int32(60401), resp.ServerPort)
	assert.Equal(t, []byte("pubkey"), resp.ServerPublicKey)
}

func TestPair_Success(t *testing.T) {
	output := &pair_device.PairOutput{
		DeviceID:        "new-device-id",
		DeviceToken:     "new-device-token",
		ServerName:      "test-server",
		ServerPublicKey: []byte("pub"),
	}
	h := newTestHandler(t, func(cfg *MobileHandlerConfig) {
		cfg.PairDevice = &mockPairDeviceUsecase{pairOutput: output}
	})
	ctx := localhostPeerCtx()

	resp, err := h.Pair(ctx, &pb.PairRequest{
		PairingToken: "token-123",
		DeviceName:   "My Phone",
	})
	require.NoError(t, err)
	assert.Equal(t, "new-device-id", resp.DeviceId)
	assert.Equal(t, "new-device-token", resp.DeviceToken)
	assert.Equal(t, "test-server", resp.ServerName)
	assert.Equal(t, "test-id", resp.ServerId)
	assert.Equal(t, []byte("pub"), resp.ServerPublicKey)
}

func TestSendCommand_NewTask_Success(t *testing.T) {
	h := newTestHandler(t)
	ctx := localhostPeerCtx()

	resp, err := h.SendCommand(ctx, &pb.SendCommandRequest{
		SessionId: "s1",
		Command:   &pb.SendCommandRequest_NewTask{NewTask: "analyze code"},
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Empty(t, resp.ErrorMessage)
}

func TestSendCommand_AskUserReply_Success(t *testing.T) {
	h := newTestHandler(t)
	ctx := localhostPeerCtx()

	resp, err := h.SendCommand(ctx, &pb.SendCommandRequest{
		SessionId: "s1",
		Command: &pb.SendCommandRequest_AskUserReply{
			AskUserReply: &pb.AskUserResponse{
				Question: "Continue?",
				Answer:   "Yes",
			},
		},
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestSendCommand_Cancel_Success(t *testing.T) {
	h := newTestHandler(t)
	ctx := localhostPeerCtx()

	resp, err := h.SendCommand(ctx, &pb.SendCommandRequest{
		SessionId: "s1",
		Command:   &pb.SendCommandRequest_Cancel{Cancel: true},
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestSendCommand_MissingSessionID(t *testing.T) {
	h := newTestHandler(t)
	ctx := localhostPeerCtx()

	_, err := h.SendCommand(ctx, &pb.SendCommandRequest{
		Command: &pb.SendCommandRequest_NewTask{NewTask: "task"},
	})
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestSendCommand_MissingCommand(t *testing.T) {
	h := newTestHandler(t)
	ctx := localhostPeerCtx()

	_, err := h.SendCommand(ctx, &pb.SendCommandRequest{
		SessionId: "s1",
	})
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// --- Helper function tests ---

func TestExtractPeerIP(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{
			name: "IPv4 TCP address",
			ctx: peer.NewContext(context.Background(), &peer.Peer{
				Addr: &net.TCPAddr{IP: net.ParseIP("192.168.1.5"), Port: 9999},
			}),
			want: "192.168.1.5",
		},
		{
			name: "IPv6 TCP address",
			ctx: peer.NewContext(context.Background(), &peer.Peer{
				Addr: &net.TCPAddr{IP: net.ParseIP("::1"), Port: 9999},
			}),
			want: "::1",
		},
		{
			name: "no peer info",
			ctx:  context.Background(),
			want: "unknown",
		},
		{
			name: "non-TCP addr with host:port",
			ctx: peer.NewContext(context.Background(), &peer.Peer{
				Addr: fakeAddr("10.0.0.1:8080"),
			}),
			want: "10.0.0.1",
		},
		{
			name: "non-TCP addr without port",
			ctx: peer.NewContext(context.Background(), &peer.Peer{
				Addr: fakeAddr("unix-socket"),
			}),
			want: "unix-socket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPeerIP(tt.ctx)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsLocalhost(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want bool
	}{
		{
			name: "127.0.0.1",
			ctx: peer.NewContext(context.Background(), &peer.Peer{
				Addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1234},
			}),
			want: true,
		},
		{
			name: "::1 (IPv6 loopback)",
			ctx: peer.NewContext(context.Background(), &peer.Peer{
				Addr: &net.TCPAddr{IP: net.ParseIP("::1"), Port: 1234},
			}),
			want: true,
		},
		{
			name: "remote IP",
			ctx: peer.NewContext(context.Background(), &peer.Peer{
				Addr: &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 1234},
			}),
			want: false,
		},
		{
			name: "no peer",
			ctx:  context.Background(),
			want: false,
		},
		{
			name: "non-TCP localhost string",
			ctx: peer.NewContext(context.Background(), &peer.Peer{
				Addr: fakeAddr("127.0.0.1:5555"),
			}),
			want: true,
		},
		{
			name: "non-TCP ::1 without brackets (unparseable)",
			ctx: peer.NewContext(context.Background(), &peer.Peer{
				Addr: fakeAddr("::1:5555"),
			}),
			// net.SplitHostPort fails on "::1:5555" (too many colons, no brackets)
			want: false,
		},
		{
			name: "non-TCP [::1] with brackets",
			ctx: peer.NewContext(context.Background(), &peer.Peer{
				Addr: fakeAddr("[::1]:5555"),
			}),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isLocalhost(tt.ctx)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- Constructor validation tests ---

func TestNewMobileHandler_Validation(t *testing.T) {
	validCfg := MobileHandlerConfig{
		PairDevice:    &mockPairDeviceUsecase{},
		ListSessions:  &mockListMobileSessionsUsecase{},
		MobileCommand: &mockMobileCommandUsecase{},
		EventSub:      &mockMobileEventSubscriber{ch: make(chan *pb.SessionEvent)},
		DeviceAuth:    &mockDeviceAuthenticator{},
		PairLimiter:   &mockRateLimiter{},
		TokenLimiter:  &mockRateLimiter{},
		ServerName:    "srv",
		ServerID:      "id",
		ServerPort:    60401,
	}

	tests := []struct {
		name    string
		modify  func(*MobileHandlerConfig)
		wantErr string
	}{
		{
			name:   "valid config",
			modify: func(_ *MobileHandlerConfig) {},
		},
		{
			name:    "nil PairDevice",
			modify:  func(c *MobileHandlerConfig) { c.PairDevice = nil },
			wantErr: "pair device usecase is required",
		},
		{
			name:    "nil ListSessions",
			modify:  func(c *MobileHandlerConfig) { c.ListSessions = nil },
			wantErr: "list sessions usecase is required",
		},
		{
			name:    "nil MobileCommand",
			modify:  func(c *MobileHandlerConfig) { c.MobileCommand = nil },
			wantErr: "mobile command usecase is required",
		},
		{
			name:    "nil EventSub",
			modify:  func(c *MobileHandlerConfig) { c.EventSub = nil },
			wantErr: "event subscriber is required",
		},
		{
			name:    "nil DeviceAuth",
			modify:  func(c *MobileHandlerConfig) { c.DeviceAuth = nil },
			wantErr: "device authenticator is required",
		},
		{
			name:    "empty ServerName",
			modify:  func(c *MobileHandlerConfig) { c.ServerName = "" },
			wantErr: "server name is required",
		},
		{
			name:    "empty ServerID",
			modify:  func(c *MobileHandlerConfig) { c.ServerID = "" },
			wantErr: "server id is required",
		},
		{
			name:    "nil PairLimiter",
			modify:  func(c *MobileHandlerConfig) { c.PairLimiter = nil },
			wantErr: "pair rate limiter is required",
		},
		{
			name:    "nil TokenLimiter",
			modify:  func(c *MobileHandlerConfig) { c.TokenLimiter = nil },
			wantErr: "token rate limiter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validCfg
			tt.modify(&cfg)

			h, err := NewMobileHandler(cfg)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, h)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, h)
		})
	}
}

// --- mapFlowStatusToSessionState tests ---

func TestMapFlowStatusToSessionState(t *testing.T) {
	tests := []struct {
		input domain.FlowStatus
		want  pb.SessionState
	}{
		{domain.FlowStatusRunning, pb.SessionState_SESSION_STATE_ACTIVE},
		{domain.FlowStatusCompleted, pb.SessionState_SESSION_STATE_COMPLETED},
		{domain.FlowStatusFailed, pb.SessionState_SESSION_STATE_FAILED},
		{"unknown", pb.SessionState_SESSION_STATE_IDLE},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			got := mapFlowStatusToSessionState(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- ListSessions tests ---

func TestListSessions_Success(t *testing.T) {
	now := time.Now()
	sessions := []list_mobile_sessions.MobileSession{
		{
			SessionID:      "s1",
			ProjectKey:     "proj1",
			ProjectRoot:    "/home/user/project",
			Status:         domain.FlowStatusRunning,
			CurrentTask:    "analyze code",
			StartedAt:      now,
			LastActivityAt: now,
			HasAskUser:     true,
			Platform:       "linux",
		},
	}
	h := newTestHandler(t, func(cfg *MobileHandlerConfig) {
		cfg.ListSessions = &mockListMobileSessionsUsecase{sessions: sessions}
	})
	ctx := localhostPeerCtx()

	resp, err := h.ListSessions(ctx, &pb.ListSessionsRequest{})
	require.NoError(t, err)
	assert.Len(t, resp.Sessions, 1)
	assert.Equal(t, "s1", resp.Sessions[0].SessionId)
	assert.Equal(t, "proj1", resp.Sessions[0].ProjectKey)
	assert.Equal(t, pb.SessionState_SESSION_STATE_ACTIVE, resp.Sessions[0].Status)
	assert.Equal(t, "analyze code", resp.Sessions[0].CurrentTask)
	assert.True(t, resp.Sessions[0].HasAskUser)
	assert.Equal(t, "linux", resp.Sessions[0].Platform)
	assert.Equal(t, "test-server", resp.ServerName)
	assert.Equal(t, "test-id", resp.ServerId)
}

func TestListSessions_InternalError(t *testing.T) {
	h := newTestHandler(t, func(cfg *MobileHandlerConfig) {
		cfg.ListSessions = &mockListMobileSessionsUsecase{err: fmt.Errorf("db broken")}
	})
	ctx := localhostPeerCtx()

	_, err := h.ListSessions(ctx, &pb.ListSessionsRequest{})
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	// Internal error must not leak
	assert.NotContains(t, st.Message(), "db broken")
}

// --- RevokeDevice tests ---

func TestRevokeDevice_Success(t *testing.T) {
	h := newTestHandler(t)
	ctx := localhostPeerCtx()

	resp, err := h.RevokeDevice(ctx, &pb.RevokeDeviceRequest{DeviceId: "d1"})
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestRevokeDevice_MissingDeviceID(t *testing.T) {
	h := newTestHandler(t)
	ctx := localhostPeerCtx()

	_, err := h.RevokeDevice(ctx, &pb.RevokeDeviceRequest{})
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// --- fakeAddr helper for non-TCP peer testing ---

type fakeAddr string

func (f fakeAddr) Network() string { return "fake" }
func (f fakeAddr) String() string  { return string(f) }

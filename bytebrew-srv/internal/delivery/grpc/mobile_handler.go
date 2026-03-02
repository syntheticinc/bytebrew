package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/usecase/list_mobile_sessions"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/usecase/pair_device"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// PairDeviceUsecase defines the pair device operations
type PairDeviceUsecase interface {
	GeneratePairingToken(ctx context.Context, serverID string) (*domain.PairingToken, error)
	Pair(ctx context.Context, input pair_device.PairInput) (*pair_device.PairOutput, error)
	ListDevices(ctx context.Context) ([]*domain.MobileDevice, error)
	RevokeDevice(ctx context.Context, deviceID string) error
}

// ListMobileSessionsUsecase defines the list sessions operation
type ListMobileSessionsUsecase interface {
	Execute(ctx context.Context) ([]list_mobile_sessions.MobileSession, error)
}

// MobileCommandUsecase defines the mobile command operations
type MobileCommandUsecase interface {
	SendNewTask(ctx context.Context, sessionID, task string) error
	SendAskUserReply(ctx context.Context, sessionID, question, answer string) error
	CancelSession(ctx context.Context, sessionID string) error
}

// MobileEventSubscriber defines event subscription for mobile clients
type MobileEventSubscriber interface {
	Subscribe(sessionID, subscriberID string) (<-chan *pb.SessionEvent, func())
	GetMissedEvents(sessionID, lastEventID string) []*pb.SessionEvent
}

// DeviceAuthenticator authenticates mobile devices by token
type DeviceAuthenticator interface {
	GetDeviceByToken(ctx context.Context, deviceToken string) (*domain.MobileDevice, error)
}

// RateLimiter checks whether a request from the given key is allowed
type RateLimiter interface {
	Allow(key string) error
}

// PairingWaiterService allows CLI to wait for and be notified about pairing completion.
type PairingWaiterService interface {
	Register(token string) <-chan domain.PairingNotification
	Notify(token string, notification domain.PairingNotification)
	Unregister(token string)
}

// localhostAdminDevice is a sentinel device returned for authenticated
// localhost connections that do not carry a device-token.
var localhostAdminDevice = &domain.MobileDevice{
	ID:   "localhost-admin",
	Name: "CLI (localhost)",
}

// MobileHandler handles MobileService gRPC requests
type MobileHandler struct {
	pb.UnimplementedMobileServiceServer
	pairDevice     PairDeviceUsecase
	listSessions   ListMobileSessionsUsecase
	mobileCommand  MobileCommandUsecase
	eventSub       MobileEventSubscriber
	deviceAuth     DeviceAuthenticator
	pairLimiter    RateLimiter
	tokenLimiter   RateLimiter
	pairingWaiter  PairingWaiterService
	serverName     string
	serverID       string
	serverPort     int32
}

// MobileHandlerConfig holds configuration for MobileHandler
type MobileHandlerConfig struct {
	PairDevice    PairDeviceUsecase
	ListSessions  ListMobileSessionsUsecase
	MobileCommand MobileCommandUsecase
	EventSub      MobileEventSubscriber
	DeviceAuth    DeviceAuthenticator
	PairLimiter   RateLimiter
	TokenLimiter  RateLimiter
	PairingWaiter PairingWaiterService
	ServerName    string
	ServerID      string
	ServerPort    int32
}

// NewMobileHandler creates a new MobileHandler with validated config
func NewMobileHandler(cfg MobileHandlerConfig) (*MobileHandler, error) {
	if cfg.PairDevice == nil {
		return nil, fmt.Errorf("pair device usecase is required")
	}
	if cfg.ListSessions == nil {
		return nil, fmt.Errorf("list sessions usecase is required")
	}
	if cfg.MobileCommand == nil {
		return nil, fmt.Errorf("mobile command usecase is required")
	}
	if cfg.EventSub == nil {
		return nil, fmt.Errorf("event subscriber is required")
	}
	if cfg.DeviceAuth == nil {
		return nil, fmt.Errorf("device authenticator is required")
	}
	if cfg.ServerName == "" {
		return nil, fmt.Errorf("server name is required")
	}
	if cfg.ServerID == "" {
		return nil, fmt.Errorf("server id is required")
	}
	if cfg.PairLimiter == nil {
		return nil, fmt.Errorf("pair rate limiter is required")
	}
	if cfg.TokenLimiter == nil {
		return nil, fmt.Errorf("token rate limiter is required")
	}

	return &MobileHandler{
		pairDevice:    cfg.PairDevice,
		listSessions:  cfg.ListSessions,
		mobileCommand: cfg.MobileCommand,
		eventSub:      cfg.EventSub,
		deviceAuth:    cfg.DeviceAuth,
		pairLimiter:   cfg.PairLimiter,
		tokenLimiter:  cfg.TokenLimiter,
		pairingWaiter: cfg.PairingWaiter,
		serverName:    cfg.ServerName,
		serverID:      cfg.ServerID,
		serverPort:    cfg.ServerPort,
	}, nil
}

// GeneratePairingToken creates a temporary pairing token for mobile device pairing.
// This endpoint does NOT require device authentication (called by CLI to initiate pairing).
func (h *MobileHandler) GeneratePairingToken(ctx context.Context, _ *pb.GeneratePairingTokenRequest) (*pb.GeneratePairingTokenResponse, error) {
	peerIP := extractPeerIP(ctx)
	if err := h.tokenLimiter.Allow(peerIP); err != nil {
		slog.WarnContext(ctx, "token generation rate limited", "peer_ip", peerIP)
		return nil, status.Errorf(codes.ResourceExhausted, "too many requests, try again later")
	}

	slog.InfoContext(ctx, "generating pairing token")

	token, err := h.pairDevice.GeneratePairingToken(ctx, h.serverID)
	if err != nil {
		slog.ErrorContext(ctx, "generate pairing token failed", "error", err)
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	return &pb.GeneratePairingTokenResponse{
		ShortCode:       token.ShortCode,
		Token:           token.Token,
		ExpiresAt:       token.ExpiresAt.Unix(),
		ServerName:      h.serverName,
		ServerId:        h.serverID,
		ServerPort:      h.serverPort,
		ServerPublicKey: token.ServerPublicKey,
	}, nil
}

// Pair authenticates a mobile device using a pairing token.
// This endpoint does NOT require device authentication (it IS the pairing endpoint).
func (h *MobileHandler) Pair(ctx context.Context, req *pb.PairRequest) (*pb.PairResponse, error) {
	peerIP := extractPeerIP(ctx)
	if err := h.pairLimiter.Allow(peerIP); err != nil {
		slog.WarnContext(ctx, "pair rate limited", "peer_ip", peerIP)
		return nil, status.Errorf(codes.ResourceExhausted, "too many requests, try again later")
	}

	slog.InfoContext(ctx, "mobile pair request", "device_name", req.GetDeviceName())

	output, err := h.pairDevice.Pair(ctx, pair_device.PairInput{
		TokenOrCode:     req.GetPairingToken(),
		DeviceName:      req.GetDeviceName(),
		MobilePublicKey: req.GetMobilePublicKey(),
	})
	if err != nil {
		slog.ErrorContext(ctx, "pair device failed", "error", err)
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	// Notify any CLI waiting for this pairing token to be consumed
	if h.pairingWaiter != nil && output.Token != "" {
		h.pairingWaiter.Notify(output.Token, domain.PairingNotification{
			DeviceName: req.GetDeviceName(),
			DeviceID:   output.DeviceID,
		})
	}

	return &pb.PairResponse{
		DeviceId:        output.DeviceID,
		DeviceToken:     output.DeviceToken,
		ServerName:      output.ServerName,
		ServerId:        h.serverID,
		ServerPublicKey: output.ServerPublicKey,
	}, nil
}

// ListSessions returns all active CLI sessions on this server.
func (h *MobileHandler) ListSessions(ctx context.Context, _ *pb.ListSessionsRequest) (*pb.ListSessionsResponse, error) {
	if _, err := h.authenticateDevice(ctx); err != nil {
		return nil, err
	}

	sessions, err := h.listSessions.Execute(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "list sessions failed", "error", err)
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	protoSessions := make([]*pb.MobileSession, 0, len(sessions))
	for _, s := range sessions {
		protoSessions = append(protoSessions, &pb.MobileSession{
			SessionId:      s.SessionID,
			ProjectKey:     s.ProjectKey,
			ProjectRoot:    s.ProjectRoot,
			Status:         mapFlowStatusToSessionState(s.Status),
			CurrentTask:    s.CurrentTask,
			StartedAt:      s.StartedAt.Unix(),
			LastActivityAt: s.LastActivityAt.Unix(),
			HasAskUser:     s.HasAskUser,
			Platform:       s.Platform,
		})
	}

	return &pb.ListSessionsResponse{
		Sessions:   protoSessions,
		ServerName: h.serverName,
		ServerId:   h.serverID,
	}, nil
}

// SubscribeSession subscribes to real-time events from a specific session.
// If last_event_id is provided, missed events are sent first (backfill) before
// switching to live streaming.
func (h *MobileHandler) SubscribeSession(req *pb.SubscribeSessionRequest, stream grpc.ServerStreamingServer[pb.SessionEvent]) error {
	ctx := stream.Context()

	if _, err := h.authenticateDevice(ctx); err != nil {
		return err
	}

	sessionID := req.GetSessionId()
	if sessionID == "" {
		return status.Error(codes.InvalidArgument, "session_id is required")
	}

	lastEventID := req.GetLastEventId()
	slog.InfoContext(ctx, "mobile subscribe session",
		"session_id", sessionID,
		"last_event_id", lastEventID,
	)

	subscriberID := fmt.Sprintf("mobile-%d", time.Now().UnixNano())
	eventCh, unsubscribe := h.eventSub.Subscribe(sessionID, subscriberID)
	defer unsubscribe()

	// Backfill: send missed events before starting live stream
	if lastEventID != "" {
		missed := h.eventSub.GetMissedEvents(sessionID, lastEventID)
		slog.InfoContext(ctx, "backfill missed events",
			"session_id", sessionID,
			"count", len(missed),
		)
		for _, event := range missed {
			if err := stream.Send(event); err != nil {
				slog.ErrorContext(ctx, "failed to send backfill event", "error", err, "session_id", sessionID)
				return status.Errorf(codes.Internal, "internal error")
			}
		}
	}

	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "mobile subscriber disconnected", "session_id", sessionID, "subscriber_id", subscriberID)
			return nil
		case event, ok := <-eventCh:
			if !ok {
				slog.InfoContext(ctx, "event channel closed", "session_id", sessionID)
				return nil
			}
			if err := stream.Send(event); err != nil {
				slog.ErrorContext(ctx, "failed to send event to mobile", "error", err, "session_id", sessionID)
				return status.Errorf(codes.Internal, "internal error")
			}
		}
	}
}

// SendCommand sends a command to a session based on the command type.
func (h *MobileHandler) SendCommand(ctx context.Context, req *pb.SendCommandRequest) (*pb.SendCommandResponse, error) {
	if _, err := h.authenticateDevice(ctx); err != nil {
		return nil, err
	}

	sessionID := req.GetSessionId()
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}

	slog.InfoContext(ctx, "mobile send command", "session_id", sessionID)

	var err error
	switch cmd := req.GetCommand().(type) {
	case *pb.SendCommandRequest_NewTask:
		err = h.mobileCommand.SendNewTask(ctx, sessionID, cmd.NewTask)
	case *pb.SendCommandRequest_AskUserReply:
		reply := cmd.AskUserReply
		err = h.mobileCommand.SendAskUserReply(ctx, sessionID, reply.GetQuestion(), reply.GetAnswer())
	case *pb.SendCommandRequest_Cancel:
		err = h.mobileCommand.CancelSession(ctx, sessionID)
	default:
		return nil, status.Error(codes.InvalidArgument, "command is required")
	}

	if err != nil {
		slog.ErrorContext(ctx, "send command failed", "error", err, "session_id", sessionID)
		return &pb.SendCommandResponse{
			Success:      false,
			ErrorMessage: "command failed",
		}, nil
	}

	return &pb.SendCommandResponse{Success: true}, nil
}

// ListDevices returns all paired mobile devices.
func (h *MobileHandler) ListDevices(ctx context.Context, _ *pb.ListDevicesRequest) (*pb.ListDevicesResponse, error) {
	if _, err := h.authenticateDevice(ctx); err != nil {
		return nil, err
	}

	devices, err := h.pairDevice.ListDevices(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "list devices failed", "error", err)
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	protoDevices := make([]*pb.PairedDevice, 0, len(devices))
	for _, d := range devices {
		protoDevices = append(protoDevices, &pb.PairedDevice{
			DeviceId:   d.ID,
			DeviceName: d.Name,
			PairedAt:   d.PairedAt.Unix(),
			LastSeenAt: d.LastSeenAt.Unix(),
		})
	}

	return &pb.ListDevicesResponse{Devices: protoDevices}, nil
}

// RevokeDevice revokes a paired device's access.
func (h *MobileHandler) RevokeDevice(ctx context.Context, req *pb.RevokeDeviceRequest) (*pb.RevokeDeviceResponse, error) {
	if _, err := h.authenticateDevice(ctx); err != nil {
		return nil, err
	}

	if req.GetDeviceId() == "" {
		return nil, status.Error(codes.InvalidArgument, "device_id is required")
	}

	if err := h.pairDevice.RevokeDevice(ctx, req.GetDeviceId()); err != nil {
		slog.ErrorContext(ctx, "revoke device failed", "error", err, "device_id", req.GetDeviceId())
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	return &pb.RevokeDeviceResponse{Success: true}, nil
}

// Ping checks connectivity and returns server info.
func (h *MobileHandler) Ping(ctx context.Context, req *pb.MobilePingRequest) (*pb.MobilePongResponse, error) {
	return &pb.MobilePongResponse{
		Timestamp:  time.Now().UnixMilli(),
		ServerName: h.serverName,
		ServerId:   h.serverID,
	}, nil
}

// WaitForPairing blocks until the given pairing token is consumed by a mobile
// device, then sends a single event with the paired device info and closes.
// Called by CLI after displaying the QR code.
func (h *MobileHandler) WaitForPairing(req *pb.WaitForPairingRequest, stream grpc.ServerStreamingServer[pb.WaitForPairingEvent]) error {
	ctx := stream.Context()

	token := req.GetToken()
	if token == "" {
		return status.Error(codes.InvalidArgument, "token is required")
	}

	if h.pairingWaiter == nil {
		return status.Error(codes.Internal, "pairing waiter not configured")
	}

	slog.InfoContext(ctx, "waiting for pairing", "token_prefix", truncateToken(token))

	ch := h.pairingWaiter.Register(token)
	defer h.pairingWaiter.Unregister(token)

	select {
	case <-ctx.Done():
		slog.InfoContext(ctx, "wait for pairing cancelled", "token_prefix", truncateToken(token))
		return status.Error(codes.Canceled, "client disconnected")
	case result, ok := <-ch:
		if !ok {
			// Channel closed without a result (e.g. Unregister called externally)
			return status.Error(codes.Aborted, "pairing wait aborted")
		}

		slog.InfoContext(ctx, "pairing completed, notifying CLI",
			"device_name", result.DeviceName,
			"device_id", result.DeviceID,
		)

		if err := stream.Send(&pb.WaitForPairingEvent{
			DeviceName: result.DeviceName,
			DeviceId:   result.DeviceID,
		}); err != nil {
			slog.ErrorContext(ctx, "failed to send pairing event", "error", err)
			return status.Errorf(codes.Internal, "failed to send event")
		}

		return nil
	}
}

// truncateToken returns the first 8 characters of a token for logging.
func truncateToken(token string) string {
	if len(token) > 8 {
		return token[:8]
	}
	return token
}

// authenticateDevice extracts the device-token from gRPC metadata and validates it.
// Localhost connections (CLI on the same machine) are allowed without device-token
// and return the localhostAdminDevice sentinel.
func (h *MobileHandler) authenticateDevice(ctx context.Context) (*domain.MobileDevice, error) {
	// Allow localhost connections without auth (CLI admin access)
	if isLocalhost(ctx) {
		md, _ := metadata.FromIncomingContext(ctx)
		if md == nil || len(md.Get("device-token")) == 0 {
			return localhostAdminDevice, nil
		}
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	tokens := md.Get("device-token")
	if len(tokens) == 0 {
		return nil, status.Error(codes.Unauthenticated, "missing device-token")
	}

	device, err := h.deviceAuth.GetDeviceByToken(ctx, tokens[0])
	if err != nil {
		slog.ErrorContext(ctx, "device authentication failed", "error", err)
		return nil, status.Error(codes.Unauthenticated, "invalid device token")
	}
	if device == nil {
		return nil, status.Error(codes.Unauthenticated, "invalid device token")
	}

	// Skip UpdateLastSeen for the immutable localhost sentinel.
	if device != localhostAdminDevice {
		device.UpdateLastSeen()
	}

	return device, nil
}

// extractPeerIP returns the IP address of the gRPC caller.
// Falls back to "unknown" if peer info is unavailable.
func extractPeerIP(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return "unknown"
	}

	addr, ok := p.Addr.(*net.TCPAddr)
	if ok {
		return addr.IP.String()
	}

	host, _, err := net.SplitHostPort(p.Addr.String())
	if err != nil {
		return p.Addr.String()
	}
	return host
}

// isLocalhost checks if the gRPC call originates from localhost.
func isLocalhost(ctx context.Context) bool {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return false
	}

	addr, ok := p.Addr.(*net.TCPAddr)
	if !ok {
		host, _, err := net.SplitHostPort(p.Addr.String())
		if err != nil {
			return false
		}
		return host == "127.0.0.1" || host == "::1" || strings.HasPrefix(host, "localhost")
	}

	return addr.IP.IsLoopback()
}

// mapFlowStatusToSessionState converts domain FlowStatus to proto SessionState.
func mapFlowStatusToSessionState(s domain.FlowStatus) pb.SessionState {
	switch s {
	case domain.FlowStatusRunning:
		return pb.SessionState_SESSION_STATE_ACTIVE
	case domain.FlowStatusCompleted:
		return pb.SessionState_SESSION_STATE_COMPLETED
	case domain.FlowStatusFailed:
		return pb.SessionState_SESSION_STATE_FAILED
	default:
		return pb.SessionState_SESSION_STATE_IDLE
	}
}

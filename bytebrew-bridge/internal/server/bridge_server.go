package server

import (
	"context"

	bridgev1 "github.com/syntheticinc/bytebrew/bytebrew-bridge/api/proto/gen"
	"google.golang.org/grpc"
)

// RelayPool abstracts the connection pool for relay operations.
type RelayPool interface {
	RegisterServer(stream grpc.BidiStreamingServer[bridgev1.BridgeFrame, bridgev1.BridgeFrame]) error
	ConnectMobile(stream grpc.BidiStreamingServer[bridgev1.BridgeFrame, bridgev1.BridgeFrame]) error
	ListOnline() []*bridgev1.OnlineServer
}

// BridgeServer implements the BridgeService gRPC API.
// It delegates all relay logic to the RelayPool.
type BridgeServer struct {
	bridgev1.UnimplementedBridgeServiceServer
	pool RelayPool
}

// NewBridgeServer creates a new bridge server backed by the given relay pool.
func NewBridgeServer(pool RelayPool) *BridgeServer {
	return &BridgeServer{pool: pool}
}

// RegisterServer handles server registration and bidirectional frame relay.
func (s *BridgeServer) RegisterServer(stream grpc.BidiStreamingServer[bridgev1.BridgeFrame, bridgev1.BridgeFrame]) error {
	return s.pool.RegisterServer(stream)
}

// Connect handles mobile device connection and bidirectional frame relay.
func (s *BridgeServer) Connect(stream grpc.BidiStreamingServer[bridgev1.BridgeFrame, bridgev1.BridgeFrame]) error {
	return s.pool.ConnectMobile(stream)
}

// ListOnlineServers returns all currently registered servers.
func (s *BridgeServer) ListOnlineServers(_ context.Context, _ *bridgev1.ListOnlineRequest) (*bridgev1.ListOnlineResponse, error) {
	servers := s.pool.ListOnline()
	return &bridgev1.ListOnlineResponse{Servers: servers}, nil
}

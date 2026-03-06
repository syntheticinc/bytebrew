package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/config"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/logger"
)

// Server represents the gRPC server
type Server struct {
	grpcServer *grpc.Server
	listener   net.Listener
	logger     *logger.Logger
	config     config.ServerConfig
}

// buildGRPCOpts creates the common gRPC server options shared by all constructors.
func buildGRPCOpts(cfg config.ServerConfig, log *logger.Logger, licenseInfo *domain.LicenseInfo) []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.MaxRecvMsgSize(cfg.GRPC.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(cfg.GRPC.MaxSendMsgSize),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    cfg.GRPC.Keepalive.Time,
			Timeout: cfg.GRPC.Keepalive.Timeout,
		}),
		grpc.ConnectionTimeout(cfg.GRPC.ConnectionTimeout),
		// Unary interceptors
		grpc.ChainUnaryInterceptor(
			RecoveryInterceptor(log),
			LoggingInterceptor(log),
			LicenseUnaryInterceptor(licenseInfo),
			ErrorMappingInterceptor(),
		),
		// Stream interceptors
		grpc.ChainStreamInterceptor(
			StreamRecoveryInterceptor(log),
			StreamLoggingInterceptor(log),
			LicenseStreamInterceptor(licenseInfo),
			StreamErrorMappingInterceptor(),
		),
	}
}

// newServerFromListener creates a Server from an already-established listener.
func newServerFromListener(listener net.Listener, log *logger.Logger, cfg config.ServerConfig, licenseInfo *domain.LicenseInfo) *Server {
	opts := buildGRPCOpts(cfg, log, licenseInfo)
	grpcServer := grpc.NewServer(opts...)

	return &Server{
		grpcServer: grpcServer,
		listener:   listener,
		logger:     log,
		config:     cfg,
	}
}

// NewServer creates a new gRPC server that listens on the address from config.
// licenseInfo controls license enforcement: Blocked rejects sessions, Grace adds warning headers.
func NewServer(cfg config.ServerConfig, log *logger.Logger, licenseInfo *domain.LicenseInfo) (*Server, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	return newServerFromListener(listener, log, cfg, licenseInfo), nil
}

// NewServerWithListener creates a gRPC server with a pre-existing listener.
// Used in managed mode and port fallback where the OS assigns a random port.
// cfg provides gRPC options (keepalive, message sizes, timeouts).
func NewServerWithListener(listener net.Listener, cfg config.ServerConfig, log *logger.Logger, licenseInfo *domain.LicenseInfo) *Server {
	return newServerFromListener(listener, log, cfg, licenseInfo)
}

// ActualPort returns the port the server is listening on.
// Useful in managed mode where the OS assigns a random port.
func (s *Server) ActualPort() int {
	return s.listener.Addr().(*net.TCPAddr).Port
}

// RegisterServices registers all gRPC services
func (s *Server) RegisterServices(
	flowHandler FlowServiceHandler,
	indexingHandler IndexingServiceHandler,
	clientOpsHandler ClientOperationsServiceHandler,
) {
	// Register FlowService
	if flowHandler != nil {
		pb.RegisterFlowServiceServer(s.grpcServer, flowHandler)
		s.logger.Info("FlowService registered")
	}

	// IndexingService will be implemented as separate gRPC service (see task 003)
	// ClientOperationsService is implemented via StreamBasedClientOperationsProxy using FlowService bidirectional stream
}

// Start starts the gRPC server
func (s *Server) Start(ctx context.Context) error {
	s.logger.InfoContext(ctx, "Starting gRPC server",
		"address", s.listener.Addr().String(),
	)

	errChan := make(chan error, 1)
	go func() {
		if err := s.grpcServer.Serve(s.listener); err != nil {
			errChan <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return s.Shutdown(context.Background())
	}
}

// Shutdown gracefully shuts down the gRPC server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.InfoContext(ctx, "Shutting down gRPC server")

	// Graceful stop with timeout
	stopped := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(stopped)
	}()

	// Wait for graceful stop or force stop after timeout
	select {
	case <-stopped:
		s.logger.InfoContext(ctx, "gRPC server stopped gracefully")
		return nil
	case <-time.After(30 * time.Second):
		s.grpcServer.Stop()
		s.logger.WarnContext(ctx, "gRPC server force stopped after timeout")
		return nil
	}
}

// Handler interfaces (to be implemented)
type FlowServiceHandler interface {
	pb.FlowServiceServer
}
type IndexingServiceHandler interface{}
type ClientOperationsServiceHandler interface{}

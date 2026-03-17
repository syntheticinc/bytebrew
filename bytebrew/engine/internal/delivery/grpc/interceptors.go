package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/syntheticinc/bytebrew/bytebrew/engine/api/proto/gen"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/pkg/errors"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/pkg/logger"
)

// LoggingInterceptor logs all gRPC requests and responses
func LoggingInterceptor(log *logger.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		log.InfoContext(ctx, "gRPC request started",
			"method", info.FullMethod,
		)

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		if err != nil {
			log.ErrorContext(ctx, "gRPC request failed",
				"method", info.FullMethod,
				"duration", duration,
				"error", err,
			)
		} else {
			log.InfoContext(ctx, "gRPC request completed",
				"method", info.FullMethod,
				"duration", duration,
			)
		}

		return resp, err
	}
}

// StreamLoggingInterceptor logs all streaming gRPC requests
func StreamLoggingInterceptor(log *logger.Logger) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		log.InfoContext(ss.Context(), "gRPC stream started",
			"method", info.FullMethod,
			"is_client_stream", info.IsClientStream,
			"is_server_stream", info.IsServerStream,
		)

		err := handler(srv, ss)

		duration := time.Since(start)
		if err != nil {
			log.ErrorContext(ss.Context(), "gRPC stream failed",
				"method", info.FullMethod,
				"duration", duration,
				"error", err,
			)
		} else {
			log.InfoContext(ss.Context(), "gRPC stream completed",
				"method", info.FullMethod,
				"duration", duration,
			)
		}

		return err
	}
}

// RecoveryInterceptor recovers from panics in gRPC handlers
func RecoveryInterceptor(log *logger.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.ErrorContext(ctx, "Panic recovered in gRPC handler",
					"method", info.FullMethod,
					"panic", r,
				)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}

// StreamRecoveryInterceptor recovers from panics in streaming gRPC handlers
func StreamRecoveryInterceptor(log *logger.Logger) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log.ErrorContext(ss.Context(), "Panic recovered in gRPC stream handler",
					"method", info.FullMethod,
					"panic", r,
				)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(srv, ss)
	}
}

// ErrorMappingInterceptor maps domain errors to gRPC status codes
func ErrorMappingInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			return resp, mapDomainErrorToGRPC(err)
		}
		return resp, nil
	}
}

// StreamErrorMappingInterceptor maps domain errors to gRPC status codes for streams
func StreamErrorMappingInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		err := handler(srv, ss)
		if err != nil {
			return mapDomainErrorToGRPC(err)
		}
		return nil
	}
}

// mapDomainErrorToGRPC maps domain errors to gRPC status codes
func mapDomainErrorToGRPC(err error) error {
	code := errors.GetCode(err)

	switch code {
	case errors.CodeInvalidInput:
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.CodeNotFound:
		return status.Error(codes.NotFound, err.Error())
	case errors.CodeAlreadyExists:
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.CodeUnauthorized:
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.CodeForbidden:
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.CodeTimeout:
		return status.Error(codes.DeadlineExceeded, err.Error())
	case errors.CodeUnavailable:
		return status.Error(codes.Unavailable, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

// mapDomainErrorToProto maps domain errors to proto Error message
func mapDomainErrorToProto(err error) *pb.Error {
	if err == nil {
		return nil
	}

	code := errors.GetCode(err)

	return &pb.Error{
		Code:    code,
		Message: err.Error(),
	}
}

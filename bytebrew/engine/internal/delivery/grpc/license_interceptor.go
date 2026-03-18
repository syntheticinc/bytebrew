package grpc

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
)

// LicenseUnaryInterceptor returns a gRPC unary interceptor that checks license status.
// nil licenseInfo (CE mode) -> allow all requests.
// Blocked -> reject with PermissionDenied (LICENSE_REQUIRED).
// Grace -> add warning header, allow.
// Active -> allow.
func LicenseUnaryInterceptor(licenseInfo *domain.LicenseInfo) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if licenseInfo == nil {
			return handler(ctx, req)
		}

		if licenseInfo.Status == domain.LicenseBlocked {
			return nil, status.Error(codes.PermissionDenied, "LICENSE_REQUIRED: valid license needed to use this service")
		}

		if licenseInfo.Status == domain.LicenseGrace {
			if err := grpc.SetHeader(ctx, metadata.Pairs("x-license-warning", "license expiring soon")); err != nil {
				slog.WarnContext(ctx, "failed to set license warning header", "error", err)
			}
		}

		return handler(ctx, req)
	}
}

// LicenseStreamInterceptor returns a gRPC stream interceptor that checks license status.
// nil licenseInfo (CE mode) -> allow all requests.
// Blocked -> reject with PermissionDenied (LICENSE_REQUIRED).
// Grace -> add warning header, allow.
// Active -> allow.
func LicenseStreamInterceptor(licenseInfo *domain.LicenseInfo) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if licenseInfo == nil {
			return handler(srv, ss)
		}

		if licenseInfo.Status == domain.LicenseBlocked {
			return status.Error(codes.PermissionDenied, "LICENSE_REQUIRED: valid license needed to use this service")
		}

		if licenseInfo.Status == domain.LicenseGrace {
			if err := grpc.SetHeader(ss.Context(), metadata.Pairs("x-license-warning", "license expiring soon")); err != nil {
				slog.WarnContext(ss.Context(), "failed to set license warning header", "error", err)
			}
		}

		return handler(srv, ss)
	}
}

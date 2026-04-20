package grpc

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	pluginpkg "github.com/syntheticinc/bytebrew/engine/pkg/plugin"
)

// TenantUnaryInterceptor extracts tenant_id from incoming metadata and puts
// it in the request context. When requireTenant is true, a missing or empty
// tenant_id rejects the call. Otherwise it falls back to domain.CETenantID.
func TenantUnaryInterceptor(verifier pluginpkg.JWTVerifier, requireTenant bool) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		tid, err := extractTenantFromMD(ctx, verifier, requireTenant)
		if err != nil {
			return nil, err
		}
		return handler(domain.WithTenantID(ctx, tid), req)
	}
}

// TenantStreamInterceptor is the streaming equivalent of TenantUnaryInterceptor.
func TenantStreamInterceptor(verifier pluginpkg.JWTVerifier, requireTenant bool) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		tid, err := extractTenantFromMD(ss.Context(), verifier, requireTenant)
		if err != nil {
			return err
		}
		return handler(srv, &wrappedStream{ServerStream: ss, ctx: domain.WithTenantID(ss.Context(), tid)})
	}
}

func extractTenantFromMD(ctx context.Context, verifier pluginpkg.JWTVerifier, requireTenant bool) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		if requireTenant {
			return "", status.Error(codes.Unauthenticated, "missing metadata")
		}
		return domain.CETenantID, nil
	}

	vals := md.Get("authorization")
	if len(vals) == 0 {
		if requireTenant {
			return "", status.Error(codes.Unauthenticated, "missing authorization")
		}
		return domain.CETenantID, nil
	}

	// The metadata value must use the "Bearer <token>" scheme. Previously we
	// relied on strings.TrimPrefix, which returns the original string when the
	// prefix is absent — that meant any non-Bearer value (e.g. "Basic …",
	// "Token …", or a raw token) would be passed to the verifier as-is,
	// silently downgrading the contract at the edge of the system.
	if !strings.HasPrefix(vals[0], "Bearer ") {
		return "", status.Error(codes.Unauthenticated, "invalid authorization scheme")
	}
	token := strings.TrimPrefix(vals[0], "Bearer ")
	if verifier == nil {
		if requireTenant {
			return "", status.Error(codes.Internal, "no verifier configured")
		}
		return domain.CETenantID, nil
	}

	claims, err := verifier.Verify(token)
	if err != nil {
		return "", status.Error(codes.Unauthenticated, err.Error())
	}

	if requireTenant && claims.TenantID == "" {
		return "", status.Error(codes.PermissionDenied, "tenant_id required")
	}

	if claims.TenantID == "" {
		return domain.CETenantID, nil
	}
	return claims.TenantID, nil
}

// wrappedStream overrides the context of a grpc.ServerStream.
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context { return w.ctx }

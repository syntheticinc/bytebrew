package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// mockServerStream implements grpc.ServerStream for testing.
type mockServerStream struct {
	grpc.ServerStream
	ctx     context.Context
	headers metadata.MD
}

func newMockServerStream() *mockServerStream {
	return &mockServerStream{
		ctx: context.Background(),
	}
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}

func (m *mockServerStream) SetHeader(md metadata.MD) error {
	m.headers = metadata.Join(m.headers, md)
	return nil
}

func (m *mockServerStream) SendHeader(md metadata.MD) error {
	m.headers = metadata.Join(m.headers, md)
	return nil
}

func TestLicenseStreamInterceptor(t *testing.T) {
	tests := []struct {
		name           string
		status         domain.LicenseStatus
		wantErr        bool
		wantCode       codes.Code
		wantMsg        string
		wantHeader     bool
		handlerCalled  bool
	}{
		{
			name:          "blocked returns PermissionDenied",
			status:        domain.LicenseBlocked,
			wantErr:       true,
			wantCode:      codes.PermissionDenied,
			wantMsg:       "LICENSE_REQUIRED",
			handlerCalled: false,
		},
		{
			name:          "grace allows with warning header",
			status:        domain.LicenseGrace,
			wantErr:       false,
			wantHeader:    true,
			handlerCalled: true,
		},
		{
			name:          "active allows without warning",
			status:        domain.LicenseActive,
			wantErr:       false,
			wantHeader:    false,
			handlerCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			licenseInfo := &domain.LicenseInfo{
				Status: tt.status,
			}

			interceptor := LicenseStreamInterceptor(licenseInfo)
			handlerCalled := false
			handler := func(srv interface{}, stream grpc.ServerStream) error {
				handlerCalled = true
				return nil
			}

			ss := newMockServerStream()
			// Use grpc.SetHeader-compatible context
			ss.ctx = grpc.NewContextWithServerTransportStream(
				context.Background(),
				&mockServerTransportStream{headers: &ss.headers},
			)

			err := interceptor(nil, ss, &grpc.StreamServerInfo{}, handler)

			if tt.wantErr {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
				assert.Contains(t, st.Message(), tt.wantMsg)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.handlerCalled, handlerCalled)

			if tt.wantHeader {
				assert.Contains(t, ss.headers.Get("x-license-warning"), "license expiring soon")
			}
		})
	}
}

func TestLicenseUnaryInterceptor(t *testing.T) {
	tests := []struct {
		name          string
		status        domain.LicenseStatus
		wantErr       bool
		wantCode      codes.Code
		wantMsg       string
		handlerCalled bool
	}{
		{
			name:          "blocked returns PermissionDenied",
			status:        domain.LicenseBlocked,
			wantErr:       true,
			wantCode:      codes.PermissionDenied,
			wantMsg:       "LICENSE_REQUIRED",
			handlerCalled: false,
		},
		{
			name:          "grace allows request",
			status:        domain.LicenseGrace,
			wantErr:       false,
			handlerCalled: true,
		},
		{
			name:          "active allows request",
			status:        domain.LicenseActive,
			wantErr:       false,
			handlerCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			licenseInfo := &domain.LicenseInfo{
				Status: tt.status,
			}

			interceptor := LicenseUnaryInterceptor(licenseInfo)
			handlerCalled := false
			handler := func(ctx context.Context, req interface{}) (interface{}, error) {
				handlerCalled = true
				return "ok", nil
			}

			ctx := grpc.NewContextWithServerTransportStream(
				context.Background(),
				&mockServerTransportStream{headers: new(metadata.MD)},
			)

			resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)

			if tt.wantErr {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
				assert.Contains(t, st.Message(), tt.wantMsg)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "ok", resp)
			}

			assert.Equal(t, tt.handlerCalled, handlerCalled)
		})
	}
}

// mockServerTransportStream implements grpc.ServerTransportStream for grpc.SetHeader.
type mockServerTransportStream struct {
	headers *metadata.MD
}

func (m *mockServerTransportStream) Method() string { return "/test/Method" }
func (m *mockServerTransportStream) SetHeader(md metadata.MD) error {
	*m.headers = metadata.Join(*m.headers, md)
	return nil
}
func (m *mockServerTransportStream) SendHeader(md metadata.MD) error {
	*m.headers = metadata.Join(*m.headers, md)
	return nil
}
func (m *mockServerTransportStream) SetTrailer(md metadata.MD) error { return nil }

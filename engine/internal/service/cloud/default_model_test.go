package cloud

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockModelProxy struct {
	response string
	err      error
}

func (m *mockModelProxy) Chat(ctx context.Context, messages []map[string]string) (string, error) {
	return m.response, m.err
}

func TestDefaultModelService_Chat(t *testing.T) {
	svc := NewDefaultModelService(
		DefaultModelConfig{ModelName: "GLM-4.7", MaxReqsPerMonth: 100},
		&mockModelProxy{response: "Hello!"},
	)

	result, err := svc.Chat(context.Background(), "tenant-1", []map[string]string{
		{"role": "user", "content": "Hi"},
	})
	require.NoError(t, err)
	assert.Equal(t, "Hello!", result)
	assert.Equal(t, 99, svc.RemainingRequests("tenant-1"))
}

func TestDefaultModelService_RateLimit(t *testing.T) {
	// AC-PRICE-06: 100 req/month limit
	svc := NewDefaultModelService(
		DefaultModelConfig{ModelName: "GLM-4.7", MaxReqsPerMonth: 3},
		&mockModelProxy{response: "ok"},
	)

	// Use 3 requests
	for i := 0; i < 3; i++ {
		_, err := svc.Chat(context.Background(), "tenant-1", nil)
		require.NoError(t, err)
	}

	assert.Equal(t, 0, svc.RemainingRequests("tenant-1"))

	// 4th request should fail
	_, err := svc.Chat(context.Background(), "tenant-1", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "limit reached")
	assert.Contains(t, err.Error(), "Add your own API key")
}

func TestDefaultModelService_TenantIsolation(t *testing.T) {
	svc := NewDefaultModelService(
		DefaultModelConfig{ModelName: "GLM-4.7", MaxReqsPerMonth: 5},
		&mockModelProxy{response: "ok"},
	)

	// Tenant 1 uses 3 requests
	for i := 0; i < 3; i++ {
		_, err := svc.Chat(context.Background(), "tenant-1", nil)
		require.NoError(t, err)
	}

	// Tenant 2 still has full quota
	assert.Equal(t, 5, svc.RemainingRequests("tenant-2"))
	assert.Equal(t, 2, svc.RemainingRequests("tenant-1"))
}

func TestDefaultModelService_ModelName(t *testing.T) {
	svc := NewDefaultModelService(DefaultDefaultModelConfig(), nil)
	assert.Equal(t, "GLM-4.7", svc.ModelName())
}

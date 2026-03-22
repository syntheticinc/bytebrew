package llm

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type retryMockModel struct {
	generateFunc func(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error)
	callCount    int
}

func (m *retryMockModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	m.callCount++
	return m.generateFunc(ctx, input, opts...)
}

func (m *retryMockModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, nil
}

func (m *retryMockModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return m, nil
}

func TestRetryWrapper_SuccessOnFirstTry(t *testing.T) {
	mock := &retryMockModel{
		generateFunc: func(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
			return &schema.Message{Content: "hello"}, nil
		},
	}

	wrapper := NewRetryWrapper(mock, 3, 10*time.Millisecond, 5*time.Second)
	result, err := wrapper.Generate(context.Background(), nil)

	require.NoError(t, err)
	assert.Equal(t, "hello", result.Content)
	assert.Equal(t, 1, mock.callCount)
}

func TestRetryWrapper_RetriableErrorThenSuccess(t *testing.T) {
	callNum := 0
	mock := &retryMockModel{
		generateFunc: func(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
			callNum++
			if callNum < 3 {
				return nil, fmt.Errorf("503 service unavailable")
			}
			return &schema.Message{Content: "recovered"}, nil
		},
	}

	wrapper := NewRetryWrapper(mock, 3, 10*time.Millisecond, 5*time.Second)
	result, err := wrapper.Generate(context.Background(), nil)

	require.NoError(t, err)
	assert.Equal(t, "recovered", result.Content)
	assert.Equal(t, 3, mock.callCount)
}

func TestRetryWrapper_NonRetriableError(t *testing.T) {
	mock := &retryMockModel{
		generateFunc: func(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
			return nil, fmt.Errorf("401 unauthorized")
		},
	}

	wrapper := NewRetryWrapper(mock, 3, 10*time.Millisecond, 5*time.Second)
	_, err := wrapper.Generate(context.Background(), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
	assert.Equal(t, 1, mock.callCount)
}

func TestRetryWrapper_AllRetriesExhausted(t *testing.T) {
	mock := &retryMockModel{
		generateFunc: func(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
			return nil, fmt.Errorf("503 service unavailable")
		},
	}

	wrapper := NewRetryWrapper(mock, 2, 10*time.Millisecond, 5*time.Second)
	_, err := wrapper.Generate(context.Background(), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "all 3 retries failed")
	assert.Equal(t, 3, mock.callCount) // initial + 2 retries
}

func TestRetryWrapper_ContextCancelled(t *testing.T) {
	mock := &retryMockModel{
		generateFunc: func(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
			return nil, fmt.Errorf("503 service unavailable")
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	wrapper := NewRetryWrapper(mock, 3, time.Second, 5*time.Second)
	_, err := wrapper.Generate(ctx, nil)

	require.Error(t, err)
	// Should fail fast due to cancelled context, not exhaust all retries
	assert.True(t, mock.callCount <= 2)
}

func TestRetryWrapper_WithTools(t *testing.T) {
	mock := &retryMockModel{
		generateFunc: func(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
			return &schema.Message{Content: "ok"}, nil
		},
	}

	wrapper := NewRetryWrapper(mock, 2, 10*time.Millisecond, 5*time.Second)
	newWrapper, err := wrapper.WithTools([]*schema.ToolInfo{{Name: "test"}})

	require.NoError(t, err)
	require.NotNil(t, newWrapper)

	// Verify it's a RetryWrapper with same config
	rw, ok := newWrapper.(*RetryWrapper)
	require.True(t, ok)
	assert.Equal(t, 2, rw.maxRetries)
}

func TestRetryWrapper_StreamDelegatesToInner(t *testing.T) {
	mock := &retryMockModel{}
	wrapper := NewRetryWrapper(mock, 3, 10*time.Millisecond, 5*time.Second)

	reader, err := wrapper.Stream(context.Background(), nil)
	require.NoError(t, err)
	assert.Nil(t, reader) // mock returns nil
}

func TestIsRetriable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"503 error", fmt.Errorf("503 service unavailable"), true},
		{"429 rate limit", fmt.Errorf("429 too many requests"), true},
		{"502 bad gateway", fmt.Errorf("502 bad gateway"), true},
		{"timeout", fmt.Errorf("request timeout"), true},
		{"deadline exceeded", fmt.Errorf("context deadline exceeded"), true},
		{"connection refused", fmt.Errorf("connection refused"), true},
		{"connection reset", fmt.Errorf("connection reset by peer"), true},
		{"eof", fmt.Errorf("unexpected EOF"), true},
		{"401 unauthorized", fmt.Errorf("401 unauthorized"), false},
		{"403 forbidden", fmt.Errorf("403 forbidden"), false},
		{"404 not found", fmt.Errorf("404 not found"), false},
		{"400 bad request", fmt.Errorf("400 bad request"), false},
		{"invalid request", fmt.Errorf("invalid request body"), false},
		{"unknown error", fmt.Errorf("something went wrong"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isRetriable(tt.err))
		})
	}
}

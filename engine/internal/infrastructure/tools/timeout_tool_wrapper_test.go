package tools

import (
	"context"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// slowTool is a test tool that responds after a configurable delay.
type slowTool struct {
	delay time.Duration
	name  string
}

func (t *slowTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: t.name}, nil
}

func (t *slowTool) InvokableRun(ctx context.Context, _ string, _ ...tool.Option) (string, error) {
	select {
	case <-time.After(t.delay):
		return "ok", nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// TC-RESIL-04: tool completes within timeout → no error
func TestTimeoutToolWrapper_CompletesWithinTimeout(t *testing.T) {
	inner := &slowTool{delay: 10 * time.Millisecond, name: "fast-tool"}
	wrapped := NewTimeoutToolWrapper(inner, 200) // 200ms timeout

	result, err := wrapped.InvokableRun(context.Background(), "{}")
	require.NoError(t, err)
	assert.Equal(t, "ok", result)
}

// TC-RESIL-04: tool exceeds timeout → structured tool_timeout error
func TestTimeoutToolWrapper_TimeoutExceeded(t *testing.T) {
	inner := &slowTool{delay: 500 * time.Millisecond, name: "slow-mcp-tool"}
	wrapped := NewTimeoutToolWrapper(inner, 50) // 50ms timeout

	_, err := wrapped.InvokableRun(context.Background(), "{}")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tool_timeout")
	assert.Contains(t, err.Error(), "slow-mcp-tool")
	assert.Contains(t, err.Error(), "50ms")
}

// TC-RESIL-04: Info passthrough works
func TestTimeoutToolWrapper_InfoPassthrough(t *testing.T) {
	inner := &slowTool{name: "my-tool"}
	wrapped := NewTimeoutToolWrapper(inner, 1000)

	info, err := wrapped.Info(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "my-tool", info.Name)
}

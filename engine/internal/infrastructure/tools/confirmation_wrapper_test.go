package tools

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockConfirmationRequester implements ConfirmationRequester for testing.
type mockConfirmationRequester struct {
	confirmed bool
	err       error

	lastToolName string
	lastArgs     string
}

func (m *mockConfirmationRequester) RequestConfirmation(ctx context.Context, toolName string, args string) (bool, error) {
	m.lastToolName = toolName
	m.lastArgs = args
	return m.confirmed, m.err
}

// mockInnerTool implements tool.InvokableTool for testing.
type mockInnerTool struct {
	name   string
	result string
	err    error
	called bool
}

func (m *mockInnerTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: m.name,
		Desc: "mock tool",
	}, nil
}

func (m *mockInnerTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	m.called = true
	return m.result, m.err
}

func TestConfirmationWrapper_Confirmed(t *testing.T) {
	inner := &mockInnerTool{name: "write_file", result: "file written"}
	requester := &mockConfirmationRequester{confirmed: true}
	wrapped := NewConfirmationWrapper(inner, requester)

	result, err := wrapped.InvokableRun(context.Background(), `{"path":"main.go"}`)
	require.NoError(t, err)
	assert.Equal(t, "file written", result)
	assert.True(t, inner.called)
	assert.Equal(t, "write_file", requester.lastToolName)
	assert.Equal(t, `{"path":"main.go"}`, requester.lastArgs)
}

func TestConfirmationWrapper_Denied(t *testing.T) {
	inner := &mockInnerTool{name: "delete_file", result: "deleted"}
	requester := &mockConfirmationRequester{confirmed: false}
	wrapped := NewConfirmationWrapper(inner, requester)

	result, err := wrapped.InvokableRun(context.Background(), `{"path":"important.go"}`)
	require.NoError(t, err)
	assert.Contains(t, result, "cancelled by user")
	assert.Contains(t, result, "delete_file")
	assert.False(t, inner.called)
}

func TestConfirmationWrapper_RequesterError(t *testing.T) {
	inner := &mockInnerTool{name: "execute_command"}
	requester := &mockConfirmationRequester{err: fmt.Errorf("connection lost")}
	wrapped := NewConfirmationWrapper(inner, requester)

	_, err := wrapped.InvokableRun(context.Background(), `{"cmd":"rm -rf /"}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "request confirmation")
	assert.False(t, inner.called)
}

func TestConfirmationWrapper_Info_DelegatesToInner(t *testing.T) {
	inner := &mockInnerTool{name: "my_tool"}
	requester := &mockConfirmationRequester{}
	wrapped := NewConfirmationWrapper(inner, requester)

	info, err := wrapped.Info(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "my_tool", info.Name)
}

func TestConfirmationWrapper_InnerToolError(t *testing.T) {
	inner := &mockInnerTool{
		name: "write_file",
		err:  fmt.Errorf("disk full"),
	}
	requester := &mockConfirmationRequester{confirmed: true}
	wrapped := NewConfirmationWrapper(inner, requester)

	_, err := wrapped.InvokableRun(context.Background(), `{}`)
	require.Error(t, err)
	assert.Equal(t, "disk full", err.Error())
	assert.True(t, inner.called)
}

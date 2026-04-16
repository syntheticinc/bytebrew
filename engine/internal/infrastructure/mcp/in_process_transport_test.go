package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInProcessTransport_ImplementsTransport(t *testing.T) {
	var _ Transport = (*InProcessTransport)(nil)
}

func TestNewInProcessTransport_NilHandler_ReturnsError(t *testing.T) {
	_, err := NewInProcessTransport(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "handler must not be nil")
}

func TestInProcessTransport_Start_NoOp(t *testing.T) {
	transport, err := NewInProcessTransport(func(_ context.Context, _ *Request) (*Response, error) {
		return nil, nil
	})
	require.NoError(t, err)
	require.NotNil(t, transport)
	err = transport.Start(context.Background())
	require.NoError(t, err)
}

func TestInProcessTransport_Close_NoOp(t *testing.T) {
	transport, err := NewInProcessTransport(func(_ context.Context, _ *Request) (*Response, error) {
		return nil, nil
	})
	require.NoError(t, err)
	err = transport.Close()
	require.NoError(t, err)
}

func TestInProcessTransport_Send_RoutesToHandler(t *testing.T) {
	expectedResult, _ := json.Marshal(map[string]string{"status": "ok"})

	transport, err := NewInProcessTransport(func(_ context.Context, req *Request) (*Response, error) {
		assert.Equal(t, "test/method", req.Method)
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  expectedResult,
		}, nil
	})
	require.NoError(t, err)

	resp, err := transport.Send(context.Background(), &Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test/method",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.JSONEq(t, `{"status":"ok"}`, string(resp.Result))
}

func TestInProcessTransport_Send_NilRequest(t *testing.T) {
	transport, err := NewInProcessTransport(func(_ context.Context, _ *Request) (*Response, error) {
		return nil, nil
	})
	require.NoError(t, err)
	_, err = transport.Send(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "request must not be nil")
}

func TestInProcessTransport_Send_HandlerError(t *testing.T) {
	transport, err := NewInProcessTransport(func(_ context.Context, _ *Request) (*Response, error) {
		return nil, fmt.Errorf("handler failed")
	})
	require.NoError(t, err)

	resp, err := transport.Send(context.Background(), &Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "failing/method",
	})
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "handler failed")
}

func TestInProcessTransport_Send_PassesContext(t *testing.T) {
	type ctxKey string
	key := ctxKey("test-key")

	transport, err := NewInProcessTransport(func(ctx context.Context, _ *Request) (*Response, error) {
		val, ok := ctx.Value(key).(string)
		assert.True(t, ok)
		assert.Equal(t, "test-value", val)
		return &Response{JSONRPC: "2.0", ID: 1}, nil
	})
	require.NoError(t, err)

	ctx := context.WithValue(context.Background(), key, "test-value")
	resp, err := transport.Send(ctx, &Request{JSONRPC: "2.0", ID: 1, Method: "ctx/test"})
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestInProcessTransport_Notify_NoOp(t *testing.T) {
	called := false
	transport, err := NewInProcessTransport(func(_ context.Context, _ *Request) (*Response, error) {
		called = true
		return nil, nil
	})
	require.NoError(t, err)

	// Notify should not call the handler
	transport.Notify(context.Background(), &Request{
		JSONRPC: "2.0",
		Method:  "notifications/test",
	})
	assert.False(t, called)
}

func TestInProcessTransport_WorksWithClient(t *testing.T) {
	// Verify the transport integrates correctly with Client.Connect() flow:
	// Client sends: initialize → notifications/initialized (notify) → tools/list
	callCount := 0

	transport, err := NewInProcessTransport(func(_ context.Context, req *Request) (*Response, error) {
		callCount++
		switch req.Method {
		case "initialize":
			result, _ := json.Marshal(map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]interface{}{},
				"serverInfo": map[string]interface{}{
					"name":    "test-server",
					"version": "1.0.0",
				},
			})
			return &Response{JSONRPC: "2.0", ID: req.ID, Result: result}, nil
		case "tools/list":
			result, _ := json.Marshal(ToolsListResult{
				Tools: []MCPTool{
					{Name: "test_tool", Description: "A test tool"},
				},
			})
			return &Response{JSONRPC: "2.0", ID: req.ID, Result: result}, nil
		default:
			return nil, fmt.Errorf("unexpected method: %s", req.Method)
		}
	})
	require.NoError(t, err)

	client := NewClient("test-server", transport)
	err = client.Connect(context.Background())
	require.NoError(t, err)

	assert.True(t, client.IsConnected())
	assert.Equal(t, 2, callCount) // initialize + tools/list (notify doesn't call handler)

	tools := client.ListTools()
	require.Len(t, tools, 1)
	assert.Equal(t, "test_tool", tools[0].Name)

	require.NoError(t, client.Close())
}

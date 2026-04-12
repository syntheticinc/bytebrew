package tools

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEscalationHandler implements EscalationHandler for testing.
type mockEscalationHandler struct {
	lastSessionID string
	lastAgentName string
	lastReason    string
	result        string
	err           error
}

func (m *mockEscalationHandler) Escalate(ctx context.Context, sessionID, agentName, reason string) (string, error) {
	m.lastSessionID = sessionID
	m.lastAgentName = agentName
	m.lastReason = reason
	return m.result, m.err
}

func TestEscalateTool_Info(t *testing.T) {
	handler := &mockEscalationHandler{}
	tool := NewEscalateTool("sess-1", "test-agent", handler)

	info, err := tool.Info(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "escalate", info.Name)
	assert.Contains(t, info.Desc, "human")
}

func TestEscalateTool_InvokableRun_Success(t *testing.T) {
	handler := &mockEscalationHandler{
		result: "Escalation triggered: transfer_to_user",
	}
	tool := NewEscalateTool("sess-1", "test-agent", handler)

	result, err := tool.InvokableRun(context.Background(), `{"reason":"billing issue"}`)
	require.NoError(t, err)
	assert.Contains(t, result, "transfer_to_user")
	assert.Equal(t, "sess-1", handler.lastSessionID)
	assert.Equal(t, "test-agent", handler.lastAgentName)
	assert.Equal(t, "billing issue", handler.lastReason)
}

func TestEscalateTool_InvokableRun_EmptyReason(t *testing.T) {
	handler := &mockEscalationHandler{}
	tool := NewEscalateTool("sess-1", "test-agent", handler)

	result, err := tool.InvokableRun(context.Background(), `{"reason":""}`)
	require.NoError(t, err)
	assert.Contains(t, result, "[ERROR]")
	assert.Contains(t, result, "reason is required")
}

func TestEscalateTool_InvokableRun_InvalidJSON(t *testing.T) {
	handler := &mockEscalationHandler{}
	tool := NewEscalateTool("sess-1", "test-agent", handler)

	result, err := tool.InvokableRun(context.Background(), `{bad json}`)
	require.NoError(t, err)
	assert.Contains(t, result, "[ERROR]")
	assert.Contains(t, result, "Invalid arguments")
}

func TestEscalateTool_InvokableRun_HandlerError(t *testing.T) {
	handler := &mockEscalationHandler{
		err: fmt.Errorf("capability not found"),
	}
	tool := NewEscalateTool("sess-1", "test-agent", handler)

	result, err := tool.InvokableRun(context.Background(), `{"reason":"help needed"}`)
	require.NoError(t, err)
	assert.Contains(t, result, "[ERROR]")
	assert.Contains(t, result, "capability not found")
}

func TestEscalateTool_Factory_NilHandler(t *testing.T) {
	store := NewBuiltinToolStore()
	RegisterAllBuiltins(store)

	factory, ok := store.Get("escalate")
	require.True(t, ok, "escalate factory should be registered")

	// With nil handler → nil tool
	tool := factory(ToolDependencies{})
	assert.Nil(t, tool)
}

func TestEscalateTool_Factory_WithHandler(t *testing.T) {
	store := NewBuiltinToolStore()
	RegisterAllBuiltins(store)

	factory, ok := store.Get("escalate")
	require.True(t, ok)

	handler := &mockEscalationHandler{result: "ok"}
	tool := factory(ToolDependencies{
		SessionID:         "s1",
		AgentName:         "a1",
		EscalationHandler: handler,
	})
	require.NotNil(t, tool)

	info, _ := tool.Info(context.Background())
	assert.Equal(t, "escalate", info.Name)
}

package policy

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

type mockAuditWriter struct {
	entries []string
}

func (m *mockAuditWriter) WriteAudit(ctx context.Context, agentName, action, detail string) {
	m.entries = append(m.entries, fmt.Sprintf("%s: %s - %s", agentName, action, detail))
}

type mockWebhookSender struct {
	calls []string
	err   error
}

func (m *mockWebhookSender) Send(ctx context.Context, url string, payload map[string]interface{}, auth domain.MCPAuthConfig) error {
	m.calls = append(m.calls, url)
	return m.err
}

func TestEngine_BlockAction(t *testing.T) {
	// AC-POL-04: block action blocks tool execution with message
	rules := []*domain.PolicyRule{
		{
			AgentName: "support",
			Condition: domain.PolicyCondition{Type: domain.PolicyCondBeforeToolCall},
			Action: domain.PolicyAction{
				Type:    domain.PolicyActionBlock,
				Message: "dangerous tool blocked",
			},
			Enabled: true,
		},
	}
	engine := New(rules, nil, nil)

	result := engine.EvaluateBefore(context.Background(), ToolCallContext{
		AgentName: "support",
		ToolName:  "execute_command",
	})
	assert.True(t, result.Blocked)
	assert.Equal(t, "dangerous tool blocked", result.BlockMessage)
}

func TestEngine_InjectHeader(t *testing.T) {
	// AC-POL-02: inject_header adds headers to MCP requests
	rules := []*domain.PolicyRule{
		{
			AgentName: "support",
			Condition: domain.PolicyCondition{Type: domain.PolicyCondBeforeToolCall},
			Action: domain.PolicyAction{
				Type:    domain.PolicyActionInjectHeader,
				Headers: map[string]string{"X-Tenant-ID": "tenant-42"},
			},
			Enabled: true,
		},
	}
	engine := New(rules, nil, nil)

	result := engine.EvaluateBefore(context.Background(), ToolCallContext{
		AgentName: "support",
		ToolName:  "search_knowledge",
	})
	assert.False(t, result.Blocked)
	assert.Equal(t, "tenant-42", result.InjectedHeaders["X-Tenant-ID"])
}

func TestEngine_ToolMatches(t *testing.T) {
	rules := []*domain.PolicyRule{
		{
			AgentName: "support",
			Condition: domain.PolicyCondition{
				Type:    domain.PolicyCondToolMatches,
				Pattern: "delete_*",
			},
			Action:  domain.PolicyAction{Type: domain.PolicyActionBlock, Message: "delete blocked"},
			Enabled: true,
		},
	}
	engine := New(rules, nil, nil)

	// Matches
	result := engine.EvaluateBefore(context.Background(), ToolCallContext{
		AgentName: "support",
		ToolName:  "delete_user",
	})
	assert.True(t, result.Blocked)

	// Does not match
	result = engine.EvaluateBefore(context.Background(), ToolCallContext{
		AgentName: "support",
		ToolName:  "read_file",
	})
	assert.False(t, result.Blocked)
}

func TestEngine_WriteAudit(t *testing.T) {
	audit := &mockAuditWriter{}
	rules := []*domain.PolicyRule{
		{
			AgentName: "support",
			Condition: domain.PolicyCondition{Type: domain.PolicyCondAfterToolCall},
			Action:    domain.PolicyAction{Type: domain.PolicyActionWriteAudit},
			Enabled:   true,
		},
	}
	engine := New(rules, audit, nil)

	engine.EvaluateAfter(context.Background(), ToolCallContext{
		AgentName: "support",
		ToolName:  "search_knowledge",
		Arguments: `{"query":"test"}`,
	})
	assert.Len(t, audit.entries, 1)
	assert.Contains(t, audit.entries[0], "policy_triggered")
}

func TestEngine_WebhookNotify(t *testing.T) {
	// AC-POL-03: webhook uses same auth types
	webhook := &mockWebhookSender{}
	rules := []*domain.PolicyRule{
		{
			AgentName: "support",
			Condition: domain.PolicyCondition{Type: domain.PolicyCondAfterToolCall},
			Action: domain.PolicyAction{
				Type:       domain.PolicyActionNotify,
				WebhookURL: "https://hooks.example.com/notify",
				AuthConfig: domain.MCPAuthConfig{Type: domain.MCPAuthAPIKey, KeyEnv: "WEBHOOK_KEY"},
			},
			Enabled: true,
		},
	}
	engine := New(rules, nil, webhook)

	engine.EvaluateAfter(context.Background(), ToolCallContext{
		AgentName: "support",
		ToolName:  "execute_command",
	})
	assert.Len(t, webhook.calls, 1)
	assert.Equal(t, "https://hooks.example.com/notify", webhook.calls[0])
}

func TestEngine_ErrorOccurred(t *testing.T) {
	audit := &mockAuditWriter{}
	rules := []*domain.PolicyRule{
		{
			AgentName: "support",
			Condition: domain.PolicyCondition{Type: domain.PolicyCondErrorOccurred},
			Action:    domain.PolicyAction{Type: domain.PolicyActionWriteAudit},
			Enabled:   true,
		},
	}
	engine := New(rules, audit, nil)

	// No error — rule should not fire
	engine.EvaluateAfter(context.Background(), ToolCallContext{
		AgentName: "support",
		ToolName:  "read_file",
	})
	assert.Len(t, audit.entries, 0)

	// With error — rule fires
	engine.EvaluateAfter(context.Background(), ToolCallContext{
		AgentName: "support",
		ToolName:  "read_file",
		Error:     fmt.Errorf("file not found"),
	})
	assert.Len(t, audit.entries, 1)
}

func TestEngine_TimeRange(t *testing.T) {
	rules := []*domain.PolicyRule{
		{
			AgentName: "support",
			Condition: domain.PolicyCondition{
				Type:  domain.PolicyCondTimeRange,
				Start: "09:00",
				End:   "17:00",
			},
			Action:  domain.PolicyAction{Type: domain.PolicyActionBlock, Message: "business hours only"},
			Enabled: true,
		},
	}
	engine := New(rules, nil, nil)

	// Within range
	inRange := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	result := engine.EvaluateBefore(context.Background(), ToolCallContext{
		AgentName: "support",
		ToolName:  "read_file",
		Timestamp: inRange,
	})
	assert.True(t, result.Blocked)

	// Outside range
	outRange := time.Date(2026, 1, 1, 20, 0, 0, 0, time.UTC)
	result = engine.EvaluateBefore(context.Background(), ToolCallContext{
		AgentName: "support",
		ToolName:  "read_file",
		Timestamp: outRange,
	})
	assert.False(t, result.Blocked)
}

func TestEngine_DisabledRuleSkipped(t *testing.T) {
	rules := []*domain.PolicyRule{
		{
			AgentName: "support",
			Condition: domain.PolicyCondition{Type: domain.PolicyCondBeforeToolCall},
			Action:    domain.PolicyAction{Type: domain.PolicyActionBlock},
			Enabled:   false, // disabled
		},
	}
	engine := New(rules, nil, nil)

	result := engine.EvaluateBefore(context.Background(), ToolCallContext{
		AgentName: "support",
		ToolName:  "anything",
	})
	assert.False(t, result.Blocked)
}

func TestEngine_DifferentAgentSkipped(t *testing.T) {
	rules := []*domain.PolicyRule{
		{
			AgentName: "admin",
			Condition: domain.PolicyCondition{Type: domain.PolicyCondBeforeToolCall},
			Action:    domain.PolicyAction{Type: domain.PolicyActionBlock},
			Enabled:   true,
		},
	}
	engine := New(rules, nil, nil)

	result := engine.EvaluateBefore(context.Background(), ToolCallContext{
		AgentName: "support", // different agent
		ToolName:  "anything",
	})
	assert.False(t, result.Blocked)
}

package policy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// TestPolicyEngine_Block_PreventsTool verifies that a block rule with
// tool_matches condition blocks matching tool calls.
func TestPolicyEngine_Block_PreventsTool(t *testing.T) {
	rules := []*domain.PolicyRule{
		{
			AgentName: "agent1",
			Condition: domain.PolicyCondition{
				Type:    domain.PolicyCondToolMatches,
				Pattern: "delete_*",
			},
			Action: domain.PolicyAction{
				Type:    domain.PolicyActionBlock,
				Message: "deletion not allowed",
			},
			Enabled: true,
		},
	}
	engine := New(rules, nil, nil)

	result := engine.EvaluateBefore(context.Background(), ToolCallContext{
		AgentName: "agent1",
		ToolName:  "delete_user",
	})

	assert.True(t, result.Blocked)
	assert.Contains(t, result.BlockMessage, "deletion not allowed")
}

// TestPolicyEngine_InjectHeader verifies that inject_header action
// adds custom headers to the result.
func TestPolicyEngine_InjectHeader(t *testing.T) {
	rules := []*domain.PolicyRule{
		{
			AgentName: "agent1",
			Condition: domain.PolicyCondition{Type: domain.PolicyCondBeforeToolCall},
			Action: domain.PolicyAction{
				Type:    domain.PolicyActionInjectHeader,
				Headers: map[string]string{"X-Tenant": "abc"},
			},
			Enabled: true,
		},
	}
	engine := New(rules, nil, nil)

	result := engine.EvaluateBefore(context.Background(), ToolCallContext{
		AgentName: "agent1",
		ToolName:  "search_knowledge",
	})

	assert.False(t, result.Blocked)
	assert.Equal(t, "abc", result.InjectedHeaders["X-Tenant"])
}

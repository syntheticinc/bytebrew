package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicyConditionType_IsValid(t *testing.T) {
	tests := []struct {
		ct    PolicyConditionType
		valid bool
	}{
		{PolicyCondBeforeToolCall, true},
		{PolicyCondAfterToolCall, true},
		{PolicyCondToolMatches, true},
		{PolicyCondTimeRange, true},
		{PolicyCondErrorOccurred, true},
		{PolicyConditionType("free_text"), false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.valid, tt.ct.IsValid(), string(tt.ct))
	}
}

func TestPolicyActionType_IsValid(t *testing.T) {
	tests := []struct {
		at    PolicyActionType
		valid bool
	}{
		{PolicyActionBlock, true},
		{PolicyActionLogToWebhook, true},
		{PolicyActionNotify, true},
		{PolicyActionInjectHeader, true},
		{PolicyActionWriteAudit, true},
		{PolicyActionType("custom"), false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.valid, tt.at.IsValid(), string(tt.at))
	}
}

func TestPolicyRule_Validate(t *testing.T) {
	tests := []struct {
		name    string
		rule    PolicyRule
		wantErr bool
	}{
		{
			"valid block",
			PolicyRule{
				AgentName: "test",
				Condition: PolicyCondition{Type: PolicyCondBeforeToolCall},
				Action:    PolicyAction{Type: PolicyActionBlock, Message: "blocked"},
			},
			false,
		},
		{
			"valid tool_matches with pattern",
			PolicyRule{
				AgentName: "test",
				Condition: PolicyCondition{Type: PolicyCondToolMatches, Pattern: "delete_*"},
				Action:    PolicyAction{Type: PolicyActionBlock},
			},
			false,
		},
		{
			"tool_matches without pattern",
			PolicyRule{
				AgentName: "test",
				Condition: PolicyCondition{Type: PolicyCondToolMatches},
				Action:    PolicyAction{Type: PolicyActionBlock},
			},
			true,
		},
		{
			"empty agent_name",
			PolicyRule{
				Condition: PolicyCondition{Type: PolicyCondBeforeToolCall},
				Action:    PolicyAction{Type: PolicyActionBlock},
			},
			true,
		},
		{
			"invalid condition",
			PolicyRule{
				AgentName: "test",
				Condition: PolicyCondition{Type: "custom_cond"},
				Action:    PolicyAction{Type: PolicyActionBlock},
			},
			true,
		},
		{
			"invalid action",
			PolicyRule{
				AgentName: "test",
				Condition: PolicyCondition{Type: PolicyCondBeforeToolCall},
				Action:    PolicyAction{Type: "custom_action"},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

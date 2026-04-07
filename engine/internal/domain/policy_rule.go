package domain

import "fmt"

// PolicyConditionType represents a typed policy condition (AC-POL-01: dropdown, not free text).
type PolicyConditionType string

const (
	PolicyCondBeforeToolCall PolicyConditionType = "before_tool_call"
	PolicyCondAfterToolCall  PolicyConditionType = "after_tool_call"
	PolicyCondToolMatches    PolicyConditionType = "tool_matches"
	PolicyCondTimeRange      PolicyConditionType = "time_range"
	PolicyCondErrorOccurred  PolicyConditionType = "error_occurred"
)

// IsValid returns true if the condition type is recognized.
func (c PolicyConditionType) IsValid() bool {
	switch c {
	case PolicyCondBeforeToolCall, PolicyCondAfterToolCall,
		PolicyCondToolMatches, PolicyCondTimeRange, PolicyCondErrorOccurred:
		return true
	}
	return false
}

// PolicyActionType represents a policy action to execute.
type PolicyActionType string

const (
	PolicyActionBlock        PolicyActionType = "block"
	PolicyActionLogToWebhook PolicyActionType = "log_to_webhook"
	PolicyActionNotify       PolicyActionType = "notify"
	PolicyActionInjectHeader PolicyActionType = "inject_header" // AC-POL-02
	PolicyActionWriteAudit   PolicyActionType = "write_audit"
)

// IsValid returns true if the action type is recognized.
func (a PolicyActionType) IsValid() bool {
	switch a {
	case PolicyActionBlock, PolicyActionLogToWebhook, PolicyActionNotify,
		PolicyActionInjectHeader, PolicyActionWriteAudit:
		return true
	}
	return false
}

// PolicyCondition defines a condition that triggers a policy.
type PolicyCondition struct {
	Type    PolicyConditionType
	Pattern string // for tool_matches: glob pattern (e.g. "delete_*")
	Start   string // for time_range: HH:MM
	End     string // for time_range: HH:MM
}

// PolicyAction defines an action to execute when a policy fires.
type PolicyAction struct {
	Type       PolicyActionType
	Message    string            // for block: message to agent
	WebhookURL string            // for log_to_webhook / notify
	AuthType   MCPAuthType       // webhook auth type (AC-POL-03)
	AuthConfig MCPAuthConfig     // webhook auth config
	Headers    map[string]string // for inject_header: key-value pairs (AC-POL-02)
}

// PermissionLevel represents the agent permission level.
type PermissionLevel string

const (
	PermissionStandard   PermissionLevel = "standard"
	PermissionRestricted PermissionLevel = "restricted"
	PermissionCustom     PermissionLevel = "custom"
)

// PolicyRule defines a "When [condition] → Do [action]" rule.
type PolicyRule struct {
	ID        string
	AgentName string
	Condition PolicyCondition
	Action    PolicyAction
	Enabled   bool
}

// Validate validates the PolicyRule.
func (r *PolicyRule) Validate() error {
	if r.AgentName == "" {
		return fmt.Errorf("agent_name is required")
	}
	if !r.Condition.Type.IsValid() {
		return fmt.Errorf("invalid condition type: %s", r.Condition.Type)
	}
	if !r.Action.Type.IsValid() {
		return fmt.Errorf("invalid action type: %s", r.Action.Type)
	}
	if r.Condition.Type == PolicyCondToolMatches && r.Pattern() == "" {
		return fmt.Errorf("tool_matches condition requires a pattern")
	}
	return nil
}

// Pattern returns the condition's pattern (convenience accessor).
func (r *PolicyRule) Pattern() string {
	return r.Condition.Pattern
}

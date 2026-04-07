package policy

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// AuditWriter writes audit log entries (AC-POL write_audit action).
type AuditWriter interface {
	WriteAudit(ctx context.Context, agentName, action, detail string)
}

// WebhookSender sends webhook notifications (AC-POL-03 uses same auth types).
type WebhookSender interface {
	Send(ctx context.Context, url string, payload map[string]interface{}, authConfig domain.MCPAuthConfig) error
}

// ToolCallContext holds context for evaluating policies around a tool call.
type ToolCallContext struct {
	AgentName string
	ToolName  string
	Arguments string
	Result    string // only set for after_tool_call
	Error     error  // only set if tool errored
	Timestamp time.Time
}

// EvaluationResult holds the result of policy evaluation.
type EvaluationResult struct {
	Blocked       bool
	BlockMessage  string
	InjectedHeaders map[string]string // AC-POL-02: headers to inject into MCP requests
}

// Engine evaluates policy rules against tool calls.
type Engine struct {
	rules       []*domain.PolicyRule
	auditWriter AuditWriter
	webhook     WebhookSender
}

// New creates a new policy engine.
func New(rules []*domain.PolicyRule, auditWriter AuditWriter, webhook WebhookSender) *Engine {
	return &Engine{
		rules:       rules,
		auditWriter: auditWriter,
		webhook:     webhook,
	}
}

// EvaluateBefore evaluates before_tool_call policies.
// Returns a result that may block the tool call (AC-POL-04).
func (e *Engine) EvaluateBefore(ctx context.Context, tc ToolCallContext) EvaluationResult {
	result := EvaluationResult{
		InjectedHeaders: make(map[string]string),
	}

	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}
		if rule.AgentName != tc.AgentName {
			continue
		}
		if !e.matchesCondition(rule.Condition, tc, true) {
			continue
		}

		e.applyAction(ctx, rule, tc, &result)
		if result.Blocked {
			return result // short-circuit on block
		}
	}
	return result
}

// EvaluateAfter evaluates after_tool_call policies.
func (e *Engine) EvaluateAfter(ctx context.Context, tc ToolCallContext) {
	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}
		if rule.AgentName != tc.AgentName {
			continue
		}
		if !e.matchesCondition(rule.Condition, tc, false) {
			continue
		}

		var result EvaluationResult
		e.applyAction(ctx, rule, tc, &result)
	}
}

// matchesCondition checks if a rule condition matches the tool call context.
func (e *Engine) matchesCondition(cond domain.PolicyCondition, tc ToolCallContext, isBefore bool) bool {
	switch cond.Type {
	case domain.PolicyCondBeforeToolCall:
		return isBefore

	case domain.PolicyCondAfterToolCall:
		return !isBefore

	case domain.PolicyCondToolMatches:
		matched, _ := filepath.Match(cond.Pattern, tc.ToolName)
		return matched

	case domain.PolicyCondTimeRange:
		now := tc.Timestamp
		if now.IsZero() {
			now = time.Now()
		}
		return e.inTimeRange(now, cond.Start, cond.End)

	case domain.PolicyCondErrorOccurred:
		return tc.Error != nil
	}
	return false
}

// applyAction executes the policy action.
func (e *Engine) applyAction(ctx context.Context, rule *domain.PolicyRule, tc ToolCallContext, result *EvaluationResult) {
	action := rule.Action

	switch action.Type {
	case domain.PolicyActionBlock:
		// AC-POL-04: block tool execution with message to agent
		result.Blocked = true
		result.BlockMessage = action.Message
		if result.BlockMessage == "" {
			result.BlockMessage = fmt.Sprintf("tool %q blocked by policy", tc.ToolName)
		}
		slog.InfoContext(ctx, "[Policy] tool blocked",
			"agent", tc.AgentName, "tool", tc.ToolName, "message", result.BlockMessage)

	case domain.PolicyActionInjectHeader:
		// AC-POL-02: inject custom headers into MCP tool requests
		for k, v := range action.Headers {
			result.InjectedHeaders[k] = v
		}

	case domain.PolicyActionWriteAudit:
		if e.auditWriter != nil {
			detail := fmt.Sprintf("tool=%s, args=%s", tc.ToolName, tc.Arguments)
			e.auditWriter.WriteAudit(ctx, tc.AgentName, "policy_triggered", detail)
		}

	case domain.PolicyActionLogToWebhook, domain.PolicyActionNotify:
		if e.webhook != nil {
			payload := map[string]interface{}{
				"agent":     tc.AgentName,
				"tool":      tc.ToolName,
				"action":    string(action.Type),
				"timestamp": time.Now().Format(time.RFC3339),
			}
			if tc.Error != nil {
				payload["error"] = tc.Error.Error()
			}
			if err := e.webhook.Send(ctx, action.WebhookURL, payload, action.AuthConfig); err != nil {
				slog.ErrorContext(ctx, "[Policy] webhook send failed",
					"url", action.WebhookURL, "error", err)
			}
		}
	}
}

// inTimeRange checks if the current time is within a HH:MM range.
func (e *Engine) inTimeRange(now time.Time, start, end string) bool {
	startTime, err := time.Parse("15:04", start)
	if err != nil {
		return false
	}
	endTime, err := time.Parse("15:04", end)
	if err != nil {
		return false
	}

	nowMinutes := now.Hour()*60 + now.Minute()
	startMinutes := startTime.Hour()*60 + startTime.Minute()
	endMinutes := endTime.Hour()*60 + endTime.Minute()

	if startMinutes <= endMinutes {
		return nowMinutes >= startMinutes && nowMinutes <= endMinutes
	}
	// Wraps around midnight
	return nowMinutes >= startMinutes || nowMinutes <= endMinutes
}

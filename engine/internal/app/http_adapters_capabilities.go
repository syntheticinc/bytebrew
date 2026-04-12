package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/tools"
	"github.com/syntheticinc/bytebrew/engine/internal/service/policy"
	"github.com/syntheticinc/bytebrew/engine/internal/service/turn_executor"
	"gorm.io/gorm"
)

// capabilityConfigReader implements tools.CapabilityConfigReader.
// Reads any capability config from DB by agent name and type.
type capabilityConfigReader struct {
	db *gorm.DB
}

func (r *capabilityConfigReader) ReadConfig(ctx context.Context, agentName, capType string) (map[string]interface{}, error) {
	return resolveCapabilityConfigFromDB(r.db, ctx, agentName, capType)
}

// guardrailConfigResolver resolves guardrail capability config from DB for an agent.
// Implements infrastructure.GuardrailConfigResolver.
type guardrailConfigResolver struct {
	db *gorm.DB
}

func (r *guardrailConfigResolver) ResolveGuardrailConfig(ctx context.Context, agentName string) (*turn_executor.GuardrailCheckConfig, error) {
	config, err := resolveCapabilityConfigFromDB(r.db, ctx, agentName, "guardrail")
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, nil
	}

	mode, _ := config["mode"].(string)
	if mode == "" {
		return nil, nil
	}

	onFailure, _ := config["on_failure"].(string)
	fallbackText, _ := config["fallback_text"].(string)
	jsonSchema, _ := config["json_schema"].(string)
	judgePrompt, _ := config["judge_prompt"].(string)
	judgeModel, _ := config["judge_model"].(string)
	webhookURL, _ := config["webhook_url"].(string)

	maxRetries := 3
	if mr, ok := config["max_retries"].(float64); ok && mr > 0 {
		maxRetries = int(mr)
	}

	strict, _ := config["strict"].(bool)

	result := &turn_executor.GuardrailCheckConfig{
		Mode:         mode,
		OnFailure:    onFailure,
		MaxRetries:   maxRetries,
		FallbackText: fallbackText,
		JSONSchema:   jsonSchema,
		JudgePrompt:  judgePrompt,
		JudgeModel:   judgeModel,
		WebhookURL:   webhookURL,
		Strict:       strict,
	}
	// Strict mode overrides on_failure to "error" (always block failing output)
	if strict {
		result.OnFailure = "error"
	}
	return result, nil
}

// resolveCapabilityConfigFromDB reads capability config from DB by agent name and type.
// Shared by guardrailConfigResolver, capabilityConfigReader, and dynamicPolicyEvaluatorAdapter.
func resolveCapabilityConfigFromDB(db *gorm.DB, ctx context.Context, agentName, capType string) (map[string]interface{}, error) {
	var agentID string
	if err := db.WithContext(ctx).
		Raw("SELECT id FROM agents WHERE name = ?", agentName).
		Scan(&agentID).Error; err != nil || agentID == "" {
		return nil, nil
	}

	var cap models.CapabilityModel
	if err := db.WithContext(ctx).
		Where("agent_id = ? AND type = ?", agentID, capType).
		First(&cap).Error; err != nil {
		return nil, nil
	}

	if cap.Config == "" {
		return nil, nil
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(cap.Config), &config); err != nil {
		return nil, fmt.Errorf("parse %s config: %w", capType, err)
	}

	return config, nil
}

// dynamicPolicyEvaluatorAdapter resolves policy rules per-agent from capabilities DB.
// Implements tools.PolicyEvaluator.
type dynamicPolicyEvaluatorAdapter struct {
	db *gorm.DB
}

func (a *dynamicPolicyEvaluatorAdapter) EvaluateBefore(ctx context.Context, tc tools.PolicyToolCallContext) tools.PolicyEvalResult {
	rules := a.loadRulesForAgent(ctx, tc.AgentName)
	if len(rules) == 0 {
		return tools.PolicyEvalResult{}
	}

	engine := policy.New(rules, nil, nil)
	result := engine.EvaluateBefore(ctx, policy.ToolCallContext{
		AgentName: tc.AgentName,
		ToolName:  tc.ToolName,
		Arguments: tc.Arguments,
		Result:    tc.Result,
		Error:     tc.Error,
		Timestamp: tc.Timestamp,
	})

	return tools.PolicyEvalResult{
		Blocked:      result.Blocked,
		BlockMessage: result.BlockMessage,
	}
}

func (a *dynamicPolicyEvaluatorAdapter) EvaluateAfter(ctx context.Context, tc tools.PolicyToolCallContext) {
	rules := a.loadRulesForAgent(ctx, tc.AgentName)
	if len(rules) == 0 {
		return
	}

	engine := policy.New(rules, nil, nil)
	engine.EvaluateAfter(ctx, policy.ToolCallContext{
		AgentName: tc.AgentName,
		ToolName:  tc.ToolName,
		Arguments: tc.Arguments,
		Result:    tc.Result,
		Error:     tc.Error,
		Timestamp: tc.Timestamp,
	})
}

func (a *dynamicPolicyEvaluatorAdapter) loadRulesForAgent(ctx context.Context, agentName string) []*domain.PolicyRule {
	var agentID string
	if err := a.db.WithContext(ctx).
		Raw("SELECT id FROM agents WHERE name = ?", agentName).
		Scan(&agentID).Error; err != nil || agentID == "" {
		return nil
	}

	var cap models.CapabilityModel
	if err := a.db.WithContext(ctx).
		Where("agent_id = ? AND type = ?", agentID, "policies").
		First(&cap).Error; err != nil {
		return nil
	}

	if cap.Config == "" {
		return nil
	}

	// Policy config is stored as {"rules": [...]}.
	type policyRuleRaw struct {
		AgentName string `json:"agent_name"`
		Enabled   bool   `json:"enabled"`
		Condition struct {
			Type    string `json:"type"`
			Pattern string `json:"pattern"`
			Start   string `json:"start"`
			End     string `json:"end"`
		} `json:"condition"`
		Action struct {
			Type       string            `json:"type"`
			Message    string            `json:"message"`
			WebhookURL string            `json:"webhook_url"`
			Headers    map[string]string `json:"headers"`
		} `json:"action"`
	}

	// Parse config: try {"rules": [...]} wrapper, then direct array fallback.
	var rawRules []policyRuleRaw
	var wrapper struct {
		Rules []policyRuleRaw `json:"rules"`
	}
	if err := json.Unmarshal([]byte(cap.Config), &wrapper); err == nil && len(wrapper.Rules) > 0 {
		rawRules = wrapper.Rules
	} else if err := json.Unmarshal([]byte(cap.Config), &rawRules); err != nil {
		slog.WarnContext(ctx, "failed to parse policies config", "agent", agentName, "error", err)
		return nil
	}

	rules := make([]*domain.PolicyRule, 0, len(rawRules))
	for _, raw := range rawRules {
		effectiveAgent := raw.AgentName
		if effectiveAgent == "" {
			effectiveAgent = agentName
		}
		rules = append(rules, &domain.PolicyRule{
			AgentName: effectiveAgent,
			Enabled:   raw.Enabled,
			Condition: domain.PolicyCondition{
				Type:    domain.PolicyConditionType(raw.Condition.Type),
				Pattern: raw.Condition.Pattern,
				Start:   raw.Condition.Start,
				End:     raw.Condition.End,
			},
			Action: domain.PolicyAction{
				Type:       domain.PolicyActionType(raw.Action.Type),
				Message:    raw.Action.Message,
				WebhookURL: raw.Action.WebhookURL,
				Headers:    raw.Action.Headers,
			},
		})
	}

	return rules
}

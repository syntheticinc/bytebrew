package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// EscalationHandler handles escalation actions triggered by the escalate tool.
// Defined on the consumer side (tools package).
type EscalationHandler interface {
	Escalate(ctx context.Context, sessionID, agentName, reason string) (string, error)
}

// escalateArgs represents arguments for the escalate tool.
type escalateArgs struct {
	Reason string `json:"reason"`
}

// EscalateTool triggers escalation when the agent determines human intervention is needed.
type EscalateTool struct {
	sessionID string
	agentName string
	handler   EscalationHandler
}

// NewEscalateTool creates a new escalate tool.
func NewEscalateTool(sessionID, agentName string, handler EscalationHandler) tool.InvokableTool {
	return &EscalateTool{
		sessionID: sessionID,
		agentName: agentName,
		handler:   handler,
	}
}

// Info returns tool information for LLM.
func (t *EscalateTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "escalate",
		Desc: `Escalates the conversation to a human operator or external system.
Use this tool when you cannot resolve the user's issue, when the user explicitly requests human help,
or when the situation requires human judgment (e.g., billing disputes, account security, complex complaints).`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"reason": {
				Type:     schema.String,
				Desc:     "Why escalation is needed — brief explanation for the human operator",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun executes the escalation.
func (t *EscalateTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args escalateArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		slog.ErrorContext(ctx, "[EscalateTool] failed to parse arguments",
			"error", err, "raw", argumentsInJSON)
		return fmt.Sprintf("[ERROR] Invalid arguments: %v. Please provide a reason.", err), nil
	}

	if args.Reason == "" {
		return "[ERROR] reason is required for escalation.", nil
	}

	slog.InfoContext(ctx, "[EscalateTool] escalating",
		"session_id", t.sessionID, "agent", t.agentName, "reason", args.Reason)

	result, err := t.handler.Escalate(ctx, t.sessionID, t.agentName, args.Reason)
	if err != nil {
		slog.ErrorContext(ctx, "[EscalateTool] escalation failed",
			"session_id", t.sessionID, "agent", t.agentName, "error", err)
		return fmt.Sprintf("[ERROR] Escalation failed: %v", err), nil
	}

	return result, nil
}

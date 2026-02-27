package agent

import (
	"context"
	"fmt"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/agents"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/service/engine"
	einotool "github.com/cloudwego/eino/components/tool"
)

// runAgentWithEngine is the generic execution method for any agent type.
// Used by both coder (via runCodeAgentWithEngine) and researcher/reviewer agents.
func (p *AgentPool) runAgentWithEngine(
	ctx context.Context,
	sessionID, projectKey, agentID string,
	flowType domain.FlowType,
	subtaskID string,
	input string,
) (string, error) {
	p.mu.RLock()
	eng := p.engine
	flowProvider := p.flowProvider
	toolResolver := p.toolResolver
	toolDeps := p.toolDeps
	sessionDir := p.sessionDirName
	reminders := p.contextReminders
	p.mu.RUnlock()

	if eng == nil || flowProvider == nil || toolResolver == nil || toolDeps == nil {
		return "", fmt.Errorf("engine dependencies not configured")
	}

	flow, err := flowProvider.GetFlow(ctx, flowType)
	if err != nil {
		return "", fmt.Errorf("get %s flow: %w", flowType, err)
	}

	deps := toolDeps.GetDependencies(sessionID, projectKey)
	p.mu.RLock()
	sessionProxy, ok := p.sessionProxies[sessionID]
	p.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("no proxy for session: %s", sessionID)
	}
	deps.Proxy = sessionProxy

	resolvedTools, err := toolResolver.Resolve(ctx, flow.ToolNames, deps)
	if err != nil {
		return "", fmt.Errorf("resolve tools: %w", err)
	}

	baseTools := make([]einotool.BaseTool, len(resolvedTools))
	for i, t := range resolvedTools {
		baseTools[i] = t
	}

	eventCb := func(event *domain.AgentEvent) error {
		event.AgentID = agentID
		p.mu.RLock()
		cb := p.sessionEventCallbacks[sessionID]
		p.mu.RUnlock()
		if cb == nil {
			return nil
		}
		return cb(event)
	}

	var compressor engine.MessageCompressor
	if flow.MaxContextSize > 0 {
		compressor = engine.MessageCompressor(agents.NewContextRewriter(flow.MaxContextSize))
	}

	execCfg := engine.ExecutionConfig{
		SessionID:         sessionID,
		AgentID:           agentID,
		Flow:              flow,
		Tools:             baseTools,
		Input:             input,
		ChatModel:         p.modelSelector.Select(flowType),
		Streaming:         false,
		EventCallback:     eventCb,
		ContextReminders:  reminders,
		ModelName:         p.modelSelector.ModelName(flowType),
		AgentConfig:       p.agentConfig,
		ParentAgentID:     "supervisor",
		SubtaskID:         subtaskID,
		SessionDirName:    sessionDir,
		MessageCompressor: compressor,
	}

	result, err := eng.Execute(ctx, execCfg)
	if err != nil {
		return "", fmt.Errorf("execute engine: %w", err)
	}

	return result.Answer, nil
}

// runCodeAgentWithEngine executes a coder agent for a specific subtask.
// Delegates to the generic runAgentWithEngine with coder-specific input.
func (p *AgentPool) runCodeAgentWithEngine(
	ctx context.Context,
	sessionID, projectKey, agentID string,
	subtask *domain.Subtask,
) (string, error) {
	input := buildCodeAgentInput(subtask)
	return p.runAgentWithEngine(ctx, sessionID, projectKey, agentID, domain.FlowTypeCoder, subtask.ID, input)
}

func buildCodeAgentInput(subtask *domain.Subtask) string {
	input := fmt.Sprintf("Subtask: %s\n\nDescription: %s", subtask.Title, subtask.Description)
	if len(subtask.FilesInvolved) > 0 {
		input += fmt.Sprintf("\n\nRelevant files: %v", subtask.FilesInvolved)
	}
	return input
}

package agent

import (
	"context"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/tools"
)

// AgentPoolAdapter adapts AgentPool to tools.AgentPoolForTool interface
type AgentPoolAdapter struct {
	pool *AgentPool
}

// NewAgentPoolAdapter creates an adapter
func NewAgentPoolAdapter(pool *AgentPool) *AgentPoolAdapter {
	return &AgentPoolAdapter{pool: pool}
}

func (a *AgentPoolAdapter) Spawn(ctx context.Context, sessionID, projectKey, subtaskID string, blocking bool) (string, error) {
	return a.pool.Spawn(ctx, sessionID, projectKey, subtaskID, blocking)
}

func (a *AgentPoolAdapter) SpawnWithDescription(ctx context.Context, sessionID, projectKey string, flowType domain.FlowType, description string, blocking bool) (string, error) {
	return a.pool.SpawnWithDescription(ctx, sessionID, projectKey, flowType, description, blocking)
}

func (a *AgentPoolAdapter) WaitForAllSessionAgents(ctx context.Context, sessionID string) (tools.WaitResult, error) {
	result, err := a.pool.WaitForAllSessionAgents(ctx, sessionID)
	if err != nil {
		return tools.WaitResult{}, err
	}
	// Convert AgentCompletionInfo to tools.AgentCompletionInfo
	toolResults := make(map[string]tools.AgentCompletionInfo)
	for k, v := range result.Results {
		toolResults[k] = tools.AgentCompletionInfo{
			AgentID:   v.AgentID,
			SubtaskID: v.SubtaskID,
			Status:    v.Status,
			Result:    v.Result,
			Error:     v.Error,
		}
	}
	return tools.WaitResult{
		AllDone:              result.AllDone,
		Interrupted:          result.Interrupted,
		IsInterruptResponder: result.IsInterruptResponder,
		UserMessage:          result.UserMessage,
		StillRunning:         result.StillRunning,
		Results:              toolResults,
	}, nil
}

func (a *AgentPoolAdapter) HasBlockingWait(sessionID string) bool {
	return a.pool.HasBlockingWait(sessionID)
}

func (a *AgentPoolAdapter) NotifyUserMessage(sessionID, message string) {
	a.pool.NotifyUserMessage(sessionID, message)
}

func (a *AgentPoolAdapter) GetStatusInfo(agentID string) (*tools.AgentInfo, bool) {
	snap, ok := a.pool.GetStatus(agentID)
	if !ok {
		return nil, false
	}
	return &tools.AgentInfo{
		ID:        snap.ID,
		SubtaskID: snap.SubtaskID,
		Status:    snap.Status,
		Result:    snap.Result,
		Error:     snap.Error,
	}, true
}

func (a *AgentPoolAdapter) GetAllAgentInfos() []tools.AgentInfo {
	snapshots := a.pool.GetAllAgents()
	result := make([]tools.AgentInfo, 0, len(snapshots))
	for _, snap := range snapshots {
		result = append(result, tools.AgentInfo{
			ID:        snap.ID,
			SubtaskID: snap.SubtaskID,
			Status:    snap.Status,
			Result:    snap.Result,
			Error:     snap.Error,
		})
	}
	return result
}

func (a *AgentPoolAdapter) StopAgent(agentID string) error {
	return a.pool.StopAgent(agentID)
}

func (a *AgentPoolAdapter) RestartAgent(ctx context.Context, agentID string, blocking bool) (string, error) {
	return a.pool.RestartAgent(ctx, agentID, blocking)
}

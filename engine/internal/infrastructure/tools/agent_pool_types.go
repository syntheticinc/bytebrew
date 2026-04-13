package tools

import (
	"context"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// AgentInfo holds minimal agent info for listing.
type AgentInfo struct {
	ID        string
	SubtaskID string
	Status    string
	Result    string
	Error     string
}

// WaitResult describes the result of waiting for session agents.
type WaitResult struct {
	AllDone              bool                           // true if all agents completed
	Interrupted          bool                           // true if interrupted by user message
	IsInterruptResponder bool                           // true = this call should return full INTERRUPT
	UserMessage          string                         // user message that caused interrupt
	StillRunning         []string                       // agent IDs still running
	Results              map[string]AgentCompletionInfo // completed agents
	Summaries            []AgentSummary                 // agent summaries (used by generic SpawnTool)
}

// AgentCompletionInfo holds completion info for an agent.
type AgentCompletionInfo struct {
	AgentID   string
	SubtaskID string
	Status    string
	Result    string
	Error     string
}

// AgentPoolForTool is a simplified interface for multi-agent pool operations (consumer-side).
// Implemented by agent.AgentPoolAdapter.
type AgentPoolForTool interface {
	SpawnWithDescription(ctx context.Context, sessionID, projectKey string, flowType domain.FlowType, description string, blocking bool) (string, error)
	WaitForAllSessionAgents(ctx context.Context, sessionID string) (WaitResult, error)
	HasBlockingWait(sessionID string) bool
	NotifyUserMessage(sessionID, message string)
	GetStatusInfo(agentID string) (*AgentInfo, bool)
	GetAllAgentInfos() []AgentInfo
	StopAgent(agentID string) error
	RestartAgent(ctx context.Context, agentID string, blocking bool) (string, error)
}

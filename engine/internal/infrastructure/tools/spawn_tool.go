package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// GenericAgentSpawner is a consumer-side interface for spawn/wait/stop operations.
// Used by the generic SpawnTool (as opposed to AgentPoolForTool used by the legacy spawn_code_agent).
type GenericAgentSpawner interface {
	SpawnAgent(ctx context.Context, params SpawnParams) (string, error)
	WaitForAllSessionAgents(ctx context.Context, sessionID string) (WaitResult, error)
	HasBlockingWait(sessionID string) bool
	NotifyUserMessage(sessionID, message string)
	StopAgent(agentID string) error
}

// GenericAgentInspector is a consumer-side interface for agent status/list queries.
type GenericAgentInspector interface {
	GetStatusInfo(agentID string) (*AgentInfo, bool)
	GetAllAgentInfos() []AgentInfo
}

// SpawnParams describes parameters for spawning an agent.
type SpawnParams struct {
	SessionID   string
	AgentName   string
	Description string
	TaskID      string
	Blocking    bool
}

// NewSpawnTool creates a generic spawn tool for a specific target agent.
// Tool name will be "spawn_{targetAgentName}".
func NewSpawnTool(targetAgentName string, sessionID string, spawner GenericAgentSpawner, inspector GenericAgentInspector) tool.InvokableTool {
	return &spawnTool{
		targetAgent: targetAgentName,
		sessionID:   sessionID,
		spawner:     spawner,
		inspector:   inspector,
	}
}

type spawnTool struct {
	targetAgent string
	sessionID   string
	spawner     GenericAgentSpawner
	inspector   GenericAgentInspector
}

type spawnToolArgs struct {
	Action      string `json:"action"`
	Description string `json:"description"`
	AgentID     string `json:"agent_id"`
}

func (t *spawnTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "spawn_" + t.targetAgent,
		Desc: fmt.Sprintf("Spawn agent '%s' to handle a subtask. Returns agent summary when done.", t.targetAgent),
		ParamsOneOf: schema.NewParamsOneOfByParams(
			map[string]*schema.ParameterInfo{
				"action": {
					Type:     schema.String,
					Desc:     "Action: spawn (create agent), wait (wait for all agents), status (check agent), list (all agents), stop (terminate agent)",
					Required: true,
					Enum:     []string{"spawn", "wait", "status", "list", "stop"},
				},
				"description": {
					Type: "string",
					Desc: "Task description for the spawned agent (required for 'spawn' action)",
				},
				"agent_id": {
					Type: "string",
					Desc: "Agent ID (for 'status' and 'stop' actions)",
				},
			},
		),
	}, nil
}

func (t *spawnTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args spawnToolArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}

	switch args.Action {
	case "spawn":
		return t.handleSpawn(ctx, args)
	case "wait":
		return t.handleWait(ctx)
	case "status":
		return t.handleStatus(args)
	case "list":
		return t.handleList()
	case "stop":
		return t.handleStop(args)
	default:
		return "", fmt.Errorf("unknown action %q", args.Action)
	}
}

func (t *spawnTool) handleSpawn(ctx context.Context, args spawnToolArgs) (string, error) {
	if args.Description == "" {
		return "", fmt.Errorf("description required for spawn action")
	}

	agentID, err := t.spawner.SpawnAgent(ctx, SpawnParams{
		SessionID:   t.sessionID,
		AgentName:   t.targetAgent,
		Description: args.Description,
		Blocking:    true,
	})
	if err != nil {
		return "", fmt.Errorf("spawn agent: %w", err)
	}

	return fmt.Sprintf("Agent spawned with ID: %s", agentID), nil
}

func (t *spawnTool) handleWait(ctx context.Context) (string, error) {
	result, err := t.spawner.WaitForAllSessionAgents(ctx, t.sessionID)
	if err != nil {
		return "", fmt.Errorf("wait for agents: %w", err)
	}

	data, err := json.Marshal(result.Summaries)
	if err != nil {
		return "", fmt.Errorf("marshal wait result: %w", err)
	}

	return string(data), nil
}

func (t *spawnTool) handleStatus(args spawnToolArgs) (string, error) {
	if args.AgentID == "" {
		return "", fmt.Errorf("agent_id required for status action")
	}

	info, ok := t.inspector.GetStatusInfo(args.AgentID)
	if !ok {
		return fmt.Sprintf("Agent %s not found", args.AgentID), nil
	}

	data, err := json.Marshal(info)
	if err != nil {
		return "", fmt.Errorf("marshal agent info: %w", err)
	}

	return string(data), nil
}

func (t *spawnTool) handleList() (string, error) {
	infos := t.inspector.GetAllAgentInfos()

	data, err := json.Marshal(infos)
	if err != nil {
		return "", fmt.Errorf("marshal agent infos: %w", err)
	}

	return string(data), nil
}

func (t *spawnTool) handleStop(args spawnToolArgs) (string, error) {
	if args.AgentID == "" {
		return "", fmt.Errorf("agent_id required for stop action")
	}

	if err := t.spawner.StopAgent(args.AgentID); err != nil {
		return "", fmt.Errorf("stop agent: %w", err)
	}

	return fmt.Sprintf("Agent %s stopped", args.AgentID), nil
}

// AgentSummary holds completion summary for an agent (used in WaitResult.Summaries).
type AgentSummary struct {
	AgentID   string `json:"agent_id"`
	AgentName string `json:"agent_name"`
	Summary   string `json:"summary"`
	Status    string `json:"status"`
}

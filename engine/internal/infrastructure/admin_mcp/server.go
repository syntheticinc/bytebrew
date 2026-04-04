// Package admin_mcp provides an in-process MCP server that exposes
// Engine admin operations (agents, models, triggers, MCP servers, config)
// as MCP tools. It defines its own consumer-side interfaces and does NOT
// import from delivery/http.
package admin_mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/mcp"
)

// ---------------------------------------------------------------------------
// Consumer-side interfaces (defined here, NOT imported from delivery/http)
// ---------------------------------------------------------------------------

// AgentInfo is a summary of an agent.
type AgentInfo struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	ToolsCount   int    `json:"tools_count"`
	Kit          string `json:"kit,omitempty"`
	HasKnowledge bool   `json:"has_knowledge"`
}

// AgentEscalation holds escalation settings.
type AgentEscalation struct {
	Action     string   `json:"action"`
	WebhookURL string   `json:"webhook_url,omitempty"`
	Triggers   []string `json:"triggers"`
}

// AgentDetail is the full agent information.
type AgentDetail struct {
	AgentInfo
	ModelID        *uint            `json:"model_id,omitempty"`
	SystemPrompt   string           `json:"system_prompt"`
	KnowledgePath  string           `json:"knowledge_path,omitempty"`
	Tools          []string         `json:"tools"`
	CanSpawn       []string         `json:"can_spawn,omitempty"`
	Lifecycle      string           `json:"lifecycle"`
	ToolExecution  string           `json:"tool_execution"`
	MaxSteps       int              `json:"max_steps"`
	MaxContextSize int              `json:"max_context_size"`
	ConfirmBefore  []string         `json:"confirm_before,omitempty"`
	MCPServers     []string         `json:"mcp_servers,omitempty"`
	Escalation     *AgentEscalation `json:"escalation,omitempty"`
}

// CreateAgentRequest is the request body for creating/updating an agent.
type CreateAgentRequest struct {
	Name           string           `json:"name"`
	Model          string           `json:"model,omitempty"`
	ModelID        *uint            `json:"model_id,omitempty"`
	SystemPrompt   string           `json:"system_prompt"`
	Kit            string           `json:"kit,omitempty"`
	KnowledgePath  string           `json:"knowledge_path,omitempty"`
	Lifecycle      string           `json:"lifecycle,omitempty"`
	ToolExecution  string           `json:"tool_execution,omitempty"`
	MaxSteps       int              `json:"max_steps,omitempty"`
	MaxContextSize int              `json:"max_context_size,omitempty"`
	ConfirmBefore  []string         `json:"confirm_before,omitempty"`
	Tools          []string         `json:"tools,omitempty"`
	CanSpawn       []string         `json:"can_spawn,omitempty"`
	MCPServers     []string         `json:"mcp_servers,omitempty"`
	Escalation     *AgentEscalation `json:"escalation,omitempty"`
}

// AgentManager provides agent CRUD operations.
type AgentManager interface {
	ListAgents(ctx context.Context) ([]AgentInfo, error)
	GetAgent(ctx context.Context, name string) (*AgentDetail, error)
	CreateAgent(ctx context.Context, req CreateAgentRequest) (*AgentDetail, error)
	UpdateAgent(ctx context.Context, name string, req CreateAgentRequest) (*AgentDetail, error)
	DeleteAgent(ctx context.Context, name string) error
}

// ModelResponse is the API representation of an LLM model.
type ModelResponse struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	BaseURL    string `json:"base_url,omitempty"`
	ModelName  string `json:"model_name"`
	HasAPIKey  bool   `json:"has_api_key"`
	APIVersion string `json:"api_version,omitempty"`
	CreatedAt  string `json:"created_at"`
}

// CreateModelRequest is the request body for creating/updating a model.
type CreateModelRequest struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	BaseURL    string `json:"base_url,omitempty"`
	ModelName  string `json:"model_name"`
	APIKey     string `json:"api_key,omitempty"`
	APIVersion string `json:"api_version,omitempty"`
}

// ModelManager provides model CRUD operations.
type ModelManager interface {
	ListModels(ctx context.Context) ([]ModelResponse, error)
	CreateModel(ctx context.Context, req CreateModelRequest) (*ModelResponse, error)
	UpdateModel(ctx context.Context, name string, req CreateModelRequest) (*ModelResponse, error)
	DeleteModel(ctx context.Context, name string) error
}

// TriggerResponse is the API representation of a trigger.
type TriggerResponse struct {
	ID          uint   `json:"id"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	AgentID     uint   `json:"agent_id"`
	AgentName   string `json:"agent_name,omitempty"`
	Schedule    string `json:"schedule,omitempty"`
	WebhookPath string `json:"webhook_path,omitempty"`
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled"`
	LastFiredAt string `json:"last_fired_at,omitempty"`
	CreatedAt   string `json:"created_at"`
}

// CreateTriggerRequest is the request body for creating/updating a trigger.
type CreateTriggerRequest struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	AgentID     uint   `json:"agent_id"`
	Schedule    string `json:"schedule,omitempty"`
	WebhookPath string `json:"webhook_path,omitempty"`
	Description string `json:"description,omitempty"`
	Enabled     *bool  `json:"enabled,omitempty"`
}

// TriggerManager provides trigger CRUD operations.
type TriggerManager interface {
	ListTriggers(ctx context.Context) ([]TriggerResponse, error)
	CreateTrigger(ctx context.Context, req CreateTriggerRequest) (*TriggerResponse, error)
	UpdateTrigger(ctx context.Context, id uint, req CreateTriggerRequest) (*TriggerResponse, error)
	DeleteTrigger(ctx context.Context, id uint) error
}

// MCPServerResponse is the API representation of an MCP server.
type MCPServerResponse struct {
	ID             uint              `json:"id"`
	Name           string            `json:"name"`
	Type           string            `json:"type"`
	Command        string            `json:"command,omitempty"`
	Args           []string          `json:"args,omitempty"`
	URL            string            `json:"url,omitempty"`
	EnvVars        map[string]string `json:"env_vars,omitempty"`
	ForwardHeaders []string          `json:"forward_headers,omitempty"`
	IsWellKnown    bool              `json:"is_well_known"`
	Agents         []string          `json:"agents"`
}

// MCPServerLister provides MCP server listing.
type MCPServerLister interface {
	ListMCPServers(ctx context.Context) ([]MCPServerResponse, error)
}

// ToolMetadataResponse is the metadata for a single tool.
type ToolMetadataResponse struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	SecurityZone string `json:"security_zone"`
	RiskWarning  string `json:"risk_warning,omitempty"`
}

// ToolMetadataProvider returns metadata for all known tools.
type ToolMetadataProvider interface {
	GetAllToolMetadata() []ToolMetadataResponse
}

// ConfigExporter handles YAML config export and import.
type ConfigExporter interface {
	ExportYAML(ctx context.Context) ([]byte, error)
	ImportYAML(ctx context.Context, yamlData []byte) error
}

// Reloader triggers in-memory state reload after mutations.
type Reloader interface {
	Reload(ctx context.Context) error
}

// ---------------------------------------------------------------------------
// Server
// ---------------------------------------------------------------------------

// ServerConfig holds the dependencies for the admin MCP server.
type ServerConfig struct {
	AgentManager         AgentManager
	ModelManager         ModelManager
	TriggerManager       TriggerManager
	MCPServerLister      MCPServerLister
	ToolMetadataProvider ToolMetadataProvider
	ConfigExporter       ConfigExporter
	Reloader             Reloader
}

// Server is an in-process MCP server that exposes admin operations as MCP tools.
type Server struct {
	agents       AgentManager
	models       ModelManager
	triggers     TriggerManager
	mcpServers   MCPServerLister
	toolMetadata ToolMetadataProvider
	config       ConfigExporter
	reloader     Reloader
	tools        []mcp.MCPTool
}

// NewServer creates a new admin MCP server. All interface parameters may be nil;
// the corresponding tools will return errors when called.
func NewServer(cfg ServerConfig) *Server {
	s := &Server{
		agents:       cfg.AgentManager,
		models:       cfg.ModelManager,
		triggers:     cfg.TriggerManager,
		mcpServers:   cfg.MCPServerLister,
		toolMetadata: cfg.ToolMetadataProvider,
		config:       cfg.ConfigExporter,
		reloader:     cfg.Reloader,
	}
	s.tools = s.buildToolList()
	return s
}

// Handle is the mcp.Handler function that dispatches JSON-RPC requests.
func (s *Server) Handle(ctx context.Context, req *mcp.Request) (*mcp.Response, error) {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "notifications/initialized":
		return &mcp.Response{JSONRPC: "2.0", ID: req.ID}, nil
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	default:
		return &mcp.Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &mcp.RPCError{Code: -32601, Message: fmt.Sprintf("method not found: %s", req.Method)},
		}, nil
	}
}

func (s *Server) handleInitialize(req *mcp.Request) (*mcp.Response, error) {
	result, _ := json.Marshal(map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "admin-api",
			"version": "1.0.0",
		},
	})
	return &mcp.Response{JSONRPC: "2.0", ID: req.ID, Result: result}, nil
}

func (s *Server) handleToolsList(req *mcp.Request) (*mcp.Response, error) {
	result, err := json.Marshal(mcp.ToolsListResult{Tools: s.tools})
	if err != nil {
		return nil, fmt.Errorf("marshal tools list: %w", err)
	}
	return &mcp.Response{JSONRPC: "2.0", ID: req.ID, Result: result}, nil
}

func (s *Server) handleToolsCall(ctx context.Context, req *mcp.Request) (*mcp.Response, error) {
	// Parse params to get tool name and arguments.
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		return toolErrorResponse(req.ID, "invalid params"), nil
	}

	var callParams struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := json.Unmarshal(paramsBytes, &callParams); err != nil {
		return toolErrorResponse(req.ID, "invalid params: "+err.Error()), nil
	}

	result, toolErr := s.dispatchTool(ctx, callParams.Name, callParams.Arguments)
	if toolErr != "" {
		return toolErrorResponse(req.ID, toolErr), nil
	}

	return toolSuccessResponse(req.ID, result), nil
}

// dispatchTool routes a tool call to the appropriate handler.
// Returns (result string, error message). If error message is non-empty, it's a tool error.
func (s *Server) dispatchTool(ctx context.Context, name string, args map[string]interface{}) (string, string) {
	switch name {
	// Agents
	case "list_agents":
		return s.toolListAgents(ctx)
	case "get_agent":
		return s.toolGetAgent(ctx, args)
	case "create_agent":
		return s.toolCreateAgent(ctx, args)
	case "update_agent":
		return s.toolUpdateAgent(ctx, args)
	case "delete_agent":
		return s.toolDeleteAgent(ctx, args)
	// Models
	case "list_models":
		return s.toolListModels(ctx)
	case "create_model":
		return s.toolCreateModel(ctx, args)
	case "update_model":
		return s.toolUpdateModel(ctx, args)
	case "delete_model":
		return s.toolDeleteModel(ctx, args)
	// Triggers
	case "list_triggers":
		return s.toolListTriggers(ctx)
	case "create_trigger":
		return s.toolCreateTrigger(ctx, args)
	case "update_trigger":
		return s.toolUpdateTrigger(ctx, args)
	case "delete_trigger":
		return s.toolDeleteTrigger(ctx, args)
	// MCP Servers
	case "list_mcp_servers":
		return s.toolListMCPServers(ctx)
	// Tools
	case "list_tools":
		return s.toolListTools()
	// Config
	case "export_config":
		return s.toolExportConfig(ctx)
	case "import_config":
		return s.toolImportConfig(ctx, args)
	default:
		return "", fmt.Sprintf("unknown tool: %s", name)
	}
}

// ---------------------------------------------------------------------------
// Tool implementations
// ---------------------------------------------------------------------------

// --- Agents ---

func (s *Server) toolListAgents(ctx context.Context) (string, string) {
	if s.agents == nil {
		return "", "agent manager not configured"
	}
	agents, err := s.agents.ListAgents(ctx)
	if err != nil {
		return "", fmt.Sprintf("list agents: %s", err)
	}
	return marshalResult(agents)
}

func (s *Server) toolGetAgent(ctx context.Context, args map[string]interface{}) (string, string) {
	if s.agents == nil {
		return "", "agent manager not configured"
	}
	name, ok := stringArg(args, "name")
	if !ok {
		return "", "missing required parameter: name"
	}
	agent, err := s.agents.GetAgent(ctx, name)
	if err != nil {
		return "", fmt.Sprintf("get agent: %s", err)
	}
	if agent == nil {
		return "", fmt.Sprintf("agent not found: %s", name)
	}
	return marshalResult(agent)
}

func (s *Server) toolCreateAgent(ctx context.Context, args map[string]interface{}) (string, string) {
	if s.agents == nil {
		return "", "agent manager not configured"
	}
	var req CreateAgentRequest
	if err := remarshal(args, &req); err != nil {
		return "", fmt.Sprintf("invalid arguments: %s", err)
	}
	if req.Name == "" {
		return "", "missing required parameter: name"
	}
	agent, err := s.agents.CreateAgent(ctx, req)
	if err != nil {
		return "", fmt.Sprintf("create agent: %s", err)
	}
	s.reload(ctx)
	return marshalResult(agent)
}

func (s *Server) toolUpdateAgent(ctx context.Context, args map[string]interface{}) (string, string) {
	if s.agents == nil {
		return "", "agent manager not configured"
	}
	name, ok := stringArg(args, "name")
	if !ok {
		return "", "missing required parameter: name"
	}
	var req CreateAgentRequest
	if err := remarshal(args, &req); err != nil {
		return "", fmt.Sprintf("invalid arguments: %s", err)
	}
	agent, err := s.agents.UpdateAgent(ctx, name, req)
	if err != nil {
		return "", fmt.Sprintf("update agent: %s", err)
	}
	s.reload(ctx)
	return marshalResult(agent)
}

func (s *Server) toolDeleteAgent(ctx context.Context, args map[string]interface{}) (string, string) {
	if s.agents == nil {
		return "", "agent manager not configured"
	}
	name, ok := stringArg(args, "name")
	if !ok {
		return "", "missing required parameter: name"
	}
	if err := s.agents.DeleteAgent(ctx, name); err != nil {
		return "", fmt.Sprintf("delete agent: %s", err)
	}
	s.reload(ctx)
	result, _ := json.Marshal(map[string]string{"deleted": name})
	return string(result), ""
}

// --- Models ---

func (s *Server) toolListModels(ctx context.Context) (string, string) {
	if s.models == nil {
		return "", "model manager not configured"
	}
	models, err := s.models.ListModels(ctx)
	if err != nil {
		return "", fmt.Sprintf("list models: %s", err)
	}
	return marshalResult(models)
}

func (s *Server) toolCreateModel(ctx context.Context, args map[string]interface{}) (string, string) {
	if s.models == nil {
		return "", "model manager not configured"
	}
	var req CreateModelRequest
	if err := remarshal(args, &req); err != nil {
		return "", fmt.Sprintf("invalid arguments: %s", err)
	}
	if req.Name == "" {
		return "", "missing required parameter: name"
	}
	model, err := s.models.CreateModel(ctx, req)
	if err != nil {
		return "", fmt.Sprintf("create model: %s", err)
	}
	s.reload(ctx)
	return marshalResult(model)
}

func (s *Server) toolUpdateModel(ctx context.Context, args map[string]interface{}) (string, string) {
	if s.models == nil {
		return "", "model manager not configured"
	}
	name, ok := stringArg(args, "name")
	if !ok {
		return "", "missing required parameter: name"
	}
	var req CreateModelRequest
	if err := remarshal(args, &req); err != nil {
		return "", fmt.Sprintf("invalid arguments: %s", err)
	}
	model, err := s.models.UpdateModel(ctx, name, req)
	if err != nil {
		return "", fmt.Sprintf("update model: %s", err)
	}
	s.reload(ctx)
	return marshalResult(model)
}

func (s *Server) toolDeleteModel(ctx context.Context, args map[string]interface{}) (string, string) {
	if s.models == nil {
		return "", "model manager not configured"
	}
	name, ok := stringArg(args, "name")
	if !ok {
		return "", "missing required parameter: name"
	}
	if err := s.models.DeleteModel(ctx, name); err != nil {
		return "", fmt.Sprintf("delete model: %s", err)
	}
	s.reload(ctx)
	result, _ := json.Marshal(map[string]string{"deleted": name})
	return string(result), ""
}

// --- Triggers ---

func (s *Server) toolListTriggers(ctx context.Context) (string, string) {
	if s.triggers == nil {
		return "", "trigger manager not configured"
	}
	triggers, err := s.triggers.ListTriggers(ctx)
	if err != nil {
		return "", fmt.Sprintf("list triggers: %s", err)
	}
	return marshalResult(triggers)
}

func (s *Server) toolCreateTrigger(ctx context.Context, args map[string]interface{}) (string, string) {
	if s.triggers == nil {
		return "", "trigger manager not configured"
	}
	var req CreateTriggerRequest
	if err := remarshal(args, &req); err != nil {
		return "", fmt.Sprintf("invalid arguments: %s", err)
	}
	trigger, err := s.triggers.CreateTrigger(ctx, req)
	if err != nil {
		return "", fmt.Sprintf("create trigger: %s", err)
	}
	s.reload(ctx)
	return marshalResult(trigger)
}

func (s *Server) toolUpdateTrigger(ctx context.Context, args map[string]interface{}) (string, string) {
	if s.triggers == nil {
		return "", "trigger manager not configured"
	}
	idVal, ok := numericArg(args, "id")
	if !ok {
		return "", "missing required parameter: id"
	}
	var req CreateTriggerRequest
	if err := remarshal(args, &req); err != nil {
		return "", fmt.Sprintf("invalid arguments: %s", err)
	}
	trigger, err := s.triggers.UpdateTrigger(ctx, uint(idVal), req)
	if err != nil {
		return "", fmt.Sprintf("update trigger: %s", err)
	}
	s.reload(ctx)
	return marshalResult(trigger)
}

func (s *Server) toolDeleteTrigger(ctx context.Context, args map[string]interface{}) (string, string) {
	if s.triggers == nil {
		return "", "trigger manager not configured"
	}
	idVal, ok := numericArg(args, "id")
	if !ok {
		return "", "missing required parameter: id"
	}
	if err := s.triggers.DeleteTrigger(ctx, uint(idVal)); err != nil {
		return "", fmt.Sprintf("delete trigger: %s", err)
	}
	s.reload(ctx)
	result, _ := json.Marshal(map[string]int64{"deleted": int64(idVal)})
	return string(result), ""
}

// --- MCP Servers ---

func (s *Server) toolListMCPServers(ctx context.Context) (string, string) {
	if s.mcpServers == nil {
		return "", "mcp server lister not configured"
	}
	servers, err := s.mcpServers.ListMCPServers(ctx)
	if err != nil {
		return "", fmt.Sprintf("list mcp servers: %s", err)
	}
	return marshalResult(servers)
}

// --- Tools ---

func (s *Server) toolListTools() (string, string) {
	if s.toolMetadata == nil {
		return "", "tool metadata provider not configured"
	}
	meta := s.toolMetadata.GetAllToolMetadata()
	return marshalResult(meta)
}

// --- Config ---

func (s *Server) toolExportConfig(ctx context.Context) (string, string) {
	if s.config == nil {
		return "", "config exporter not configured"
	}
	data, err := s.config.ExportYAML(ctx)
	if err != nil {
		return "", fmt.Sprintf("export config: %s", err)
	}
	return string(data), ""
}

func (s *Server) toolImportConfig(ctx context.Context, args map[string]interface{}) (string, string) {
	if s.config == nil {
		return "", "config exporter not configured"
	}
	yamlStr, ok := stringArg(args, "yaml_content")
	if !ok {
		return "", "missing required parameter: yaml_content"
	}
	if err := s.config.ImportYAML(ctx, []byte(yamlStr)); err != nil {
		return "", fmt.Sprintf("import config: %s", err)
	}
	s.reload(ctx)
	return `{"status":"imported"}`, ""
}

// ---------------------------------------------------------------------------
// Tool definitions (MCP tool list)
// ---------------------------------------------------------------------------

func (s *Server) buildToolList() []mcp.MCPTool {
	return []mcp.MCPTool{
		// Agents
		{
			Name:        "list_agents",
			Description: "List all configured agents with summary info (name, tools count, kit, knowledge status).",
			InputSchema: mustSchema(map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}),
		},
		{
			Name:        "get_agent",
			Description: "Get full details of an agent by name, including system prompt, tools, lifecycle, MCP servers, and escalation config.",
			InputSchema: mustSchema(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string", "description": "Agent name"},
				},
				"required": []string{"name"},
			}),
		},
		{
			Name:        "create_agent",
			Description: "Create a new agent. Required: name, system_prompt. Optional: model (model name), lifecycle, tool_execution, max_steps, tools, mcp_servers, can_spawn, confirm_before, escalation.",
			InputSchema: mustSchema(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name":             map[string]interface{}{"type": "string", "description": "Unique agent name (lowercase, hyphens allowed)"},
					"system_prompt":    map[string]interface{}{"type": "string", "description": "System prompt for the agent"},
					"model":            map[string]interface{}{"type": "string", "description": "Model name to use"},
					"lifecycle":        map[string]interface{}{"type": "string", "description": "Agent lifecycle: persistent or ephemeral"},
					"tool_execution":   map[string]interface{}{"type": "string", "description": "Tool execution mode: sequential or parallel"},
					"max_steps":        map[string]interface{}{"type": "integer", "description": "Maximum reasoning steps"},
					"max_context_size": map[string]interface{}{"type": "integer", "description": "Maximum context window size"},
					"tools":            map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Builtin tool names"},
					"mcp_servers":      map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "MCP server names"},
					"can_spawn":        map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Agents this agent can spawn"},
					"confirm_before":   map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Tools requiring user confirmation"},
				},
				"required": []string{"name", "system_prompt"},
			}),
		},
		{
			Name:        "update_agent",
			Description: "Update an existing agent by name. Pass only the fields you want to change.",
			InputSchema: mustSchema(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name":             map[string]interface{}{"type": "string", "description": "Agent name to update"},
					"system_prompt":    map[string]interface{}{"type": "string", "description": "New system prompt"},
					"model":            map[string]interface{}{"type": "string", "description": "Model name"},
					"lifecycle":        map[string]interface{}{"type": "string", "description": "Agent lifecycle"},
					"tool_execution":   map[string]interface{}{"type": "string", "description": "Tool execution mode"},
					"max_steps":        map[string]interface{}{"type": "integer", "description": "Maximum reasoning steps"},
					"max_context_size": map[string]interface{}{"type": "integer", "description": "Maximum context window size"},
					"tools":            map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Builtin tool names"},
					"mcp_servers":      map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "MCP server names"},
					"can_spawn":        map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Agents this agent can spawn"},
					"confirm_before":   map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Tools requiring user confirmation"},
				},
				"required": []string{"name"},
			}),
		},
		{
			Name:        "delete_agent",
			Description: "Delete an agent by name. WARNING: This is destructive and cannot be undone. Always confirm with the user before deleting.",
			InputSchema: mustSchema(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string", "description": "Agent name to delete"},
				},
				"required": []string{"name"},
			}),
		},
		// Models
		{
			Name:        "list_models",
			Description: "List all configured LLM models with their type, base URL, and model name.",
			InputSchema: mustSchema(map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}),
		},
		{
			Name:        "create_model",
			Description: "Create a new LLM model configuration. Required: name, type, model_name. Optional: base_url, api_key, api_version.",
			InputSchema: mustSchema(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name":        map[string]interface{}{"type": "string", "description": "Unique model config name"},
					"type":        map[string]interface{}{"type": "string", "description": "Model type: openai_compatible, anthropic, etc."},
					"model_name":  map[string]interface{}{"type": "string", "description": "Actual model name (e.g. gpt-4o, claude-3-opus)"},
					"base_url":    map[string]interface{}{"type": "string", "description": "API base URL"},
					"api_key":     map[string]interface{}{"type": "string", "description": "API key"},
					"api_version": map[string]interface{}{"type": "string", "description": "API version (for Azure)"},
				},
				"required": []string{"name", "type", "model_name"},
			}),
		},
		{
			Name:        "update_model",
			Description: "Update a model configuration by name. Pass only the fields you want to change. Leave api_key empty to keep existing key.",
			InputSchema: mustSchema(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name":        map[string]interface{}{"type": "string", "description": "Model name to update"},
					"type":        map[string]interface{}{"type": "string", "description": "Model type"},
					"model_name":  map[string]interface{}{"type": "string", "description": "Actual model name"},
					"base_url":    map[string]interface{}{"type": "string", "description": "API base URL"},
					"api_key":     map[string]interface{}{"type": "string", "description": "API key (empty = keep existing)"},
					"api_version": map[string]interface{}{"type": "string", "description": "API version"},
				},
				"required": []string{"name"},
			}),
		},
		{
			Name:        "delete_model",
			Description: "Delete a model configuration by name. WARNING: Agents using this model will stop working. Confirm with user first.",
			InputSchema: mustSchema(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string", "description": "Model name to delete"},
				},
				"required": []string{"name"},
			}),
		},
		// Triggers
		{
			Name:        "list_triggers",
			Description: "List all triggers (cron schedules, webhooks) with their type, agent, schedule, and enabled status.",
			InputSchema: mustSchema(map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}),
		},
		{
			Name:        "create_trigger",
			Description: "Create a new trigger. Required: type (cron/webhook), title, agent_id. For cron: schedule (cron expression). For webhook: webhook_path.",
			InputSchema: mustSchema(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"type":         map[string]interface{}{"type": "string", "description": "Trigger type: cron or webhook"},
					"title":        map[string]interface{}{"type": "string", "description": "Trigger title"},
					"agent_id":     map[string]interface{}{"type": "integer", "description": "ID of the agent to trigger"},
					"schedule":     map[string]interface{}{"type": "string", "description": "Cron expression (for cron type)"},
					"webhook_path": map[string]interface{}{"type": "string", "description": "Webhook path (for webhook type)"},
					"description":  map[string]interface{}{"type": "string", "description": "Trigger description"},
					"enabled":      map[string]interface{}{"type": "boolean", "description": "Whether trigger is enabled"},
				},
				"required": []string{"type", "title", "agent_id"},
			}),
		},
		{
			Name:        "update_trigger",
			Description: "Update a trigger by ID. Pass only the fields you want to change.",
			InputSchema: mustSchema(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id":           map[string]interface{}{"type": "integer", "description": "Trigger ID to update"},
					"type":         map[string]interface{}{"type": "string", "description": "Trigger type"},
					"title":        map[string]interface{}{"type": "string", "description": "Trigger title"},
					"agent_id":     map[string]interface{}{"type": "integer", "description": "Agent ID"},
					"schedule":     map[string]interface{}{"type": "string", "description": "Cron expression"},
					"webhook_path": map[string]interface{}{"type": "string", "description": "Webhook path"},
					"description":  map[string]interface{}{"type": "string", "description": "Description"},
					"enabled":      map[string]interface{}{"type": "boolean", "description": "Whether trigger is enabled"},
				},
				"required": []string{"id"},
			}),
		},
		{
			Name:        "delete_trigger",
			Description: "Delete a trigger by ID. Confirm with user first.",
			InputSchema: mustSchema(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{"type": "integer", "description": "Trigger ID to delete"},
				},
				"required": []string{"id"},
			}),
		},
		// MCP Servers
		{
			Name:        "list_mcp_servers",
			Description: "List all configured MCP servers with their type, command/URL, and connected agents.",
			InputSchema: mustSchema(map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}),
		},
		// Tools
		{
			Name:        "list_tools",
			Description: "List all available builtin tools with their description and security zone (safe/cautious/dangerous).",
			InputSchema: mustSchema(map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}),
		},
		// Config
		{
			Name:        "export_config",
			Description: "Export the full Engine configuration as YAML (agents, models, triggers, MCP servers).",
			InputSchema: mustSchema(map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}),
		},
		{
			Name:        "import_config",
			Description: "Import Engine configuration from YAML. Uses upsert semantics: creates new entities, updates existing ones, never deletes. Triggers a full reload after import.",
			InputSchema: mustSchema(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"yaml_content": map[string]interface{}{"type": "string", "description": "YAML configuration content"},
				},
				"required": []string{"yaml_content"},
			}),
		},
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (s *Server) reload(ctx context.Context) {
	if s.reloader == nil {
		return
	}
	if err := s.reloader.Reload(ctx); err != nil {
		slog.ErrorContext(ctx, "admin-api: reload after mutation failed", "error", err)
	}
}

// marshalResult serializes a value to JSON string. Returns ("", errorMsg) on failure.
func marshalResult(v interface{}) (string, string) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Sprintf("marshal result: %s", err)
	}
	return string(data), ""
}

// stringArg extracts a string argument from the args map.
func stringArg(args map[string]interface{}, key string) (string, bool) {
	v, ok := args[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// numericArg extracts a numeric argument (JSON numbers are float64).
func numericArg(args map[string]interface{}, key string) (float64, bool) {
	v, ok := args[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(n, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

// remarshal converts a map[string]interface{} to a typed struct via JSON round-trip.
func remarshal(src interface{}, dst interface{}) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

// mustSchema marshals a schema definition to json.RawMessage, panicking on error.
func mustSchema(schema map[string]interface{}) json.RawMessage {
	data, err := json.Marshal(schema)
	if err != nil {
		panic(fmt.Sprintf("admin_mcp: invalid schema: %v", err))
	}
	return data
}

// toolSuccessResponse builds a successful MCP tool call response.
func toolSuccessResponse(id interface{}, text string) *mcp.Response {
	result, _ := json.Marshal(mcp.ToolCallResult{
		Content: []mcp.ToolContent{{Type: "text", Text: text}},
	})
	return &mcp.Response{JSONRPC: "2.0", ID: id, Result: result}
}

// toolErrorResponse builds an MCP tool call response with isError: true.
func toolErrorResponse(id interface{}, errMsg string) *mcp.Response {
	result, _ := json.Marshal(mcp.ToolCallResult{
		Content: []mcp.ToolContent{{Type: "text", Text: errMsg}},
		IsError: true,
	})
	return &mcp.Response{JSONRPC: "2.0", ID: id, Result: result}
}

package admin_mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/mcp"
)

// ---------------------------------------------------------------------------
// Mock implementations
// ---------------------------------------------------------------------------

type mockAgentManager struct {
	agents    []AgentInfo
	details   map[string]*AgentDetail
	created   *AgentDetail
	updated   *AgentDetail
	deleted   string
	createErr error
	updateErr error
	deleteErr error
	listErr   error
	getErr    error
}

func (m *mockAgentManager) ListAgents(_ context.Context) ([]AgentInfo, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.agents, nil
}

func (m *mockAgentManager) GetAgent(_ context.Context, name string) (*AgentDetail, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.details[name], nil
}

func (m *mockAgentManager) CreateAgent(_ context.Context, req CreateAgentRequest) (*AgentDetail, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	detail := &AgentDetail{
		AgentInfo: AgentInfo{Name: req.Name},
		SystemPrompt: req.SystemPrompt,
		Lifecycle:    req.Lifecycle,
		MCPServers:   req.MCPServers,
	}
	m.created = detail
	return detail, nil
}

func (m *mockAgentManager) UpdateAgent(_ context.Context, name string, req CreateAgentRequest) (*AgentDetail, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	detail := &AgentDetail{
		AgentInfo: AgentInfo{Name: name},
		SystemPrompt: req.SystemPrompt,
	}
	m.updated = detail
	return detail, nil
}

func (m *mockAgentManager) DeleteAgent(_ context.Context, name string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deleted = name
	return nil
}

type mockModelManager struct {
	models    []ModelResponse
	created   *ModelResponse
	updated   *ModelResponse
	deleted   string
	createErr error
	updateErr error
	deleteErr error
	listErr   error
}

func (m *mockModelManager) ListModels(_ context.Context) ([]ModelResponse, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.models, nil
}

func (m *mockModelManager) CreateModel(_ context.Context, req CreateModelRequest) (*ModelResponse, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	resp := &ModelResponse{ID: 1, Name: req.Name, Type: req.Type, ModelName: req.ModelName}
	m.created = resp
	return resp, nil
}

func (m *mockModelManager) UpdateModel(_ context.Context, name string, req CreateModelRequest) (*ModelResponse, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	resp := &ModelResponse{ID: 1, Name: name, Type: req.Type, ModelName: req.ModelName}
	m.updated = resp
	return resp, nil
}

func (m *mockModelManager) DeleteModel(_ context.Context, name string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deleted = name
	return nil
}

type mockTriggerManager struct {
	triggers  []TriggerResponse
	created   *TriggerResponse
	updated   *TriggerResponse
	deletedID uint
	createErr error
	updateErr error
	deleteErr error
	listErr   error
}

func (m *mockTriggerManager) ListTriggers(_ context.Context) ([]TriggerResponse, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.triggers, nil
}

func (m *mockTriggerManager) CreateTrigger(_ context.Context, req CreateTriggerRequest) (*TriggerResponse, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	resp := &TriggerResponse{ID: 1, Type: req.Type, Title: req.Title, AgentID: req.AgentID}
	m.created = resp
	return resp, nil
}

func (m *mockTriggerManager) UpdateTrigger(_ context.Context, id uint, req CreateTriggerRequest) (*TriggerResponse, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	resp := &TriggerResponse{ID: id, Type: req.Type, Title: req.Title}
	m.updated = resp
	return resp, nil
}

func (m *mockTriggerManager) DeleteTrigger(_ context.Context, id uint) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deletedID = id
	return nil
}

type mockMCPServerLister struct {
	servers []MCPServerResponse
	listErr error
}

func (m *mockMCPServerLister) ListMCPServers(_ context.Context) ([]MCPServerResponse, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.servers, nil
}

type mockToolMetadataProvider struct {
	metadata []ToolMetadataResponse
}

func (m *mockToolMetadataProvider) GetAllToolMetadata() []ToolMetadataResponse {
	return m.metadata
}

type mockConfigExporter struct {
	yamlData  []byte
	imported  []byte
	exportErr error
	importErr error
}

func (m *mockConfigExporter) ExportYAML(_ context.Context) ([]byte, error) {
	if m.exportErr != nil {
		return nil, m.exportErr
	}
	return m.yamlData, nil
}

func (m *mockConfigExporter) ImportYAML(_ context.Context, data []byte) error {
	if m.importErr != nil {
		return m.importErr
	}
	m.imported = data
	return nil
}

type mockReloader struct {
	callCount int
	err       error
}

func (m *mockReloader) Reload(_ context.Context) error {
	m.callCount++
	return m.err
}

// ---------------------------------------------------------------------------
// Helper: call a tool via the MCP protocol
// ---------------------------------------------------------------------------

func callTool(t *testing.T, s *Server, toolName string, args map[string]interface{}) (string, bool) {
	t.Helper()
	resp, err := s.Handle(context.Background(), &mcp.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      toolName,
			"arguments": args,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Nil(t, resp.Error)

	var result mcp.ToolCallResult
	require.NoError(t, json.Unmarshal(resp.Result, &result))
	require.NotEmpty(t, result.Content)
	return result.Content[0].Text, result.IsError
}

func newTestServer(cfg ServerConfig) *Server {
	return NewServer(cfg)
}

// ---------------------------------------------------------------------------
// Tests: Protocol
// ---------------------------------------------------------------------------

func TestServer_Handle_Initialize(t *testing.T) {
	s := newTestServer(ServerConfig{})
	resp, err := s.Handle(context.Background(), &mcp.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Result, &result))
	assert.Equal(t, "2024-11-05", result["protocolVersion"])

	serverInfo := result["serverInfo"].(map[string]interface{})
	assert.Equal(t, "admin-api", serverInfo["name"])
}

func TestServer_Handle_ToolsList_Returns17Tools(t *testing.T) {
	s := newTestServer(ServerConfig{})
	resp, err := s.Handle(context.Background(), &mcp.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	var result mcp.ToolsListResult
	require.NoError(t, json.Unmarshal(resp.Result, &result))
	assert.Len(t, result.Tools, 17)

	// Verify all expected tool names
	names := make(map[string]bool)
	for _, tool := range result.Tools {
		names[tool.Name] = true
	}
	expected := []string{
		"list_agents", "get_agent", "create_agent", "update_agent", "delete_agent",
		"list_models", "create_model", "update_model", "delete_model",
		"list_triggers", "create_trigger", "update_trigger", "delete_trigger",
		"list_mcp_servers", "list_tools",
		"export_config", "import_config",
	}
	for _, name := range expected {
		assert.True(t, names[name], "missing tool: %s", name)
	}
}

func TestServer_Handle_UnknownMethod(t *testing.T) {
	s := newTestServer(ServerConfig{})
	resp, err := s.Handle(context.Background(), &mcp.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "unknown/method",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Error)
	assert.Equal(t, -32601, resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "method not found")
}

func TestServer_Handle_UnknownTool(t *testing.T) {
	s := newTestServer(ServerConfig{})
	text, isErr := callTool(t, s, "nonexistent_tool", nil)
	assert.True(t, isErr)
	assert.Contains(t, text, "unknown tool")
}

// ---------------------------------------------------------------------------
// Tests: Agent tools
// ---------------------------------------------------------------------------

func TestServer_ListAgents_HappyPath(t *testing.T) {
	agents := &mockAgentManager{
		agents: []AgentInfo{
			{Name: "bot-1", ToolsCount: 3},
			{Name: "bot-2", ToolsCount: 5, Kit: "developer"},
		},
	}
	s := newTestServer(ServerConfig{AgentManager: agents})

	text, isErr := callTool(t, s, "list_agents", nil)
	assert.False(t, isErr)

	var result []AgentInfo
	require.NoError(t, json.Unmarshal([]byte(text), &result))
	assert.Len(t, result, 2)
	assert.Equal(t, "bot-1", result[0].Name)
}

func TestServer_ListAgents_Error(t *testing.T) {
	agents := &mockAgentManager{listErr: fmt.Errorf("db error")}
	s := newTestServer(ServerConfig{AgentManager: agents})

	text, isErr := callTool(t, s, "list_agents", nil)
	assert.True(t, isErr)
	assert.Contains(t, text, "db error")
}

func TestServer_ListAgents_NilManager(t *testing.T) {
	s := newTestServer(ServerConfig{})
	text, isErr := callTool(t, s, "list_agents", nil)
	assert.True(t, isErr)
	assert.Contains(t, text, "not configured")
}

func TestServer_GetAgent_HappyPath(t *testing.T) {
	agents := &mockAgentManager{
		details: map[string]*AgentDetail{
			"my-bot": {
				AgentInfo:    AgentInfo{Name: "my-bot"},
				SystemPrompt: "You are helpful.",
				MCPServers:   []string{"admin-api"},
			},
		},
	}
	s := newTestServer(ServerConfig{AgentManager: agents})

	text, isErr := callTool(t, s, "get_agent", map[string]interface{}{"name": "my-bot"})
	assert.False(t, isErr)

	var result AgentDetail
	require.NoError(t, json.Unmarshal([]byte(text), &result))
	assert.Equal(t, "my-bot", result.Name)
	assert.Equal(t, "You are helpful.", result.SystemPrompt)
}

func TestServer_GetAgent_NotFound(t *testing.T) {
	agents := &mockAgentManager{details: map[string]*AgentDetail{}}
	s := newTestServer(ServerConfig{AgentManager: agents})

	text, isErr := callTool(t, s, "get_agent", map[string]interface{}{"name": "nonexistent"})
	assert.True(t, isErr)
	assert.Contains(t, text, "not found")
}

func TestServer_GetAgent_MissingName(t *testing.T) {
	agents := &mockAgentManager{}
	s := newTestServer(ServerConfig{AgentManager: agents})

	text, isErr := callTool(t, s, "get_agent", map[string]interface{}{})
	assert.True(t, isErr)
	assert.Contains(t, text, "missing required parameter: name")
}

func TestServer_CreateAgent_HappyPath(t *testing.T) {
	agents := &mockAgentManager{}
	reloader := &mockReloader{}
	s := newTestServer(ServerConfig{AgentManager: agents, Reloader: reloader})

	text, isErr := callTool(t, s, "create_agent", map[string]interface{}{
		"name":          "new-bot",
		"system_prompt": "Be helpful",
		"lifecycle":     "persistent",
		"mcp_servers":   []interface{}{"admin-api"},
	})
	assert.False(t, isErr)

	var result AgentDetail
	require.NoError(t, json.Unmarshal([]byte(text), &result))
	assert.Equal(t, "new-bot", result.Name)
	assert.NotNil(t, agents.created)
	assert.Equal(t, 1, reloader.callCount, "reloader should be called after create")
}

func TestServer_CreateAgent_MissingName(t *testing.T) {
	agents := &mockAgentManager{}
	s := newTestServer(ServerConfig{AgentManager: agents})

	text, isErr := callTool(t, s, "create_agent", map[string]interface{}{
		"system_prompt": "Be helpful",
	})
	assert.True(t, isErr)
	assert.Contains(t, text, "missing required parameter: name")
}

func TestServer_CreateAgent_Error(t *testing.T) {
	agents := &mockAgentManager{createErr: fmt.Errorf("duplicate")}
	s := newTestServer(ServerConfig{AgentManager: agents})

	text, isErr := callTool(t, s, "create_agent", map[string]interface{}{
		"name":          "dup-bot",
		"system_prompt": "test",
	})
	assert.True(t, isErr)
	assert.Contains(t, text, "duplicate")
}

func TestServer_UpdateAgent_HappyPath(t *testing.T) {
	agents := &mockAgentManager{}
	reloader := &mockReloader{}
	s := newTestServer(ServerConfig{AgentManager: agents, Reloader: reloader})

	text, isErr := callTool(t, s, "update_agent", map[string]interface{}{
		"name":          "my-bot",
		"system_prompt": "Updated prompt",
	})
	assert.False(t, isErr)

	var result AgentDetail
	require.NoError(t, json.Unmarshal([]byte(text), &result))
	assert.Equal(t, "my-bot", result.Name)
	assert.Equal(t, 1, reloader.callCount)
}

func TestServer_DeleteAgent_HappyPath(t *testing.T) {
	agents := &mockAgentManager{}
	reloader := &mockReloader{}
	s := newTestServer(ServerConfig{AgentManager: agents, Reloader: reloader})

	text, isErr := callTool(t, s, "delete_agent", map[string]interface{}{"name": "my-bot"})
	assert.False(t, isErr)
	assert.Contains(t, text, "my-bot")
	assert.Equal(t, "my-bot", agents.deleted)
	assert.Equal(t, 1, reloader.callCount)
}

func TestServer_DeleteAgent_Error(t *testing.T) {
	agents := &mockAgentManager{deleteErr: fmt.Errorf("not found")}
	s := newTestServer(ServerConfig{AgentManager: agents})

	text, isErr := callTool(t, s, "delete_agent", map[string]interface{}{"name": "x"})
	assert.True(t, isErr)
	assert.Contains(t, text, "not found")
}

// ---------------------------------------------------------------------------
// Tests: Model tools
// ---------------------------------------------------------------------------

func TestServer_ListModels_HappyPath(t *testing.T) {
	models := &mockModelManager{
		models: []ModelResponse{
			{ID: 1, Name: "gpt4", Type: "openai_compatible", ModelName: "gpt-4o"},
		},
	}
	s := newTestServer(ServerConfig{ModelManager: models})

	text, isErr := callTool(t, s, "list_models", nil)
	assert.False(t, isErr)

	var result []ModelResponse
	require.NoError(t, json.Unmarshal([]byte(text), &result))
	assert.Len(t, result, 1)
	assert.Equal(t, "gpt4", result[0].Name)
}

func TestServer_CreateModel_HappyPath(t *testing.T) {
	models := &mockModelManager{}
	reloader := &mockReloader{}
	s := newTestServer(ServerConfig{ModelManager: models, Reloader: reloader})

	text, isErr := callTool(t, s, "create_model", map[string]interface{}{
		"name":       "my-model",
		"type":       "openai_compatible",
		"model_name": "gpt-4o",
	})
	assert.False(t, isErr)

	var result ModelResponse
	require.NoError(t, json.Unmarshal([]byte(text), &result))
	assert.Equal(t, "my-model", result.Name)
	assert.Equal(t, 1, reloader.callCount)
}

func TestServer_CreateModel_MissingName(t *testing.T) {
	models := &mockModelManager{}
	s := newTestServer(ServerConfig{ModelManager: models})

	text, isErr := callTool(t, s, "create_model", map[string]interface{}{
		"type":       "openai_compatible",
		"model_name": "gpt-4o",
	})
	assert.True(t, isErr)
	assert.Contains(t, text, "missing required parameter: name")
}

func TestServer_UpdateModel_HappyPath(t *testing.T) {
	models := &mockModelManager{}
	reloader := &mockReloader{}
	s := newTestServer(ServerConfig{ModelManager: models, Reloader: reloader})

	text, isErr := callTool(t, s, "update_model", map[string]interface{}{
		"name":       "my-model",
		"model_name": "gpt-4o-mini",
	})
	assert.False(t, isErr)

	var result ModelResponse
	require.NoError(t, json.Unmarshal([]byte(text), &result))
	assert.Equal(t, "my-model", result.Name)
	assert.Equal(t, 1, reloader.callCount)
}

func TestServer_DeleteModel_HappyPath(t *testing.T) {
	models := &mockModelManager{}
	reloader := &mockReloader{}
	s := newTestServer(ServerConfig{ModelManager: models, Reloader: reloader})

	text, isErr := callTool(t, s, "delete_model", map[string]interface{}{"name": "old-model"})
	assert.False(t, isErr)
	assert.Contains(t, text, "old-model")
	assert.Equal(t, "old-model", models.deleted)
	assert.Equal(t, 1, reloader.callCount)
}

func TestServer_DeleteModel_NilManager(t *testing.T) {
	s := newTestServer(ServerConfig{})
	text, isErr := callTool(t, s, "delete_model", map[string]interface{}{"name": "x"})
	assert.True(t, isErr)
	assert.Contains(t, text, "not configured")
}

// ---------------------------------------------------------------------------
// Tests: Trigger tools
// ---------------------------------------------------------------------------

func TestServer_ListTriggers_HappyPath(t *testing.T) {
	triggers := &mockTriggerManager{
		triggers: []TriggerResponse{
			{ID: 1, Type: "cron", Title: "Daily Report", AgentName: "reporter"},
		},
	}
	s := newTestServer(ServerConfig{TriggerManager: triggers})

	text, isErr := callTool(t, s, "list_triggers", nil)
	assert.False(t, isErr)

	var result []TriggerResponse
	require.NoError(t, json.Unmarshal([]byte(text), &result))
	assert.Len(t, result, 1)
	assert.Equal(t, "Daily Report", result[0].Title)
}

func TestServer_CreateTrigger_HappyPath(t *testing.T) {
	triggers := &mockTriggerManager{}
	reloader := &mockReloader{}
	s := newTestServer(ServerConfig{TriggerManager: triggers, Reloader: reloader})

	text, isErr := callTool(t, s, "create_trigger", map[string]interface{}{
		"type":     "cron",
		"title":    "Hourly Check",
		"agent_id": float64(1),
		"schedule": "0 * * * *",
	})
	assert.False(t, isErr)

	var result TriggerResponse
	require.NoError(t, json.Unmarshal([]byte(text), &result))
	assert.Equal(t, "Hourly Check", result.Title)
	assert.Equal(t, 1, reloader.callCount)
}

func TestServer_UpdateTrigger_HappyPath(t *testing.T) {
	triggers := &mockTriggerManager{}
	reloader := &mockReloader{}
	s := newTestServer(ServerConfig{TriggerManager: triggers, Reloader: reloader})

	text, isErr := callTool(t, s, "update_trigger", map[string]interface{}{
		"id":    float64(5),
		"title": "Updated Trigger",
	})
	assert.False(t, isErr)

	var result TriggerResponse
	require.NoError(t, json.Unmarshal([]byte(text), &result))
	assert.Equal(t, uint(5), result.ID)
	assert.Equal(t, 1, reloader.callCount)
}

func TestServer_UpdateTrigger_MissingID(t *testing.T) {
	triggers := &mockTriggerManager{}
	s := newTestServer(ServerConfig{TriggerManager: triggers})

	text, isErr := callTool(t, s, "update_trigger", map[string]interface{}{
		"title": "No ID",
	})
	assert.True(t, isErr)
	assert.Contains(t, text, "missing required parameter: id")
}

func TestServer_DeleteTrigger_HappyPath(t *testing.T) {
	triggers := &mockTriggerManager{}
	reloader := &mockReloader{}
	s := newTestServer(ServerConfig{TriggerManager: triggers, Reloader: reloader})

	text, isErr := callTool(t, s, "delete_trigger", map[string]interface{}{"id": float64(3)})
	assert.False(t, isErr)
	assert.Contains(t, text, "3")
	assert.Equal(t, uint(3), triggers.deletedID)
	assert.Equal(t, 1, reloader.callCount)
}

// ---------------------------------------------------------------------------
// Tests: MCP Server tools
// ---------------------------------------------------------------------------

func TestServer_ListMCPServers_HappyPath(t *testing.T) {
	mcpServers := &mockMCPServerLister{
		servers: []MCPServerResponse{
			{ID: 1, Name: "github-mcp", Type: "stdio"},
		},
	}
	s := newTestServer(ServerConfig{MCPServerLister: mcpServers})

	text, isErr := callTool(t, s, "list_mcp_servers", nil)
	assert.False(t, isErr)

	var result []MCPServerResponse
	require.NoError(t, json.Unmarshal([]byte(text), &result))
	assert.Len(t, result, 1)
	assert.Equal(t, "github-mcp", result[0].Name)
}

func TestServer_ListMCPServers_NilLister(t *testing.T) {
	s := newTestServer(ServerConfig{})
	text, isErr := callTool(t, s, "list_mcp_servers", nil)
	assert.True(t, isErr)
	assert.Contains(t, text, "not configured")
}

// ---------------------------------------------------------------------------
// Tests: Tool metadata
// ---------------------------------------------------------------------------

func TestServer_ListTools_HappyPath(t *testing.T) {
	toolMeta := &mockToolMetadataProvider{
		metadata: []ToolMetadataResponse{
			{Name: "read_file", Description: "Read a file", SecurityZone: "safe"},
			{Name: "execute_command", Description: "Execute a command", SecurityZone: "dangerous"},
		},
	}
	s := newTestServer(ServerConfig{ToolMetadataProvider: toolMeta})

	text, isErr := callTool(t, s, "list_tools", nil)
	assert.False(t, isErr)

	var result []ToolMetadataResponse
	require.NoError(t, json.Unmarshal([]byte(text), &result))
	assert.Len(t, result, 2)
}

// ---------------------------------------------------------------------------
// Tests: Config tools
// ---------------------------------------------------------------------------

func TestServer_ExportConfig_HappyPath(t *testing.T) {
	cfg := &mockConfigExporter{yamlData: []byte("agents:\n  - name: bot\n")}
	s := newTestServer(ServerConfig{ConfigExporter: cfg})

	text, isErr := callTool(t, s, "export_config", nil)
	assert.False(t, isErr)
	assert.Contains(t, text, "agents:")
	assert.Contains(t, text, "bot")
}

func TestServer_ExportConfig_Error(t *testing.T) {
	cfg := &mockConfigExporter{exportErr: fmt.Errorf("export failed")}
	s := newTestServer(ServerConfig{ConfigExporter: cfg})

	text, isErr := callTool(t, s, "export_config", nil)
	assert.True(t, isErr)
	assert.Contains(t, text, "export failed")
}

func TestServer_ImportConfig_HappyPath(t *testing.T) {
	cfg := &mockConfigExporter{}
	reloader := &mockReloader{}
	s := newTestServer(ServerConfig{ConfigExporter: cfg, Reloader: reloader})

	text, isErr := callTool(t, s, "import_config", map[string]interface{}{
		"yaml_content": "agents:\n  - name: imported-bot\n",
	})
	assert.False(t, isErr)
	assert.Contains(t, text, "imported")
	assert.Contains(t, string(cfg.imported), "imported-bot")
	assert.Equal(t, 1, reloader.callCount)
}

func TestServer_ImportConfig_MissingYAML(t *testing.T) {
	cfg := &mockConfigExporter{}
	s := newTestServer(ServerConfig{ConfigExporter: cfg})

	text, isErr := callTool(t, s, "import_config", map[string]interface{}{})
	assert.True(t, isErr)
	assert.Contains(t, text, "missing required parameter: yaml_content")
}

func TestServer_ImportConfig_Error(t *testing.T) {
	cfg := &mockConfigExporter{importErr: fmt.Errorf("invalid yaml")}
	s := newTestServer(ServerConfig{ConfigExporter: cfg})

	text, isErr := callTool(t, s, "import_config", map[string]interface{}{
		"yaml_content": "bad yaml",
	})
	assert.True(t, isErr)
	assert.Contains(t, text, "invalid yaml")
}

// ---------------------------------------------------------------------------
// Tests: Reload behavior
// ---------------------------------------------------------------------------

func TestServer_MutatingTools_CallReloader(t *testing.T) {
	reloader := &mockReloader{}
	agents := &mockAgentManager{details: map[string]*AgentDetail{}}
	models := &mockModelManager{}
	triggers := &mockTriggerManager{}
	cfg := &mockConfigExporter{}

	s := newTestServer(ServerConfig{
		AgentManager:   agents,
		ModelManager:   models,
		TriggerManager: triggers,
		ConfigExporter: cfg,
		Reloader:       reloader,
	})

	// Each mutating tool should call reloader once
	mutatingCalls := []struct {
		tool string
		args map[string]interface{}
	}{
		{"create_agent", map[string]interface{}{"name": "a1", "system_prompt": "test"}},
		{"update_agent", map[string]interface{}{"name": "a1"}},
		{"delete_agent", map[string]interface{}{"name": "a1"}},
		{"create_model", map[string]interface{}{"name": "m1", "type": "openai_compatible", "model_name": "gpt-4o"}},
		{"update_model", map[string]interface{}{"name": "m1"}},
		{"delete_model", map[string]interface{}{"name": "m1"}},
		{"create_trigger", map[string]interface{}{"type": "cron", "title": "t1", "agent_id": float64(1)}},
		{"update_trigger", map[string]interface{}{"id": float64(1), "title": "t1"}},
		{"delete_trigger", map[string]interface{}{"id": float64(1)}},
		{"import_config", map[string]interface{}{"yaml_content": "agents: []"}},
	}

	for _, call := range mutatingCalls {
		callTool(t, s, call.tool, call.args)
	}

	assert.Equal(t, len(mutatingCalls), reloader.callCount,
		"each mutating tool should call reloader exactly once")
}

func TestServer_ReadOnlyTools_DoNotCallReloader(t *testing.T) {
	reloader := &mockReloader{}
	agents := &mockAgentManager{
		agents:  []AgentInfo{{Name: "bot"}},
		details: map[string]*AgentDetail{"bot": {AgentInfo: AgentInfo{Name: "bot"}}},
	}
	models := &mockModelManager{models: []ModelResponse{{Name: "m"}}}
	triggers := &mockTriggerManager{triggers: []TriggerResponse{{Title: "t"}}}
	mcpServers := &mockMCPServerLister{servers: []MCPServerResponse{{Name: "s"}}}
	toolMeta := &mockToolMetadataProvider{metadata: []ToolMetadataResponse{{Name: "tool"}}}
	cfg := &mockConfigExporter{yamlData: []byte("test")}

	s := newTestServer(ServerConfig{
		AgentManager:         agents,
		ModelManager:         models,
		TriggerManager:       triggers,
		MCPServerLister:      mcpServers,
		ToolMetadataProvider: toolMeta,
		ConfigExporter:       cfg,
		Reloader:             reloader,
	})

	readOnlyCalls := []struct {
		tool string
		args map[string]interface{}
	}{
		{"list_agents", nil},
		{"get_agent", map[string]interface{}{"name": "bot"}},
		{"list_models", nil},
		{"list_triggers", nil},
		{"list_mcp_servers", nil},
		{"list_tools", nil},
		{"export_config", nil},
	}

	for _, call := range readOnlyCalls {
		callTool(t, s, call.tool, call.args)
	}

	assert.Equal(t, 0, reloader.callCount,
		"read-only tools should never call reloader")
}

// ---------------------------------------------------------------------------
// Tests: Integration with Client
// ---------------------------------------------------------------------------

func TestServer_FullClientIntegration(t *testing.T) {
	agents := &mockAgentManager{
		agents: []AgentInfo{{Name: "test-bot", ToolsCount: 2}},
	}
	s := newTestServer(ServerConfig{AgentManager: agents})

	transport := mcp.NewInProcessTransport(s.Handle)
	client := mcp.NewClient("admin-api", transport)

	err := client.Connect(context.Background())
	require.NoError(t, err)
	assert.True(t, client.IsConnected())

	tools := client.ListTools()
	assert.Len(t, tools, 17)

	// Call list_agents through the client
	result, isError, err := client.CallTool(context.Background(), "list_agents", nil)
	require.NoError(t, err)
	assert.False(t, isError)

	var agentList []AgentInfo
	require.NoError(t, json.Unmarshal([]byte(result), &agentList))
	assert.Len(t, agentList, 1)
	assert.Equal(t, "test-bot", agentList[0].Name)
}

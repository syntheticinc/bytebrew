package app

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	// Migrate only tables needed for config import/export
	// (full AutoMigrate uses PostgreSQL-specific syntax like DEFAULT now()).
	require.NoError(t, db.AutoMigrate(
		&models.LLMProviderModel{},
		&models.MCPServerModel{},
		&models.AgentModel{},
		&models.AgentToolModel{},
		&models.AgentSpawnTarget{},
		&models.AgentEscalation{},
		&models.AgentEscalationTrigger{},
		&models.AgentMCPServer{},
		&models.TriggerModel{},
	))
	return db
}

func seedTestData(t *testing.T, db *gorm.DB) {
	t.Helper()

	// Model
	model := models.LLMProviderModel{
		Name:            "gpt-4o",
		Type:            "openai_compatible",
		BaseURL:         "https://api.openai.com/v1",
		ModelName:       "gpt-4o",
		APIKeyEncrypted: "encrypted-secret",
	}
	require.NoError(t, db.Create(&model).Error)

	// MCP Server
	envJSON, _ := json.Marshal(map[string]string{"API_KEY": "secret123"})
	argsJSON, _ := json.Marshal([]string{"--port", "3000"})
	mcpServer := models.MCPServerModel{
		Name:    "shop-api",
		Type:    "http",
		URL:     "http://shop-api:3000/mcp",
		Args:    string(argsJSON),
		EnvVars: string(envJSON),
	}
	require.NoError(t, db.Create(&mcpServer).Error)

	// Agent
	agent := models.AgentModel{
		Name:           "sales",
		ModelID:        &model.ID,
		SystemPrompt:   "You are a sales assistant.",
		Lifecycle:      "persistent",
		ToolExecution:  "sequential",
		MaxSteps:       30,
		MaxContextSize: 16000,
		ConfirmBefore:  "delete_order,refund",
	}
	require.NoError(t, db.Create(&agent).Error)

	// Agent tools
	require.NoError(t, db.Create(&models.AgentToolModel{
		AgentID: agent.ID, ToolType: "builtin", ToolName: "web_search", SortOrder: 0,
	}).Error)
	require.NoError(t, db.Create(&models.AgentToolModel{
		AgentID: agent.ID, ToolType: "builtin", ToolName: "ask_user", SortOrder: 1,
	}).Error)

	// Agent MCP server link
	require.NoError(t, db.Create(&models.AgentMCPServer{
		AgentID: agent.ID, MCPServerID: mcpServer.ID,
	}).Error)

	// Second agent (spawn target)
	researcher := models.AgentModel{
		Name:           "researcher",
		SystemPrompt:   "You research things.",
		Lifecycle:      "spawn",
		ToolExecution:  "parallel",
		MaxSteps:       20,
		MaxContextSize: 8000,
	}
	require.NoError(t, db.Create(&researcher).Error)

	// Spawn target
	require.NoError(t, db.Create(&models.AgentSpawnTarget{
		AgentID: agent.ID, TargetAgentID: researcher.ID,
	}).Error)

	// Escalation
	esc := models.AgentEscalation{
		AgentID:    agent.ID,
		Action:     "transfer_to_human",
		WebhookURL: "https://hooks.example.com/escalate",
	}
	require.NoError(t, db.Create(&esc).Error)
	require.NoError(t, db.Create(&models.AgentEscalationTrigger{
		EscalationID: esc.ID, Keyword: "angry",
	}).Error)

	// Trigger
	require.NoError(t, db.Create(&models.TriggerModel{
		Type: "cron", Title: "Morning report", AgentID: agent.ID,
		Schedule: "0 9 * * *", Description: "Daily report", Enabled: true,
	}).Error)
}

func TestExportYAML(t *testing.T) {
	db := setupTestDB(t)
	seedTestData(t, db)
	adapter := &configImportExportHTTPAdapter{db: db}

	data, err := adapter.ExportYAML(context.Background())
	require.NoError(t, err)

	output := string(data)

	// Header present
	assert.True(t, strings.HasPrefix(output, "# ByteBrew Engine Configuration"))

	// Parse YAML part (skip header comments)
	var cfg configYAML
	require.NoError(t, yaml.Unmarshal(data, &cfg))

	// Agents
	require.Len(t, cfg.Agents, 2)
	sales := findAgentYAML(cfg.Agents, "sales")
	require.NotNil(t, sales)
	assert.Equal(t, "You are a sales assistant.", sales.SystemPrompt)
	assert.Equal(t, "gpt-4o", sales.ModelName)
	assert.Equal(t, "persistent", sales.Lifecycle)
	assert.Equal(t, 30, sales.MaxSteps)
	assert.Equal(t, []string{"web_search", "ask_user"}, sales.Tools)
	assert.Equal(t, []string{"researcher"}, sales.CanSpawn)
	assert.Equal(t, []string{"shop-api"}, sales.MCPServers)
	assert.Equal(t, []string{"delete_order", "refund"}, sales.ConfirmBefore)

	// Escalation
	require.NotNil(t, sales.Escalation)
	assert.Equal(t, "transfer_to_human", sales.Escalation.Action)
	assert.Equal(t, []string{"angry"}, sales.Escalation.Triggers)

	// Models — API key must NOT be present
	require.Len(t, cfg.Models, 1)
	assert.Equal(t, "gpt-4o", cfg.Models[0].Name)
	assert.Equal(t, "openai_compatible", cfg.Models[0].Type)
	assert.NotContains(t, output, "encrypted-secret")

	// MCP Servers — env vars must be masked
	require.Len(t, cfg.MCPServers, 1)
	assert.Equal(t, "shop-api", cfg.MCPServers[0].Name)
	assert.Equal(t, "${API_KEY}", cfg.MCPServers[0].EnvVars["API_KEY"])

	// Triggers
	require.Len(t, cfg.Triggers, 1)
	assert.Equal(t, "Morning report", cfg.Triggers[0].Title)
	assert.Equal(t, "sales", cfg.Triggers[0].AgentName)
	assert.Equal(t, "0 9 * * *", cfg.Triggers[0].Schedule)
}

func TestImportYAML(t *testing.T) {
	db := setupTestDB(t)
	adapter := &configImportExportHTTPAdapter{db: db}

	yamlData := `
agents:
  - name: "support"
    system_prompt: "You help customers."
    model_name: "claude-3"
    lifecycle: "persistent"
    tool_execution: "sequential"
    max_steps: 25
    max_context_size: 12000
    tools:
      - web_search
    can_spawn: []
    mcp_servers: []

models:
  - name: "claude-3"
    type: "anthropic"
    base_url: "https://api.anthropic.com"
    model_name: "claude-3-opus-20240229"

mcp_servers:
  - name: "crm"
    type: "http"
    url: "http://crm:8080/mcp"

triggers:
  - title: "Hourly check"
    type: "cron"
    agent_name: "support"
    schedule: "0 * * * *"
    description: "Check tickets"
    enabled: true
`
	err := adapter.ImportYAML(context.Background(), []byte(yamlData))
	require.NoError(t, err)

	// Verify models in DB
	var llms []models.LLMProviderModel
	require.NoError(t, db.Find(&llms).Error)
	require.Len(t, llms, 1)
	assert.Equal(t, "claude-3", llms[0].Name)

	// Verify MCP servers
	var mcps []models.MCPServerModel
	require.NoError(t, db.Find(&mcps).Error)
	require.Len(t, mcps, 1)
	assert.Equal(t, "crm", mcps[0].Name)

	// Verify agents
	var agents []models.AgentModel
	require.NoError(t, db.Preload("Model").Preload("Tools").Find(&agents).Error)
	require.Len(t, agents, 1)
	assert.Equal(t, "support", agents[0].Name)
	assert.Equal(t, "claude-3", agents[0].Model.Name)
	require.Len(t, agents[0].Tools, 1)
	assert.Equal(t, "web_search", agents[0].Tools[0].ToolName)

	// Verify triggers
	var triggers []models.TriggerModel
	require.NoError(t, db.Find(&triggers).Error)
	require.Len(t, triggers, 1)
	assert.Equal(t, "Hourly check", triggers[0].Title)
	assert.Equal(t, agents[0].ID, triggers[0].AgentID)
}

func TestImportYAML_UpdateExisting(t *testing.T) {
	db := setupTestDB(t)
	adapter := &configImportExportHTTPAdapter{db: db}

	// First import
	yamlData := `
models:
  - name: "gpt-4o"
    type: "openai_compatible"
    base_url: "https://api.openai.com/v1"
    model_name: "gpt-4o"
agents:
  - name: "bot"
    system_prompt: "V1 prompt"
    model_name: "gpt-4o"
    lifecycle: "persistent"
    tool_execution: "sequential"
    max_steps: 10
    max_context_size: 8000
    tools:
      - web_search
`
	require.NoError(t, adapter.ImportYAML(context.Background(), []byte(yamlData)))

	// Second import with updated values
	yamlData2 := `
models:
  - name: "gpt-4o"
    type: "openai_compatible"
    base_url: "https://api.openai.com/v2"
    model_name: "gpt-4o-2024-08-06"
agents:
  - name: "bot"
    system_prompt: "V2 prompt"
    model_name: "gpt-4o"
    lifecycle: "persistent"
    tool_execution: "parallel"
    max_steps: 50
    max_context_size: 32000
    tools:
      - ask_user
      - web_search
`
	require.NoError(t, adapter.ImportYAML(context.Background(), []byte(yamlData2)))

	// Should still have 1 model (updated)
	var llms []models.LLMProviderModel
	require.NoError(t, db.Find(&llms).Error)
	require.Len(t, llms, 1)
	assert.Equal(t, "https://api.openai.com/v2", llms[0].BaseURL)
	assert.Equal(t, "gpt-4o-2024-08-06", llms[0].ModelName)

	// Should still have 1 agent (updated)
	var agents []models.AgentModel
	require.NoError(t, db.Preload("Tools").Find(&agents).Error)
	require.Len(t, agents, 1)
	assert.Equal(t, "V2 prompt", agents[0].SystemPrompt)
	assert.Equal(t, "parallel", agents[0].ToolExecution)
	assert.Equal(t, 50, agents[0].MaxSteps)
	require.Len(t, agents[0].Tools, 2)
}

func TestImportYAML_InvalidYAML(t *testing.T) {
	db := setupTestDB(t)
	adapter := &configImportExportHTTPAdapter{db: db}

	err := adapter.ImportYAML(context.Background(), []byte("{{invalid"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse yaml")
}

func TestImportYAML_MissingModelReference(t *testing.T) {
	db := setupTestDB(t)
	adapter := &configImportExportHTTPAdapter{db: db}

	yamlData := `
agents:
  - name: "bot"
    system_prompt: "Test"
    model_name: "nonexistent"
    lifecycle: "persistent"
    tool_execution: "sequential"
    max_steps: 10
    max_context_size: 8000
`
	err := adapter.ImportYAML(context.Background(), []byte(yamlData))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestExportImportRoundTrip(t *testing.T) {
	db := setupTestDB(t)
	seedTestData(t, db)
	adapter := &configImportExportHTTPAdapter{db: db}

	// Export
	exported, err := adapter.ExportYAML(context.Background())
	require.NoError(t, err)

	// Import into fresh DB
	db2 := setupTestDB(t)
	adapter2 := &configImportExportHTTPAdapter{db: db2}
	require.NoError(t, adapter2.ImportYAML(context.Background(), exported))

	// Re-export from second DB
	exported2, err := adapter2.ExportYAML(context.Background())
	require.NoError(t, err)

	// Parse both and compare structure
	var cfg1, cfg2 configYAML
	require.NoError(t, yaml.Unmarshal(exported, &cfg1))
	require.NoError(t, yaml.Unmarshal(exported2, &cfg2))

	assert.Equal(t, len(cfg1.Agents), len(cfg2.Agents))
	assert.Equal(t, len(cfg1.Models), len(cfg2.Models))
	assert.Equal(t, len(cfg1.MCPServers), len(cfg2.MCPServers))
	assert.Equal(t, len(cfg1.Triggers), len(cfg2.Triggers))

	// Agent names match
	for _, a1 := range cfg1.Agents {
		a2 := findAgentYAML(cfg2.Agents, a1.Name)
		require.NotNil(t, a2, "agent %q missing after round-trip", a1.Name)
		assert.Equal(t, a1.SystemPrompt, a2.SystemPrompt)
		assert.Equal(t, a1.Lifecycle, a2.Lifecycle)
	}
}

func TestExportYAML_EmptyDB(t *testing.T) {
	db := setupTestDB(t)
	adapter := &configImportExportHTTPAdapter{db: db}

	data, err := adapter.ExportYAML(context.Background())
	require.NoError(t, err)

	var cfg configYAML
	require.NoError(t, yaml.Unmarshal(data, &cfg))
	assert.Empty(t, cfg.Agents)
	assert.Empty(t, cfg.Models)
	assert.Empty(t, cfg.MCPServers)
	assert.Empty(t, cfg.Triggers)
}

func TestSplitCSV(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty", "", nil},
		{"single", "web_search", []string{"web_search"}},
		{"multiple", "a,b,c", []string{"a", "b", "c"}},
		{"with spaces", " a , b , c ", []string{"a", "b", "c"}},
		{"trailing comma", "a,b,", []string{"a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitCSV(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsEnvPlaceholder(t *testing.T) {
	assert.True(t, isEnvPlaceholder("${API_KEY}"))
	assert.False(t, isEnvPlaceholder("real-value"))
	assert.False(t, isEnvPlaceholder("${partial"))
}

func findAgentYAML(agents []agentYAML, name string) *agentYAML {
	for i := range agents {
		if agents[i].Name == name {
			return &agents[i]
		}
	}
	return nil
}

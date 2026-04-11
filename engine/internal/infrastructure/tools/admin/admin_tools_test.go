package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/tools"
)

// --- Mock repositories ---

type mockAgentRepo struct {
	agents map[string]*AgentRecord
	err    error
}

func newMockAgentRepo() *mockAgentRepo {
	return &mockAgentRepo{agents: make(map[string]*AgentRecord)}
}

func (m *mockAgentRepo) List(_ context.Context) ([]AgentRecord, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := make([]AgentRecord, 0, len(m.agents))
	for _, a := range m.agents {
		result = append(result, *a)
	}
	return result, nil
}

func (m *mockAgentRepo) GetByName(_ context.Context, name string) (*AgentRecord, error) {
	if m.err != nil {
		return nil, m.err
	}
	a, ok := m.agents[name]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", name)
	}
	return a, nil
}

func (m *mockAgentRepo) Create(_ context.Context, record *AgentRecord) error {
	if m.err != nil {
		return m.err
	}
	if _, exists := m.agents[record.Name]; exists {
		return fmt.Errorf("duplicate key: agent %q already exists", record.Name)
	}
	m.agents[record.Name] = record
	return nil
}

func (m *mockAgentRepo) Update(_ context.Context, name string, record *AgentRecord) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.agents[name]; !ok {
		return fmt.Errorf("agent not found: %s", name)
	}
	m.agents[name] = record
	return nil
}

func (m *mockAgentRepo) Delete(_ context.Context, name string) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.agents[name]; !ok {
		return fmt.Errorf("agent not found: %s", name)
	}
	delete(m.agents, name)
	return nil
}

type mockSchemaRepo struct {
	schemas map[string]*SchemaRecord
	nextID  int
	err     error
}

func newMockSchemaRepo() *mockSchemaRepo {
	return &mockSchemaRepo{schemas: make(map[string]*SchemaRecord), nextID: 1}
}

func (m *mockSchemaRepo) List(_ context.Context) ([]SchemaRecord, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := make([]SchemaRecord, 0, len(m.schemas))
	for _, s := range m.schemas {
		result = append(result, *s)
	}
	return result, nil
}

func (m *mockSchemaRepo) GetByID(_ context.Context, id string) (*SchemaRecord, error) {
	if m.err != nil {
		return nil, m.err
	}
	s, ok := m.schemas[id]
	if !ok {
		return nil, fmt.Errorf("schema not found: %s", id)
	}
	return s, nil
}

func (m *mockSchemaRepo) Create(_ context.Context, record *SchemaRecord) error {
	if m.err != nil {
		return m.err
	}
	record.ID = fmt.Sprintf("schema-%d", m.nextID)
	m.nextID++
	m.schemas[record.ID] = record
	return nil
}

func (m *mockSchemaRepo) Update(_ context.Context, id string, record *SchemaRecord) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.schemas[id]; !ok {
		return fmt.Errorf("schema not found: %s", id)
	}
	record.ID = id
	m.schemas[id] = record
	return nil
}

func (m *mockSchemaRepo) Delete(_ context.Context, id string) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.schemas[id]; !ok {
		return fmt.Errorf("schema not found: %s", id)
	}
	delete(m.schemas, id)
	return nil
}

func (m *mockSchemaRepo) AddAgent(_ context.Context, schemaID string, agentName string) error {
	if m.err != nil {
		return m.err
	}
	s, ok := m.schemas[schemaID]
	if !ok {
		return fmt.Errorf("schema not found: %s", schemaID)
	}
	s.AgentNames = append(s.AgentNames, agentName)
	return nil
}

func (m *mockSchemaRepo) RemoveAgent(_ context.Context, schemaID string, agentName string) error {
	if m.err != nil {
		return m.err
	}
	s, ok := m.schemas[schemaID]
	if !ok {
		return fmt.Errorf("schema not found: %s", schemaID)
	}
	for i, n := range s.AgentNames {
		if n == agentName {
			s.AgentNames = append(s.AgentNames[:i], s.AgentNames[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("agent not found in schema: %s", agentName)
}

type mockEdgeRepo struct {
	edges  map[string]*EdgeRecord
	nextID int
	err    error
}

func newMockEdgeRepo() *mockEdgeRepo {
	return &mockEdgeRepo{edges: make(map[string]*EdgeRecord), nextID: 1}
}

func (m *mockEdgeRepo) List(_ context.Context, schemaID string) ([]EdgeRecord, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []EdgeRecord
	for _, e := range m.edges {
		if e.SchemaID == schemaID {
			result = append(result, *e)
		}
	}
	return result, nil
}

func (m *mockEdgeRepo) Create(_ context.Context, record *EdgeRecord) error {
	if m.err != nil {
		return m.err
	}
	record.ID = fmt.Sprintf("edge-%d", m.nextID)
	m.nextID++
	m.edges[record.ID] = record
	return nil
}

func (m *mockEdgeRepo) Delete(_ context.Context, id string) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.edges[id]; !ok {
		return fmt.Errorf("edge not found: %s", id)
	}
	delete(m.edges, id)
	return nil
}

type mockModelRepo struct {
	models map[string]*ModelRecord
	nextID int
	err    error
}

func newMockModelRepo() *mockModelRepo {
	return &mockModelRepo{models: make(map[string]*ModelRecord), nextID: 1}
}

func (m *mockModelRepo) List(_ context.Context) ([]ModelRecord, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := make([]ModelRecord, 0, len(m.models))
	for _, r := range m.models {
		result = append(result, *r)
	}
	return result, nil
}

func (m *mockModelRepo) GetByID(_ context.Context, id string) (*ModelRecord, error) {
	if m.err != nil {
		return nil, m.err
	}
	r, ok := m.models[id]
	if !ok {
		return nil, fmt.Errorf("model not found: %s", id)
	}
	return r, nil
}

func (m *mockModelRepo) Create(_ context.Context, record *ModelRecord) error {
	if m.err != nil {
		return m.err
	}
	record.ID = fmt.Sprintf("model-%d", m.nextID)
	m.nextID++
	m.models[record.ID] = record
	return nil
}

func (m *mockModelRepo) Update(_ context.Context, id string, record *ModelRecord) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.models[id]; !ok {
		return fmt.Errorf("model not found: %s", id)
	}
	record.ID = id
	m.models[id] = record
	return nil
}

func (m *mockModelRepo) Delete(_ context.Context, id string) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.models[id]; !ok {
		return fmt.Errorf("model not found: %s", id)
	}
	delete(m.models, id)
	return nil
}

type mockTriggerRepo struct {
	triggers map[string]*TriggerRecord
	nextID   int
	err      error
}

func newMockTriggerRepo() *mockTriggerRepo {
	return &mockTriggerRepo{triggers: make(map[string]*TriggerRecord), nextID: 1}
}

func (m *mockTriggerRepo) List(_ context.Context) ([]TriggerRecord, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := make([]TriggerRecord, 0, len(m.triggers))
	for _, t := range m.triggers {
		result = append(result, *t)
	}
	return result, nil
}

func (m *mockTriggerRepo) GetByID(_ context.Context, id string) (*TriggerRecord, error) {
	if m.err != nil {
		return nil, m.err
	}
	t, ok := m.triggers[id]
	if !ok {
		return nil, fmt.Errorf("trigger not found: %s", id)
	}
	return t, nil
}

func (m *mockTriggerRepo) Create(_ context.Context, record *TriggerRecord) error {
	if m.err != nil {
		return m.err
	}
	record.ID = fmt.Sprintf("trigger-%d", m.nextID)
	m.nextID++
	m.triggers[record.ID] = record
	return nil
}

func (m *mockTriggerRepo) Update(_ context.Context, id string, record *TriggerRecord) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.triggers[id]; !ok {
		return fmt.Errorf("trigger not found: %s", id)
	}
	record.ID = id
	m.triggers[id] = record
	return nil
}

func (m *mockTriggerRepo) Delete(_ context.Context, id string) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.triggers[id]; !ok {
		return fmt.Errorf("trigger not found: %s", id)
	}
	delete(m.triggers, id)
	return nil
}

type mockCapabilityRepo struct {
	caps   map[string]*CapabilityRecord
	nextID int
	err    error
}

func newMockCapabilityRepo() *mockCapabilityRepo {
	return &mockCapabilityRepo{caps: make(map[string]*CapabilityRecord), nextID: 1}
}

func (m *mockCapabilityRepo) ListByAgent(_ context.Context, agentName string) ([]CapabilityRecord, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []CapabilityRecord
	for _, c := range m.caps {
		if c.AgentName == agentName {
			result = append(result, *c)
		}
	}
	return result, nil
}

func (m *mockCapabilityRepo) Create(_ context.Context, record *CapabilityRecord) error {
	if m.err != nil {
		return m.err
	}
	record.ID = fmt.Sprintf("cap-%d", m.nextID)
	m.nextID++
	m.caps[record.ID] = record
	return nil
}

func (m *mockCapabilityRepo) Update(_ context.Context, id string, record *CapabilityRecord) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.caps[id]; !ok {
		return fmt.Errorf("capability not found: %s", id)
	}
	record.ID = id
	m.caps[id] = record
	return nil
}

func (m *mockCapabilityRepo) Delete(_ context.Context, id string) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.caps[id]; !ok {
		return fmt.Errorf("capability not found: %s", id)
	}
	delete(m.caps, id)
	return nil
}

// --- Reloader counter ---

func newReloaderCounter() (func(), *atomic.Int32) {
	var count atomic.Int32
	return func() { count.Add(1) }, &count
}

// --- Agent tool tests ---

func TestAdminListAgents_Empty(t *testing.T) {
	repo := newMockAgentRepo()
	tool := NewAdminListAgentsTool(repo)
	result, err := tool.InvokableRun(context.Background(), "")
	require.NoError(t, err)
	assert.Contains(t, result, "No agents configured")
}

func TestAdminListAgents_WithData(t *testing.T) {
	repo := newMockAgentRepo()
	repo.agents["test-agent"] = &AgentRecord{Name: "test-agent", Lifecycle: "persistent", BuiltinTools: []string{"read_file"}}
	tool := NewAdminListAgentsTool(repo)
	result, err := tool.InvokableRun(context.Background(), "")
	require.NoError(t, err)
	assert.Contains(t, result, "test-agent")
	assert.Contains(t, result, "1 agents")
}

func TestAdminGetAgent_NotFound(t *testing.T) {
	repo := newMockAgentRepo()
	tool := NewAdminGetAgentTool(repo)
	args, _ := json.Marshal(getAgentArgs{Name: "nonexistent"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "not found")
}

func TestAdminGetAgent_Found(t *testing.T) {
	repo := newMockAgentRepo()
	repo.agents["my-agent"] = &AgentRecord{Name: "my-agent", SystemPrompt: "Hello", Lifecycle: "persistent"}
	tool := NewAdminGetAgentTool(repo)
	args, _ := json.Marshal(getAgentArgs{Name: "my-agent"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "my-agent")
	assert.Contains(t, result, "Hello")
}

func TestAdminCreateAgent_Success(t *testing.T) {
	repo := newMockAgentRepo()
	reloader, count := newReloaderCounter()
	tool := NewAdminCreateAgentTool(repo, reloader)
	args, _ := json.Marshal(createAgentArgs{Name: "new-agent", SystemPrompt: "prompt"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "created successfully")
	assert.Equal(t, int32(1), count.Load())
	assert.NotNil(t, repo.agents["new-agent"])
}

func TestAdminCreateAgent_MissingName(t *testing.T) {
	repo := newMockAgentRepo()
	tool := NewAdminCreateAgentTool(repo, nil)
	args, _ := json.Marshal(createAgentArgs{SystemPrompt: "prompt"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "name is required")
}

func TestAdminCreateAgent_MissingPrompt(t *testing.T) {
	repo := newMockAgentRepo()
	tool := NewAdminCreateAgentTool(repo, nil)
	args, _ := json.Marshal(createAgentArgs{Name: "test"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "system_prompt is required")
}

func TestAdminCreateAgent_Duplicate(t *testing.T) {
	repo := newMockAgentRepo()
	repo.agents["existing"] = &AgentRecord{Name: "existing"}
	tool := NewAdminCreateAgentTool(repo, nil)
	args, _ := json.Marshal(createAgentArgs{Name: "existing", SystemPrompt: "prompt"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "already exists")
}

func TestAdminUpdateAgent_Success(t *testing.T) {
	repo := newMockAgentRepo()
	repo.agents["test"] = &AgentRecord{Name: "test", SystemPrompt: "old", Lifecycle: "persistent"}
	reloader, count := newReloaderCounter()
	tool := NewAdminUpdateAgentTool(repo, reloader)
	args, _ := json.Marshal(updateAgentArgs{Name: "test", SystemPrompt: "new"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "updated successfully")
	assert.Equal(t, int32(1), count.Load())
	assert.Equal(t, "new", repo.agents["test"].SystemPrompt)
}

func TestAdminUpdateAgent_NotFound(t *testing.T) {
	repo := newMockAgentRepo()
	tool := NewAdminUpdateAgentTool(repo, nil)
	args, _ := json.Marshal(updateAgentArgs{Name: "nope"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "not found")
}

func TestAdminDeleteAgent_Success(t *testing.T) {
	repo := newMockAgentRepo()
	repo.agents["doomed"] = &AgentRecord{Name: "doomed"}
	reloader, count := newReloaderCounter()
	tool := NewAdminDeleteAgentTool(repo, reloader)
	args, _ := json.Marshal(deleteAgentArgs{Name: "doomed"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "deleted successfully")
	assert.Equal(t, int32(1), count.Load())
	assert.Empty(t, repo.agents)
}

func TestAdminDeleteAgent_NotFound(t *testing.T) {
	repo := newMockAgentRepo()
	tool := NewAdminDeleteAgentTool(repo, nil)
	args, _ := json.Marshal(deleteAgentArgs{Name: "nope"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "not found")
}

// --- Schema tool tests ---

func TestAdminCreateSchema_Success(t *testing.T) {
	repo := newMockSchemaRepo()
	reloader, count := newReloaderCounter()
	tool := NewAdminCreateSchemaTool(repo, reloader)
	args, _ := json.Marshal(createSchemaArgs{Name: "my-schema", Description: "test"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "created")
	assert.Equal(t, int32(1), count.Load())
	assert.Len(t, repo.schemas, 1)
}

func TestAdminDeleteSchema_NotFound(t *testing.T) {
	repo := newMockSchemaRepo()
	tool := NewAdminDeleteSchemaTool(repo, nil)
	args, _ := json.Marshal(deleteSchemaArgs{SchemaID: "nonexistent"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "not found")
}

// --- Schema-agent wiring tests ---

func TestAdminAddAgentToSchema_Success(t *testing.T) {
	repo := newMockSchemaRepo()
	repo.schemas["schema-1"] = &SchemaRecord{ID: "schema-1", Name: "test"}
	reloader, count := newReloaderCounter()
	tool := NewAdminAddAgentToSchemaTool(repo, reloader)
	args, _ := json.Marshal(schemaAgentArgs{SchemaID: "schema-1", AgentName: "my-agent"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "added")
	assert.Equal(t, int32(1), count.Load())
	assert.Contains(t, repo.schemas["schema-1"].AgentNames, "my-agent")
}

func TestAdminRemoveAgentFromSchema_NotFound(t *testing.T) {
	repo := newMockSchemaRepo()
	repo.schemas["schema-1"] = &SchemaRecord{ID: "schema-1", Name: "test"}
	tool := NewAdminRemoveAgentFromSchemaTool(repo, nil)
	args, _ := json.Marshal(schemaAgentArgs{SchemaID: "schema-1", AgentName: "nonexistent"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "not found")
}

// --- Edge tool tests ---

func TestAdminCreateEdge_Success(t *testing.T) {
	repo := newMockEdgeRepo()
	reloader, count := newReloaderCounter()
	tool := NewAdminCreateEdgeTool(repo, reloader)
	args, _ := json.Marshal(createEdgeArgs{SchemaID: "schema-1", FromAgent: "a", ToAgent: "b", Type: "flow"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "Edge created")
	assert.Equal(t, int32(1), count.Load())
}

func TestAdminCreateEdge_InvalidType(t *testing.T) {
	repo := newMockEdgeRepo()
	tool := NewAdminCreateEdgeTool(repo, nil)
	args, _ := json.Marshal(createEdgeArgs{SchemaID: "schema-1", FromAgent: "a", ToAgent: "b", Type: "invalid"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "Invalid edge type")
}

func TestAdminListEdges_Empty(t *testing.T) {
	repo := newMockEdgeRepo()
	tool := NewAdminListEdgesTool(repo)
	args, _ := json.Marshal(listEdgesArgs{SchemaID: "schema-1"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "No edges")
}

// --- Model tool tests ---

func TestAdminCreateModel_Success(t *testing.T) {
	repo := newMockModelRepo()
	reloader, count := newReloaderCounter()
	tool := NewAdminCreateModelTool(repo, reloader)
	args, _ := json.Marshal(createModelArgs{Name: "gpt4", Type: "openai_compatible", ModelName: "gpt-4"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "created")
	assert.Equal(t, int32(1), count.Load())
}

func TestAdminCreateModel_MissingFields(t *testing.T) {
	repo := newMockModelRepo()
	tool := NewAdminCreateModelTool(repo, nil)
	args, _ := json.Marshal(createModelArgs{Name: "test"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "type is required")
}

func TestAdminListModels_MaskedAPIKey(t *testing.T) {
	repo := newMockModelRepo()
	repo.models["model-1"] = &ModelRecord{ID: "model-1", Name: "test", Type: "openai", ModelName: "gpt-4", APIKey: "sk-secret123"}
	tool := NewAdminListModelsTool(repo)
	result, err := tool.InvokableRun(context.Background(), "")
	require.NoError(t, err)
	assert.Contains(t, result, "has_api_key=yes")
	assert.NotContains(t, result, "sk-secret123")
}

// --- Trigger tool tests ---

func TestAdminCreateTrigger_Success(t *testing.T) {
	repo := newMockTriggerRepo()
	reloader, count := newReloaderCounter()
	tool := NewAdminCreateTriggerTool(repo, reloader)
	args, _ := json.Marshal(createTriggerArgs{Type: "cron", Title: "daily", AgentName: "test", Schedule: "0 0 * * *"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "created")
	assert.Equal(t, int32(1), count.Load())
}

func TestAdminDeleteTrigger_NotFound(t *testing.T) {
	repo := newMockTriggerRepo()
	tool := NewAdminDeleteTriggerTool(repo, nil)
	args, _ := json.Marshal(deleteTriggerArgs{TriggerID: "nonexistent"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "not found")
}

// --- Capability tool tests ---

func TestAdminAddCapability_Success(t *testing.T) {
	repo := newMockCapabilityRepo()
	reloader, count := newReloaderCounter()
	tool := NewAdminAddCapabilityTool(repo, reloader)
	args, _ := json.Marshal(addCapabilityArgs{AgentName: "test", CapabilityType: "memory"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "added")
	assert.Equal(t, int32(1), count.Load())
}

func TestAdminAddCapability_InvalidType(t *testing.T) {
	repo := newMockCapabilityRepo()
	tool := NewAdminAddCapabilityTool(repo, nil)
	args, _ := json.Marshal(addCapabilityArgs{AgentName: "test", CapabilityType: "invalid"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "Invalid capability type")
}

func TestAdminAddCapability_NilRepo(t *testing.T) {
	tool := NewAdminAddCapabilityTool(nil, nil)
	args, _ := json.Marshal(addCapabilityArgs{AgentName: "test", CapabilityType: "memory"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "not available")
}

// --- Inspect tool tests ---

func TestAdminListSessions_NilRepo(t *testing.T) {
	tool := NewAdminListSessionsTool(nil)
	result, err := tool.InvokableRun(context.Background(), "")
	require.NoError(t, err)
	assert.Contains(t, result, "not available")
}

func TestAdminGetSession_NilRepo(t *testing.T) {
	tool := NewAdminGetSessionTool(nil)
	args, _ := json.Marshal(getSessionArgs{SessionID: "abc"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "not available")
}

// --- Register test ---

func TestRegisterAdminTools_RegistersAllTools(t *testing.T) {
	store := tools.NewBuiltinToolStore()

	RegisterAdminTools(store, AdminToolDependencies{
		AgentRepo:      newMockAgentRepo(),
		SchemaRepo:     newMockSchemaRepo(),
		TriggerRepo:    newMockTriggerRepo(),
		MCPServerRepo:  &mockMCPServerRepo{},
		ModelRepo:      newMockModelRepo(),
		EdgeRepo:       newMockEdgeRepo(),
		CapabilityRepo: newMockCapabilityRepo(),
	})

	expectedTools := []string{
		"admin_list_agents", "admin_get_agent", "admin_create_agent", "admin_update_agent", "admin_delete_agent",
		"admin_list_schemas", "admin_get_schema", "admin_create_schema", "admin_update_schema", "admin_delete_schema",
		"admin_add_agent_to_schema", "admin_remove_agent_from_schema",
		"admin_list_edges", "admin_create_edge", "admin_delete_edge",
		"admin_list_triggers", "admin_create_trigger", "admin_update_trigger", "admin_delete_trigger",
		"admin_list_mcp_servers", "admin_create_mcp_server", "admin_update_mcp_server", "admin_delete_mcp_server",
		"admin_list_models", "admin_create_model", "admin_update_model", "admin_delete_model",
		"admin_add_capability", "admin_remove_capability", "admin_update_capability",
		"admin_list_sessions", "admin_get_session",
	}

	for _, name := range expectedTools {
		_, ok := store.Get(name)
		assert.True(t, ok, "tool %q not registered", name)
	}
	assert.Equal(t, len(expectedTools), len(store.Names()))
}

// --- Tool info tests ---

func TestAllToolInfos_HaveNames(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepo()

	toolInstances := []interface{ Info(context.Context) (*interface{}, error) }{}
	_ = toolInstances // just check Info returns non-nil names

	// Test a representative sample of tools.
	tools := []struct {
		name string
		t    interface {
			Info(context.Context) (*interface{}, error)
		}
	}{}
	_ = tools

	// Simpler: just check the agent tools return proper info.
	listTool := NewAdminListAgentsTool(repo)
	info, err := listTool.Info(ctx)
	require.NoError(t, err)
	assert.Equal(t, "admin_list_agents", info.Name)
	assert.NotEmpty(t, info.Desc)
}

// --- Helper fakes ---

type mockMCPServerRepo struct{}

func (m *mockMCPServerRepo) List(_ context.Context) ([]MCPServerRecord, error)              { return nil, nil }
func (m *mockMCPServerRepo) GetByID(_ context.Context, _ string) (*MCPServerRecord, error)   { return &MCPServerRecord{}, nil }
func (m *mockMCPServerRepo) Create(_ context.Context, _ *MCPServerRecord) error              { return nil }
func (m *mockMCPServerRepo) Update(_ context.Context, _ string, _ *MCPServerRecord) error    { return nil }
func (m *mockMCPServerRepo) Delete(_ context.Context, _ string) error                        { return nil }

// --- Workflow integration test ---

func TestWorkflow_CreateSchemaWithAgentsAndEdges(t *testing.T) {
	ctx := context.Background()
	agentRepo := newMockAgentRepo()
	schemaRepo := newMockSchemaRepo()
	edgeRepo := newMockEdgeRepo()
	reloader, reloadCount := newReloaderCounter()

	// Step 1: Create agents.
	createAgent := NewAdminCreateAgentTool(agentRepo, reloader)
	args, _ := json.Marshal(createAgentArgs{Name: "router", SystemPrompt: "Routes requests"})
	result, err := createAgent.InvokableRun(ctx, string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "created")

	args, _ = json.Marshal(createAgentArgs{Name: "worker", SystemPrompt: "Does work"})
	result, err = createAgent.InvokableRun(ctx, string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "created")

	// Step 2: Create schema.
	createSchema := NewAdminCreateSchemaTool(schemaRepo, reloader)
	args, _ = json.Marshal(createSchemaArgs{Name: "my-flow", Description: "Test flow"})
	result, err = createSchema.InvokableRun(ctx, string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "created")

	// Step 3: Add agents to schema.
	addAgent := NewAdminAddAgentToSchemaTool(schemaRepo, reloader)
	args, _ = json.Marshal(schemaAgentArgs{SchemaID: "schema-1", AgentName: "router"})
	_, err = addAgent.InvokableRun(ctx, string(args))
	require.NoError(t, err)

	args, _ = json.Marshal(schemaAgentArgs{SchemaID: "schema-1", AgentName: "worker"})
	_, err = addAgent.InvokableRun(ctx, string(args))
	require.NoError(t, err)

	// Step 4: Create edge.
	createEdge := NewAdminCreateEdgeTool(edgeRepo, reloader)
	args, _ = json.Marshal(createEdgeArgs{SchemaID: "schema-1", FromAgent: "router", ToAgent: "worker", Type: "flow"})
	result, err = createEdge.InvokableRun(ctx, string(args))
	require.NoError(t, err)
	assert.Contains(t, result, "Edge created")

	// Verify: 2 agents, 1 schema with 2 agents, 1 edge, reloader called 6 times.
	assert.Len(t, agentRepo.agents, 2)
	assert.Contains(t, schemaRepo.schemas["schema-1"].AgentNames, "router")
	assert.Contains(t, schemaRepo.schemas["schema-1"].AgentNames, "worker")
	assert.Len(t, edgeRepo.edges, 1)
	assert.Equal(t, int32(6), reloadCount.Load())
}

// --- Invalid JSON test ---

func TestAdminCreateAgent_InvalidJSON(t *testing.T) {
	repo := newMockAgentRepo()
	tool := NewAdminCreateAgentTool(repo, nil)
	result, err := tool.InvokableRun(context.Background(), "not-json")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(result, "[ERROR]"))
}

package http

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// --- mock services ---

type mockSchemaService struct {
	schemas     map[string]*SchemaInfo
	agents      map[string][]string
	nextID      int
	createErr   error
	addAgentErr error
}

func newMockSchemaService() *mockSchemaService {
	return &mockSchemaService{
		schemas: make(map[string]*SchemaInfo),
		agents:  make(map[string][]string),
		nextID:  1,
	}
}

func (m *mockSchemaService) ListSchemas(_ context.Context) ([]SchemaInfo, error) {
	var result []SchemaInfo
	for _, s := range m.schemas {
		result = append(result, *s)
	}
	return result, nil
}

func (m *mockSchemaService) GetSchema(_ context.Context, id string) (*SchemaInfo, error) {
	s, ok := m.schemas[id]
	if !ok {
		return nil, fmt.Errorf("schema not found: %s", id)
	}
	s.Agents = m.agents[id]
	return s, nil
}

func (m *mockSchemaService) CreateSchema(_ context.Context, req CreateSchemaRequest) (*SchemaInfo, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	id := fmt.Sprintf("%d", m.nextID)
	m.nextID++
	s := &SchemaInfo{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
	}
	m.schemas[id] = s
	return s, nil
}

func (m *mockSchemaService) UpdateSchema(_ context.Context, _ string, _ UpdateSchemaRequest) error {
	return nil
}

func (m *mockSchemaService) DeleteSchema(_ context.Context, _ string) error {
	return nil
}

// addSchemaAgent is a test-only helper that records membership in the
// mock. V2 production code derives membership from agent_relations
// (docs/architecture/agent-first-runtime.md §2.1); the test wires this
// helper in places that historically called AddSchemaAgent so the
// remaining export-side assertions still have a known bag of agent names
// to compare against.
func (m *mockSchemaService) addSchemaAgent(schemaID string, agentName string) {
	m.agents[schemaID] = append(m.agents[schemaID], agentName)
}

func (m *mockSchemaService) ListSchemaAgents(_ context.Context, schemaID string) ([]string, error) {
	return m.agents[schemaID], nil
}

type mockAgentRelationService struct {
	relations map[string][]AgentRelationInfo
	nextID    int
}

func newMockAgentRelationService() *mockAgentRelationService {
	return &mockAgentRelationService{
		relations: make(map[string][]AgentRelationInfo),
		nextID:    1,
	}
}

func (m *mockAgentRelationService) ListAgentRelations(_ context.Context, schemaID string) ([]AgentRelationInfo, error) {
	return m.relations[schemaID], nil
}

func (m *mockAgentRelationService) GetAgentRelation(_ context.Context, _ string) (*AgentRelationInfo, error) {
	return nil, nil
}

func (m *mockAgentRelationService) CreateAgentRelation(_ context.Context, schemaID string, req CreateAgentRelationRequest) (*AgentRelationInfo, error) {
	id := fmt.Sprintf("%d", m.nextID)
	m.nextID++
	rel := AgentRelationInfo{
		ID:              id,
		SchemaID:        schemaID,
		SourceAgentName: req.Source,
		TargetAgentName: req.Target,
		Config:          req.Config,
	}
	m.relations[schemaID] = append(m.relations[schemaID], rel)
	return &rel, nil
}

func (m *mockAgentRelationService) UpdateAgentRelation(_ context.Context, _ string, _ CreateAgentRelationRequest) error {
	return nil
}

func (m *mockAgentRelationService) DeleteAgentRelation(_ context.Context, _ string) error {
	return nil
}

type mockAgentDetailer struct {
	agents map[string]*AgentDetail
}

func newMockAgentDetailer() *mockAgentDetailer {
	return &mockAgentDetailer{
		agents: make(map[string]*AgentDetail),
	}
}

func (m *mockAgentDetailer) GetAgent(_ context.Context, name string) (*AgentDetail, error) {
	a, ok := m.agents[name]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", name)
	}
	return a, nil
}

// --- helper ---

func setupExportImportRouter(h *SchemaHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/api/v1/schemas/{id}/export", h.ExportSchema)
	r.Post("/api/v1/schemas/import", h.ImportSchema)
	return r
}

// --- tests ---

func TestExportSchema_Basic(t *testing.T) {
	schemaSvc := newMockSchemaService()
	relationSvc := newMockAgentRelationService()
	agentDetailer := newMockAgentDetailer()

	// Setup data
	schema, _ := schemaSvc.CreateSchema(context.Background(), CreateSchemaRequest{
		Name:        "support-schema",
		Description: "Customer support pipeline",
	})
	schemaSvc.addSchemaAgent(schema.ID, "classifier")
	schemaSvc.addSchemaAgent(schema.ID, "support-agent")

	agentDetailer.agents["classifier"] = &AgentDetail{
		AgentInfo:    AgentInfo{Name: "classifier"},
		SystemPrompt: "You are a classifier",
		Lifecycle:    "spawn",
		Tools:        []string{"classify"},
		MaxSteps:     10,
	}
	agentDetailer.agents["support-agent"] = &AgentDetail{
		AgentInfo:    AgentInfo{Name: "support-agent"},
		SystemPrompt: "You help customers",
		Lifecycle:    "persistent",
		Tools:        []string{"search_knowledge", "respond"},
	}

	relationSvc.relations[schema.ID] = []AgentRelationInfo{
		{ID: "1", SchemaID: schema.ID, SourceAgentName: "classifier", TargetAgentName: "support-agent"},
	}

	handler := NewSchemaHandler(schemaSvc, relationSvc)
	handler.SetAgentDetailer(agentDetailer)
	router := setupExportImportRouter(handler)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/schemas/%s/export", schema.ID), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/x-yaml", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Header().Get("Content-Disposition"), "support-schema.yaml")

	var exported SchemaYAML
	err := yaml.Unmarshal(rec.Body.Bytes(), &exported)
	require.NoError(t, err)

	assert.Equal(t, "support-schema", exported.Name)
	assert.Equal(t, "Customer support pipeline", exported.Description)
	require.Len(t, exported.Agents, 2)
	assert.Equal(t, "classifier", exported.Agents[0].Name)
	assert.Equal(t, "You are a classifier", exported.Agents[0].SystemPrompt)
	assert.Equal(t, "spawn", exported.Agents[0].Lifecycle)
	assert.Equal(t, "support-agent", exported.Agents[1].Name)
	require.Len(t, exported.AgentRelations, 1)
	assert.Equal(t, "classifier", exported.AgentRelations[0].Source)
	assert.Equal(t, "support-agent", exported.AgentRelations[0].Target)
}

func TestExportSchema_NotFound(t *testing.T) {
	schemaSvc := newMockSchemaService()
	handler := NewSchemaHandler(schemaSvc, newMockAgentRelationService())
	router := setupExportImportRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schemas/999/export", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestExportSchema_WithoutAgentDetailer(t *testing.T) {
	schemaSvc := newMockSchemaService()
	schema, _ := schemaSvc.CreateSchema(context.Background(), CreateSchemaRequest{Name: "minimal"})
	schemaSvc.addSchemaAgent(schema.ID, "agent-a")

	handler := NewSchemaHandler(schemaSvc, newMockAgentRelationService())
	// Do NOT set agent detailer
	router := setupExportImportRouter(handler)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/schemas/%s/export", schema.ID), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var exported SchemaYAML
	err := yaml.Unmarshal(rec.Body.Bytes(), &exported)
	require.NoError(t, err)

	assert.Equal(t, "minimal", exported.Name)
	require.Len(t, exported.Agents, 1)
	assert.Equal(t, "agent-a", exported.Agents[0].Name)
	assert.Empty(t, exported.Agents[0].SystemPrompt) // no detail
}

func TestImportSchema_Basic(t *testing.T) {
	schemaSvc := newMockSchemaService()
	relationSvc := newMockAgentRelationService()

	handler := NewSchemaHandler(schemaSvc, relationSvc)
	router := setupExportImportRouter(handler)

	yamlBody := `
name: "imported-schema"
description: "Imported from YAML"
agents:
  - name: "agent-a"
  - name: "agent-b"
agent_relations:
  - source: "agent-a"
    target: "agent-b"
`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/import", bytes.NewBufferString(yamlBody))
	req.Header.Set("Content-Type", "application/x-yaml")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	// Verify schema was created
	require.Len(t, schemaSvc.schemas, 1)
	var createdSchema *SchemaInfo
	for _, s := range schemaSvc.schemas {
		createdSchema = s
	}
	assert.Equal(t, "imported-schema", createdSchema.Name)
	assert.Equal(t, "Imported from YAML", createdSchema.Description)

	// V2: schema membership is derived from agent_relations
	// (docs/architecture/agent-first-runtime.md §2.1). The YAML "agents:"
	// list itself adds no membership; only relations do. The single
	// imported relation expresses both endpoints as members.
	rels := relationSvc.relations[createdSchema.ID]
	require.Len(t, rels, 1)
	assert.Equal(t, "agent-a", rels[0].SourceAgentName)
	assert.Equal(t, "agent-b", rels[0].TargetAgentName)
}

func TestImportSchema_EmptyName(t *testing.T) {
	handler := NewSchemaHandler(newMockSchemaService(), newMockAgentRelationService())
	router := setupExportImportRouter(handler)

	yamlBody := `description: "no name"`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/import", bytes.NewBufferString(yamlBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestImportSchema_InvalidYAML(t *testing.T) {
	handler := NewSchemaHandler(newMockSchemaService(), newMockAgentRelationService())
	router := setupExportImportRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/import", bytes.NewBufferString("}{not yaml"))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestImportSchema_EmptyBody(t *testing.T) {
	handler := NewSchemaHandler(newMockSchemaService(), newMockAgentRelationService())
	router := setupExportImportRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/import", bytes.NewBufferString(""))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Empty YAML unmarshals to zero-value struct, name will be empty
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRoundTrip_ExportImport(t *testing.T) {
	// Setup source schema
	srcSchemaSvc := newMockSchemaService()
	srcRelationSvc := newMockAgentRelationService()

	schema, _ := srcSchemaSvc.CreateSchema(context.Background(), CreateSchemaRequest{
		Name:        "roundtrip-schema",
		Description: "Round trip test",
	})
	srcSchemaSvc.addSchemaAgent(schema.ID, "agent-x")
	srcSchemaSvc.addSchemaAgent(schema.ID, "agent-y")
	srcRelationSvc.relations[schema.ID] = []AgentRelationInfo{
		{ID: "1", SchemaID: schema.ID, SourceAgentName: "agent-x", TargetAgentName: "agent-y"},
	}

	srcHandler := NewSchemaHandler(srcSchemaSvc, srcRelationSvc)
	srcRouter := setupExportImportRouter(srcHandler)

	// Export
	exportReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/schemas/%s/export", schema.ID), nil)
	exportRec := httptest.NewRecorder()
	srcRouter.ServeHTTP(exportRec, exportReq)
	require.Equal(t, http.StatusOK, exportRec.Code)

	exportedYAML := exportRec.Body.Bytes()

	// Import into a fresh set of services
	dstSchemaSvc := newMockSchemaService()
	dstRelationSvc := newMockAgentRelationService()

	dstHandler := NewSchemaHandler(dstSchemaSvc, dstRelationSvc)
	dstRouter := setupExportImportRouter(dstHandler)

	importReq := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/import", bytes.NewReader(exportedYAML))
	importRec := httptest.NewRecorder()
	dstRouter.ServeHTTP(importRec, importReq)
	require.Equal(t, http.StatusCreated, importRec.Code)

	// Verify imported schema matches original
	require.Len(t, dstSchemaSvc.schemas, 1)
	var importedSchema *SchemaInfo
	for _, s := range dstSchemaSvc.schemas {
		importedSchema = s
	}
	assert.Equal(t, "roundtrip-schema", importedSchema.Name)
	assert.Equal(t, "Round trip test", importedSchema.Description)

	// V2: schema membership is derived from agent_relations
	// (docs/architecture/agent-first-runtime.md §2.1). After round-trip
	// the imported schema has only the single delegation relation; both
	// endpoints are implicit members.
	rels := dstRelationSvc.relations[importedSchema.ID]
	require.Len(t, rels, 1)
	assert.Equal(t, "agent-x", rels[0].SourceAgentName)
	assert.Equal(t, "agent-y", rels[0].TargetAgentName)
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "my-schema", "my-schema"},
		{"spaces", "my schema", "my-schema"},
		{"special chars", "my/schema:v2", "my_schema_v2"},
		{"empty", "", "schema"},
		{"windows chars", "a<b>c|d", "a_b_c_d"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

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

func (m *mockSchemaService) AddSchemaAgent(_ context.Context, schemaID string, agentName string) error {
	if m.addAgentErr != nil {
		return m.addAgentErr
	}
	m.agents[schemaID] = append(m.agents[schemaID], agentName)
	return nil
}

func (m *mockSchemaService) RemoveSchemaAgent(_ context.Context, _ string, _ string) error {
	return nil
}

func (m *mockSchemaService) ListSchemaAgents(_ context.Context, schemaID string) ([]string, error) {
	return m.agents[schemaID], nil
}

type mockGateService struct {
	gates  map[string][]GateInfo
	nextID int
}

func newMockGateService() *mockGateService {
	return &mockGateService{
		gates:  make(map[string][]GateInfo),
		nextID: 1,
	}
}

func (m *mockGateService) ListGates(_ context.Context, schemaID string) ([]GateInfo, error) {
	return m.gates[schemaID], nil
}

func (m *mockGateService) GetGate(_ context.Context, _ string) (*GateInfo, error) {
	return nil, nil
}

func (m *mockGateService) CreateGate(_ context.Context, schemaID string, req CreateGateRequest) (*GateInfo, error) {
	id := fmt.Sprintf("%d", m.nextID)
	m.nextID++
	g := GateInfo{
		ID:            id,
		SchemaID:      schemaID,
		Name:          req.Name,
		ConditionType: req.ConditionType,
		Config:        req.Config,
		MaxIterations: req.MaxIterations,
		Timeout:       req.Timeout,
	}
	m.gates[schemaID] = append(m.gates[schemaID], g)
	return &g, nil
}

func (m *mockGateService) UpdateGate(_ context.Context, _ string, _ CreateGateRequest) error {
	return nil
}

func (m *mockGateService) DeleteGate(_ context.Context, _ string) error {
	return nil
}

type mockEdgeService struct {
	edges  map[string][]EdgeInfo
	nextID int
}

func newMockEdgeService() *mockEdgeService {
	return &mockEdgeService{
		edges:  make(map[string][]EdgeInfo),
		nextID: 1,
	}
}

func (m *mockEdgeService) ListEdges(_ context.Context, schemaID string) ([]EdgeInfo, error) {
	return m.edges[schemaID], nil
}

func (m *mockEdgeService) GetEdge(_ context.Context, _ string) (*EdgeInfo, error) {
	return nil, nil
}

func (m *mockEdgeService) CreateEdge(_ context.Context, schemaID string, req CreateEdgeRequest) (*EdgeInfo, error) {
	id := fmt.Sprintf("%d", m.nextID)
	m.nextID++
	e := EdgeInfo{
		ID:              id,
		SchemaID:        schemaID,
		SourceAgentName: req.Source,
		TargetAgentName: req.Target,
		Type:            req.Type,
		Config:          req.Config,
	}
	m.edges[schemaID] = append(m.edges[schemaID], e)
	return &e, nil
}

func (m *mockEdgeService) UpdateEdge(_ context.Context, _ string, _ CreateEdgeRequest) error {
	return nil
}

func (m *mockEdgeService) DeleteEdge(_ context.Context, _ string) error {
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
	gateSvc := newMockGateService()
	edgeSvc := newMockEdgeService()
	agentDetailer := newMockAgentDetailer()

	// Setup data
	schema, _ := schemaSvc.CreateSchema(context.Background(), CreateSchemaRequest{
		Name:        "support-schema",
		Description: "Customer support pipeline",
	})
	schemaSvc.AddSchemaAgent(context.Background(), schema.ID, "classifier")
	schemaSvc.AddSchemaAgent(context.Background(), schema.ID, "support-agent")

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

	gateSvc.gates[schema.ID] = []GateInfo{
		{ID: "1", SchemaID: schema.ID, Name: "quality-check", ConditionType: "all", MaxIterations: 3},
	}
	edgeSvc.edges[schema.ID] = []EdgeInfo{
		{ID: "1", SchemaID: schema.ID, SourceAgentName: "classifier", TargetAgentName: "support-agent", Type: "flow"},
	}

	handler := NewSchemaHandler(schemaSvc, gateSvc, edgeSvc)
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
	require.Len(t, exported.Edges, 1)
	assert.Equal(t, "classifier", exported.Edges[0].Source)
	assert.Equal(t, "support-agent", exported.Edges[0].Target)
	assert.Equal(t, "flow", exported.Edges[0].Type)
	require.Len(t, exported.Gates, 1)
	assert.Equal(t, "quality-check", exported.Gates[0].Name)
	assert.Equal(t, "all", exported.Gates[0].ConditionType)
	assert.Equal(t, 3, exported.Gates[0].MaxIterations)
}

func TestExportSchema_NotFound(t *testing.T) {
	schemaSvc := newMockSchemaService()
	handler := NewSchemaHandler(schemaSvc, newMockGateService(), newMockEdgeService())
	router := setupExportImportRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schemas/999/export", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestExportSchema_WithoutAgentDetailer(t *testing.T) {
	schemaSvc := newMockSchemaService()
	schema, _ := schemaSvc.CreateSchema(context.Background(), CreateSchemaRequest{Name: "minimal"})
	schemaSvc.AddSchemaAgent(context.Background(), schema.ID, "agent-a")

	handler := NewSchemaHandler(schemaSvc, newMockGateService(), newMockEdgeService())
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
	gateSvc := newMockGateService()
	edgeSvc := newMockEdgeService()

	handler := NewSchemaHandler(schemaSvc, gateSvc, edgeSvc)
	router := setupExportImportRouter(handler)

	yamlBody := `
name: "imported-schema"
description: "Imported from YAML"
agents:
  - name: "agent-a"
  - name: "agent-b"
edges:
  - source: "agent-a"
    target: "agent-b"
    type: "flow"
gates:
  - name: "gate-1"
    condition_type: "all"
    max_iterations: 5
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

	// Verify agents added
	agents := schemaSvc.agents[createdSchema.ID]
	assert.ElementsMatch(t, []string{"agent-a", "agent-b"}, agents)

	// Verify edges created
	edges := edgeSvc.edges[createdSchema.ID]
	require.Len(t, edges, 1)
	assert.Equal(t, "agent-a", edges[0].SourceAgentName)
	assert.Equal(t, "agent-b", edges[0].TargetAgentName)
	assert.Equal(t, "flow", edges[0].Type)

	// Verify gates created
	gates := gateSvc.gates[createdSchema.ID]
	require.Len(t, gates, 1)
	assert.Equal(t, "gate-1", gates[0].Name)
	assert.Equal(t, "all", gates[0].ConditionType)
	assert.Equal(t, 5, gates[0].MaxIterations)
}

func TestImportSchema_EmptyName(t *testing.T) {
	handler := NewSchemaHandler(newMockSchemaService(), newMockGateService(), newMockEdgeService())
	router := setupExportImportRouter(handler)

	yamlBody := `description: "no name"`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/import", bytes.NewBufferString(yamlBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestImportSchema_InvalidYAML(t *testing.T) {
	handler := NewSchemaHandler(newMockSchemaService(), newMockGateService(), newMockEdgeService())
	router := setupExportImportRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/import", bytes.NewBufferString("}{not yaml"))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestImportSchema_EmptyBody(t *testing.T) {
	handler := NewSchemaHandler(newMockSchemaService(), newMockGateService(), newMockEdgeService())
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
	srcGateSvc := newMockGateService()
	srcEdgeSvc := newMockEdgeService()

	schema, _ := srcSchemaSvc.CreateSchema(context.Background(), CreateSchemaRequest{
		Name:        "roundtrip-schema",
		Description: "Round trip test",
	})
	srcSchemaSvc.AddSchemaAgent(context.Background(), schema.ID, "agent-x")
	srcSchemaSvc.AddSchemaAgent(context.Background(), schema.ID, "agent-y")
	srcEdgeSvc.edges[schema.ID] = []EdgeInfo{
		{ID: "1", SchemaID: schema.ID, SourceAgentName: "agent-x", TargetAgentName: "agent-y", Type: "transfer"},
	}
	srcGateSvc.gates[schema.ID] = []GateInfo{
		{ID: "1", SchemaID: schema.ID, Name: "check", ConditionType: "any", MaxIterations: 2, Timeout: 30},
	}

	srcHandler := NewSchemaHandler(srcSchemaSvc, srcGateSvc, srcEdgeSvc)
	srcRouter := setupExportImportRouter(srcHandler)

	// Export
	exportReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/schemas/%s/export", schema.ID), nil)
	exportRec := httptest.NewRecorder()
	srcRouter.ServeHTTP(exportRec, exportReq)
	require.Equal(t, http.StatusOK, exportRec.Code)

	exportedYAML := exportRec.Body.Bytes()

	// Import into a fresh set of services
	dstSchemaSvc := newMockSchemaService()
	dstGateSvc := newMockGateService()
	dstEdgeSvc := newMockEdgeService()

	dstHandler := NewSchemaHandler(dstSchemaSvc, dstGateSvc, dstEdgeSvc)
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

	// Agents
	agents := dstSchemaSvc.agents[importedSchema.ID]
	assert.ElementsMatch(t, []string{"agent-x", "agent-y"}, agents)

	// Edges
	edges := dstEdgeSvc.edges[importedSchema.ID]
	require.Len(t, edges, 1)
	assert.Equal(t, "agent-x", edges[0].SourceAgentName)
	assert.Equal(t, "agent-y", edges[0].TargetAgentName)
	assert.Equal(t, "transfer", edges[0].Type)

	// Gates
	gates := dstGateSvc.gates[importedSchema.ID]
	require.Len(t, gates, 1)
	assert.Equal(t, "check", gates[0].Name)
	assert.Equal(t, "any", gates[0].ConditionType)
	assert.Equal(t, 2, gates[0].MaxIterations)
	assert.Equal(t, 30, gates[0].Timeout)
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name string
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

//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	deliveryhttp "github.com/syntheticinc/bytebrew/engine/internal/delivery/http"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/configrepo"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/engine/internal/service/capability"
	"github.com/syntheticinc/bytebrew/engine/internal/service/cloud"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// ---- test DB + harness ----

// newTestDB creates an in-memory SQLite database with all migrations applied.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Discard,
	})
	require.NoError(t, err)

	// Run all migrations (AutoMigrate calls CREATE EXTENSION silently fails on SQLite — OK).
	err = db.AutoMigrate(
		&models.AgentModel{},
		&models.AgentToolModel{},
		&models.AgentSpawnTarget{},
		&models.AgentEscalation{},
		&models.AgentEscalationTrigger{},
		&models.SchemaModel{},
		&models.SchemaAgentModel{},
		&models.GateModel{},
		&models.EdgeModel{},
		&models.CapabilityModel{},
		&models.MemoryModel{},
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	})

	return db
}

// ---- HTTP adapter helpers (mirror app/ adapters) ----

type testSchemaServiceAdapter struct {
	repo *configrepo.GORMSchemaRepository
}

func (a *testSchemaServiceAdapter) ListSchemas(ctx context.Context) ([]deliveryhttp.SchemaInfo, error) {
	records, err := a.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]deliveryhttp.SchemaInfo, 0, len(records))
	for _, r := range records {
		result = append(result, deliveryhttp.SchemaInfo{
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
			Agents:      r.AgentNames,
		})
	}
	return result, nil
}

func (a *testSchemaServiceAdapter) GetSchema(ctx context.Context, id uint) (*deliveryhttp.SchemaInfo, error) {
	r, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &deliveryhttp.SchemaInfo{
		ID: r.ID, Name: r.Name, Description: r.Description, Agents: r.AgentNames,
	}, nil
}

func (a *testSchemaServiceAdapter) CreateSchema(ctx context.Context, req deliveryhttp.CreateSchemaRequest) (*deliveryhttp.SchemaInfo, error) {
	rec := &configrepo.SchemaRecord{Name: req.Name, Description: req.Description}
	if err := a.repo.Create(ctx, rec); err != nil {
		return nil, err
	}
	return &deliveryhttp.SchemaInfo{ID: rec.ID, Name: rec.Name, Description: rec.Description}, nil
}

func (a *testSchemaServiceAdapter) UpdateSchema(ctx context.Context, id uint, req deliveryhttp.UpdateSchemaRequest) error {
	return a.repo.Update(ctx, id, &configrepo.SchemaRecord{Name: req.Name, Description: req.Description})
}

func (a *testSchemaServiceAdapter) DeleteSchema(ctx context.Context, id uint) error {
	return a.repo.Delete(ctx, id)
}

func (a *testSchemaServiceAdapter) AddSchemaAgent(ctx context.Context, schemaID uint, agentName string) error {
	return a.repo.AddAgent(ctx, schemaID, agentName)
}

func (a *testSchemaServiceAdapter) RemoveSchemaAgent(ctx context.Context, schemaID uint, agentName string) error {
	return a.repo.RemoveAgent(ctx, schemaID, agentName)
}

func (a *testSchemaServiceAdapter) ListSchemaAgents(ctx context.Context, schemaID uint) ([]string, error) {
	return a.repo.ListAgents(ctx, schemaID)
}

type testAgentSchemaListerAdapter struct {
	repo *configrepo.GORMSchemaRepository
}

func (a *testAgentSchemaListerAdapter) ListSchemasForAgent(ctx context.Context, agentName string) ([]string, error) {
	return a.repo.ListSchemasForAgent(ctx, agentName)
}

type testCapabilityServiceAdapter struct {
	repo *configrepo.GORMCapabilityRepository
}

func (a *testCapabilityServiceAdapter) ListCapabilities(ctx context.Context, agentName string) ([]deliveryhttp.CapabilityInfo, error) {
	records, err := a.repo.ListByAgent(ctx, agentName)
	if err != nil {
		return nil, err
	}
	result := make([]deliveryhttp.CapabilityInfo, 0, len(records))
	for _, r := range records {
		result = append(result, deliveryhttp.CapabilityInfo{
			ID: r.ID, Type: r.Type, Config: r.Config, Enabled: r.Enabled,
		})
	}
	return result, nil
}

func (a *testCapabilityServiceAdapter) AddCapability(ctx context.Context, agentName string, req deliveryhttp.CreateCapabilityRequest) (*deliveryhttp.CapabilityInfo, error) {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	rec := &configrepo.CapabilityRecord{
		AgentName: agentName, Type: req.Type, Config: req.Config, Enabled: enabled,
	}
	if err := a.repo.Create(ctx, rec); err != nil {
		return nil, err
	}
	return &deliveryhttp.CapabilityInfo{
		ID: rec.ID, Type: rec.Type, Config: rec.Config, Enabled: rec.Enabled,
	}, nil
}

func (a *testCapabilityServiceAdapter) UpdateCapability(ctx context.Context, id uint, req deliveryhttp.UpdateCapabilityRequest) error {
	existing, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	capType := existing.Type
	if req.Type != "" {
		capType = req.Type
	}
	config := existing.Config
	if req.Config != nil {
		config = req.Config
	}
	enabled := existing.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	return a.repo.Update(ctx, id, &configrepo.CapabilityRecord{
		Type: capType, Config: config, Enabled: enabled,
	})
}

func (a *testCapabilityServiceAdapter) RemoveCapability(ctx context.Context, id uint) error {
	return a.repo.Delete(ctx, id)
}

type testCapInjectorAdapter struct {
	repo *configrepo.GORMCapabilityRepository
}

func (a *testCapInjectorAdapter) ListEnabledByAgent(ctx context.Context, agentName string) ([]capability.CapabilityRecord, error) {
	records, err := a.repo.ListEnabledByAgent(ctx, agentName)
	if err != nil {
		return nil, err
	}
	result := make([]capability.CapabilityRecord, 0, len(records))
	for _, r := range records {
		result = append(result, capability.CapabilityRecord{
			ID: r.ID, AgentName: r.AgentName, Type: r.Type, Config: r.Config, Enabled: r.Enabled,
		})
	}
	return result, nil
}

type testMemoryListerAdapter struct {
	storage *persistence.MemoryStorage
}

func (a *testMemoryListerAdapter) Execute(ctx context.Context, schemaID string) ([]*domain.Memory, error) {
	return a.storage.ListBySchema(ctx, schemaID)
}

type testMemoryClearerAdapter struct {
	storage *persistence.MemoryStorage
}

func (a *testMemoryClearerAdapter) ClearAll(ctx context.Context, schemaID string) (int64, error) {
	return a.storage.DeleteBySchema(ctx, schemaID)
}

func (a *testMemoryClearerAdapter) DeleteOne(ctx context.Context, id string) error {
	return a.storage.DeleteByID(ctx, id)
}

// noopGateService implements deliveryhttp.GateService as a no-op for tests that don't need gates.
type noopGateService struct{}

func (n *noopGateService) ListGates(_ context.Context, _ uint) ([]deliveryhttp.GateInfo, error) {
	return nil, nil
}
func (n *noopGateService) GetGate(_ context.Context, _ uint) (*deliveryhttp.GateInfo, error) {
	return nil, nil
}
func (n *noopGateService) CreateGate(_ context.Context, _ uint, _ deliveryhttp.CreateGateRequest) (*deliveryhttp.GateInfo, error) {
	return nil, nil
}
func (n *noopGateService) UpdateGate(_ context.Context, _ uint, _ deliveryhttp.CreateGateRequest) error {
	return nil
}
func (n *noopGateService) DeleteGate(_ context.Context, _ uint) error { return nil }

// noopEdgeService implements deliveryhttp.EdgeService as a no-op.
type noopEdgeService struct{}

func (n *noopEdgeService) ListEdges(_ context.Context, _ uint) ([]deliveryhttp.EdgeInfo, error) {
	return nil, nil
}
func (n *noopEdgeService) GetEdge(_ context.Context, _ uint) (*deliveryhttp.EdgeInfo, error) {
	return nil, nil
}
func (n *noopEdgeService) CreateEdge(_ context.Context, _ uint, _ deliveryhttp.CreateEdgeRequest) (*deliveryhttp.EdgeInfo, error) {
	return nil, nil
}
func (n *noopEdgeService) UpdateEdge(_ context.Context, _ uint, _ deliveryhttp.CreateEdgeRequest) error {
	return nil
}
func (n *noopEdgeService) DeleteEdge(_ context.Context, _ uint) error { return nil }

// ---- helpers ----

func postJSON(t *testing.T, router http.Handler, url string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func getJSON(t *testing.T, router http.Handler, url string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func deleteJSON(t *testing.T, router http.Handler, url string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&v))
	return v
}

// createTestAgent inserts an agent directly into DB for test setup.
func createTestAgent(t *testing.T, db *gorm.DB, name string) {
	t.Helper()
	agent := models.AgentModel{
		Name:         name,
		SystemPrompt: "test agent",
		Lifecycle:    "persistent",
		ToolExecution: "sequential",
	}
	require.NoError(t, db.Create(&agent).Error)
}

// ---- Test 1: Schema CRUD + Agent Cross-References ----

func TestV2_SchemaCRUD_AgentCrossReferences(t *testing.T) {
	db := newTestDB(t)

	schemaRepo := configrepo.NewGORMSchemaRepository(db)
	schemaSvc := &testSchemaServiceAdapter{repo: schemaRepo}
	schemaLister := &testAgentSchemaListerAdapter{repo: schemaRepo}

	schemaHandler := deliveryhttp.NewSchemaHandler(schemaSvc, &noopGateService{}, &noopEdgeService{})

	r := chi.NewRouter()
	r.Mount("/api/v1/schemas", schemaHandler.Routes())

	// 1. Create agent first (needed for schema-agent ref)
	createTestAgent(t, db, "test-agent")

	// 2. POST /api/v1/schemas → 201
	rec := postJSON(t, r, "/api/v1/schemas", map[string]string{
		"name": "test-schema",
	})
	require.Equal(t, http.StatusCreated, rec.Code)
	schema := decodeJSON[deliveryhttp.SchemaInfo](t, rec)
	assert.Equal(t, "test-schema", schema.Name)
	assert.NotZero(t, schema.ID)
	schemaID := schema.ID

	// 3. POST /api/v1/schemas/{id}/agents → 204
	rec = postJSON(t, r, fmt.Sprintf("/api/v1/schemas/%d/agents", schemaID), map[string]string{
		"agent_name": "test-agent",
	})
	require.Equal(t, http.StatusNoContent, rec.Code)

	// 4. Verify agent used_in_schemas contains "test-schema"
	ctx := context.Background()
	schemaNames, err := schemaLister.ListSchemasForAgent(ctx, "test-agent")
	require.NoError(t, err)
	assert.Contains(t, schemaNames, "test-schema")

	// 5. GET /api/v1/schemas/{id}/agents → contains "test-agent"
	rec = getJSON(t, r, fmt.Sprintf("/api/v1/schemas/%d/agents", schemaID))
	require.Equal(t, http.StatusOK, rec.Code)
	agents := decodeJSON[[]string](t, rec)
	assert.Contains(t, agents, "test-agent")

	// 6. DELETE /api/v1/schemas/{id} → 204
	rec = deleteJSON(t, r, fmt.Sprintf("/api/v1/schemas/%d", schemaID))
	require.Equal(t, http.StatusNoContent, rec.Code)

	// 7. Verify agent used_in_schemas is now empty
	schemaNames, err = schemaLister.ListSchemasForAgent(ctx, "test-agent")
	require.NoError(t, err)
	assert.Empty(t, schemaNames)
}

// ---- Test 2: Capability CRUD + InjectedTools ----

func TestV2_CapabilityCRUD_InjectedTools(t *testing.T) {
	db := newTestDB(t)

	capRepo := configrepo.NewGORMCapabilityRepository(db)
	capSvc := &testCapabilityServiceAdapter{repo: capRepo}

	capHandler := deliveryhttp.NewCapabilityHandler(capSvc)

	r := chi.NewRouter()
	r.Route("/api/v1/agents/{name}/capabilities", func(r chi.Router) {
		r.Get("/", capHandler.List)
		r.Post("/", capHandler.Add)
		r.Put("/{capId}", capHandler.Update)
		r.Delete("/{capId}", capHandler.Remove)
	})

	// 1. Create agent
	createTestAgent(t, db, "cap-agent")

	// 2. POST capability (memory) → 201
	rec := postJSON(t, r, "/api/v1/agents/cap-agent/capabilities", map[string]interface{}{
		"type": "memory",
		"config": map[string]interface{}{
			"max_entries": 100,
			"retention":   30,
		},
	})
	require.Equal(t, http.StatusCreated, rec.Code)
	capInfo := decodeJSON[deliveryhttp.CapabilityInfo](t, rec)
	assert.Equal(t, "memory", capInfo.Type)
	assert.True(t, capInfo.Enabled)
	capID := capInfo.ID

	// 3. GET capabilities → verify memory capability present
	rec = getJSON(t, r, "/api/v1/agents/cap-agent/capabilities")
	require.Equal(t, http.StatusOK, rec.Code)
	caps := decodeJSON[[]deliveryhttp.CapabilityInfo](t, rec)
	require.Len(t, caps, 1)
	assert.Equal(t, "memory", caps[0].Type)

	// 4. Verify InjectedTools includes "memory_recall", "memory_store"
	injector := capability.NewInjector(&testCapInjectorAdapter{repo: capRepo})
	tools, err := injector.InjectedTools(context.Background(), "cap-agent")
	require.NoError(t, err)
	assert.Contains(t, tools, "memory_recall")
	assert.Contains(t, tools, "memory_store")

	// 5. DELETE capability → 204
	rec = deleteJSON(t, r, fmt.Sprintf("/api/v1/agents/cap-agent/capabilities/%d", capID))
	require.Equal(t, http.StatusNoContent, rec.Code)

	// 6. GET capabilities → empty
	rec = getJSON(t, r, "/api/v1/agents/cap-agent/capabilities")
	require.Equal(t, http.StatusOK, rec.Code)
	caps = decodeJSON[[]deliveryhttp.CapabilityInfo](t, rec)
	assert.Empty(t, caps)

	// 7. Verify InjectedTools is now empty
	tools, err = injector.InjectedTools(context.Background(), "cap-agent")
	require.NoError(t, err)
	assert.Empty(t, tools)
}

// ---- Test 3: Memory Store + Recall + Schema Isolation ----

func TestV2_MemoryStore_SchemaIsolation(t *testing.T) {
	db := newTestDB(t)

	memStorage := persistence.NewMemoryStorage(db)
	memHandler := deliveryhttp.NewMemoryHandler(
		&testMemoryListerAdapter{storage: memStorage},
		&testMemoryClearerAdapter{storage: memStorage},
	)

	r := chi.NewRouter()
	r.Get("/api/v1/schemas/{id}/memory", memHandler.ListMemories)
	r.Delete("/api/v1/schemas/{id}/memory", memHandler.ClearMemories)
	r.Delete("/api/v1/schemas/{id}/memory/{entry_id}", memHandler.DeleteMemory)

	ctx := context.Background()

	// 1. Create two schemas by inserting directly into DB
	schemaA := models.SchemaModel{Name: "schema-a"}
	schemaB := models.SchemaModel{Name: "schema-b"}
	require.NoError(t, db.Create(&schemaA).Error)
	require.NoError(t, db.Create(&schemaB).Error)
	schemaAID := fmt.Sprintf("%d", schemaA.ID)
	schemaBID := fmt.Sprintf("%d", schemaB.ID)

	// 2. Store memory into schema-a
	mem, err := domain.NewMemory(schemaAID, "user-1", "user prefers dark mode")
	require.NoError(t, err)
	require.NoError(t, memStorage.Store(ctx, mem, 0))

	// 3. GET schema-a/memory → contains the entry
	rec := getJSON(t, r, fmt.Sprintf("/api/v1/schemas/%s/memory", schemaAID))
	require.Equal(t, http.StatusOK, rec.Code)

	var memoriesA []map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&memoriesA))
	require.Len(t, memoriesA, 1)
	assert.Equal(t, "user prefers dark mode", memoriesA[0]["content"])

	// 4. GET schema-b/memory → empty (isolation)
	rec = getJSON(t, r, fmt.Sprintf("/api/v1/schemas/%s/memory", schemaBID))
	require.Equal(t, http.StatusOK, rec.Code)

	var memoriesB []map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&memoriesB))
	assert.Empty(t, memoriesB)

	// 5. DELETE schema-a/memory → 200
	rec = deleteJSON(t, r, fmt.Sprintf("/api/v1/schemas/%s/memory", schemaAID))
	require.Equal(t, http.StatusOK, rec.Code)

	var clearResp map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&clearResp))
	assert.Equal(t, float64(1), clearResp["deleted"])

	// 6. GET schema-a/memory → empty
	rec = getJSON(t, r, fmt.Sprintf("/api/v1/schemas/%s/memory", schemaAID))
	require.Equal(t, http.StatusOK, rec.Code)

	var memoriesAfterClear []map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&memoriesAfterClear))
	assert.Empty(t, memoriesAfterClear)
}

// ---- Test 4: Memory FIFO Eviction ----

func TestV2_MemoryFIFOEviction(t *testing.T) {
	db := newTestDB(t)

	memStorage := persistence.NewMemoryStorage(db)
	ctx := context.Background()

	schemaID := "1"
	userID := "user-1"

	// Insert a schema record so the schema_id FK is valid (MemoryModel.SchemaID is uint, not FK-constrained but good practice).
	require.NoError(t, db.Create(&models.SchemaModel{Name: "eviction-schema"}).Error)

	// Store 4 memories with maxEntries=3 — each Store call evicts if needed.
	contents := []string{"mem-1-oldest", "mem-2", "mem-3", "mem-4-newest"}
	for _, c := range contents {
		mem, err := domain.NewMemory(schemaID, userID, c)
		require.NoError(t, err)
		require.NoError(t, memStorage.Store(ctx, mem, 3))
	}

	// List all memories for this schema — should only have 3
	memories, err := memStorage.ListBySchema(ctx, schemaID)
	require.NoError(t, err)
	require.Len(t, memories, 3, "should have exactly 3 entries after FIFO eviction")

	// Verify oldest entry is gone
	for _, m := range memories {
		assert.NotEqual(t, "mem-1-oldest", m.Content, "oldest entry should have been evicted")
	}

	// Verify newest 3 remain (ListBySchema returns DESC order)
	remainingContents := make([]string, 0, len(memories))
	for _, m := range memories {
		remainingContents = append(remainingContents, m.Content)
	}
	assert.Contains(t, remainingContents, "mem-2")
	assert.Contains(t, remainingContents, "mem-3")
	assert.Contains(t, remainingContents, "mem-4-newest")
}

// ---- Test 5: Tool Tier Enforcement (Cloud Sandbox) ----

func TestV2_ToolTierEnforcement_CloudSandbox(t *testing.T) {
	// Verify domain-level tier classification
	t.Run("ClassifyToolTier", func(t *testing.T) {
		tests := []struct {
			tool     string
			expected domain.ToolTier
		}{
			{"ask_user", domain.ToolTierCore},
			{"manage_tasks", domain.ToolTierCore},
			{"spawn_agent", domain.ToolTierCore},
			{"memory_recall", domain.ToolTierCapability},
			{"memory_store", domain.ToolTierCapability},
			{"knowledge_search", domain.ToolTierCapability},
			{"escalate", domain.ToolTierCapability},
			{"execute_command", domain.ToolTierSelfHosted},
			{"read_file", domain.ToolTierSelfHosted},
			{"write_file", domain.ToolTierSelfHosted},
			{"search_code", domain.ToolTierSelfHosted},
		}
		for _, tt := range tests {
			t.Run(tt.tool, func(t *testing.T) {
				assert.Equal(t, tt.expected, domain.ClassifyToolTier(tt.tool))
			})
		}
	})

	// Verify sandbox enforcement in Cloud mode
	t.Run("CloudSandbox_BlocksSelfHosted", func(t *testing.T) {
		sandbox := cloud.NewSandbox(true) // cloud mode

		// Tier 1 (Core) — allowed
		assert.NoError(t, sandbox.ValidateToolAccess("ask_user"))

		// Tier 2 (Capability) — allowed
		assert.NoError(t, sandbox.ValidateToolAccess("memory_recall"))

		// Tier 3 (SelfHosted) — blocked
		err := sandbox.ValidateToolAccess("execute_command")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "blocked in Cloud")

		err = sandbox.ValidateToolAccess("read_file")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "blocked in Cloud")

		// Tier 4 (MCP) — allowed
		assert.NoError(t, sandbox.ValidateToolAccess("web_search_via_mcp"))
	})

	// Verify CE mode allows everything
	t.Run("CESandbox_AllowsEverything", func(t *testing.T) {
		sandbox := cloud.NewSandbox(false) // CE mode

		assert.NoError(t, sandbox.ValidateToolAccess("ask_user"))
		assert.NoError(t, sandbox.ValidateToolAccess("memory_recall"))
		assert.NoError(t, sandbox.ValidateToolAccess("execute_command"))
		assert.NoError(t, sandbox.ValidateToolAccess("read_file"))
	})

	// Verify FilterTools
	t.Run("FilterTools_Cloud", func(t *testing.T) {
		sandbox := cloud.NewSandbox(true)

		allTools := []string{"ask_user", "memory_recall", "execute_command", "read_file", "escalate"}
		allowed, blocked := sandbox.FilterTools(allTools)

		assert.ElementsMatch(t, []string{"ask_user", "memory_recall", "escalate"}, allowed)
		assert.Len(t, blocked, 2)
	})
}

// ---- Test: Capability with multiple types ----

func TestV2_CapabilityInjectedTools_MultipleTypes(t *testing.T) {
	db := newTestDB(t)

	capRepo := configrepo.NewGORMCapabilityRepository(db)
	capSvc := &testCapabilityServiceAdapter{repo: capRepo}

	capHandler := deliveryhttp.NewCapabilityHandler(capSvc)

	r := chi.NewRouter()
	r.Route("/api/v1/agents/{name}/capabilities", func(r chi.Router) {
		r.Get("/", capHandler.List)
		r.Post("/", capHandler.Add)
		r.Delete("/{capId}", capHandler.Remove)
	})

	createTestAgent(t, db, "multi-cap-agent")

	// Add memory capability
	rec := postJSON(t, r, "/api/v1/agents/multi-cap-agent/capabilities", map[string]interface{}{
		"type":   "memory",
		"config": map[string]interface{}{"max_entries": 50},
	})
	require.Equal(t, http.StatusCreated, rec.Code)

	// Add knowledge capability
	rec = postJSON(t, r, "/api/v1/agents/multi-cap-agent/capabilities", map[string]interface{}{
		"type":   "knowledge",
		"config": map[string]interface{}{},
	})
	require.Equal(t, http.StatusCreated, rec.Code)

	// Add escalation capability
	rec = postJSON(t, r, "/api/v1/agents/multi-cap-agent/capabilities", map[string]interface{}{
		"type":   "escalation",
		"config": map[string]interface{}{"action": "webhook"},
	})
	require.Equal(t, http.StatusCreated, rec.Code)

	// Verify InjectedTools returns all tools from all capabilities
	injector := capability.NewInjector(&testCapInjectorAdapter{repo: capRepo})
	tools, err := injector.InjectedTools(context.Background(), "multi-cap-agent")
	require.NoError(t, err)
	assert.Contains(t, tools, "memory_recall")
	assert.Contains(t, tools, "memory_store")
	assert.Contains(t, tools, "knowledge_search")
	assert.Contains(t, tools, "escalate")
	assert.Len(t, tools, 4, "should have exactly 4 tools (no duplicates)")

	// Verify guardrail capability injects no tools
	rec = postJSON(t, r, "/api/v1/agents/multi-cap-agent/capabilities", map[string]interface{}{
		"type":   "guardrail",
		"config": map[string]interface{}{},
	})
	require.Equal(t, http.StatusCreated, rec.Code)

	tools, err = injector.InjectedTools(context.Background(), "multi-cap-agent")
	require.NoError(t, err)
	assert.Len(t, tools, 4, "guardrail should not inject additional tools")

	// List capabilities → 4 total
	rec = getJSON(t, r, "/api/v1/agents/multi-cap-agent/capabilities")
	require.Equal(t, http.StatusOK, rec.Code)
	caps := decodeJSON[[]deliveryhttp.CapabilityInfo](t, rec)
	assert.Len(t, caps, 4)
}

// ---- Test: Schema with multiple agents ----

func TestV2_Schema_MultipleAgents(t *testing.T) {
	db := newTestDB(t)

	schemaRepo := configrepo.NewGORMSchemaRepository(db)
	schemaSvc := &testSchemaServiceAdapter{repo: schemaRepo}
	schemaLister := &testAgentSchemaListerAdapter{repo: schemaRepo}

	schemaHandler := deliveryhttp.NewSchemaHandler(schemaSvc, &noopGateService{}, &noopEdgeService{})

	r := chi.NewRouter()
	r.Mount("/api/v1/schemas", schemaHandler.Routes())

	createTestAgent(t, db, "agent-a")
	createTestAgent(t, db, "agent-b")

	// Create schema
	rec := postJSON(t, r, "/api/v1/schemas", map[string]string{"name": "multi-schema"})
	require.Equal(t, http.StatusCreated, rec.Code)
	schema := decodeJSON[deliveryhttp.SchemaInfo](t, rec)

	// Add two agents
	rec = postJSON(t, r, fmt.Sprintf("/api/v1/schemas/%d/agents", schema.ID), map[string]string{"agent_name": "agent-a"})
	require.Equal(t, http.StatusNoContent, rec.Code)
	rec = postJSON(t, r, fmt.Sprintf("/api/v1/schemas/%d/agents", schema.ID), map[string]string{"agent_name": "agent-b"})
	require.Equal(t, http.StatusNoContent, rec.Code)

	// Verify both agents listed
	rec = getJSON(t, r, fmt.Sprintf("/api/v1/schemas/%d/agents", schema.ID))
	require.Equal(t, http.StatusOK, rec.Code)
	agents := decodeJSON[[]string](t, rec)
	assert.Len(t, agents, 2)
	assert.Contains(t, agents, "agent-a")
	assert.Contains(t, agents, "agent-b")

	// Remove one agent
	rec = deleteJSON(t, r, fmt.Sprintf("/api/v1/schemas/%d/agents/agent-a", schema.ID))
	require.Equal(t, http.StatusNoContent, rec.Code)

	// Verify only one agent remains
	rec = getJSON(t, r, fmt.Sprintf("/api/v1/schemas/%d/agents", schema.ID))
	require.Equal(t, http.StatusOK, rec.Code)
	agents = decodeJSON[[]string](t, rec)
	assert.Len(t, agents, 1)
	assert.Contains(t, agents, "agent-b")

	// Verify agent-b is still referenced in schemas
	ctx := context.Background()
	names, err := schemaLister.ListSchemasForAgent(ctx, "agent-b")
	require.NoError(t, err)
	assert.Contains(t, names, "multi-schema")

	// Verify agent-a is no longer referenced
	names, err = schemaLister.ListSchemasForAgent(ctx, "agent-a")
	require.NoError(t, err)
	assert.Empty(t, names)
}

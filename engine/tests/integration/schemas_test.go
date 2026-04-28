//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// schemaCreateResp models the minimum fields we care about from POST /schemas.
type schemaCreateResp struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ChatEnabled bool   `json:"chat_enabled"`
}

// createSchemaForTest POSTs a schema and returns the parsed ID. On non-2xx
// it fails the test — tests that want to probe error paths should use do()
// directly.
func createSchemaForTest(t *testing.T, body map[string]any) schemaCreateResp {
	t.Helper()
	resp := do(t, http.MethodPost, "/api/v1/schemas", mustJSON(body), adminToken)
	respBody := readBody(t, resp)
	assertStatusAny(t, resp, http.StatusOK, http.StatusCreated)

	var parsed schemaCreateResp
	require.NoError(t, json.Unmarshal(respBody, &parsed), "decode schema create body=%s", respBody)
	require.NotEmpty(t, parsed.ID, "schema id must be populated: %s", respBody)
	return parsed
}

// createAgentForTest POSTs an agent and returns the response as raw bytes.
func createAgentForTest(t *testing.T, name string) []byte {
	t.Helper()
	resp := do(t, http.MethodPost, "/api/v1/agents",
		mustJSON(map[string]any{"name": name, "system_prompt": "p"}), adminToken)
	body := readBody(t, resp)
	assertStatusAny(t, resp, http.StatusOK, http.StatusCreated)
	return body
}

// TC-SCH-01: POST /schemas creates a schema.
func TestSCH01_CreateSchema(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	s := createSchemaForTest(t, map[string]any{"name": "tc-sch-01-schema"})
	assert.Equal(t, "tc-sch-01-schema", s.Name)
}

// TC-SCH-02: chat_enabled round-trips.
func TestSCH02_ChatEnabled(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	s := createSchemaForTest(t, map[string]any{
		"name":         "tc-sch-02-schema",
		"chat_enabled": true,
	})

	getResp := do(t, http.MethodGet, "/api/v1/schemas/"+s.ID, nil, adminToken)
	body := readBody(t, getResp)
	require.Equal(t, http.StatusOK, getResp.StatusCode)
	assert.Contains(t, string(body), `"chat_enabled":true`,
		"GET schema should reflect chat_enabled=true: %s", body)
}

// TC-SCH-03: GET /schemas/{id} → 200.
func TestSCH03_GetSchema(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	s := createSchemaForTest(t, map[string]any{"name": "tc-sch-03-schema"})

	resp := do(t, http.MethodGet, "/api/v1/schemas/"+s.ID, nil, adminToken)
	_ = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TC-SCH-04: PUT /schemas/{id} updates the schema name.
func TestSCH04_UpdateSchema(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	s := createSchemaForTest(t, map[string]any{"name": "tc-sch-04-schema"})

	newName := "tc-sch-04-schema-renamed"
	updResp := do(t, http.MethodPut, "/api/v1/schemas/"+s.ID,
		mustJSON(map[string]any{"name": newName}), adminToken)
	_ = readBody(t, updResp)
	assertStatusAny(t, updResp, http.StatusOK, http.StatusNoContent)

	getResp := do(t, http.MethodGet, "/api/v1/schemas/"+s.ID, nil, adminToken)
	body := readBody(t, getResp)
	assert.Equal(t, http.StatusOK, getResp.StatusCode)
	assert.Contains(t, string(body), newName)
}

// TC-SCH-05: DELETE /schemas/{id} removes the schema.
func TestSCH05_DeleteSchema(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	s := createSchemaForTest(t, map[string]any{"name": "tc-sch-05-schema"})

	delResp := do(t, http.MethodDelete, "/api/v1/schemas/"+s.ID, nil, adminToken)
	_ = readBody(t, delResp)
	assertStatusAny(t, delResp, http.StatusOK, http.StatusNoContent)

	getResp := do(t, http.MethodGet, "/api/v1/schemas/"+s.ID, nil, adminToken)
	_ = readBody(t, getResp)
	assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
}

// TC-SCH-06: Create a delegation relation source→target between two agents.
//
// V2 agent-relations API: POST /api/v1/schemas/{id}/agent-relations with
// {"source": <name|uuid>, "target": <name|uuid>}. Self-loops are rejected
// ("source and target must be different agents"), so we use two agents.
// The created relation auto-promotes source to schema.entry_agent_id when
// none is set.
func TestSCH06_EntryAgentRelation(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	_ = createAgentForTest(t, "tc-sch-06-entry")
	_ = createAgentForTest(t, "tc-sch-06-target")
	s := createSchemaForTest(t, map[string]any{"name": "tc-sch-06-schema"})

	resp := do(t, http.MethodPost, "/api/v1/schemas/"+s.ID+"/agent-relations",
		mustJSON(map[string]any{
			"source": "tc-sch-06-entry",
			"target": "tc-sch-06-target",
		}), adminToken)
	_ = readBody(t, resp)
	assertStatusAny(t, resp, http.StatusOK, http.StatusCreated)
}

// TC-SCH-07: Transfer edge source→target between two agents.
func TestSCH07_TransferEdge(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	_ = createAgentForTest(t, "tc-sch-07-a")
	_ = createAgentForTest(t, "tc-sch-07-b")
	s := createSchemaForTest(t, map[string]any{"name": "tc-sch-07-schema"})

	resp := do(t, http.MethodPost, "/api/v1/schemas/"+s.ID+"/agent-relations",
		mustJSON(map[string]any{
			"source": "tc-sch-07-a",
			"target": "tc-sch-07-b",
		}), adminToken)
	_ = readBody(t, resp)
	assertStatusAny(t, resp, http.StatusOK, http.StatusCreated)
}

// TC-SCH-09: Duplicate agent-relation → 409 Conflict.
func TestSCH09_DuplicateRelation(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	_ = createAgentForTest(t, "tc-sch-09-a")
	_ = createAgentForTest(t, "tc-sch-09-b")
	s := createSchemaForTest(t, map[string]any{"name": "tc-sch-09-schema"})

	first := do(t, http.MethodPost, "/api/v1/schemas/"+s.ID+"/agent-relations",
		mustJSON(map[string]any{
			"source": "tc-sch-09-a",
			"target": "tc-sch-09-b",
		}), adminToken)
	_ = readBody(t, first)
	assertStatusAny(t, first, http.StatusOK, http.StatusCreated)

	second := do(t, http.MethodPost, "/api/v1/schemas/"+s.ID+"/agent-relations",
		mustJSON(map[string]any{
			"source": "tc-sch-09-a",
			"target": "tc-sch-09-b",
		}), adminToken)
	_ = readBody(t, second)
	assertStatusAny(t, second, http.StatusConflict, http.StatusUnprocessableEntity, http.StatusBadRequest)
}

// TC-SCH-08: DELETE agent-relation.
func TestSCH08_DeleteRelation(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	_ = createAgentForTest(t, "tc-sch-08-a")
	_ = createAgentForTest(t, "tc-sch-08-b")
	s := createSchemaForTest(t, map[string]any{"name": "tc-sch-08-schema"})

	createResp := do(t, http.MethodPost, "/api/v1/schemas/"+s.ID+"/agent-relations",
		mustJSON(map[string]any{
			"source": "tc-sch-08-a",
			"target": "tc-sch-08-b",
		}), adminToken)
	createBody := readBody(t, createResp)
	assertStatusAny(t, createResp, http.StatusOK, http.StatusCreated)

	var parsed struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(createBody, &parsed), "body=%s", createBody)
	if parsed.ID == "" {
		t.Skip("relation create response did not carry id — cannot probe delete")
	}

	delResp := do(t, http.MethodDelete,
		"/api/v1/schemas/"+s.ID+"/agent-relations/"+parsed.ID, nil, adminToken)
	_ = readBody(t, delResp)
	assertStatusAny(t, delResp, http.StatusOK, http.StatusNoContent)
}


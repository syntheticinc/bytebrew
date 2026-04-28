//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TC-AGT-01: POST /agents creates a new agent and returns the name in the body.
func TestAGT01_CreateAgent(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	name := "tc-agt-01-agent"
	resp := do(t, http.MethodPost, "/api/v1/agents",
		mustJSON(map[string]any{
			"name":          name,
			"system_prompt": "test prompt",
		}), adminToken)
	body := readBody(t, resp)
	assertStatusAny(t, resp, http.StatusOK, http.StatusCreated)
	assert.Contains(t, string(body), name, "create response should contain the agent name")
}

// TC-AGT-02: GET /agents lists the created agent.
func TestAGT02_ListContainsCreated(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	name := "tc-agt-02-agent"
	createResp := do(t, http.MethodPost, "/api/v1/agents",
		mustJSON(map[string]any{"name": name, "system_prompt": "p"}), adminToken)
	_ = readBody(t, createResp)
	assertStatusAny(t, createResp, http.StatusOK, http.StatusCreated)

	listResp := do(t, http.MethodGet, "/api/v1/agents", nil, adminToken)
	body := readBody(t, listResp)
	require.Equal(t, http.StatusOK, listResp.StatusCode)
	assert.Contains(t, string(body), `"name":"`+name+`"`,
		"list should contain created agent: %s", body)
}

// TC-AGT-03: GET /agents/{name} on an existing agent → 200.
func TestAGT03_GetExisting(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	name := "tc-agt-03-agent"
	createResp := do(t, http.MethodPost, "/api/v1/agents",
		mustJSON(map[string]any{"name": name, "system_prompt": "p"}), adminToken)
	_ = readBody(t, createResp)
	assertStatusAny(t, createResp, http.StatusOK, http.StatusCreated)

	getResp := do(t, http.MethodGet, "/api/v1/agents/"+name, nil, adminToken)
	body := readBody(t, getResp)
	assert.Equal(t, http.StatusOK, getResp.StatusCode, "body=%s", body)
}

// TC-AGT-04: GET /agents/{name} on nonexistent agent → 404.
func TestAGT04_GetNonexistent(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	resp := do(t, http.MethodGet, "/api/v1/agents/does-not-exist", nil, adminToken)
	_ = readBody(t, resp)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TC-AGT-05: PUT /agents/{name} updates system_prompt; subsequent GET
// reflects the change.
func TestAGT05_UpdateAgent(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	name := "tc-agt-05-agent"
	createResp := do(t, http.MethodPost, "/api/v1/agents",
		mustJSON(map[string]any{"name": name, "system_prompt": "initial"}), adminToken)
	_ = readBody(t, createResp)
	assertStatusAny(t, createResp, http.StatusOK, http.StatusCreated)

	newPrompt := "updated-prompt-value-xyz"
	updResp := do(t, http.MethodPut, "/api/v1/agents/"+name,
		mustJSON(map[string]any{
			"name":          name,
			"system_prompt": newPrompt,
		}), adminToken)
	_ = readBody(t, updResp)
	assertStatusAny(t, updResp, http.StatusOK, http.StatusNoContent)

	getResp := do(t, http.MethodGet, "/api/v1/agents/"+name, nil, adminToken)
	body := readBody(t, getResp)
	require.Equal(t, http.StatusOK, getResp.StatusCode)
	assert.Contains(t, string(body), newPrompt,
		"GET after PUT should reflect updated prompt: %s", body)
}

// TC-AGT-06: DELETE /agents/{name} removes the agent; subsequent GET → 404.
func TestAGT06_DeleteAgent(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	name := "tc-agt-06-agent"
	createResp := do(t, http.MethodPost, "/api/v1/agents",
		mustJSON(map[string]any{"name": name, "system_prompt": "p"}), adminToken)
	_ = readBody(t, createResp)
	assertStatusAny(t, createResp, http.StatusOK, http.StatusCreated)

	delResp := do(t, http.MethodDelete, "/api/v1/agents/"+name, nil, adminToken)
	_ = readBody(t, delResp)
	assertStatusAny(t, delResp, http.StatusOK, http.StatusNoContent)

	getResp := do(t, http.MethodGet, "/api/v1/agents/"+name, nil, adminToken)
	_ = readBody(t, getResp)
	assert.Equal(t, http.StatusNotFound, getResp.StatusCode,
		"deleted agent must not be fetchable")
}

// TC-AGT-07: Duplicate name → 409 Conflict (422 also accepted).
func TestAGT07_DuplicateName(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	name := "tc-agt-07-agent"
	first := do(t, http.MethodPost, "/api/v1/agents",
		mustJSON(map[string]any{"name": name, "system_prompt": "first"}), adminToken)
	_ = readBody(t, first)
	assertStatusAny(t, first, http.StatusOK, http.StatusCreated)

	second := do(t, http.MethodPost, "/api/v1/agents",
		mustJSON(map[string]any{"name": name, "system_prompt": "second"}), adminToken)
	_ = readBody(t, second)
	assertStatusAny(t, second, http.StatusConflict, http.StatusUnprocessableEntity, http.StatusBadRequest)
}

// TC-AGT-09: Immediate GET after create — no registry staleness.
func TestAGT09_ImmediateRead(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	name := "tc-agt-09-agent"
	createResp := do(t, http.MethodPost, "/api/v1/agents",
		mustJSON(map[string]any{"name": name, "system_prompt": "p"}), adminToken)
	_ = readBody(t, createResp)
	assertStatusAny(t, createResp, http.StatusOK, http.StatusCreated)

	getResp := do(t, http.MethodGet, "/api/v1/agents/"+name, nil, adminToken)
	_ = readBody(t, getResp)
	assert.Equal(t, http.StatusOK, getResp.StatusCode,
		"agent must be readable immediately after create")
}

// TC-AGT-10: public=true is accepted by the schema.
func TestAGT10_PublicFlag(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	resp := do(t, http.MethodPost, "/api/v1/agents",
		mustJSON(map[string]any{
			"name":          "tc-agt-10-agent",
			"system_prompt": "p",
			"public":        true,
		}), adminToken)
	body := readBody(t, resp)
	assertStatusAny(t, resp, http.StatusOK, http.StatusCreated)
	_ = body
}

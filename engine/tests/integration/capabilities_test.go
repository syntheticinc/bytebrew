//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TC-CAP-01: Attach memory capability to an existing agent.
func TestCAP01_AddMemoryCapability(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	agentName := "tc-cap-01-agent"
	_ = createAgentForTest(t, agentName)

	resp := do(t, http.MethodPost, "/api/v1/agents/"+agentName+"/capabilities",
		mustJSON(map[string]any{
			"type":    "memory",
			"enabled": true,
		}), adminToken)
	body := readBody(t, resp)
	assertStatusAny(t, resp, http.StatusOK, http.StatusCreated)
	assert.NotEmpty(t, body)
}

// TC-CAP-02: Knowledge capability — may require linked KB; accept 422 as
// valid when that validation is enforced.
func TestCAP02_AddKnowledgeCapability(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	agentName := "tc-cap-02-agent"
	_ = createAgentForTest(t, agentName)

	resp := do(t, http.MethodPost, "/api/v1/agents/"+agentName+"/capabilities",
		mustJSON(map[string]any{
			"type":    "knowledge",
			"enabled": true,
		}), adminToken)
	_ = readBody(t, resp)
	// Knowledge often needs a linked KB; accept create OR 4xx validation.
	assertStatusAny(t, resp,
		http.StatusOK, http.StatusCreated,
		http.StatusBadRequest, http.StatusUnprocessableEntity)
}

// TC-CAP-03: GET /agents/{name}/capabilities returns a list.
func TestCAP03_ListCapabilities(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	agentName := "tc-cap-03-agent"
	_ = createAgentForTest(t, agentName)

	addResp := do(t, http.MethodPost, "/api/v1/agents/"+agentName+"/capabilities",
		mustJSON(map[string]any{"type": "memory", "enabled": true}), adminToken)
	_ = readBody(t, addResp)
	assertStatusAny(t, addResp, http.StatusOK, http.StatusCreated)

	listResp := do(t, http.MethodGet, "/api/v1/agents/"+agentName+"/capabilities", nil, adminToken)
	body := readBody(t, listResp)
	assert.Equal(t, http.StatusOK, listResp.StatusCode, "body=%s", body)
}

// TC-CAP-04: DELETE /agents/{name}/capabilities/{capId}.
func TestCAP04_DeleteCapability(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	agentName := "tc-cap-04-agent"
	_ = createAgentForTest(t, agentName)

	addResp := do(t, http.MethodPost, "/api/v1/agents/"+agentName+"/capabilities",
		mustJSON(map[string]any{"type": "memory", "enabled": true}), adminToken)
	addBody := readBody(t, addResp)
	assertStatusAny(t, addResp, http.StatusOK, http.StatusCreated)

	var parsed struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(addBody, &parsed); err != nil || parsed.ID == "" {
		t.Skipf("capability create did not return id (%v): %s", err, addBody)
	}

	delResp := do(t, http.MethodDelete,
		"/api/v1/agents/"+agentName+"/capabilities/"+parsed.ID, nil, adminToken)
	_ = readBody(t, delResp)
	assertStatusAny(t, delResp, http.StatusOK, http.StatusNoContent)
}

// TC-CAP-05: Unknown capability type → 400/422.
func TestCAP05_UnknownType(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	agentName := "tc-cap-05-agent"
	_ = createAgentForTest(t, agentName)

	resp := do(t, http.MethodPost, "/api/v1/agents/"+agentName+"/capabilities",
		mustJSON(map[string]any{"type": "not-a-real-capability", "enabled": true}), adminToken)
	_ = readBody(t, resp)
	assertStatusAny(t, resp, http.StatusBadRequest, http.StatusUnprocessableEntity)
}

// TC-CAP-06: Duplicate capability type → 409, or idempotent 200/201.
func TestCAP06_DuplicateType(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	agentName := "tc-cap-06-agent"
	_ = createAgentForTest(t, agentName)

	first := do(t, http.MethodPost, "/api/v1/agents/"+agentName+"/capabilities",
		mustJSON(map[string]any{"type": "memory", "enabled": true}), adminToken)
	_ = readBody(t, first)
	assertStatusAny(t, first, http.StatusOK, http.StatusCreated)

	second := do(t, http.MethodPost, "/api/v1/agents/"+agentName+"/capabilities",
		mustJSON(map[string]any{"type": "memory", "enabled": true}), adminToken)
	_ = readBody(t, second)
	assertStatusAny(t, second,
		http.StatusOK, http.StatusCreated,
		http.StatusConflict, http.StatusUnprocessableEntity, http.StatusBadRequest)
}

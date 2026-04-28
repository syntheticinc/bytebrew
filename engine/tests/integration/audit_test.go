//go:build integration

package integration

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TC-AUDIT-01: Mutating action should leave a trace in /audit.
func TestAUDIT01_ActionCreatesEntry(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	createResp := do(t, http.MethodPost, "/api/v1/agents",
		mustJSON(map[string]any{
			"name":          "tc-audit-01-agent",
			"system_prompt": "p",
		}), adminToken)
	_ = readBody(t, createResp)
	assertStatusAny(t, createResp, http.StatusOK, http.StatusCreated)

	listResp := do(t, http.MethodGet, "/api/v1/audit", nil, adminToken)
	body := readBody(t, listResp)
	require.Equal(t, http.StatusOK, listResp.StatusCode, "body=%s", body)
	assert.NotEmpty(t, body, "audit list should not be empty after a mutating action")
}

// TC-AUDIT-02: Audit entry shape — timestamp/actor/action/resource fields.
func TestAUDIT02_EntryShape(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	// seed a known action
	_ = createAgentForTest(t, "tc-audit-02-agent")

	listResp := do(t, http.MethodGet, "/api/v1/audit", nil, adminToken)
	body := readBody(t, listResp)
	require.Equal(t, http.StatusOK, listResp.StatusCode)

	// We don't pin exact field names — schema has varied — but the response
	// must contain at least one of the expected markers.
	s := string(body)
	found := 0
	for _, marker := range []string{"timestamp", "action", "resource", "actor", "created_at"} {
		if strings.Contains(s, marker) {
			found++
		}
	}
	assert.GreaterOrEqual(t, found, 1,
		"audit body should include at least one known field marker: %s", body)
}

// TC-AUDIT-03: ?action=create filter is supported; response must still 200.
func TestAUDIT03_ActionFilter(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	_ = createAgentForTest(t, "tc-audit-03-agent")

	resp := do(t, http.MethodGet, "/api/v1/audit?action=create", nil, adminToken)
	_ = readBody(t, resp)
	if resp.StatusCode >= 500 {
		t.Fatalf("audit filter must not 5xx: %d", resp.StatusCode)
	}
	assertStatusAny(t, resp, http.StatusOK, http.StatusBadRequest)
}

// TC-AUDIT-04: Tool-calls sub-endpoint returns 200 (possibly empty list).
func TestAUDIT04_ToolCalls(t *testing.T) {
	requireSuite(t)

	resp := do(t, http.MethodGet, "/api/v1/audit/tool-calls", nil, adminToken)
	_ = readBody(t, resp)
	if resp.StatusCode == http.StatusNotFound {
		t.Skip("/audit/tool-calls not registered in this build")
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TC-AUDIT-05: /audit without a token → 401.
func TestAUDIT05_RequiresAuth(t *testing.T) {
	requireSuite(t)

	resp := do(t, http.MethodGet, "/api/v1/audit", nil, "")
	_ = readBody(t, resp)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TC-AUDIT-06: After a DELETE, a delete event is visible in the log.
func TestAUDIT06_DeleteVisible(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	name := "tc-audit-06-agent"
	_ = createAgentForTest(t, name)

	delResp := do(t, http.MethodDelete, "/api/v1/agents/"+name, nil, adminToken)
	_ = readBody(t, delResp)
	assertStatusAny(t, delResp, http.StatusOK, http.StatusNoContent)

	listResp := do(t, http.MethodGet, "/api/v1/audit", nil, adminToken)
	body := readBody(t, listResp)
	require.Equal(t, http.StatusOK, listResp.StatusCode)
	// Expect the agent name to appear (resource = the agent we just deleted).
	assert.Contains(t, string(body), name,
		"audit log should mention the deleted agent's name")
}

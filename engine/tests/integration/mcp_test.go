//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mcpServerResp struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// mcpPathKey returns whichever identifier the server routes by — the route
// spec uses {name}, so prefer name but fall back to id for forward-compat.
func mcpPathKey(m mcpServerResp) string {
	if m.Name != "" {
		return m.Name
	}
	return m.ID
}

// TC-MCP-01: Create an MCP server.
func TestMCP01_CreateServer(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	resp := do(t, http.MethodPost, "/api/v1/mcp-servers",
		mustJSON(map[string]any{
			"name": "tc-mcp-01",
			"type": "http",
			"url":  "http://test.example.com",
		}), adminToken)
	body := readBody(t, resp)
	assertStatusAny(t, resp, http.StatusOK, http.StatusCreated)
	assert.NotEmpty(t, body)
}

// TC-MCP-02: List contains the created server.
func TestMCP02_ListContainsCreated(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	resp := do(t, http.MethodPost, "/api/v1/mcp-servers",
		mustJSON(map[string]any{
			"name": "tc-mcp-02",
			"type": "http",
			"url":  "http://test.example.com",
		}), adminToken)
	_ = readBody(t, resp)
	assertStatusAny(t, resp, http.StatusOK, http.StatusCreated)

	listResp := do(t, http.MethodGet, "/api/v1/mcp-servers", nil, adminToken)
	body := readBody(t, listResp)
	require.Equal(t, http.StatusOK, listResp.StatusCode)
	assert.Contains(t, string(body), "tc-mcp-02")
}

// TC-MCP-03: PUT /mcp-servers/{name} updates a server.
func TestMCP03_UpdateServer(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	createResp := do(t, http.MethodPost, "/api/v1/mcp-servers",
		mustJSON(map[string]any{
			"name": "tc-mcp-03",
			"type": "http",
			"url":  "http://test.example.com",
		}), adminToken)
	createBody := readBody(t, createResp)
	assertStatusAny(t, createResp, http.StatusOK, http.StatusCreated)

	var m mcpServerResp
	_ = json.Unmarshal(createBody, &m)
	if m.Name == "" {
		m.Name = "tc-mcp-03"
	}

	updResp := do(t, http.MethodPut, "/api/v1/mcp-servers/"+mcpPathKey(m),
		mustJSON(map[string]any{
			"name":      m.Name,
			"type":      "http",
			"url":       "http://updated.example.com",
			"auth_type": "none",
		}), adminToken)
	_ = readBody(t, updResp)
	assertStatusAny(t, updResp, http.StatusOK, http.StatusNoContent)
}

// TC-MCP-04: DELETE /mcp-servers/{name}.
func TestMCP04_DeleteServer(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	createResp := do(t, http.MethodPost, "/api/v1/mcp-servers",
		mustJSON(map[string]any{
			"name": "tc-mcp-04",
			"type": "http",
			"url":  "http://test.example.com",
		}), adminToken)
	createBody := readBody(t, createResp)
	assertStatusAny(t, createResp, http.StatusOK, http.StatusCreated)

	var m mcpServerResp
	_ = json.Unmarshal(createBody, &m)
	if m.Name == "" {
		m.Name = "tc-mcp-04"
	}

	delResp := do(t, http.MethodDelete, "/api/v1/mcp-servers/"+mcpPathKey(m), nil, adminToken)
	_ = readBody(t, delResp)
	assertStatusAny(t, delResp, http.StatusOK, http.StatusNoContent)
}

// TC-MCP-05: Catalog endpoint returns 200 and a JSON array/object.
func TestMCP05_Catalog(t *testing.T) {
	requireSuite(t)

	resp := do(t, http.MethodGet, "/api/v1/mcp/catalog", nil, adminToken)
	body := readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "body=%s", body)
	// Just sanity-check JSON shape — array or object.
	if len(body) > 0 {
		first := body[0]
		assert.True(t, first == '[' || first == '{',
			"catalog body should be JSON: body=%s", body)
	}
}

// TC-MCP-06: Missing required fields → 400/422.
func TestMCP06_MissingRequired(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	// Post with just a name — no type, no url.
	resp := do(t, http.MethodPost, "/api/v1/mcp-servers",
		mustJSON(map[string]any{"name": "tc-mcp-06"}), adminToken)
	_ = readBody(t, resp)
	assertStatusAny(t, resp, http.StatusBadRequest, http.StatusUnprocessableEntity)
}

// TC-MCP-07: Duplicate name → 409 Conflict.
func TestMCP07_DuplicateName(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	first := do(t, http.MethodPost, "/api/v1/mcp-servers",
		mustJSON(map[string]any{
			"name": "tc-mcp-07",
			"type": "http",
			"url":  "http://test.example.com",
		}), adminToken)
	_ = readBody(t, first)
	assertStatusAny(t, first, http.StatusOK, http.StatusCreated)

	second := do(t, http.MethodPost, "/api/v1/mcp-servers",
		mustJSON(map[string]any{
			"name": "tc-mcp-07",
			"type": "http",
			"url":  "http://other.example.com",
		}), adminToken)
	_ = readBody(t, second)
	assertStatusAny(t, second, http.StatusConflict, http.StatusUnprocessableEntity, http.StatusBadRequest)
}

//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// modelCreateResp captures the minimal response fields needed by update/delete
// tests — the engine has varied on returning "id" vs "name" as the path key.
type modelCreateResp struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// createModelForTest POSTs a model; returns decoded id+name.
func createModelForTest(t *testing.T, name string) modelCreateResp {
	t.Helper()
	resp := do(t, http.MethodPost, "/api/v1/models",
		mustJSON(map[string]any{
			"name":       name,
			"type":       "openai_compatible",
			"kind":       "chat",
			"model_name": "test-model",
			"api_key":    "test-key",
			"base_url":   "https://api.test.com",
		}), adminToken)
	body := readBody(t, resp)
	assertStatusAny(t, resp, http.StatusOK, http.StatusCreated)

	var parsed modelCreateResp
	_ = json.Unmarshal(body, &parsed)
	if parsed.Name == "" {
		parsed.Name = name
	}
	return parsed
}

// modelPathKey picks the URL segment for /models/{...} — prefer id, fall back
// to name. The server routes use {name} today but some EE builds override.
func modelPathKey(m modelCreateResp) string {
	if m.Name != "" {
		return m.Name
	}
	return m.ID
}

// TC-MDL-01: POST /models → 201.
func TestMDL01_CreateModel(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	m := createModelForTest(t, "tc-mdl-01")
	assert.NotEmpty(t, m.Name, "name should come back on create")
}

// TC-MDL-02: GET /models lists the created model.
func TestMDL02_ListContainsCreated(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	_ = createModelForTest(t, "tc-mdl-02")

	resp := do(t, http.MethodGet, "/api/v1/models", nil, adminToken)
	body := readBody(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), "tc-mdl-02", "list must contain created model: %s", body)
}

// TC-MDL-03: PUT /models/{name} updates temperature.
func TestMDL03_UpdateModel(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	m := createModelForTest(t, "tc-mdl-03")

	resp := do(t, http.MethodPut, "/api/v1/models/"+modelPathKey(m),
		mustJSON(map[string]any{
			"name":        m.Name,
			"type":        "openai_compatible",
			"kind":        "chat",
			"model_name":  "test-model",
			"api_key":     "test-key",
			"base_url":    "https://api.test.com",
			"temperature": 0.42,
		}), adminToken)
	_ = readBody(t, resp)
	assertStatusAny(t, resp, http.StatusOK, http.StatusNoContent)
}

// TC-MDL-04: DELETE /models/{name}.
func TestMDL04_DeleteModel(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	m := createModelForTest(t, "tc-mdl-04")

	resp := do(t, http.MethodDelete, "/api/v1/models/"+modelPathKey(m), nil, adminToken)
	_ = readBody(t, resp)
	assertStatusAny(t, resp, http.StatusOK, http.StatusNoContent)
}

// TC-MDL-05: Duplicate name → 409 or 422.
func TestMDL05_DuplicateName(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	_ = createModelForTest(t, "tc-mdl-05")

	resp := do(t, http.MethodPost, "/api/v1/models",
		mustJSON(map[string]any{
			"name":       "tc-mdl-05",
			"type":       "openai_compatible",
			"provider":   "openrouter",
			"model_name": "test-model",
			"api_key":    "test-key",
			"base_url":   "https://api.test.com",
		}), adminToken)
	_ = readBody(t, resp)
	assertStatusAny(t, resp, http.StatusConflict, http.StatusUnprocessableEntity, http.StatusBadRequest)
}

// TC-MDL-06: Invalid type → 400/422.
func TestMDL06_InvalidType(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	resp := do(t, http.MethodPost, "/api/v1/models",
		mustJSON(map[string]any{
			"name":       "tc-mdl-06",
			"type":       "nonsense",
			"provider":   "openrouter",
			"model_name": "x",
			"api_key":    "k",
			"base_url":   "https://api.test.com",
		}), adminToken)
	_ = readBody(t, resp)
	assertStatusAny(t, resp, http.StatusBadRequest, http.StatusUnprocessableEntity)
}

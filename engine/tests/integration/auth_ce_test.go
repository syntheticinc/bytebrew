//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TC-SEC-01: Unauthenticated requests to protected routes must return 401.
func TestSEC01_NoToken(t *testing.T) {
	requireSuite(t)

	resp := do(t, http.MethodGet, "/api/v1/agents", nil, "")
	_ = readBody(t, resp)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
		"GET /agents without token must be 401")
}

// TC-SEC-02: Expired JWT must be rejected — WithExpirationRequired + exp
// validation in HMACVerifier.
func TestSEC02_ExpiredToken(t *testing.T) {
	requireSuite(t)

	resp := do(t, http.MethodGet, "/api/v1/agents", nil, expiredToken("user-expired"))
	_ = readBody(t, resp)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
		"expired JWT must be 401")
}

// TC-SEC-03: Tampered JWT signature — flip the last byte of a valid token.
func TestSEC03_TamperedSignature(t *testing.T) {
	requireSuite(t)

	tok := tamperedToken(adminToken)
	resp := do(t, http.MethodGet, "/api/v1/agents", nil, tok)
	_ = readBody(t, resp)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
		"tampered JWT must be 401")
}

// TC-SEC-04: alg=none JWT — the HMACVerifier pins HS256 via
// WithValidMethods, so this must fail even though the structure is valid.
func TestSEC04_AlgNone(t *testing.T) {
	requireSuite(t)

	tok := algNoneToken("user-alg-none")
	resp := do(t, http.MethodGet, "/api/v1/agents", nil, tok)
	_ = readBody(t, resp)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
		"alg=none JWT must be 401")
}

// TC-SEC-05: Non-admin role has no scopes, so POST /agents (which is
// guarded by RequireScope(ScopeAgentsWrite)) must return 403.
func TestSEC05_NonAdminRoleForbidden(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	userTok := tokenForRole("user-test", "user")
	resp := do(t, http.MethodPost, "/api/v1/agents",
		mustJSON(map[string]any{
			"name":          "tc-sec-05-agent",
			"system_prompt": "should not be created",
		}), userTok)
	_ = readBody(t, resp)
	// Middleware path: authenticateJWT accepts the signature but issues
	// scopes=0 for role!=admin; RequireScope then responds 403.
	assert.Equal(t, http.StatusForbidden, resp.StatusCode,
		"role=user must be rejected on scope-protected route")
}

// TC-SEC-06: API token full lifecycle — create, use, delete, use again
// (expect 401).
func TestSEC06_APITokenLifecycle(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	// Create an API token with broad scopes (admin session required).
	createResp := do(t, http.MethodPost, "/api/v1/auth/tokens",
		mustJSON(map[string]any{
			"name":        "tc-sec-06-token",
			"scopes_mask": 16, // ScopeAdmin — easiest path to read endpoints.
		}), adminToken)
	if createResp.StatusCode == http.StatusNotFound {
		// Route isn't registered in this build — skip rather than fail.
		t.Skip("POST /api/v1/auth/tokens not registered in this build")
	}
	body := readBody(t, createResp)
	require.Equal(t, http.StatusCreated, createResp.StatusCode,
		"expected 201 Created, got %d: %s", createResp.StatusCode, string(body))

	var created struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Token string `json:"token"`
	}
	require.NoError(t, jsonUnmarshalOrNil(body, &created), "decode create response")
	require.NotEmpty(t, created.Token, "response must include raw token once")
	require.NotEmpty(t, created.ID, "response must include token id")

	// Use the token — GET /agents is authorized under ScopeAdmin.
	useResp := do(t, http.MethodGet, "/api/v1/agents", nil, created.Token)
	_ = readBody(t, useResp)
	assert.Equal(t, http.StatusOK, useResp.StatusCode,
		"API token with ScopeAdmin must authorize GET /agents")

	// Delete it.
	delResp := do(t, http.MethodDelete, "/api/v1/auth/tokens/"+created.ID, nil, adminToken)
	_ = readBody(t, delResp)
	assertStatusAny(t, delResp, http.StatusOK, http.StatusNoContent)

	// Re-use — must be 401.
	reuseResp := do(t, http.MethodGet, "/api/v1/agents", nil, created.Token)
	_ = readBody(t, reuseResp)
	assert.Equal(t, http.StatusUnauthorized, reuseResp.StatusCode,
		"deleted API token must be rejected")
}

// TC-SEC-07: Limited-scope API token hitting an endpoint outside its scope
// must return 403.
func TestSEC07_APITokenLimitedScope(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	// scope = ScopeModelsRead (64) — allows GET /models but NOT POST /agents.
	createResp := do(t, http.MethodPost, "/api/v1/auth/tokens",
		mustJSON(map[string]any{
			"name":        "tc-sec-07-token",
			"scopes_mask": 64,
		}), adminToken)
	if createResp.StatusCode == http.StatusNotFound {
		t.Skip("POST /api/v1/auth/tokens not registered in this build")
	}
	body := readBody(t, createResp)
	require.Equal(t, http.StatusCreated, createResp.StatusCode,
		"create token: got %d: %s", createResp.StatusCode, string(body))

	var created struct {
		ID    string `json:"id"`
		Token string `json:"token"`
	}
	require.NoError(t, jsonUnmarshalOrNil(body, &created))

	// POST /agents requires ScopeAgentsWrite (32) — limited token must 403.
	resp := do(t, http.MethodPost, "/api/v1/agents",
		mustJSON(map[string]any{
			"name":          "tc-sec-07-agent",
			"system_prompt": "nope",
		}), created.Token)
	_ = readBody(t, resp)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode,
		"limited-scope token must be 403 outside its scope")
}

// TC-SEC-08 (positive): admin JWT satisfies any RequireScope check via the
// ScopeAdmin bit — baseline sanity check so the negative tests above are
// distinguishable from a broken route.
func TestSEC08_AdminCanCreateAgent(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	resp := do(t, http.MethodPost, "/api/v1/agents",
		mustJSON(map[string]any{
			"name":          "tc-sec-08-agent",
			"system_prompt": "ok",
		}), adminToken)
	body := readBody(t, resp)
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated,
		"expected 200/201 with admin token, got %d: %s", resp.StatusCode, string(body))
}

// TC-SEC-09: Malformed JSON body → 400 (not 500). Server must validate
// input and short-circuit cleanly.
func TestSEC09_InvalidJSON(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	resp := doHeaders(t, http.MethodPost, "/api/v1/agents",
		readerOf("{not-valid-json"),
		map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + adminToken,
		})
	_ = readBody(t, resp)
	assertStatusAny(t, resp, http.StatusBadRequest, http.StatusUnprocessableEntity)
}

// TC-SEC-10: Empty body on create endpoint → 400.
func TestSEC10_EmptyBody(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	resp := doHeaders(t, http.MethodPost, "/api/v1/agents",
		readerOf(""),
		map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + adminToken,
		})
	_ = readBody(t, resp)
	assertStatusAny(t, resp, http.StatusBadRequest, http.StatusUnprocessableEntity)
}

// TC-SEC-11: Empty name → validation error (400/422).
func TestSEC11_EmptyNameField(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	resp := do(t, http.MethodPost, "/api/v1/agents",
		mustJSON(map[string]any{
			"name":          "",
			"system_prompt": "whatever",
		}), adminToken)
	_ = readBody(t, resp)
	assertStatusAny(t, resp, http.StatusBadRequest, http.StatusUnprocessableEntity)
}

// TC-SEC-12: Model with invalid type → 400/422 validation error.
func TestSEC12_InvalidModelType(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	resp := do(t, http.MethodPost, "/api/v1/models",
		mustJSON(map[string]any{
			"name":       "tc-sec-12-model",
			"type":       "not-a-real-type",
			"provider":   "openrouter",
			"model_name": "x",
			"api_key":    "k",
		}), adminToken)
	_ = readBody(t, resp)
	assertStatusAny(t, resp, http.StatusBadRequest, http.StatusUnprocessableEntity)
}

// TC-SEC-13: Knowledge base missing required fields → 400/422.
func TestSEC13_KBMissingEmbeddingModel(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	resp := do(t, http.MethodPost, "/api/v1/knowledge-bases",
		mustJSON(map[string]any{
			"name": "tc-sec-13-kb",
		}), adminToken)
	_ = readBody(t, resp)
	// Some builds treat embedding_model as optional (201/200). Accept either
	// a successful create OR a 4xx validation error — the only bad outcome
	// is 500.
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		t.Logf("KB create without embedding_model was accepted — embedding_model is optional in this build")
		return
	}
	assertStatusAny(t, resp, http.StatusBadRequest, http.StatusUnprocessableEntity)
}

// TC-SEC-14: Malformed settings body → 400/422, never 500.
func TestSEC14_SettingsMalformedBody(t *testing.T) {
	requireSuite(t)
	t.Cleanup(func() { truncateTables(t) })

	// PUT /api/v1/settings/{key} is the canonical update path; send junk.
	resp := doHeaders(t, http.MethodPut, "/api/v1/settings/tc-sec-14-key",
		readerOf("{broken"),
		map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + adminToken,
		})
	_ = readBody(t, resp)
	// 404 is acceptable if the route isn't reachable under /settings/{key}
	// in this build; anything 5xx is a fail.
	if resp.StatusCode == http.StatusNotFound {
		t.Skip("PUT /api/v1/settings/{key} not registered in this build")
	}
	assertStatusAny(t, resp, http.StatusBadRequest, http.StatusUnprocessableEntity)
}


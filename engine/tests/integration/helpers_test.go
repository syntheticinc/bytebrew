//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

// waitForCondition polls check every 100ms until it returns true or timeout expires.
// Kept for non-CE tests in this package (production_harness, streaming, ws).
func waitForCondition(t *testing.T, timeout time.Duration, check func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("condition not met within %v", timeout)
}

// --- CE HTTP test helpers ---

// httpClient is shared across CE HTTP tests. Short timeout is fine — all
// endpoints are localhost.
var httpClient = &http.Client{Timeout: 20 * time.Second}

// requireSuite skips a test when TestMain bailed on suite setup (e.g. no
// Docker). This keeps `go test -tags integration ./...` green on machines
// that can't run the real stack.
func requireSuite(t *testing.T) {
	t.Helper()
	if r := skipReason(); r != "" {
		t.Skip(r)
	}
}

// tokenFor builds an HS256 JWT with role=admin (→ ScopeAdmin via HMACVerifier)
// that expires in 1h. This is the workhorse constructor for authenticated
// test calls.
func tokenFor(sub string) string {
	return tokenForRole(sub, "admin")
}

// tokenForRole builds an HS256 JWT with the given role claim. role != "admin"
// gets empty scopes, which the CE auth middleware rejects at RequireScope
// boundaries — used to exercise 403 paths.
func tokenForRole(sub, role string) string {
	claims := jwt.MapClaims{
		"sub":  sub,
		"role": role,
		"exp":  time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(jwtSecret))
	if err != nil {
		panic(fmt.Sprintf("tokenForRole: sign: %v", err))
	}
	return signed
}

// do performs an HTTP request against baseURL. token may be empty for
// unauthenticated calls; body may be nil.
func do(t *testing.T, method, path string, body io.Reader, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, baseURL+path, body)
	require.NoError(t, err, "build request")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := httpClient.Do(req)
	require.NoError(t, err, "http do %s %s", method, path)
	return resp
}

// doHeaders is the arbitrary-headers form of do — used when a test needs to
// override Content-Type, set Authorization manually, etc.
func doHeaders(t *testing.T, method, path string, body io.Reader, headers map[string]string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, baseURL+path, body)
	require.NoError(t, err, "build request")
	if body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := httpClient.Do(req)
	require.NoError(t, err, "http do %s %s", method, path)
	return resp
}

// mustJSON marshals v and returns an io.Reader wrapping the bytes.
func mustJSON(v any) io.Reader {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("mustJSON: %v", err))
	}
	return bytes.NewReader(data)
}

// mustJSONBytes is like mustJSON but returns the raw bytes — handy when the
// payload has to be reused (e.g. also signed for HMAC).
func mustJSONBytes(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("mustJSONBytes: %v", err))
	}
	return data
}

// assertStatus fails the test if the response status code doesn't match.
// Dumps the body on failure so the cause is visible.
func assertStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode == expected {
		_ = resp.Body.Close()
		return
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	t.Fatalf("unexpected status: got %d, want %d; body=%s", resp.StatusCode, expected, strings.TrimSpace(string(body)))
}

// assertStatusAny accepts any of the expected codes and returns the one that
// matched. Useful for endpoints that have drifted between 200 and 201 over
// time.
func assertStatusAny(t *testing.T, resp *http.Response, expected ...int) int {
	t.Helper()
	for _, code := range expected {
		if resp.StatusCode == code {
			return code
		}
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	t.Fatalf("unexpected status: got %d, want one of %v; body=%s", resp.StatusCode, expected, strings.TrimSpace(string(body)))
	return 0
}

// readBody drains + closes the response body.
func readBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "read body")
	return data
}

// tenantTables is the canonical list of tables CE tests reset between cases.
// Order matters when CASCADE isn't used — children first — but we pass
// CASCADE so the list order is advisory.
var tenantTables = []string{
	"agent_mcp_servers",
	"capabilities",
	"agent_tools",
	"agent_relations",
	"knowledge_base_agents",
	"knowledge_chunks",
	"knowledge_documents",
	"knowledge_bases",
	"sessions",
	"schemas",
	"agents",
	"mcp_servers",
	"settings",
	"llm_providers",
	"engine_tasks",
	"audit_logs",
	"tool_call_events",
}

// ensureTableName guards truncateTables against typos / rename drift and
// rejects obviously unsafe table names (quotes, semicolons).
func ensureTableName(name string) string {
	if strings.ContainsAny(name, "\"';") {
		panic("suspicious table name: " + name)
	}
	return name
}

// truncateTables resets all tenant-scoped state so tests don't leak into
// each other. CASCADE handles FKs, RESTART IDENTITY resets serial counters.
//
// Uses testDB (opened by the suite) rather than opening a fresh connection
// per call — the shared pool is fast enough and avoids leaking sockets.
func truncateTables(t *testing.T) {
	t.Helper()
	if testDB == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	names := make([]string, 0, len(tenantTables))
	for _, tbl := range tenantTables {
		names = append(names, `"`+ensureTableName(tbl)+`"`)
	}
	stmt := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", strings.Join(names, ", "))
	if err := testDB.WithContext(ctx).Exec(stmt).Error; err != nil {
		// Don't fail the test — missing tables (schema drift) should be
		// visible but not block the whole suite.
		t.Logf("truncateTables exec: %v", err)
	}
}

// tamperedToken returns a token where the last character of the signature
// has been flipped — the resulting JWT has a structurally valid shape but
// fails HMAC verification.
func tamperedToken(good string) string {
	if len(good) == 0 {
		return good
	}
	last := good[len(good)-1]
	var replacement byte = 'A'
	if last == 'A' {
		replacement = 'B'
	}
	return good[:len(good)-1] + string(replacement)
}

// algNoneToken builds a JWT with alg=none and an empty signature. A
// correctly-hardened verifier (WithValidMethods(["HS256"])) must reject this
// even though the token parses.
func algNoneToken(sub string) string {
	// Manually build header.payload. with empty signature.
	header := base64URLJSON(map[string]any{"alg": "none", "typ": "JWT"})
	payload := base64URLJSON(map[string]any{
		"sub":  sub,
		"role": "admin",
		"exp":  time.Now().Add(time.Hour).Unix(),
	})
	return header + "." + payload + "."
}

// expiredToken produces an HMAC-signed JWT whose exp claim is already in the
// past. WithExpirationRequired + exp validation must reject it.
func expiredToken(sub string) string {
	claims := jwt.MapClaims{
		"sub":  sub,
		"role": "admin",
		"exp":  time.Now().Add(-time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(jwtSecret))
	if err != nil {
		panic(fmt.Sprintf("expiredToken: %v", err))
	}
	return signed
}

// base64URLJSON marshals v to JSON, base64-url encodes it (no padding), and
// returns the resulting string. Used to hand-craft JWTs for negative tests.
func base64URLJSON(v any) string {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("base64URLJSON: marshal: %v", err))
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

// readerOf wraps a plain string in a bytes.Reader-style io.Reader so tests
// can send raw (possibly malformed) bodies without pulling strings.NewReader
// into every test file's imports.
func readerOf(s string) io.Reader {
	return bytes.NewReader([]byte(s))
}

// jsonUnmarshalOrNil unmarshals body into v. Returns nil on empty body so
// tests can call it unconditionally and still succeed when the server
// returned no payload (e.g. 204 No Content).
func jsonUnmarshalOrNil(body []byte, v any) error {
	if len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, v)
}

// writeFixture writes content to dir/name, creating a file for use in tool tests.
func writeFixture(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("writeFixture %s: %v", name, err)
	}
}

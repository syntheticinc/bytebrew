package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubMCPService is an MCPService that records calls; Create/Update return
// a zero response unless the test configures otherwise. Useful for verifying
// that a handler short-circuits before reaching the service layer.
type stubMCPService struct {
	createCalls []CreateMCPServerRequest
	updateCalls []CreateMCPServerRequest
}

func (s *stubMCPService) ListMCPServers(ctx context.Context) ([]MCPServerResponse, error) {
	return nil, nil
}

func (s *stubMCPService) CreateMCPServer(ctx context.Context, req CreateMCPServerRequest) (*MCPServerResponse, error) {
	s.createCalls = append(s.createCalls, req)
	return &MCPServerResponse{ID: "id-1", Name: req.Name, Type: req.Type}, nil
}

func (s *stubMCPService) UpdateMCPServer(ctx context.Context, name string, req CreateMCPServerRequest) (*MCPServerResponse, error) {
	s.updateCalls = append(s.updateCalls, req)
	return &MCPServerResponse{ID: "id-1", Name: name, Type: req.Type}, nil
}

func (s *stubMCPService) DeleteMCPServer(ctx context.Context, name string) error {
	return nil
}

// newMCPTestRouter returns a chi router that mounts only the MCPHandler
// routes — good for unit testing without the full server wiring.
func newMCPTestRouter(svc *stubMCPService) http.Handler {
	r := chi.NewRouter()
	r.Mount("/", NewMCPHandler(svc).Routes())
	return r
}

// TestMCPHandler_Create_CE_AllowsStdio verifies that stdio MCP servers are
// accepted in the default CE deployment mode (no BYTEBREW_MODE env var).
func TestMCPHandler_Create_CE_AllowsStdio(t *testing.T) {
	t.Setenv("BYTEBREW_MODE", "ce")

	svc := &stubMCPService{}
	router := newMCPTestRouter(svc)

	body, _ := json.Marshal(CreateMCPServerRequest{
		Name:    "test-stdio",
		Type:    "stdio",
		Command: "/bin/echo",
	})

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	require.Len(t, svc.createCalls, 1, "service should be called in CE mode")
	assert.Equal(t, "stdio", svc.createCalls[0].Type)
}

// TestMCPHandler_Create_Cloud_BlocksStdio verifies that stdio transport is
// rejected with 400 in Cloud deployment mode (the security gate that
// prevents arbitrary RCE on the Cloud-hosted engine host).
func TestMCPHandler_Create_Cloud_BlocksStdio(t *testing.T) {
	t.Setenv("BYTEBREW_MODE", "cloud")

	svc := &stubMCPService{}
	router := newMCPTestRouter(svc)

	body, _ := json.Marshal(CreateMCPServerRequest{
		Name:    "test-stdio-cloud",
		Type:    "stdio",
		Command: "/bin/sh",
		Args:    []string{"-c", "whoami"},
	})

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "stdio MCP transport is disabled in Cloud")
	assert.Len(t, svc.createCalls, 0, "service must not be called when transport is blocked")
}

// TestMCPHandler_Create_RejectsDocker verifies docker transport is rejected
// regardless of deployment mode. DBML mcp_servers.type does not include
// "docker" — the CHECK constraint would fail at INSERT time if we let it
// through, so the handler must reject it up front.
func TestMCPHandler_Create_RejectsDocker(t *testing.T) {
	t.Setenv("BYTEBREW_MODE", "ce")

	svc := &stubMCPService{}
	router := newMCPTestRouter(svc)

	body, _ := json.Marshal(CreateMCPServerRequest{
		Name:    "test-docker",
		Type:    "docker",
		Command: "image:tag",
	})

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid transport type")
	assert.Len(t, svc.createCalls, 0)
}

// TestMCPHandler_Create_Cloud_AllowsHTTP verifies HTTP/SSE transports still
// work in Cloud (they stay network-bound and don't spawn processes).
func TestMCPHandler_Create_Cloud_AllowsHTTP(t *testing.T) {
	t.Setenv("BYTEBREW_MODE", "cloud")

	svc := &stubMCPService{}
	router := newMCPTestRouter(svc)

	body, _ := json.Marshal(CreateMCPServerRequest{
		Name: "test-http-cloud",
		Type: "http",
		URL:  "https://example.com/mcp",
	})

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	require.Len(t, svc.createCalls, 1)
	assert.Equal(t, "http", svc.createCalls[0].Type)
}

// TestMCPHandler_Update_Cloud_BlocksStdio verifies the update path has the
// same guard as create.
func TestMCPHandler_Update_Cloud_BlocksStdio(t *testing.T) {
	t.Setenv("BYTEBREW_MODE", "cloud")

	svc := &stubMCPService{}
	router := newMCPTestRouter(svc)

	body, _ := json.Marshal(CreateMCPServerRequest{
		Name:    "existing",
		Type:    "stdio",
		Command: "/bin/sh",
	})

	req := httptest.NewRequest(http.MethodPut, "/existing", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "stdio MCP transport is disabled in Cloud")
	assert.Len(t, svc.updateCalls, 0)
}

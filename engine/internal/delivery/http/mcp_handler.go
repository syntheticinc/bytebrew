package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// MCPServerResponse is the API representation of an MCP server.
type MCPServerResponse struct {
	ID             uint              `json:"id"`
	Name           string            `json:"name"`
	Type           string            `json:"type"`
	Command        string            `json:"command,omitempty"`
	Args           []string          `json:"args,omitempty"`
	URL            string            `json:"url,omitempty"`
	EnvVars        map[string]string `json:"env_vars,omitempty"`
	ForwardHeaders []string          `json:"forward_headers,omitempty"`
	IsWellKnown    bool              `json:"is_well_known"`
	Status         *MCPStatusInfo    `json:"status,omitempty"`
	Agents         []string          `json:"agents"`
}

// MCPStatusInfo is the runtime status of an MCP server.
type MCPStatusInfo struct {
	Status        string `json:"status"`
	StatusMessage string `json:"status_message,omitempty"`
	ToolsCount    int    `json:"tools_count"`
	ConnectedAt   string `json:"connected_at,omitempty"`
}

// CreateMCPServerRequest is the body for POST /api/v1/mcp-servers.
type CreateMCPServerRequest struct {
	Name           string            `json:"name"`
	Type           string            `json:"type"`
	Command        string            `json:"command,omitempty"`
	Args           []string          `json:"args,omitempty"`
	URL            string            `json:"url,omitempty"`
	EnvVars        map[string]string `json:"env_vars,omitempty"`
	ForwardHeaders []string          `json:"forward_headers,omitempty"`
}

// MCPService provides MCP server CRUD operations.
type MCPService interface {
	ListMCPServers(ctx context.Context) ([]MCPServerResponse, error)
	CreateMCPServer(ctx context.Context, req CreateMCPServerRequest) (*MCPServerResponse, error)
	UpdateMCPServer(ctx context.Context, name string, req CreateMCPServerRequest) (*MCPServerResponse, error)
	DeleteMCPServer(ctx context.Context, name string) error
}

// MCPHandler serves /api/v1/mcp-servers endpoints.
type MCPHandler struct {
	service MCPService
}

// NewMCPHandler creates an MCPHandler.
func NewMCPHandler(service MCPService) *MCPHandler {
	return &MCPHandler{service: service}
}

// Routes returns a chi router with MCP server endpoints mounted.
func (h *MCPHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Put("/{name}", h.Update)
	r.Delete("/{name}", h.Delete)
	return r
}

// List handles GET /api/v1/mcp-servers.
func (h *MCPHandler) List(w http.ResponseWriter, r *http.Request) {
	servers, err := h.service.ListMCPServers(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, servers)
}

// Create handles POST /api/v1/mcp-servers.
func (h *MCPHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateMCPServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}
	if req.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Type == "" {
		writeJSONError(w, http.StatusBadRequest, "type is required")
		return
	}

	server, err := h.service.CreateMCPServer(r.Context(), req)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, server)
}

// Update handles PUT /api/v1/mcp-servers/{name}.
func (h *MCPHandler) Update(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "mcp server name is required")
		return
	}

	var req CreateMCPServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}

	result, err := h.service.UpdateMCPServer(r.Context(), name, req)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// Delete handles DELETE /api/v1/mcp-servers/{name}.
func (h *MCPHandler) Delete(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "mcp server name is required")
		return
	}

	if err := h.service.DeleteMCPServer(r.Context(), name); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

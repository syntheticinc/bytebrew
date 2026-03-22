package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// AgentInfo is a summary of an agent returned in list responses.
type AgentInfo struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	ToolsCount   int    `json:"tools_count"`
	Kit          string `json:"kit,omitempty"`
	HasKnowledge bool   `json:"has_knowledge"`
}

// AgentDetail is the full agent information returned by the detail endpoint.
type AgentDetail struct {
	AgentInfo
	ModelID        *uint            `json:"model_id,omitempty"`
	SystemPrompt   string           `json:"system_prompt"`
	KnowledgePath  string           `json:"knowledge_path,omitempty"`
	Tools          []string         `json:"tools"`
	CanSpawn       []string         `json:"can_spawn,omitempty"`
	Lifecycle      string           `json:"lifecycle"`
	ToolExecution  string           `json:"tool_execution"`
	MaxSteps       int              `json:"max_steps"`
	MaxContextSize int              `json:"max_context_size"`
	ConfirmBefore  []string         `json:"confirm_before,omitempty"`
	MCPServers     []string         `json:"mcp_servers,omitempty"`
	Escalation     *AgentEscalation `json:"escalation,omitempty"`
}

// AgentEscalation holds escalation settings in the API response.
type AgentEscalation struct {
	Action     string   `json:"action"`
	WebhookURL string   `json:"webhook_url,omitempty"`
	Triggers   []string `json:"triggers"`
}

// CreateAgentRequest is the body for POST /api/v1/agents.
type CreateAgentRequest struct {
	Name           string           `json:"name"`
	ModelID        *uint            `json:"model_id,omitempty"`
	SystemPrompt   string           `json:"system_prompt"`
	Kit            string           `json:"kit,omitempty"`
	KnowledgePath  string           `json:"knowledge_path,omitempty"`
	Lifecycle      string           `json:"lifecycle,omitempty"`
	ToolExecution  string           `json:"tool_execution,omitempty"`
	MaxSteps       int              `json:"max_steps,omitempty"`
	MaxContextSize int              `json:"max_context_size,omitempty"`
	ConfirmBefore  []string         `json:"confirm_before,omitempty"`
	Tools          []string         `json:"tools,omitempty"`
	CanSpawn       []string         `json:"can_spawn,omitempty"`
	MCPServers     []string         `json:"mcp_servers,omitempty"`
	Escalation     *AgentEscalation `json:"escalation,omitempty"`
}

// AgentLister provides agent listing and detail retrieval.
type AgentLister interface {
	ListAgents(ctx context.Context) ([]AgentInfo, error)
	GetAgent(ctx context.Context, name string) (*AgentDetail, error)
}

// AgentManager extends AgentLister with create, update, and delete operations.
type AgentManager interface {
	AgentLister
	CreateAgent(ctx context.Context, req CreateAgentRequest) (*AgentDetail, error)
	UpdateAgent(ctx context.Context, name string, req CreateAgentRequest) (*AgentDetail, error)
	DeleteAgent(ctx context.Context, name string) error
}

// AgentHandler serves /api/v1/agents endpoints.
type AgentHandler struct {
	lister  AgentLister
	manager AgentManager // may be nil if only read-only mode
}

// NewAgentHandler creates an AgentHandler (read-only).
func NewAgentHandler(lister AgentLister) *AgentHandler {
	return &AgentHandler{lister: lister}
}

// NewAgentHandlerWithManager creates an AgentHandler with full CRUD support.
func NewAgentHandlerWithManager(manager AgentManager) *AgentHandler {
	return &AgentHandler{lister: manager, manager: manager}
}

// Routes returns a chi router with agent endpoints mounted.
func (h *AgentHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{name}", h.Get)
	r.Put("/{name}", h.Update)
	r.Delete("/{name}", h.Delete)
	return r
}

// List handles GET /api/v1/agents.
func (h *AgentHandler) List(w http.ResponseWriter, r *http.Request) {
	agents, err := h.lister.ListAgents(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, agents)
}

// Get handles GET /api/v1/agents/{name}.
func (h *AgentHandler) Get(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "agent name is required")
		return
	}

	agent, err := h.lister.GetAgent(r.Context(), name)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if agent == nil {
		writeJSONError(w, http.StatusNotFound, "agent not found: "+name)
		return
	}

	writeJSON(w, http.StatusOK, agent)
}

// Create handles POST /api/v1/agents.
func (h *AgentHandler) Create(w http.ResponseWriter, r *http.Request) {
	if h.manager == nil {
		writeJSONError(w, http.StatusNotImplemented, "agent creation not supported")
		return
	}

	var req CreateAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}
	if req.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.SystemPrompt == "" {
		writeJSONError(w, http.StatusBadRequest, "system_prompt is required")
		return
	}

	agent, err := h.manager.CreateAgent(r.Context(), req)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, agent)
}

// Update handles PUT /api/v1/agents/{name}.
func (h *AgentHandler) Update(w http.ResponseWriter, r *http.Request) {
	if h.manager == nil {
		writeJSONError(w, http.StatusNotImplemented, "agent update not supported")
		return
	}

	name := chi.URLParam(r, "name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "agent name is required")
		return
	}

	var req CreateAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}

	// Ensure name from URL is used (body may omit it)
	if req.Name == "" {
		req.Name = name
	}

	agent, err := h.manager.UpdateAgent(r.Context(), name, req)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, agent)
}

// Delete handles DELETE /api/v1/agents/{name}.
func (h *AgentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if h.manager == nil {
		writeJSONError(w, http.StatusNotImplemented, "agent deletion not supported")
		return
	}

	name := chi.URLParam(r, "name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "agent name is required")
		return
	}

	if err := h.manager.DeleteAgent(r.Context(), name); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

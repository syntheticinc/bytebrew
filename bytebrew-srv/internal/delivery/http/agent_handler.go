package http

import (
	"context"
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
	Tools    []string `json:"tools"`
	CanSpawn []string `json:"can_spawn,omitempty"`
}

// AgentLister provides agent listing and detail retrieval.
type AgentLister interface {
	ListAgents(ctx context.Context) ([]AgentInfo, error)
	GetAgent(ctx context.Context, name string) (*AgentDetail, error)
}

// AgentHandler serves GET /api/v1/agents and GET /api/v1/agents/{name}.
type AgentHandler struct {
	lister AgentLister
}

// NewAgentHandler creates an AgentHandler.
func NewAgentHandler(lister AgentLister) *AgentHandler {
	return &AgentHandler{lister: lister}
}

// Routes returns a chi router with agent endpoints mounted.
func (h *AgentHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Get("/{name}", h.Get)
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


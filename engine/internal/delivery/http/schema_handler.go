package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// --- Schema DTOs ---

// SchemaInfo is a summary of a schema returned in list responses.
type SchemaInfo struct {
	ID          uint     `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Agents      []string `json:"agents,omitempty"`
}

// CreateSchemaRequest is the body for POST /api/v1/schemas.
type CreateSchemaRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// UpdateSchemaRequest is the body for PUT /api/v1/schemas/{id}.
type UpdateSchemaRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// AddSchemaAgentRequest is the body for POST /api/v1/schemas/{id}/agents.
type AddSchemaAgentRequest struct {
	AgentName string `json:"agent_name"`
}

// --- Gate DTOs ---

// GateInfo is a gate returned in API responses.
type GateInfo struct {
	ID            uint                   `json:"id"`
	SchemaID      uint                   `json:"schema_id"`
	Name          string                 `json:"name"`
	ConditionType string                 `json:"condition_type"`
	Config        map[string]interface{} `json:"config,omitempty"`
	MaxIterations int                    `json:"max_iterations"`
	Timeout       int                    `json:"timeout"`
}

// CreateGateRequest is the body for POST /api/v1/schemas/{id}/gates.
type CreateGateRequest struct {
	Name          string                 `json:"name"`
	ConditionType string                 `json:"condition_type"`
	Config        map[string]interface{} `json:"config,omitempty"`
	MaxIterations int                    `json:"max_iterations,omitempty"`
	Timeout       int                    `json:"timeout,omitempty"`
}

// --- Edge DTOs ---

// EdgeInfo is an edge returned in API responses.
type EdgeInfo struct {
	ID              uint                   `json:"id"`
	SchemaID        uint                   `json:"schema_id"`
	SourceAgentName string                 `json:"source"`
	TargetAgentName string                 `json:"target"`
	Type            string                 `json:"type"`
	Config          map[string]interface{} `json:"config,omitempty"`
}

// CreateEdgeRequest is the body for POST /api/v1/schemas/{id}/edges.
type CreateEdgeRequest struct {
	Source string                 `json:"source"`
	Target string                 `json:"target"`
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// --- Service interfaces (consumer-side) ---

// SchemaService provides schema CRUD operations.
type SchemaService interface {
	ListSchemas(ctx context.Context) ([]SchemaInfo, error)
	GetSchema(ctx context.Context, id uint) (*SchemaInfo, error)
	CreateSchema(ctx context.Context, req CreateSchemaRequest) (*SchemaInfo, error)
	UpdateSchema(ctx context.Context, id uint, req UpdateSchemaRequest) error
	DeleteSchema(ctx context.Context, id uint) error
	AddSchemaAgent(ctx context.Context, schemaID uint, agentName string) error
	RemoveSchemaAgent(ctx context.Context, schemaID uint, agentName string) error
	ListSchemaAgents(ctx context.Context, schemaID uint) ([]string, error)
}

// GateService provides gate CRUD operations.
type GateService interface {
	ListGates(ctx context.Context, schemaID uint) ([]GateInfo, error)
	GetGate(ctx context.Context, id uint) (*GateInfo, error)
	CreateGate(ctx context.Context, schemaID uint, req CreateGateRequest) (*GateInfo, error)
	UpdateGate(ctx context.Context, id uint, req CreateGateRequest) error
	DeleteGate(ctx context.Context, id uint) error
}

// EdgeService provides edge CRUD operations.
type EdgeService interface {
	ListEdges(ctx context.Context, schemaID uint) ([]EdgeInfo, error)
	GetEdge(ctx context.Context, id uint) (*EdgeInfo, error)
	CreateEdge(ctx context.Context, schemaID uint, req CreateEdgeRequest) (*EdgeInfo, error)
	UpdateEdge(ctx context.Context, id uint, req CreateEdgeRequest) error
	DeleteEdge(ctx context.Context, id uint) error
}

// AgentSchemaLister provides the ability to list schemas that reference an agent.
type AgentSchemaLister interface {
	ListSchemasForAgent(ctx context.Context, agentName string) ([]string, error)
}

// --- Handler ---

// SchemaHandler serves /api/v1/schemas endpoints.
type SchemaHandler struct {
	schemas        SchemaService
	gates          GateService
	edges          EdgeService
	agentDetailer  SchemaAgentDetailer // optional, used by export
}

// NewSchemaHandler creates a SchemaHandler.
func NewSchemaHandler(schemas SchemaService, gates GateService, edges EdgeService) *SchemaHandler {
	return &SchemaHandler{schemas: schemas, gates: gates, edges: edges}
}

// Routes returns a chi router with all schema, gate, and edge endpoints.
func (h *SchemaHandler) Routes() http.Handler {
	r := chi.NewRouter()

	// Schema CRUD
	r.Get("/", h.ListSchemas)
	r.Post("/", h.CreateSchema)
	r.Get("/{id}", h.GetSchema)
	r.Put("/{id}", h.UpdateSchema)
	r.Delete("/{id}", h.DeleteSchema)

	// Schema-Agent refs
	r.Get("/{id}/agents", h.ListSchemaAgents)
	r.Post("/{id}/agents", h.AddSchemaAgent)
	r.Delete("/{id}/agents/{name}", h.RemoveSchemaAgent)

	// Gates (per-schema)
	r.Get("/{id}/gates", h.ListGates)
	r.Post("/{id}/gates", h.CreateGate)
	r.Get("/{id}/gates/{gateId}", h.GetGate)
	r.Put("/{id}/gates/{gateId}", h.UpdateGate)
	r.Delete("/{id}/gates/{gateId}", h.DeleteGate)

	// Edges (per-schema)
	r.Get("/{id}/edges", h.ListEdges)
	r.Post("/{id}/edges", h.CreateEdge)
	r.Get("/{id}/edges/{edgeId}", h.GetEdge)
	r.Put("/{id}/edges/{edgeId}", h.UpdateEdge)
	r.Delete("/{id}/edges/{edgeId}", h.DeleteEdge)

	return r
}

// --- Schema endpoints ---

func (h *SchemaHandler) ListSchemas(w http.ResponseWriter, r *http.Request) {
	schemas, err := h.schemas.ListSchemas(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, schemas)
}

func (h *SchemaHandler) GetSchema(w http.ResponseWriter, r *http.Request) {
	id, err := parseUintParam(r, "id")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	schema, err := h.schemas.GetSchema(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, schema)
}

func (h *SchemaHandler) CreateSchema(w http.ResponseWriter, r *http.Request) {
	var req CreateSchemaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}
	if req.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}

	schema, err := h.schemas.CreateSchema(r.Context(), req)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, schema)
}

func (h *SchemaHandler) UpdateSchema(w http.ResponseWriter, r *http.Request) {
	id, err := parseUintParam(r, "id")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req UpdateSchemaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}

	if err := h.schemas.UpdateSchema(r.Context(), id, req); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SchemaHandler) DeleteSchema(w http.ResponseWriter, r *http.Request) {
	id, err := parseUintParam(r, "id")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.schemas.DeleteSchema(r.Context(), id); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Schema-Agent ref endpoints ---

func (h *SchemaHandler) ListSchemaAgents(w http.ResponseWriter, r *http.Request) {
	id, err := parseUintParam(r, "id")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	agents, err := h.schemas.ListSchemaAgents(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, agents)
}

func (h *SchemaHandler) AddSchemaAgent(w http.ResponseWriter, r *http.Request) {
	id, err := parseUintParam(r, "id")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req AddSchemaAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}
	if req.AgentName == "" {
		writeJSONError(w, http.StatusBadRequest, "agent_name is required")
		return
	}

	if err := h.schemas.AddSchemaAgent(r.Context(), id, req.AgentName); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SchemaHandler) RemoveSchemaAgent(w http.ResponseWriter, r *http.Request) {
	id, err := parseUintParam(r, "id")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	name := chi.URLParam(r, "name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "agent name is required")
		return
	}

	if err := h.schemas.RemoveSchemaAgent(r.Context(), id, name); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Gate endpoints ---

func (h *SchemaHandler) ListGates(w http.ResponseWriter, r *http.Request) {
	schemaID, err := parseUintParam(r, "id")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	gates, err := h.gates.ListGates(r.Context(), schemaID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, gates)
}

func (h *SchemaHandler) GetGate(w http.ResponseWriter, r *http.Request) {
	gateID, err := parseUintParam(r, "gateId")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	gate, err := h.gates.GetGate(r.Context(), gateID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, gate)
}

func (h *SchemaHandler) CreateGate(w http.ResponseWriter, r *http.Request) {
	schemaID, err := parseUintParam(r, "id")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req CreateGateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}
	if req.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}

	gate, err := h.gates.CreateGate(r.Context(), schemaID, req)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, gate)
}

func (h *SchemaHandler) UpdateGate(w http.ResponseWriter, r *http.Request) {
	gateID, err := parseUintParam(r, "gateId")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req CreateGateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}

	if err := h.gates.UpdateGate(r.Context(), gateID, req); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SchemaHandler) DeleteGate(w http.ResponseWriter, r *http.Request) {
	gateID, err := parseUintParam(r, "gateId")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.gates.DeleteGate(r.Context(), gateID); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Edge endpoints ---

func (h *SchemaHandler) ListEdges(w http.ResponseWriter, r *http.Request) {
	schemaID, err := parseUintParam(r, "id")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	edges, err := h.edges.ListEdges(r.Context(), schemaID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, edges)
}

func (h *SchemaHandler) GetEdge(w http.ResponseWriter, r *http.Request) {
	edgeID, err := parseUintParam(r, "edgeId")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	edge, err := h.edges.GetEdge(r.Context(), edgeID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, edge)
}

func (h *SchemaHandler) CreateEdge(w http.ResponseWriter, r *http.Request) {
	schemaID, err := parseUintParam(r, "id")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req CreateEdgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}
	if req.Source == "" || req.Target == "" {
		writeJSONError(w, http.StatusBadRequest, "source and target are required")
		return
	}

	edge, err := h.edges.CreateEdge(r.Context(), schemaID, req)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, edge)
}

func (h *SchemaHandler) UpdateEdge(w http.ResponseWriter, r *http.Request) {
	edgeID, err := parseUintParam(r, "edgeId")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req CreateEdgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}

	if err := h.edges.UpdateEdge(r.Context(), edgeID, req); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SchemaHandler) DeleteEdge(w http.ResponseWriter, r *http.Request) {
	edgeID, err := parseUintParam(r, "edgeId")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.edges.DeleteEdge(r.Context(), edgeID); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Helpers ---

func parseUintParam(r *http.Request, param string) (uint, error) {
	s := chi.URLParam(r, param)
	if s == "" {
		return 0, fmt.Errorf("%s is required", param)
	}
	val, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %s", param, s)
	}
	return uint(val), nil
}

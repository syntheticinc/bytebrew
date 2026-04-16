package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// --- Schema DTOs ---

// SchemaInfo is a summary of a schema returned in list responses.
type SchemaInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Agents      []string  `json:"agents,omitempty"`
	IsSystem    bool      `json:"is_system,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
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

// --- AgentRelation DTOs ---

// AgentRelationInfo is an agent_relation returned in API responses.
//
// V2 has a single implicit DELEGATION relationship type (see
// docs/architecture/agent-first-runtime.md §3.1). Optional Config carries
// non-typing routing hints.
// AgentRelationInfo is an agent_relation returned in API responses.
// Q.5: source/target are now agent UUIDs internally but the JSON keys
// remain "source"/"target" for API backward compatibility.
type AgentRelationInfo struct {
	ID            string                 `json:"id"`
	SchemaID      string                 `json:"schema_id"`
	SourceAgentID string                 `json:"source"`
	TargetAgentID string                 `json:"target"`
	Config        map[string]interface{} `json:"config,omitempty"`
}

// CreateAgentRelationRequest is the body for POST /api/v1/schemas/{id}/agent-relations.
type CreateAgentRelationRequest struct {
	Source string                 `json:"source"`
	Target string                 `json:"target"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// --- Service interfaces (consumer-side) ---

// SchemaService provides schema CRUD operations.
//
// V2: schema membership is derived from `agent_relations` (see
// docs/architecture/agent-first-runtime.md §2.1) — there is no separate
// AddSchemaAgent / RemoveSchemaAgent surface. Adding an agent to a schema
// is done by creating a delegation relation through AgentRelationService.
type SchemaService interface {
	ListSchemas(ctx context.Context) ([]SchemaInfo, error)
	GetSchema(ctx context.Context, id string) (*SchemaInfo, error)
	CreateSchema(ctx context.Context, req CreateSchemaRequest) (*SchemaInfo, error)
	UpdateSchema(ctx context.Context, id string, req UpdateSchemaRequest) error
	DeleteSchema(ctx context.Context, id string) error
	ListSchemaAgents(ctx context.Context, schemaID string) ([]string, error)
}

// AgentRelationService provides agent-relation CRUD operations.
type AgentRelationService interface {
	ListAgentRelations(ctx context.Context, schemaID string) ([]AgentRelationInfo, error)
	GetAgentRelation(ctx context.Context, id string) (*AgentRelationInfo, error)
	CreateAgentRelation(ctx context.Context, schemaID string, req CreateAgentRelationRequest) (*AgentRelationInfo, error)
	UpdateAgentRelation(ctx context.Context, id string, req CreateAgentRelationRequest) error
	DeleteAgentRelation(ctx context.Context, id string) error
}

// AgentSchemaLister provides the ability to list schemas that reference an agent.
type AgentSchemaLister interface {
	ListSchemasForAgent(ctx context.Context, agentName string) ([]string, error)
}

// --- Handler ---

// SchemaHandler serves /api/v1/schemas endpoints.
type SchemaHandler struct {
	schemas        SchemaService
	agentRelations AgentRelationService
	agentDetailer  SchemaAgentDetailer // optional, used by export
}

// NewSchemaHandler creates a SchemaHandler.
func NewSchemaHandler(schemas SchemaService, agentRelations AgentRelationService) *SchemaHandler {
	return &SchemaHandler{schemas: schemas, agentRelations: agentRelations}
}

// Routes returns a chi router with all schema and agent-relation endpoints.
func (h *SchemaHandler) Routes() http.Handler {
	r := chi.NewRouter()

	// Schema CRUD
	r.Get("/", h.ListSchemas)
	r.Post("/", h.CreateSchema)
	r.Get("/{id}", h.GetSchema)
	r.Put("/{id}", h.UpdateSchema)
	r.Delete("/{id}", h.DeleteSchema)

	// Schema-Agent membership (read-only — derived from agent_relations).
	// Mutation is done via the agent-relations endpoints below
	// (docs/architecture/agent-first-runtime.md §2.1).
	r.Get("/{id}/agents", h.ListSchemaAgents)

	// Agent relations (per-schema)
	r.Get("/{id}/agent-relations", h.ListAgentRelations)
	r.Post("/{id}/agent-relations", h.CreateAgentRelation)
	r.Get("/{id}/agent-relations/{relationId}", h.GetAgentRelation)
	r.Put("/{id}/agent-relations/{relationId}", h.UpdateAgentRelation)
	r.Delete("/{id}/agent-relations/{relationId}", h.DeleteAgentRelation)

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
	id, err := parseStringParam(r, "id")
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
	id, err := parseStringParam(r, "id")
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
	id, err := parseStringParam(r, "id")
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
	id, err := parseStringParam(r, "id")
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

// --- AgentRelation endpoints ---

func (h *SchemaHandler) ListAgentRelations(w http.ResponseWriter, r *http.Request) {
	schemaID, err := parseStringParam(r, "id")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	rels, err := h.agentRelations.ListAgentRelations(r.Context(), schemaID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rels)
}

func (h *SchemaHandler) GetAgentRelation(w http.ResponseWriter, r *http.Request) {
	relationID, err := parseStringParam(r, "relationId")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	rel, err := h.agentRelations.GetAgentRelation(r.Context(), relationID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rel)
}

func (h *SchemaHandler) CreateAgentRelation(w http.ResponseWriter, r *http.Request) {
	schemaID, err := parseStringParam(r, "id")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req CreateAgentRelationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}
	if req.Source == "" || req.Target == "" {
		writeJSONError(w, http.StatusBadRequest, "source and target are required")
		return
	}

	rel, err := h.agentRelations.CreateAgentRelation(r.Context(), schemaID, req)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, rel)
}

func (h *SchemaHandler) UpdateAgentRelation(w http.ResponseWriter, r *http.Request) {
	relationID, err := parseStringParam(r, "relationId")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req CreateAgentRelationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}

	if err := h.agentRelations.UpdateAgentRelation(r.Context(), relationID, req); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SchemaHandler) DeleteAgentRelation(w http.ResponseWriter, r *http.Request) {
	relationID, err := parseStringParam(r, "relationId")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.agentRelations.DeleteAgentRelation(r.Context(), relationID); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Helpers ---
// parseStringParam and parseStringIDParam are defined in task_handler.go

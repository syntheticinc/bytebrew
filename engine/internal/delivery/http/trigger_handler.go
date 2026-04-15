package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// TriggerResponse is the API representation of a trigger.
//
// V2: the flat `schedule` / `webhook_path` fields collapsed into `Config`
// and the `on_complete_url` / `on_complete_headers` webhook feature is gone.
// See docs/architecture/agent-first-runtime.md §4.1 / §4.2.
type TriggerResponse struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	AgentID     string                 `json:"agent_id"`
	AgentName   string                 `json:"agent_name,omitempty"`
	SchemaID    *string                `json:"schema_id,omitempty"`
	Description string                 `json:"description,omitempty"`
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config,omitempty"`
	LastFiredAt string                 `json:"last_fired_at,omitempty"`
	CreatedAt   string                 `json:"created_at"`
}

// CreateTriggerRequest is the body for POST /api/v1/triggers.
type CreateTriggerRequest struct {
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	AgentID     string                 `json:"agent_id"`
	AgentName   string                 `json:"agent_name,omitempty"`
	SchemaID    *string                `json:"schema_id,omitempty"`
	Description string                 `json:"description,omitempty"`
	Enabled     *bool                  `json:"enabled,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// TriggerService provides trigger CRUD operations.
type TriggerService interface {
	ListTriggers(ctx context.Context) ([]TriggerResponse, error)
	ListTriggersBySchema(ctx context.Context, schemaID string) ([]TriggerResponse, error)
	CreateTrigger(ctx context.Context, req CreateTriggerRequest) (*TriggerResponse, error)
	UpdateTrigger(ctx context.Context, id string, req CreateTriggerRequest) (*TriggerResponse, error)
	DeleteTrigger(ctx context.Context, id string) error
	SetTriggerTarget(ctx context.Context, id string, agentName string) (*TriggerResponse, error)
	ClearTriggerTarget(ctx context.Context, id string) error
}

// TriggerHandler serves /api/v1/triggers endpoints.
type TriggerHandler struct {
	service TriggerService
}

// NewTriggerHandler creates a TriggerHandler.
func NewTriggerHandler(service TriggerService) *TriggerHandler {
	return &TriggerHandler{service: service}
}

// Routes returns a chi router with trigger endpoints mounted.
func (h *TriggerHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Patch("/{id}/target", h.SetTarget)
	r.Delete("/{id}/target", h.ClearTarget)
	return r
}

// List handles GET /api/v1/triggers.
// Optional query param: ?schema_id=UUID to filter by schema.
func (h *TriggerHandler) List(w http.ResponseWriter, r *http.Request) {
	if schemaID := r.URL.Query().Get("schema_id"); schemaID != "" {
		triggers, err := h.service.ListTriggersBySchema(r.Context(), schemaID)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, triggers)
		return
	}

	triggers, err := h.service.ListTriggers(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, triggers)
}

// Create handles POST /api/v1/triggers.
func (h *TriggerHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateTriggerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}
	if req.Type == "" {
		writeJSONError(w, http.StatusBadRequest, "type is required")
		return
	}
	if req.Title == "" {
		writeJSONError(w, http.StatusBadRequest, "title is required")
		return
	}
	trigger, err := h.service.CreateTrigger(r.Context(), req)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, trigger)
}

// Update handles PUT /api/v1/triggers/{id}.
func (h *TriggerHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseStringIDParam(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req CreateTriggerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}

	result, err := h.service.UpdateTrigger(r.Context(), id, req)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// Delete handles DELETE /api/v1/triggers/{id}.
func (h *TriggerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseStringIDParam(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.DeleteTrigger(r.Context(), id); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SetTarget handles PATCH /api/v1/triggers/{id}/target.
// Connects a trigger to an agent — enables canvas-driven routing.
func (h *TriggerHandler) SetTarget(w http.ResponseWriter, r *http.Request) {
	id, err := parseStringIDParam(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	var body struct {
		AgentName string `json:"agent_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}
	if body.AgentName == "" {
		writeJSONError(w, http.StatusBadRequest, "agent_name is required")
		return
	}

	result, err := h.service.SetTriggerTarget(r.Context(), id, body.AgentName)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// ClearTarget handles DELETE /api/v1/triggers/{id}/target.
// Disconnects a trigger from its agent — disables routing without deleting the trigger.
func (h *TriggerHandler) ClearTarget(w http.ResponseWriter, r *http.Request) {
	id, err := parseStringIDParam(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.ClearTriggerTarget(r.Context(), id); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

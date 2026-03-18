package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// TriggerResponse is the API representation of a trigger.
type TriggerResponse struct {
	ID          uint   `json:"id"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	AgentID     uint   `json:"agent_id"`
	AgentName   string `json:"agent_name,omitempty"`
	Schedule    string `json:"schedule,omitempty"`
	WebhookPath string `json:"webhook_path,omitempty"`
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled"`
	LastFiredAt string `json:"last_fired_at,omitempty"`
	CreatedAt   string `json:"created_at"`
}

// CreateTriggerRequest is the body for POST /api/v1/triggers.
type CreateTriggerRequest struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	AgentID     uint   `json:"agent_id"`
	Schedule    string `json:"schedule,omitempty"`
	WebhookPath string `json:"webhook_path,omitempty"`
	Description string `json:"description,omitempty"`
	Enabled     *bool  `json:"enabled,omitempty"`
}

// TriggerService provides trigger CRUD operations.
type TriggerService interface {
	ListTriggers(ctx context.Context) ([]TriggerResponse, error)
	CreateTrigger(ctx context.Context, req CreateTriggerRequest) (*TriggerResponse, error)
	UpdateTrigger(ctx context.Context, id uint, req CreateTriggerRequest) (*TriggerResponse, error)
	DeleteTrigger(ctx context.Context, id uint) error
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
	return r
}

// List handles GET /api/v1/triggers.
func (h *TriggerHandler) List(w http.ResponseWriter, r *http.Request) {
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
	if req.AgentID == 0 {
		writeJSONError(w, http.StatusBadRequest, "agent_id is required")
		return
	}

	trigger, err := h.service.CreateTrigger(r.Context(), req)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, trigger)
}

// Update handles PUT /api/v1/triggers/{id}.
func (h *TriggerHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r)
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
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// Delete handles DELETE /api/v1/triggers/{id}.
func (h *TriggerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.DeleteTrigger(r.Context(), id); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

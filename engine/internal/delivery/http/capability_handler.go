package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// CapabilityInfo is a capability returned in API responses.
type CapabilityInfo struct {
	ID      string                 `json:"id"`
	Type    string                 `json:"type"`
	Config  map[string]interface{} `json:"config,omitempty"`
	Enabled bool                   `json:"enabled"`
}

// CreateCapabilityRequest is the body for POST /api/v1/agents/{name}/capabilities.
type CreateCapabilityRequest struct {
	Type    string                 `json:"type"`
	Config  map[string]interface{} `json:"config,omitempty"`
	Enabled *bool                  `json:"enabled,omitempty"` // pointer to distinguish absent from false
}

// UpdateCapabilityRequest is the body for PUT /api/v1/agents/{name}/capabilities/{id}.
type UpdateCapabilityRequest struct {
	Type    string                 `json:"type,omitempty"`
	Config  map[string]interface{} `json:"config,omitempty"`
	Enabled *bool                  `json:"enabled,omitempty"`
}

// CapabilityService provides capability CRUD for an agent.
type CapabilityService interface {
	ListCapabilities(ctx context.Context, agentName string) ([]CapabilityInfo, error)
	AddCapability(ctx context.Context, agentName string, req CreateCapabilityRequest) (*CapabilityInfo, error)
	UpdateCapability(ctx context.Context, id string, req UpdateCapabilityRequest) error
	RemoveCapability(ctx context.Context, id string) error
}

// CapabilityHandler serves /api/v1/agents/{name}/capabilities endpoints.
type CapabilityHandler struct {
	service CapabilityService
}

// NewCapabilityHandler creates a CapabilityHandler.
func NewCapabilityHandler(service CapabilityService) *CapabilityHandler {
	return &CapabilityHandler{service: service}
}

// List handles GET /api/v1/agents/{name}/capabilities.
func (h *CapabilityHandler) List(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "agent name is required")
		return
	}

	caps, err := h.service.ListCapabilities(r.Context(), name)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, caps)
}

// Add handles POST /api/v1/agents/{name}/capabilities.
func (h *CapabilityHandler) Add(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "agent name is required")
		return
	}

	var req CreateCapabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}
	if req.Type == "" {
		writeJSONError(w, http.StatusBadRequest, "type is required")
		return
	}
	// BUG-001: Validate capability type against allowed list.
	validTypes := map[string]bool{
		"memory": true, "knowledge": true, "escalation": true,
		"guardrail": true, "output_schema": true, "recovery": true, "policies": true,
	}
	if !validTypes[req.Type] {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid capability type %q: must be one of memory, knowledge, escalation, guardrail, output_schema, recovery, policies", req.Type))
		return
	}

	cap, err := h.service.AddCapability(r.Context(), name, req)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, cap)
}

// Update handles PUT /api/v1/agents/{name}/capabilities/{id}.
func (h *CapabilityHandler) Update(w http.ResponseWriter, r *http.Request) {
	capID, err := parseStringParam(r, "capId")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req UpdateCapabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}

	if err := h.service.UpdateCapability(r.Context(), capID, req); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Remove handles DELETE /api/v1/agents/{name}/capabilities/{id}.
func (h *CapabilityHandler) Remove(w http.ResponseWriter, r *http.Request) {
	capID, err := parseStringParam(r, "capId")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.RemoveCapability(r.Context(), capID); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

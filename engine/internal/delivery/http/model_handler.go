package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// ModelResponse is the API representation of an LLM provider model.
type ModelResponse struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	BaseURL   string `json:"base_url,omitempty"`
	ModelName string `json:"model_name"`
	HasAPIKey bool   `json:"has_api_key"`
	CreatedAt string `json:"created_at"`
}

// CreateModelRequest is the body for POST /api/v1/models.
type CreateModelRequest struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	BaseURL   string `json:"base_url,omitempty"`
	ModelName string `json:"model_name"`
	APIKey    string `json:"api_key,omitempty"`
}

// ModelVerifyResult contains the result of model connectivity verification.
type ModelVerifyResult struct {
	Connectivity   string  `json:"connectivity"`
	ToolCalling    string  `json:"tool_calling"`
	ResponseTimeMs int64   `json:"response_time_ms"`
	ModelName      string  `json:"model_name"`
	Provider       string  `json:"provider"`
	Error          *string `json:"error"`
}

// ModelService provides LLM model CRUD operations.
type ModelService interface {
	ListModels(ctx context.Context) ([]ModelResponse, error)
	CreateModel(ctx context.Context, req CreateModelRequest) (*ModelResponse, error)
	UpdateModel(ctx context.Context, name string, req CreateModelRequest) (*ModelResponse, error)
	DeleteModel(ctx context.Context, name string) error
	VerifyModel(ctx context.Context, name string) (*ModelVerifyResult, error)
}

// ModelHandler serves /api/v1/models endpoints.
type ModelHandler struct {
	service ModelService
}

// NewModelHandler creates a ModelHandler.
func NewModelHandler(service ModelService) *ModelHandler {
	return &ModelHandler{service: service}
}

// Routes returns a chi router with model endpoints mounted.
func (h *ModelHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Put("/{name}", h.Update)
	r.Delete("/{name}", h.Delete)
	r.Post("/{name}/verify", h.Verify)
	return r
}

// List handles GET /api/v1/models.
func (h *ModelHandler) List(w http.ResponseWriter, r *http.Request) {
	models, err := h.service.ListModels(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models)
}

// Create handles POST /api/v1/models.
func (h *ModelHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateModelRequest
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
	if req.ModelName == "" {
		writeJSONError(w, http.StatusBadRequest, "model_name is required")
		return
	}

	model, err := h.service.CreateModel(r.Context(), req)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, model)
}

// Update handles PUT /api/v1/models/{name}.
func (h *ModelHandler) Update(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "model name is required")
		return
	}

	var req CreateModelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}

	result, err := h.service.UpdateModel(r.Context(), name, req)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// Delete handles DELETE /api/v1/models/{name}.
func (h *ModelHandler) Delete(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "model name is required")
		return
	}

	if err := h.service.DeleteModel(r.Context(), name); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Verify handles POST /api/v1/models/{name}/verify.
func (h *ModelHandler) Verify(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "model name is required")
		return
	}

	result, err := h.service.VerifyModel(r.Context(), name)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

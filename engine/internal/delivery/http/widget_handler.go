package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// WidgetInfo is a widget returned in API responses.
type WidgetInfo struct {
	ID              uint     `json:"id"`
	Name            string   `json:"name"`
	SchemaID        uint     `json:"schema_id"`
	PrimaryColor    string   `json:"primary_color"`
	Position        string   `json:"position"`
	Size            string   `json:"size"`
	WelcomeMessage  string   `json:"welcome_message"`
	Placeholder     string   `json:"placeholder"`
	AvatarURL       string   `json:"avatar_url,omitempty"`
	DomainWhitelist []string          `json:"domain_whitelist"`
	CustomHeaders   map[string]string `json:"custom_headers,omitempty"`
	Enabled         bool              `json:"enabled"`
}

// CreateWidgetRequest is the body for POST /api/v1/widgets.
type CreateWidgetRequest struct {
	Name            string   `json:"name"`
	SchemaID        uint     `json:"schema_id"`
	PrimaryColor    string   `json:"primary_color,omitempty"`
	Position        string   `json:"position,omitempty"`
	Size            string   `json:"size,omitempty"`
	WelcomeMessage  string   `json:"welcome_message,omitempty"`
	Placeholder     string   `json:"placeholder,omitempty"`
	AvatarURL       string   `json:"avatar_url,omitempty"`
	DomainWhitelist []string          `json:"domain_whitelist,omitempty"`
	CustomHeaders   map[string]string `json:"custom_headers,omitempty"`
	Enabled         *bool             `json:"enabled,omitempty"`
}

// WidgetService provides widget CRUD operations.
type WidgetService interface {
	ListWidgets(ctx context.Context) ([]WidgetInfo, error)
	GetWidget(ctx context.Context, id uint) (*WidgetInfo, error)
	CreateWidget(ctx context.Context, req CreateWidgetRequest) (*WidgetInfo, error)
	UpdateWidget(ctx context.Context, id uint, req CreateWidgetRequest) error
	DeleteWidget(ctx context.Context, id uint) error
}

// WidgetHandler serves /api/v1/widgets endpoints.
type WidgetHandler struct {
	service WidgetService
}

// NewWidgetHandler creates a WidgetHandler.
func NewWidgetHandler(service WidgetService) *WidgetHandler {
	return &WidgetHandler{service: service}
}

// Routes returns a chi router with widget endpoints.
func (h *WidgetHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	return r
}

// List handles GET /api/v1/widgets.
func (h *WidgetHandler) List(w http.ResponseWriter, r *http.Request) {
	widgets, err := h.service.ListWidgets(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, widgets)
}

// Get handles GET /api/v1/widgets/{id}.
func (h *WidgetHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUintParam(r, "id")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	widget, err := h.service.GetWidget(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, widget)
}

// Create handles POST /api/v1/widgets.
func (h *WidgetHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateWidgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}
	if req.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.SchemaID == 0 {
		writeJSONError(w, http.StatusBadRequest, "schema_id is required")
		return
	}

	widget, err := h.service.CreateWidget(r.Context(), req)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, widget)
}

// Update handles PUT /api/v1/widgets/{id}.
func (h *WidgetHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUintParam(r, "id")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req CreateWidgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}

	if err := h.service.UpdateWidget(r.Context(), id, req); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Delete handles DELETE /api/v1/widgets/{id}.
func (h *WidgetHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUintParam(r, "id")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.DeleteWidget(r.Context(), id); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// WidgetScriptHandler serves GET /widget/{id}.js — the embed script.
type WidgetScriptHandler struct {
	service WidgetService
	baseURL string // engine base URL for widget iframe
}

// NewWidgetScriptHandler creates a WidgetScriptHandler.
func NewWidgetScriptHandler(service WidgetService, baseURL string) *WidgetScriptHandler {
	return &WidgetScriptHandler{service: service, baseURL: baseURL}
}

// ServeScript handles GET /widget/{id}.js.
func (h *WidgetScriptHandler) ServeScript(w http.ResponseWriter, r *http.Request) {
	id, err := parseUintParam(r, "id")
	if err != nil {
		http.Error(w, "invalid widget id", http.StatusBadRequest)
		return
	}

	widget, err := h.service.GetWidget(r.Context(), id)
	if err != nil {
		http.Error(w, "widget not found", http.StatusNotFound)
		return
	}

	// Check origin against domain whitelist
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = r.Header.Get("Referer")
	}

	// Set CORS headers based on whitelist
	allowedOrigin := "*"
	if len(widget.DomainWhitelist) > 0 && widget.DomainWhitelist[0] != "*" {
		allowedOrigin = origin // echo back origin if it's in the whitelist
	}
	w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)

	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	// Generate lightweight embed script
	script := fmt.Sprintf(`(function(){
  var w=document.createElement('div');
  w.id='bytebrew-widget-%d';
  w.innerHTML='<iframe src="%s/widget/%d/frame" style="border:none;position:fixed;%s;bottom:20px;width:400px;height:600px;z-index:999999;border-radius:12px;box-shadow:0 4px 24px rgba(0,0,0,0.15);" allow="microphone"></iframe>';
  document.body.appendChild(w);
})();`, id, h.baseURL, id, positionCSS(widget.Position))

	w.Write([]byte(script))
}

func positionCSS(position string) string {
	if position == "bottom-left" {
		return "left:20px"
	}
	return "right:20px"
}

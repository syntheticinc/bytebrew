package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// WidgetInfo is a widget returned in API responses.
type WidgetInfo struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Schema          string            `json:"schema"`
	Status          string            `json:"status"` // active, disabled
	PrimaryColor    string            `json:"primary_color"`
	Position        string            `json:"position"`
	Size            string            `json:"size"`
	WelcomeMessage  string            `json:"welcome_message"`
	PlaceholderText string            `json:"placeholder_text"`
	AvatarURL       string            `json:"avatar_url,omitempty"`
	DomainWhitelist string            `json:"domain_whitelist"`
	CustomHeaders   map[string]string `json:"custom_headers,omitempty"`
	CreatedAt       string            `json:"created_at,omitempty"`
}

// CreateWidgetRequest is the body for POST/PUT /api/v1/widgets.
type CreateWidgetRequest struct {
	Name            string            `json:"name"`
	Schema          string            `json:"schema"`
	Status          string            `json:"status,omitempty"` // active, disabled
	PrimaryColor    string            `json:"primary_color,omitempty"`
	Position        string            `json:"position,omitempty"`
	Size            string            `json:"size,omitempty"`
	WelcomeMessage  string            `json:"welcome_message,omitempty"`
	PlaceholderText string            `json:"placeholder_text,omitempty"`
	AvatarURL       string            `json:"avatar_url,omitempty"`
	DomainWhitelist string            `json:"domain_whitelist,omitempty"`
	CustomHeaders   map[string]string `json:"custom_headers,omitempty"`
}

// WidgetService provides widget CRUD operations.
type WidgetService interface {
	ListWidgets(ctx context.Context) ([]WidgetInfo, error)
	GetWidget(ctx context.Context, id string) (*WidgetInfo, error)
	CreateWidget(ctx context.Context, req CreateWidgetRequest) (*WidgetInfo, error)
	UpdateWidget(ctx context.Context, id string, req CreateWidgetRequest) error
	DeleteWidget(ctx context.Context, id string) error
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
	id, err := parseStringParam(r, "id")
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
	if req.Schema == "" {
		writeJSONError(w, http.StatusBadRequest, "schema is required")
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
	id, err := parseStringParam(r, "id")
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

	// Return the updated widget so the frontend can update its state.
	updated, err := h.service.GetWidget(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// Delete handles DELETE /api/v1/widgets/{id}.
func (h *WidgetHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseStringParam(r, "id")
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
}

// NewWidgetScriptHandler creates a WidgetScriptHandler.
func NewWidgetScriptHandler(service WidgetService) *WidgetScriptHandler {
	return &WidgetScriptHandler{service: service}
}

// ServeScript handles GET /widget/{id}.js.
func (h *WidgetScriptHandler) ServeScript(w http.ResponseWriter, r *http.Request) {
	id, err := parseStringParam(r, "id")
	if err != nil {
		http.Error(w, "invalid widget id", http.StatusBadRequest)
		return
	}

	widget, err := h.service.GetWidget(r.Context(), id)
	if err != nil {
		http.Error(w, "widget not found", http.StatusNotFound)
		return
	}

	// Check if widget is enabled.
	if widget.Status == "disabled" {
		http.Error(w, "widget is disabled", http.StatusForbidden)
		return
	}

	// Origin validation against domain whitelist.
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = r.Header.Get("Referer")
	}
	allowedOrigin := "*"
	if widget.DomainWhitelist != "" {
		if !isOriginAllowed(origin, widget.DomainWhitelist) {
			http.Error(w, "origin not allowed", http.StatusForbidden)
			return
		}
		allowedOrigin = origin
	}
	w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	// Derive the base URL from the request (scheme + host) for loading widget.js.
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	if fwd := r.Header.Get("X-Forwarded-Proto"); fwd != "" {
		scheme = fwd
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, r.Host)

	// Generate bootstrap script that loads widget.js with admin-configured data attributes.
	script := fmt.Sprintf(`(function(){
  if(document.getElementById('bytebrew-widget-%s'))return;
  var s=document.createElement('script');
  s.id='bytebrew-widget-%s';
  s.src='%s/widget.js';
  s.setAttribute('data-widget-id','%s');
  s.setAttribute('data-agent','%s');
  s.setAttribute('data-endpoint','%s');
  s.setAttribute('data-position','%s');
  s.setAttribute('data-theme','light');
  s.setAttribute('data-title','%s');
  s.setAttribute('data-primary-color','%s');
  s.setAttribute('data-welcome','%s');
  s.setAttribute('data-placeholder','%s');
  document.body.appendChild(s);
})();`,
		id, id, baseURL, id,
		escapeJS(widget.Schema),
		baseURL,
		escapeJS(widget.Position),
		escapeJS(widget.Name),
		escapeJS(widget.PrimaryColor),
		escapeJS(widget.WelcomeMessage),
		escapeJS(widget.PlaceholderText),
	)

	_, _ = w.Write([]byte(script))
}

// isOriginAllowed checks if the origin matches the comma-separated domain whitelist.
func isOriginAllowed(origin, whitelist string) bool {
	if origin == "" {
		return true // no origin header (e.g., direct browser navigation)
	}
	for _, d := range strings.Split(whitelist, ",") {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if d == "*" {
			return true
		}
		if strings.EqualFold(d, origin) {
			return true
		}
		// Allow subdomain matching: whitelist "example.com" matches "https://sub.example.com"
		if strings.HasSuffix(strings.ToLower(origin), strings.ToLower(d)) {
			return true
		}
	}
	return false
}

// escapeJS escapes a string for safe embedding in a JS string literal.
func escapeJS(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", ``)
	return s
}

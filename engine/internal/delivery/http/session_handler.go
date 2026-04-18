package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// SessionResponse is the API representation of a session.
type SessionResponse struct {
	ID        string `json:"id"`
	Title     string `json:"title,omitempty"`
	SchemaID  string `json:"schema_id,omitempty"`
	UserSub   string `json:"user_sub,omitempty"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// PaginatedSessionResponse wraps a page of sessions with pagination metadata.
type PaginatedSessionResponse struct {
	Data       []SessionResponse `json:"data"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PerPage    int               `json:"per_page"`
	TotalPages int               `json:"total_pages"`
}

// CreateSessionRequest is the body for POST /api/v1/sessions.
type CreateSessionRequest struct {
	ID       string `json:"id,omitempty"`
	Title    string `json:"title,omitempty"`
	SchemaID string `json:"schema_id,omitempty"`
	UserSub  string `json:"user_sub,omitempty"`
}

// UpdateSessionRequest is the body for PUT /api/v1/sessions/{id}.
type UpdateSessionRequest struct {
	Title  *string `json:"title,omitempty"`
	Status *string `json:"status,omitempty"`
}

// SessionService provides session CRUD operations.
type SessionService interface {
	ListSessions(ctx context.Context, agentName, userSub, status, from, to string, page, perPage int) ([]SessionResponse, int64, error)
	GetSession(ctx context.Context, id string) (*SessionResponse, error)
	CreateSession(ctx context.Context, req CreateSessionRequest) (*SessionResponse, error)
	UpdateSession(ctx context.Context, id string, req UpdateSessionRequest) (*SessionResponse, error)
	DeleteSession(ctx context.Context, id string) error
}

// EventResponse is the API representation of a session event (message, tool call, reasoning, etc.).
type EventResponse struct {
	ID        string          `json:"id"`
	EventType string          `json:"event_type"`
	AgentID   string          `json:"agent_id,omitempty"`
	CallID    string          `json:"call_id,omitempty"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt string          `json:"created_at"`
}

// EventService provides event query operations for a session.
type EventService interface {
	ListEvents(ctx context.Context, sessionID string) ([]EventResponse, error)
}

// SessionHandler serves /api/v1/sessions endpoints.
type SessionHandler struct {
	service    SessionService
	eventSvc EventService
}

// NewSessionHandler creates a SessionHandler.
func NewSessionHandler(service SessionService) *SessionHandler {
	return &SessionHandler{service: service}
}

// SetEventService sets the optional EventService for listing chat history.
func (h *SessionHandler) SetEventService(svc EventService) {
	h.eventSvc = svc
}

// Routes returns a chi router with session endpoints mounted.
func (h *SessionHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Get("/{id}/messages", h.ListMessages)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	return r
}

// List handles GET /api/v1/sessions.
func (h *SessionHandler) List(w http.ResponseWriter, r *http.Request) {
	agentName := r.URL.Query().Get("agent_name")
	userSub := r.URL.Query().Get("user_sub")
	status := r.URL.Query().Get("status")
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	page := 1
	perPage := 20

	if v := r.URL.Query().Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	if v := r.URL.Query().Get("per_page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			if p > 100 {
				p = 100
			}
			perPage = p
		}
	}

	sessions, total, err := h.service.ListSessions(r.Context(), agentName, userSub, status, from, to, page, perPage)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	totalPages := int(total) / perPage
	if int(total)%perPage != 0 {
		totalPages++
	}

	writeJSON(w, http.StatusOK, PaginatedSessionResponse{
		Data:       sessions,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	})
}

// Get handles GET /api/v1/sessions/{id}.
func (h *SessionHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "id parameter is required")
		return
	}

	session, err := h.service.GetSession(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if session == nil {
		writeJSONError(w, http.StatusNotFound, fmt.Sprintf("session not found: %s", id))
		return
	}

	writeJSON(w, http.StatusOK, session)
}

// Create handles POST /api/v1/sessions.
func (h *SessionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}
	session, err := h.service.CreateSession(r.Context(), req)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, session)
}

// Update handles PUT /api/v1/sessions/{id}.
func (h *SessionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "id parameter is required")
		return
	}

	var req UpdateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}

	session, err := h.service.UpdateSession(r.Context(), id, req)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if session == nil {
		writeJSONError(w, http.StatusNotFound, fmt.Sprintf("session not found: %s", id))
		return
	}

	writeJSON(w, http.StatusOK, session)
}

// Delete handles DELETE /api/v1/sessions/{id}.
func (h *SessionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "id parameter is required")
		return
	}

	if err := h.service.DeleteSession(r.Context(), id); err != nil {
		writeDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListMessages handles GET /api/v1/sessions/{id}/messages.
// Returns session events (messages, tool calls, reasoning) in chronological order.
func (h *SessionHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "id parameter is required")
		return
	}

	if h.eventSvc == nil {
		writeJSON(w, http.StatusOK, []EventResponse{})
		return
	}

	events, err := h.eventSvc.ListEvents(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, events)
}

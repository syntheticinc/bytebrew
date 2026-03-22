package http

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

// AuditResponse is the API representation of an audit log entry.
type AuditResponse struct {
	ID        uint   `json:"id"`
	Timestamp string `json:"timestamp"`
	ActorType string `json:"actor_type"`
	ActorID   string `json:"actor_id"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
	Details   string `json:"details"`
}

// PaginatedAuditResponse wraps a page of audit logs with pagination metadata.
type PaginatedAuditResponse struct {
	Data       []AuditResponse `json:"data"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PerPage    int             `json:"per_page"`
	TotalPages int             `json:"total_pages"`
}

// AuditService provides audit log query operations.
type AuditService interface {
	ListAuditLogs(ctx context.Context, actorType, action, resource string, from, to *time.Time, page, perPage int) ([]AuditResponse, int64, error)
}

// AuditHandler serves /api/v1/audit endpoints.
type AuditHandler struct {
	service AuditService
}

// NewAuditHandler creates an AuditHandler.
func NewAuditHandler(service AuditService) *AuditHandler {
	return &AuditHandler{service: service}
}

// List handles GET /api/v1/audit.
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	actorType := r.URL.Query().Get("actor_type")
	action := r.URL.Query().Get("action")
	resource := r.URL.Query().Get("resource")

	var from, to *time.Time
	if v := r.URL.Query().Get("from"); v != "" {
		t, err := parseTimeParam(v)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid 'from' date: use RFC3339 or YYYY-MM-DD format")
			return
		}
		from = &t
	}
	if v := r.URL.Query().Get("to"); v != "" {
		t, err := parseTimeParam(v)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid 'to' date: use RFC3339 or YYYY-MM-DD format")
			return
		}
		to = &t
	}

	page := 1
	perPage := 50

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

	logs, total, err := h.service.ListAuditLogs(r.Context(), actorType, action, resource, from, to, page, perPage)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	totalPages := int(total) / perPage
	if int(total)%perPage != 0 {
		totalPages++
	}

	writeJSON(w, http.StatusOK, PaginatedAuditResponse{
		Data:       logs,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	})
}

// parseTimeParam parses a time string in RFC3339 or YYYY-MM-DD format.
func parseTimeParam(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", s)
}

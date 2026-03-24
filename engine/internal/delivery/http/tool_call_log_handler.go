package http

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

// ToolCallFilters holds query parameters for filtering tool call audit entries.
type ToolCallFilters struct {
	SessionID string
	AgentName string
	ToolName  string
	Status    string // "completed" or "failed"
	UserID    string
	From      *time.Time
	To        *time.Time
}

// ToolCallEntry represents a single tool call in the audit log response.
type ToolCallEntry struct {
	ID         uint      `json:"id"`
	SessionID  string    `json:"session_id"`
	AgentName  string    `json:"agent_name"`
	ToolName   string    `json:"tool_name"`
	Input      string    `json:"input"`
	Output     string    `json:"output"`
	Status     string    `json:"status"`
	DurationMs int64     `json:"duration_ms"`
	UserID     string    `json:"user_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// PaginatedToolCallResponse wraps tool call entries with pagination metadata.
type PaginatedToolCallResponse struct {
	Data       []ToolCallEntry `json:"data"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PerPage    int             `json:"per_page"`
	TotalPages int             `json:"total_pages"`
}

// ToolCallEventQuerier provides tool call audit query operations.
type ToolCallEventQuerier interface {
	QueryToolCalls(ctx context.Context, filters ToolCallFilters, page, perPage int) ([]ToolCallEntry, int64, error)
}

// ToolCallLogHandler serves GET /api/v1/audit/tool-calls.
type ToolCallLogHandler struct {
	querier ToolCallEventQuerier
}

// NewToolCallLogHandler creates a ToolCallLogHandler.
func NewToolCallLogHandler(querier ToolCallEventQuerier) *ToolCallLogHandler {
	return &ToolCallLogHandler{querier: querier}
}

// List handles GET /api/v1/audit/tool-calls.
func (h *ToolCallLogHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var filters ToolCallFilters
	filters.SessionID = q.Get("session_id")
	filters.AgentName = q.Get("agent")
	filters.ToolName = q.Get("tool")
	filters.Status = q.Get("status")
	filters.UserID = q.Get("user_id")

	if v := q.Get("from"); v != "" {
		t, err := parseTimeParam(v)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid 'from' date: use RFC3339 or YYYY-MM-DD format")
			return
		}
		filters.From = &t
	}
	if v := q.Get("to"); v != "" {
		t, err := parseTimeParam(v)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid 'to' date: use RFC3339 or YYYY-MM-DD format")
			return
		}
		filters.To = &t
	}

	page := 1
	perPage := 50

	if v := q.Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	if v := q.Get("per_page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			if p > 100 {
				p = 100
			}
			perPage = p
		}
	}

	entries, total, err := h.querier.QueryToolCalls(r.Context(), filters, page, perPage)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if entries == nil {
		entries = []ToolCallEntry{}
	}

	totalPages := int(total) / perPage
	if int(total)%perPage != 0 {
		totalPages++
	}

	writeJSON(w, http.StatusOK, PaginatedToolCallResponse{
		Data:       entries,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	})
}

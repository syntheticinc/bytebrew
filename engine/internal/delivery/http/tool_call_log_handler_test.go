package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockToolCallQuerier struct {
	entries []ToolCallEntry
	total   int64
	err     error

	lastFilters ToolCallFilters
	lastPage    int
	lastPerPage int
}

func (m *mockToolCallQuerier) QueryToolCalls(_ context.Context, filters ToolCallFilters, page, perPage int) ([]ToolCallEntry, int64, error) {
	m.lastFilters = filters
	m.lastPage = page
	m.lastPerPage = perPage
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.entries, m.total, nil
}

func newToolCallRouter(handler *ToolCallLogHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/api/v1/audit/tool-calls", handler.List)
	return r
}

func TestToolCallLogHandler_List(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	svc := &mockToolCallQuerier{
		entries: []ToolCallEntry{
			{
				ID:         "1",
				SessionID:  "sess-1",
				AgentName:  "supervisor",
				ToolName:   "read_file",
				Input:      `{"path":"main.go"}`,
				Output:     "file contents...",
				Status:     "completed",
				DurationMs: 120,
				CreatedAt:  now,
			},
			{
				ID:         "2",
				SessionID:  "sess-1",
				AgentName:  "code-agent-abc",
				ToolName:   "execute_command",
				Input:      `{"command":"go build"}`,
				Output:     "build error",
				Status:     "failed",
				DurationMs: 5000,
				CreatedAt:  now.Add(-time.Minute),
			},
		},
		total: 2,
	}
	handler := NewToolCallLogHandler(svc)
	r := newToolCallRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/tool-calls", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp PaginatedToolCallResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))

	assert.Equal(t, int64(2), resp.Total)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 50, resp.PerPage)
	assert.Equal(t, 1, resp.TotalPages)
	assert.Len(t, resp.Data, 2)
	assert.Equal(t, "1", resp.Data[0].ID)
	assert.Equal(t, "read_file", resp.Data[0].ToolName)
	assert.Equal(t, "supervisor", resp.Data[0].AgentName)
	assert.Equal(t, "completed", resp.Data[0].Status)
	assert.Equal(t, int64(120), resp.Data[0].DurationMs)
	assert.Equal(t, "failed", resp.Data[1].Status)
}

func TestToolCallLogHandler_List_Filters(t *testing.T) {
	svc := &mockToolCallQuerier{
		entries: []ToolCallEntry{
			{ID: "10", SessionID: "sess-42", ToolName: "search_code", Status: "completed"},
		},
		total: 1,
	}
	handler := NewToolCallLogHandler(svc)
	r := newToolCallRouter(handler)

	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/audit/tool-calls?session_id=sess-42&tool=search_code&agent=supervisor&status=completed&user_id=user-1&from=2026-03-01&to=2026-03-24T23:59:59Z",
		nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp PaginatedToolCallResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, int64(1), resp.Total)
	assert.Len(t, resp.Data, 1)

	// Verify filters passed to querier.
	assert.Equal(t, "sess-42", svc.lastFilters.SessionID)
	assert.Equal(t, "search_code", svc.lastFilters.ToolName)
	assert.Equal(t, "supervisor", svc.lastFilters.AgentName)
	assert.Equal(t, "completed", svc.lastFilters.Status)
	assert.Equal(t, "user-1", svc.lastFilters.UserID)
	require.NotNil(t, svc.lastFilters.From)
	require.NotNil(t, svc.lastFilters.To)

	expectedFrom, _ := time.Parse("2006-01-02", "2026-03-01")
	assert.Equal(t, expectedFrom, *svc.lastFilters.From)

	expectedTo, _ := time.Parse(time.RFC3339, "2026-03-24T23:59:59Z")
	assert.Equal(t, expectedTo, *svc.lastFilters.To)
}

func TestToolCallLogHandler_List_Pagination(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		total          int64
		wantPage       int
		wantPerPage    int
		wantTotalPages int
	}{
		{
			name:           "default pagination",
			query:          "",
			total:          150,
			wantPage:       1,
			wantPerPage:    50,
			wantTotalPages: 3,
		},
		{
			name:           "custom page and per_page",
			query:          "?page=3&per_page=25",
			total:          75,
			wantPage:       3,
			wantPerPage:    25,
			wantTotalPages: 3,
		},
		{
			name:           "per_page capped at 100",
			query:          "?per_page=200",
			total:          500,
			wantPage:       1,
			wantPerPage:    100,
			wantTotalPages: 5,
		},
		{
			name:           "zero total",
			query:          "",
			total:          0,
			wantPage:       1,
			wantPerPage:    50,
			wantTotalPages: 0,
		},
		{
			name:           "partial last page",
			query:          "?per_page=30",
			total:          100,
			wantPage:       1,
			wantPerPage:    30,
			wantTotalPages: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockToolCallQuerier{
				entries: []ToolCallEntry{},
				total:   tt.total,
			}
			handler := NewToolCallLogHandler(svc)
			r := newToolCallRouter(handler)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/tool-calls"+tt.query, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)

			var resp PaginatedToolCallResponse
			require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))

			assert.Equal(t, tt.total, resp.Total)
			assert.Equal(t, tt.wantPage, resp.Page)
			assert.Equal(t, tt.wantPerPage, resp.PerPage)
			assert.Equal(t, tt.wantTotalPages, resp.TotalPages)

			assert.Equal(t, tt.wantPage, svc.lastPage)
			assert.Equal(t, tt.wantPerPage, svc.lastPerPage)
		})
	}
}

func TestToolCallLogHandler_List_EmptyResult(t *testing.T) {
	svc := &mockToolCallQuerier{
		entries: nil,
		total:   0,
	}
	handler := NewToolCallLogHandler(svc)
	r := newToolCallRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/tool-calls", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp PaginatedToolCallResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))

	assert.Equal(t, int64(0), resp.Total)
	assert.NotNil(t, resp.Data, "data should be an empty array, not null")
	assert.Len(t, resp.Data, 0)
	assert.Equal(t, 0, resp.TotalPages)
}

func TestToolCallLogHandler_List_InvalidDateReturnsError(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  string
	}{
		{"invalid from", "?from=not-a-date", "invalid 'from' date"},
		{"invalid to", "?to=xyz123", "invalid 'to' date"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockToolCallQuerier{}
			handler := NewToolCallLogHandler(svc)
			r := newToolCallRouter(handler)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/tool-calls"+tt.query, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), tt.want)
		})
	}
}

func TestToolCallLogHandler_List_ServiceError(t *testing.T) {
	svc := &mockToolCallQuerier{err: fmt.Errorf("connection refused")}
	handler := NewToolCallLogHandler(svc)
	r := newToolCallRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/tool-calls", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "connection refused")
}

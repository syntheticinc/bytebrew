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

type mockAuditService struct {
	logs  []AuditResponse
	total int64
	err   error

	lastActorType string
	lastAction    string
	lastResource  string
	lastFrom      *time.Time
	lastTo        *time.Time
	lastPage      int
	lastPerPage   int
}

func (m *mockAuditService) ListAuditLogs(_ context.Context, actorType, action, resource string, from, to *time.Time, page, perPage int) ([]AuditResponse, int64, error) {
	m.lastActorType = actorType
	m.lastAction = action
	m.lastResource = resource
	m.lastFrom = from
	m.lastTo = to
	m.lastPage = page
	m.lastPerPage = perPage
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.logs, m.total, nil
}

func newAuditRouter(handler *AuditHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/api/v1/audit", handler.List)
	return r
}

func TestAuditHandler_List(t *testing.T) {
	svc := &mockAuditService{
		logs: []AuditResponse{
			{ID: "1", Timestamp: "2026-03-19T10:00:00Z", ActorType: "admin", ActorID: "admin@test.com", Action: "create", Resource: "agent", Details: "created agent foo"},
			{ID: "2", Timestamp: "2026-03-19T09:00:00Z", ActorType: "api_token", ActorID: "tok-123", Action: "delete", Resource: "trigger", Details: "deleted trigger bar"},
		},
		total: 2,
	}
	handler := NewAuditHandler(svc)
	r := newAuditRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp PaginatedAuditResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))

	assert.Equal(t, int64(2), resp.Total)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 50, resp.PerPage)
	assert.Equal(t, 1, resp.TotalPages)
	assert.Len(t, resp.Data, 2)
	assert.Equal(t, "1", resp.Data[0].ID)
	assert.Equal(t, "admin", resp.Data[0].ActorType)

	// Verify defaults passed to service
	assert.Equal(t, 1, svc.lastPage)
	assert.Equal(t, 50, svc.lastPerPage)
	assert.Empty(t, svc.lastActorType)
	assert.Empty(t, svc.lastAction)
	assert.Empty(t, svc.lastResource)
	assert.Nil(t, svc.lastFrom)
	assert.Nil(t, svc.lastTo)
}

func TestAuditHandler_List_WithFilters(t *testing.T) {
	svc := &mockAuditService{
		logs:  []AuditResponse{{ID: "5", Timestamp: "2026-03-15T12:00:00Z", ActorType: "admin", ActorID: "admin", Action: "create", Resource: "agent"}},
		total: 1,
	}
	handler := NewAuditHandler(svc)
	r := newAuditRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit?actor_type=admin&action=create&resource=agent&from=2026-03-01&to=2026-03-19T23:59:59Z", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp PaginatedAuditResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))

	assert.Equal(t, int64(1), resp.Total)
	assert.Len(t, resp.Data, 1)

	// Verify filters passed to service
	assert.Equal(t, "admin", svc.lastActorType)
	assert.Equal(t, "create", svc.lastAction)
	assert.Equal(t, "agent", svc.lastResource)
	require.NotNil(t, svc.lastFrom)
	require.NotNil(t, svc.lastTo)

	expectedFrom, _ := time.Parse("2006-01-02", "2026-03-01")
	assert.Equal(t, expectedFrom, *svc.lastFrom)

	expectedTo, _ := time.Parse(time.RFC3339, "2026-03-19T23:59:59Z")
	assert.Equal(t, expectedTo, *svc.lastTo)
}

func TestAuditHandler_List_Pagination(t *testing.T) {
	tests := []struct {
		name            string
		query           string
		total           int64
		wantPage        int
		wantPerPage     int
		wantTotalPages  int
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
			query:          "?page=2&per_page=25",
			total:          75,
			wantPage:       2,
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
			svc := &mockAuditService{
				logs:  []AuditResponse{},
				total: tt.total,
			}
			handler := NewAuditHandler(svc)
			r := newAuditRouter(handler)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/audit"+tt.query, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)

			var resp PaginatedAuditResponse
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

func TestAuditHandler_List_InvalidDateReturnsError(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  string
	}{
		{"invalid from", "?from=not-a-date", "invalid 'from' date"},
		{"invalid to", "?to=123456", "invalid 'to' date"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockAuditService{}
			handler := NewAuditHandler(svc)
			r := newAuditRouter(handler)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/audit"+tt.query, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), tt.want)
		})
	}
}

func TestAuditHandler_List_ServiceError(t *testing.T) {
	svc := &mockAuditService{err: fmt.Errorf("database connection lost")}
	handler := NewAuditHandler(svc)
	r := newAuditRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "database connection lost")
}

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSessionService struct {
	sessions []SessionResponse
	total    int64
	session  *SessionResponse
	created  *SessionResponse
	updated  *SessionResponse
	err      error

	lastListAgentName string
	lastListUserSub   string
	lastListStatus    string
	lastListFrom      string
	lastListTo        string
	lastListPage      int
	lastListPerPage   int
	lastDeleteID      string
}

func (m *mockSessionService) ListSessions(_ context.Context, agentName, userSub, status, from, to string, page, perPage int) ([]SessionResponse, int64, error) {
	m.lastListAgentName = agentName
	m.lastListUserSub = userSub
	m.lastListStatus = status
	m.lastListFrom = from
	m.lastListTo = to
	m.lastListPage = page
	m.lastListPerPage = perPage
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.sessions, m.total, nil
}

func (m *mockSessionService) GetSession(_ context.Context, id string) (*SessionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.session != nil && m.session.ID == id {
		return m.session, nil
	}
	return nil, nil
}

func (m *mockSessionService) CreateSession(_ context.Context, _ CreateSessionRequest) (*SessionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.created, nil
}

func (m *mockSessionService) UpdateSession(_ context.Context, id string, _ UpdateSessionRequest) (*SessionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.updated != nil {
		return m.updated, nil
	}
	return nil, nil
}

func (m *mockSessionService) DeleteSession(_ context.Context, id string) error {
	m.lastDeleteID = id
	return m.err
}

func newSessionRouter(handler *SessionHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Mount("/sessions", handler.Routes())
	return r
}

func TestSessionHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		sessions   []SessionResponse
		total      int64
		wantStatus int
		wantTotal  int64
	}{
		{
			name:  "returns paginated sessions",
			query: "?page=1&per_page=10",
			sessions: []SessionResponse{
				{ID: "s1", UserSub: "u1", Status: "active", CreatedAt: "2026-03-19T10:00:00Z", UpdatedAt: "2026-03-19T10:05:00Z"},
			},
			total:      1,
			wantStatus: http.StatusOK,
			wantTotal:  1,
		},
		{
			name:       "empty list",
			query:      "",
			sessions:   []SessionResponse{},
			total:      0,
			wantStatus: http.StatusOK,
			wantTotal:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockSessionService{sessions: tt.sessions, total: tt.total}
			handler := NewSessionHandler(svc)
			router := newSessionRouter(handler)

			req := httptest.NewRequest(http.MethodGet, "/sessions"+tt.query, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			var resp PaginatedSessionResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
			assert.Equal(t, tt.wantTotal, resp.Total)
			assert.Equal(t, len(tt.sessions), len(resp.Data))
		})
	}
}

func TestSessionHandler_List_Filters(t *testing.T) {
	svc := &mockSessionService{sessions: []SessionResponse{}, total: 0}
	handler := NewSessionHandler(svc)
	router := newSessionRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/sessions?agent_name=sales&user_sub=u1&status=active&page=2&per_page=5", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "sales", svc.lastListAgentName)
	assert.Equal(t, "u1", svc.lastListUserSub)
	assert.Equal(t, "active", svc.lastListStatus)
	assert.Equal(t, 2, svc.lastListPage)
	assert.Equal(t, 5, svc.lastListPerPage)
}

func TestSessionHandler_Get(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		session    *SessionResponse
		wantStatus int
	}{
		{
			name:       "found",
			id:         "s1",
			session:    &SessionResponse{ID: "s1", UserSub: "u1", Status: "active", CreatedAt: "2026-03-19T10:00:00Z", UpdatedAt: "2026-03-19T10:05:00Z"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "not found",
			id:         "s999",
			session:    &SessionResponse{ID: "s1"},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockSessionService{session: tt.session}
			handler := NewSessionHandler(svc)
			router := newSessionRouter(handler)

			req := httptest.NewRequest(http.MethodGet, "/sessions/"+tt.id, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestSessionHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		created    *SessionResponse
		wantStatus int
	}{
		{
			name:       "valid request",
			body:       `{"user_sub":"u1","title":"Help me"}`,
			created:    &SessionResponse{ID: "s1", UserSub: "u1", Title: "Help me", Status: "active", CreatedAt: "2026-03-19T10:00:00Z", UpdatedAt: "2026-03-19T10:00:00Z"},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid json",
			body:       `{invalid}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockSessionService{created: tt.created}
			handler := NewSessionHandler(svc)
			router := newSessionRouter(handler)

			req := httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusCreated {
				var resp SessionResponse
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
				assert.Equal(t, tt.created.ID, resp.ID)
				assert.Equal(t, tt.created.Title, resp.Title)
			}
		})
	}
}

func TestSessionHandler_Update(t *testing.T) {
	updated := &SessionResponse{ID: "s1", UserSub: "u1", Title: "New title", Status: "active", CreatedAt: "2026-03-19T10:00:00Z", UpdatedAt: "2026-03-19T10:06:00Z"}
	svc := &mockSessionService{updated: updated}
	handler := NewSessionHandler(svc)
	router := newSessionRouter(handler)

	body := `{"title":"New title"}`
	req := httptest.NewRequest(http.MethodPut, "/sessions/s1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp SessionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "New title", resp.Title)
}

func TestSessionHandler_Update_NotFound(t *testing.T) {
	svc := &mockSessionService{} // updated is nil
	handler := NewSessionHandler(svc)
	router := newSessionRouter(handler)

	body := `{"title":"New title"}`
	req := httptest.NewRequest(http.MethodPut, "/sessions/s999", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSessionHandler_Delete(t *testing.T) {
	svc := &mockSessionService{}
	handler := NewSessionHandler(svc)
	router := newSessionRouter(handler)

	req := httptest.NewRequest(http.MethodDelete, "/sessions/s1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "s1", svc.lastDeleteID)
}

func TestSessionHandler_Delete_Error(t *testing.T) {
	svc := &mockSessionService{err: fmt.Errorf("session not found: s999")}
	handler := NewSessionHandler(svc)
	router := newSessionRouter(handler)

	req := httptest.NewRequest(http.MethodDelete, "/sessions/s999", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestSessionHandler_List_PerPageCap(t *testing.T) {
	svc := &mockSessionService{sessions: []SessionResponse{}, total: 0}
	handler := NewSessionHandler(svc)
	router := newSessionRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/sessions?per_page=200", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 100, svc.lastListPerPage) // capped at 100
}

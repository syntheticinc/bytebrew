package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockKnowledgeStats struct {
	docs    int
	chunks  int
	lastIdx *time.Time
	err     error
}

func (m *mockKnowledgeStats) GetStats(_ context.Context, _ string) (int, int, *time.Time, error) {
	return m.docs, m.chunks, m.lastIdx, m.err
}

type mockKnowledgeReindexer struct {
	called    bool
	agentName string
	err       error
}

func (m *mockKnowledgeReindexer) Reindex(_ context.Context, agentName string) error {
	m.called = true
	m.agentName = agentName
	return m.err
}

func newKnowledgeRouter(handler *KnowledgeHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/api/v1/agents/{name}/knowledge/status", handler.Status)
	r.Post("/api/v1/agents/{name}/knowledge/reindex", handler.Reindex)
	return r
}

func TestKnowledgeHandler_Status(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	stats := &mockKnowledgeStats{docs: 5, chunks: 42, lastIdx: &now}
	handler := NewKnowledgeHandler(stats, nil)
	router := newKnowledgeRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/sales/knowledge/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, `"agent":"sales"`)
	assert.Contains(t, body, `"documents":5`)
	assert.Contains(t, body, `"chunks":42`)
}

func TestKnowledgeHandler_Status_NoDocuments(t *testing.T) {
	stats := &mockKnowledgeStats{docs: 0, chunks: 0, lastIdx: nil}
	handler := NewKnowledgeHandler(stats, nil)
	router := newKnowledgeRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/empty-agent/knowledge/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"documents":0`)
}

func TestKnowledgeHandler_Reindex(t *testing.T) {
	reindexer := &mockKnowledgeReindexer{}
	handler := NewKnowledgeHandler(&mockKnowledgeStats{}, reindexer)
	router := newKnowledgeRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/sales/knowledge/reindex", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), `"indexing_started"`)
}

func TestKnowledgeHandler_Reindex_NoReindexer(t *testing.T) {
	handler := NewKnowledgeHandler(&mockKnowledgeStats{}, nil)
	router := newKnowledgeRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/sales/knowledge/reindex", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotImplemented, w.Code)
}

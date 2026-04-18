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
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

type mockMemoryLister struct {
	memories []*domain.Memory
	err      error
}

func (m *mockMemoryLister) Execute(ctx context.Context, schemaID string) ([]*domain.Memory, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.memories, nil
}

type mockMemoryClearer struct {
	deletedCount int64
	err          error
}

func (m *mockMemoryClearer) ClearAll(ctx context.Context, schemaID string) (int64, error) {
	return m.deletedCount, m.err
}

func (m *mockMemoryClearer) DeleteOne(ctx context.Context, id string) error {
	return m.err
}

func setupMemoryRouter(lister MemoryLister, clearer MemoryClearer) *chi.Mux {
	handler := NewMemoryHandler(lister, clearer)
	r := chi.NewRouter()
	r.Get("/api/v1/schemas/{id}/memory", handler.ListMemories)
	r.Delete("/api/v1/schemas/{id}/memory", handler.ClearMemories)
	r.Delete("/api/v1/schemas/{id}/memory/{entry_id}", handler.DeleteMemory)
	return r
}

func TestMemoryHandler_ListMemories(t *testing.T) {
	lister := &mockMemoryLister{
		memories: []*domain.Memory{
			{ID: "1", SchemaID: "10", UserSub: "user-1", Content: "user prefers dark mode", CreatedAt: time.Now()},
			{ID: "2", SchemaID: "10", UserSub: "user-1", Content: "user name is Alice", CreatedAt: time.Now()},
		},
	}
	r := setupMemoryRouter(lister, &mockMemoryClearer{})

	req := httptest.NewRequest("GET", "/api/v1/schemas/10/memory", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []memoryResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Len(t, resp, 2)
	assert.Equal(t, "user prefers dark mode", resp[0].Content)
}

func TestMemoryHandler_ListMemories_Empty(t *testing.T) {
	lister := &mockMemoryLister{memories: []*domain.Memory{}}
	r := setupMemoryRouter(lister, &mockMemoryClearer{})

	req := httptest.NewRequest("GET", "/api/v1/schemas/10/memory", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []memoryResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Len(t, resp, 0)
}

func TestMemoryHandler_ClearMemories(t *testing.T) {
	clearer := &mockMemoryClearer{deletedCount: 5}
	r := setupMemoryRouter(&mockMemoryLister{}, clearer)

	req := httptest.NewRequest("DELETE", "/api/v1/schemas/10/memory", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, float64(5), resp["deleted"])
}

func TestMemoryHandler_DeleteMemory(t *testing.T) {
	clearer := &mockMemoryClearer{}
	r := setupMemoryRouter(&mockMemoryLister{}, clearer)

	req := httptest.NewRequest("DELETE", "/api/v1/schemas/10/memory/42", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMemoryHandler_DeleteMemory_Error(t *testing.T) {
	clearer := &mockMemoryClearer{err: fmt.Errorf("memory not found: 999")}
	r := setupMemoryRouter(&mockMemoryLister{}, clearer)

	req := httptest.NewRequest("DELETE", "/api/v1/schemas/10/memory/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAgentCounter struct {
	count int
}

func (m *mockAgentCounter) Count() int { return m.count }

func TestHealthHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		agentsCount int
	}{
		{"basic response", "1.0.0", 3},
		{"zero agents", "2.0.0-beta", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHealthHandler(tt.version, &mockAgentCounter{count: tt.agentsCount})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			var resp HealthResponse
			err := json.NewDecoder(rec.Body).Decode(&resp)
			require.NoError(t, err)
			assert.Equal(t, "ok", resp.Status)
			assert.Equal(t, tt.version, resp.Version)
			assert.Equal(t, tt.agentsCount, resp.AgentsCount)
			assert.NotEmpty(t, resp.Uptime)
		})
	}
}

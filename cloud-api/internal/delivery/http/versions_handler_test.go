package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionsHandler_GetEngineVersion(t *testing.T) {
	handler := NewVersionsHandler("1.2.3")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/versions/engine", nil)
	w := httptest.NewRecorder()

	handler.GetEngineVersion(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp successResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "1.2.3", data["latest"])
	assert.Equal(t, "0.3.0", data["min_supported"])
	assert.Equal(t, "bytebrew/engine:1.2.3", data["docker_image"])
	assert.Equal(t, "https://bytebrew.ai/releases/v1.2.3/", data["download_url"])
	assert.Equal(t, "https://github.com/syntheticinc/bytebrew/releases/tag/v1.2.3", data["changelog_url"])
}

func TestVersionsHandler_DifferentVersion(t *testing.T) {
	handler := NewVersionsHandler("0.5.0")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/versions/engine", nil)
	w := httptest.NewRecorder()

	handler.GetEngineVersion(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp successResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "0.5.0", data["latest"])
	assert.Contains(t, data["download_url"], "v0.5.0")
}

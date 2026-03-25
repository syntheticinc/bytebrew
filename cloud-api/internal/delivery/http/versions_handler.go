package http

import (
	"net/http"
)

// VersionsHandler serves public version information endpoints.
type VersionsHandler struct {
	latestVersion string
}

// NewVersionsHandler creates a VersionsHandler with the given latest engine version.
func NewVersionsHandler(latestVersion string) *VersionsHandler {
	return &VersionsHandler{latestVersion: latestVersion}
}

// GetEngineVersion handles GET /api/v1/versions/engine.
func (h *VersionsHandler) GetEngineVersion(w http.ResponseWriter, _ *http.Request) {
	JSON(w, http.StatusOK, map[string]string{
		"latest":        h.latestVersion,
		"min_supported": "0.3.0",
		"docker_image":  "bytebrew/engine:" + h.latestVersion,
		"download_url":  "https://bytebrew.ai/releases/v" + h.latestVersion + "/",
		"changelog_url": "https://github.com/syntheticinc/bytebrew/releases/tag/v" + h.latestVersion,
	})
}

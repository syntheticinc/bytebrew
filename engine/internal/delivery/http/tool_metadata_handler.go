package http

import (
	"net/http"
)

// ToolMetadataProvider returns metadata for all known tools.
type ToolMetadataProvider interface {
	GetAllToolMetadata() []ToolMetadataResponse
}

// ToolMetadataResponse is the JSON shape returned by the tool metadata endpoint.
type ToolMetadataResponse struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	SecurityZone string `json:"security_zone"`
	RiskWarning  string `json:"risk_warning,omitempty"`
}

// ToolMetadataHandler serves tool metadata for the admin dashboard.
type ToolMetadataHandler struct {
	provider ToolMetadataProvider
}

// NewToolMetadataHandler creates a new handler.
func NewToolMetadataHandler(provider ToolMetadataProvider) *ToolMetadataHandler {
	return &ToolMetadataHandler{provider: provider}
}

// List returns metadata for all known tools.
func (h *ToolMetadataHandler) List(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.provider.GetAllToolMetadata())
}

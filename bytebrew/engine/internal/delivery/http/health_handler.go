package http

import (
	"encoding/json"
	"net/http"
	"time"
)

// HealthResponse is the JSON body returned by the health endpoint.
type HealthResponse struct {
	Status      string `json:"status"`
	Version     string `json:"version"`
	Uptime      string `json:"uptime"`
	AgentsCount int    `json:"agents_count"`
}

// AgentCounter provides a count of currently registered agents.
type AgentCounter interface {
	Count() int
}

// HealthHandler serves GET /api/v1/health.
type HealthHandler struct {
	version      string
	startedAt    time.Time
	agentCounter AgentCounter
}

// NewHealthHandler creates a HealthHandler.
func NewHealthHandler(version string, agentCounter AgentCounter) *HealthHandler {
	return &HealthHandler{
		version:      version,
		startedAt:    time.Now(),
		agentCounter: agentCounter,
	}
}

// ServeHTTP handles the health check request.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:      "ok",
		Version:     h.version,
		Uptime:      time.Since(h.startedAt).Round(time.Second).String(),
		AgentsCount: h.agentCounter.Count(),
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

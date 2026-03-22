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
	Database    string `json:"database,omitempty"`
}

// AgentCounter provides a count of currently registered agents.
type AgentCounter interface {
	Count() int
}

// DBPinger checks database connectivity.
type DBPinger interface {
	Ping() error
}

// HealthHandler serves GET /api/v1/health.
type HealthHandler struct {
	version      string
	startedAt    time.Time
	agentCounter AgentCounter
	dbPinger     DBPinger // optional, nil if no DB
}

// NewHealthHandler creates a HealthHandler.
func NewHealthHandler(version string, agentCounter AgentCounter) *HealthHandler {
	return &HealthHandler{
		version:      version,
		startedAt:    time.Now(),
		agentCounter: agentCounter,
	}
}

// SetDBPinger sets the database pinger for health checks.
func (h *HealthHandler) SetDBPinger(p DBPinger) { h.dbPinger = p }

// ServeHTTP handles the health check request.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	dbStatus := ""

	if h.dbPinger != nil {
		if err := h.dbPinger.Ping(); err != nil {
			status = "degraded"
			dbStatus = "error: " + err.Error()
		} else {
			dbStatus = "connected"
		}
	}

	resp := HealthResponse{
		Status:      status,
		Version:     h.version,
		Uptime:      time.Since(h.startedAt).Round(time.Second).String(),
		AgentsCount: h.agentCounter.Count(),
		Database:    dbStatus,
	}

	statusCode := http.StatusOK
	if status != "ok" {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(resp)
}

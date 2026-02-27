package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/syntheticinc/bytebrew/bytebrew-relay/internal/usecase/validate"
)

// LicenseValidator validates license JWTs.
type LicenseValidator interface {
	Execute(ctx context.Context, licenseJWT string) (*validate.Result, error)
}

// SessionManager manages active sessions.
type SessionManager interface {
	Register(userID, sessionID, tier string, seatsAllowed int) error
	Heartbeat(sessionID string) error
	Release(sessionID string) error
	TotalActive() int
}

// CacheCounter reports cache statistics.
type CacheCounter interface {
	Count() int
	IsWithinGrace() bool
}

// RelayHandler handles HTTP requests for the relay service.
type RelayHandler struct {
	validator LicenseValidator
	sessions  SessionManager
	cache     CacheCounter
}

// New creates a new RelayHandler.
func New(validator LicenseValidator, sessions SessionManager, cache CacheCounter) *RelayHandler {
	return &RelayHandler{
		validator: validator,
		sessions:  sessions,
		cache:     cache,
	}
}

// validateRequest is the request body for POST /relay/v1/validate.
type validateRequest struct {
	LicenseJWT string `json:"license_jwt"`
	UserID     string `json:"user_id"`
	SessionID  string `json:"session_id"`
}

// validateResponse is the response body for POST /relay/v1/validate.
type validateResponse struct {
	Valid   bool   `json:"valid"`
	Tier    string `json:"tier,omitempty"`
	Message string `json:"message,omitempty"`
}

// Validate handles POST /relay/v1/validate.
func (h *RelayHandler) Validate(w http.ResponseWriter, r *http.Request) {
	var req validateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, validateResponse{
			Valid:   false,
			Message: "invalid request body",
		})
		return
	}

	if req.LicenseJWT == "" {
		writeJSON(w, http.StatusBadRequest, validateResponse{
			Valid:   false,
			Message: "license_jwt is required",
		})
		return
	}

	result, err := h.validator.Execute(r.Context(), req.LicenseJWT)
	if err != nil {
		slog.ErrorContext(r.Context(), "validation error", "error", err)
		writeJSON(w, http.StatusInternalServerError, validateResponse{
			Valid:   false,
			Message: "internal error",
		})
		return
	}

	if !result.Valid {
		writeJSON(w, http.StatusOK, validateResponse{
			Valid:   false,
			Message: result.Message,
		})
		return
	}

	// Register session if session_id provided
	if req.SessionID != "" && req.UserID != "" {
		if err := h.sessions.Register(req.UserID, req.SessionID, result.Tier, result.SeatsAllowed); err != nil {
			slog.WarnContext(r.Context(), "session registration failed",
				"error", err,
				"user_id", req.UserID,
				"session_id", req.SessionID,
			)
			writeJSON(w, http.StatusConflict, validateResponse{
				Valid:   false,
				Message: "seat limit exceeded",
			})
			return
		}
	}

	writeJSON(w, http.StatusOK, validateResponse{
		Valid:   true,
		Tier:    result.Tier,
		Message: result.Message,
	})
}

// heartbeatRequest is the request body for POST /relay/v1/heartbeat.
type heartbeatRequest struct {
	SessionID string `json:"session_id"`
}

// heartbeatResponse is the response body for POST /relay/v1/heartbeat.
type heartbeatResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

// Heartbeat handles POST /relay/v1/heartbeat.
func (h *RelayHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	var req heartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, heartbeatResponse{OK: false, Message: "invalid request body"})
		return
	}

	if req.SessionID == "" {
		writeJSON(w, http.StatusBadRequest, heartbeatResponse{OK: false, Message: "session_id is required"})
		return
	}

	if err := h.sessions.Heartbeat(req.SessionID); err != nil {
		writeJSON(w, http.StatusNotFound, heartbeatResponse{OK: false, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, heartbeatResponse{OK: true})
}

// releaseRequest is the request body for POST /relay/v1/release.
type releaseRequest struct {
	SessionID string `json:"session_id"`
}

// releaseResponse is the response body for POST /relay/v1/release.
type releaseResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

// Release handles POST /relay/v1/release.
func (h *RelayHandler) Release(w http.ResponseWriter, r *http.Request) {
	var req releaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, releaseResponse{OK: false, Message: "invalid request body"})
		return
	}

	if req.SessionID == "" {
		writeJSON(w, http.StatusBadRequest, releaseResponse{OK: false, Message: "session_id is required"})
		return
	}

	if err := h.sessions.Release(req.SessionID); err != nil {
		writeJSON(w, http.StatusNotFound, releaseResponse{OK: false, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, releaseResponse{OK: true})
}

// statusResponse is the response body for GET /relay/v1/status.
type statusResponse struct {
	Status            string `json:"status"`
	CloudAPIConnected bool   `json:"cloud_api_connected"`
	CachedLicenses    int    `json:"cached_licenses"`
	ActiveSessions    int    `json:"active_sessions"`
}

// Status handles GET /relay/v1/status.
func (h *RelayHandler) Status(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, statusResponse{
		Status:            "ok",
		CloudAPIConnected: h.cache.IsWithinGrace(),
		CachedLicenses:    h.cache.Count(),
		ActiveSessions:    h.sessions.TotalActive(),
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to write response", "error", err)
	}
}

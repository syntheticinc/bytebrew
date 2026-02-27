package http

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/internal/delivery/http/middleware"
	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/internal/usecase/activate"
	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/internal/usecase/refresh_license"
)

type activateUsecase interface {
	Execute(ctx context.Context, input activate.Input) (*activate.Output, error)
}

type refreshLicenseUsecase interface {
	Execute(ctx context.Context, input refresh_license.Input) (*refresh_license.Output, error)
}

// LicenseHandler handles license endpoints.
type LicenseHandler struct {
	activateUC activateUsecase
	refreshUC  refreshLicenseUsecase
}

// NewLicenseHandler creates a new LicenseHandler.
func NewLicenseHandler(activateUC activateUsecase, refreshUC refreshLicenseUsecase) *LicenseHandler {
	return &LicenseHandler{
		activateUC: activateUC,
		refreshUC:  refreshUC,
	}
}

type licenseResponse struct {
	License string `json:"license"`
}

type refreshLicenseRequest struct {
	CurrentLicense string `json:"current_license"`
}

// Activate handles POST /license/activate.
func (h *LicenseHandler) Activate(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	email := middleware.GetEmail(r.Context())

	out, err := h.activateUC.Execute(r.Context(), activate.Input{
		UserID: userID,
		Email:  email,
	})
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusOK, licenseResponse{License: out.LicenseJWT})
}

// Refresh handles POST /license/refresh.
func (h *LicenseHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	email := middleware.GetEmail(r.Context())

	var req refreshLicenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, invalidBodyError)
		return
	}

	out, err := h.refreshUC.Execute(r.Context(), refresh_license.Input{
		UserID:         userID,
		Email:          email,
		CurrentLicense: req.CurrentLicense,
	})
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusOK, licenseResponse{License: out.LicenseJWT})
}

// Status handles GET /license/status.
// Decodes the license JWT payload without cryptographic verification.
// This is intentional: the endpoint is behind auth for access control,
// but shows decoded claims for client display purposes.
// Verification is not needed because the client already trusts the token it holds.
func (h *LicenseHandler) Status(w http.ResponseWriter, r *http.Request) {
	license := r.URL.Query().Get("license")
	if license == "" {
		Error(w, invalidLicenseParamError)
		return
	}

	parts := strings.SplitN(license, ".", 3)
	if len(parts) != 3 {
		Error(w, malformedJWTError)
		return
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		Error(w, malformedJWTError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprintf(w, `{"data":%s}`, payload); err != nil {
		slog.ErrorContext(r.Context(), "failed to write status response", "error", err)
	}
}

// Download handles GET /license/download.
// Returns the license JWT as a downloadable file.
func (h *LicenseHandler) Download(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	email := middleware.GetEmail(r.Context())

	out, err := h.activateUC.Execute(r.Context(), activate.Input{
		UserID: userID,
		Email:  email,
	})
	if err != nil {
		Error(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/jwt")
	w.Header().Set("Content-Disposition", `attachment; filename="license.jwt"`)
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(out.LicenseJWT)); err != nil {
		slog.ErrorContext(r.Context(), "failed to write download response", "error", err)
	}
}

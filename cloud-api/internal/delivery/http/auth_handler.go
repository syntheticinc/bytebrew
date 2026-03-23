package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/usecase/google_login"
	"github.com/syntheticinc/bytebrew/cloud-api/internal/usecase/login"
	"github.com/syntheticinc/bytebrew/cloud-api/internal/usecase/refresh_auth"
	"github.com/syntheticinc/bytebrew/cloud-api/internal/usecase/register"
	"github.com/syntheticinc/bytebrew/cloud-api/internal/usecase/resend_verification"
	"github.com/syntheticinc/bytebrew/cloud-api/internal/usecase/verify_email"
	"github.com/syntheticinc/bytebrew/cloud-api/pkg/errors"
)

type registerUsecase interface {
	Execute(ctx context.Context, input register.Input) (*register.Output, error)
}

type loginUsecase interface {
	Execute(ctx context.Context, input login.Input) (*login.Output, error)
}

type refreshAuthUsecase interface {
	Execute(ctx context.Context, input refresh_auth.Input) (*refresh_auth.Output, error)
}

type googleLoginUsecase interface {
	Execute(ctx context.Context, input google_login.Input) (*google_login.Output, error)
}

type verifyEmailUsecase interface {
	Execute(ctx context.Context, input verify_email.Input) (*verify_email.Output, error)
}

type resendVerificationUsecase interface {
	Execute(ctx context.Context, input resend_verification.Input) error
}

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	registerUC           registerUsecase
	loginUC              loginUsecase
	refreshAuthUC        refreshAuthUsecase
	googleLoginUC        googleLoginUsecase
	verifyEmailUC        verifyEmailUsecase
	resendVerificationUC resendVerificationUsecase
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(
	registerUC registerUsecase,
	loginUC loginUsecase,
	refreshAuthUC refreshAuthUsecase,
	googleLoginUC googleLoginUsecase,
	verifyEmailUC verifyEmailUsecase,
	resendVerificationUC resendVerificationUsecase,
) *AuthHandler {
	return &AuthHandler{
		registerUC:           registerUC,
		loginUC:              loginUC,
		refreshAuthUC:        refreshAuthUC,
		googleLoginUC:        googleLoginUC,
		verifyEmailUC:        verifyEmailUC,
		resendVerificationUC: resendVerificationUC,
	}
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	UserID       string `json:"user_id"`
}

type registerResponse struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

// Register handles POST /auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, invalidBodyError)
		return
	}

	out, err := h.registerUC.Execute(r.Context(), register.Input{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusCreated, registerResponse{
		UserID:  out.UserID,
		Message: out.Message,
	})
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, invalidBodyError)
		return
	}

	out, err := h.loginUC.Execute(r.Context(), login.Input{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusOK, authResponse{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
		UserID:       out.UserID,
	})
}

type googleLoginRequest struct {
	IDToken string `json:"id_token"`
}

// GoogleLogin handles POST /auth/google.
func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	if h.googleLoginUC == nil {
		Error(w, errors.Unavailable("google login is not configured", nil))
		return
	}

	var req googleLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, invalidBodyError)
		return
	}

	out, err := h.googleLoginUC.Execute(r.Context(), google_login.Input{
		IDToken: req.IDToken,
	})
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusOK, authResponse{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
		UserID:       out.UserID,
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type refreshResponse struct {
	AccessToken string `json:"access_token"`
}

// RefreshToken handles POST /auth/refresh.
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, invalidBodyError)
		return
	}

	out, err := h.refreshAuthUC.Execute(r.Context(), refresh_auth.Input{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusOK, refreshResponse{
		AccessToken: out.AccessToken,
	})
}

type verifyEmailRequest struct {
	Token string `json:"token"`
}

// VerifyEmail handles POST /auth/verify-email.
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req verifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, invalidBodyError)
		return
	}

	out, err := h.verifyEmailUC.Execute(r.Context(), verify_email.Input{
		Token: req.Token,
	})
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusOK, authResponse{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
		UserID:       out.UserID,
	})
}

type resendVerificationRequest struct {
	Email string `json:"email"`
}

// ResendVerification handles POST /auth/resend-verification.
func (h *AuthHandler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	var req resendVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, invalidBodyError)
		return
	}

	err := h.resendVerificationUC.Execute(r.Context(), resend_verification.Input{
		Email: req.Email,
	})
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusOK, map[string]string{
		"message": "if an account exists with this email, a verification link has been sent",
	})
}

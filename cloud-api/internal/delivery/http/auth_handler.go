package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/usecase/login"
	"github.com/syntheticinc/bytebrew/cloud-api/internal/usecase/refresh_auth"
	"github.com/syntheticinc/bytebrew/cloud-api/internal/usecase/register"
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

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	registerUC    registerUsecase
	loginUC       loginUsecase
	refreshAuthUC refreshAuthUsecase
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(registerUC registerUsecase, loginUC loginUsecase, refreshAuthUC refreshAuthUsecase) *AuthHandler {
	return &AuthHandler{
		registerUC:    registerUC,
		loginUC:       loginUC,
		refreshAuthUC: refreshAuthUC,
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

	JSON(w, http.StatusCreated, authResponse{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
		UserID:       out.UserID,
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

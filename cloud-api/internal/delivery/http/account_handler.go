package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/delivery/http/middleware"
	"github.com/syntheticinc/bytebrew/cloud-api/internal/usecase/change_password"
	"github.com/syntheticinc/bytebrew/cloud-api/internal/usecase/delete_account"
	"github.com/syntheticinc/bytebrew/cloud-api/internal/usecase/forgot_password"
	"github.com/syntheticinc/bytebrew/cloud-api/internal/usecase/reset_password"
)

type changePasswordUsecase interface {
	Execute(ctx context.Context, input change_password.Input) error
}

type deleteAccountUsecase interface {
	Execute(ctx context.Context, input delete_account.Input) error
}

type forgotPasswordUsecase interface {
	Execute(ctx context.Context, input forgot_password.Input) error
}

type resetPasswordUsecase interface {
	Execute(ctx context.Context, input reset_password.Input) error
}

// AccountHandler handles account management endpoints.
type AccountHandler struct {
	changePasswordUC changePasswordUsecase
	deleteAccountUC  deleteAccountUsecase
	forgotPasswordUC forgotPasswordUsecase
	resetPasswordUC  resetPasswordUsecase
}

// NewAccountHandler creates a new AccountHandler.
func NewAccountHandler(
	changePasswordUC changePasswordUsecase,
	deleteAccountUC deleteAccountUsecase,
	forgotPasswordUC forgotPasswordUsecase,
	resetPasswordUC resetPasswordUsecase,
) *AccountHandler {
	return &AccountHandler{
		changePasswordUC: changePasswordUC,
		deleteAccountUC:  deleteAccountUC,
		forgotPasswordUC: forgotPasswordUC,
		resetPasswordUC:  resetPasswordUC,
	}
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangePassword handles POST /auth/change-password.
func (h *AccountHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, invalidBodyError)
		return
	}

	userID := middleware.GetUserID(r.Context())
	err := h.changePasswordUC.Execute(r.Context(), change_password.Input{
		UserID:          userID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	})
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "password changed"})
}

type deleteAccountRequest struct {
	Password string `json:"password"`
}

// DeleteAccount handles DELETE /users/me.
func (h *AccountHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	var req deleteAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, invalidBodyError)
		return
	}

	userID := middleware.GetUserID(r.Context())
	err := h.deleteAccountUC.Execute(r.Context(), delete_account.Input{
		UserID:   userID,
		Password: req.Password,
	})
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "account deleted"})
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

// ForgotPassword handles POST /auth/forgot-password.
func (h *AccountHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, invalidBodyError)
		return
	}

	err := h.forgotPasswordUC.Execute(r.Context(), forgot_password.Input{
		Email: req.Email,
	})
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "if an account exists with this email, you will receive a reset link"})
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// ResetPassword handles POST /auth/reset-password.
func (h *AccountHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, invalidBodyError)
		return
	}

	err := h.resetPasswordUC.Execute(r.Context(), reset_password.Input{
		Token:       req.Token,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "password reset successfully"})
}

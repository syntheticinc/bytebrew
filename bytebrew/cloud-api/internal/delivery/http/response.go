package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/pkg/errors"
)

type successResponse struct {
	Data interface{} `json:"data"`
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSON writes a success JSON response.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(successResponse{Data: data}); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

// Error writes an error JSON response based on domain error code.
// For internal errors, logs the full error and returns a generic message to avoid leaking details.
func Error(w http.ResponseWriter, err error) {
	code := errors.GetCode(err)
	status := mapCodeToHTTPStatus(code)

	if code == errors.CodeInternal {
		slog.Error("internal error", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if encErr := json.NewEncoder(w).Encode(errorResponse{
		Error: errorBody{
			Code:    code,
			Message: errors.UserMessage(err),
		},
	}); encErr != nil {
		slog.Error("failed to encode error response", "error", encErr)
	}
}

func mapCodeToHTTPStatus(code string) int {
	switch code {
	case errors.CodeInvalidInput:
		return http.StatusBadRequest
	case errors.CodeUnauthorized:
		return http.StatusUnauthorized
	case errors.CodeForbidden:
		return http.StatusForbidden
	case errors.CodeNotFound:
		return http.StatusNotFound
	case errors.CodeAlreadyExists:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

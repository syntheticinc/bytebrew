package http

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// parseStringParam extracts a named URL path parameter and returns an error if it is empty.
func parseStringParam(r *http.Request, name string) (string, error) {
	val := chi.URLParam(r, name)
	if val == "" {
		return "", fmt.Errorf("missing required path parameter: %s", name)
	}
	return val, nil
}

// parseStringIDParam extracts the "id" URL path parameter and returns an error if it is empty.
func parseStringIDParam(r *http.Request) (string, error) {
	return parseStringParam(r, "id")
}

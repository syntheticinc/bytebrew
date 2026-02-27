package cloudapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- Helpers ---

// makeJWT creates a minimal unsigned JWT with the given payload for testing.
func makeJWT(payload map[string]interface{}) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	body, _ := json.Marshal(payload)
	claims := base64.RawURLEncoding.EncodeToString(body)
	sig := base64.RawURLEncoding.EncodeToString([]byte("sig"))
	return header + "." + claims + "." + sig
}

// --- ValidateLicense tests ---

func TestValidateLicense_Success(t *testing.T) {
	expiry := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/license/validate" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content type")
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Bearer test-token, got %s", r.Header.Get("Authorization"))
		}

		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body["license_jwt"] == "" {
			t.Fatal("expected license_jwt in request body")
		}

		resp := map[string]interface{}{
			"valid":  true,
			"tier":   "personal",
			"seats":  1,
			"expiry": expiry.Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := New(srv.URL, "test-token")
	jwt := makeJWT(map[string]interface{}{"seats_allowed": 1})

	info, err := client.ValidateLicense(context.Background(), jwt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Tier != "personal" {
		t.Fatalf("expected tier personal, got %s", info.Tier)
	}
	if info.SeatsAllowed != 1 {
		t.Fatalf("expected 1 seat, got %d", info.SeatsAllowed)
	}
	if !info.ExpiresAt.Equal(expiry) {
		t.Fatalf("expected expiry %v, got %v", expiry, info.ExpiresAt)
	}
}

func TestValidateLicense_SeatsFromJWT_Fallback(t *testing.T) {
	// Cloud API returns seats=0, so client should fall back to JWT parsing.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"valid": true,
			"tier":  "teams",
			"seats": 0, // Zero — triggers JWT fallback
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := New(srv.URL, "")
	jwt := makeJWT(map[string]interface{}{"seats_allowed": 5})

	info, err := client.ValidateLicense(context.Background(), jwt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.SeatsAllowed != 5 {
		t.Fatalf("expected 5 seats from JWT fallback, got %d", info.SeatsAllowed)
	}
}

func TestValidateLicense_InvalidLicense(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"valid":   false,
			"message": "license expired",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := New(srv.URL, "")
	jwt := makeJWT(map[string]interface{}{})

	_, err := client.ValidateLicense(context.Background(), jwt)
	if err == nil {
		t.Fatal("expected error for invalid license")
	}
	expectedMsg := "license invalid: license expired"
	if err.Error() != expectedMsg {
		t.Fatalf("expected error %q, got %q", expectedMsg, err.Error())
	}
}

func TestValidateLicense_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal server error")
	}))
	defer srv.Close()

	client := New(srv.URL, "")
	jwt := makeJWT(map[string]interface{}{})

	_, err := client.ValidateLicense(context.Background(), jwt)
	if err == nil {
		t.Fatal("expected error for server error response")
	}
	expected := fmt.Sprintf("cloud api returned %d: internal server error", http.StatusInternalServerError)
	if err.Error() != expected {
		t.Fatalf("expected error %q, got %q", expected, err.Error())
	}
}

func TestValidateLicense_NoAuthToken(t *testing.T) {
	var capturedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		resp := map[string]interface{}{
			"valid": true,
			"tier":  "trial",
			"seats": 1,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := New(srv.URL, "") // No auth token
	jwt := makeJWT(map[string]interface{}{})

	_, err := client.ValidateLicense(context.Background(), jwt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedAuth != "" {
		t.Fatalf("expected no Authorization header, got %q", capturedAuth)
	}
}

// --- RefreshLicense tests ---

func TestRefreshLicense_Success(t *testing.T) {
	newJWT := "new-refreshed-jwt-token"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/license/refresh" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body["current_license"] == "" {
			t.Fatal("expected current_license in request body")
		}

		resp := map[string]string{"license": newJWT}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := New(srv.URL, "auth-token")
	result, err := client.RefreshLicense(context.Background(), "old-jwt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != newJWT {
		t.Fatalf("expected %q, got %q", newJWT, result)
	}
}

func TestRefreshLicense_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprint(w, "bad gateway")
	}))
	defer srv.Close()

	client := New(srv.URL, "")
	_, err := client.RefreshLicense(context.Background(), "old-jwt")
	if err == nil {
		t.Fatal("expected error for server error response")
	}
	expected := fmt.Sprintf("cloud api returned %d: bad gateway", http.StatusBadGateway)
	if err.Error() != expected {
		t.Fatalf("expected error %q, got %q", expected, err.Error())
	}
}

func TestRefreshLicense_EmptyLicenseInResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{"license": ""}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := New(srv.URL, "")
	_, err := client.RefreshLicense(context.Background(), "old-jwt")
	if err == nil {
		t.Fatal("expected error for empty license in response")
	}
	if err.Error() != "empty license in refresh response" {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- seatsFromJWT tests ---

func TestSeatsFromJWT_ValidPayload(t *testing.T) {
	jwt := makeJWT(map[string]interface{}{"seats_allowed": 5})
	seats := seatsFromJWT(jwt)
	if seats != 5 {
		t.Fatalf("expected 5, got %d", seats)
	}
}

func TestSeatsFromJWT_ZeroSeats_DefaultsToOne(t *testing.T) {
	jwt := makeJWT(map[string]interface{}{"seats_allowed": 0})
	seats := seatsFromJWT(jwt)
	if seats != 1 {
		t.Fatalf("expected 1 (default), got %d", seats)
	}
}

func TestSeatsFromJWT_MissingField_DefaultsToOne(t *testing.T) {
	jwt := makeJWT(map[string]interface{}{"tier": "personal"})
	seats := seatsFromJWT(jwt)
	if seats != 1 {
		t.Fatalf("expected 1 (default), got %d", seats)
	}
}

func TestSeatsFromJWT_InvalidJWT_DefaultsToOne(t *testing.T) {
	tests := []struct {
		name string
		jwt  string
	}{
		{"empty string", ""},
		{"no dots", "notajwt"},
		{"one part", "header.payload"},
		{"invalid base64", "header.!!!invalid!!!.sig"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seats := seatsFromJWT(tt.jwt)
			if seats != 1 {
				t.Fatalf("expected 1 (default), got %d", seats)
			}
		})
	}
}

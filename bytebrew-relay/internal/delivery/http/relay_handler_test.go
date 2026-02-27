package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/syntheticinc/bytebrew/bytebrew-relay/internal/usecase/validate"
)

// --- Mocks ---

type mockLicenseValidator struct {
	result *validate.Result
	err    error
}

func (m *mockLicenseValidator) Execute(_ context.Context, _ string) (*validate.Result, error) {
	return m.result, m.err
}

type mockSessionManager struct {
	registerErr    error
	heartbeatErr   error
	releaseErr     error
	totalActive    int
	registerCalls  int
	heartbeatCalls int
	releaseCalls   int
}

func (m *mockSessionManager) Register(_, _, _ string, _ int) error {
	m.registerCalls++
	return m.registerErr
}

func (m *mockSessionManager) Heartbeat(_ string) error {
	m.heartbeatCalls++
	return m.heartbeatErr
}

func (m *mockSessionManager) Release(_ string) error {
	m.releaseCalls++
	return m.releaseErr
}

func (m *mockSessionManager) TotalActive() int {
	return m.totalActive
}

type mockCacheCounter struct {
	count       int
	withinGrace bool
}

func (m *mockCacheCounter) Count() int {
	return m.count
}

func (m *mockCacheCounter) IsWithinGrace() bool {
	return m.withinGrace
}

// --- Helpers ---

func postJSON(handler http.HandlerFunc, body interface{}) *httptest.ResponseRecorder {
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func getRequest(handler http.HandlerFunc) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func decodeJSON(t *testing.T, rr *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	if err := json.NewDecoder(rr.Body).Decode(v); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
}

// --- Validate tests ---

func TestValidate_ValidJWT_ReturnsOK(t *testing.T) {
	h := New(
		&mockLicenseValidator{
			result: &validate.Result{Valid: true, Tier: "personal", SeatsAllowed: 1},
		},
		&mockSessionManager{},
		&mockCacheCounter{},
	)

	body := validateRequest{LicenseJWT: "valid-jwt-token", UserID: "user-1", SessionID: "sess-1"}
	rr := postJSON(h.Validate, body)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp validateResponse
	decodeJSON(t, rr, &resp)
	if !resp.Valid {
		t.Fatal("expected valid=true")
	}
	if resp.Tier != "personal" {
		t.Fatalf("expected tier personal, got %s", resp.Tier)
	}
}

func TestValidate_EmptyJWT_Returns400(t *testing.T) {
	h := New(
		&mockLicenseValidator{},
		&mockSessionManager{},
		&mockCacheCounter{},
	)

	body := validateRequest{LicenseJWT: ""}
	rr := postJSON(h.Validate, body)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	var resp validateResponse
	decodeJSON(t, rr, &resp)
	if resp.Valid {
		t.Fatal("expected valid=false for empty JWT")
	}
	if resp.Message == "" {
		t.Fatal("expected error message for empty JWT")
	}
}

func TestValidate_InvalidBody_Returns400(t *testing.T) {
	h := New(
		&mockLicenseValidator{},
		&mockSessionManager{},
		&mockCacheCounter{},
	)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Validate(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestValidate_ValidatorError_Returns500(t *testing.T) {
	h := New(
		&mockLicenseValidator{err: fmt.Errorf("cloud api down")},
		&mockSessionManager{},
		&mockCacheCounter{},
	)

	body := validateRequest{LicenseJWT: "some-jwt-token"}
	rr := postJSON(h.Validate, body)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}

	var resp validateResponse
	decodeJSON(t, rr, &resp)
	if resp.Valid {
		t.Fatal("expected valid=false on validator error")
	}
}

func TestValidate_InvalidLicense_Returns200WithValidFalse(t *testing.T) {
	h := New(
		&mockLicenseValidator{
			result: &validate.Result{Valid: false, Message: "license expired"},
		},
		&mockSessionManager{},
		&mockCacheCounter{},
	)

	body := validateRequest{LicenseJWT: "expired-jwt-ok"}
	rr := postJSON(h.Validate, body)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp validateResponse
	decodeJSON(t, rr, &resp)
	if resp.Valid {
		t.Fatal("expected valid=false for expired license")
	}
	if resp.Message != "license expired" {
		t.Fatalf("expected message 'license expired', got %q", resp.Message)
	}
}

func TestValidate_SessionRegistrationConflict_Returns409(t *testing.T) {
	h := New(
		&mockLicenseValidator{
			result: &validate.Result{Valid: true, Tier: "personal", SeatsAllowed: 1},
		},
		&mockSessionManager{registerErr: fmt.Errorf("seat limit reached: 1/1")},
		&mockCacheCounter{},
	)

	body := validateRequest{LicenseJWT: "valid-jwt-ok12", UserID: "user-1", SessionID: "sess-2"}
	rr := postJSON(h.Validate, body)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rr.Code)
	}

	var resp validateResponse
	decodeJSON(t, rr, &resp)
	if resp.Valid {
		t.Fatal("expected valid=false on session conflict")
	}
}

func TestValidate_NoSessionID_SkipsRegistration(t *testing.T) {
	sessions := &mockSessionManager{}
	h := New(
		&mockLicenseValidator{
			result: &validate.Result{Valid: true, Tier: "personal", SeatsAllowed: 1},
		},
		sessions,
		&mockCacheCounter{},
	)

	body := validateRequest{LicenseJWT: "valid-jwt-ok12"}
	rr := postJSON(h.Validate, body)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if sessions.registerCalls != 0 {
		t.Fatalf("expected 0 register calls, got %d", sessions.registerCalls)
	}
}

// --- Heartbeat tests ---

func TestHeartbeat_ValidSession_ReturnsOK(t *testing.T) {
	h := New(
		&mockLicenseValidator{},
		&mockSessionManager{},
		&mockCacheCounter{},
	)

	body := heartbeatRequest{SessionID: "sess-1"}
	rr := postJSON(h.Heartbeat, body)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp heartbeatResponse
	decodeJSON(t, rr, &resp)
	if !resp.OK {
		t.Fatal("expected ok=true for valid heartbeat")
	}
}

func TestHeartbeat_EmptySessionID_Returns400(t *testing.T) {
	h := New(
		&mockLicenseValidator{},
		&mockSessionManager{},
		&mockCacheCounter{},
	)

	body := heartbeatRequest{SessionID: ""}
	rr := postJSON(h.Heartbeat, body)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHeartbeat_UnknownSession_Returns404(t *testing.T) {
	h := New(
		&mockLicenseValidator{},
		&mockSessionManager{heartbeatErr: fmt.Errorf("session not found: unknown-sess")},
		&mockCacheCounter{},
	)

	body := heartbeatRequest{SessionID: "unknown-sess"}
	rr := postJSON(h.Heartbeat, body)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}

	var resp heartbeatResponse
	decodeJSON(t, rr, &resp)
	if resp.OK {
		t.Fatal("expected ok=false for unknown session")
	}
}

func TestHeartbeat_InvalidBody_Returns400(t *testing.T) {
	h := New(
		&mockLicenseValidator{},
		&mockSessionManager{},
		&mockCacheCounter{},
	)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("{")))
	rr := httptest.NewRecorder()
	h.Heartbeat(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

// --- Release tests ---

func TestRelease_ValidSession_ReturnsOK(t *testing.T) {
	h := New(
		&mockLicenseValidator{},
		&mockSessionManager{},
		&mockCacheCounter{},
	)

	body := releaseRequest{SessionID: "sess-1"}
	rr := postJSON(h.Release, body)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp releaseResponse
	decodeJSON(t, rr, &resp)
	if !resp.OK {
		t.Fatal("expected ok=true for valid release")
	}
}

func TestRelease_EmptySessionID_Returns400(t *testing.T) {
	h := New(
		&mockLicenseValidator{},
		&mockSessionManager{},
		&mockCacheCounter{},
	)

	body := releaseRequest{SessionID: ""}
	rr := postJSON(h.Release, body)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestRelease_UnknownSession_Returns404(t *testing.T) {
	h := New(
		&mockLicenseValidator{},
		&mockSessionManager{releaseErr: fmt.Errorf("session not found: unknown-sess")},
		&mockCacheCounter{},
	)

	body := releaseRequest{SessionID: "unknown-sess"}
	rr := postJSON(h.Release, body)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}

	var resp releaseResponse
	decodeJSON(t, rr, &resp)
	if resp.OK {
		t.Fatal("expected ok=false for unknown session")
	}
}

func TestRelease_InvalidBody_Returns400(t *testing.T) {
	h := New(
		&mockLicenseValidator{},
		&mockSessionManager{},
		&mockCacheCounter{},
	)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("")))
	rr := httptest.NewRecorder()
	h.Release(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

// --- Status tests ---

func TestStatus_ReturnsHealth(t *testing.T) {
	h := New(
		&mockLicenseValidator{},
		&mockSessionManager{totalActive: 3},
		&mockCacheCounter{count: 5, withinGrace: true},
	)

	rr := getRequest(h.Status)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp statusResponse
	decodeJSON(t, rr, &resp)

	if resp.Status != "ok" {
		t.Fatalf("expected status ok, got %s", resp.Status)
	}
	if !resp.CloudAPIConnected {
		t.Fatal("expected cloud_api_connected=true")
	}
	if resp.CachedLicenses != 5 {
		t.Fatalf("expected 5 cached licenses, got %d", resp.CachedLicenses)
	}
	if resp.ActiveSessions != 3 {
		t.Fatalf("expected 3 active sessions, got %d", resp.ActiveSessions)
	}
}

func TestStatus_NoGrace_CloudAPIDisconnected(t *testing.T) {
	h := New(
		&mockLicenseValidator{},
		&mockSessionManager{totalActive: 0},
		&mockCacheCounter{count: 0, withinGrace: false},
	)

	rr := getRequest(h.Status)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp statusResponse
	decodeJSON(t, rr, &resp)

	if resp.CloudAPIConnected {
		t.Fatal("expected cloud_api_connected=false when not within grace")
	}
}

func TestStatus_ResponseContentType(t *testing.T) {
	h := New(
		&mockLicenseValidator{},
		&mockSessionManager{},
		&mockCacheCounter{},
	)

	rr := getRequest(h.Status)

	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %s", ct)
	}
}

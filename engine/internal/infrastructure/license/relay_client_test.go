package license

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelayClient_ValidateViaRelay_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/relay/v1/validate", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var req relayValidateRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "test-jwt", req.LicenseJWT)
		assert.Equal(t, "user-1", req.UserID)
		assert.Equal(t, "session-1", req.SessionID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(relayValidateResponse{
			Valid: true,
			Tier:  "personal",
		})
	}))
	defer server.Close()

	client := NewRelayClient(server.URL)
	info := client.ValidateViaRelay(context.Background(), "test-jwt", "user-1", "session-1")

	assert.Equal(t, domain.LicenseActive, info.Status)
	assert.Equal(t, domain.TierPersonal, info.Tier)
	assert.True(t, info.Features.FullAutonomy)
}

func TestRelayClient_ValidateViaRelay_Invalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(relayValidateResponse{
			Valid:   false,
			Message: "license expired",
		})
	}))
	defer server.Close()

	client := NewRelayClient(server.URL)
	info := client.ValidateViaRelay(context.Background(), "expired-jwt", "user-1", "session-1")

	assert.Equal(t, domain.LicenseBlocked, info.Status)
}

func TestRelayClient_ValidateViaRelay_GracePeriod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(relayValidateResponse{
			Valid:   true,
			Tier:    "personal",
			Message: "cloud api unavailable, using cached license (grace period)",
		})
	}))
	defer server.Close()

	client := NewRelayClient(server.URL)
	info := client.ValidateViaRelay(context.Background(), "test-jwt", "user-1", "session-1")

	assert.Equal(t, domain.LicenseGrace, info.Status)
	assert.Equal(t, domain.TierPersonal, info.Tier)
}

func TestRelayClient_ValidateViaRelay_RelayUnavailable_NoCache(t *testing.T) {
	client := NewRelayClient("http://localhost:1") // unreachable

	info := client.ValidateViaRelay(context.Background(), "test-jwt", "user-1", "session-1")

	assert.Equal(t, domain.LicenseBlocked, info.Status)
}

func TestRelayClient_ValidateViaRelay_RelayUnavailable_WithCache(t *testing.T) {
	// First call succeeds
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(relayValidateResponse{
				Valid: true,
				Tier:  "teams",
			})
			return
		}
		// Subsequent calls fail
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewRelayClient(server.URL)

	// First call - succeeds and caches
	info := client.ValidateViaRelay(context.Background(), "test-jwt", "user-1", "session-1")
	assert.Equal(t, domain.LicenseActive, info.Status)
	assert.Equal(t, domain.TierTeams, info.Tier)

	// Second call - relay returns error, should use cache
	info2 := client.ValidateViaRelay(context.Background(), "test-jwt", "user-1", "session-1")
	assert.Equal(t, domain.LicenseActive, info2.Status)
	assert.Equal(t, domain.TierTeams, info2.Tier)
}

func TestRelayClient_Heartbeat_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/relay/v1/validate" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(relayValidateResponse{Valid: true, Tier: "personal"})
			return
		}

		assert.Equal(t, "/relay/v1/heartbeat", r.URL.Path)

		var req relayHeartbeatRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "session-1", req.SessionID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(relayHeartbeatResponse{OK: true})
	}))
	defer server.Close()

	client := NewRelayClient(server.URL)
	client.ValidateViaRelay(context.Background(), "jwt", "user-1", "session-1")

	err := client.Heartbeat(context.Background())
	assert.NoError(t, err)
}

func TestRelayClient_Heartbeat_NoSession(t *testing.T) {
	client := NewRelayClient("http://localhost:8080")

	err := client.Heartbeat(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no active session")
}

func TestRelayClient_Release_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/relay/v1/validate" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(relayValidateResponse{Valid: true, Tier: "personal"})
			return
		}

		assert.Equal(t, "/relay/v1/release", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}))
	defer server.Close()

	client := NewRelayClient(server.URL)
	client.ValidateViaRelay(context.Background(), "jwt", "user-1", "session-1")

	err := client.Release(context.Background())
	assert.NoError(t, err)
}

func TestFeaturesForTier(t *testing.T) {
	tests := []struct {
		tier           string
		fullAutonomy   bool
		parallelAgents int
	}{
		{"personal", true, -1},
		{"teams", true, -1},
		{"trial", true, 2},
		{"unknown", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.tier, func(t *testing.T) {
			f := featuresForTier(tt.tier)
			assert.Equal(t, tt.fullAutonomy, f.FullAutonomy)
			assert.Equal(t, tt.parallelAgents, f.ParallelAgents)
		})
	}
}

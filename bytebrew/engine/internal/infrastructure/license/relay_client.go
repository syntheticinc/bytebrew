package license

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
)

// RelayClient validates licenses through a ByteBrew Relay service.
// Used in On-Premises deployments where bytebrew-srv does not have direct
// internet access to Cloud API.
type RelayClient struct {
	baseURL    string
	httpClient *http.Client
	mu         sync.RWMutex
	cached     *domain.LicenseInfo
	sessionID  string
}

// NewRelayClient creates a new relay client.
func NewRelayClient(relayAddress string) *RelayClient {
	return &RelayClient{
		baseURL: strings.TrimRight(relayAddress, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// relayValidateRequest is the request body for POST /relay/v1/validate.
type relayValidateRequest struct {
	LicenseJWT string `json:"license_jwt"`
	UserID     string `json:"user_id,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
}

// relayValidateResponse is the response body from POST /relay/v1/validate.
type relayValidateResponse struct {
	Valid   bool   `json:"valid"`
	Tier    string `json:"tier"`
	Message string `json:"message"`
}

// relayHeartbeatRequest is the request body for POST /relay/v1/heartbeat.
type relayHeartbeatRequest struct {
	SessionID string `json:"session_id"`
}

// relayHeartbeatResponse is the response from POST /relay/v1/heartbeat.
type relayHeartbeatResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// ValidateViaRelay validates a license JWT through the relay service.
func (c *RelayClient) ValidateViaRelay(ctx context.Context, licenseJWT, userID, sessionID string) *domain.LicenseInfo {
	c.mu.Lock()
	c.sessionID = sessionID
	c.mu.Unlock()

	body := relayValidateRequest{
		LicenseJWT: licenseJWT,
		UserID:     userID,
		SessionID:  sessionID,
	}
	data, err := json.Marshal(body)
	if err != nil {
		slog.ErrorContext(ctx, "failed to marshal relay request", "error", err)
		return c.fallbackOrBlocked()
	}

	url := c.baseURL + "/relay/v1/validate"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		slog.ErrorContext(ctx, "failed to create relay request", "error", err)
		return c.fallbackOrBlocked()
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.WarnContext(ctx, "relay unavailable, using cached license", "error", err)
		return c.fallbackOrBlocked()
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		slog.WarnContext(ctx, "relay returned error",
			"status", resp.StatusCode,
			"body", string(respBody),
		)
		return c.fallbackOrBlocked()
	}

	var result relayValidateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		slog.WarnContext(ctx, "failed to decode relay response", "error", err)
		return c.fallbackOrBlocked()
	}

	if !result.Valid {
		slog.WarnContext(ctx, "relay rejected license", "message", result.Message)
		return domain.BlockedLicense()
	}

	info := &domain.LicenseInfo{
		Tier:   domain.LicenseTier(result.Tier),
		Status: domain.LicenseActive,
		Features: featuresForTier(result.Tier),
	}

	if result.Message != "" {
		// Grace period warning from relay
		info.Status = domain.LicenseGrace
		slog.WarnContext(ctx, "relay license in grace", "message", result.Message)
	}

	c.mu.Lock()
	c.cached = info
	c.mu.Unlock()
	return info
}

// Heartbeat sends a heartbeat to the relay to keep the session alive.
func (c *RelayClient) Heartbeat(ctx context.Context) error {
	c.mu.RLock()
	sid := c.sessionID
	c.mu.RUnlock()

	if sid == "" {
		return fmt.Errorf("no active session")
	}

	body := relayHeartbeatRequest{SessionID: sid}
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal heartbeat: %w", err)
	}

	url := c.baseURL + "/relay/v1/heartbeat"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create heartbeat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("relay heartbeat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("relay heartbeat returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result relayHeartbeatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode heartbeat response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("relay heartbeat failed: %s", result.Message)
	}

	return nil
}

// Release releases the session on the relay.
func (c *RelayClient) Release(ctx context.Context) error {
	c.mu.RLock()
	sid := c.sessionID
	c.mu.RUnlock()

	if sid == "" {
		return nil
	}

	body := map[string]string{"session_id": sid}
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal release: %w", err)
	}

	url := c.baseURL + "/relay/v1/release"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create release request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("relay release: %w", err)
	}
	defer resp.Body.Close()

	c.mu.Lock()
	c.sessionID = ""
	c.mu.Unlock()
	return nil
}

// fallbackOrBlocked returns cached license if available, otherwise blocked.
func (c *RelayClient) fallbackOrBlocked() *domain.LicenseInfo {
	c.mu.RLock()
	cached := c.cached
	c.mu.RUnlock()

	if cached != nil {
		slog.Warn("using cached license from relay")
		return cached
	}
	return domain.BlockedLicense()
}

// featuresForTier returns default features based on tier.
func featuresForTier(tier string) domain.LicenseFeatures {
	switch tier {
	case "personal", "teams":
		return domain.LicenseFeatures{
			FullAutonomy:     true,
			ParallelAgents:   -1,
			ExploreCodebase:  true,
			TraceSymbol:      true,
			CodebaseIndexing: true,
		}
	case "trial":
		return domain.LicenseFeatures{
			FullAutonomy:     true,
			ParallelAgents:   2,
			ExploreCodebase:  true,
			TraceSymbol:      true,
			CodebaseIndexing: false,
		}
	default:
		return domain.LicenseFeatures{}
	}
}

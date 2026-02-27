package cloudapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// LicenseInfo holds the result of a Cloud API license validation.
type LicenseInfo struct {
	Tier         string
	SeatsAllowed int
	ExpiresAt    time.Time
}

// Client communicates with the Vector Cloud API to validate and refresh licenses.
type Client struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

// New creates a new Cloud API client.
func New(baseURL, authToken string) *Client {
	return &Client{
		baseURL:   strings.TrimRight(baseURL, "/"),
		authToken: authToken,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ValidateLicense calls Cloud API to validate a license JWT.
// Returns tier, seats allowed, expiry time, or error.
func (c *Client) ValidateLicense(ctx context.Context, licenseJWT string) (*LicenseInfo, error) {
	body := map[string]string{
		"license_jwt": licenseJWT,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/api/v1/license/validate"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call cloud api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cloud api returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Valid    bool   `json:"valid"`
		Tier     string `json:"tier"`
		Seats    int    `json:"seats"`
		Expiry   string `json:"expiry"` // RFC3339
		Message  string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if !result.Valid {
		return nil, fmt.Errorf("license invalid: %s", result.Message)
	}

	info := &LicenseInfo{
		Tier:         result.Tier,
		SeatsAllowed: result.Seats,
	}
	if result.Expiry != "" {
		t, err := time.Parse(time.RFC3339, result.Expiry)
		if err != nil {
			slog.WarnContext(ctx, "failed to parse expiry from cloud api", "expiry", result.Expiry, "error", err)
		} else {
			info.ExpiresAt = t
		}
	}

	// If Cloud API doesn't return seats, decode from JWT payload
	if info.SeatsAllowed == 0 {
		info.SeatsAllowed = seatsFromJWT(licenseJWT)
	}

	return info, nil
}

// RefreshLicense calls Cloud API to get a refreshed license JWT.
func (c *Client) RefreshLicense(ctx context.Context, currentJWT string) (string, error) {
	body := map[string]string{
		"current_license": currentJWT,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/api/v1/license/refresh"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call cloud api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("cloud api returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		License string `json:"license"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if result.License == "" {
		return "", fmt.Errorf("empty license in refresh response")
	}

	return result.License, nil
}

// seatsFromJWT extracts seats_allowed from the JWT payload without verifying the signature.
// Returns 1 as default if parsing fails or seats_allowed is absent/zero.
func seatsFromJWT(jwtStr string) int {
	parts := strings.Split(jwtStr, ".")
	if len(parts) != 3 {
		return 1
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 1
	}

	var claims struct {
		SeatsAllowed int `json:"seats_allowed"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return 1
	}

	if claims.SeatsAllowed > 0 {
		return claims.SeatsAllowed
	}
	return 1
}

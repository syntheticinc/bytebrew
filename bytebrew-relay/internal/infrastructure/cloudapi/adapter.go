package cloudapi

import (
	"context"

	"github.com/syntheticinc/bytebrew/bytebrew-relay/internal/usecase/refresh"
	"github.com/syntheticinc/bytebrew/bytebrew-relay/internal/usecase/validate"
)

// ValidateAdapter adapts Client to validate.CloudAPIClient interface.
type ValidateAdapter struct {
	client *Client
}

// NewValidateAdapter creates a new adapter for the validate usecase.
func NewValidateAdapter(client *Client) *ValidateAdapter {
	return &ValidateAdapter{client: client}
}

// ValidateLicense delegates to the underlying Client and maps the result.
func (a *ValidateAdapter) ValidateLicense(ctx context.Context, licenseJWT string) (*validate.CloudAPIResult, error) {
	info, err := a.client.ValidateLicense(ctx, licenseJWT)
	if err != nil {
		return nil, err
	}
	return &validate.CloudAPIResult{
		Tier:         info.Tier,
		SeatsAllowed: info.SeatsAllowed,
		ExpiresAt:    info.ExpiresAt,
	}, nil
}

// RefreshAdapter adapts Client to refresh.CloudAPIClient interface.
type RefreshAdapter struct {
	client *Client
}

// NewRefreshAdapter creates a new adapter for the refresh usecase.
func NewRefreshAdapter(client *Client) *RefreshAdapter {
	return &RefreshAdapter{client: client}
}

// RefreshLicense delegates to the underlying Client.
func (a *RefreshAdapter) RefreshLicense(ctx context.Context, currentJWT string) (string, error) {
	return a.client.RefreshLicense(ctx, currentJWT)
}

// ValidateLicense delegates to the underlying Client and maps the result.
func (a *RefreshAdapter) ValidateLicense(ctx context.Context, licenseJWT string) (*refresh.ValidationResult, error) {
	info, err := a.client.ValidateLicense(ctx, licenseJWT)
	if err != nil {
		return nil, err
	}
	return &refresh.ValidationResult{
		Tier:         info.Tier,
		SeatsAllowed: info.SeatsAllowed,
		ExpiresAt:    info.ExpiresAt,
	}, nil
}

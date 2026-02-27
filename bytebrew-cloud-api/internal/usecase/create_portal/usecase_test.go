package create_portal

import (
	"context"
	"fmt"
	"testing"

	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mocks ---

type mockCustomerReader struct {
	customer *domain.StripeCustomer
	err      error
}

func (m *mockCustomerReader) GetByUserID(_ context.Context, _ string) (*domain.StripeCustomer, error) {
	return m.customer, m.err
}

type mockPortalCreator struct {
	url string
	err error
}

func (m *mockPortalCreator) CreatePortalSession(_ context.Context, _, _ string) (string, error) {
	return m.url, m.err
}

// --- tests ---

func TestExecute(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		input          Input
		customerReader *mockCustomerReader
		portalCreator  *mockPortalCreator
		returnURL      string
		wantURL        string
		wantErr        string
		wantCode       string
	}{
		{
			name:     "empty user ID",
			input:    Input{UserID: ""},
			wantErr:  "user_id is required",
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:  "happy path — existing stripe customer",
			input: Input{UserID: "u-1"},
			customerReader: &mockCustomerReader{customer: &domain.StripeCustomer{
				UserID:     "u-1",
				CustomerID: "cus_123",
			}},
			portalCreator: &mockPortalCreator{url: "https://billing.stripe.com/portal"},
			returnURL:     "https://return.example.com",
			wantURL:       "https://billing.stripe.com/portal",
		},
		{
			name:           "no customer found",
			input:          Input{UserID: "u-1"},
			customerReader: &mockCustomerReader{customer: nil},
			wantErr:        "no active Stripe subscription",
			wantCode:       errors.CodeNotFound,
		},
		{
			name:           "customer reader error",
			input:          Input{UserID: "u-1"},
			customerReader: &mockCustomerReader{err: fmt.Errorf("db down")},
			wantErr:        "get stripe customer",
			wantCode:       errors.CodeInternal,
		},
		{
			name:  "portal creator error",
			input: Input{UserID: "u-1"},
			customerReader: &mockCustomerReader{customer: &domain.StripeCustomer{
				UserID:     "u-1",
				CustomerID: "cus_123",
			}},
			portalCreator: &mockPortalCreator{err: fmt.Errorf("stripe api error")},
			wantErr:       "create portal session",
			wantCode:      errors.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customerReader := tt.customerReader
			if customerReader == nil {
				customerReader = &mockCustomerReader{}
			}
			portalCreator := tt.portalCreator
			if portalCreator == nil {
				portalCreator = &mockPortalCreator{}
			}

			uc := New(customerReader, portalCreator, tt.returnURL)
			got, err := uc.Execute(ctx, tt.input)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				if tt.wantCode != "" {
					assert.True(t, errors.Is(err, tt.wantCode), "expected code %s, got %s", tt.wantCode, errors.GetCode(err))
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantURL, got.PortalURL)
		})
	}
}

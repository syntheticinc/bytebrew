package create_checkout

import (
	"context"
	"fmt"
	"testing"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/cloud-api/pkg/errors"
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

type mockCustomerSaver struct {
	calledUserID     string
	calledCustomerID string
	err              error
}

func (m *mockCustomerSaver) Upsert(_ context.Context, userID, customerID string) error {
	m.calledUserID = userID
	m.calledCustomerID = customerID
	return m.err
}

type mockCustomerCreator struct {
	customerID string
	err        error
}

func (m *mockCustomerCreator) CreateCustomer(_ context.Context, _ string, _ map[string]string) (string, error) {
	return m.customerID, m.err
}

type mockSessionCreator struct {
	url          string
	err          error
	calledParams CheckoutParams
}

func (m *mockSessionCreator) CreateCheckoutSession(_ context.Context, params CheckoutParams) (string, error) {
	m.calledParams = params
	return m.url, m.err
}

type mockPriceResolver struct {
	priceID string
	err     error
}

func (m *mockPriceResolver) PriceIDForPlan(_, _ string) (string, error) {
	return m.priceID, m.err
}

// --- tests ---

func TestExecute(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name            string
		input           Input
		customerReader  *mockCustomerReader
		customerSaver   *mockCustomerSaver
		customerCreator *mockCustomerCreator
		sessionCreator  *mockSessionCreator
		priceResolver   *mockPriceResolver
		trialDays       int64
		wantURL         string
		wantErr         string
		wantCode        string
	}{
		{
			name:     "empty user ID",
			input:    Input{UserID: "", Email: "test@example.com", Plan: "personal", Period: "monthly"},
			wantErr:  "user_id is required",
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:     "empty email",
			input:    Input{UserID: "u-1", Email: "", Plan: "personal", Period: "monthly"},
			wantErr:  "email is required",
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:     "invalid plan",
			input:    Input{UserID: "u-1", Email: "test@example.com", Plan: "invalid", Period: "monthly"},
			wantErr:  "invalid plan",
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:     "old plan name starter rejected",
			input:    Input{UserID: "u-1", Email: "test@example.com", Plan: "starter", Period: "monthly"},
			wantErr:  "invalid plan",
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:     "old plan name pro rejected",
			input:    Input{UserID: "u-1", Email: "test@example.com", Plan: "pro", Period: "monthly"},
			wantErr:  "invalid plan",
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:     "invalid period",
			input:    Input{UserID: "u-1", Email: "test@example.com", Plan: "personal", Period: "weekly"},
			wantErr:  "invalid period",
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:  "existing customer uses stored customer ID",
			input: Input{UserID: "u-1", Email: "test@example.com", Plan: "personal", Period: "monthly"},
			customerReader: &mockCustomerReader{customer: &domain.StripeCustomer{
				UserID:     "u-1",
				CustomerID: "cus_existing",
			}},
			customerSaver:   &mockCustomerSaver{},
			customerCreator: &mockCustomerCreator{},
			sessionCreator:  &mockSessionCreator{url: "https://checkout.stripe.com/session"},
			priceResolver:   &mockPriceResolver{priceID: "price_personal_monthly"},
			wantURL:         "https://checkout.stripe.com/session",
		},
		{
			name:            "new customer is created and saved",
			input:           Input{UserID: "u-1", Email: "new@example.com", Plan: "personal", Period: "annual"},
			customerReader:  &mockCustomerReader{customer: nil},
			customerSaver:   &mockCustomerSaver{},
			customerCreator: &mockCustomerCreator{customerID: "cus_new"},
			sessionCreator:  &mockSessionCreator{url: "https://checkout.stripe.com/new"},
			priceResolver:   &mockPriceResolver{priceID: "price_personal_annual"},
			wantURL:         "https://checkout.stripe.com/new",
		},
		{
			name:           "customer reader error",
			input:          Input{UserID: "u-1", Email: "test@example.com", Plan: "personal", Period: "monthly"},
			customerReader: &mockCustomerReader{err: fmt.Errorf("db down")},
			wantErr:        "get stripe customer",
			wantCode:       errors.CodeInternal,
		},
		{
			name:            "customer creator error",
			input:           Input{UserID: "u-1", Email: "test@example.com", Plan: "personal", Period: "monthly"},
			customerReader:  &mockCustomerReader{customer: nil},
			customerSaver:   &mockCustomerSaver{},
			customerCreator: &mockCustomerCreator{err: fmt.Errorf("stripe api error")},
			wantErr:         "create stripe customer",
			wantCode:        errors.CodeInternal,
		},
		{
			name:  "price resolver error",
			input: Input{UserID: "u-1", Email: "test@example.com", Plan: "teams", Period: "monthly"},
			customerReader: &mockCustomerReader{customer: &domain.StripeCustomer{
				UserID:     "u-1",
				CustomerID: "cus_existing",
			}},
			customerSaver: &mockCustomerSaver{},
			priceResolver: &mockPriceResolver{err: fmt.Errorf("no price configured for teams_monthly")},
			wantErr:       "invalid plan",
			wantCode:      errors.CodeInvalidInput,
		},
		{
			name:  "session creator error",
			input: Input{UserID: "u-1", Email: "test@example.com", Plan: "personal", Period: "monthly"},
			customerReader: &mockCustomerReader{customer: &domain.StripeCustomer{
				UserID:     "u-1",
				CustomerID: "cus_existing",
			}},
			customerSaver:  &mockCustomerSaver{},
			sessionCreator: &mockSessionCreator{err: fmt.Errorf("stripe session error")},
			priceResolver:  &mockPriceResolver{priceID: "price_personal_monthly"},
			wantErr:        "create checkout session",
			wantCode:       errors.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customerReader := tt.customerReader
			if customerReader == nil {
				customerReader = &mockCustomerReader{}
			}
			customerSaver := tt.customerSaver
			if customerSaver == nil {
				customerSaver = &mockCustomerSaver{}
			}
			customerCreator := tt.customerCreator
			if customerCreator == nil {
				customerCreator = &mockCustomerCreator{}
			}
			sessionCreator := tt.sessionCreator
			if sessionCreator == nil {
				sessionCreator = &mockSessionCreator{}
			}
			priceResolver := tt.priceResolver
			if priceResolver == nil {
				priceResolver = &mockPriceResolver{}
			}

			uc := New(
				customerReader, customerSaver, customerCreator, sessionCreator, priceResolver,
				"https://success", "https://cancel", tt.trialDays,
			)
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
			assert.Equal(t, tt.wantURL, got.CheckoutURL)
		})
	}
}

func TestExecute_NewCustomerSavedCorrectly(t *testing.T) {
	ctx := context.Background()
	saver := &mockCustomerSaver{}

	uc := New(
		&mockCustomerReader{customer: nil},
		saver,
		&mockCustomerCreator{customerID: "cus_new_123"},
		&mockSessionCreator{url: "https://checkout.stripe.com/session"},
		&mockPriceResolver{priceID: "price_personal_m"},
		"https://success", "https://cancel", 0,
	)

	_, err := uc.Execute(ctx, Input{
		UserID: "u-1", Email: "test@example.com", Plan: "personal", Period: "monthly",
	})
	require.NoError(t, err)

	assert.Equal(t, "u-1", saver.calledUserID)
	assert.Equal(t, "cus_new_123", saver.calledCustomerID)
}

func TestExecute_CheckoutParamsPassedCorrectly(t *testing.T) {
	ctx := context.Background()
	sessionCreator := &mockSessionCreator{url: "https://checkout.stripe.com/session"}

	uc := New(
		&mockCustomerReader{customer: &domain.StripeCustomer{
			UserID:     "u-1",
			CustomerID: "cus_abc",
		}},
		&mockCustomerSaver{},
		&mockCustomerCreator{},
		sessionCreator,
		&mockPriceResolver{priceID: "price_teams_annual"},
		"https://success.example.com", "https://cancel.example.com", 14,
	)

	_, err := uc.Execute(ctx, Input{
		UserID: "u-1", Email: "test@example.com", Plan: "teams", Period: "annual",
	})
	require.NoError(t, err)

	assert.Equal(t, "cus_abc", sessionCreator.calledParams.CustomerID)
	assert.Equal(t, "price_teams_annual", sessionCreator.calledParams.PriceID)
	assert.Equal(t, int64(14), sessionCreator.calledParams.TrialDays)
	assert.Equal(t, "https://success.example.com", sessionCreator.calledParams.SuccessURL)
	assert.Equal(t, "https://cancel.example.com", sessionCreator.calledParams.CancelURL)
	assert.Equal(t, "u-1", sessionCreator.calledParams.Metadata["user_id"])
}

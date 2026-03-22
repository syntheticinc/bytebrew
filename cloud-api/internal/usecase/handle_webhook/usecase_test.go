package handle_webhook

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/cloud-api/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mocks ---

type mockUserIDResolver struct {
	userID string
	err    error
}

func (m *mockUserIDResolver) GetUserIDByCustomerID(_ context.Context, _ string) (string, error) {
	return m.userID, m.err
}

type mockSubReader struct {
	sub *domain.Subscription
	err error
}

func (m *mockSubReader) GetByUserID(_ context.Context, _ string) (*domain.Subscription, error) {
	return m.sub, m.err
}

type mockSubUpdater struct {
	updateFullCalled   bool
	updateStatusCalled bool
	lastTier           domain.LicenseTier
	lastStatus         domain.SubscriptionStatus
	lastSubID          string
	lastProxyLimit     int
	err                error
}

func (m *mockSubUpdater) UpdateFull(_ context.Context, _ string, tier domain.LicenseTier, status domain.SubscriptionStatus, _, _ *time.Time, subID string, proxyStepsLimit int) error {
	m.updateFullCalled = true
	m.lastTier = tier
	m.lastStatus = status
	m.lastSubID = subID
	m.lastProxyLimit = proxyStepsLimit
	return m.err
}

func (m *mockSubUpdater) UpdateStatus(_ context.Context, _ string, status domain.SubscriptionStatus) error {
	m.updateStatusCalled = true
	m.lastStatus = status
	return m.err
}

type mockSubCreator struct {
	createCalled bool
	lastSub      *domain.Subscription
	err          error
}

func (m *mockSubCreator) Create(_ context.Context, sub *domain.Subscription) (*domain.Subscription, error) {
	m.createCalled = true
	m.lastSub = sub
	if m.err != nil {
		return nil, m.err
	}
	sub.ID = "sub-uuid"
	return sub, nil
}

type mockEventStore struct {
	processed bool
	isErr     error
	markErr   error
}

func (m *mockEventStore) IsProcessed(_ context.Context, _ string) (bool, error) {
	return m.processed, m.isErr
}

func (m *mockEventStore) MarkProcessed(_ context.Context, _, _ string) error {
	return m.markErr
}

type mockTierResolver struct {
	tier domain.LicenseTier
	ok   bool
}

func (m *mockTierResolver) TierForPriceID(_ string) (domain.LicenseTier, bool) {
	return m.tier, m.ok
}

type mockProxyResetter struct {
	resetCalled bool
	err         error
}

func (m *mockProxyResetter) ResetProxySteps(_ context.Context, _ string) error {
	m.resetCalled = true
	return m.err
}

type mockTeamSeatsUpdater struct {
	team            *domain.Team
	getErr          error
	updateErr       error
	updateCalled    bool
	lastMaxSeats    int
}

func (m *mockTeamSeatsUpdater) GetTeamByOwnerID(_ context.Context, _ string) (*domain.Team, error) {
	return m.team, m.getErr
}

func (m *mockTeamSeatsUpdater) UpdateTeamMaxSeats(_ context.Context, _ string, maxSeats int) error {
	m.updateCalled = true
	m.lastMaxSeats = maxSeats
	return m.updateErr
}

// --- tests ---

func TestExecute(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	tests := []struct {
		name               string
		event              Event
		userIDResolver     *mockUserIDResolver
		subReader          *mockSubReader
		subUpdater         *mockSubUpdater
		subCreator         *mockSubCreator
		eventStore         *mockEventStore
		tierResolver       *mockTierResolver
		proxyResetter      *mockProxyResetter
		teamSeats          *mockTeamSeatsUpdater
		wantFullCalled     bool
		wantStatusCalled   bool
		wantCreateCalled   bool
		wantResetCalled    bool
		wantSeatsCalled    bool
		wantSeatsValue     int
		wantTier           domain.LicenseTier
		wantStatus         domain.SubscriptionStatus
		wantSubID          string
		wantProxyLimit     int
		wantErr            string
		wantCode           string
	}{
		{
			name: "subscription created with known price",
			event: Event{
				ID:   "evt_1",
				Type: "customer.subscription.created",
				Data: EventData{
					CustomerID:         "cus_1",
					SubscriptionID:     "sub_1",
					Status:             "active",
					PriceID:            "price_personal",
					CurrentPeriodStart: &now,
					CurrentPeriodEnd:   &now,
				},
			},
			userIDResolver: &mockUserIDResolver{userID: "u-1"},
			subReader:      &mockSubReader{sub: &domain.Subscription{UserID: "u-1"}},
			subUpdater:     &mockSubUpdater{},
			subCreator:     &mockSubCreator{},
			eventStore:     &mockEventStore{processed: false},
			tierResolver:   &mockTierResolver{tier: domain.TierPersonal, ok: true},
			proxyResetter:  &mockProxyResetter{},
			wantFullCalled: true,
			wantTier:       domain.TierPersonal,
			wantStatus:     domain.StatusActive,
			wantSubID:      "sub_1",
			wantProxyLimit: 300,
		},
		{
			name: "subscription updated with trialing status",
			event: Event{
				ID:   "evt_2",
				Type: "customer.subscription.updated",
				Data: EventData{
					CustomerID:     "cus_1",
					SubscriptionID: "sub_1",
					Status:         "trialing",
					PriceID:        "price_personal",
				},
			},
			userIDResolver: &mockUserIDResolver{userID: "u-1"},
			subReader:      &mockSubReader{sub: &domain.Subscription{UserID: "u-1"}},
			subUpdater:     &mockSubUpdater{},
			subCreator:     &mockSubCreator{},
			eventStore:     &mockEventStore{processed: false},
			tierResolver:   &mockTierResolver{tier: domain.TierPersonal, ok: true},
			proxyResetter:  &mockProxyResetter{},
			wantFullCalled: true,
			wantTier:       domain.TierPersonal,
			wantStatus:     domain.StatusTrialing,
			wantSubID:      "sub_1",
			wantProxyLimit: 300,
		},
		{
			name: "subscription created for user without subscription creates new",
			event: Event{
				ID:   "evt_new_sub",
				Type: "customer.subscription.created",
				Data: EventData{
					CustomerID:     "cus_1",
					SubscriptionID: "sub_1",
					Status:         "trialing",
					PriceID:        "price_personal",
				},
			},
			userIDResolver:   &mockUserIDResolver{userID: "u-1"},
			subReader:        &mockSubReader{sub: nil}, // no existing subscription
			subUpdater:       &mockSubUpdater{},
			subCreator:       &mockSubCreator{},
			eventStore:       &mockEventStore{processed: false},
			tierResolver:     &mockTierResolver{tier: domain.TierPersonal, ok: true},
			proxyResetter:    &mockProxyResetter{},
			wantFullCalled:   true,
			wantCreateCalled: true,
			wantTier:         domain.TierPersonal,
			wantStatus:       domain.StatusTrialing,
			wantSubID:        "sub_1",
			wantProxyLimit:   300,
		},
		{
			name: "trial subscription created with limit 0",
			event: Event{
				ID:   "evt_trial",
				Type: "customer.subscription.created",
				Data: EventData{
					CustomerID:     "cus_1",
					SubscriptionID: "sub_trial",
					Status:         "trialing",
					PriceID:        "price_trial",
				},
			},
			userIDResolver:   &mockUserIDResolver{userID: "u-1"},
			subReader:        &mockSubReader{sub: nil},
			subUpdater:       &mockSubUpdater{},
			subCreator:       &mockSubCreator{},
			eventStore:       &mockEventStore{processed: false},
			tierResolver:     &mockTierResolver{tier: domain.TierTrial, ok: true},
			proxyResetter:    &mockProxyResetter{},
			wantFullCalled:   true,
			wantCreateCalled: true,
			wantTier:         domain.TierTrial,
			wantStatus:       domain.StatusTrialing,
			wantSubID:        "sub_trial",
			wantProxyLimit:   0, // Trial has no monthly cap
		},
		{
			name: "subscription deleted sets canceled status",
			event: Event{
				ID:   "evt_3",
				Type: "customer.subscription.deleted",
				Data: EventData{CustomerID: "cus_1"},
			},
			userIDResolver:   &mockUserIDResolver{userID: "u-1"},
			subReader:        &mockSubReader{sub: &domain.Subscription{UserID: "u-1"}},
			subUpdater:       &mockSubUpdater{},
			subCreator:       &mockSubCreator{},
			eventStore:       &mockEventStore{processed: false},
			tierResolver:     &mockTierResolver{},
			proxyResetter:    &mockProxyResetter{},
			wantStatusCalled: true,
			wantStatus:       domain.StatusCanceled,
		},
		{
			name: "payment failed sets past_due",
			event: Event{
				ID:   "evt_4",
				Type: "invoice.payment_failed",
				Data: EventData{InvoiceCustomerID: "cus_1"},
			},
			userIDResolver:   &mockUserIDResolver{userID: "u-1"},
			subReader:        &mockSubReader{sub: &domain.Subscription{UserID: "u-1"}},
			subUpdater:       &mockSubUpdater{},
			subCreator:       &mockSubCreator{},
			eventStore:       &mockEventStore{processed: false},
			tierResolver:     &mockTierResolver{},
			proxyResetter:    &mockProxyResetter{},
			wantStatusCalled: true,
			wantStatus:       domain.StatusPastDue,
		},
		{
			name: "payment succeeded restores from past_due to active and resets proxy",
			event: Event{
				ID:   "evt_5",
				Type: "invoice.payment_succeeded",
				Data: EventData{InvoiceCustomerID: "cus_1"},
			},
			userIDResolver:   &mockUserIDResolver{userID: "u-1"},
			subReader:        &mockSubReader{sub: &domain.Subscription{UserID: "u-1", Status: domain.StatusPastDue}},
			subUpdater:       &mockSubUpdater{},
			subCreator:       &mockSubCreator{},
			eventStore:       &mockEventStore{processed: false},
			tierResolver:     &mockTierResolver{},
			proxyResetter:    &mockProxyResetter{},
			wantStatusCalled: true,
			wantResetCalled:  true,
			wantStatus:       domain.StatusActive,
		},
		{
			name: "payment succeeded resets proxy even if not past_due",
			event: Event{
				ID:   "evt_6",
				Type: "invoice.payment_succeeded",
				Data: EventData{InvoiceCustomerID: "cus_1"},
			},
			userIDResolver:  &mockUserIDResolver{userID: "u-1"},
			subReader:       &mockSubReader{sub: &domain.Subscription{UserID: "u-1", Status: domain.StatusActive}},
			subUpdater:      &mockSubUpdater{},
			subCreator:      &mockSubCreator{},
			eventStore:      &mockEventStore{processed: false},
			tierResolver:    &mockTierResolver{},
			proxyResetter:   &mockProxyResetter{},
			wantResetCalled: true,
			// no status update expected (already active)
		},
		{
			name: "trial_will_end is logged without error",
			event: Event{
				ID:   "evt_trial_end",
				Type: "customer.subscription.trial_will_end",
				Data: EventData{
					CustomerID:     "cus_1",
					SubscriptionID: "sub_1",
				},
			},
			userIDResolver: &mockUserIDResolver{},
			subReader:      &mockSubReader{},
			subUpdater:     &mockSubUpdater{},
			subCreator:     &mockSubCreator{},
			eventStore:     &mockEventStore{processed: false},
			tierResolver:   &mockTierResolver{},
			proxyResetter:  &mockProxyResetter{},
			// no update expected — just logging
		},
		{
			name: "idempotent — already processed event is skipped",
			event: Event{
				ID:   "evt_dup",
				Type: "customer.subscription.created",
				Data: EventData{CustomerID: "cus_1"},
			},
			userIDResolver: &mockUserIDResolver{},
			subReader:      &mockSubReader{},
			subUpdater:     &mockSubUpdater{},
			subCreator:     &mockSubCreator{},
			eventStore:     &mockEventStore{processed: true},
			tierResolver:   &mockTierResolver{},
			proxyResetter:  &mockProxyResetter{},
			// no update expected
		},
		{
			name: "unknown event type is ignored",
			event: Event{
				ID:   "evt_unknown",
				Type: "unknown.event.type",
			},
			userIDResolver: &mockUserIDResolver{},
			subReader:      &mockSubReader{},
			subUpdater:     &mockSubUpdater{},
			subCreator:     &mockSubCreator{},
			eventStore:     &mockEventStore{processed: false},
			tierResolver:   &mockTierResolver{},
			proxyResetter:  &mockProxyResetter{},
			// no update expected
		},
		{
			name: "event store error",
			event: Event{
				ID:   "evt_err",
				Type: "customer.subscription.created",
			},
			userIDResolver: &mockUserIDResolver{},
			subReader:      &mockSubReader{},
			subUpdater:     &mockSubUpdater{},
			subCreator:     &mockSubCreator{},
			eventStore:     &mockEventStore{isErr: fmt.Errorf("db error")},
			tierResolver:   &mockTierResolver{},
			proxyResetter:  &mockProxyResetter{},
			wantErr:        "check event idempotency",
			wantCode:       errors.CodeInternal,
		},
		{
			name: "mark processed error returns error",
			event: Event{
				ID:   "evt_mark_err",
				Type: "unknown.event.type",
			},
			userIDResolver: &mockUserIDResolver{},
			subReader:      &mockSubReader{},
			subUpdater:     &mockSubUpdater{},
			subCreator:     &mockSubCreator{},
			eventStore:     &mockEventStore{processed: false, markErr: fmt.Errorf("db write failed")},
			tierResolver:   &mockTierResolver{},
			proxyResetter:  &mockProxyResetter{},
			wantErr:        "mark event as processed",
			wantCode:       errors.CodeInternal,
		},
		{
			name: "user ID resolver error",
			event: Event{
				ID:   "evt_err2",
				Type: "customer.subscription.created",
				Data: EventData{CustomerID: "cus_1", PriceID: "price_personal"},
			},
			userIDResolver: &mockUserIDResolver{err: fmt.Errorf("db down")},
			subReader:      &mockSubReader{},
			subUpdater:     &mockSubUpdater{},
			subCreator:     &mockSubCreator{},
			eventStore:     &mockEventStore{processed: false},
			tierResolver:   &mockTierResolver{tier: domain.TierPersonal, ok: true},
			proxyResetter:  &mockProxyResetter{},
			wantErr:        "resolve user by stripe customer",
			wantCode:       errors.CodeInternal,
		},
		{
			name: "user not found for customer — no error",
			event: Event{
				ID:   "evt_no_user",
				Type: "customer.subscription.created",
				Data: EventData{CustomerID: "cus_unknown", PriceID: "price_personal"},
			},
			userIDResolver: &mockUserIDResolver{userID: ""}, // not found
			subReader:      &mockSubReader{},
			subUpdater:     &mockSubUpdater{},
			subCreator:     &mockSubCreator{},
			eventStore:     &mockEventStore{processed: false},
			tierResolver:   &mockTierResolver{tier: domain.TierPersonal, ok: true},
			proxyResetter:  &mockProxyResetter{},
			// no error, no update
		},
		{
			name: "unknown price returns error",
			event: Event{
				ID:   "evt_unknown_price",
				Type: "customer.subscription.created",
				Data: EventData{
					CustomerID:     "cus_1",
					SubscriptionID: "sub_1",
					Status:         "active",
					PriceID:        "price_unknown",
				},
			},
			userIDResolver: &mockUserIDResolver{userID: "u-1"},
			subReader:      &mockSubReader{sub: &domain.Subscription{UserID: "u-1"}},
			subUpdater:     &mockSubUpdater{},
			subCreator:     &mockSubCreator{},
			eventStore:     &mockEventStore{processed: false},
			tierResolver:   &mockTierResolver{ok: false},
			proxyResetter:  &mockProxyResetter{},
			wantErr:        "unknown price ID",
			wantCode:       errors.CodeInternal,
		},
		{
			name: "payment succeeded proxy reset error",
			event: Event{
				ID:   "evt_reset_err",
				Type: "invoice.payment_succeeded",
				Data: EventData{InvoiceCustomerID: "cus_1"},
			},
			userIDResolver: &mockUserIDResolver{userID: "u-1"},
			subReader:      &mockSubReader{sub: &domain.Subscription{UserID: "u-1", Status: domain.StatusActive}},
			subUpdater:     &mockSubUpdater{},
			subCreator:     &mockSubCreator{},
			eventStore:     &mockEventStore{processed: false},
			tierResolver:   &mockTierResolver{},
			proxyResetter:  &mockProxyResetter{err: fmt.Errorf("reset failed")},
			wantErr:        "reset proxy steps",
			wantCode:       errors.CodeInternal,
		},
		{
			name: "teams subscription syncs seats from quantity",
			event: Event{
				ID:   "evt_teams",
				Type: "customer.subscription.updated",
				Data: EventData{
					CustomerID:     "cus_1",
					SubscriptionID: "sub_teams",
					Status:         "active",
					PriceID:        "price_teams",
					Quantity:       5,
				},
			},
			userIDResolver:  &mockUserIDResolver{userID: "u-1"},
			subReader:       &mockSubReader{sub: &domain.Subscription{UserID: "u-1"}},
			subUpdater:      &mockSubUpdater{},
			subCreator:      &mockSubCreator{},
			eventStore:      &mockEventStore{processed: false},
			tierResolver:    &mockTierResolver{tier: domain.TierTeams, ok: true},
			proxyResetter:   &mockProxyResetter{},
			teamSeats:       &mockTeamSeatsUpdater{team: &domain.Team{ID: "team-1", OwnerID: "u-1"}},
			wantFullCalled:  true,
			wantSeatsCalled: true,
			wantSeatsValue:  5,
			wantTier:        domain.TierTeams,
			wantStatus:      domain.StatusActive,
			wantSubID:       "sub_teams",
			wantProxyLimit:  300,
		},
		{
			name: "teams subscription with zero quantity does not sync seats",
			event: Event{
				ID:   "evt_teams_zero",
				Type: "customer.subscription.updated",
				Data: EventData{
					CustomerID:     "cus_1",
					SubscriptionID: "sub_teams",
					Status:         "active",
					PriceID:        "price_teams",
					Quantity:       0,
				},
			},
			userIDResolver: &mockUserIDResolver{userID: "u-1"},
			subReader:      &mockSubReader{sub: &domain.Subscription{UserID: "u-1"}},
			subUpdater:     &mockSubUpdater{},
			subCreator:     &mockSubCreator{},
			eventStore:     &mockEventStore{processed: false},
			tierResolver:   &mockTierResolver{tier: domain.TierTeams, ok: true},
			proxyResetter:  &mockProxyResetter{},
			teamSeats:      &mockTeamSeatsUpdater{team: &domain.Team{ID: "team-1", OwnerID: "u-1"}},
			wantFullCalled: true,
			wantTier:       domain.TierTeams,
			wantStatus:     domain.StatusActive,
			wantSubID:      "sub_teams",
			wantProxyLimit: 300,
			// wantSeatsCalled is false — zero quantity means no sync
		},
		{
			name: "personal tier does not sync seats even with quantity",
			event: Event{
				ID:   "evt_personal_qty",
				Type: "customer.subscription.updated",
				Data: EventData{
					CustomerID:     "cus_1",
					SubscriptionID: "sub_1",
					Status:         "active",
					PriceID:        "price_personal",
					Quantity:       1,
				},
			},
			userIDResolver: &mockUserIDResolver{userID: "u-1"},
			subReader:      &mockSubReader{sub: &domain.Subscription{UserID: "u-1"}},
			subUpdater:     &mockSubUpdater{},
			subCreator:     &mockSubCreator{},
			eventStore:     &mockEventStore{processed: false},
			tierResolver:   &mockTierResolver{tier: domain.TierPersonal, ok: true},
			proxyResetter:  &mockProxyResetter{},
			teamSeats:      &mockTeamSeatsUpdater{team: &domain.Team{ID: "team-1", OwnerID: "u-1"}},
			wantFullCalled: true,
			wantTier:       domain.TierPersonal,
			wantStatus:     domain.StatusActive,
			wantSubID:      "sub_1",
			wantProxyLimit: 300,
			// wantSeatsCalled is false — personal tier does not sync seats
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			teamSeats := tt.teamSeats
			if teamSeats == nil {
				teamSeats = &mockTeamSeatsUpdater{}
			}
			uc := New(tt.userIDResolver, tt.subReader, tt.subUpdater, tt.subCreator, tt.eventStore, tt.tierResolver, tt.proxyResetter, teamSeats)
			err := uc.Execute(ctx, tt.event)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				if tt.wantCode != "" {
					assert.True(t, errors.Is(err, tt.wantCode), "expected code %s, got %s", tt.wantCode, errors.GetCode(err))
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantFullCalled, tt.subUpdater.updateFullCalled, "UpdateFull called")
			assert.Equal(t, tt.wantStatusCalled, tt.subUpdater.updateStatusCalled, "UpdateStatus called")
			assert.Equal(t, tt.wantCreateCalled, tt.subCreator.createCalled, "Create called")
			assert.Equal(t, tt.wantResetCalled, tt.proxyResetter.resetCalled, "ResetProxySteps called")

			if tt.wantFullCalled {
				assert.Equal(t, tt.wantTier, tt.subUpdater.lastTier)
				assert.Equal(t, tt.wantStatus, tt.subUpdater.lastStatus)
				assert.Equal(t, tt.wantSubID, tt.subUpdater.lastSubID)
				assert.Equal(t, tt.wantProxyLimit, tt.subUpdater.lastProxyLimit)
			}
			if tt.wantStatusCalled {
				assert.Equal(t, tt.wantStatus, tt.subUpdater.lastStatus)
			}
			if tt.wantCreateCalled {
				require.NotNil(t, tt.subCreator.lastSub)
				assert.Equal(t, tt.wantProxyLimit, tt.subCreator.lastSub.ProxyStepsLimit)
				assert.True(t, tt.subCreator.lastSub.BYOKEnabled)
			}
			if tt.wantSeatsCalled {
				assert.True(t, teamSeats.updateCalled, "UpdateTeamMaxSeats called")
				assert.Equal(t, tt.wantSeatsValue, teamSeats.lastMaxSeats)
			}
		})
	}
}

func TestMapStripeStatus(t *testing.T) {
	tests := []struct {
		stripe string
		want   domain.SubscriptionStatus
	}{
		{"active", domain.StatusActive},
		{"trialing", domain.StatusTrialing},
		{"past_due", domain.StatusPastDue},
		{"canceled", domain.StatusCanceled},
		{"unpaid", domain.StatusCanceled},
		{"incomplete_expired", domain.StatusCanceled},
		{"unknown_status", domain.StatusCanceled},
	}

	for _, tt := range tests {
		t.Run(tt.stripe, func(t *testing.T) {
			got := mapStripeStatus(tt.stripe)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProxyStepsLimitForTier(t *testing.T) {
	tests := []struct {
		tier domain.LicenseTier
		want int
	}{
		{domain.TierTrial, 0},
		{domain.TierPersonal, 300},
		{domain.TierTeams, 300},
	}

	for _, tt := range tests {
		t.Run(string(tt.tier), func(t *testing.T) {
			got := proxyStepsLimitForTier(tt.tier)
			assert.Equal(t, tt.want, got)
		})
	}
}

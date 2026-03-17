//go:build integration

package http_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/domain"
	pgstore "github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/infrastructure/postgres"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/create_checkout"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/create_portal"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/handle_webhook"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/migrations"
)

// --- billing test environment ---

type billingTestEnv struct {
	pool               *pgxpool.Pool
	userRepo           *pgstore.UserRepository
	subRepo            *pgstore.SubscriptionRepository
	eventRepo          *pgstore.StripeEventRepository
	stripeCustomerRepo *pgstore.StripeCustomerRepository
}

func newBillingTestEnv(t *testing.T) *billingTestEnv {
	t.Helper()
	ctx := context.Background()

	ctr, err := tcpostgres.Run(ctx,
		"postgres:17-alpine",
		tcpostgres.WithDatabase("bytebrew_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
		tcpostgres.WithSQLDriver("pgx"),
	)
	testcontainers.CleanupContainer(t, ctr)
	require.NoError(t, err)

	connStr, err := ctr.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	source, err := iofs.New(migrations.FS, ".")
	require.NoError(t, err)

	m, err := migrate.NewWithSourceInstance("iofs", source, connStr)
	require.NoError(t, err)
	require.NoError(t, m.Up())

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return &billingTestEnv{
		pool:               pool,
		userRepo:           pgstore.NewUserRepository(pool),
		subRepo:            pgstore.NewSubscriptionRepository(pool),
		eventRepo:          pgstore.NewStripeEventRepository(pool),
		stripeCustomerRepo: pgstore.NewStripeCustomerRepository(pool),
	}
}

func (e *billingTestEnv) truncate(t *testing.T) {
	t.Helper()
	_, err := e.pool.Exec(context.Background(),
		"TRUNCATE TABLE processed_stripe_events, stripe_customers, subscriptions, users CASCADE")
	require.NoError(t, err)
}

// createTestUser creates a user and a free subscription, returning userID.
func (e *billingTestEnv) createTestUser(t *testing.T, email string) string {
	t.Helper()
	ctx := context.Background()

	user, err := e.userRepo.Create(ctx, &domain.User{
		Email:        email,
		PasswordHash: "$2a$10$testhashfortesting00uAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
	})
	require.NoError(t, err)

	_, err = e.subRepo.Create(ctx, &domain.Subscription{
		UserID: user.ID,
		Tier:   domain.TierPersonal,
		Status: domain.StatusActive,
	})
	require.NoError(t, err)

	return user.ID
}

// setStripeCustomerID creates a stripe customer mapping for the user.
func (e *billingTestEnv) setStripeCustomerID(t *testing.T, userID, customerID string) {
	t.Helper()
	err := e.stripeCustomerRepo.Upsert(context.Background(), userID, customerID)
	require.NoError(t, err)
}

// getSubscription reads the current subscription from DB.
func (e *billingTestEnv) getSubscription(t *testing.T, userID string) *domain.Subscription {
	t.Helper()
	sub, err := e.subRepo.GetByUserID(context.Background(), userID)
	require.NoError(t, err)
	require.NotNil(t, sub, "subscription must exist for user %s", userID)
	return sub
}

// --- mock Stripe dependencies ---

type mockCustomerCreator struct {
	customerID string
	called     bool
}

func (m *mockCustomerCreator) CreateCustomer(_ context.Context, _ string, _ map[string]string) (string, error) {
	m.called = true
	return m.customerID, nil
}

type mockSessionCreator struct {
	url string
}

func (m *mockSessionCreator) CreateCheckoutSession(_ context.Context, _ create_checkout.CheckoutParams) (string, error) {
	return m.url, nil
}

type mockPriceResolver struct {
	priceID string
}

func (m *mockPriceResolver) PriceIDForPlan(_, _ string) (string, error) {
	return m.priceID, nil
}

type mockPortalCreator struct {
	url string
}

func (m *mockPortalCreator) CreatePortalSession(_ context.Context, _, _ string) (string, error) {
	return m.url, nil
}

type mockTierResolver struct {
	mapping map[string]domain.LicenseTier
}

func (m *mockTierResolver) TierForPriceID(priceID string) (domain.LicenseTier, bool) {
	tier, ok := m.mapping[priceID]
	return tier, ok
}

// --- webhook integration tests ---

func TestWebhookIntegration_SubscriptionCreated(t *testing.T) {
	env := newBillingTestEnv(t)
	env.truncate(t)
	ctx := context.Background()

	userID := env.createTestUser(t, "webhook-create@example.com")
	env.setStripeCustomerID(t, userID, "cus_wh_1")

	now := time.Now().Truncate(time.Microsecond)
	periodEnd := now.Add(30 * 24 * time.Hour).Truncate(time.Microsecond)

	uc := handle_webhook.New(
		env.stripeCustomerRepo,
		env.subRepo,
		env.subRepo,
		env.subRepo,
		env.eventRepo,
		&mockTierResolver{mapping: map[string]domain.LicenseTier{
			"price_personal_monthly": domain.TierPersonal,
		}},
		env.subRepo,
	)

	err := uc.Execute(ctx, handle_webhook.Event{
		ID:   "evt_sub_created_1",
		Type: "customer.subscription.created",
		Data: handle_webhook.EventData{
			CustomerID:         "cus_wh_1",
			SubscriptionID:     "sub_stripe_1",
			Status:             "active",
			PriceID:            "price_personal_monthly",
			CurrentPeriodStart: &now,
			CurrentPeriodEnd:   &periodEnd,
		},
	})
	require.NoError(t, err)

	sub := env.getSubscription(t, userID)
	assert.Equal(t, domain.TierPersonal, sub.Tier)
	assert.Equal(t, domain.StatusActive, sub.Status)
	require.NotNil(t, sub.StripeSubscriptionID)
	assert.Equal(t, "sub_stripe_1", *sub.StripeSubscriptionID)
	require.NotNil(t, sub.CurrentPeriodStart)
	assert.WithinDuration(t, now, *sub.CurrentPeriodStart, time.Second)
	require.NotNil(t, sub.CurrentPeriodEnd)
	assert.WithinDuration(t, periodEnd, *sub.CurrentPeriodEnd, time.Second)
}

func TestWebhookIntegration_PaymentFailed(t *testing.T) {
	env := newBillingTestEnv(t)
	env.truncate(t)
	ctx := context.Background()

	userID := env.createTestUser(t, "webhook-fail@example.com")
	env.setStripeCustomerID(t, userID, "cus_wh_2")

	// Upgrade to personal first
	err := env.subRepo.UpdateFull(ctx, userID, domain.TierPersonal, domain.StatusActive, nil, nil, "sub_2", 300)
	require.NoError(t, err)

	uc := handle_webhook.New(
		env.stripeCustomerRepo,
		env.subRepo,
		env.subRepo,
		env.subRepo,
		env.eventRepo,
		&mockTierResolver{},
		env.subRepo,
	)

	err = uc.Execute(ctx, handle_webhook.Event{
		ID:   "evt_payment_failed_1",
		Type: "invoice.payment_failed",
		Data: handle_webhook.EventData{
			InvoiceCustomerID: "cus_wh_2",
		},
	})
	require.NoError(t, err)

	sub := env.getSubscription(t, userID)
	assert.Equal(t, domain.StatusPastDue, sub.Status)
	assert.Equal(t, domain.TierPersonal, sub.Tier, "tier should remain personal")
}

func TestWebhookIntegration_SubscriptionDeleted(t *testing.T) {
	env := newBillingTestEnv(t)
	env.truncate(t)
	ctx := context.Background()

	userID := env.createTestUser(t, "webhook-del@example.com")
	env.setStripeCustomerID(t, userID, "cus_wh_3")

	// Upgrade to personal first
	err := env.subRepo.UpdateFull(ctx, userID, domain.TierPersonal, domain.StatusActive, nil, nil, "sub_3", 300)
	require.NoError(t, err)

	uc := handle_webhook.New(
		env.stripeCustomerRepo,
		env.subRepo,
		env.subRepo,
		env.subRepo,
		env.eventRepo,
		&mockTierResolver{},
		env.subRepo,
	)

	err = uc.Execute(ctx, handle_webhook.Event{
		ID:   "evt_sub_deleted_1",
		Type: "customer.subscription.deleted",
		Data: handle_webhook.EventData{
			CustomerID: "cus_wh_3",
		},
	})
	require.NoError(t, err)

	sub := env.getSubscription(t, userID)
	assert.Equal(t, domain.TierPersonal, sub.Tier, "tier preserved after deletion, only status changes")
	assert.Equal(t, domain.StatusCanceled, sub.Status)
}

func TestWebhookIntegration_Idempotency(t *testing.T) {
	env := newBillingTestEnv(t)
	env.truncate(t)
	ctx := context.Background()

	userID := env.createTestUser(t, "webhook-idemp@example.com")
	env.setStripeCustomerID(t, userID, "cus_wh_4")

	uc := handle_webhook.New(
		env.stripeCustomerRepo,
		env.subRepo,
		env.subRepo,
		env.subRepo,
		env.eventRepo,
		&mockTierResolver{mapping: map[string]domain.LicenseTier{
			"price_personal_monthly": domain.TierPersonal,
		}},
		env.subRepo,
	)

	event := handle_webhook.Event{
		ID:   "evt_idempotent_1",
		Type: "customer.subscription.created",
		Data: handle_webhook.EventData{
			CustomerID:     "cus_wh_4",
			SubscriptionID: "sub_4",
			Status:         "active",
			PriceID:        "price_personal_monthly",
		},
	}

	// First call processes the event
	err := uc.Execute(ctx, event)
	require.NoError(t, err)

	sub := env.getSubscription(t, userID)
	assert.Equal(t, domain.TierPersonal, sub.Tier)

	// Reset subscription to trial to verify second call is skipped
	err = env.subRepo.UpdateFull(ctx, userID, domain.TierTrial, domain.StatusActive, nil, nil, "", 0)
	require.NoError(t, err)

	// Second call with same event_id should be skipped
	err = uc.Execute(ctx, event)
	require.NoError(t, err)

	// Subscription should remain trial (event was not re-processed)
	sub = env.getSubscription(t, userID)
	assert.Equal(t, domain.TierTrial, sub.Tier, "idempotent event should not be re-processed")

	// Verify processed_stripe_events has exactly one entry
	var count int
	err = env.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM processed_stripe_events WHERE event_id = $1", "evt_idempotent_1",
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "should have exactly one processed event entry")
}

// --- checkout integration tests ---

func TestCheckoutIntegration_NewCustomer(t *testing.T) {
	env := newBillingTestEnv(t)
	env.truncate(t)
	ctx := context.Background()

	userID := env.createTestUser(t, "checkout-new@example.com")

	customerCreator := &mockCustomerCreator{customerID: "cus_new_123"}

	uc := create_checkout.New(
		env.stripeCustomerRepo,
		env.stripeCustomerRepo,
		customerCreator,
		&mockSessionCreator{url: "https://checkout.stripe.com/test-session"},
		&mockPriceResolver{priceID: "price_personal_monthly"},
		"https://success.example.com", "https://cancel.example.com", 0,
	)

	out, err := uc.Execute(ctx, create_checkout.Input{
		UserID: userID,
		Email:  "checkout-new@example.com",
		Plan:   "personal",
		Period: "monthly",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://checkout.stripe.com/test-session", out.CheckoutURL)

	// Verify stripe_customer was saved in stripe_customers table
	sc, err := env.stripeCustomerRepo.GetByUserID(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, sc)
	assert.Equal(t, "cus_new_123", sc.CustomerID)
	assert.True(t, customerCreator.called, "CustomerCreator should have been called")
}

func TestCheckoutIntegration_ExistingCustomer(t *testing.T) {
	env := newBillingTestEnv(t)
	env.truncate(t)
	ctx := context.Background()

	userID := env.createTestUser(t, "checkout-exist@example.com")
	env.setStripeCustomerID(t, userID, "cus_existing_456")

	customerCreator := &mockCustomerCreator{customerID: "cus_should_not_be_used"}

	uc := create_checkout.New(
		env.stripeCustomerRepo,
		env.stripeCustomerRepo,
		customerCreator,
		&mockSessionCreator{url: "https://checkout.stripe.com/existing-session"},
		&mockPriceResolver{priceID: "price_personal_monthly"},
		"https://success.example.com", "https://cancel.example.com", 0,
	)

	out, err := uc.Execute(ctx, create_checkout.Input{
		UserID: userID,
		Email:  "checkout-exist@example.com",
		Plan:   "personal",
		Period: "monthly",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://checkout.stripe.com/existing-session", out.CheckoutURL)

	// CustomerCreator should NOT have been called
	assert.False(t, customerCreator.called, "CustomerCreator should not be called for existing customer")

	// stripe_customer should remain the same
	sc, err := env.stripeCustomerRepo.GetByUserID(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, sc)
	assert.Equal(t, "cus_existing_456", sc.CustomerID)
}

// --- portal integration tests ---

func TestPortalIntegration_NoStripeCustomer(t *testing.T) {
	env := newBillingTestEnv(t)
	env.truncate(t)
	ctx := context.Background()

	userID := env.createTestUser(t, "portal-no-cust@example.com")

	uc := create_portal.New(
		env.stripeCustomerRepo,
		&mockPortalCreator{url: "https://billing.stripe.com/portal"},
		"https://return.example.com",
	)

	_, err := uc.Execute(ctx, create_portal.Input{UserID: userID})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no active Stripe subscription found")
}

func TestPortalIntegration_WithStripeCustomer(t *testing.T) {
	env := newBillingTestEnv(t)
	env.truncate(t)
	ctx := context.Background()

	userID := env.createTestUser(t, "portal-ok@example.com")
	env.setStripeCustomerID(t, userID, "cus_portal_1")

	uc := create_portal.New(
		env.stripeCustomerRepo,
		&mockPortalCreator{url: "https://billing.stripe.com/portal-session"},
		"https://return.example.com",
	)

	out, err := uc.Execute(ctx, create_portal.Input{UserID: userID})
	require.NoError(t, err)
	assert.Equal(t, "https://billing.stripe.com/portal-session", out.PortalURL)
}

// --- full lifecycle integration test ---

func TestBillingIntegration_FullLifecycle(t *testing.T) {
	env := newBillingTestEnv(t)
	env.truncate(t)
	ctx := context.Background()

	// Step 1: Create user with personal subscription
	userID := env.createTestUser(t, "lifecycle@example.com")

	sub := env.getSubscription(t, userID)
	assert.Equal(t, domain.TierPersonal, sub.Tier)
	assert.Equal(t, domain.StatusActive, sub.Status)

	// Step 2: Checkout creates Stripe customer and saves ID
	customerCreator := &mockCustomerCreator{customerID: "cus_lifecycle_1"}
	checkoutUC := create_checkout.New(
		env.stripeCustomerRepo,
		env.stripeCustomerRepo,
		customerCreator,
		&mockSessionCreator{url: "https://checkout.stripe.com/lifecycle"},
		&mockPriceResolver{priceID: "price_personal_monthly"},
		"https://success.example.com", "https://cancel.example.com", 0,
	)

	out, err := checkoutUC.Execute(ctx, create_checkout.Input{
		UserID: userID,
		Email:  "lifecycle@example.com",
		Plan:   "personal",
		Period: "monthly",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://checkout.stripe.com/lifecycle", out.CheckoutURL)

	sc, err := env.stripeCustomerRepo.GetByUserID(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, sc)
	assert.Equal(t, "cus_lifecycle_1", sc.CustomerID)
	sub = env.getSubscription(t, userID)
	assert.Equal(t, domain.TierPersonal, sub.Tier, "tier still personal until webhook")

	// Step 3: Webhook subscription.created -> tier=personal, status=active
	tierResolver := &mockTierResolver{mapping: map[string]domain.LicenseTier{
		"price_personal_monthly": domain.TierPersonal,
	}}
	webhookUC := handle_webhook.New(env.stripeCustomerRepo, env.subRepo, env.subRepo, env.subRepo, env.eventRepo, tierResolver, env.subRepo)

	now := time.Now().Truncate(time.Microsecond)
	periodEnd := now.Add(30 * 24 * time.Hour).Truncate(time.Microsecond)

	err = webhookUC.Execute(ctx, handle_webhook.Event{
		ID:   "evt_lifecycle_1",
		Type: "customer.subscription.created",
		Data: handle_webhook.EventData{
			CustomerID:         "cus_lifecycle_1",
			SubscriptionID:     "sub_lifecycle_1",
			Status:             "active",
			PriceID:            "price_personal_monthly",
			CurrentPeriodStart: &now,
			CurrentPeriodEnd:   &periodEnd,
		},
	})
	require.NoError(t, err)

	sub = env.getSubscription(t, userID)
	assert.Equal(t, domain.TierPersonal, sub.Tier)
	assert.Equal(t, domain.StatusActive, sub.Status)
	require.NotNil(t, sub.StripeSubscriptionID)
	assert.Equal(t, "sub_lifecycle_1", *sub.StripeSubscriptionID)

	// Step 4: Webhook invoice.payment_failed -> status=past_due
	err = webhookUC.Execute(ctx, handle_webhook.Event{
		ID:   "evt_lifecycle_2",
		Type: "invoice.payment_failed",
		Data: handle_webhook.EventData{
			InvoiceCustomerID: "cus_lifecycle_1",
		},
	})
	require.NoError(t, err)

	sub = env.getSubscription(t, userID)
	assert.Equal(t, domain.TierPersonal, sub.Tier, "tier should remain personal after payment failure")
	assert.Equal(t, domain.StatusPastDue, sub.Status)

	// Step 5: Webhook invoice.payment_succeeded -> status=active (restored)
	err = webhookUC.Execute(ctx, handle_webhook.Event{
		ID:   "evt_lifecycle_3",
		Type: "invoice.payment_succeeded",
		Data: handle_webhook.EventData{
			InvoiceCustomerID: "cus_lifecycle_1",
		},
	})
	require.NoError(t, err)

	sub = env.getSubscription(t, userID)
	assert.Equal(t, domain.TierPersonal, sub.Tier)
	assert.Equal(t, domain.StatusActive, sub.Status, "status should be restored to active")

	// Step 6: Webhook subscription.deleted -> status=canceled (tier preserved)
	err = webhookUC.Execute(ctx, handle_webhook.Event{
		ID:   "evt_lifecycle_4",
		Type: "customer.subscription.deleted",
		Data: handle_webhook.EventData{
			CustomerID: "cus_lifecycle_1",
		},
	})
	require.NoError(t, err)

	sub = env.getSubscription(t, userID)
	assert.Equal(t, domain.TierPersonal, sub.Tier, "tier preserved after deletion, only status changes")
	assert.Equal(t, domain.StatusCanceled, sub.Status)
}

package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	httpdelivery "github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/delivery/http"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/delivery/http/middleware"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/infrastructure/crypto"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/infrastructure/email"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/infrastructure/postgres"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/infrastructure/ratelimit"
	stripeinfra "github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/infrastructure/stripe"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/accept_invite"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/activate"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/change_password"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/create_checkout"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/create_portal"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/create_team"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/delete_account"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/forgot_password"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/get_usage"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/handle_webhook"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/invite_member"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/list_members"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/login"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/proxy_llm"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/refresh_auth"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/refresh_license"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/register"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/remove_member"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/reset_password"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/migrations"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/pkg/config"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	migrateOnly := flag.Bool("migrate-only", false, "Run migrations and exit")
	flag.Parse()

	absConfigPath, err := resolveConfigPath(*configPath)
	if err != nil {
		log.Fatalf("Failed to resolve config path: %v", err)
	}

	cfg, err := config.Load(absConfigPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	slog.Info("Starting ByteBrew Cloud API", "port", cfg.Server.Port)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Database
	pool, err := connectDB(ctx, cfg.Database.URL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := runMigrations(cfg.Database.URL); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	slog.Info("Migrations applied")

	if *migrateOnly {
		slog.Info("Migrations complete, exiting (--migrate-only)")
		return
	}

	// Infrastructure
	privateKey, publicKey := parseEd25519Key(cfg.License.PrivateKeyHex)
	userRepo := postgres.NewUserRepository(pool)
	subRepo := postgres.NewSubscriptionRepository(pool)
	stripeCustomerRepo := postgres.NewStripeCustomerRepository(pool)
	teamRepo := postgres.NewTeamRepository(pool)
	tokenSigner := crypto.NewAuthTokenSigner(
		[]byte(cfg.Auth.JWTSecret),
		cfg.Auth.AccessTokenTTL,
		cfg.Auth.RefreshTokenTTL,
	)
	licenseSigner := crypto.NewLicenseSigner(privateKey)
	passwordHasher := crypto.NewBcryptHasher(bcrypt.DefaultCost)

	// Email
	var emailSender emailSenderI
	if cfg.Email.ResendAPIKey != "" {
		emailSender = email.NewResendSender(cfg.Email.ResendAPIKey, cfg.Email.FromEmail, cfg.Email.FrontendURL)
		slog.Info("Email sending enabled (Resend)")
	} else {
		emailSender = email.NewNoopSender()
		slog.Info("Email sending disabled (noop)")
	}

	// Adapters
	tokenVerifier := &tokenVerifierAdapter{signer: tokenSigner}
	licenseVerifier := &licenseVerifierAdapter{publicKey: publicKey}

	// Usecases
	registerUC := register.New(userRepo, tokenSigner, passwordHasher)
	loginUC := login.New(userRepo, tokenSigner, passwordHasher)
	refreshAuthUC := refresh_auth.New(&refreshTokenVerifierAdapter{signer: tokenSigner}, tokenSigner, userRepo)
	activateUC := activate.New(subRepo, licenseSigner, teamRepo)
	refreshUC := refresh_license.New(subRepo, licenseSigner, licenseVerifier, teamRepo)

	// Stripe billing (optional)
	var billingHandler *httpdelivery.BillingHandler
	var webhookHandler *httpdelivery.WebhookHandler
	var seatUpdater invite_member.SeatUpdater = &noopSeatUpdater{}
	var subCanceller delete_account.SubscriptionCanceller = &noopSubCanceller{}
	if cfg.Stripe.SecretKey != "" {
		slog.Info("Stripe billing enabled")

		checkoutClient := stripeinfra.NewCheckoutClient(cfg.Stripe.SecretKey)
		priceResolver := stripeinfra.NewPriceResolver(cfg.Stripe.Prices)
		eventRepo := postgres.NewStripeEventRepository(pool)
		seatUpdater = checkoutClient
		subCanceller = checkoutClient

		sessionAdapter := &checkoutSessionAdapter{client: checkoutClient}

		checkoutUC := create_checkout.New(
			stripeCustomerRepo, stripeCustomerRepo, checkoutClient, sessionAdapter, priceResolver,
			cfg.Stripe.SuccessURL, cfg.Stripe.CancelURL, cfg.Stripe.TrialDays,
		)
		webhookUC := handle_webhook.New(stripeCustomerRepo, subRepo, subRepo, subRepo, eventRepo, priceResolver, subRepo, teamRepo)
		portalReturnURL := cfg.Stripe.SuccessURL[:len(cfg.Stripe.SuccessURL)-len("/success")]
		portalUC := create_portal.New(stripeCustomerRepo, checkoutClient, portalReturnURL)

		billingHandler = httpdelivery.NewBillingHandler(checkoutUC, portalUC)
		webhookHandler = httpdelivery.NewWebhookHandler(webhookUC, cfg.Stripe.WebhookSecret)
	} else {
		slog.Info("Stripe billing disabled (no secret_key)")
	}

	// Teams (after Stripe — per-seat billing needs SeatUpdater)
	createTeamUC := create_team.New(teamRepo, teamRepo)
	inviteMemberUC := invite_member.New(teamRepo, teamRepo, teamRepo, emailSender, subRepo, seatUpdater, teamRepo)
	acceptInviteUC := accept_invite.New(teamRepo, teamRepo, userRepo, teamRepo)
	removeMemberUC := remove_member.New(teamRepo, teamRepo, subRepo, seatUpdater, teamRepo, teamRepo)
	listMembersUC := list_members.New(teamRepo, teamRepo, teamRepo)

	// Usage
	usageUC := get_usage.New(subRepo)
	usageHandler := httpdelivery.NewUsageHandler(usageUC)

	// LLM Proxy (optional — only when DeepInfra key is configured)
	var proxyHandler *httpdelivery.ProxyHandler
	if cfg.DeepInfra.APIKey != "" {
		trialLimiter := ratelimit.NewTrialLimiter(cfg.Trial.StepsPerHour)
		proxyUC := proxy_llm.New(subRepo, trialLimiter, &cfg.ModelRouting)
		proxyHandler = httpdelivery.NewProxyHandler(proxyUC, subRepo, cfg.DeepInfra.APIKey, cfg.DeepInfra.BaseURL)
		slog.Info("LLM proxy enabled", "base_url", cfg.DeepInfra.BaseURL)
	} else {
		slog.Info("LLM proxy disabled (no deepinfra.api_key)")
	}

	// Account management
	tokenGenerator := crypto.NewSecureTokenGenerator()
	changePasswordUC := change_password.New(userRepo, userRepo, passwordHasher)
	deleteAccountUC := delete_account.New(userRepo, userRepo, passwordHasher, subRepo, subCanceller)
	forgotPasswordUC := forgot_password.New(userRepo, userRepo, emailSender, tokenGenerator, cfg.Auth.PasswordResetTTL)
	resetPasswordUC := reset_password.New(userRepo, userRepo, passwordHasher)

	// Handlers & Router
	authHandler := httpdelivery.NewAuthHandler(registerUC, loginUC, refreshAuthUC)
	licenseHandler := httpdelivery.NewLicenseHandler(activateUC, refreshUC)
	teamHandler := httpdelivery.NewTeamHandler(createTeamUC, inviteMemberUC, acceptInviteUC, removeMemberUC, listMembersUC)
	accountHandler := httpdelivery.NewAccountHandler(changePasswordUC, deleteAccountUC, forgotPasswordUC, resetPasswordUC)
	router := httpdelivery.NewRouter(httpdelivery.RouterConfig{
		AuthHandler:    authHandler,
		LicenseHandler: licenseHandler,
		BillingHandler: billingHandler,
		WebhookHandler: webhookHandler,
		UsageHandler:   usageHandler,
		ProxyHandler:   proxyHandler,
		TeamHandler:    teamHandler,
		AccountHandler: accountHandler,
		TokenVerifier:  tokenVerifier,
		CORSOrigins:    cfg.CORS.AllowedOrigins,
	})

	// HTTP Server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("HTTP server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		slog.Info("Received signal, shutting down...", "signal", sig)
	case err := <-serverErr:
		slog.Error("HTTP server failed", "error", err)
	}

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Shutdown error", "error", err)
	}
	slog.Info("Server stopped")
}

// tokenVerifierAdapter adapts crypto.AuthTokenSigner to middleware.TokenVerifier.
type tokenVerifierAdapter struct {
	signer *crypto.AuthTokenSigner
}

func (a *tokenVerifierAdapter) VerifyAccessToken(tokenString string) (*middleware.AccessClaims, error) {
	claims, err := a.signer.VerifyAccessToken(tokenString)
	if err != nil {
		return nil, err
	}
	return &middleware.AccessClaims{
		UserID: claims.UserID,
		Email:  claims.Email,
	}, nil
}

// licenseVerifierAdapter adapts crypto.VerifyLicense to refresh_license.LicenseVerifier.
type licenseVerifierAdapter struct {
	publicKey ed25519.PublicKey
}

func (a *licenseVerifierAdapter) VerifyLicense(tokenString string) (*refresh_license.LicenseClaims, error) {
	claims, err := crypto.VerifyLicense(a.publicKey, tokenString)
	if err != nil {
		return nil, err
	}
	var expiresAt *time.Time
	if claims.ExpiresAt != nil {
		t := claims.ExpiresAt.Time
		expiresAt = &t
	}
	return &refresh_license.LicenseClaims{
		Subject:             claims.Subject,
		Email:               claims.Email,
		Tier:                claims.Tier,
		ExpiresAt:           expiresAt,
		ProxyStepsRemaining: claims.ProxyStepsRemaining,
	}, nil
}

// refreshTokenVerifierAdapter adapts crypto.AuthTokenSigner to refresh_auth.TokenVerifier.
type refreshTokenVerifierAdapter struct {
	signer *crypto.AuthTokenSigner
}

func (a *refreshTokenVerifierAdapter) VerifyRefreshToken(tokenString string) (*refresh_auth.RefreshClaims, error) {
	claims, err := a.signer.VerifyRefreshToken(tokenString)
	if err != nil {
		return nil, err
	}
	return &refresh_auth.RefreshClaims{
		UserID: claims.UserID,
	}, nil
}

// checkoutSessionAdapter bridges stripe.CheckoutClient to create_checkout.CheckoutSessionCreator.
type checkoutSessionAdapter struct {
	client *stripeinfra.CheckoutClient
}

func (a *checkoutSessionAdapter) CreateCheckoutSession(ctx context.Context, params create_checkout.CheckoutParams) (string, error) {
	return a.client.CreateCheckoutSession(ctx, stripeinfra.CheckoutParams{
		CustomerID: params.CustomerID,
		PriceID:    params.PriceID,
		Plan:       params.Plan,
		TrialDays:  params.TrialDays,
		SuccessURL: params.SuccessURL,
		CancelURL:  params.CancelURL,
		Metadata:   params.Metadata,
	})
}

// emailSenderI is a union of all email-sending interfaces needed by usecases.
type emailSenderI interface {
	SendTeamInvite(ctx context.Context, email, teamName, inviteToken string) error
	SendPasswordReset(ctx context.Context, email, token string) error
}

// noopSeatUpdater is used when Stripe is not configured.
type noopSeatUpdater struct{}

func (n *noopSeatUpdater) UpdateSubscriptionQuantity(_ context.Context, _ string, _ int64) error {
	return nil
}

// noopSubCanceller is used when Stripe is not configured.
type noopSubCanceller struct{}

func (n *noopSubCanceller) CancelSubscription(_ context.Context, _ string) error {
	return nil
}

func resolveConfigPath(configPath string) (string, error) {
	if filepath.IsAbs(configPath) {
		return configPath, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	return filepath.Join(wd, configPath), nil
}

func connectDB(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	slog.Info("Connected to database")

	return pool, nil
}

func runMigrations(databaseURL string) error {
	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("create migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, databaseURL)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

func parseEd25519Key(privateKeyHex string) (ed25519.PrivateKey, ed25519.PublicKey) {
	privBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		log.Fatalf("Invalid license private key hex: %v", err)
	}
	if len(privBytes) != ed25519.PrivateKeySize {
		log.Fatalf("Invalid Ed25519 private key length: got %d, want %d", len(privBytes), ed25519.PrivateKeySize)
	}
	privateKey := ed25519.PrivateKey(privBytes)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	return privateKey, publicKey
}

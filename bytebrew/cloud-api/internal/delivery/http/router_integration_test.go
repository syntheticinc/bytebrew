//go:build integration

package http_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

	httpdelivery "github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/delivery/http"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/infrastructure/crypto"
	pgstore "github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/infrastructure/postgres"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/activate"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/login"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/refresh_auth"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/refresh_license"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/register"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/migrations"
)

// newIntegrationTestEnv creates a full-stack test environment with a real PostgreSQL database.
func newIntegrationTestEnv(t *testing.T) *testEnv {
	t.Helper()
	ctx := context.Background()

	// Start PostgreSQL container
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

	// Run migrations
	source, err := iofs.New(migrations.FS, ".")
	require.NoError(t, err)

	m, err := migrate.NewWithSourceInstance("iofs", source, connStr)
	require.NoError(t, err)
	require.NoError(t, m.Up())

	// Create pool
	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	// Real repos
	userRepo := pgstore.NewUserRepository(pool)
	subRepo := pgstore.NewSubscriptionRepository(pool)

	// Real crypto
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	tokenSigner := crypto.NewAuthTokenSigner(
		[]byte("test-secret-at-least-32-bytes!!!"),
		15*time.Minute,
		7*24*time.Hour,
	)
	licenseSigner := crypto.NewLicenseSigner(priv)
	passwordHasher := crypto.NewBcryptHasher(4) // MinCost for speed

	// Usecases
	registerUC := register.New(userRepo, tokenSigner, passwordHasher)
	loginUC := login.New(userRepo, tokenSigner, passwordHasher)
	refreshAuthUC := refresh_auth.New(&refreshTokenVerifierAdapter{signer: tokenSigner}, tokenSigner, userRepo)
	activateUC := activate.New(subRepo, licenseSigner, nil)
	refreshUC := refresh_license.New(subRepo, licenseSigner, &licenseVerifierAdapter{publicKey: pub}, nil)

	// Handlers & Router
	authHandler := httpdelivery.NewAuthHandler(registerUC, loginUC, refreshAuthUC)
	licenseHandler := httpdelivery.NewLicenseHandler(activateUC, refreshUC)
	router := httpdelivery.NewRouter(httpdelivery.RouterConfig{
		AuthHandler:    authHandler,
		LicenseHandler: licenseHandler,
		TokenVerifier:  &tokenVerifierAdapter{signer: tokenSigner},
		CORSOrigins:    []string{"http://localhost:3000"},
	})

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return &testEnv{
		server:      server,
		publicKey:   pub,
		tokenSigner: tokenSigner,
		subCreator:  subRepo,
	}
}

func TestFullStackWithRealDB(t *testing.T) {
	env := newIntegrationTestEnv(t)
	email := "fullstack@example.com"
	password := "securepassword123"

	// Step 1: Register
	regResp, err := postJSON(env.server.URL+"/api/v1/auth/register", map[string]string{
		"email":    email,
		"password": password,
	})
	require.NoError(t, err)
	regBody := readBody(t, regResp)
	assert.Equal(t, http.StatusCreated, regResp.StatusCode, "register should return 201")

	var regData authResponseData
	require.NoError(t, json.Unmarshal(regBody, &regData))
	assert.NotEmpty(t, regData.Data.AccessToken)
	assert.NotEmpty(t, regData.Data.RefreshToken)
	assert.NotEmpty(t, regData.Data.UserID)

	// Step 2: Login
	loginResp, err := postJSON(env.server.URL+"/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": password,
	})
	require.NoError(t, err)
	loginBody := readBody(t, loginResp)
	assert.Equal(t, http.StatusOK, loginResp.StatusCode, "login should return 200")

	var loginData authResponseData
	require.NoError(t, json.Unmarshal(loginBody, &loginData))
	assert.NotEmpty(t, loginData.Data.AccessToken)
	assert.Equal(t, regData.Data.UserID, loginData.Data.UserID)

	// Step 3: Create subscription (register no longer creates one)
	env.createSubscription(t, regData.Data.UserID, domain.TierPersonal)

	// Step 4: Activate license
	activateResp, err := postJSON(
		env.server.URL+"/api/v1/license/activate",
		nil,
		"Authorization", "Bearer "+loginData.Data.AccessToken,
	)
	require.NoError(t, err)
	activateBody := readBody(t, activateResp)
	assert.Equal(t, http.StatusOK, activateResp.StatusCode, "activate should return 200")

	var activateData licenseResponseData
	require.NoError(t, json.Unmarshal(activateBody, &activateData))
	assert.NotEmpty(t, activateData.Data.License)

	// Step 5: Verify license JWT claims
	claims, err := crypto.VerifyLicense(env.publicKey, activateData.Data.License)
	require.NoError(t, err)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, string(domain.TierPersonal), claims.Tier)
	assert.Equal(t, loginData.Data.UserID, claims.Subject)
	assert.True(t, claims.Features.FullAutonomy)
	assert.Equal(t, -1, claims.Features.ParallelAgents)

	// Step 5: Refresh license (no subscription changes)
	refreshResp, err := postJSON(
		env.server.URL+"/api/v1/license/refresh",
		map[string]string{"current_license": activateData.Data.License},
		"Authorization", "Bearer "+loginData.Data.AccessToken,
	)
	require.NoError(t, err)
	refreshBody := readBody(t, refreshResp)
	assert.Equal(t, http.StatusOK, refreshResp.StatusCode, "refresh should return 200")

	var refreshData licenseResponseData
	require.NoError(t, json.Unmarshal(refreshBody, &refreshData))
	assert.Equal(t, activateData.Data.License, refreshData.Data.License, "unchanged subscription should return same license")

	// Step 6: Download license
	downloadResp, err := getWithAuth(
		env.server.URL+"/api/v1/license/download",
		loginData.Data.AccessToken,
	)
	require.NoError(t, err)
	downloadBody := readBody(t, downloadResp)
	assert.Equal(t, http.StatusOK, downloadResp.StatusCode, "download should return 200")
	assert.Equal(t, "application/jwt", downloadResp.Header.Get("Content-Type"))
	assert.Contains(t, downloadResp.Header.Get("Content-Disposition"), "license.jwt")

	// Verify downloaded JWT
	downloadClaims, err := crypto.VerifyLicense(env.publicKey, string(downloadBody))
	require.NoError(t, err)
	assert.Equal(t, email, downloadClaims.Email)
	assert.Equal(t, string(domain.TierPersonal), downloadClaims.Tier)
}

func TestFullStackDuplicateRegister(t *testing.T) {
	env := newIntegrationTestEnv(t)
	email := "dup-fullstack@example.com"
	password := "securepassword123"

	// First registration succeeds
	resp1, err := postJSON(env.server.URL+"/api/v1/auth/register", map[string]string{
		"email":    email,
		"password": password,
	})
	require.NoError(t, err)
	readBody(t, resp1)
	require.Equal(t, http.StatusCreated, resp1.StatusCode)

	// Second registration with same email fails
	resp2, err := postJSON(env.server.URL+"/api/v1/auth/register", map[string]string{
		"email":    email,
		"password": password,
	})
	require.NoError(t, err)
	body := readBody(t, resp2)
	assert.Equal(t, http.StatusConflict, resp2.StatusCode)

	var errData errorResponseData
	require.NoError(t, json.Unmarshal(body, &errData))
	assert.Equal(t, "ALREADY_EXISTS", errData.Error.Code)
}

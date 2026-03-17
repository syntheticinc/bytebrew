package http_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	httpdelivery "github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/delivery/http"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/delivery/http/middleware"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/infrastructure/crypto"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/activate"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/login"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/refresh_auth"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/refresh_license"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/register"
)

// --- In-memory repositories ---

type inMemoryUserRepo struct {
	mu    sync.Mutex
	users map[string]*domain.User // key: email
	seq   int
}

func newInMemoryUserRepo() *inMemoryUserRepo {
	return &inMemoryUserRepo{users: make(map[string]*domain.User)}
}

func (r *inMemoryUserRepo) Create(_ context.Context, user *domain.User) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[user.Email]; exists {
		return nil, fmt.Errorf("duplicate email")
	}

	r.seq++
	created := &domain.User{
		ID:           fmt.Sprintf("user-%d", r.seq),
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		CreatedAt:    user.CreatedAt,
	}
	r.users[user.Email] = created
	return created, nil
}

func (r *inMemoryUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.users[email], nil
}

type inMemorySubRepo struct {
	mu   sync.Mutex
	subs map[string]*domain.Subscription // key: userID
	seq  int
}

func newInMemorySubRepo() *inMemorySubRepo {
	return &inMemorySubRepo{subs: make(map[string]*domain.Subscription)}
}

func (r *inMemorySubRepo) Create(_ context.Context, sub *domain.Subscription) (*domain.Subscription, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.seq++
	sub.ID = fmt.Sprintf("sub-%d", r.seq)
	now := time.Now()
	sub.CreatedAt = now
	sub.UpdatedAt = now
	r.subs[sub.UserID] = sub
	return sub, nil
}

func (r *inMemorySubRepo) GetByUserID(_ context.Context, userID string) (*domain.Subscription, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.subs[userID], nil
}

// --- Adapters (same as in main.go but for tests) ---

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
		Subject:   claims.Subject,
		Email:     claims.Email,
		Tier:      claims.Tier,
		ExpiresAt: expiresAt,
	}, nil
}

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

// inMemoryUserRepoWithGetByID wraps inMemoryUserRepo and adds GetByID support.
type inMemoryUserRepoWithGetByID struct {
	*inMemoryUserRepo
}

func (r *inMemoryUserRepoWithGetByID) GetByID(_ context.Context, id string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, nil
}

// --- Test helpers ---

// subscriptionCreator is the subset of subscription repository needed by test helpers.
type subscriptionCreator interface {
	Create(ctx context.Context, sub *domain.Subscription) (*domain.Subscription, error)
}

type testEnv struct {
	server      *httptest.Server
	publicKey   ed25519.PublicKey
	tokenSigner *crypto.AuthTokenSigner
	subCreator  subscriptionCreator
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	userRepo := &inMemoryUserRepoWithGetByID{newInMemoryUserRepo()}
	subRepo := newInMemorySubRepo()
	tokenSigner := crypto.NewAuthTokenSigner(
		[]byte("test-secret-at-least-32-bytes!!!"),
		15*time.Minute,
		7*24*time.Hour,
	)
	licenseSigner := crypto.NewLicenseSigner(priv)
	passwordHasher := crypto.NewBcryptHasher(4) // MinCost for speed

	registerUC := register.New(userRepo, tokenSigner, passwordHasher)
	loginUC := login.New(userRepo, tokenSigner, passwordHasher)
	refreshAuthUC := refresh_auth.New(&refreshTokenVerifierAdapter{signer: tokenSigner}, tokenSigner, userRepo)
	activateUC := activate.New(subRepo, licenseSigner, nil)
	refreshUC := refresh_license.New(subRepo, licenseSigner, &licenseVerifierAdapter{publicKey: pub}, nil)

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

func postJSON(url string, body interface{}, headers ...string) (*http.Response, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	for i := 0; i < len(headers)-1; i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}

	return http.DefaultClient.Do(req)
}

func getWithAuth(url, token string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return http.DefaultClient.Do(req)
}

func readBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("failed to close response body: %v", err)
		}
	}()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return b
}

func (e *testEnv) createSubscription(t *testing.T, userID string, tier domain.LicenseTier) {
	t.Helper()
	_, err := e.subCreator.Create(context.Background(), &domain.Subscription{
		UserID: userID,
		Tier:   tier,
		Status: domain.StatusActive,
	})
	require.NoError(t, err)
}

type authResponseData struct {
	Data struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		UserID       string `json:"user_id"`
	} `json:"data"`
}

type licenseResponseData struct {
	Data struct {
		License string `json:"license"`
	} `json:"data"`
}

type errorResponseData struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// --- Tests ---

func TestHealthCheck(t *testing.T) {
	env := newTestEnv(t)

	resp, err := http.Get(env.server.URL + "/health")
	require.NoError(t, err)
	body := readBody(t, resp)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t, `{"status":"ok"}`, string(body))
}

func TestRegisterLoginActivate(t *testing.T) {
	env := newTestEnv(t)
	email := "test@example.com"
	password := "securepassword123"

	// Step 1: Register
	regResp, err := postJSON(env.server.URL+"/api/v1/auth/register", map[string]string{
		"email":    email,
		"password": password,
	})
	require.NoError(t, err)
	body := readBody(t, regResp)
	assert.Equal(t, http.StatusCreated, regResp.StatusCode)

	var regData authResponseData
	require.NoError(t, json.Unmarshal(body, &regData))
	assert.NotEmpty(t, regData.Data.AccessToken)
	assert.NotEmpty(t, regData.Data.RefreshToken)
	assert.NotEmpty(t, regData.Data.UserID)

	// Step 2: Login with same credentials
	loginResp, err := postJSON(env.server.URL+"/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": password,
	})
	require.NoError(t, err)
	loginBody := readBody(t, loginResp)
	assert.Equal(t, http.StatusOK, loginResp.StatusCode)

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
	assert.Equal(t, http.StatusOK, activateResp.StatusCode)

	var activateData licenseResponseData
	require.NoError(t, json.Unmarshal(activateBody, &activateData))
	assert.NotEmpty(t, activateData.Data.License)

	// Step 5: Verify license JWT with public key
	claims, err := crypto.VerifyLicense(env.publicKey, activateData.Data.License)
	require.NoError(t, err)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, string(domain.TierPersonal), claims.Tier)
	assert.Equal(t, loginData.Data.UserID, claims.Subject)
	assert.True(t, claims.Features.FullAutonomy)
	assert.Equal(t, -1, claims.Features.ParallelAgents)
}

func TestRegisterDuplicateEmail(t *testing.T) {
	env := newTestEnv(t)
	email := "duplicate@example.com"
	password := "securepassword123"

	// First registration succeeds
	resp1, err := postJSON(env.server.URL+"/api/v1/auth/register", map[string]string{
		"email":    email,
		"password": password,
	})
	require.NoError(t, err)
	readBody(t, resp1)
	assert.Equal(t, http.StatusCreated, resp1.StatusCode)

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

func TestLoginWrongPassword(t *testing.T) {
	env := newTestEnv(t)
	email := "user@example.com"
	password := "correctpassword123"

	// Register
	regResp, err := postJSON(env.server.URL+"/api/v1/auth/register", map[string]string{
		"email":    email,
		"password": password,
	})
	require.NoError(t, err)
	readBody(t, regResp)
	require.Equal(t, http.StatusCreated, regResp.StatusCode)

	// Login with wrong password
	loginResp, err := postJSON(env.server.URL+"/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": "wrongpassword123",
	})
	require.NoError(t, err)
	body := readBody(t, loginResp)
	assert.Equal(t, http.StatusUnauthorized, loginResp.StatusCode)

	var errData errorResponseData
	require.NoError(t, json.Unmarshal(body, &errData))
	assert.Equal(t, "UNAUTHORIZED", errData.Error.Code)
}

func TestActivateWithoutAuth(t *testing.T) {
	env := newTestEnv(t)

	// Activate without Bearer token
	resp, err := postJSON(env.server.URL+"/api/v1/license/activate", nil)
	require.NoError(t, err)
	body := readBody(t, resp)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	var errData errorResponseData
	require.NoError(t, json.Unmarshal(body, &errData))
	assert.Equal(t, "UNAUTHORIZED", errData.Error.Code)
}

func TestRefreshLicenseUnchanged(t *testing.T) {
	env := newTestEnv(t)
	email := "refresh@example.com"
	password := "securepassword123"

	// Register
	regResp, err := postJSON(env.server.URL+"/api/v1/auth/register", map[string]string{
		"email":    email,
		"password": password,
	})
	require.NoError(t, err)
	regBody := readBody(t, regResp)
	require.Equal(t, http.StatusCreated, regResp.StatusCode)

	var regData authResponseData
	require.NoError(t, json.Unmarshal(regBody, &regData))

	// Create subscription (register no longer creates one)
	env.createSubscription(t, regData.Data.UserID, domain.TierPersonal)

	// Activate to get license JWT
	activateResp, err := postJSON(
		env.server.URL+"/api/v1/license/activate",
		nil,
		"Authorization", "Bearer "+regData.Data.AccessToken,
	)
	require.NoError(t, err)
	activateBody := readBody(t, activateResp)
	require.Equal(t, http.StatusOK, activateResp.StatusCode)

	var activateData licenseResponseData
	require.NoError(t, json.Unmarshal(activateBody, &activateData))
	license1 := activateData.Data.License

	// Refresh with the same license (no subscription changes)
	refreshResp, err := postJSON(
		env.server.URL+"/api/v1/license/refresh",
		map[string]string{"current_license": license1},
		"Authorization", "Bearer "+regData.Data.AccessToken,
	)
	require.NoError(t, err)
	refreshBody := readBody(t, refreshResp)
	assert.Equal(t, http.StatusOK, refreshResp.StatusCode)

	var refreshData licenseResponseData
	require.NoError(t, json.Unmarshal(refreshBody, &refreshData))

	// Same license returned (no changes to subscription)
	assert.Equal(t, license1, refreshData.Data.License)
}

func TestLicenseDownload(t *testing.T) {
	env := newTestEnv(t)
	email := "download@example.com"
	password := "securepassword123"

	// Register
	regResp, err := postJSON(env.server.URL+"/api/v1/auth/register", map[string]string{
		"email":    email,
		"password": password,
	})
	require.NoError(t, err)
	regBody := readBody(t, regResp)
	require.Equal(t, http.StatusCreated, regResp.StatusCode)

	var regData authResponseData
	require.NoError(t, json.Unmarshal(regBody, &regData))

	// Create subscription (register no longer creates one)
	env.createSubscription(t, regData.Data.UserID, domain.TierPersonal)

	// Download license
	resp, err := getWithAuth(
		env.server.URL+"/api/v1/license/download",
		regData.Data.AccessToken,
	)
	require.NoError(t, err)
	body := readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/jwt", resp.Header.Get("Content-Type"))
	assert.Contains(t, resp.Header.Get("Content-Disposition"), "license.jwt")

	// Verify the downloaded JWT
	claims, err := crypto.VerifyLicense(env.publicKey, string(body))
	require.NoError(t, err)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, string(domain.TierPersonal), claims.Tier)
}

func TestLoginNonexistentUser(t *testing.T) {
	env := newTestEnv(t)

	resp, err := postJSON(env.server.URL+"/api/v1/auth/login", map[string]string{
		"email":    "nobody@example.com",
		"password": "somepassword123",
	})
	require.NoError(t, err)
	body := readBody(t, resp)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	var errData errorResponseData
	require.NoError(t, json.Unmarshal(body, &errData))
	assert.Equal(t, "UNAUTHORIZED", errData.Error.Code)
}

func TestRegisterInvalidInput(t *testing.T) {
	env := newTestEnv(t)

	tests := []struct {
		name     string
		body     map[string]string
		wantCode string
	}{
		{
			name:     "empty email",
			body:     map[string]string{"email": "", "password": "securepassword123"},
			wantCode: "INVALID_INPUT",
		},
		{
			name:     "short password",
			body:     map[string]string{"email": "user@example.com", "password": "short"},
			wantCode: "INVALID_INPUT",
		},
		{
			name:     "invalid email format",
			body:     map[string]string{"email": "not-an-email", "password": "securepassword123"},
			wantCode: "INVALID_INPUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := postJSON(env.server.URL+"/api/v1/auth/register", tt.body)
			require.NoError(t, err)
			body := readBody(t, resp)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

			var errData errorResponseData
			require.NoError(t, json.Unmarshal(body, &errData))
			assert.Equal(t, tt.wantCode, errData.Error.Code)
		})
	}
}

func TestProtectedRoutesRequireAuth(t *testing.T) {
	env := newTestEnv(t)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/license/activate"},
		{http.MethodPost, "/api/v1/license/refresh"},
		{http.MethodGet, "/api/v1/license/download"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req, err := http.NewRequest(route.method, env.server.URL+route.path, nil)
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			readBody(t, resp)

			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})
	}
}

func TestLicenseStatus(t *testing.T) {
	env := newTestEnv(t)
	email := "status@example.com"
	password := "securepassword123"

	// Register and get access token
	regResp, err := postJSON(env.server.URL+"/api/v1/auth/register", map[string]string{
		"email":    email,
		"password": password,
	})
	require.NoError(t, err)
	regBody := readBody(t, regResp)
	require.Equal(t, http.StatusCreated, regResp.StatusCode)

	var regData authResponseData
	require.NoError(t, json.Unmarshal(regBody, &regData))

	// Create subscription (register no longer creates one)
	env.createSubscription(t, regData.Data.UserID, domain.TierPersonal)

	// Activate to get license JWT
	activateResp, err := postJSON(
		env.server.URL+"/api/v1/license/activate",
		nil,
		"Authorization", "Bearer "+regData.Data.AccessToken,
	)
	require.NoError(t, err)
	activateBody := readBody(t, activateResp)
	require.Equal(t, http.StatusOK, activateResp.StatusCode)

	var activateData licenseResponseData
	require.NoError(t, json.Unmarshal(activateBody, &activateData))

	// Get license status (parses JWT payload without verification)
	statusResp, err := getWithAuth(
		env.server.URL+"/api/v1/license/status?license="+activateData.Data.License,
		regData.Data.AccessToken,
	)
	require.NoError(t, err)
	statusBody := readBody(t, statusResp)
	assert.Equal(t, http.StatusOK, statusResp.StatusCode)

	// Parse the status response
	var statusResult map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(statusBody, &statusResult))
	assert.Contains(t, string(statusResult["data"]), email)
	assert.Contains(t, string(statusResult["data"]), string(domain.TierPersonal))
}

func TestActivateWithoutSubscription(t *testing.T) {
	env := newTestEnv(t)

	// Register (no subscription created)
	regResp, err := postJSON(env.server.URL+"/api/v1/auth/register", map[string]string{
		"email":    "nosub@example.com",
		"password": "securepassword123",
	})
	require.NoError(t, err)
	regBody := readBody(t, regResp)
	require.Equal(t, http.StatusCreated, regResp.StatusCode)

	var regData authResponseData
	require.NoError(t, json.Unmarshal(regBody, &regData))

	// Activate without subscription should fail
	activateResp, err := postJSON(
		env.server.URL+"/api/v1/license/activate",
		nil,
		"Authorization", "Bearer "+regData.Data.AccessToken,
	)
	require.NoError(t, err)
	readBody(t, activateResp)
	assert.Equal(t, http.StatusForbidden, activateResp.StatusCode)
}

func TestInternalErrorDoesNotLeakDetails(t *testing.T) {
	env := newTestEnv(t)

	// Register
	regResp, err := postJSON(env.server.URL+"/api/v1/auth/register", map[string]string{
		"email":    "leak@example.com",
		"password": "securepassword123",
	})
	require.NoError(t, err)
	regBody := readBody(t, regResp)
	require.Equal(t, http.StatusCreated, regResp.StatusCode)

	var regData authResponseData
	require.NoError(t, json.Unmarshal(regBody, &regData))

	// Refresh with invalid license should return UNAUTHORIZED, not internal error details
	refreshResp, err := postJSON(
		env.server.URL+"/api/v1/license/refresh",
		map[string]string{"current_license": "invalid.jwt.token"},
		"Authorization", "Bearer "+regData.Data.AccessToken,
	)
	require.NoError(t, err)
	body := readBody(t, refreshResp)
	assert.Equal(t, http.StatusUnauthorized, refreshResp.StatusCode)

	var errData errorResponseData
	require.NoError(t, json.Unmarshal(body, &errData))
	assert.Equal(t, "UNAUTHORIZED", errData.Error.Code)
	// Should not contain internal error details like stack traces or db errors
	assert.NotContains(t, errData.Error.Message, "parse license")
	assert.NotContains(t, errData.Error.Message, "stack")
	assert.NotContains(t, errData.Error.Message, "sql")
}

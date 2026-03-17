package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

func generateTestKeyPair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key pair: %v", err)
	}
	return pub, priv
}

func TestLicenseSigner_Roundtrip(t *testing.T) {
	pub, priv := generateTestKeyPair(t)
	signer := NewLicenseSigner(priv)

	expiry := time.Now().Add(30 * 24 * time.Hour)
	grace := domain.GraceFromExpiry(expiry)
	info := domain.LicenseInfo{
		UserID:              "user-789",
		Email:               "bob@example.com",
		Tier:                domain.TierPersonal,
		ExpiresAt:           expiry,
		GraceUntil:          grace,
		Features:            domain.FeaturesForTier(domain.TierPersonal),
		ProxyStepsRemaining: 253,
		ProxyStepsLimit:     300,
		BYOKEnabled:         true,
	}

	tokenStr, err := signer.SignLicense(info)
	if err != nil {
		t.Fatalf("SignLicense: %v", err)
	}

	claims, err := VerifyLicense(pub, tokenStr)
	if err != nil {
		t.Fatalf("VerifyLicense: %v", err)
	}

	if claims.Subject != "user-789" {
		t.Errorf("Subject = %q, want %q", claims.Subject, "user-789")
	}
	if claims.Email != "bob@example.com" {
		t.Errorf("Email = %q, want %q", claims.Email, "bob@example.com")
	}
	if claims.Tier != "personal" {
		t.Errorf("Tier = %q, want %q", claims.Tier, "personal")
	}
	if claims.Issuer != "bytebrew-cloud-api" {
		t.Errorf("Issuer = %q, want %q", claims.Issuer, "bytebrew-cloud-api")
	}
	if claims.GraceUntil == nil {
		t.Fatal("GraceUntil is nil")
	}
	if claims.ExpiresAt == nil {
		t.Fatal("ExpiresAt is nil")
	}
	if claims.ProxyStepsRemaining != 253 {
		t.Errorf("ProxyStepsRemaining = %d, want 253", claims.ProxyStepsRemaining)
	}
	if claims.ProxyStepsLimit != 300 {
		t.Errorf("ProxyStepsLimit = %d, want 300", claims.ProxyStepsLimit)
	}
	if !claims.BYOKEnabled {
		t.Error("BYOKEnabled should be true")
	}
}

func TestLicenseSigner_TamperedToken_Fails(t *testing.T) {
	pub, priv := generateTestKeyPair(t)
	signer := NewLicenseSigner(priv)

	info := domain.LicenseInfo{
		UserID:     "user-789",
		Email:      "bob@example.com",
		Tier:       domain.TierTeams,
		ExpiresAt:  time.Now().Add(30 * 24 * time.Hour),
		GraceUntil: time.Now().Add(60 * 24 * time.Hour),
		Features:   domain.FeaturesForTier(domain.TierTeams),
	}

	tokenStr, err := signer.SignLicense(info)
	if err != nil {
		t.Fatalf("SignLicense: %v", err)
	}

	// Flip a byte in the middle of the signature portion to ensure
	// the tampering actually changes the decoded signature bytes.
	tampered := []byte(tokenStr)
	sigIdx := len(tampered) - 20 // well inside the signature
	if tampered[sigIdx] == 'X' {
		tampered[sigIdx] = 'Y'
	} else {
		tampered[sigIdx] = 'X'
	}

	_, err = VerifyLicense(pub, string(tampered))
	if err == nil {
		t.Fatal("expected error for tampered token, got nil")
	}
}

func TestLicenseSigner_WrongKey_Fails(t *testing.T) {
	_, priv := generateTestKeyPair(t)
	otherPub, _ := generateTestKeyPair(t)
	signer := NewLicenseSigner(priv)

	info := domain.LicenseInfo{
		UserID:     "user-789",
		Email:      "bob@example.com",
		Tier:       domain.TierPersonal,
		ExpiresAt:  time.Now().Add(30 * 24 * time.Hour),
		GraceUntil: time.Now().Add(60 * 24 * time.Hour),
		Features:   domain.FeaturesForTier(domain.TierPersonal),
	}

	tokenStr, err := signer.SignLicense(info)
	if err != nil {
		t.Fatalf("SignLicense: %v", err)
	}

	_, err = VerifyLicense(otherPub, tokenStr)
	if err == nil {
		t.Fatal("expected error when verifying with wrong public key, got nil")
	}
}

func TestLicenseSigner_HS256Rejected(t *testing.T) {
	pub, _ := generateTestKeyPair(t)

	// Create an HS256 token and try to verify as EdDSA.
	claims := LicenseClaims{
		Email: "bob@example.com",
		Tier:  "pro",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-789",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "bytebrew-cloud-api",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte("some-hmac-secret"))
	if err != nil {
		t.Fatalf("sign HS256 token: %v", err)
	}

	_, err = VerifyLicense(pub, tokenStr)
	if err == nil {
		t.Fatal("expected error for HS256 token verified as EdDSA, got nil")
	}
}

func TestLicenseSigner_TrialTierFeatures(t *testing.T) {
	pub, priv := generateTestKeyPair(t)
	signer := NewLicenseSigner(priv)

	features := domain.FeaturesForTier(domain.TierTrial)
	info := domain.LicenseInfo{
		UserID:     "user-trial",
		Email:      "trial@example.com",
		Tier:       domain.TierTrial,
		ExpiresAt:  time.Now().Add(30 * 24 * time.Hour),
		GraceUntil: time.Now().Add(60 * 24 * time.Hour),
		Features:   features,
	}

	tokenStr, err := signer.SignLicense(info)
	if err != nil {
		t.Fatalf("SignLicense: %v", err)
	}

	claims, err := VerifyLicense(pub, tokenStr)
	if err != nil {
		t.Fatalf("VerifyLicense: %v", err)
	}

	if !claims.Features.FullAutonomy {
		t.Error("Trial tier: FullAutonomy should be true")
	}
	if claims.Features.ParallelAgents != -1 {
		t.Errorf("Trial tier: ParallelAgents = %d, want -1 (unlimited)", claims.Features.ParallelAgents)
	}
	if !claims.Features.ExploreCodebase {
		t.Error("Trial tier: ExploreCodebase should be true")
	}
	if !claims.Features.TraceSymbol {
		t.Error("Trial tier: TraceSymbol should be true")
	}
	if !claims.Features.CodebaseIndexing {
		t.Error("Trial tier: CodebaseIndexing should be true")
	}
}

func TestLicenseFeatures_RoundTrip(t *testing.T) {
	pub, priv := generateTestKeyPair(t)
	signer := NewLicenseSigner(priv)

	// Use custom (non-default) feature values to ensure every field
	// survives the Sign → JWT → Verify round-trip independently.
	features := domain.LicenseFeatures{
		FullAutonomy:     true,
		ParallelAgents:   3,
		ExploreCodebase:  false,
		TraceSymbol:      false,
		CodebaseIndexing: true,
	}

	info := domain.LicenseInfo{
		UserID:              "user-feat-rt",
		Email:               "features@example.com",
		Tier:                domain.TierPersonal,
		ExpiresAt:           time.Now().Add(30 * 24 * time.Hour),
		GraceUntil:          time.Now().Add(33 * 24 * time.Hour),
		Features:            features,
		ProxyStepsRemaining: 100,
		ProxyStepsLimit:     300,
		BYOKEnabled:         true,
		MaxSeats:            1,
	}

	tokenStr, err := signer.SignLicense(info)
	if err != nil {
		t.Fatalf("SignLicense: %v", err)
	}

	claims, err := VerifyLicense(pub, tokenStr)
	if err != nil {
		t.Fatalf("VerifyLicense: %v", err)
	}

	// Verify every LicenseFeatures field individually.
	if claims.Features.FullAutonomy != features.FullAutonomy {
		t.Errorf("FullAutonomy = %v, want %v", claims.Features.FullAutonomy, features.FullAutonomy)
	}
	if claims.Features.ParallelAgents != features.ParallelAgents {
		t.Errorf("ParallelAgents = %d, want %d", claims.Features.ParallelAgents, features.ParallelAgents)
	}
	if claims.Features.ExploreCodebase != features.ExploreCodebase {
		t.Errorf("ExploreCodebase = %v, want %v", claims.Features.ExploreCodebase, features.ExploreCodebase)
	}
	if claims.Features.TraceSymbol != features.TraceSymbol {
		t.Errorf("TraceSymbol = %v, want %v", claims.Features.TraceSymbol, features.TraceSymbol)
	}
	if claims.Features.CodebaseIndexing != features.CodebaseIndexing {
		t.Errorf("CodebaseIndexing = %v, want %v", claims.Features.CodebaseIndexing, features.CodebaseIndexing)
	}

	// Also verify the top-level LicenseInfo fields survived.
	if claims.ProxyStepsRemaining != 100 {
		t.Errorf("ProxyStepsRemaining = %d, want 100", claims.ProxyStepsRemaining)
	}
	if claims.ProxyStepsLimit != 300 {
		t.Errorf("ProxyStepsLimit = %d, want 300", claims.ProxyStepsLimit)
	}
	if !claims.BYOKEnabled {
		t.Error("BYOKEnabled should be true")
	}
	if claims.MaxSeats != 1 {
		t.Errorf("MaxSeats = %d, want 1", claims.MaxSeats)
	}
}

func TestLicenseSigner_PersonalTierFeatures(t *testing.T) {
	pub, priv := generateTestKeyPair(t)
	signer := NewLicenseSigner(priv)

	features := domain.FeaturesForTier(domain.TierPersonal)
	info := domain.LicenseInfo{
		UserID:     "user-personal",
		Email:      "personal@example.com",
		Tier:       domain.TierPersonal,
		ExpiresAt:  time.Now().Add(30 * 24 * time.Hour),
		GraceUntil: time.Now().Add(60 * 24 * time.Hour),
		Features:   features,
	}

	tokenStr, err := signer.SignLicense(info)
	if err != nil {
		t.Fatalf("SignLicense: %v", err)
	}

	claims, err := VerifyLicense(pub, tokenStr)
	if err != nil {
		t.Fatalf("VerifyLicense: %v", err)
	}

	if !claims.Features.FullAutonomy {
		t.Error("Personal tier: FullAutonomy should be true")
	}
	if claims.Features.ParallelAgents != -1 {
		t.Errorf("Personal tier: ParallelAgents = %d, want -1 (unlimited)", claims.Features.ParallelAgents)
	}
	if !claims.Features.ExploreCodebase {
		t.Error("Personal tier: ExploreCodebase should be true")
	}
	if !claims.Features.TraceSymbol {
		t.Error("Personal tier: TraceSymbol should be true")
	}
	if !claims.Features.CodebaseIndexing {
		t.Error("Personal tier: CodebaseIndexing should be true")
	}
}

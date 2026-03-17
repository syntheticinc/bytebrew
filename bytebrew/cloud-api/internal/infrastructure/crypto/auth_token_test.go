package crypto

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuthTokenSigner_AccessToken_Roundtrip(t *testing.T) {
	signer := NewAuthTokenSigner([]byte("test-secret"), 15*time.Minute, 7*24*time.Hour)

	tokenStr, err := signer.SignAccessToken("user-123", "alice@example.com")
	if err != nil {
		t.Fatalf("SignAccessToken: %v", err)
	}

	claims, err := signer.VerifyAccessToken(tokenStr)
	if err != nil {
		t.Fatalf("VerifyAccessToken: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-123")
	}
	if claims.Email != "alice@example.com" {
		t.Errorf("Email = %q, want %q", claims.Email, "alice@example.com")
	}
	if claims.Issuer != "bytebrew-cloud-api" {
		t.Errorf("Issuer = %q, want %q", claims.Issuer, "bytebrew-cloud-api")
	}
}

func TestAuthTokenSigner_RefreshToken_Roundtrip(t *testing.T) {
	signer := NewAuthTokenSigner([]byte("test-secret"), 15*time.Minute, 7*24*time.Hour)

	tokenStr, err := signer.SignRefreshToken("user-456")
	if err != nil {
		t.Fatalf("SignRefreshToken: %v", err)
	}

	claims, err := signer.VerifyRefreshToken(tokenStr)
	if err != nil {
		t.Fatalf("VerifyRefreshToken: %v", err)
	}

	if claims.UserID != "user-456" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-456")
	}
	if claims.Issuer != "bytebrew-cloud-api" {
		t.Errorf("Issuer = %q, want %q", claims.Issuer, "bytebrew-cloud-api")
	}
}

func TestAuthTokenSigner_TamperedToken_Fails(t *testing.T) {
	signer := NewAuthTokenSigner([]byte("test-secret"), 15*time.Minute, 7*24*time.Hour)

	tokenStr, err := signer.SignAccessToken("user-123", "alice@example.com")
	if err != nil {
		t.Fatalf("SignAccessToken: %v", err)
	}

	// Tamper with token by flipping a character in the signature.
	tampered := []byte(tokenStr)
	lastChar := tampered[len(tampered)-1]
	if lastChar == 'A' {
		tampered[len(tampered)-1] = 'B'
	} else {
		tampered[len(tampered)-1] = 'A'
	}

	_, err = signer.VerifyAccessToken(string(tampered))
	if err == nil {
		t.Fatal("expected error for tampered token, got nil")
	}
}

func TestAuthTokenSigner_ExpiredToken_Fails(t *testing.T) {
	// Use 1ms TTL so the token expires almost immediately.
	signer := NewAuthTokenSigner([]byte("test-secret"), 1*time.Millisecond, 1*time.Millisecond)

	accessToken, err := signer.SignAccessToken("user-123", "alice@example.com")
	if err != nil {
		t.Fatalf("SignAccessToken: %v", err)
	}
	refreshToken, err := signer.SignRefreshToken("user-123")
	if err != nil {
		t.Fatalf("SignRefreshToken: %v", err)
	}

	time.Sleep(5 * time.Millisecond)

	_, err = signer.VerifyAccessToken(accessToken)
	if err == nil {
		t.Fatal("expected error for expired access token, got nil")
	}

	_, err = signer.VerifyRefreshToken(refreshToken)
	if err == nil {
		t.Fatal("expected error for expired refresh token, got nil")
	}
}

func TestAuthTokenSigner_WrongSecret_Fails(t *testing.T) {
	signer1 := NewAuthTokenSigner([]byte("secret-one"), 15*time.Minute, 7*24*time.Hour)
	signer2 := NewAuthTokenSigner([]byte("secret-two"), 15*time.Minute, 7*24*time.Hour)

	tokenStr, err := signer1.SignAccessToken("user-123", "alice@example.com")
	if err != nil {
		t.Fatalf("SignAccessToken: %v", err)
	}

	_, err = signer2.VerifyAccessToken(tokenStr)
	if err == nil {
		t.Fatal("expected error when verifying with wrong secret, got nil")
	}
}

func TestAuthTokenSigner_WrongSigningMethod_Rejected(t *testing.T) {
	signer := NewAuthTokenSigner([]byte("test-secret"), 15*time.Minute, 7*24*time.Hour)

	// Create a token signed with "none" method (no signature).
	claims := AccessClaims{
		UserID: "user-123",
		Email:  "alice@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "bytebrew-cloud-api",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenStr, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none token: %v", err)
	}

	_, err = signer.VerifyAccessToken(tokenStr)
	if err == nil {
		t.Fatal("expected error for 'none' signing method, got nil")
	}
}

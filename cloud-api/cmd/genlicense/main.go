package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/cloud-api/internal/infrastructure/crypto"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: genlicense <private_key_hex> <email> [tier] [days]\n")
		fmt.Fprintf(os.Stderr, "  tier: personal (default), teams, trial\n")
		fmt.Fprintf(os.Stderr, "  days: license duration in days (default: 365)\n")
		os.Exit(1)
	}

	privateKeyHex := os.Args[1]
	email := os.Args[2]

	tier := domain.LicenseTier("personal")
	if len(os.Args) > 3 {
		tier = domain.LicenseTier(os.Args[3])
	}

	days := 365
	if len(os.Args) > 4 {
		fmt.Sscanf(os.Args[4], "%d", &days)
	}

	privBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		log.Fatalf("Invalid private key hex: %v", err)
	}
	privateKey := ed25519.PrivateKey(privBytes)

	expiresAt := time.Now().Add(time.Duration(days) * 24 * time.Hour)

	info := domain.LicenseInfo{
		UserID:              email,
		Email:               email,
		Tier:                tier,
		ExpiresAt:           expiresAt,
		GraceUntil:          domain.GraceFromExpiry(expiresAt),
		Features:            domain.FeaturesForTier(tier),
		ProxyStepsRemaining: 999999,
		ProxyStepsLimit:     999999,
		BYOKEnabled:         true,
		MaxSeats:            1,
	}

	signer := crypto.NewLicenseSigner(privateKey)
	jwt, err := signer.SignLicense(info)
	if err != nil {
		log.Fatalf("Failed to sign license: %v", err)
	}

	fmt.Print(jwt)
}

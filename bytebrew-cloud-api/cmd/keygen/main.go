package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--help" {
		fmt.Println("Usage: keygen")
		fmt.Println("  Generates an Ed25519 key pair for license signing.")
		fmt.Println("  Private key -> set as LICENSE_PRIVATE_KEY_HEX in .env")
		fmt.Println("  Public key  -> embed in bytebrew-srv via -ldflags")
		return
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate key pair: %v", err)
	}

	fmt.Printf("PRIVATE_KEY (hex, 128 chars -- keep secret!):\n%s\n\n", hex.EncodeToString(priv))
	fmt.Printf("PUBLIC_KEY (hex, 64 chars -- embed in binaries):\n%s\n", hex.EncodeToString(pub))
}

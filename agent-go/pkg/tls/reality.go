package tls

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// RealityKeyPair holds x25519 public/private keys and short ID
type RealityKeyPair struct {
	PrivateKey string
	PublicKey  string
	ShortID    string
}

// GenerateRealityKeyPair creates standard x25519 keypair for VLESS Reality
func GenerateRealityKeyPair() (*RealityKeyPair, error) {
	curve := ecdh.X25519()
	privKey, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate x25519 private key: %w", err)
	}

	pubKey := privKey.PublicKey()

	// Reality uses RawURLEncoding (no padding, url safe)
	privBase64 := base64.RawURLEncoding.EncodeToString(privKey.Bytes())
	pubBase64 := base64.RawURLEncoding.EncodeToString(pubKey.Bytes())

	// Generate 16-hex short ID (8 bytes)
	shortBytes := make([]byte, 8)
	if _, err := rand.Read(shortBytes); err != nil {
		return nil, fmt.Errorf("failed to generate short ID: %w", err)
	}
	shortID := hex.EncodeToString(shortBytes)

	return &RealityKeyPair{
		PrivateKey: privBase64,
		PublicKey:  pubBase64,
		ShortID:    shortID,
	}, nil
}

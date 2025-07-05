package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// PKCE represents the PKCE (Proof Key for Code Exchange) parameters
type PKCE struct {
	Verifier  string `json:"verifier"`
	Challenge string `json:"challenge"`
}

// GeneratePKCE generates a new PKCE verifier and challenge pair
func GeneratePKCE() (*PKCE, error) {
	// Generate random bytes for verifier
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Create base64url encoded verifier (without padding)
	verifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	// Create SHA256 hash of verifier
	hash := sha256.Sum256([]byte(verifier))
	
	// Create base64url encoded challenge (without padding)
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	return &PKCE{
		Verifier:  verifier,
		Challenge: challenge,
	}, nil
}
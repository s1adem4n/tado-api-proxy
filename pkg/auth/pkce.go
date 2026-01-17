package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// GenerateCodeVerifier creates code verifier for PKCE
func GenerateCodeVerifier() (string, error) {
	// 96 bytes encodes to exactly 128 characters in base64url
	bytes := make([]byte, 96)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	verifier := base64.RawURLEncoding.EncodeToString(bytes)
	return verifier, nil
}

// GenerateCodeChallenge creates a code challenge from a verifier
// using the S256 method (SHA256 hash).
func GenerateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])
	return challenge
}

// GeneratePKCE generates both a code verifier and its corresponding challenge
// for use in OAuth 2.0 PKCE flow.
func GeneratePKCE() (verifier, challenge string, err error) {
	verifier, err = GenerateCodeVerifier()
	if err != nil {
		return "", "", err
	}
	challenge = GenerateCodeChallenge(verifier)
	return verifier, challenge, nil
}

package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenerateState creates a cryptographically random state parameter
// for CSRF protection in OAuth flows.
func GenerateState() (string, error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random state: %w", err)
	}

	// Base64 URL encode without padding
	state := base64.RawURLEncoding.EncodeToString(bytes)
	return state, nil
}

// ValidateState compares the original state with the callback state
// to prevent CSRF attacks.
func ValidateState(original, callback string) bool {
	return original != "" && original == callback
}

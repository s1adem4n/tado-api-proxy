package tokens

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// TokenAuthProvider defines the interface for token authentication operations.
// This decouples the token manager from the tado client.
type TokenAuthProvider interface {
	// RefreshToken refreshes the given token using the refresh_token grant.
	RefreshToken(ctx context.Context, clientID, refreshToken, platform string) (*TokenResult, error)
	// Authorize performs password grant authentication.
	Authorize(ctx context.Context, clientID, redirectURI, scope, email, password, platform string) (*TokenResult, error)
}

// TokenResult contains the result of a token operation.
type TokenResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// Manager handles token lifecycle including refresh and access.
type Manager struct {
	app          core.App
	authProvider TokenAuthProvider

	// Per-token mutex to prevent concurrent refresh operations
	tokenMutexes   map[string]*sync.Mutex
	tokenMutexesMu sync.Mutex
}

// NewManager creates a new token manager.
func NewManager(app core.App, authProvider TokenAuthProvider) *Manager {
	return &Manager{
		app:          app,
		authProvider: authProvider,
		tokenMutexes: make(map[string]*sync.Mutex),
	}
}

// getTokenMutex returns the mutex for a specific token, creating one if needed.
func (m *Manager) getTokenMutex(tokenID string) *sync.Mutex {
	m.tokenMutexesMu.Lock()
	defer m.tokenMutexesMu.Unlock()

	if mu, ok := m.tokenMutexes[tokenID]; ok {
		return mu
	}

	mu := &sync.Mutex{}
	m.tokenMutexes[tokenID] = mu
	return mu
}

// tryFixToken attempts to fix an invalid token.
func (m *Manager) tryFixToken(ctx context.Context, tokenRecord *core.Record) error {
	client, err := m.app.FindRecordById("clients", tokenRecord.GetString("client"))
	if err != nil {
		return err
	}

	fmt.Println("tryFixToken")
	if client.GetString("type") == "passwordGrant" {
		fmt.Println("password grant")
		return m.fixPasswordGrantToken(ctx, tokenRecord, client)
	} else if client.GetString("type") == "deviceCode" {
		return m.fixDeviceCodeToken(tokenRecord)
	}

	return nil
}

// fixPasswordGrantToken re-authenticates using password grant.
func (m *Manager) fixPasswordGrantToken(ctx context.Context, tokenRecord *core.Record, clientRecord *core.Record) error {
	mu := m.getTokenMutex(tokenRecord.Id)
	mu.Lock()
	defer mu.Unlock()

	account, err := m.app.FindRecordById("accounts", tokenRecord.GetString("account"))
	if err != nil {
		return err
	}

	newToken, err := m.authProvider.Authorize(
		ctx,
		clientRecord.GetString("clientID"),
		clientRecord.GetString("redirectURI"),
		clientRecord.GetString("scope"),
		account.GetString("email"),
		account.GetString("password"),
		clientRecord.GetString("platform"),
	)
	if err != nil {
		return err
	}

	tokenRecord.Set("status", "valid")
	tokenRecord.Set("accessToken", newToken.AccessToken)
	tokenRecord.Set("refreshToken", newToken.RefreshToken)
	tokenRecord.Set("expires", CalculateTokenExpiry(newToken.ExpiresIn))

	if err := m.app.Save(tokenRecord); err != nil {
		return err
	}

	m.app.Logger().Info("fixed password grant token", "id", tokenRecord.Id)
	return nil
}

// fixDeviceCodeToken checks if the rate limit has reset and re-enables the token.
func (m *Manager) fixDeviceCodeToken(tokenRecord *core.Record) error {
	cutoff, err := GetRatelimitCutoff()
	if err != nil {
		return err
	}

	lastUsed := tokenRecord.GetDateTime("lastUsed")
	if time.Now().After(cutoff) && lastUsed.Time().Before(cutoff) {
		mu := m.getTokenMutex(tokenRecord.Id)
		mu.Lock()
		defer mu.Unlock()

		tokenRecord.Set("status", "valid")
		if err := m.app.Save(tokenRecord); err != nil {
			return err
		}
		m.app.Logger().Info("enabled token because of rate-limit reset", "id", tokenRecord.Id)
	}

	return nil
}

// refreshToken performs the actual token refresh with proper locking.
func (m *Manager) refreshToken(ctx context.Context, tokenRecord *core.Record) error {
	mu := m.getTokenMutex(tokenRecord.Id)
	mu.Lock()
	defer mu.Unlock()

	// Re-check after acquiring lock (another goroutine may have refreshed it)
	tokenRecord, err := m.app.FindRecordById("tokens", tokenRecord.Id)
	if err != nil {
		return err
	}

	expires := tokenRecord.GetDateTime("expires")
	bufferedExpiry := expires.Add(-60 * time.Second)
	if time.Now().Before(bufferedExpiry.Time()) && tokenRecord.GetString("status") == "valid" {
		return nil
	}

	clientRecord, err := m.app.FindRecordById("clients", tokenRecord.GetString("client"))
	if err != nil {
		return err
	}

	newToken, err := m.authProvider.RefreshToken(
		ctx,
		clientRecord.GetString("clientID"),
		tokenRecord.GetString("refreshToken"),
		clientRecord.GetString("platform"),
	)
	if err != nil {
		tokenRecord.Set("status", "invalid")
		m.app.Save(tokenRecord)
		return err
	}

	if newToken.AccessToken == "" || newToken.RefreshToken == "" {
		tokenRecord.Set("status", "invalid")
		m.app.Save(tokenRecord)
		return fmt.Errorf("empty access or refresh token received")
	}

	tokenRecord.Set("status", "valid")
	tokenRecord.Set("accessToken", newToken.AccessToken)
	tokenRecord.Set("refreshToken", newToken.RefreshToken)
	tokenRecord.Set("expires", CalculateTokenExpiry(newToken.ExpiresIn))

	if err := m.app.Save(tokenRecord); err != nil {
		return err
	}

	m.app.Logger().Info("refreshed token", "id", tokenRecord.Id)
	return nil
}

// GetValidToken retrieves a valid token, refreshing it first if necessary.
// This is the main method the proxy should use to get a token.
// It ensures the token is fresh and valid before returning.
func (m *Manager) GetValidToken(ctx context.Context, tokenRecord *core.Record) (*core.Record, error) {
	m.refreshToken(ctx, tokenRecord)

	// Re-fetch the token record to get updated values
	var err error
	tokenRecord, err = m.app.FindRecordById("tokens", tokenRecord.Id)
	if err != nil {
		return nil, err
	}

	if tokenRecord.GetString("status") != "valid" {
		err := m.tryFixToken(ctx, tokenRecord)
		if err != nil {
			return nil, err
		}

		// Re-fetch the token record to get updated values
		tokenRecord, err = m.app.FindRecordById("tokens", tokenRecord.Id)
		if err != nil {
			return nil, err
		}
	}

	return tokenRecord, nil
}

// MarkTokenInvalid marks a token as invalid (e.g., after a 401 response).
func (m *Manager) MarkTokenInvalid(tokenID string) error {
	mu := m.getTokenMutex(tokenID)
	mu.Lock()
	defer mu.Unlock()

	tokenRecord, err := m.app.FindRecordById("tokens", tokenID)
	if err != nil {
		return err
	}

	tokenRecord.Set("status", "invalid")
	tokenRecord.Set("used", time.Now())
	return m.app.Save(tokenRecord)
}

// UpdateTokenUsed updates the token's last used timestamp.
func (m *Manager) UpdateTokenUsed(tokenID string) error {
	tokenRecord, err := m.app.FindRecordById("tokens", tokenID)
	if err != nil {
		return err
	}

	tokenRecord.Set("used", time.Now())
	return m.app.Save(tokenRecord)
}

// GetRatelimitCutoff returns the cutoff time for rate limit calculations.
func GetRatelimitCutoff() (time.Time, error) {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		return time.Time{}, err
	}

	now := time.Now().In(loc)
	cutoff := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, loc)

	if now.Before(cutoff) {
		cutoff = cutoff.Add(-24 * time.Hour)
	}

	return cutoff.UTC(), nil
}

// CalculateTokenExpiry calculates the expiry time from expires_in seconds.
func CalculateTokenExpiry(expiresIn int) time.Time {
	return time.Now().Add(time.Duration(expiresIn) * time.Second)
}

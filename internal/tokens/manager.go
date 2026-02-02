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

	// Context for the background refresh loop
	ctx    context.Context
	cancel context.CancelFunc
}

// NewManager creates a new token manager.
func NewManager(app core.App, authProvider TokenAuthProvider) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		app:          app,
		authProvider: authProvider,
		tokenMutexes: make(map[string]*sync.Mutex),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start begins the background token refresh worker.
func (m *Manager) Start() {
	go m.refreshWorker()
}

// Stop stops the background token refresh worker.
func (m *Manager) Stop() {
	m.cancel()
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

// refreshWorker runs in the background and refreshes enabled tokens.
func (m *Manager) refreshWorker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := m.refreshEnabledTokens(); err != nil {
				m.app.Logger().Error("failed to refresh tokens", "error", err)
			}
		case <-m.ctx.Done():
			return
		}
	}
}

// refreshEnabledTokens refreshes all enabled tokens that need refreshing.
func (m *Manager) refreshEnabledTokens() error {
	// Only refresh tokens that are enabled (disabled = false)
	tokens, err := m.app.FindRecordsByFilter(
		"tokens",
		"disabled = false",
		"", 0, 0,
		nil,
	)
	if err != nil {
		return err
	}

	for _, tokenRecord := range tokens {
		// Try to fix invalid tokens first
		if tokenRecord.GetString("status") != "valid" {
			if err := m.tryFixToken(tokenRecord); err != nil {
				m.app.Logger().Error("failed to fix token", "id", tokenRecord.Id, "error", err)
			}
		}

		// Refresh the token if needed
		if err := m.refreshTokenIfNeeded(tokenRecord); err != nil {
			m.app.Logger().Error("failed to refresh token", "id", tokenRecord.Id, "error", err)
		}
	}

	return nil
}

// tryFixToken attempts to fix an invalid token.
func (m *Manager) tryFixToken(tokenRecord *core.Record) error {
	client, err := m.app.FindRecordById("clients", tokenRecord.GetString("client"))
	if err != nil {
		return err
	}

	if client.GetString("type") == "passwordGrant" {
		return m.fixPasswordGrantToken(tokenRecord, client)
	} else if client.GetString("type") == "deviceCode" {
		return m.fixDeviceCodeToken(tokenRecord)
	}

	return nil
}

// fixPasswordGrantToken re-authenticates using password grant.
func (m *Manager) fixPasswordGrantToken(tokenRecord *core.Record, clientRecord *core.Record) error {
	mu := m.getTokenMutex(tokenRecord.Id)
	mu.Lock()
	defer mu.Unlock()

	account, err := m.app.FindRecordById("accounts", tokenRecord.GetString("account"))
	if err != nil {
		return err
	}

	newToken, err := m.authProvider.Authorize(
		m.ctx,
		clientRecord.GetString("clientID"),
		clientRecord.GetString("redirectURI"),
		clientRecord.GetString("scope"),
		account.GetString("email"),
		account.GetString("password"),
		clientRecord.GetString("name"),
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

// refreshTokenIfNeeded refreshes a token if it's close to expiry.
func (m *Manager) refreshTokenIfNeeded(tokenRecord *core.Record) error {
	expires := tokenRecord.GetDateTime("expires")
	bufferedExpiry := expires.Add(-1 * time.Minute)

	// Only skip refresh if not expired AND valid
	if time.Now().Before(bufferedExpiry.Time()) && tokenRecord.GetString("status") == "valid" {
		return nil
	}

	return m.doRefreshToken(tokenRecord)
}

// doRefreshToken performs the actual token refresh with proper locking.
func (m *Manager) doRefreshToken(tokenRecord *core.Record) error {
	mu := m.getTokenMutex(tokenRecord.Id)
	mu.Lock()
	defer mu.Unlock()

	// Re-check after acquiring lock (another goroutine may have refreshed it)
	tokenRecord, err := m.app.FindRecordById("tokens", tokenRecord.Id)
	if err != nil {
		return err
	}

	expires := tokenRecord.GetDateTime("expires")
	bufferedExpiry := expires.Add(-1 * time.Minute)
	if time.Now().Before(bufferedExpiry.Time()) && tokenRecord.GetString("status") == "valid" {
		return nil
	}

	clientRecord, err := m.app.FindRecordById("clients", tokenRecord.GetString("client"))
	if err != nil {
		return err
	}

	newToken, err := m.authProvider.RefreshToken(
		m.ctx,
		clientRecord.GetString("clientID"),
		tokenRecord.GetString("refreshToken"),
		clientRecord.GetString("name"),
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
func (m *Manager) GetValidToken(tokenRecord *core.Record) (*core.Record, error) {
	// Check if token needs refresh
	expires := tokenRecord.GetDateTime("expires")
	bufferedExpiry := expires.Add(-1 * time.Minute)
	needsRefresh := time.Now().After(bufferedExpiry.Time()) || tokenRecord.GetString("status") != "valid"

	if needsRefresh {
		if err := m.doRefreshToken(tokenRecord); err != nil {
			clientRecord, err := m.app.FindRecordById("clients", tokenRecord.GetString("client"))
			if err != nil {
				return nil, err
			}

			if clientRecord.GetString("type") == "passwordGrant" {
				if err := m.fixPasswordGrantToken(tokenRecord, clientRecord); err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}

		// Re-fetch the token record to get updated values
		var err error
		tokenRecord, err = m.app.FindRecordById("tokens", tokenRecord.Id)
		if err != nil {
			return nil, err
		}
	}

	return tokenRecord, nil
}

// EnsureTokenValid ensures a token is valid, refreshing if needed.
// Returns true if the token is valid (or was successfully refreshed).
func (m *Manager) EnsureTokenValid(tokenID string) (bool, error) {
	tokenRecord, err := m.app.FindRecordById("tokens", tokenID)
	if err != nil {
		return false, err
	}

	_, err = m.GetValidToken(tokenRecord)
	if err != nil {
		return false, err
	}

	return true, nil
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

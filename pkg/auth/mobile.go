package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

// Mobile OAuth constants - from Tado mobile app
const (
	MobileClientID      = "eec8b609-9e2d-4403-9336-4f62a475271e"
	RedirectURI         = "tado://auth/redirect"
	Scope               = "home.user offline_access"
	AuthorizeURL        = "https://login.tado.com/oauth2/authorize"
	DeviceName          = "iPhone/iPod Safari"
	DeviceType          = "BROWSER"
	CodeChallengeMethod = "S256"
	MobileUserAgent     = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Mobile/15E148 Safari/604.1"
	MobileContentType   = "application/x-www-form-urlencoded"
	MobileAcceptHeader  = "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
)

type MobileAuthConfig struct {
	Email    string
	Password string
	Timezone string
}

// MobileAuth implements OAuth 2.0 with PKCE authentication flow.
// This replicates the Tado mobile app authentication process
type MobileAuth struct {
	config       *MobileAuthConfig
	httpClient   *http.Client
	state        string
	codeVerifier string
}

type TokenResponse struct {
	AccessToken    string `json:"access_token"`
	RefreshToken   string `json:"refresh_token"`
	RefreshTokenID string `json:"refresh_token_id"`
	ExpiresIn      int    `json:"expires_in"`
	TokenType      string `json:"token_type"`
	Scope          string `json:"scope"`
	UserID         string `json:"userId"`
}

func NewMobileAuth(config *MobileAuthConfig) *MobileAuth {
	jar, _ := cookiejar.New(nil)
	return &MobileAuth{
		config: config,
		httpClient: &http.Client{
			Jar: jar,
			// don't follow redirects automatically as we need to extract location headers
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (m *MobileAuth) GetToken(ctx context.Context) (*Token, error) {
	slog.Debug("starting mobile auth flow")
	verifier, challenge, err := GeneratePKCE()
	if err != nil {
		return nil, fmt.Errorf("failed to generate PKCE: %w", err)
	}
	m.codeVerifier = verifier

	state, err := GenerateState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}
	m.state = state

	slog.Debug("performing mobile authorization")
	authCode, err := m.authorize(ctx, m.config.Email, m.config.Password, challenge, state)
	if err != nil {
		return nil, fmt.Errorf("authorization failed: %w", err)
	}

	slog.Debug("exchanging code for token")
	tokenResp, err := m.exchangeCodeForToken(ctx, authCode, verifier)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	token := NewToken(
		tokenResp.AccessToken,
		tokenResp.RefreshToken,
		tokenResp.ExpiresIn,
	)

	return token, nil
}

// authorize performs the authorization flow and returns the authorization code
func (m *MobileAuth) authorize(ctx context.Context, username, password, challenge, state string) (string, error) {
	formData := m.buildAuthorizeForm(username, password, challenge, state)

	req, err := http.NewRequestWithContext(ctx, "POST", AuthorizeURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create authorize request: %w", err)
	}

	m.setCommonHeaders(req)
	req.Header.Set("Content-Type", MobileContentType)
	req.Header.Set("Origin", "https://login.tado.com")
	req.Header.Set("Referer", "https://login.tado.com/")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("authorize request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("authorize returned status %d: %s", resp.StatusCode, string(body))
	}

	// follow redirect to /oauth2/complete-registration
	location1 := resp.Header.Get("Location")
	if location1 == "" {
		return "", fmt.Errorf("no location header in authorize response")
	}

	location2, err := m.followRedirect(ctx, location1)
	if err != nil {
		return "", fmt.Errorf("failed to follow complete-registration redirect: %w", err)
	}

	// follow redirect to /oauth2/consent
	location3, err := m.followRedirect(ctx, location2)
	if err != nil {
		return "", fmt.Errorf("failed to follow consent redirect: %w", err)
	}

	// extract authorization code
	code, callbackState, err := m.extractAuthCode(location3)
	if err != nil {
		return "", fmt.Errorf("failed to extract auth code: %w", err)
	}

	// validate state to prevent CSRF
	if !ValidateState(m.state, callbackState) {
		return "", fmt.Errorf("state validation failed")
	}

	return code, nil
}

// followRedirect follows a redirect URL and returns the next location header
func (m *MobileAuth) followRedirect(ctx context.Context, location string) (string, error) {
	redirectURL := location
	if !strings.HasPrefix(location, "http") {
		redirectURL = "https://login.tado.com" + location
	}

	req, err := http.NewRequestWithContext(ctx, "GET", redirectURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create redirect request: %w", err)
	}

	m.setCommonHeaders(req)
	req.Header.Set("Referer", "https://login.tado.com/")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("redirect request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("redirect returned status %d: %s", resp.StatusCode, string(body))
	}

	nextLocation := resp.Header.Get("Location")
	if nextLocation == "" {
		return "", fmt.Errorf("no location header in redirect response")
	}

	return nextLocation, nil
}

// extractAuthCode extracts the authorization code from the redirect URI
func (m *MobileAuth) extractAuthCode(redirectURI string) (code, state string, err error) {
	// Parse the redirect URI: tado://auth/redirect?code=...&state=...
	u, err := url.Parse(redirectURI)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse redirect URI: %w", err)
	}

	code = u.Query().Get("code")
	if code == "" {
		return "", "", fmt.Errorf("no code in redirect URI")
	}

	state = u.Query().Get("state")
	return code, state, nil
}

// exchangeCodeForToken exchanges the authorization code for access and refresh tokens
func (m *MobileAuth) exchangeCodeForToken(ctx context.Context, code, verifier string) (*TokenResponse, error) {
	formData := url.Values{}
	formData.Set("scope", Scope)
	formData.Set("code", code)
	formData.Set("client_id", MobileClientID)
	formData.Set("grant_type", "authorization_code")
	formData.Set("code_verifier", verifier)
	formData.Set("redirect_uri", RedirectURI)

	req, err := http.NewRequestWithContext(ctx, "POST", TokenURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", MobileContentType)
	req.Header.Set("Accept", "*/*")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request returned status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokenResp, nil
}

// buildAuthorizeForm constructs the form data for the authorize request
func (m *MobileAuth) buildAuthorizeForm(username, password, challenge, state string) url.Values {
	formData := url.Values{}
	formData.Set("captcha_token", "")
	formData.Set("client_id", MobileClientID)
	formData.Set("code_challenge", challenge)
	formData.Set("code_challenge_method", CodeChallengeMethod)
	formData.Set("metaData.device.name", DeviceName)
	formData.Set("metaData.device.type", DeviceType)
	formData.Set("nonce", "")
	formData.Set("oauth_context", "")
	formData.Set("pendingIdPLinkId", "")
	formData.Set("redirect_uri", RedirectURI)
	formData.Set("response_mode", "")
	formData.Set("response_type", "code")
	formData.Set("scope", Scope)
	formData.Set("state", state)
	formData.Set("timezone", m.config.Timezone)
	formData.Set("user_code", "")
	formData.Set("userVerifyingPlatformAuthenticatorAvailable", "true")
	formData.Set("loginId", username)
	formData.Set("password", password)

	return formData
}

// setCommonHeaders sets headers common to all requests
func (m *MobileAuth) setCommonHeaders(req *http.Request) {
	req.Header.Set("User-Agent", MobileUserAgent)
	req.Header.Set("Accept", MobileAcceptHeader)
}

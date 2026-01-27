package tado

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	TokenURL      = "https://login.tado.com/oauth2/token"
	AuthorizeURL  = "https://login.tado.com/oauth2/authorize"
	DeviceAuthURL = "https://login.tado.com/oauth2/device_authorize"
	TenantID      = "1d543ad5-a8ac-4704-b9e2-26838b4d6513"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	UserID       string `json:"userId"`
}

func (c *Client) Authorize(
	ctx context.Context,
	clientID, redirectURI, scope,
	email, password string,
	platform string,
) (*TokenResponse, error) {
	verifier, err := GenerateCodeVerifier()
	if err != nil {
		return nil, err
	}
	challenge := GenerateCodeChallenge(verifier)

	state, err := GenerateState()
	if err != nil {
		return nil, err
	}

	// Create auth client with appropriate fingerprint
	authClient := NewAuthClient(platform)

	// Build initial authorize URL
	initURL, err := url.Parse(AuthorizeURL)
	if err != nil {
		return nil, err
	}

	q := initURL.Query()
	q.Set("client_id", clientID)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("redirect_uri", redirectURI)
	q.Set("response_type", "code")
	q.Set("scope", scope)
	q.Set("state", state)
	initURL.RawQuery = q.Encode()

	// Step 1: GET the authorize page to get cookies
	resp, err := authClient.R().
		SetContext(ctx).
		SetHeader("sec-fetch-site", "none").
		Get(initURL.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get authorize page: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get authorize page (%d)", resp.StatusCode)
	}

	// Step 2: POST the login form
	// Build form data in exact order matching real traffic
	formData := url.Values{}
	formData.Set("captcha_token", "")
	formData.Set("client_id", clientID)
	formData.Set("code_challenge", challenge)
	formData.Set("code_challenge_method", "S256")
	if platform == "web" {
		formData.Set("metaData.device.name", "Linux Firefox")
	} else {
		formData.Set("metaData.device.name", "iPhone/iPod Safari")
	}
	formData.Set("metaData.device.type", "BROWSER")
	formData.Set("nonce", "")
	formData.Set("oauth_context", "")
	formData.Set("pendingIdPLinkId", "")
	formData.Set("redirect_uri", redirectURI)
	formData.Set("response_mode", "")
	formData.Set("response_type", "code")
	formData.Set("scope", scope)
	formData.Set("state", state)
	formData.Set("tenantId", TenantID)
	formData.Set("timezone", "Europe/Berlin")
	formData.Set("user_code", "")
	formData.Set("userVerifyingPlatformAuthenticatorAvailable", "true")
	formData.Set("loginId", email)
	formData.Set("password", password)

	resp, err = authClient.R().
		SetContext(ctx).
		SetHeader("content-type", "application/x-www-form-urlencoded").
		SetHeader("origin", "https://login.tado.com").
		SetHeader("referer", "https://login.tado.com/").
		SetBodyString(formData.Encode()).
		Post(AuthorizeURL)
	if err != nil {
		return nil, fmt.Errorf("authorization failed: %w", err)
	}

	if resp.StatusCode != http.StatusFound {
		return nil, fmt.Errorf("authorization failed (%d): %s", resp.StatusCode, resp.String())
	}

	// Step 3: Follow redirects until we reach the redirect URI
	location := resp.GetHeader("Location")
	for location != "" && !strings.HasPrefix(location, redirectURI) {
		absLocation := c.absURL(location)

		resp, err = authClient.R().
			SetContext(ctx).
			SetHeader("referer", "https://login.tado.com/").
			Get(absLocation)
		if err != nil {
			return nil, fmt.Errorf("redirect failed: %w", err)
		}

		if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("redirect failed (%d): %s", resp.StatusCode, resp.String())
		}

		location = resp.GetHeader("Location")
	}

	if location == "" {
		return nil, fmt.Errorf("failed to reach redirect URI")
	}

	// Extract code from location
	locationURL, err := url.Parse(location)
	if err != nil {
		return nil, err
	}
	code := locationURL.Query().Get("code")
	if code == "" {
		return nil, fmt.Errorf("no code in redirect URI")
	}

	// Step 4: Exchange code for token using the token client
	tokenClient := NewTokenClient(platform)

	// Build token form data matching real traffic order
	tokenData := url.Values{}
	tokenData.Set("code", code)
	tokenData.Set("code_verifier", verifier)
	tokenData.Set("redirect_uri", redirectURI)
	tokenData.Set("scope", scope)
	tokenData.Set("grant_type", "authorization_code")
	tokenData.Set("client_id", clientID)

	var tokenResp TokenResponse
	resp, err = tokenClient.R().
		SetContext(ctx).
		SetHeader("content-type", "application/x-www-form-urlencoded").
		SetBodyString(tokenData.Encode()).
		SetSuccessResult(&tokenResp).
		Post(TokenURL)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed (%d): %s", resp.StatusCode, resp.String())
	}

	return &tokenResp, nil
}

type DeviceAuthResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

func (c *Client) DeviceAuthorize(ctx context.Context, clientID, scope string) (*DeviceAuthResponse, error) {
	tokenClient := NewIOSSafariTokenClient()

	authURL, err := url.Parse(DeviceAuthURL)
	if err != nil {
		return nil, err
	}

	q := authURL.Query()
	q.Set("client_id", clientID)
	q.Set("scope", scope)
	authURL.RawQuery = q.Encode()

	var deviceResp DeviceAuthResponse
	resp, err := tokenClient.R().
		SetContext(ctx).
		SetSuccessResult(&deviceResp).
		Post(authURL.String())
	if err != nil {
		return nil, fmt.Errorf("device authorize failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device authorize failed (%d): %s", resp.StatusCode, resp.String())
	}

	return &deviceResp, nil
}

func (c *Client) ExchangeDeviceCode(ctx context.Context, clientID, deviceCode string) (*TokenResponse, error) {
	tokenClient := NewIOSSafariTokenClient()

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("device_code", deviceCode)
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	var tokenResp TokenResponse
	resp, err := tokenClient.R().
		SetContext(ctx).
		SetHeader("content-type", "application/x-www-form-urlencoded").
		SetBodyString(data.Encode()).
		SetSuccessResult(&tokenResp).
		Post(TokenURL)
	if err != nil {
		return nil, fmt.Errorf("device token request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device token request failed (%d): %s", resp.StatusCode, resp.String())
	}

	return &tokenResp, nil
}

func (c *Client) RefreshToken(ctx context.Context, clientID, refreshToken, platform string) (*TokenResponse, error) {
	tokenClient := NewTokenClient(platform)

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	var tokenResp TokenResponse
	resp, err := tokenClient.R().
		SetContext(ctx).
		SetHeader("content-type", "application/x-www-form-urlencoded").
		SetBodyString(data.Encode()).
		SetSuccessResult(&tokenResp).
		Post(TokenURL)
	if err != nil {
		return nil, fmt.Errorf("refresh token failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{StatusCode: resp.StatusCode, Body: resp.String()}
	}

	return &tokenResp, nil
}

func CalculateTokenExpiry(expiresIn int) time.Time {
	return time.Now().Add(time.Duration(expiresIn) * time.Second)
}

func GenerateCodeVerifier() (string, error) {
	bytes := make([]byte, 96)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func GenerateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func GenerateState() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

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

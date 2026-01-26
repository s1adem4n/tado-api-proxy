package tado

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	TokenURL      = "https://login.tado.com/oauth2/token"
	AuthorizeURL  = "https://login.tado.com/oauth2/authorize"
	DeviceAuthURL = "https://login.tado.com/oauth2/device_authorize"
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

	httpClient, err := c.createHTTPClient()
	if err != nil {
		return nil, err
	}

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

	// Get the cookies
	req, err := http.NewRequestWithContext(ctx, "GET", initURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	// POST /oauth2/authorize
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("code_challenge", challenge)
	data.Set("code_challenge_method", "S256")
	data.Set("redirect_uri", redirectURI)
	data.Set("response_type", "code")
	data.Set("scope", scope)
	data.Set("state", state)
	data.Set("loginId", email)
	data.Set("password", password)

	req, err = http.NewRequestWithContext(ctx, "POST", AuthorizeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", UserAgent)

	resp, err = httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("authorize failed (%d): %s", resp.StatusCode, string(body))
	}

	location := resp.Header.Get("Location")
	for location != "" && !strings.HasPrefix(location, redirectURI) {
		req, err = http.NewRequestWithContext(ctx, "GET", c.absURL(location), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", UserAgent)

		resp, err = httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusFound {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("redirect failed (%d): %s", resp.StatusCode, string(body))
		}

		location = resp.Header.Get("Location")
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

	// Extract code from location
	locationURL, err = url.Parse(location)
	if err != nil {
		return nil, err
	}
	code = locationURL.Query().Get("code")
	if code == "" {
		return nil, fmt.Errorf("no code in redirect URI")
	}

	// Exchange code for token
	data = url.Values{}
	data.Set("client_id", clientID)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("code_verifier", verifier)
	data.Set("redirect_uri", redirectURI)

	req, err = http.NewRequestWithContext(ctx, "POST", TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", UserAgent)

	resp, err = httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed (%d): %s", resp.StatusCode, string(body))
	}

	var tr TokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, err
	}

	return &tr, nil
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
	httpClient, err := c.createHTTPClient()
	if err != nil {
		return nil, err
	}

	authURL, err := url.Parse(DeviceAuthURL)
	if err != nil {
		return nil, err
	}

	q := authURL.Query()
	q.Set("client_id", clientID)
	q.Set("scope", scope)
	authURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "POST", authURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device authorize failed (%d): %s", resp.StatusCode, string(body))
	}

	var dar DeviceAuthResponse
	if err := json.Unmarshal(body, &dar); err != nil {
		return nil, err
	}

	return &dar, nil
}

func (c *Client) ExchangeDeviceCode(ctx context.Context, clientID, deviceCode string) (*TokenResponse, error) {
	httpClient, err := c.createHTTPClient()
	if err != nil {
		return nil, err
	}

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("device_code", deviceCode)
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	req, err := http.NewRequestWithContext(ctx, "POST", TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", UserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		// Device flow returns error while pending
		return nil, fmt.Errorf("device token request failed (%d): %s", resp.StatusCode, string(body))
	}

	var tr TokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, err
	}

	return &tr, nil
}

func (c *Client) RefreshToken(ctx context.Context, clientID, refreshToken string) (*TokenResponse, error) {
	httpClient, err := c.createHTTPClient()
	if err != nil {
		return nil, err
	}

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequestWithContext(ctx, "POST", TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", UserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	var tr TokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, err
	}

	return &tr, nil
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

func GetRatelimtCutoff() (time.Time, error) {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		return time.Time{}, err
	}

	now := time.Now().In(loc)
	cutoff := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, loc)

	return cutoff, nil
}

package tado

import (
	"context"

	"github.com/s1adem4n/tado-api-proxy/internal/tokens"
)

// TokenManager returns the token manager for use by other components.
func (c *Client) TokenManager() *tokens.Manager {
	return c.tokenManager
}

// TokenAuthProvider implementation - adapts tado.Client to the tokens.TokenAuthProvider interface

// RefreshTokenForManager implements tokens.TokenAuthProvider.
func (c *Client) RefreshToken(ctx context.Context, clientID, refreshToken, platform string) (*tokens.TokenResult, error) {
	resp, err := c.refreshToken(ctx, clientID, refreshToken, platform)
	if err != nil {
		return nil, err
	}
	return &tokens.TokenResult{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		ExpiresIn:    resp.ExpiresIn,
	}, nil
}

// AuthorizeForManager implements tokens.TokenAuthProvider.
func (c *Client) Authorize(ctx context.Context, clientID, redirectURI, scope, email, password, platform string) (*tokens.TokenResult, error) {
	resp, err := c.authorize(ctx, clientID, redirectURI, scope, email, password, platform)
	if err != nil {
		return nil, err
	}
	return &tokens.TokenResult{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		ExpiresIn:    resp.ExpiresIn,
	}, nil
}

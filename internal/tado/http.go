package tado

import (
	"github.com/imroc/req/v3"
)

func NewAuthClient(platform string) *req.Client {
	if platform == "mobile" {
		return NewIOSSafariAuthClient()
	}
	return NewFirefoxAuthClient()
}

// NewTokenClient creates a token client for the given client type
func NewTokenClient(platform string) *req.Client {
	if platform == "mobile" {
		return NewIOSSafariTokenClient()
	}
	return NewFirefoxTokenClient()
}

// NewAPIClient creates an API client for the given client type
func NewAPIClient(platform string) *req.Client {
	if platform == "mobile" {
		return NewIOSSafariAPIClient()
	}
	return NewFirefoxAPIClient()
}

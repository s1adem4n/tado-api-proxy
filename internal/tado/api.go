package tado

import (
	"context"
	"fmt"
	"net/http"

	"github.com/imroc/req/v3"
)

const (
	BaseURL = "https://my.tado.com"
)

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api error (%d): %s", e.StatusCode, e.Body)
}

type MeResponse struct {
	ID    string `json:"id"`
	Homes []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
}

func (c *Client) GetMe(ctx context.Context, accessToken, platform string) (*MeResponse, error) {
	var apiClient *req.Client
	if platform == "mobile" {
		apiClient = NewIOSSafariAPIClient()
	} else {
		apiClient = NewFirefoxAPIClient()
	}

	var meResp MeResponse
	resp, err := apiClient.R().
		SetContext(ctx).
		SetHeader("authorization", "Bearer "+accessToken).
		SetQueryParam("ngsw-bypass", "true").
		SetSuccessResult(&meResp).
		Get(BaseURL + "/api/v2/me")
	if err != nil {
		return nil, fmt.Errorf("failed to get /me: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{StatusCode: resp.StatusCode, Body: resp.String()}
	}

	return &meResp, nil
}

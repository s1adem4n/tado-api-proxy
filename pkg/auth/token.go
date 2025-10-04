package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	TokenURL = "https://login.tado.com/oauth2/token"
)

type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int       `json:"expires_in"`
	Expiry       time.Time `json:"expiry"` // Custom field, not included in token response
}

func NewToken(
	accessToken,
	refreshToken string,
	expiresIn int,
) *Token {
	return &Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		Expiry:       time.Now().Add(time.Duration(expiresIn) * time.Second),
	}
}

func (t *Token) Refresh(ctx context.Context) error {
	values := url.Values{}
	values.Add("client_id", ClientID)
	values.Add("grant_type", "refresh_token")
	values.Add("scope", "home.user")
	values.Add("refresh_token", t.RefreshToken)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		TokenURL+"?"+values.Encode(),
		nil,
	)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to refresh token: %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(t); err != nil {
		return err
	}
	t.Expiry = time.Now().Add(time.Duration(t.ExpiresIn) * time.Second)

	return nil
}

func (t *Token) Test(ctx context.Context) error {
	if !t.Valid() {
		return fmt.Errorf("token is not valid")
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		BaseURL+"/me",
		nil,
	)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+t.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token test failed: %s", resp.Status)
	}

	return nil
}

func (t *Token) Valid() bool {
	return t.AccessToken != "" &&
		t.RefreshToken != "" &&
		time.Now().Before(t.Expiry)
}

func (t *Token) Load(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(t)
}

func (t *Token) Save(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(t)
}

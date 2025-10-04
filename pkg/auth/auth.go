package auth

import (
	"context"
	"io"
	"log"
	"maps"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	BaseURL  = "https://my.tado.com"
	ClientID = "af44f89e-ae86-4ebe-905f-6bf759cf6473"
)

type Handler struct {
	browserAuth *BrowserAuth
	tokenPath   string
	token       *Token
	lock        sync.RWMutex
}

func NewHandler(browserAuth *BrowserAuth, tokenPath string) *Handler {
	return &Handler{
		browserAuth: browserAuth,
		tokenPath:   tokenPath,
		token:       &Token{},
		lock:        sync.RWMutex{},
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := h.GetToken()
	if token == nil {
		http.Error(w, "No token available", http.StatusInternalServerError)
		return
	}

	if time.Now().After(token.Expiry) {
		log.Print("Token expired, refreshing")

		err := token.Refresh(r.Context())
		if err != nil {
			log.Print("Failed to refresh token, using browser to get a new one: " + err.Error())
			token, err = h.browserAuth.GetToken(r.Context())
			if err != nil {
				http.Error(w, "Failed to get token from browser: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}

		err = h.SetToken(token)
		if err != nil {
			http.Error(w, "Failed to save refreshed token: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	resp, err := h.proxyRequest(r)
	if err != nil {
		http.Error(w, "Failed to forward request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Sometimes the token is refreshed, but it is still considered invalid by the API.
	// When this happens, we get a fresh token from the browser and retry once.
	if resp.StatusCode == http.StatusUnauthorized {
		log.Print("Request returned unauthorized, getting fresh token from browser")

		token, err = h.browserAuth.GetToken(r.Context())
		if err != nil {
			http.Error(w, "Failed to get token from browser: "+err.Error(), http.StatusInternalServerError)
			return
		}

		err = h.SetToken(token)
		if err != nil {
			http.Error(w, "Failed to save browser token: "+err.Error(), http.StatusInternalServerError)
			return
		}

		resp.Body.Close()

		resp, err = h.proxyRequest(r)
		if err != nil {
			http.Error(w, "Failed to retry request: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()
	}

	maps.Copy(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Print("Failed to read response body: " + err.Error())
	}
}

func (h *Handler) proxyRequest(r *http.Request) (*http.Response, error) {
	req, err := http.NewRequestWithContext(
		r.Context(),
		r.Method,
		BaseURL+r.URL.Path+"?"+r.URL.RawQuery,
		r.Body,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+h.token.AccessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	return http.DefaultClient.Do(req)
}

func (h *Handler) Init(ctx context.Context) error {
	err := h.token.Load(h.tokenPath)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	err = h.token.Test(ctx)
	if err != nil {
		log.Print("Loaded token is invalid, using browser to get a new one")

		h.token, err = h.browserAuth.GetToken(ctx)
		if err != nil {
			return err
		}
	}

	return h.token.Save(h.tokenPath)
}

func (h *Handler) GetToken() *Token {
	h.lock.RLock()
	defer h.lock.RUnlock()

	return h.token
}

func (h *Handler) SetToken(token *Token) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.token = token
	return h.token.Save(h.tokenPath)
}

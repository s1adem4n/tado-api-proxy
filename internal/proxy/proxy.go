package proxy

import (
	"io"
	"maps"
	"net/http"

	"github.com/s1adem4n/tado-api-proxy/pkg/auth"
)

const (
	BaseURL = "https://my.tado.com"
)

type Handler struct {
	authHandler *auth.Handler
}

func NewHandler(authHandler *auth.Handler) *Handler {
	return &Handler{
		authHandler: authHandler,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token, err := h.authHandler.GetAccessToken()
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	req, err := http.NewRequestWithContext(
		r.Context(),
		r.Method,
		BaseURL+r.URL.Path+"?"+r.URL.RawQuery,
		r.Body,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header = r.Header.Clone()
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	maps.Copy(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

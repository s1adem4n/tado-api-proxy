package proxy

import (
	"io"
	"maps"
	"net/http"

	"github.com/s1adem4n/tado-api-proxy/internal/stats"
	"github.com/s1adem4n/tado-api-proxy/pkg/auth"
)

const (
	BaseURL = "https://my.tado.com"
)

type Handler struct {
	authHandler  *auth.Handler
	statsTracker *stats.Tracker
}

func NewHandler(authHandler *auth.Handler, statsTracker *stats.Tracker) *Handler {
	return &Handler{
		authHandler:  authHandler,
		statsTracker: statsTracker,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.statsTracker.Record()

	token, err := h.authHandler.GetAccessToken()
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	url := BaseURL + r.URL.Path
	if r.URL.RawQuery != "" {
		url += "?" + r.URL.RawQuery
	}

	req, err := http.NewRequestWithContext(
		r.Context(),
		r.Method,
		url,
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

package proxy

import (
	"io"
	"log/slog"
	"maps"
	"net/http"
	"strings"
	"time"

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

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func NewHandler(authHandler *auth.Handler, statsTracker *stats.Tracker) *Handler {
	return &Handler{
		authHandler:  authHandler,
		statsTracker: statsTracker,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if strings.HasPrefix(r.URL.Path, "/api") {
		h.statsTracker.Record()
	}

	lrw := &loggingResponseWriter{w, http.StatusOK}
	defer func() {
		slog.Info("request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", lrw.statusCode,
			"duration", time.Since(start),
		)
	}()

	token, err := h.authHandler.GetAccessToken()
	if err != nil {
		slog.Error("failed to get access token", "error", err)
		http.Error(lrw, err.Error(), http.StatusUnauthorized)
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
		slog.Error("failed to create proxy request", "error", err)
		http.Error(lrw, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header = r.Header.Clone()
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("upstream request failed", "error", err)
		http.Error(lrw, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	maps.Copy(lrw.Header(), resp.Header)
	lrw.WriteHeader(resp.StatusCode)
	_, err = io.Copy(lrw, resp.Body)
	if err != nil {
		slog.Error("failed to copy upstream response", "error", err)
		return
	}
}

package proxy

import (
	"bytes"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	UserAgent = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Mobile/15E148 Safari/604.1"
)

type Handler struct {
	app core.App
}

func NewHandler(app core.App) *Handler {
	return &Handler{
		app: app,
	}
}

func (h *Handler) Register() {
	h.app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.Any("/api/v2/{path...}", h.HandleRequest)

		return e.Next()
	})

	h.app.Cron().MustAdd("clean-request-logs", "0 * * * *", func() {
		h.app.Logger().Info("cleaning request logs")
		err := h.CleanRequestLogs()
		if err != nil {
			h.app.Logger().Error("failed to clean request logs", "error", err)
		}
	})
}

func (h *Handler) HandleRequest(e *core.RequestEvent) error {
	filter := "status = 'valid'"

	accountEmail := e.Request.Header.Get("X-Tado-Email")
	if accountEmail != "" {
		filter += " && account.email = {:email}"
	}

	homeID := extractHomeID(e.Request.URL.Path)
	if homeID != "" {
		filter += " && account.homes.tadoID ?= {:homeID}"
	}

	tokens, err := h.app.FindRecordsByFilter(
		"tokens",
		filter, "used", 0, 0,
		dbx.Params{
			"email":  accountEmail,
			"homeID": homeID,
		},
	)
	if err != nil {
		return err
	}

	if len(tokens) == 0 {
		return e.BadRequestError("no valid tokens found", nil)
	}

	bodyBytes, err := io.ReadAll(e.Request.Body)
	if err != nil {
		return err
	}
	defer e.Request.Body.Close()

	url := url.URL{
		Scheme:   "https",
		Host:     "my.tado.com",
		Path:     e.Request.URL.Path,
		RawQuery: e.Request.URL.RawQuery,
	}

	for _, token := range tokens {
		req, err := http.NewRequestWithContext(
			e.Request.Context(),
			e.Request.Method,
			url.String(),
			io.NopCloser(bytes.NewReader(bodyBytes)),
		)
		if err != nil {
			return err
		}

		req.Header = e.Request.Header.Clone()
		req.Header.Del("X-Tado-Email")
		req.Header.Set("Authorization", "Bearer "+token.GetString("accessToken"))
		req.Header.Set("User-Agent", UserAgent)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}

		if resp.StatusCode == http.StatusUnauthorized {
			token.Set("used", time.Now())
			token.Set("status", "invalid")
			if err := h.app.Save(token); err != nil {
				resp.Body.Close()
				return err
			}
			resp.Body.Close()
			continue
		}

		token.Set("used", time.Now())
		if err := h.app.Save(token); err != nil {
			resp.Body.Close()
			return err
		}

		maps.Copy(e.Response.Header(), resp.Header)
		e.Response.WriteHeader(resp.StatusCode)
		_, err = io.Copy(e.Response, resp.Body)
		resp.Body.Close()
		if err != nil {
			return err
		}

		requestsCollection, err := h.app.FindCollectionByNameOrId("requests")
		if err != nil {
			return err
		}

		requestRecord := core.NewRecord(requestsCollection)

		requestRecord.Set("token", token.Id)
		requestRecord.Set("method", e.Request.Method)
		requestRecord.Set("url", url.String())
		requestRecord.Set("status", resp.StatusCode)
		if err := h.app.Save(requestRecord); err != nil {
			slog.Error("failed to log request", "error", err)
		}

		return nil
	}

	return e.UnauthorizedError("no valid tokens found", nil)
}

func (h *Handler) CleanRequestLogs() error {
	threshold := time.Now().Add(-7 * 24 * time.Hour)

	records, err := h.app.FindRecordsByFilter(
		"requests",
		"created < {:threshold}",
		"", 0, 0,
		dbx.Params{
			"threshold": threshold,
		},
	)
	if err != nil {
		return err
	}

	err = h.app.RunInTransaction(func(txApp core.App) error {
		for _, r := range records {
			if err := txApp.Delete(r); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func extractHomeID(path string) string {
	re := regexp.MustCompile(`^/api/v2/homes/(\d+)`)
	matches := re.FindStringSubmatch(path)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}

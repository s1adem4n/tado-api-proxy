package proxy

import (
	"bytes"
	"fmt"
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
		e.Router.Any("/api/v2/{path...}", h.HandleProxyRequest)
		e.Router.GET("/api/ratelimits", h.HandleRatelimitsRequest)

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

func (h *Handler) HandleProxyRequest(e *core.RequestEvent) error {
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

	var validTokens []*core.Record
	var totalUsed int
	var totalLimit int

	// cutoff at 00:00 in CET
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		return err
	}
	now := time.Now().In(loc)
	cutoff := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	for _, token := range tokens {
		client, err := h.app.FindRecordById("clients", token.GetString("client"))
		if err != nil {
			return err
		}

		totalLimit += client.GetInt("dailyLimit")

		var count int
		err = h.app.DB().NewQuery(
			"SELECT count(*) FROM requests WHERE token = {:tokenID} AND created > {:cutoff}",
		).Bind(dbx.Params{
			"tokenID": token.Id,
			"cutoff":  cutoff,
		}).Row(&count)
		if err != nil {
			return err
		}

		totalUsed += count

		if count < client.GetInt("dailyLimit") {
			validTokens = append(validTokens, token)
		}
	}

	// Read body for reuse
	bodyBytes, err := io.ReadAll(e.Request.Body)
	if err != nil {
		return err
	}
	e.Request.Body.Close()

	url := url.URL{
		Scheme:   "https",
		Host:     "my.tado.com",
		Path:     e.Request.URL.Path,
		RawQuery: e.Request.URL.RawQuery,
	}

	for _, token := range validTokens {
		req, err := http.NewRequestWithContext(
			e.Request.Context(),
			e.Request.Method,
			url.String(),
			bytes.NewReader(bodyBytes),
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

		e.Response.Header().Set(
			"Ratelimit",
			fmt.Sprintf(`"perday";r=%d`, totalLimit-totalUsed-1),
		)
		e.Response.Header().Set(
			"Ratelimit-Policy",
			fmt.Sprintf(`"perday";q=%d;w=86400`, totalLimit),
		)

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

func (h *Handler) HandleRatelimitsRequest(e *core.RequestEvent) error {
	tokens, err := h.app.FindRecordsByFilter(
		"tokens",
		"", "used", 0, 0,
		nil,
	)
	if err != nil {
		return err
	}

	if len(tokens) == 0 {
		return e.JSON(200, nil)
	}

	usage := map[string]any{}
	cutoff, err := getRatelimtCutoff()
	if err != nil {
		return err
	}

	for _, token := range tokens {
		client, err := h.app.FindRecordById("clients", token.GetString("client"))
		if err != nil {
			return err
		}

		var count int
		err = h.app.DB().NewQuery(
			"SELECT count(*) FROM requests WHERE token = {:tokenID} AND created > {:cutoff}",
		).Bind(dbx.Params{
			"tokenID": token.Id,
			"cutoff":  cutoff,
		}).Row(&count)
		if err != nil {
			return err
		}

		usage[token.Id] = map[string]any{
			"used":      count,
			"limit":     client.GetInt("dailyLimit"),
			"remaining": client.GetInt("dailyLimit") - count,
			"status":    token.GetString("status"),
		}
	}

	return e.JSON(200, usage)
}

func (h *Handler) CleanRequestLogs() error {
	cutoff := time.Now().Add(-7 * 24 * time.Hour)

	records, err := h.app.FindRecordsByFilter(
		"requests",
		"created < {:cutoff}",
		"", 0, 0,
		dbx.Params{
			"cutoff": cutoff,
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

func getRatelimtCutoff() (time.Time, error) {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		return time.Time{}, err
	}

	now := time.Now().In(loc)
	cutoff := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	return cutoff, nil
}

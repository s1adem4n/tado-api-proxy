package proxy

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/imroc/req/v3"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/s1adem4n/tado-api-proxy/internal/tado"
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
		e.Router.GET("/api/stats", h.HandleStatsRequest)

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

	// Prefer deviceCode tokens as they don't get users banned
	var preferredTokens []struct {
		client *core.Record
		token  *core.Record
	}
	var otherTokens []struct {
		client *core.Record
		token  *core.Record
	}
	var totalUsed int
	var totalLimit int

	cutoff, err := tado.GetRatelimitCutoff()
	if err != nil {
		return err
	}

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
			if client.GetString("type") == "deviceCode" {
				preferredTokens = append(preferredTokens, struct {
					client *core.Record
					token  *core.Record
				}{
					client: client,
					token:  token,
				})
			} else {
				otherTokens = append(otherTokens, struct {
					client *core.Record
					token  *core.Record
				}{
					client: client,
					token:  token,
				})
			}
		}
	}

	// Read body for reuse
	bodyBytes, err := io.ReadAll(e.Request.Body)
	if err != nil {
		return err
	}
	e.Request.Body.Close()

	targetURL := url.URL{
		Scheme:   "https",
		Host:     "my.tado.com",
		Path:     e.Request.URL.Path,
		RawQuery: e.Request.URL.RawQuery,
	}

	validTokens := append(preferredTokens, otherTokens...)

	for _, t := range validTokens {
		var apiClient *req.Client
		if t.client.GetString("platform") == "mobile" {
			apiClient = tado.NewIOSSafariAPIClient()
		} else {
			apiClient = tado.NewFirefoxAPIClient()
		}

		// Create request with iOS Safari fingerprinting
		request := apiClient.R().
			SetContext(e.Request.Context()).
			SetHeader("authorization", "Bearer "+t.token.GetString("accessToken"))

		// Add query param to match real traffic
		if targetURL.RawQuery != "" {
			targetURL.RawQuery += "&ngsw-bypass=true"
		} else {
			targetURL.RawQuery = "ngsw-bypass=true"
		}

		// Set content type if present in original request
		if ct := e.Request.Header.Get("Content-Type"); ct != "" {
			request.SetHeader("content-type", ct)
		}

		// Set body if present
		if len(bodyBytes) > 0 {
			request.SetBody(bytes.NewReader(bodyBytes))
		}

		var resp *req.Response

		switch e.Request.Method {
		case http.MethodGet:
			resp, err = request.Get(targetURL.String())
		case http.MethodPost:
			resp, err = request.Post(targetURL.String())
		case http.MethodPut:
			resp, err = request.Put(targetURL.String())
		case http.MethodDelete:
			resp, err = request.Delete(targetURL.String())
		case http.MethodPatch:
			resp, err = request.Patch(targetURL.String())
		default:
			return e.BadRequestError("unsupported method", nil)
		}

		if err != nil {
			slog.Error("proxy request failed", "error", err)
			continue
		}

		if resp.StatusCode == http.StatusUnauthorized {
			t.token.Set("used", time.Now())
			t.token.Set("status", "invalid")
			if err := h.app.Save(t.token); err != nil {
				return err
			}
			continue
		}

		t.token.Set("used", time.Now())
		if err := h.app.Save(t.token); err != nil {
			return err
		}

		// Copy response headers
		for k, v := range resp.Header {
			for _, vv := range v {
				e.Response.Header().Add(k, vv)
			}
		}

		e.Response.Header().Set(
			"Ratelimit",
			fmt.Sprintf(`"perday";r=%d`, totalLimit-totalUsed-1),
		)
		e.Response.Header().Set(
			"Ratelimit-Policy",
			fmt.Sprintf(`"perday";q=%d;w=86400`, totalLimit),
		)

		e.Response.WriteHeader(resp.StatusCode)

		_, err = e.Response.Write(resp.Bytes())
		if err != nil {
			return err
		}

		requestsCollection, err := h.app.FindCollectionByNameOrId("requests")
		if err != nil {
			return err
		}

		requestRecord := core.NewRecord(requestsCollection)

		requestRecord.Set("token", t.token.Id)
		requestRecord.Set("method", e.Request.Method)
		requestRecord.Set("url", targetURL.String())
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
	cutoff, err := tado.GetRatelimitCutoff()
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

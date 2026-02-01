package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
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

// tokenWithClient pairs a token record with its associated client record.
type tokenWithClient struct {
	client *core.Record
	token  *core.Record
}

// tokenSelection contains categorized tokens and usage stats.
type tokenSelection struct {
	preferred  []tokenWithClient
	other      []tokenWithClient
	totalUsed  int
	totalLimit int
}

// proxyResult contains the result of a proxy request attempt.
type proxyResult struct {
	response *req.Response
	token    tokenWithClient
}

func (h *Handler) HandleProxyRequest(e *core.RequestEvent) error {
	tokens, err := h.findTokens(e)
	if err != nil {
		return err
	}
	if len(tokens) == 0 {
		return e.BadRequestError("no valid tokens found", nil)
	}

	selection, err := h.categorizeTokens(tokens)
	if err != nil {
		return err
	}

	bodyBytes, err := io.ReadAll(e.Request.Body)
	if err != nil {
		return err
	}
	e.Request.Body.Close()

	targetURL := h.buildTargetURL(e.Request.URL)

	validTokens := append(selection.preferred, selection.other...)
	for _, t := range validTokens {
		result, err := h.tryProxyRequest(e, t, targetURL, bodyBytes)
		if err != nil {
			continue
		}
		if result == nil {
			continue
		}

		h.updateClientRateLimit(t.client, result.response.Header.Get("ratelimit-policy"))

		h.writeProxyResponse(e, result.response, selection.totalLimit, selection.totalUsed)
		h.logRequest(t.token.Id, e.Request.Method, targetURL.String(), result.response.StatusCode)

		return nil
	}

	return e.UnauthorizedError("no valid tokens found", nil)
}

// findTokens retrieves valid tokens based on request headers and path.
func (h *Handler) findTokens(e *core.RequestEvent) ([]*core.Record, error) {
	filter := "status = 'valid' && disabled = false"

	accountEmail := e.Request.Header.Get("X-Tado-Email")
	if accountEmail != "" {
		filter += " && account.email = {:email}"
	}

	homeID := extractHomeID(e.Request.URL.Path)
	if homeID != "" {
		filter += " && account.homes.tadoID ?= {:homeID}"
	}

	return h.app.FindRecordsByFilter(
		"tokens",
		filter, "used", 0, 0,
		dbx.Params{
			"email":  accountEmail,
			"homeID": homeID,
		},
	)
}

// categorizeTokens separates tokens into preferred (deviceCode) and other types,
// filtering out those that have exceeded their rate limit.
func (h *Handler) categorizeTokens(tokens []*core.Record) (*tokenSelection, error) {
	cutoff, err := tado.GetRatelimitCutoff()
	if err != nil {
		return nil, err
	}

	selection := &tokenSelection{}

	for _, token := range tokens {
		client, err := h.app.FindRecordById("clients", token.GetString("client"))
		if err != nil {
			return nil, err
		}

		selection.totalLimit += client.GetInt("dailyLimit")
		count, err := h.getTokenUsageCount(token.Id, cutoff)
		if err != nil {
			return nil, err
		}

		selection.totalUsed += count
		if count >= client.GetInt("dailyLimit") {
			continue
		}

		tc := tokenWithClient{client: client, token: token}
		if client.GetString("type") == "deviceCode" {
			selection.preferred = append(selection.preferred, tc)
		} else {
			selection.other = append(selection.other, tc)
		}
	}
	return selection, nil
}

// getTokenUsageCount returns the number of requests made with a token since the cutoff time.
func (h *Handler) getTokenUsageCount(tokenID string, cutoff time.Time) (int, error) {
	var count int

	err := h.app.DB().NewQuery(
		"SELECT count(*) FROM requests WHERE token = {:tokenID} AND created > {:cutoff}",
	).Bind(dbx.Params{
		"tokenID": tokenID,
		"cutoff":  cutoff,
	}).Row(&count)

	return count, err
}

// buildTargetURL constructs the target URL for the Tado API.
func (h *Handler) buildTargetURL(requestURL *url.URL) url.URL {
	targetURL := url.URL{
		Scheme:   "https",
		Host:     "my.tado.com",
		Path:     requestURL.Path,
		RawQuery: requestURL.RawQuery,
	}

	// Add query param to match real traffic
	if targetURL.RawQuery != "" {
		targetURL.RawQuery += "&ngsw-bypass=true"
	} else {
		targetURL.RawQuery = "ngsw-bypass=true"
	}

	return targetURL
}

// tryProxyRequest attempts to proxy the request using the given token.
// Returns nil result if the token is invalid and should be skipped.
func (h *Handler) tryProxyRequest(e *core.RequestEvent, t tokenWithClient, targetURL url.URL, bodyBytes []byte) (*proxyResult, error) {
	apiClient := h.createAPIClient(t.client)
	request := apiClient.R().
		SetContext(e.Request.Context()).
		SetHeader("authorization", "Bearer "+t.token.GetString("accessToken"))

	for k, v := range e.Request.Header {
		if k == "Authorization" || k == "X-Tado-Email" || k == "Host" || k == "Accept-Encoding" {
			continue
		}
		for _, vv := range v {
			request.SetHeader(k, vv)
		}
	}

	if len(bodyBytes) > 0 {
		request.SetBody(bytes.NewReader(bodyBytes))
	}

	resp, err := request.Send(e.Request.Method, targetURL.String())
	if err != nil {
		h.app.Logger().Error("proxy request failed", "error", err)
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		h.markTokenInvalid(t.token)
		return nil, nil
	}

	h.updateTokenUsed(t.token)

	return &proxyResult{response: resp, token: t}, nil
}

// createAPIClient creates the appropriate API client based on the client platform.
func (h *Handler) createAPIClient(client *core.Record) *req.Client {
	if client.GetString("platform") == "mobile" {
		return tado.NewIOSSafariAPIClient()
	}
	return tado.NewFirefoxAPIClient()
}

// markTokenInvalid marks a token as invalid and saves it.
func (h *Handler) markTokenInvalid(token *core.Record) {
	token.Set("used", time.Now())
	token.Set("status", "invalid")
	if err := h.app.Save(token); err != nil {
		h.app.Logger().Error("failed to mark token invalid", "error", err)
	}
}

// updateTokenUsed updates the token's last used timestamp.
func (h *Handler) updateTokenUsed(token *core.Record) {
	token.Set("used", time.Now())
	if err := h.app.Save(token); err != nil {
		h.app.Logger().Error("failed to update token used time", "error", err)
	}
}

// updateClientRateLimit updates the client's rate limit if the response header indicates a change.
func (h *Handler) updateClientRateLimit(client *core.Record, policyHeader string) {
	if policyHeader == "" {
		return
	}

	re := regexp.MustCompile(`q=(\d+)`)
	matches := re.FindStringSubmatch(policyHeader)
	if len(matches) < 2 {
		return
	}

	limit, err := strconv.Atoi(matches[1])
	if err != nil {
		return
	}

	if client.GetInt("dailyLimit") != limit {
		client.Set("dailyLimit", limit)
		if err := h.app.Save(client); err != nil {
			h.app.Logger().Error("failed to update client rate limit", "client", client.GetString("name"), "error", err)
		} else {
			h.app.Logger().Info("updated client rate limit", "client", client.GetString("name"), "limit", limit)
		}
	}
}

// writeProxyResponse writes the proxy response to the client.
func (h *Handler) writeProxyResponse(e *core.RequestEvent, resp *req.Response, totalLimit, totalUsed int) {
	for k, v := range resp.Header {
		for _, vv := range v {
			e.Response.Header().Add(k, vv)
		}
	}

	rateLimitPolicy := fmt.Sprintf(`"perday";q=%d;w=86400`, totalLimit)
	rateLimit := fmt.Sprintf(`"perday";r=%d`, totalLimit-totalUsed-1)

	e.Response.Header().Set("Ratelimit-Policy", rateLimitPolicy)
	e.Response.Header().Set("Ratelimit", fmt.Sprintf(`"perday";r=%d`, totalLimit-totalUsed-1))

	// compatibilty for tado_hijack
	e.Response.Header()["RateLimit-Policy"] = []string{rateLimitPolicy}
	e.Response.Header()["RateLimit"] = []string{rateLimit}

	e.Response.WriteHeader(resp.StatusCode)
	if _, err := e.Response.Write(resp.Bytes()); err != nil {
		h.app.Logger().Error("failed to write response", "error", err)
	}
}

// logRequest creates a request log entry in the database.
func (h *Handler) logRequest(tokenID, method, url string, status int) {
	requestsCollection, err := h.app.FindCollectionByNameOrId("requests")
	if err != nil {
		h.app.Logger().Error("failed to find requests collection", "error", err)
		return
	}

	requestRecord := core.NewRecord(requestsCollection)
	requestRecord.Set("token", tokenID)
	requestRecord.Set("method", method)
	requestRecord.Set("url", url)
	requestRecord.Set("status", status)

	if err := h.app.Save(requestRecord); err != nil {
		h.app.Logger().Error("failed to log request", "error", err)
	}
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

	cutoff, err := tado.GetRatelimitCutoff()
	if err != nil {
		return err
	}

	usage := map[string]any{}

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

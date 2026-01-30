package tado

import (
	"time"

	"github.com/imroc/req/v3"
	"github.com/imroc/req/v3/http2"
)

const (
	// Firefox User Agent from real traffic
	FirefoxUserAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:147.0) Gecko/20100101 Firefox/147.0"
)

// Firefox HTTP/2 settings matching real traffic
var (
	firefoxHttp2Settings = []http2.Setting{
		{
			ID:  http2.SettingHeaderTableSize,
			Val: 65536,
		},
		{
			ID:  http2.SettingInitialWindowSize,
			Val: 131072,
		},
		{
			ID:  http2.SettingMaxFrameSize,
			Val: 16384,
		},
	}

	firefoxPseudoHeaderOrder = []string{
		":method",
		":path",
		":authority",
		":scheme",
	}

	// Firefox priority frames matching real traffic
	firefoxPriorityFrames = []http2.PriorityFrame{
		{
			StreamID: 3,
			PriorityParam: http2.PriorityParam{
				StreamDep: 0,
				Exclusive: false,
				Weight:    200,
			},
		},
		{
			StreamID: 5,
			PriorityParam: http2.PriorityParam{
				StreamDep: 0,
				Exclusive: false,
				Weight:    100,
			},
		},
		{
			StreamID: 7,
			PriorityParam: http2.PriorityParam{
				StreamDep: 0,
				Exclusive: false,
				Weight:    0,
			},
		},
		{
			StreamID: 9,
			PriorityParam: http2.PriorityParam{
				StreamDep: 7,
				Exclusive: false,
				Weight:    0,
			},
		},
		{
			StreamID: 11,
			PriorityParam: http2.PriorityParam{
				StreamDep: 3,
				Exclusive: false,
				Weight:    0,
			},
		},
		{
			StreamID: 13,
			PriorityParam: http2.PriorityParam{
				StreamDep: 0,
				Exclusive: false,
				Weight:    240,
			},
		},
	}

	// Header order for Firefox auth POST requests matching real traffic
	firefoxAuthHeaderOrder = []string{
		"user-agent",
		"accept",
		"accept-language",
		"accept-encoding",
		"referer",
		"content-type",
		"content-length",
		"origin",
		"dnt",
		"sec-gpc",
		"connection",
		"cookie",
		"upgrade-insecure-requests",
		"sec-fetch-dest",
		"sec-fetch-mode",
		"sec-fetch-site",
		"sec-fetch-user",
		"priority",
		"pragma",
		"cache-control",
		"te",
	}

	// Header order for Firefox token exchange requests matching real traffic
	firefoxTokenHeaderOrder = []string{
		"user-agent",
		"accept",
		"accept-language",
		"accept-encoding",
		"referer",
		"content-type",
		"content-length",
		"origin",
		"dnt",
		"sec-gpc",
		"connection",
		"sec-fetch-dest",
		"sec-fetch-mode",
		"sec-fetch-site",
		"pragma",
		"cache-control",
		"te",
	}

	// Header order for Firefox API requests matching real traffic
	firefoxAPIHeaderOrder = []string{
		"user-agent",
		"accept",
		"accept-language",
		"accept-encoding",
		"referer",
		"x-amzn-trace-id",
		"authorization",
		"origin",
		"dnt",
		"sec-gpc",
		"connection",
		"sec-fetch-dest",
		"sec-fetch-mode",
		"sec-fetch-site",
	}

	firefoxHeaderPriority = http2.PriorityParam{
		StreamDep: 13,
		Exclusive: false,
		Weight:    41,
	}
)

// NewFirefoxClient creates an HTTP client that impersonates Firefox
// with proper TLS fingerprinting and header ordering matching real traffic
func NewFirefoxClient() *req.Client {
	client := req.C().
		// Set Firefox TLS fingerprint
		SetTLSFingerprintFirefox().
		// Set HTTP/2 settings matching Firefox
		SetHTTP2SettingsFrame(firefoxHttp2Settings...).
		SetHTTP2ConnectionFlow(12517377).
		SetHTTP2PriorityFrames(firefoxPriorityFrames...).
		// Set pseudo header order for HTTP/2
		SetCommonPseudoHeaderOder(firefoxPseudoHeaderOrder...).
		// Set HTTP/2 header priority
		SetHTTP2HeaderPriority(firefoxHeaderPriority).
		// Set timeout
		SetTimeout(30 * time.Second).
		// Enable auto-decompression for gzip/deflate/br responses
		EnableAutoDecompress().
		// Disable automatic redirect following (we handle redirects manually)
		SetRedirectPolicy(req.NoRedirectPolicy())

	return client
}

// NewFirefoxAuthClient creates an HTTP client configured for auth requests (Firefox)
func NewFirefoxAuthClient() *req.Client {
	client := NewFirefoxClient().
		SetCommonHeaderOrder(firefoxAuthHeaderOrder...).
		SetCommonHeaders(map[string]string{
			"accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			"accept-language":           "de,en;q=0.9",
			"accept-encoding":           "gzip, deflate, br, zstd",
			"dnt":                       "1",
			"sec-gpc":                   "1",
			"connection":                "keep-alive",
			"upgrade-insecure-requests": "1",
			"sec-fetch-dest":            "document",
			"sec-fetch-mode":            "navigate",
			"sec-fetch-site":            "same-origin",
			"sec-fetch-user":            "?1",
			"priority":                  "u=0, i",
			"pragma":                    "no-cache",
			"cache-control":             "no-cache",
			"te":                        "trailers",
		}).
		SetUserAgent(FirefoxUserAgent)

	return client
}

// NewFirefoxTokenClient creates an HTTP client configured for token exchange requests (Firefox)
func NewFirefoxTokenClient() *req.Client {
	client := NewFirefoxClient().
		SetCommonHeaderOrder(firefoxTokenHeaderOrder...).
		SetCommonHeaders(map[string]string{
			"accept":          "application/json, text/plain, */*",
			"accept-language": "de,en;q=0.9",
			"accept-encoding": "gzip, deflate, br, zstd",
			"referer":         "https://app.tado.com/",
			"origin":          "https://app.tado.com",
			"dnt":             "1",
			"sec-gpc":         "1",
			"connection":      "keep-alive",
			"sec-fetch-dest":  "empty",
			"sec-fetch-mode":  "cors",
			"sec-fetch-site":  "same-site",
			"pragma":          "no-cache",
			"cache-control":   "no-cache",
			"te":              "trailers",
		}).
		SetUserAgent(FirefoxUserAgent)

	return client
}

// NewFirefoxAPIClient creates an HTTP client configured for API requests (Firefox)
func NewFirefoxAPIClient() *req.Client {
	client := NewFirefoxClient().
		SetCommonHeaderOrder(firefoxAPIHeaderOrder...).
		SetCommonHeaders(map[string]string{
			"accept":          "application/json, text/plain, */*",
			"accept-language": "de,en;q=0.9",
			"accept-encoding": "gzip, deflate, br, zstd",
			"referer":         "https://app.tado.com/",
			"x-amzn-trace-id": "tado=webapp-release/v3835",
			"origin":          "https://app.tado.com",
			"dnt":             "1",
			"sec-gpc":         "1",
			"connection":      "keep-alive",
			"sec-fetch-dest":  "empty",
			"sec-fetch-mode":  "cors",
			"sec-fetch-site":  "same-site",
		}).
		SetUserAgent(FirefoxUserAgent)

	return client
}

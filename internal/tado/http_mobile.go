package tado

import (
	"time"

	"github.com/imroc/req/v3"
	"github.com/imroc/req/v3/http2"
)

const (
	// User agent matching real iOS Safari traffic
	IOSUserAgent = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.2 Mobile/15E148 Safari/604.1"

	// User agent for API requests
	IOSAPIUserAgent = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148"
)

// iOS Safari HTTP/2 settings matching real traffic
var (
	iosSafariHttp2Settings = []http2.Setting{
		{
			ID:  http2.SettingInitialWindowSize,
			Val: 4194304,
		},
		{
			ID:  http2.SettingMaxConcurrentStreams,
			Val: 100,
		},
	}

	iosSafariPseudoHeaderOrder = []string{
		":method",
		":scheme",
		":path",
		":authority",
	}

	// Header order for auth requests (POST to oauth2/authorize) matching real iOS traffic
	iosAuthHeaderOrder = []string{
		"accept",
		"content-type",
		"sec-fetch-site",
		"origin",
		"sec-fetch-mode",
		"user-agent",
		"referer",
		"sec-fetch-dest",
		"content-length",
		"accept-language",
		"priority",
		"accept-encoding",
		"cookie",
	}

	// Header order for token exchange requests matching real iOS traffic
	iosTokenHeaderOrder = []string{
		"accept",
		"content-type",
		"accept-language",
		"accept-encoding",
		"user-agent",
		"priority",
		"content-length",
	}

	// Header order for API requests matching real iOS traffic
	iosAPIHeaderOrder = []string{
		"accept",
		"authorization",
		"x-amzn-trace-id",
		"accept-language",
		"user-agent",
		"priority",
		"accept-encoding",
	}

	iosSafariHeaderPriority = http2.PriorityParam{
		StreamDep: 0,
		Exclusive: false,
		Weight:    254,
	}
)

// NewIOSSafariClient creates an HTTP client that impersonates iOS Safari
// with proper TLS fingerprinting and header ordering matching real traffic
func NewIOSSafariClient() *req.Client {
	client := req.C().
		// Set iOS Safari TLS fingerprint
		SetTLSFingerprintIOS().
		// Set HTTP/2 settings matching iOS Safari
		SetHTTP2SettingsFrame(iosSafariHttp2Settings...).
		SetHTTP2ConnectionFlow(10485760).
		// Set pseudo header order for HTTP/2
		SetCommonPseudoHeaderOder(iosSafariPseudoHeaderOrder...).
		// Set HTTP/2 header priority
		SetHTTP2HeaderPriority(iosSafariHeaderPriority).
		// Set timeout
		SetTimeout(30 * time.Second).
		// Enable auto-decompression for gzip/deflate/br responses
		EnableAutoDecompress().
		// Disable automatic redirect following (we handle redirects manually)
		SetRedirectPolicy(req.NoRedirectPolicy())

	return client
}

// NewIOSSafariAuthClient creates an HTTP client configured for auth requests
func NewIOSSafariAuthClient() *req.Client {
	client := NewIOSSafariClient().
		SetCommonHeaderOrder(iosAuthHeaderOrder...).
		SetCommonHeaders(map[string]string{
			"accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			"accept-language": "de-DE,de;q=0.9",
			"accept-encoding": "gzip, deflate, br",
			"priority":        "u=0, i",
			"sec-fetch-dest":  "document",
			"sec-fetch-mode":  "navigate",
			"sec-fetch-site":  "same-origin",
		}).
		SetUserAgent(IOSUserAgent)

	return client
}

// NewIOSSafariTokenClient creates an HTTP client configured for token exchange requests
func NewIOSSafariTokenClient() *req.Client {
	client := NewIOSSafariClient().
		SetCommonHeaderOrder(iosTokenHeaderOrder...).
		SetCommonHeaders(map[string]string{
			"accept":          "*/*",
			"accept-language": "de-DE,de;q=0.9",
			"accept-encoding": "gzip, deflate, br",
			"priority":        "u=3",
		}).
		// Token requests use the native app user agent
		SetUserAgent("tado/14903 CFNetwork/3860.300.31 Darwin/25.2.0")

	return client
}

// NewIOSSafariAPIClient creates an HTTP client configured for API requests
func NewIOSSafariAPIClient() *req.Client {
	client := NewIOSSafariClient().
		SetCommonHeaderOrder(iosAPIHeaderOrder...).
		SetCommonHeaders(map[string]string{
			"accept":          "application/json, text/plain, */*",
			"accept-language": "de-DE,de;q=0.9",
			"accept-encoding": "gzip, deflate, br",
			"priority":        "u=3",
			"x-amzn-trace-id": "tado=iOS-14903",
		}).
		SetUserAgent(IOSAPIUserAgent)

	return client
}

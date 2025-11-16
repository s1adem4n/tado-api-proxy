package proxy

import (
	"net/http/httptest"
	"testing"
)

func TestURLConstruction(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		rawQuery       string
		expectedURL    string
	}{
		{
			name:        "Path without query parameters",
			path:        "/api/v2/homes/12345/zones/2/schedule/timetables/0/blocks/MONDAY_TO_SUNDAY",
			rawQuery:    "",
			expectedURL: "https://my.tado.com/api/v2/homes/12345/zones/2/schedule/timetables/0/blocks/MONDAY_TO_SUNDAY",
		},
		{
			name:        "Path with query parameters",
			path:        "/api/v2/me",
			rawQuery:    "foo=bar&baz=qux",
			expectedURL: "https://my.tado.com/api/v2/me?foo=bar&baz=qux",
		},
		{
			name:        "Path with single query parameter",
			path:        "/api/v2/homes/12345",
			rawQuery:    "detail=true",
			expectedURL: "https://my.tado.com/api/v2/homes/12345?detail=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test request
			req := httptest.NewRequest("GET", tt.path+"?"+tt.rawQuery, nil)
			
			// Construct the target URL using the same logic as ServeHTTP
			targetURL := BaseURL + req.URL.Path
			if req.URL.RawQuery != "" {
				targetURL += "?" + req.URL.RawQuery
			}

			if targetURL != tt.expectedURL {
				t.Errorf("Expected URL %q, got %q", tt.expectedURL, targetURL)
			}
		})
	}
}

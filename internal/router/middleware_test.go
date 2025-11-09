package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestCSRFProtectionMiddleware(t *testing.T) {
	logger := testutil.TestLogger(t)
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	protected := csrfProtectionMiddleware(logger, handler)

	tests := []struct {
		name           string
		method         string
		headers        map[string]string
		expectedStatus int
		description    string
	}{
		{
			name:           "GET request should be allowed (safe method)",
			method:         "GET",
			headers:        map[string]string{},
			expectedStatus: http.StatusOK,
			description:    "Safe methods like GET should always be allowed",
		},
		{
			name:           "HEAD request should be allowed (safe method)",
			method:         "HEAD",
			headers:        map[string]string{},
			expectedStatus: http.StatusOK,
			description:    "Safe methods like HEAD should always be allowed",
		},
		{
			name:           "OPTIONS request should be allowed (safe method)",
			method:         "OPTIONS",
			headers:        map[string]string{},
			expectedStatus: http.StatusOK,
			description:    "Safe methods like OPTIONS should always be allowed",
		},
		{
			name:   "POST request with Sec-Fetch-Site same-origin should be allowed",
			method: "POST",
			headers: map[string]string{
				"Sec-Fetch-Site": "same-origin",
			},
			expectedStatus: http.StatusOK,
			description:    "Same-origin requests should be allowed",
		},
		{
			name:   "POST request with Sec-Fetch-Site none should be allowed",
			method: "POST",
			headers: map[string]string{
				"Sec-Fetch-Site": "none",
			},
			expectedStatus: http.StatusOK,
			description:    "Direct navigation (none) should be allowed",
		},
		{
			name:   "POST request with Sec-Fetch-Site cross-site should be rejected",
			method: "POST",
			headers: map[string]string{
				"Sec-Fetch-Site": "cross-site",
			},
			expectedStatus: http.StatusForbidden,
			description:    "Cross-site POST requests should be rejected with 403",
		},
		{
			name:   "PUT request with Sec-Fetch-Site cross-site should be rejected",
			method: "PUT",
			headers: map[string]string{
				"Sec-Fetch-Site": "cross-site",
			},
			expectedStatus: http.StatusForbidden,
			description:    "Cross-site PUT requests should be rejected with 403",
		},
		{
			name:   "DELETE request with Sec-Fetch-Site cross-site should be rejected",
			method: "DELETE",
			headers: map[string]string{
				"Sec-Fetch-Site": "cross-site",
			},
			expectedStatus: http.StatusForbidden,
			description:    "Cross-site DELETE requests should be rejected with 403",
		},
		{
			name:    "POST request without headers (assumed same-origin/non-browser)",
			method:  "POST",
			headers: map[string]string{
				// No headers
			},
			expectedStatus: http.StatusOK,
			description:    "Requests without Sec-Fetch-Site are assumed same-origin or non-browser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "http://example.com/test", nil)

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			rr := httptest.NewRecorder()

			protected.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("%s: expected status %d, got %d. Error: %s. Description: %s",
					tt.name, tt.expectedStatus, rr.Code, rr.Body, tt.description)
			}
		})
	}
}

func TestCSRFProtectionMiddleware_WithOriginHeader(t *testing.T) {
	logger := testutil.TestLogger(t)
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	protected := csrfProtectionMiddleware(logger, handler)

	tests := []struct {
		name           string
		method         string
		origin         string
		host           string
		expectedStatus int
		description    string
	}{
		{
			name:           "POST with matching Origin and Host should be allowed",
			method:         "POST",
			origin:         "http://example.com",
			host:           "example.com",
			expectedStatus: http.StatusOK,
			description:    "Same-origin requests via Origin header should be allowed",
		},
		{
			name:           "POST with different Origin and Host should be rejected",
			method:         "POST",
			origin:         "http://attacker.com",
			host:           "example.com",
			expectedStatus: http.StatusForbidden,
			description:    "Cross-origin requests should be rejected",
		},
		{
			name:           "GET with different Origin and Host should be allowed",
			method:         "GET",
			origin:         "http://attacker.com",
			host:           "example.com",
			expectedStatus: http.StatusOK,
			description:    "Safe methods should be allowed even cross-origin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "http://"+tt.host+"/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.host != "" {
				req.Host = tt.host
			}

			rr := httptest.NewRecorder()

			protected.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("%s: expected status %d, got %d. Description: %s",
					tt.name, tt.expectedStatus, rr.Code, tt.description)
			}
		})
	}
}

func TestCSRFProtectionMiddleware_TrustedOrigins(t *testing.T) {
	logger := testutil.TestLogger(t)
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	t.Setenv("EXPENSETRACE_TRUSTED_ORIGINS", "https://another.com")
	protected := csrfProtectionMiddleware(logger, handler)

	tests := []struct {
		name           string
		method         string
		origin         string
		host           string
		expectedStatus int
		description    string
	}{
		{
			name:           "POST with trusted Origin and Host should be allowed",
			method:         "POST",
			origin:         "https://another.com",
			host:           "example.com",
			expectedStatus: http.StatusOK,
			description:    "Same-origin requests via Origin header should be allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "http://"+tt.host+"/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.host != "" {
				req.Host = tt.host
			}

			rr := httptest.NewRecorder()

			protected.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("%s: expected status %d, got %d. Description: %s",
					tt.name, tt.expectedStatus, rr.Code, tt.description)
			}
		})
	}
}

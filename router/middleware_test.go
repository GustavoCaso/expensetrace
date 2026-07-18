package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GustavoCaso/expensetrace/domain"
	"github.com/GustavoCaso/expensetrace/testutil"
)

func TestAuthMiddlewareRedirectsWithoutCookie(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, _ := testutil.SetupTestStorage(t, logger)
	r := newRouter(s, logger)

	protected := authMiddleware(r, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/expenses", nil)
	rr := httptest.NewRecorder()
	protected.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("expected status %d, got %d", http.StatusSeeOther, rr.Code)
	}
	if location := rr.Header().Get("Location"); location != "/signin" {
		t.Errorf("expected redirect to /signin, got %q", location)
	}
}

func TestAuthMiddlewareRedirectsWithInvalidSession(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, _ := testutil.SetupTestStorage(t, logger)
	r := newRouter(s, logger)

	protected := authMiddleware(r, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/expenses", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "invalid-session"})
	rr := httptest.NewRecorder()
	protected.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("expected status %d, got %d", http.StatusSeeOther, rr.Code)
	}
	if location := rr.Header().Get("Location"); location != "/signin" {
		t.Errorf("expected redirect to /signin, got %q", location)
	}
}

func TestAuthMiddlewareSkipsAuthEndpoints(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, _ := testutil.SetupTestStorage(t, logger)
	r := newRouter(s, logger)

	for _, path := range []string{"/signin", "/signup", "/static/css/base.css"} {
		t.Run(path, func(t *testing.T) {
			var base domain.ViewBase
			protected := authMiddleware(r, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				base = viewBaseFromContext(req.Context())
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, path, nil)
			rr := httptest.NewRecorder()
			protected.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
			}
			if base.LoggedIn {
				t.Error("expected base view not being created for auth endpoints ")
			}
		})
	}
}

func TestAuthMiddlewareSetsUserAndViewBase(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)
	r := newRouter(s, logger)

	tests := []struct {
		path         string
		expectedPage string
	}{
		{"/", pageReports},
		{"/expenses", pageExpenses},
		{"/expense/1", pageExpenses},
		{"/categories", pageCategories},
		{"/category/new", pageCategories},
		{"/import", pageImport},
		{"/profile", pageProfile},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			var userID int64
			var base domain.ViewBase
			protected := authMiddleware(r, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				userID = userIDFromContext(req.Context())
				base = viewBaseFromContext(req.Context())
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
			rr := httptest.NewRecorder()
			protected.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
			}
			if userID != user.ID() {
				t.Errorf("expected user ID %d in context, got %d", user.ID(), userID)
			}
			if !base.LoggedIn {
				t.Error("expected ViewBase.LoggedIn to be true")
			}
			if base.Username != user.Username() {
				t.Errorf("expected username %q, got %q", user.Username(), base.Username)
			}
			if base.UsernameInitials != "TE" {
				t.Errorf("expected initials %q, got %q", "TE", base.UsernameInitials)
			}
			if base.CurrentPage != tt.expectedPage {
				t.Errorf("expected current page %q, got %q", tt.expectedPage, base.CurrentPage)
			}
		})
	}
}

func TestCurrentPageFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/", pageReports},
		{"/unknown", pageReports},
		{"/expenses", pageExpenses},
		{"/expenses/export", pageExpenses},
		{"/expense/new", pageExpenses},
		{"/categories", pageCategories},
		{"/category/uncategorized", pageCategories},
		{"/import", pageImport},
		{"/import/execute", pageImport},
		{"/profile", pageProfile},
		{"/profile/password", pageProfile},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := currentPageFromPath(tt.path); got != tt.expected {
				t.Errorf("currentPageFromPath(%q) = %q, expected %q", tt.path, got, tt.expected)
			}
		})
	}
}

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

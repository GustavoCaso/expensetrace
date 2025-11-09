package router

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/logger"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader wraps the regular ResponseWriter so we can store the status code to later log it.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(logger *logger.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the response writer to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Call the next handler
		next.ServeHTTP(wrapped, r)

		// Log the request
		duration := time.Since(start)
		logger.Debug("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration", duration.String(),
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)
	})
}

func xFrameDenyHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func liveReloadMiddleware(router *router, handlder http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := router.parseTemplates()
		if err != nil {
			router.logger.Warn("Error parsing templates during live reload", "error", err.Error())
		}

		handlder.ServeHTTP(w, r)
	})
}

// csrfProtectionMiddleware provides CSRF protection using Go 1.25's http.CrossOriginProtection.
// It protects against Cross-Site Request Forgery attacks by checking the Sec-Fetch-Site header
// or comparing the Origin header with the Host header for non-safe HTTP methods.
//
// The middleware automatically rejects non-safe cross-origin browser requests.
// Safe methods (GET, HEAD, OPTIONS) are always allowed.
//
// You can configure trusted origins via the EXPENSETRACE_TRUSTED_ORIGINS environment variable
// (comma-separated list) if you need to allow cross-origin requests from specific domains.
func csrfProtectionMiddleware(next http.Handler) http.Handler {
	// Create CrossOriginProtection with default settings
	// The zero value is valid and provides CSRF protection
	csrf := &http.CrossOriginProtection{}

	// Allow trusted origins from environment variable if specified
	// This is useful if you have multiple domains serving the same app
	if trustedOrigins := os.Getenv("EXPENSETRACE_TRUSTED_ORIGINS"); trustedOrigins != "" {
		origins := strings.Split(trustedOrigins, ",")
		for _, origin := range origins {
			origin = strings.TrimSpace(origin)
			if origin != "" {
				err := csrf.AddTrustedOrigin(origin)
				if err != nil {
					// Log error but continue - don't fail the entire app
					// In production, you might want to handle this differently
					continue
				}
			}
		}
	}

	// Return the handler that applies CSRF protection
	return csrf.Handler(next)
}

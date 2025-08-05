package router

import (
	"net/http"
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

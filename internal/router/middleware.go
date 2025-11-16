package router

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/logger"
	"github.com/GustavoCaso/expensetrace/internal/matcher"
	"github.com/GustavoCaso/expensetrace/internal/storage"
)

type contextKey string

const userIDKey contextKey = "userID"

// userIDFromContext retrieves user ID from context.
func userIDFromContext(ctx context.Context) int64 {
	if userID, ok := ctx.Value(userIDKey).(int64); ok {
		return userID
	}
	return 0
}

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

func authMiddleware(router *router, s storage.Storage, logger *logger.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for auth endpoints and static files
		path := r.URL.Path
		if strings.HasPrefix(path, "/signin") ||
			strings.HasPrefix(path, "/signup") ||
			strings.HasPrefix(path, "/static/") {
			next.ServeHTTP(w, r)
			return
		}

		// Get session cookie
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			// No session cookie, redirect to signin
			http.Redirect(w, r, "/signin", http.StatusSeeOther)
			return
		}

		// Get session from database
		session, err := s.GetSession(r.Context(), cookie.Value)
		if err != nil {
			var notFoundErr *storage.NotFoundError
			if errors.As(err, &notFoundErr) {
				// Session not found or expired, redirect to signin
				http.Redirect(w, r, "/signin", http.StatusSeeOther)
				return
			}
			logger.Error("Failed to get session", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Add user ID to context
		ctx := context.WithValue(r.Context(), userIDKey, session.UserID())

		if router.matcher == nil {
			categories, categoryErr := s.GetCategories(context.Background(), session.UserID())
			if categoryErr != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			matcher := matcher.New(categories)
			router.matcher = matcher
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

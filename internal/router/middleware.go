package router

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/domain"
	"github.com/GustavoCaso/expensetrace/internal/logger"
)

type contextKey string

const (
	userIDKey   contextKey = "userID"
	viewBaseKey contextKey = "viewBase"
)

const (
	pageReports    = "reports"
	pageExpenses   = "expenses"
	pageCategories = "categories"
	pageImport     = "import"
	pageProfile    = "profile"
)

// userIDFromContext retrieves user ID from context.
func userIDFromContext(ctx context.Context) int64 {
	if userID, ok := ctx.Value(userIDKey).(int64); ok {
		return userID
	}
	return 0
}

// viewBaseFromContext retrieves the ViewBase stored by authMiddleware.
func viewBaseFromContext(ctx context.Context) domain.ViewBase {
	if base, ok := ctx.Value(viewBaseKey).(domain.ViewBase); ok {
		return base
	}
	return domain.ViewBase{}
}

// currentPageFromPath derives the current page from the request URL path.
func currentPageFromPath(path string) string {
	segment, _, _ := strings.Cut(strings.TrimPrefix(path, "/"), "/")
	switch segment {
	case "expense", "expenses":
		return pageExpenses
	case "category", "categories":
		return pageCategories
	case "import":
		return pageImport
	case "profile":
		return pageProfile
	default:
		return pageReports
	}
}

// getInitials returns the first two characters of a username in uppercase.
func getInitials(username string) string {
	if len(username) == 0 {
		return ""
	}
	if len(username) == 1 {
		return strings.ToUpper(username)
	}
	return strings.ToUpper(username[:2])
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

func authMiddleware(router *router, next http.Handler) http.Handler {
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

		// Resolve the session to its user
		user, err := router.authService.AuthenticatedUser(r.Context(), cookie.Value)
		if err != nil {
			var notFoundErr *domain.NotFoundError
			if errors.As(err, &notFoundErr) {
				// Session not found or expired, redirect to signin
				http.Redirect(w, r, "/signin", http.StatusSeeOther)
				return
			}
			router.logger.Error("Failed to authenticate session", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Add user ID and view base data to context
		ctx := context.WithValue(r.Context(), userIDKey, user.ID())
		ctx = context.WithValue(ctx, viewBaseKey, domain.ViewBase{
			LoggedIn:         true,
			Username:         user.Username(),
			UsernameInitials: getInitials(user.Username()),
			CurrentPage:      currentPageFromPath(path),
		})

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func csrfProtectionMiddleware(logger *logger.Logger, next http.Handler) http.Handler {
	csrf := &http.CrossOriginProtection{}

	if trustedOrigins := os.Getenv("EXPENSETRACE_TRUSTED_ORIGINS"); trustedOrigins != "" {
		origins := strings.Split(trustedOrigins, ",")
		for _, origin := range origins {
			origin = strings.TrimSpace(origin)
			if origin != "" {
				err := csrf.AddTrustedOrigin(origin)
				if err != nil {
					logger.Error("error adding trusted origin", "err", err.Error())
					continue
				}
			}
		}
	}

	return csrf.Handler(next)
}

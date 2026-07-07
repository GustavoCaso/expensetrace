package router

import (
	"net/http"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/domain"
	"github.com/GustavoCaso/expensetrace/internal/router/service/auth"
)

const (
	sessionCookieName = "session_id"
	maxFormSize       = 1 << 20 // 1MB
	// sessionDuration mirrors auth.SessionDuration; kept as a router-level
	// alias since it's referenced throughout router tests for cookie setup.
	sessionDuration = auth.SessionDuration
)

type authHandler struct {
	router *router
}

func (a *authHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /signup", a.signupPage)
	mux.HandleFunc("POST /signup", a.signup)
	mux.HandleFunc("GET /signin", a.signinPage)
	mux.HandleFunc("POST /signin", a.signin)
	mux.HandleFunc("POST /signout", a.signout)
}

func (a *authHandler) signupPage(w http.ResponseWriter, _ *http.Request) {
	data := domain.ViewBase{
		LoggedIn: false,
	}

	a.router.templates.Render(w, "pages/auth/signup.html", data)
}

func (a *authHandler) signup(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFormSize)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	sessionID, expiresAt, validationErr, err := a.router.authService.Signup(
		r.Context(),
		username,
		password,
		confirmPassword,
	)
	if err != nil {
		a.router.logger.Error("Failed to signup", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if validationErr != nil {
		a.renderSignupError(w, validationErr.Error())
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // Secure flag controlled by EXPENSETRACE_SECURE_COOKIES
		Name:     sessionCookieName,
		Value:    sessionID,
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   a.router.secureCookie,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *authHandler) signinPage(w http.ResponseWriter, _ *http.Request) {
	data := domain.ViewBase{
		LoggedIn: false,
	}

	a.router.templates.Render(w, "pages/auth/signin.html", data)
}

func (a *authHandler) signin(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFormSize)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	sessionID, expiresAt, validationErr, err := a.router.authService.Signin(r.Context(), username, password)
	if err != nil {
		a.router.logger.Error("Failed to signin", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if validationErr != nil {
		a.renderSigninError(w, validationErr.Error())
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // Secure flag controlled by EXPENSETRACE_SECURE_COOKIES
		Name:     sessionCookieName,
		Value:    sessionID,
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   a.router.secureCookie,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *authHandler) signout(w http.ResponseWriter, r *http.Request) {
	// Get session cookie
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		// Delete session from database
		if err = a.router.authService.Signout(r.Context(), cookie.Value); err != nil {
			a.router.logger.Error("Failed to delete session", "error", err)
		}
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // Secure flag controlled by EXPENSETRACE_SECURE_COOKIES
		Name:     sessionCookieName,
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   a.router.secureCookie,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})

	//nolint:canonicalheader //HTMX header
	w.Header().Set("HX-Redirect", "/")
}

func (a *authHandler) renderSignupError(w http.ResponseWriter, errorMsg string) {
	data := domain.ViewBase{
		Error:    errorMsg,
		LoggedIn: false,
	}

	a.router.templates.Render(w, "pages/auth/signup.html", data)
}

func (a *authHandler) renderSigninError(w http.ResponseWriter, errorMsg string) {
	data := domain.ViewBase{
		Error:    errorMsg,
		LoggedIn: false,
	}

	a.router.templates.Render(w, "pages/auth/signin.html", data)
}

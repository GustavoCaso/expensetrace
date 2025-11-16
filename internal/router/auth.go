package router

import (
	"errors"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/util"
)

const (
	sessionCookieName = "session_id"
	sessionDuration   = 7 * 24 * time.Hour // 7 days
	minPasswordLength = 8
	sessionIDLength   = 32
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
	data := viewBase{
		LoggedIn: false,
	}

	a.router.templates.Render(w, "pages/auth/signup.html", data)
}

func (a *authHandler) signup(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	// Validate input
	if username == "" || password == "" {
		a.renderSignupError(w, "Username and password are required")
		return
	}

	if password != confirmPassword {
		a.renderSignupError(w, "Passwords do not match")
		return
	}

	if len(password) < minPasswordLength {
		a.renderSignupError(w, "Password must be at least 8 characters long")
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		a.router.logger.Error("Failed to hash password", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Create user
	user, err := a.router.storage.CreateUser(r.Context(), username, string(hashedPassword))
	if err != nil {
		a.router.logger.Error("Failed to create user", "error", err, "username", username)
		a.renderSignupError(w, "Username already exists or database error occurred")
		return
	}

	// Create session
	sessionID := generateSessionID()

	expiresAt := time.Now().Add(sessionDuration)
	_, err = a.router.storage.CreateSession(r.Context(), user.ID(), sessionID, expiresAt)
	if err != nil {
		a.router.logger.Error("Failed to create session", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Expires:  expiresAt,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})

	// Create default Exclude category for new user
	_, err = a.router.storage.CreateCategory(r.Context(), user.ID(), storage.ExcludeCategory, "$a")
	if err != nil {
		a.router.logger.Error("Failed to create exclude category for new user", "error", err, "user_id", user.ID())
	}

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *authHandler) signinPage(w http.ResponseWriter, _ *http.Request) {
	data := viewBase{
		LoggedIn: false,
	}

	a.router.templates.Render(w, "pages/auth/signin.html", data)
}

func (a *authHandler) signin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	// Validate input
	if username == "" || password == "" {
		a.renderSigninError(w, "Username and password are required")
		return
	}

	// Get user
	user, err := a.router.storage.GetUserByUsername(r.Context(), username)
	if err != nil {
		var notFoundErr *storage.NotFoundError
		if errors.As(err, &notFoundErr) {
			a.renderSigninError(w, "Invalid username or password")
			return
		}
		a.router.logger.Error("Failed to get user", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash()), []byte(password))
	if err != nil {
		a.renderSigninError(w, "Invalid username or password")
		return
	}

	// Create session
	sessionID := generateSessionID()

	expiresAt := time.Now().Add(sessionDuration)
	_, err = a.router.storage.CreateSession(r.Context(), user.ID(), sessionID, expiresAt)
	if err != nil {
		a.router.logger.Error("Failed to create session", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Expires:  expiresAt,
		HttpOnly: true,
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
		if err = a.router.storage.DeleteSession(r.Context(), cookie.Value); err != nil {
			a.router.logger.Error("Failed to delete session", "error", err)
		}
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Hx-Redirect", "/")
}

func (a *authHandler) renderSignupError(w http.ResponseWriter, errorMsg string) {
	data := viewBase{
		Error:    errorMsg,
		LoggedIn: false,
	}

	a.router.templates.Render(w, "pages/auth/signup.html", data)
}

func (a *authHandler) renderSigninError(w http.ResponseWriter, errorMsg string) {
	data := viewBase{
		Error:    errorMsg,
		LoggedIn: false,
	}

	a.router.templates.Render(w, "pages/auth/signin.html", data)
}

func generateSessionID() string {
	return util.RandomString(sessionIDLength)
}

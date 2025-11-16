package router

import (
	"errors"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"github.com/GustavoCaso/expensetrace/internal/storage"
)

type profileHandler struct {
	router *router
}

func (p *profileHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /profile", func(w http.ResponseWriter, r *http.Request) {
		p.profilePage(w, r, nil, nil)
	})
	mux.HandleFunc("POST /profile/username", p.updateUsername)
	mux.HandleFunc("POST /profile/password", p.updatePassword)
}

func (p *profileHandler) profilePage(w http.ResponseWriter, r *http.Request, banner *banner, err error) {
	ctx := r.Context()
	base := newViewBase(ctx, p.router.storage, p.router.logger, pageProfile)
	if banner != nil {
		base.Banner = *banner
	}

	if err != nil {
		base.Error = err.Error()
	}

	p.router.templates.Render(w, "pages/profile/index.html", base)
}

func (p *profileHandler) updateUsername(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := userIDFromContext(ctx)

	if err := r.ParseForm(); err != nil {
		p.renderError(w, r, err)
		return
	}

	newUsername := r.FormValue("username")

	// Validate input
	if newUsername == "" {
		p.renderError(w, r, errors.New("username is required"))
		return
	}

	// Check if username already exists
	_, err := p.router.storage.GetUserByUsername(ctx, newUsername)
	if err == nil {
		p.renderError(w, r, errors.New("username already exists"))
		return
	}

	var notFoundErr *storage.NotFoundError
	if !errors.As(err, &notFoundErr) {
		p.router.logger.Error("Failed to check username", "error", err)
		p.renderError(w, r, errors.New("internal Server Error"))
		return
	}

	// Update username
	err = p.router.storage.UpdateUsername(ctx, userID, newUsername)
	if err != nil {
		p.router.logger.Error("Failed to update username", "error", err, "user_id", userID)
		p.renderError(w, r, errors.New("failed to update username"))
		return
	}

	p.router.logger.Info("Username updated", "user_id", userID, "new_username", newUsername)
	p.renderSuccess(w, r, "Username updated successfully")
}

func (p *profileHandler) updatePassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := userIDFromContext(ctx)

	if err := r.ParseForm(); err != nil {
		p.renderError(w, r, errors.New("invalid form data"))
		return
	}

	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")

	// Validate input
	if currentPassword == "" || newPassword == "" || confirmPassword == "" {
		p.renderError(w, r, errors.New("all fields are required"))
		return
	}

	if newPassword != confirmPassword {
		p.renderError(w, r, errors.New("new passwords do not match"))
		return
	}

	if len(newPassword) < minPasswordLength {
		p.renderError(w, r, errors.New("password must be at least 8 characters long"))
		return
	}

	// Get current user
	user, err := p.router.storage.GetUserByID(ctx, userID)
	if err != nil {
		p.router.logger.Error("Failed to get user", "error", err, "user_id", userID)
		p.renderError(w, r, errors.New("internal Server Error"))
		return
	}

	// Verify current password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash()), []byte(currentPassword))
	if err != nil {
		p.renderError(w, r, errors.New("current password is incorrect"))
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		p.router.logger.Error("Failed to hash password", "error", err)
		p.renderError(w, r, errors.New("internal Server Error"))
		return
	}

	// Update password
	err = p.router.storage.UpdatePassword(ctx, userID, string(hashedPassword))
	if err != nil {
		p.router.logger.Error("Failed to update password", "error", err, "user_id", userID)
		p.renderError(w, r, errors.New("failed to update password"))
		return
	}

	p.router.logger.Info("Password updated", "user_id", userID)
	p.renderSuccess(w, r, "Password changed successfully")
}

func (p *profileHandler) renderError(w http.ResponseWriter, r *http.Request, err error) {
	p.profilePage(w, r, nil, err)
}

func (p *profileHandler) renderSuccess(w http.ResponseWriter, r *http.Request, message string) {
	data := &banner{
		Icon:    "✓",
		Message: message,
	}
	p.profilePage(w, r, data, nil)
}

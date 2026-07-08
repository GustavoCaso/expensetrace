package router

import (
	"errors"
	"net/http"

	"github.com/GustavoCaso/expensetrace/internal/domain"
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

func (p *profileHandler) profilePage(w http.ResponseWriter, r *http.Request, banner *domain.Banner, err error) {
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

	r.Body = http.MaxBytesReader(w, r.Body, maxFormSize)
	if err := r.ParseForm(); err != nil {
		p.renderError(w, r, err)
		return
	}

	newUsername := r.FormValue("username")

	validationErr, err := p.router.profileService.UpdateUsername(ctx, userID, newUsername)
	if err != nil {
		p.router.logger.Error("Failed to update username", "error", err, "user_id", userID)
		p.renderError(w, r, err)
		return
	}

	if validationErr != nil {
		p.renderError(w, r, validationErr)
		return
	}

	p.renderSuccess(w, r, "Username updated successfully")
}

func (p *profileHandler) updatePassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := userIDFromContext(ctx)

	r.Body = http.MaxBytesReader(w, r.Body, maxFormSize)
	if err := r.ParseForm(); err != nil {
		p.renderError(w, r, errors.New("invalid form data"))
		return
	}

	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")

	validationErr, err := p.router.profileService.UpdatePassword(
		ctx,
		userID,
		currentPassword,
		newPassword,
		confirmPassword,
	)
	if err != nil {
		p.router.logger.Error("Failed to update password", "error", err, "user_id", userID)
		p.renderError(w, r, err)
		return
	}

	if validationErr != nil {
		p.renderError(w, r, validationErr)
		return
	}

	p.renderSuccess(w, r, "Password changed successfully")
}

func (p *profileHandler) renderError(w http.ResponseWriter, r *http.Request, err error) {
	p.profilePage(w, r, nil, err)
}

func (p *profileHandler) renderSuccess(w http.ResponseWriter, r *http.Request, message string) {
	data := &domain.Banner{
		Icon:    "✓",
		Message: message,
	}
	p.profilePage(w, r, data, nil)
}

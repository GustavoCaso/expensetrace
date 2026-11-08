package router

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/GustavoCaso/expensetrace/domain"
	"github.com/GustavoCaso/expensetrace/mcpserver"
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
	mux.HandleFunc("POST /profile/tokens", p.createToken)
	mux.HandleFunc("POST /profile/tokens/{id}/revoke", p.revokeToken)
}

func (p *profileHandler) profilePage(
	w http.ResponseWriter,
	r *http.Request,
	banner *domain.Banner,
	err error,
) {
	ctx := r.Context()
	data := domain.ProfileViewData{
		ViewBase: viewBaseFromContext(ctx),
		MCPURL:   mcpURL(r),
	}
	if banner != nil {
		data.Banner = *banner
	}

	if err != nil {
		data.Error = err.Error()
	}

	tokens, listErr := p.router.apiTokenService.List(ctx, userIDFromContext(ctx))
	if listErr != nil {
		p.router.logger.Error("Failed to list api tokens", "error", listErr)
		if data.Error == "" {
			data.Error = "Failed to load API tokens"
		}
	}
	data.APITokens = tokens

	p.router.renderHTML(w, http.StatusOK, data, "base", "pages/profile/index.html")
}

// mcpURL builds the absolute MCP endpoint URL as seen by the browser, so the
// profile page can show users what to paste into their agent.
func mcpURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host + mcpserver.Path
}

func (p *profileHandler) createToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := userIDFromContext(ctx)

	r.Body = http.MaxBytesReader(w, r.Body, maxFormSize)
	if err := r.ParseForm(); err != nil {
		p.renderError(w, r, errors.New("invalid form data"))
		return
	}

	_, plaintext, validationErr, err := p.router.apiTokenService.Create(ctx, userID, r.FormValue("name"))
	if err != nil {
		p.renderError(w, r, errors.New("failed to create API token"))
		return
	}

	if validationErr != nil {
		p.renderError(w, r, validationErr)
		return
	}

	p.profilePage(w, r, &domain.Banner{
		Icon:    "✓",
		Message: "API token created. Copy it now, it will not be shown again.",
		Code:    plaintext,
	}, nil)
}

func (p *profileHandler) revokeToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := userIDFromContext(ctx)

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		p.renderError(w, r, errors.New("invalid token id"))
		return
	}

	if revokeErr := p.router.apiTokenService.Revoke(ctx, userID, id); revokeErr != nil {
		var notFoundErr *domain.NotFoundError
		if errors.As(revokeErr, &notFoundErr) {
			p.renderError(w, r, errors.New("API token not found"))
			return
		}
		p.router.logger.Error("Failed to revoke api token", "error", revokeErr, "user_id", userID)
		p.renderError(w, r, errors.New("failed to revoke API token"))
		return
	}

	p.renderSuccess(w, r, "API token revoked")
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

package router

import (
	"net/http"

	"github.com/GustavoCaso/expensetrace/domain"
)

func (a *authHandler) forgotPasswordPage(w http.ResponseWriter, _ *http.Request) {
	data := domain.ForgotPasswordView{
		ViewBase: domain.ViewBase{LoggedIn: false},
	}

	a.router.renderHTML(w, http.StatusOK, data, "base", "pages/auth/forgot_password.html")
}

func (a *authHandler) forgotPassword(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFormSize)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")

	_, _, err := a.router.authService.RequestPasswordReset(r.Context(), username)
	if err != nil {
		a.router.logger.Error("Failed to request password reset", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := domain.ForgotPasswordView{
		ViewBase:  domain.ViewBase{LoggedIn: false},
		Submitted: true,
	}

	a.router.renderHTML(w, http.StatusOK, data, "base", "pages/auth/forgot_password.html")
}

func (a *authHandler) resetPasswordPage(w http.ResponseWriter, r *http.Request) {
	data := domain.ResetPasswordView{
		ViewBase: domain.ViewBase{LoggedIn: false},
		Token:    r.URL.Query().Get("token"),
	}

	a.router.renderHTML(w, http.StatusOK, data, "base", "pages/auth/reset_password.html")
}

func (a *authHandler) resetPassword(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFormSize)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	token := r.FormValue("token")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	validationErr, err := a.router.authService.ResetPassword(r.Context(), token, password, confirmPassword)
	if err != nil {
		a.router.logger.Error("Failed to reset password", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if validationErr != nil {
		data := domain.ResetPasswordView{
			ViewBase: domain.ViewBase{Error: validationErr.Error(), LoggedIn: false},
			Token:    token,
		}
		a.router.renderHTML(w, http.StatusOK, data, "base", "pages/auth/reset_password.html")
		return
	}

	http.Redirect(w, r, "/signin", http.StatusSeeOther)
}

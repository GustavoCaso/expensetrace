package domain

type Banner struct {
	Icon    string
	Message string
	// Code is an optional value rendered verbatim in monospace below the
	// message, selectable in one click (e.g. a freshly created API token).
	Code string
}

type ViewBase struct {
	Error            string
	Banner           Banner
	CurrentPage      string
	LoggedIn         bool
	Username         string
	UsernameInitials string
}

// ForgotPasswordView is the view model for the forgot-password page.
type ForgotPasswordView struct {
	ViewBase
	Submitted bool
}

// ResetPasswordView is the view model for the reset-password page.
type ResetPasswordView struct {
	ViewBase
	Token string
}

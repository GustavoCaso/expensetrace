package router

import (
	"context"
	"strings"

	"github.com/GustavoCaso/expensetrace/internal/logger"
	"github.com/GustavoCaso/expensetrace/internal/storage"
)

const (
	pageReports    = "reports"
	pageExpenses   = "expenses"
	pageCategories = "categories"
	pageImport     = "import"
	pageProfile    = "profile"
)

type banner struct {
	Icon    string
	Message string
}

type viewBase struct {
	Error            string
	Banner           banner
	CurrentPage      string
	LoggedIn         bool
	Username         string
	UsernameInitials string
}

// newViewBase creates a new viewBase with user information from context.
func newViewBase(ctx context.Context, s storage.Storage, logger *logger.Logger, currentPage string) viewBase {
	userID := userIDFromContext(ctx)
	username := ""
	usernameInitials := ""
	loggedIn := false

	if userID != 0 {
		loggedIn = true
		user, err := s.GetUserByID(ctx, userID)
		if err != nil {
			logger.Error("Failed to get user for view data", "error", err, "user_id", userID)
		} else {
			username = user.Username()
			usernameInitials = getInitials(username)
		}
	}

	return viewBase{
		LoggedIn:         loggedIn,
		Username:         username,
		UsernameInitials: usernameInitials,
		CurrentPage:      currentPage,
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

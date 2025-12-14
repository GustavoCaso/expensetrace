package testutil

import (
	"net/http"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/util"
)

func SetupAuthCookie(
	t *testing.T,
	s storage.Storage,
	req *http.Request,
	user storage.User,
	cookieKey string,
	duration time.Duration,
) {
	const idLength = 16
	sessionID := util.GenerateRandomID(idLength)
	expiresAt := time.Now().Add(duration)
	_, err := s.CreateSession(t.Context(), user.ID(), sessionID, expiresAt)
	if err != nil {
		t.Fatal("failed to create session")
	}
	cookie := &http.Cookie{
		Name:     cookieKey,
		Value:    sessionID,
		Expires:  expiresAt,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	}
	req.Header.Set("Cookie", cookie.String())
}

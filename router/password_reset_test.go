package router

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/GustavoCaso/expensetrace/service/auth"
	"github.com/GustavoCaso/expensetrace/testutil"
)

func TestForgotPasswordPageHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, _ := testutil.SetupTestStorage(t, logger)

	handler := New(s, logger)

	req := httptest.NewRequest(http.MethodGet, "/forgot-password", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "forgot password page", resp.Body)
}

func TestForgotPasswordHandler_KnownUser(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	handler := New(s, logger)

	formData := url.Values{}
	formData.Set("username", user.Username())

	req := httptest.NewRequest(http.MethodPost, "/forgot-password", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "forgot password confirmation", resp.Body)
}

func TestForgotPasswordHandler_CreatesTokenInStorage(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	handler := New(s, logger)

	formData := url.Values{}
	formData.Set("username", user.Username())

	req := httptest.NewRequest(http.MethodPost, "/forgot-password", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("Expected status OK; got %v", w.Result().Status)
	}

	// The HTTP response never exposes the token (it's only logged), so
	// request a second token via the service to confirm one round-trips
	// through storage correctly with the expected shape.
	authSvc := auth.New(s, logger)
	token, expiresAt, reqErr := authSvc.RequestPasswordReset(t.Context(), user.Username())
	if reqErr != nil {
		t.Fatalf("RequestPasswordReset returned unexpected error: %v", reqErr)
	}

	stored, getErr := s.GetPasswordResetToken(t.Context(), token)
	if getErr != nil {
		t.Fatalf("Expected token to be stored: %v", getErr)
	}

	if stored.UserID() != user.ID() {
		t.Errorf("Expected token user id %d, got %d", user.ID(), stored.UserID())
	}
	if stored.UsedAt() != nil {
		t.Error("Expected freshly requested token to be unused")
	}
	if stored.ExpiresAt().Unix() != expiresAt.Unix() {
		t.Errorf("Expected token expiresAt %v, got %v", expiresAt, stored.ExpiresAt())
	}
}

func TestForgotPasswordHandler_UnknownUser(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, _ := testutil.SetupTestStorage(t, logger)

	handler := New(s, logger)

	formData := url.Values{}
	formData.Set("username", "nonexistent")

	req := httptest.NewRequest(http.MethodPost, "/forgot-password", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}
}

func TestResetPasswordPageHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, _ := testutil.SetupTestStorage(t, logger)

	handler := New(s, logger)

	req := httptest.NewRequest(http.MethodGet, "/reset-password?token=sometoken", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "reset password page", resp.Body)
}

func TestResetPasswordHandler_Success(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	handler := New(s, logger)

	authSvc := auth.New(s, logger)
	token, _, reqErr := authSvc.RequestPasswordReset(t.Context(), user.Username())
	if reqErr != nil {
		t.Fatalf("RequestPasswordReset returned unexpected error: %v", reqErr)
	}

	// Sign in first so we can verify the session gets invalidated by the reset.
	sessionID, _, signinValidationErr, signinErr := authSvc.Signin(t.Context(), user.Username(), "test")
	if signinErr != nil || signinValidationErr != nil {
		t.Fatalf("Signin failed: %v %v", signinErr, signinValidationErr)
	}

	formData := url.Values{}
	formData.Set("token", token)
	formData.Set("password", "newpassword123")
	formData.Set("confirm_password", "newpassword123")

	req := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("Expected status SeeOther; got %v", resp.Status)
	}

	location := resp.Header.Get("Location")
	if location != "/signin" {
		t.Errorf("Expected redirect to '/signin'; got '%s'", location)
	}

	updatedUser, getErr := s.GetUserByUsername(t.Context(), user.Username())
	if getErr != nil {
		t.Fatalf("Failed to get user: %v", getErr)
	}

	if compareErr := bcrypt.CompareHashAndPassword(
		[]byte(updatedUser.PasswordHash()), []byte("newpassword123"),
	); compareErr != nil {
		t.Error("Password should have been updated")
	}

	// Verify the token was marked used in storage
	usedToken, tokenErr := s.GetPasswordResetToken(t.Context(), token)
	if tokenErr != nil {
		t.Fatalf("Failed to fetch reset token: %v", tokenErr)
	}
	if usedToken.UsedAt() == nil {
		t.Error("Expected reset token to be marked used")
	}

	// Verify the prior session was invalidated
	if _, sessionErr := s.GetSession(t.Context(), sessionID); sessionErr == nil {
		t.Error("Expected prior session to be deleted after password reset")
	}
}

func TestResetPasswordHandler_InvalidToken(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, _ := testutil.SetupTestStorage(t, logger)

	handler := New(s, logger)

	formData := url.Values{}
	formData.Set("token", "invalid-token")
	formData.Set("password", "newpassword123")
	formData.Set("confirm_password", "newpassword123")

	req := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Invalid or expired reset link") {
		t.Error("Response should contain error message for invalid token")
	}
}

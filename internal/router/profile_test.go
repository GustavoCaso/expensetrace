package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestProfilePageHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "profile page", resp.Body)
}

func TestUpdateUsernameHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	formData := url.Values{}
	formData.Set("username", "newusername")

	req := httptest.NewRequest(http.MethodPost, "/profile/username", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "update username", resp.Body)

	body := w.Body.String()
	if !strings.Contains(body, "Username updated successfully") {
		t.Error("Response should contain success message")
	}

	updatedUser, err := s.GetUserByID(context.Background(), user.ID())
	if err != nil {
		t.Fatalf("Failed to retrieve updated user: %v", err)
	}

	if updatedUser.Username() != "newusername" {
		t.Errorf("Expected username 'newusername', got '%s'", updatedUser.Username())
	}
}

func TestUpdateUsernameHandlerEmptyUsername(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	formData := url.Values{}
	formData.Set("username", "")

	req := httptest.NewRequest(http.MethodPost, "/profile/username", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	body := w.Body.String()
	if !strings.Contains(body, "username is required") {
		t.Error("Response should contain error message for empty username")
	}
}

func TestUpdateUsernameHandlerDuplicateUsername(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	// Create another user with a different username
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	_, err := s.CreateUser(context.Background(), "existinguser", string(hashedPassword))
	if err != nil {
		t.Fatalf("Failed to create second user: %v", err)
	}

	handler, _ := New(s, logger)

	formData := url.Values{}
	formData.Set("username", "existinguser")

	req := httptest.NewRequest(http.MethodPost, "/profile/username", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	body := w.Body.String()
	if !strings.Contains(body, "username already exists") {
		t.Error("Response should contain error message for duplicate username")
	}
}

func TestUpdateUsernameHandlerFormParseError(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	req := httptest.NewRequest(http.MethodPost, "/profile/username", strings.NewReader("%zzzzz"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "update username form parse error", resp.Body)
}

func TestUpdatePasswordHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	formData := url.Values{}
	formData.Set("current_password", "test")
	formData.Set("new_password", "newpassword123")
	formData.Set("confirm_password", "newpassword123")

	req := httptest.NewRequest(http.MethodPost, "/profile/password", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "update password", resp.Body)

	body := w.Body.String()
	if !strings.Contains(body, "Password changed successfully") {
		t.Error("Response should contain success message")
	}

	updatedUser, err := s.GetUserByID(context.Background(), user.ID())
	if err != nil {
		t.Fatalf("Failed to retrieve updated user: %v", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(updatedUser.PasswordHash()), []byte("newpassword123"))
	if err != nil {
		t.Error("Password should be updated to new password")
	}
}

func TestUpdatePasswordHandlerValidationErrors(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	tests := []struct {
		name          string
		formData      map[string]string
		expectedError string
	}{
		{
			name: "missing current password",
			formData: map[string]string{
				"new_password":     "newpassword123",
				"confirm_password": "newpassword123",
			},
			expectedError: "all fields are required",
		},
		{
			name: "missing new password",
			formData: map[string]string{
				"current_password": "password123",
				"confirm_password": "newpassword123",
			},
			expectedError: "all fields are required",
		},
		{
			name: "missing confirm password",
			formData: map[string]string{
				"current_password": "password123",
				"new_password":     "newpassword123",
			},
			expectedError: "all fields are required",
		},
		{
			name: "passwords do not match",
			formData: map[string]string{
				"current_password": "password123",
				"new_password":     "newpassword123",
				"confirm_password": "differentpassword",
			},
			expectedError: "new passwords do not match",
		},
		{
			name: "password too short",
			formData: map[string]string{
				"current_password": "password123",
				"new_password":     "short",
				"confirm_password": "short",
			},
			expectedError: "password must be at least 8 characters long",
		},
		{
			name: "incorrect current password",
			formData: map[string]string{
				"current_password": "wrongpassword123",
				"new_password":     "newpassword123",
				"confirm_password": "newpassword123",
			},
			expectedError: "current password is incorrect",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formData := url.Values{}
			for key, value := range tt.formData {
				formData.Set(key, value)
			}

			req := httptest.NewRequest(http.MethodPost, "/profile/password", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			resp := w.Result()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status OK for validation error handling; got %v", resp.Status)
			}

			body := w.Body.String()
			if !strings.Contains(body, tt.expectedError) {
				t.Errorf("Expected error message '%s' not found in response", tt.expectedError)
			}

			if strings.Contains(body, "Password changed successfully") {
				t.Error("Should not show success message when there are validation errors")
			}
		})
	}
}

func TestUpdatePasswordHandlerFormParseError(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	req := httptest.NewRequest(http.MethodPost, "/profile/password", strings.NewReader("%zzzzz"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	body := w.Body.String()
	if !strings.Contains(body, "invalid form data") {
		t.Error("Response should contain error message for form parse error")
	}
}

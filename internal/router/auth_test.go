package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestSignupPageHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, _ := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	req := httptest.NewRequest(http.MethodGet, "/signup", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "signup page", resp.Body)
}

func TestSignupHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, _ := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	formData := url.Values{}
	formData.Set("username", "newuser")
	formData.Set("password", "password123")
	formData.Set("confirm_password", "password123")

	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("Expected status SeeOther; got %v", resp.Status)
	}

	location := resp.Header.Get("Location")
	if location != "/" {
		t.Errorf("Expected redirect to '/'; got '%s'", location)
	}

	// Verify user was created
	user, err := s.GetUserByUsername(context.Background(), "newuser")
	if err != nil {
		t.Fatalf("Failed to get created user: %v", err)
	}

	if user.Username() != "newuser" {
		t.Errorf("Expected username 'newuser', got '%s'", user.Username())
	}

	// Verify password is hashed correctly
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash()), []byte("password123"))
	if err != nil {
		t.Error("Password should be hashed correctly")
	}

	// Verify session cookie was set
	cookies := resp.Cookies()
	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == sessionCookieName {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("Expected session cookie to be set")
	}

	if sessionCookie.HttpOnly != true {
		t.Error("Session cookie should be HttpOnly")
	}

	if sessionCookie.SameSite != http.SameSiteStrictMode {
		t.Error("Session cookie should have SameSite=Strict")
	}

	// Verify Exclude category was created
	categories, err := s.GetCategories(context.Background(), user.ID())
	if err != nil {
		t.Fatalf("Failed to get categories: %v", err)
	}

	foundExclude := false
	for _, category := range categories {
		if category.Name() == storage.ExcludeCategory {
			foundExclude = true
			break
		}
	}

	if !foundExclude {
		t.Error("Expected Exclude category to be created for new user")
	}
}

func TestSignupHandlerValidationErrors(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, _ := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	tests := []struct {
		name          string
		formData      map[string]string
		expectedError string
	}{
		{
			name: "missing username",
			formData: map[string]string{
				"password":         "password123",
				"confirm_password": "password123",
			},
			expectedError: "Username and password are required",
		},
		{
			name: "missing password",
			formData: map[string]string{
				"username":         "testuser",
				"confirm_password": "password123",
			},
			expectedError: "Username and password are required",
		},
		{
			name: "passwords do not match",
			formData: map[string]string{
				"username":         "testuser",
				"password":         "password123",
				"confirm_password": "differentpassword",
			},
			expectedError: "Passwords do not match",
		},
		{
			name: "password too short",
			formData: map[string]string{
				"username":         "testuser",
				"password":         "short",
				"confirm_password": "short",
			},
			expectedError: "Password must be at least 8 characters long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formData := url.Values{}
			for key, value := range tt.formData {
				formData.Set(key, value)
			}

			req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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

			// Verify no redirect occurred
			location := resp.Header.Get("Location")
			if location != "" {
				t.Error("Should not redirect when there are validation errors")
			}
		})
	}
}

func TestSignupHandlerDuplicateUsername(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	formData := url.Values{}
	formData.Set("username", user.Username())
	formData.Set("password", "password123")
	formData.Set("confirm_password", "password123")

	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Username already exists or database error occurred") {
		t.Error("Response should contain error message for duplicate username")
	}
}

func TestSignupHandlerFormParseError(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, _ := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader("%zzzzz"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest; got %v", resp.Status)
	}
}

func TestSigninPageHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, _ := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	req := httptest.NewRequest(http.MethodGet, "/signin", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "signin page", resp.Body)
}

func TestSigninHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	formData := url.Values{}
	formData.Set("username", user.Username())
	formData.Set("password", "test")

	req := httptest.NewRequest(http.MethodPost, "/signin", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("Expected status SeeOther; got %v", resp.Status)
	}

	location := resp.Header.Get("Location")
	if location != "/" {
		t.Errorf("Expected redirect to '/'; got '%s'", location)
	}

	// Verify session cookie was set
	cookies := resp.Cookies()
	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == sessionCookieName {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("Expected session cookie to be set")
	}

	if sessionCookie.HttpOnly != true {
		t.Error("Session cookie should be HttpOnly")
	}

	if sessionCookie.SameSite != http.SameSiteStrictMode {
		t.Error("Session cookie should have SameSite=Strict")
	}
}

func TestSigninHandlerValidationErrors(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, _ := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	tests := []struct {
		name          string
		formData      map[string]string
		expectedError string
	}{
		{
			name: "missing username",
			formData: map[string]string{
				"password": "password123",
			},
			expectedError: "Username and password are required",
		},
		{
			name: "missing password",
			formData: map[string]string{
				"username": "testuser",
			},
			expectedError: "Username and password are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formData := url.Values{}
			for key, value := range tt.formData {
				formData.Set(key, value)
			}

			req := httptest.NewRequest(http.MethodPost, "/signin", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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

			// Verify no redirect occurred
			location := resp.Header.Get("Location")
			if location != "" {
				t.Error("Should not redirect when there are validation errors")
			}
		})
	}
}

func TestSigninHandlerInvalidCredentials(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	tests := []struct {
		name     string
		username string
		password string
	}{
		{
			name:     "wrong username",
			username: "nonexistent",
			password: "password123",
		},
		{
			name:     "wrong password",
			username: user.Username(),
			password: "wrongpassword123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formData := url.Values{}
			formData.Set("username", tt.username)
			formData.Set("password", tt.password)

			req := httptest.NewRequest(http.MethodPost, "/signin", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			resp := w.Result()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status OK; got %v", resp.Status)
			}

			body := w.Body.String()
			if !strings.Contains(body, "Invalid username or password") {
				t.Error("Response should contain error message for invalid credentials")
			}

			// Verify no redirect occurred
			location := resp.Header.Get("Location")
			if location != "" {
				t.Error("Should not redirect when credentials are invalid")
			}
		})
	}
}

func TestSigninHandlerFormParseError(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, _ := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	req := httptest.NewRequest(http.MethodPost, "/signin", strings.NewReader("%zzzzz"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest; got %v", resp.Status)
	}
}

func TestSignoutHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	req := httptest.NewRequest(http.MethodPost, "/signout", nil)
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	// Verify HX-Redirect header was set
	hxRedirect := resp.Header.Get("Hx-Redirect")
	if hxRedirect != "/" {
		t.Errorf("Expected Hx-Redirect to '/'; got '%s'", hxRedirect)
	}

	// Verify session cookie was cleared
	cookies := resp.Cookies()
	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == sessionCookieName {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("Expected session cookie to be set for clearing")
	}

	if sessionCookie.Value != "" {
		t.Error("Session cookie value should be empty")
	}

	if sessionCookie.MaxAge != 0 && !sessionCookie.Expires.IsZero() {
		// Cookie should be expired
		if sessionCookie.Expires.Unix() != 0 {
			t.Error("Session cookie should have expiration set to epoch")
		}
	}
}

func TestSignoutHandlerWithoutCookie(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, _ := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	req := httptest.NewRequest(http.MethodPost, "/signout", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	// Without a cookie, the auth middleware will redirect to /signin
	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("Expected status SeeOther; got %v", resp.Status)
	}

	location := resp.Header.Get("Location")
	if location != "/signin" {
		t.Errorf("Expected redirect to '/signin'; got '%s'", location)
	}
}

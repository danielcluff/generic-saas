package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielsaas/generic-saas/internal/database"
)

func setupTestService() (*Service, database.Database) {
	db := database.NewMemoryDatabase()
	service := NewService(db)
	SetService(service) // Set for global handlers
	return service, db
}

func TestService_Register(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Valid registration",
			requestBody: `{
				"name": "John Doe",
				"email": "john@example.com",
				"password": "password123"
			}`,
			expectedStatus: http.StatusCreated,
			expectedError:  "",
		},
		{
			name: "Empty name",
			requestBody: `{
				"name": "",
				"email": "john@example.com",
				"password": "password123"
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Name is required",
		},
		{
			name: "Invalid email",
			requestBody: `{
				"name": "John Doe",
				"email": "invalid-email",
				"password": "password123"
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid email format",
		},
		{
			name: "Short password",
			requestBody: `{
				"name": "John Doe",
				"email": "john@example.com",
				"password": "short"
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Password must be at least 8 characters",
		},
		{
			name:           "Invalid JSON",
			requestBody:    `{invalid json`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, _ := setupTestService()

			req := httptest.NewRequest("POST", "/auth/register", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			service.Register(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedError != "" {
				var response ErrorResponse
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode error response: %v", err)
				}
				if response.Error != tt.expectedError {
					t.Errorf("Expected error '%s', got '%s'", tt.expectedError, response.Error)
				}
			} else {
				var response AuthResponse
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode success response: %v", err)
				}
				if response.User.Email == "" {
					t.Error("Expected user data in response")
				}
			}
		})
	}
}

func TestService_Register_DuplicateEmail(t *testing.T) {
	service, db := setupTestService()

	// Create first user
	user := &database.User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "hashedpassword",
	}
	_, err := db.Users().CreateUser(context.Background(), user)
	if err != nil {
		t.Fatalf("Failed to create first user: %v", err)
	}

	// Try to register with same email
	requestBody := `{
		"name": "Jane Doe",
		"email": "john@example.com",
		"password": "password123"
	}`

	req := httptest.NewRequest("POST", "/auth/register", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	service.Register(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("Expected status %d, got %d", http.StatusConflict, rr.Code)
	}

	var response ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}
	expectedError := "User with this email already exists"
	if response.Error != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, response.Error)
	}
}

func TestService_Login(t *testing.T) {
	service, db := setupTestService()

	// Create a test user with hashed password
	hashedPassword := "$2a$10$5dPLX3zUSjWoaGGf.xz2muh.QGPmYFmKiFmdgiVCMOuuew0MA9AhC" // "password123"
	user := &database.User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: hashedPassword,
	}
	_, err := db.Users().CreateUser(context.Background(), user)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Valid login",
			requestBody: `{
				"email": "john@example.com",
				"password": "password123"
			}`,
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name: "Wrong password",
			requestBody: `{
				"email": "john@example.com",
				"password": "wrongpassword"
			}`,
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid email or password",
		},
		{
			name: "Non-existent user",
			requestBody: `{
				"email": "nonexistent@example.com",
				"password": "password123"
			}`,
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid email or password",
		},
		{
			name: "Empty email",
			requestBody: `{
				"email": "",
				"password": "password123"
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Email and password are required",
		},
		{
			name: "Empty password",
			requestBody: `{
				"email": "john@example.com",
				"password": ""
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Email and password are required",
		},
		{
			name:           "Invalid JSON",
			requestBody:    `{invalid json`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			service.Login(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedError != "" {
				var response ErrorResponse
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode error response: %v", err)
				}
				if response.Error != tt.expectedError {
					t.Errorf("Expected error '%s', got '%s'", tt.expectedError, response.Error)
				}
			} else {
				var response AuthResponse
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode success response: %v", err)
				}
				if response.Token == "" {
					t.Error("Expected token in response")
				}
				if response.User.Email != "john@example.com" {
					t.Errorf("Expected user email 'john@example.com', got '%s'", response.User.Email)
				}
			}
		})
	}
}

func TestHandleLogin_WithGlobalService(t *testing.T) {
	// Test the global handler functions
	_, db := setupTestService()

	// Create a test user with hashed password
	hashedPassword := "$2a$10$5dPLX3zUSjWoaGGf.xz2muh.QGPmYFmKiFmdgiVCMOuuew0MA9AhC" // "password123"
	user := &database.User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: hashedPassword,
	}
	_, err := db.Users().CreateUser(context.Background(), user)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	requestBody := `{
		"email": "john@example.com",
		"password": "password123"
	}`

	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	HandleLogin(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var response AuthResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Token == "" {
		t.Error("Expected token in response")
	}
}

func TestHandleRegister_WithGlobalService(t *testing.T) {
	// Test the global handler functions
	setupTestService()

	requestBody := `{
		"name": "John Doe",
		"email": "john@example.com",
		"password": "password123"
	}`

	req := httptest.NewRequest("POST", "/auth/register", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	HandleRegister(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, rr.Code)
	}

	var response AuthResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.User.Email != "john@example.com" {
		t.Errorf("Expected user email 'john@example.com', got '%s'", response.User.Email)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	setupTestService()

	tests := []struct {
		name    string
		method  string
		handler func(http.ResponseWriter, *http.Request)
	}{
		{"Register GET", "GET", HandleRegister},
		{"Register PUT", "PUT", HandleRegister},
		{"Login GET", "GET", HandleLogin},
		{"Login PUT", "PUT", HandleLogin},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/auth/test", nil)
			rr := httptest.NewRecorder()

			tt.handler(rr, req)

			if rr.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
			}

			var response ErrorResponse
			if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode error response: %v", err)
			}
			if response.Error != "Method not allowed" {
				t.Errorf("Expected error 'Method not allowed', got '%s'", response.Error)
			}
		})
	}
}

func TestValidateRegisterRequest(t *testing.T) {
	tests := []struct {
		name     string
		request  RegisterRequest
		hasError bool
		error    string
	}{
		{
			name: "Valid request",
			request: RegisterRequest{
				Name:     "John Doe",
				Email:    "john@example.com",
				Password: "password123",
			},
			hasError: false,
		},
		{
			name: "Empty name",
			request: RegisterRequest{
				Name:     "",
				Email:    "john@example.com",
				Password: "password123",
			},
			hasError: true,
			error:    "Name is required",
		},
		{
			name: "Short name",
			request: RegisterRequest{
				Name:     "A",
				Email:    "john@example.com",
				Password: "password123",
			},
			hasError: true,
			error:    "Name must be at least 2 characters",
		},
		{
			name: "Invalid email",
			request: RegisterRequest{
				Name:     "John Doe",
				Email:    "invalid-email",
				Password: "password123",
			},
			hasError: true,
			error:    "Invalid email format",
		},
		{
			name: "Short password",
			request: RegisterRequest{
				Name:     "John Doe",
				Email:    "john@example.com",
				Password: "short",
			},
			hasError: true,
			error:    "Password must be at least 8 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRegisterRequest(tt.request)

			if tt.hasError {
				if err == nil {
					t.Error("Expected error, but got none")
				} else if err.Error() != tt.error {
					t.Errorf("Expected error '%s', got '%s'", tt.error, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
			}
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email string
		valid bool
	}{
		{"john@example.com", true},
		{"test+tag@example.co.uk", true},
		{"user.name@example.org", true},
		{"invalid-email", false},
		{"@example.com", false},
		{"user@", false},
		{"user@.com", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := isValidEmail(tt.email)
			if result != tt.valid {
				t.Errorf("Expected isValidEmail('%s') = %v, got %v", tt.email, tt.valid, result)
			}
		})
	}
}
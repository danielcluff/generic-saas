package email

import (
	"errors"
	"strings"
	"testing"
)

func TestNewEmailService(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			errorMsg:    "email config cannot be nil",
		},
		{
			name: "invalid provider",
			config: &Config{
				Provider:  "invalid",
				FromEmail: "test@example.com",
				FromName:  "Test",
			},
			expectError: true,
		},
		{
			name: "missing from email",
			config: &Config{
				Provider: ProviderSMTP,
				FromName: "Test",
			},
			expectError: true,
			errorMsg:    "from email is required",
		},
		{
			name: "missing from name",
			config: &Config{
				Provider:  ProviderSMTP,
				FromEmail: "test@example.com",
			},
			expectError: true,
			errorMsg:    "from name is required",
		},
		{
			name: "invalid from email",
			config: &Config{
				Provider:  ProviderSMTP,
				FromEmail: "invalid-email",
				FromName:  "Test",
			},
			expectError: true,
			errorMsg:    "from email is invalid",
		},
		{
			name: "valid SMTP config",
			config: &Config{
				Provider:  ProviderSMTP,
				FromEmail: "test@example.com",
				FromName:  "Test Service",
			},
			expectError: false,
		},
		{
			name: "SendGrid missing API key",
			config: &Config{
				Provider:  ProviderSendGrid,
				FromEmail: "test@example.com",
				FromName:  "Test Service",
			},
			expectError: true,
			errorMsg:    "SendGrid API key is required",
		},
		{
			name: "SES missing credentials",
			config: &Config{
				Provider:  ProviderSES,
				FromEmail: "test@example.com",
				FromName:  "Test Service",
				SESRegion: "us-east-1",
			},
			expectError: true,
			errorMsg:    "SES credentials are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewEmailService(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if service == nil {
				t.Errorf("expected service to be created")
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &Config{
				FromEmail:        "test@example.com",
				FromName:         "Test Service",
				RateLimitPerHour: 100,
			},
			expectError: false,
		},
		{
			name: "sets default rate limit",
			config: &Config{
				FromEmail: "test@example.com",
				FromName:  "Test Service",
				// RateLimitPerHour not set
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check default rate limit was set
			if tt.config.RateLimitPerHour == 0 {
				if tt.config.RateLimitPerHour != 100 {
					t.Errorf("expected default rate limit to be set to 100, got %d", tt.config.RateLimitPerHour)
				}
			}
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		valid bool
	}{
		{"valid email", "test@example.com", true},
		{"valid email with subdomain", "user@mail.example.com", true},
		{"empty email", "", false},
		{"missing @", "testexample.com", false},
		{"missing domain", "test@", false},
		{"missing local", "@example.com", false},
		{"multiple @", "test@@example.com", false},
		{"localhost domain", "test@localhost", false},
		{"IP address domain", "test@127.0.0.1", false},
		{"private IP", "test@192.168.1.1", false},
		{"private IP range", "test@10.0.0.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidEmail(tt.email)
			if result != tt.valid {
				t.Errorf("isValidEmail(%q) = %v, want %v", tt.email, result, tt.valid)
			}
		})
	}
}

func TestGenerateSecureToken(t *testing.T) {
	// Test multiple generations to ensure randomness
	tokens := make(map[string]bool)

	for i := 0; i < 1000; i++ {
		token, err := generateSecureToken()
		if err != nil {
			t.Fatalf("generateSecureToken() failed: %v", err)
		}

		if len(token) == 0 {
			t.Errorf("generateSecureToken() returned empty token")
		}

		// Check for duplicates (very unlikely with crypto/rand)
		if tokens[token] {
			t.Errorf("generateSecureToken() generated duplicate token: %s", token)
		}
		tokens[token] = true

		// Verify base64 URL encoding
		if len(token) < 40 { // 32 bytes base64 encoded should be longer
			t.Errorf("generateSecureToken() returned suspiciously short token: %s", token)
		}
	}
}

func TestGenerateNumericCode(t *testing.T) {
	tests := []struct {
		name        string
		length      int
		expectError bool
	}{
		{"valid 6 digit", 6, false},
		{"valid 4 digit", 4, false},
		{"valid 8 digit", 8, false},
		{"zero length", 0, true},
		{"negative length", -1, true},
		{"too long", 11, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := generateNumericCode(tt.length)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(code) != tt.length {
				t.Errorf("expected code length %d, got %d", tt.length, len(code))
			}

			// Verify all characters are digits
			for _, char := range code {
				if char < '0' || char > '9' {
					t.Errorf("generateNumericCode() returned non-numeric character: %c", char)
				}
			}
		})
	}

	// Test for randomness
	codes := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		code, err := generateNumericCode(6)
		if err != nil {
			t.Fatalf("generateNumericCode() failed: %v", err)
		}

		if codes[code] {
			// Some duplicates are expected with 6-digit codes, but not many
			t.Logf("generateNumericCode() generated duplicate: %s", code)
		}
		codes[code] = true
	}

	// With 6-digit codes (1M possibilities), we should see good distribution
	if len(codes) < 900 { // Expect high uniqueness in 1000 generations
		t.Errorf("generateNumericCode() showed poor randomness: only %d unique codes in 1000 generations", len(codes))
	}
}

func TestHashToken(t *testing.T) {
	tests := []struct {
		token1 string
		token2 string
		equal  bool
	}{
		{"same", "same", true},
		{"different", "tokens", false},
		{"", "", true},
		{"test", "Test", false}, // Case sensitive
	}

	for _, tt := range tests {
		hash1 := hashToken(tt.token1)
		hash2 := hashToken(tt.token2)

		equal := hash1 == hash2

		if equal != tt.equal {
			t.Errorf("hashToken(%q) == hashToken(%q) = %v, want %v",
				tt.token1, tt.token2, equal, tt.equal)
		}

		// Verify hash length (SHA-256 = 32 bytes)
		if len(hash1) != 32 {
			t.Errorf("hashToken() returned hash of length %d, expected 32", len(hash1))
		}
	}
}

func TestValidateTokenConstantTime(t *testing.T) {
	token1 := "valid_token_123"
	token2 := "invalid_token_456"

	// Test correct validation
	if !validateTokenConstantTime(token1, token1) {
		t.Errorf("validateTokenConstantTime() failed for identical tokens")
	}

	// Test incorrect validation
	if validateTokenConstantTime(token1, token2) {
		t.Errorf("validateTokenConstantTime() should fail for different tokens")
	}

	// Test empty tokens
	if !validateTokenConstantTime("", "") {
		t.Errorf("validateTokenConstantTime() failed for empty tokens")
	}
}

func TestSanitizeEmail(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		expectError bool
	}{
		{"valid email", "test@example.com", false},
		{"email with newline", "test@example.com\n", true},
		{"email with carriage return", "test@example.com\r", true},
		{"email with both", "test@example.com\r\n", true},
		{"newline in middle", "test\n@example.com", true},
		{"empty email", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sanitizeEmail(tt.email)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateEmailAddress(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		expectError bool
		errorType   error
	}{
		{"valid email", "test@example.com", false, nil},
		{"email with newline", "test@example.com\n", true, nil},
		{"invalid format", "invalid-email", true, ErrInvalidEmail},
		{"localhost domain", "test@localhost", true, ErrInvalidEmail},
		{"empty email", "", true, ErrInvalidEmail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEmailAddress(tt.email)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorType != nil && !errors.Is(err, tt.errorType) {
					t.Errorf("expected error type %v, got %v", tt.errorType, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestAddSecurityHeaders(t *testing.T) {
	email := &Email{
		To:      "test@example.com",
		Subject: "Test Subject",
		Body:    "Test Body",
	}

	addSecurityHeaders(email)

	expectedHeaders := map[string]string{
		"X-Priority":               "1",
		"X-Auto-Response-Suppress": "All",
		"Precedence":               "bulk",
		"X-Mailer":                 "SaaSPlatform-Auth",
	}

	if email.Headers == nil {
		t.Fatal("addSecurityHeaders() did not initialize headers map")
	}

	for key, expectedValue := range expectedHeaders {
		if value, exists := email.Headers[key]; !exists {
			t.Errorf("addSecurityHeaders() missing header: %s", key)
		} else if value != expectedValue {
			t.Errorf("addSecurityHeaders() header %s = %q, want %q", key, value, expectedValue)
		}
	}
}

// Test with pre-existing headers
func TestAddSecurityHeadersWithExistingHeaders(t *testing.T) {
	email := &Email{
		To:      "test@example.com",
		Subject: "Test Subject",
		Body:    "Test Body",
		Headers: map[string]string{
			"Custom-Header": "Custom-Value",
		},
	}

	addSecurityHeaders(email)

	// Check custom header is preserved
	if value, exists := email.Headers["Custom-Header"]; !exists {
		t.Errorf("addSecurityHeaders() removed existing header")
	} else if value != "Custom-Value" {
		t.Errorf("addSecurityHeaders() modified existing header value")
	}

	// Check security headers are added
	if value, exists := email.Headers["X-Priority"]; !exists {
		t.Errorf("addSecurityHeaders() did not add X-Priority header")
	} else if value != "1" {
		t.Errorf("addSecurityHeaders() X-Priority = %q, want %q", value, "1")
	}
}
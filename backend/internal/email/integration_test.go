package email

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestEmailServiceIntegration tests the email service integration with real providers
// These tests require actual provider credentials and should be run with integration flags
func TestEmailServiceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	tests := []struct {
		name     string
		provider string
		setup    func() *Config
		skip     bool
	}{
		{
			name:     "SendGrid Integration",
			provider: "sendgrid",
			setup: func() *Config {
				apiKey := os.Getenv("SENDGRID_API_KEY")
				if apiKey == "" {
					return nil
				}
				return &Config{
					Provider:        ProviderSendGrid,
					SendGridAPIKey:  apiKey,
					FromEmail:       os.Getenv("TEST_FROM_EMAIL"),
					FromName:        "SaaSPlatform Test",
					RateLimitPerHour: 10,
				}
			},
		},
		{
			name:     "SES Integration",
			provider: "ses",
			setup: func() *Config {
				accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
				secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
				region := os.Getenv("SES_REGION")
				if accessKey == "" || secretKey == "" || region == "" {
					return nil
				}
				return &Config{
					Provider:           ProviderSES,
					SESAccessKeyID:     accessKey,
					SESSecretAccessKey: secretKey,
					SESRegion:          region,
					FromEmail:          os.Getenv("TEST_FROM_EMAIL"),
					FromName:           "SaaSPlatform Test",
					RateLimitPerHour:   10,
				}
			},
		},
		{
			name:     "SMTP Integration",
			provider: "smtp",
			setup: func() *Config {
				return &Config{
					Provider:         ProviderSMTP,
					FromEmail:        "test@example.com",
					FromName:         "SaaSPlatform Test",
					RateLimitPerHour: 10,
				}
			},
		},
	}

	testEmail := os.Getenv("TEST_EMAIL")
	if testEmail == "" {
		testEmail = "test@example.com"
		t.Logf("Using default test email: %s", testEmail)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.setup()
			if config == nil {
				t.Skipf("Skipping %s test - missing required environment variables", tt.provider)
				return
			}

			service, err := NewEmailService(config)
			if err != nil {
				t.Fatalf("Failed to create email service: %v", err)
			}

			ctx := context.Background()

			// Test welcome email
			t.Run("welcome_email", func(t *testing.T) {
				err := service.SendWelcomeEmail(ctx, testEmail, "Integration Test User")
				if err != nil {
					t.Errorf("SendWelcomeEmail failed: %v", err)
				} else {
					t.Logf("Welcome email sent successfully via %s", tt.provider)
				}
			})

			// Test password reset email
			t.Run("password_reset", func(t *testing.T) {
				securityCtx := SecurityContext{
					RequestIP:   "127.0.0.1",
					UserAgent:   "Integration Test Agent",
					RequestTime: time.Now(),
				}

				err := service.SendPasswordResetCode(ctx, testEmail, "123456", securityCtx)
				if err != nil {
					t.Errorf("SendPasswordResetCode failed: %v", err)
				} else {
					t.Logf("Password reset email sent successfully via %s", tt.provider)
				}
			})

			// Test email verification
			t.Run("email_verification", func(t *testing.T) {
				verificationURL := "https://app.saasplatform.com/verify?token=test-token-123"
				err := service.SendEmailVerification(ctx, testEmail, "Integration Test User", verificationURL)
				if err != nil {
					t.Errorf("SendEmailVerification failed: %v", err)
				} else {
					t.Logf("Email verification sent successfully via %s", tt.provider)
				}
			})

			// Test security alert
			t.Run("security_alert", func(t *testing.T) {
				securityCtx := SecurityContext{
					RequestIP:   "192.168.1.100",
					UserAgent:   "Suspicious Agent",
					RequestTime: time.Now(),
				}

				err := service.SendSecurityAlert(ctx, testEmail, "Integration test security alert", securityCtx)
				if err != nil {
					t.Errorf("SendSecurityAlert failed: %v", err)
				} else {
					t.Logf("Security alert sent successfully via %s", tt.provider)
				}
			})
		})
	}
}

// TestProviderSwitching tests the factory pattern for switching between providers
func TestProviderSwitching(t *testing.T) {
	configs := []*Config{
		{
			Provider:  ProviderSMTP,
			FromEmail: "test@example.com",
			FromName:  "Test SMTP",
		},
		{
			Provider:        ProviderSendGrid,
			SendGridAPIKey:  "test-key",
			FromEmail:       "test@example.com",
			FromName:        "Test SendGrid",
		},
		{
			Provider:           ProviderSES,
			SESAccessKeyID:     "test-access-key",
			SESSecretAccessKey: "test-secret-key",
			SESRegion:          "us-east-1",
			FromEmail:          "test@example.com",
			FromName:           "Test SES",
		},
	}

	expectedProviders := []string{"SMTP", "SendGrid", "Amazon SES"}

	for i, config := range configs {
		service, err := NewEmailService(config)
		if err != nil {
			t.Fatalf("Failed to create service for config %d: %v", i, err)
		}

		if service.GetProviderName() != expectedProviders[i] {
			t.Errorf("Expected provider %s, got %s", expectedProviders[i], service.GetProviderName())
		}
	}
}

// TestRateLimitingIntegration tests rate limiting behavior across providers
func TestRateLimitingIntegration(t *testing.T) {
	// This would require a real database connection to test properly
	// For now, we'll just test the configuration is properly passed through
	config := &Config{
		Provider:         ProviderSMTP,
		FromEmail:        "test@example.com",
		FromName:         "Test Service",
		RateLimitPerHour: 5,
	}

	service, err := NewEmailService(config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Verify service was created successfully
	if service == nil {
		t.Error("Service should not be nil")
	}

	// In a real integration test, we would:
	// 1. Create a token manager with real DB connection
	// 2. Make multiple requests rapidly
	// 3. Verify rate limiting kicks in
	t.Log("Rate limiting integration test would require database connection")
}

// TestErrorHandling tests error scenarios across providers
func TestErrorHandling(t *testing.T) {
	errorCases := []struct {
		name     string
		config   *Config
		expectError bool
	}{
		{
			name: "Invalid SendGrid API Key",
			config: &Config{
				Provider:        ProviderSendGrid,
				SendGridAPIKey:  "",
				FromEmail:       "test@example.com",
				FromName:        "Test",
			},
			expectError: true,
		},
		{
			name: "Invalid SES Region",
			config: &Config{
				Provider:           ProviderSES,
				SESAccessKeyID:     "test",
				SESSecretAccessKey: "test",
				SESRegion:          "",
				FromEmail:          "test@example.com",
				FromName:           "Test",
			},
			expectError: true,
		},
		{
			name: "Invalid Email Format",
			config: &Config{
				Provider:  ProviderSMTP,
				FromEmail: "invalid-email",
				FromName:  "Test",
			},
			expectError: true,
		},
	}

	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewEmailService(tc.config)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestConcurrentEmailSending tests concurrent email sending
func TestConcurrentEmailSending(t *testing.T) {
	config := &Config{
		Provider:  ProviderSMTP,
		FromEmail: "test@example.com",
		FromName:  "Test Service",
	}

	service, err := NewEmailService(config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx := context.Background()
	numGoroutines := 10

	done := make(chan error, numGoroutines)

	// Send emails concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			email := &Email{
				To:      "test@example.com",
				Subject: "Concurrent Test",
				Body:    "Test email body",
			}
			done <- service.SendEmail(ctx, email)
		}(i)
	}

	// Collect results
	errors := 0
	for i := 0; i < numGoroutines; i++ {
		if err := <-done; err != nil {
			errors++
			t.Logf("Concurrent send error: %v", err)
		}
	}

	if errors > 0 {
		t.Logf("Had %d errors out of %d concurrent sends", errors, numGoroutines)
	}
}
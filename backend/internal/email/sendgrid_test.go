package email

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNewSendGridService(t *testing.T) {
	config := &Config{
		SendGridAPIKey: "test-api-key",
		FromEmail:      "test@example.com",
		FromName:       "Test Service",
	}

	service, err := NewSendGridService(config)
	if err != nil {
		t.Fatalf("NewSendGridService() failed: %v", err)
	}

	if service == nil {
		t.Fatal("NewSendGridService() returned nil service")
	}

	if service.fromEmail != config.FromEmail {
		t.Errorf("fromEmail = %q, want %q", service.fromEmail, config.FromEmail)
	}

	if service.fromName != config.FromName {
		t.Errorf("fromName = %q, want %q", service.fromName, config.FromName)
	}

	if service.GetProviderName() != "SendGrid" {
		t.Errorf("GetProviderName() = %q, want %q", service.GetProviderName(), "SendGrid")
	}
}

func TestSendGridServiceSendEmail(t *testing.T) {
	config := &Config{
		SendGridAPIKey: "test-api-key",
		FromEmail:      "test@example.com",
		FromName:       "Test Service",
	}

	service, err := NewSendGridService(config)
	if err != nil {
		t.Fatalf("Failed to create SendGrid service: %v", err)
	}

	tests := []struct {
		name        string
		email       *Email
		expectError bool
	}{
		{
			name: "valid email",
			email: &Email{
				To:      "recipient@example.com",
				Subject: "Test Subject",
				Body:    "<h1>Test Body</h1>",
			},
			expectError: false,
		},
		{
			name: "invalid recipient email",
			email: &Email{
				To:      "invalid-email",
				Subject: "Test Subject",
				Body:    "Test Body",
			},
			expectError: true,
		},
		{
			name: "email with newline in recipient",
			email: &Email{
				To:      "test@example.com\n",
				Subject: "Test Subject",
				Body:    "Test Body",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := service.SendEmail(ctx, tt.email)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			// For valid emails, we expect them to fail due to invalid API key
			// but they should pass validation
			if err != nil && !strings.Contains(err.Error(), "401") && !strings.Contains(err.Error(), "POST") {
				t.Errorf("unexpected error type: %v", err)
			}

			// Verify security headers were added for valid emails
			if !tt.expectError && tt.email.Headers != nil {
				expectedHeaders := []string{"X-Priority", "X-Auto-Response-Suppress", "Precedence", "X-Mailer"}
				for _, header := range expectedHeaders {
					if _, exists := tt.email.Headers[header]; !exists {
						t.Errorf("missing security header: %s", header)
					}
				}
			}
		})
	}
}

func TestSendGridServiceMethods(t *testing.T) {
	config := &Config{
		SendGridAPIKey: "test-api-key",
		FromEmail:      "test@example.com",
		FromName:       "Test Service",
	}

	service, err := NewSendGridService(config)
	if err != nil {
		t.Fatalf("Failed to create SendGrid service: %v", err)
	}

	ctx := context.Background()

	t.Run("SendWelcomeEmail", func(t *testing.T) {
		err := service.SendWelcomeEmail(ctx, "user@example.com", "John Doe")
		// We expect this to fail due to invalid API key, but should pass validation
		if err != nil && !strings.Contains(err.Error(), "401") && !strings.Contains(err.Error(), "POST") {
			t.Errorf("unexpected error type: %v", err)
		}
	})

	t.Run("SendPasswordResetCode", func(t *testing.T) {
		securityCtx := SecurityContext{
			RequestIP:   "192.168.1.100",
			UserAgent:   "Test User Agent",
			RequestTime: time.Now(),
		}

		err := service.SendPasswordResetCode(ctx, "user@example.com", "123456", securityCtx)
		if err != nil && !strings.Contains(err.Error(), "401") && !strings.Contains(err.Error(), "POST") {
			t.Errorf("unexpected error type: %v", err)
		}
	})

	t.Run("SendEmailVerification", func(t *testing.T) {
		err := service.SendEmailVerification(ctx, "user@example.com", "John Doe", "https://example.com/verify?token=abc123")
		if err != nil && !strings.Contains(err.Error(), "401") && !strings.Contains(err.Error(), "POST") {
			t.Errorf("unexpected error type: %v", err)
		}
	})

	t.Run("SendSecurityAlert", func(t *testing.T) {
		securityCtx := SecurityContext{
			RequestIP:   "192.168.1.100",
			UserAgent:   "Test User Agent",
			RequestTime: time.Now(),
		}

		err := service.SendSecurityAlert(ctx, "user@example.com", "Suspicious login detected", securityCtx)
		if err != nil && !strings.Contains(err.Error(), "401") && !strings.Contains(err.Error(), "POST") {
			t.Errorf("unexpected error type: %v", err)
		}
	})
}

func TestSendGridTemplateBuilders(t *testing.T) {
	config := &Config{
		SendGridAPIKey: "test-api-key",
		FromEmail:      "test@example.com",
		FromName:       "Test Service",
	}

	service, err := NewSendGridService(config)
	if err != nil {
		t.Fatalf("Failed to create SendGrid service: %v", err)
	}

	t.Run("welcome template", func(t *testing.T) {
		template := service.buildWelcomeEmailTemplate("John Doe")

		if !strings.Contains(template, "John Doe") {
			t.Error("welcome template should contain user name")
		}
		if !strings.Contains(template, "Welcome to SaaSPlatform") {
			t.Error("welcome template should contain welcome message")
		}
		if !strings.Contains(template, "DOCTYPE html") {
			t.Error("welcome template should be valid HTML")
		}
	})

	t.Run("password reset template", func(t *testing.T) {
		securityCtx := SecurityContext{
			RequestIP:   "192.168.1.100",
			UserAgent:   "Test Agent",
			RequestTime: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		}

		template := service.buildPasswordResetTemplate("123456", securityCtx)

		if !strings.Contains(template, "123456") {
			t.Error("password reset template should contain reset code")
		}
		if !strings.Contains(template, "192.168.1.100") {
			t.Error("password reset template should contain request IP")
		}
		if !strings.Contains(template, "2023-01-01 12:00:00 UTC") {
			t.Error("password reset template should contain formatted time")
		}
	})

	t.Run("email verification template", func(t *testing.T) {
		verificationURL := "https://example.com/verify?token=abc123"
		template := service.buildEmailVerificationTemplate("Jane Doe", verificationURL)

		if !strings.Contains(template, "Jane Doe") {
			t.Error("verification template should contain user name")
		}
		if !strings.Contains(template, verificationURL) {
			t.Error("verification template should contain verification URL")
		}
		// URL should appear twice (button and fallback text)
		count := strings.Count(template, verificationURL)
		if count < 2 {
			t.Errorf("verification URL should appear at least twice, found %d times", count)
		}
	})

	t.Run("security alert template", func(t *testing.T) {
		alertMessage := "Suspicious login detected"
		securityCtx := SecurityContext{
			RequestIP:   "192.168.1.100",
			UserAgent:   "Suspicious Agent",
			RequestTime: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		}

		template := service.buildSecurityAlertTemplate(alertMessage, securityCtx)

		if !strings.Contains(template, alertMessage) {
			t.Error("security alert template should contain alert message")
		}
		if !strings.Contains(template, "192.168.1.100") {
			t.Error("security alert template should contain request IP")
		}
		if !strings.Contains(template, "Suspicious Agent") {
			t.Error("security alert template should contain user agent")
		}
	})
}

func TestSendGridEmailValidation(t *testing.T) {
	config := &Config{
		SendGridAPIKey: "test-api-key",
		FromEmail:      "test@example.com",
		FromName:       "Test Service",
	}

	service, err := NewSendGridService(config)
	if err != nil {
		t.Fatalf("Failed to create SendGrid service: %v", err)
	}

	ctx := context.Background()

	// Test various invalid email addresses
	invalidEmails := []string{
		"invalid",
		"@example.com",
		"test@",
		"test@localhost",
		"test@127.0.0.1",
		"test\n@example.com",
		"test@example.com\n",
	}

	for _, email := range invalidEmails {
		testEmail := &Email{
			To:      email,
			Subject: "Test",
			Body:    "Test",
		}

		err := service.SendEmail(ctx, testEmail)
		if err == nil {
			t.Errorf("SendEmail() should fail for invalid email: %s", email)
		}
	}
}

func TestSendGridHTMLTemplateStructure(t *testing.T) {
	config := &Config{
		SendGridAPIKey: "test-api-key",
		FromEmail:      "test@example.com",
		FromName:       "Test Service",
	}

	service, err := NewSendGridService(config)
	if err != nil {
		t.Fatalf("Failed to create SendGrid service: %v", err)
	}

	templates := []struct {
		name     string
		template string
	}{
		{"welcome", service.buildWelcomeEmailTemplate("Test User")},
		{"password_reset", service.buildPasswordResetTemplate("123456", SecurityContext{
			RequestIP:   "192.168.1.1",
			UserAgent:   "Test Agent",
			RequestTime: time.Now(),
		})},
		{"email_verification", service.buildEmailVerificationTemplate("Test User", "https://example.com/verify")},
		{"security_alert", service.buildSecurityAlertTemplate("Test Alert", SecurityContext{
			RequestIP:   "192.168.1.1",
			UserAgent:   "Test Agent",
			RequestTime: time.Now(),
		})},
	}

	for _, tt := range templates {
		t.Run(tt.name, func(t *testing.T) {
			// Check basic HTML structure
			if !strings.Contains(tt.template, "<!DOCTYPE html>") {
				t.Error("template should have DOCTYPE declaration")
			}
			if !strings.Contains(tt.template, "<html") {
				t.Error("template should have html tag")
			}
			if !strings.Contains(tt.template, "<head>") {
				t.Error("template should have head section")
			}
			if !strings.Contains(tt.template, "<body") {
				t.Error("template should have body tag")
			}
			// Check for inline styles (our templates use inline CSS, not embedded styles)
			if !strings.Contains(tt.template, "style=") {
				t.Error("template should contain inline styles")
			}

			// Check for security and branding (only some templates have branding)
			if tt.name == "welcome" || tt.name == "password_reset" {
				if !strings.Contains(tt.template, "SaaSPlatform") {
					t.Error("template should contain branding")
				}
			}

			// Check that template is not empty or too short
			if len(tt.template) < 1000 {
				t.Errorf("template seems too short (%d chars), might be incomplete", len(tt.template))
			}
		})
	}
}
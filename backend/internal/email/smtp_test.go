package email

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNewSMTPService(t *testing.T) {
	config := &Config{
		FromEmail: "test@example.com",
		FromName:  "Test Service",
	}

	service, err := NewSMTPService(config)
	if err != nil {
		t.Fatalf("NewSMTPService() failed: %v", err)
	}

	if service == nil {
		t.Fatal("NewSMTPService() returned nil service")
	}

	if service.fromEmail != config.FromEmail {
		t.Errorf("fromEmail = %q, want %q", service.fromEmail, config.FromEmail)
	}

	if service.fromName != config.FromName {
		t.Errorf("fromName = %q, want %q", service.fromName, config.FromName)
	}

	if service.GetProviderName() != "SMTP" {
		t.Errorf("GetProviderName() = %q, want %q", service.GetProviderName(), "SMTP")
	}
}

func TestSMTPServiceSendEmail(t *testing.T) {
	service := &SMTPService{
		fromEmail: "test@example.com",
		fromName:  "Test Service",
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

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Verify security headers were added
			if tt.email.Headers == nil {
				t.Error("security headers were not added")
				return
			}

			expectedHeaders := []string{"X-Priority", "X-Auto-Response-Suppress", "Precedence", "X-Mailer"}
			for _, header := range expectedHeaders {
				if _, exists := tt.email.Headers[header]; !exists {
					t.Errorf("missing security header: %s", header)
				}
			}
		})
	}
}

func TestSMTPServiceSendWelcomeEmail(t *testing.T) {
	service := &SMTPService{
		fromEmail: "test@example.com",
		fromName:  "Test Service",
	}

	ctx := context.Background()
	err := service.SendWelcomeEmail(ctx, "user@example.com", "John Doe")

	if err != nil {
		t.Errorf("SendWelcomeEmail() failed: %v", err)
	}
}

func TestSMTPServiceSendPasswordResetCode(t *testing.T) {
	service := &SMTPService{
		fromEmail: "test@example.com",
		fromName:  "Test Service",
	}

	securityCtx := SecurityContext{
		RequestIP:   "192.168.1.100",
		UserAgent:   "Test User Agent",
		RequestTime: time.Now(),
	}

	ctx := context.Background()
	err := service.SendPasswordResetCode(ctx, "user@example.com", "123456", securityCtx)

	if err != nil {
		t.Errorf("SendPasswordResetCode() failed: %v", err)
	}
}

func TestSMTPServiceSendEmailVerification(t *testing.T) {
	service := &SMTPService{
		fromEmail: "test@example.com",
		fromName:  "Test Service",
	}

	ctx := context.Background()
	err := service.SendEmailVerification(ctx, "user@example.com", "John Doe", "https://example.com/verify?token=abc123")

	if err != nil {
		t.Errorf("SendEmailVerification() failed: %v", err)
	}
}

func TestSMTPServiceSendSecurityAlert(t *testing.T) {
	service := &SMTPService{
		fromEmail: "test@example.com",
		fromName:  "Test Service",
	}

	securityCtx := SecurityContext{
		RequestIP:   "192.168.1.100",
		UserAgent:   "Test User Agent",
		RequestTime: time.Now(),
	}

	ctx := context.Background()
	err := service.SendSecurityAlert(ctx, "user@example.com", "Suspicious login detected", securityCtx)

	if err != nil {
		t.Errorf("SendSecurityAlert() failed: %v", err)
	}
}

func TestSMTPBuildMessage(t *testing.T) {
	service := &SMTPService{
		fromEmail: "test@example.com",
		fromName:  "Test Service",
	}

	email := &Email{
		To:      "recipient@example.com",
		Subject: "Test Subject",
		Body:    "<h1>Test HTML Body</h1>",
		Headers: map[string]string{
			"X-Custom": "Custom Value",
		},
	}

	message := service.buildMessage(email)

	// Check required headers are present
	expectedSubstrings := []string{
		"From: Test Service <test@example.com>",
		"To: recipient@example.com",
		"Subject: Test Subject",
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
		"X-Custom: Custom Value",
		"<h1>Test HTML Body</h1>",
	}

	for _, expected := range expectedSubstrings {
		if !strings.Contains(message, expected) {
			t.Errorf("buildMessage() missing expected content: %q", expected)
		}
	}

	// Check proper CRLF line endings in headers
	lines := strings.Split(message, "\n")
	headerEnded := false
	for _, line := range lines {
		if line == "\r" {
			headerEnded = true
			continue
		}
		if !headerEnded && line != "" && !strings.HasSuffix(line, "\r") {
			t.Errorf("buildMessage() header line missing CRLF: %q", line)
		}
	}
}

func TestSMTPTemplateBuilders(t *testing.T) {
	service := &SMTPService{
		fromEmail: "test@example.com",
		fromName:  "Test Service",
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

func TestSMTPEmailValidation(t *testing.T) {
	service := &SMTPService{
		fromEmail: "test@example.com",
		fromName:  "Test Service",
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
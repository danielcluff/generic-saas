package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/danielsaas/generic-saas/internal/config"
)

// SendGridService implements EmailService for SendGrid
type SendGridService struct {
	apiKey    string
	fromEmail string
	fromName  string
	client    *http.Client
}

// SendGrid API structures
type sendGridEmail struct {
	Personalizations []sendGridPersonalization `json:"personalizations"`
	From             sendGridEmailAddress      `json:"from"`
	Subject          string                    `json:"subject"`
	Content          []sendGridContent         `json:"content"`
	Headers          map[string]string         `json:"headers,omitempty"`
}

type sendGridPersonalization struct {
	To []sendGridEmailAddress `json:"to"`
}

type sendGridEmailAddress struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type sendGridContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// NewSendGridService creates a new SendGrid email service
func NewSendGridService(config *Config) (*SendGridService, error) {
	if config.SendGridAPIKey == "" {
		return nil, fmt.Errorf("SendGrid API key is required")
	}

	return &SendGridService{
		apiKey:    config.SendGridAPIKey,
		fromEmail: config.FromEmail,
		fromName:  config.FromName,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// SendEmail sends an email using SendGrid
func (s *SendGridService) SendEmail(ctx context.Context, email *Email) error {
	if err := validateEmailAddress(email.To); err != nil {
		return err
	}

	// Add security headers
	addSecurityHeaders(email)

	// Build SendGrid email structure
	sgEmail := sendGridEmail{
		Personalizations: []sendGridPersonalization{
			{
				To: []sendGridEmailAddress{
					{Email: email.To},
				},
			},
		},
		From: sendGridEmailAddress{
			Email: s.fromEmail,
			Name:  s.fromName,
		},
		Subject: email.Subject,
		Content: []sendGridContent{
			{
				Type:  "text/html",
				Value: email.Body,
			},
		},
		Headers: email.Headers,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(sgEmail)
	if err != nil {
		return fmt.Errorf("failed to marshal email: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.sendgrid.com/v3/mail/send", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("SendGrid API error: status %d", resp.StatusCode)
	}

	return nil
}

// SendWelcomeEmail sends a welcome email
func (s *SendGridService) SendWelcomeEmail(ctx context.Context, to, name string) error {
	subject := config.GetAppConfig().GetWelcomeSubject()
	body := s.buildWelcomeEmailTemplate(name)

	email := &Email{
		To:      to,
		From:    s.fromEmail,
		Subject: subject,
		Body:    body,
	}

	return s.SendEmail(ctx, email)
}

// SendPasswordResetCode sends a password reset code
func (s *SendGridService) SendPasswordResetCode(ctx context.Context, to, code string, securityCtx SecurityContext) error {
	subject := config.GetAppConfig().GetPasswordResetSubject()
	body := s.buildPasswordResetTemplate(code, securityCtx)

	email := &Email{
		To:      to,
		From:    s.fromEmail,
		Subject: subject,
		Body:    body,
	}

	return s.SendEmail(ctx, email)
}

// SendEmailVerification sends an email verification link
func (s *SendGridService) SendEmailVerification(ctx context.Context, to, name, verificationURL string) error {
	subject := config.GetAppConfig().GetVerificationSubject()
	body := s.buildEmailVerificationTemplate(name, verificationURL)

	email := &Email{
		To:      to,
		From:    s.fromEmail,
		Subject: subject,
		Body:    body,
	}

	return s.SendEmail(ctx, email)
}

// SendSecurityAlert sends a security alert notification
func (s *SendGridService) SendSecurityAlert(ctx context.Context, to, alertMessage string, securityCtx SecurityContext) error {
	subject := config.GetAppConfig().GetSecurityAlertSubject()
	body := s.buildSecurityAlertTemplate(alertMessage, securityCtx)

	email := &Email{
		To:      to,
		From:    s.fromEmail,
		Subject: subject,
		Body:    body,
	}

	return s.SendEmail(ctx, email)
}

// GetProviderName returns the provider name
func (s *SendGridService) GetProviderName() string {
	return "SendGrid"
}

// buildWelcomeEmailTemplate creates a welcome email template
func (s *SendGridService) buildWelcomeEmailTemplate(name string) string {
	appConfig := config.GetAppConfig()
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Welcome to %s</title>
</head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="text-align: center; margin-bottom: 30px;">
        <h1 style="color: #1f2937;">Welcome to %s!</h1>
    </div>

    <div style="background: #f9fafb; padding: 20px; border-radius: 8px; margin-bottom: 20px;">
        <h2 style="color: #1f2937; margin-top: 0;">Hello %s,</h2>
        <p style="color: #4b5563; line-height: 1.6;">
            Thank you for joining %s! Your account has been successfully created.
        </p>
        <p style="color: #4b5563; line-height: 1.6;">
            You can now access all the features of your %s dashboard.
        </p>
    </div>

    <div style="text-align: center; margin: 30px 0;">
        <a href="%s"
           style="background: #3b82f6; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; display: inline-block;">
            Go to Dashboard
        </a>
    </div>

    <hr style="border: none; border-top: 1px solid #e5e7eb; margin: 30px 0;">

    <p style="color: #6b7280; font-size: 12px; text-align: center;">
        This email was sent from %s. If you have any questions, please contact our support team.
    </p>
</body>
</html>
`, appConfig.AppDisplayName, appConfig.AppDisplayName, name, appConfig.AppDisplayName, appConfig.AppDisplayName, appConfig.DashboardURL, appConfig.AppDisplayName)
}

// buildPasswordResetTemplate creates a password reset email template
func (s *SendGridService) buildPasswordResetTemplate(code string, securityCtx SecurityContext) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Password Reset Code</title>
</head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="text-align: center; margin-bottom: 30px;">
        <h1 style="color: #1f2937;">Password Reset Request</h1>
    </div>

    <div style="background: #fef3c7; padding: 20px; border-radius: 8px; margin-bottom: 20px; border-left: 4px solid #f59e0b;">
        <h2 style="color: #92400e; margin-top: 0;">Security Code Required</h2>
        <p style="color: #92400e; line-height: 1.6;">
            We received a request to reset your password. Use the code below to continue:
        </p>

        <div style="background: white; padding: 15px; border-radius: 6px; text-align: center; margin: 20px 0;">
            <div style="font-size: 24px; font-weight: bold; color: #1f2937; letter-spacing: 3px; font-family: 'Courier New', monospace;">
                %s
            </div>
        </div>

        <p style="color: #92400e; line-height: 1.6; font-weight: bold;">
            This code expires in 15 minutes.
        </p>
    </div>

    <div style="background: #f3f4f6; padding: 15px; border-radius: 6px; margin-bottom: 20px;">
        <h3 style="color: #374151; margin-top: 0;">Security Information:</h3>
        <ul style="color: #4b5563; margin: 0; padding-left: 20px;">
            <li>Request from IP: %s</li>
            <li>Time: %s</li>
        </ul>
    </div>

    <div style="background: #fef2f2; padding: 15px; border-radius: 6px; border-left: 4px solid #ef4444;">
        <p style="color: #dc2626; margin: 0; line-height: 1.6;">
            <strong>Important:</strong> If you didn't request this password reset, please ignore this email and contact our support team immediately.
        </p>
    </div>

    <hr style="border: none; border-top: 1px solid #e5e7eb; margin: 30px 0;">

    <p style="color: #6b7280; font-size: 12px; text-align: center;">
        This email was sent from SaaSPlatform. Never share your reset code with anyone.
    </p>
</body>
</html>
`, code, securityCtx.RequestIP, securityCtx.RequestTime.Format("2006-01-02 15:04:05 UTC"))
}

// buildEmailVerificationTemplate creates an email verification template
func (s *SendGridService) buildEmailVerificationTemplate(name, verificationURL string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Verify Your Email</title>
</head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="text-align: center; margin-bottom: 30px;">
        <h1 style="color: #1f2937;">Verify Your Email Address</h1>
    </div>

    <div style="background: #f9fafb; padding: 20px; border-radius: 8px; margin-bottom: 20px;">
        <h2 style="color: #1f2937; margin-top: 0;">Hello %s,</h2>
        <p style="color: #4b5563; line-height: 1.6;">
            Please verify your email address to complete your account setup.
        </p>
        <p style="color: #4b5563; line-height: 1.6;">
            Click the button below to verify your email:
        </p>
    </div>

    <div style="text-align: center; margin: 30px 0;">
        <a href="%s"
           style="background: #10b981; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; display: inline-block;">
            Verify Email Address
        </a>
    </div>

    <div style="background: #f3f4f6; padding: 15px; border-radius: 6px;">
        <p style="color: #4b5563; margin: 0; line-height: 1.6; font-size: 14px;">
            If you can't click the button, copy and paste this link into your browser:<br>
            <a href="%s" style="color: #3b82f6; word-break: break-all;">%s</a>
        </p>
    </div>

    <hr style="border: none; border-top: 1px solid #e5e7eb; margin: 30px 0;">

    <p style="color: #6b7280; font-size: 12px; text-align: center;">
        This verification link expires in 48 hours. If you didn't create an account, please ignore this email.
    </p>
</body>
</html>
`, name, verificationURL, verificationURL, verificationURL)
}

// buildSecurityAlertTemplate creates a security alert template
func (s *SendGridService) buildSecurityAlertTemplate(alertMessage string, securityCtx SecurityContext) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Security Alert</title>
</head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="text-align: center; margin-bottom: 30px;">
        <h1 style="color: #dc2626;">🚨 Security Alert</h1>
    </div>

    <div style="background: #fef2f2; padding: 20px; border-radius: 8px; margin-bottom: 20px; border-left: 4px solid #ef4444;">
        <h2 style="color: #dc2626; margin-top: 0;">Important Security Notice</h2>
        <p style="color: #7f1d1d; line-height: 1.6;">
            %s
        </p>
    </div>

    <div style="background: #f3f4f6; padding: 15px; border-radius: 6px; margin-bottom: 20px;">
        <h3 style="color: #374151; margin-top: 0;">Activity Details:</h3>
        <ul style="color: #4b5563; margin: 0; padding-left: 20px;">
            <li>IP Address: %s</li>
            <li>Time: %s</li>
            <li>User Agent: %s</li>
        </ul>
    </div>

    <div style="text-align: center; margin: 30px 0;">
        <a href="https://app.saasplatform.com/settings/security"
           style="background: #dc2626; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; display: inline-block;">
            Review Security Settings
        </a>
    </div>

    <hr style="border: none; border-top: 1px solid #e5e7eb; margin: 30px 0;">

    <p style="color: #6b7280; font-size: 12px; text-align: center;">
        If this activity seems suspicious, please contact our support team immediately.
    </p>
</body>
</html>
`, alertMessage, securityCtx.RequestIP, securityCtx.RequestTime.Format("2006-01-02 15:04:05 UTC"), securityCtx.UserAgent)
}
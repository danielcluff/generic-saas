package email

import (
	"context"
	"fmt"

	"github.com/danielsaas/generic-saas/internal/config"
)

// SESService implements EmailService for Amazon SES
// This is a placeholder implementation - you'd need to add AWS SDK dependencies
type SESService struct {
	region     string
	accessKey  string
	secretKey  string
	fromEmail  string
	fromName   string
}

// NewSESService creates a new SES email service
func NewSESService(config *Config) (*SESService, error) {
	if config.SESRegion == "" {
		return nil, fmt.Errorf("SES region is required")
	}

	if config.SESAccessKeyID == "" || config.SESSecretAccessKey == "" {
		return nil, fmt.Errorf("SES credentials are required")
	}

	return &SESService{
		region:    config.SESRegion,
		accessKey: config.SESAccessKeyID,
		secretKey: config.SESSecretAccessKey,
		fromEmail: config.FromEmail,
		fromName:  config.FromName,
	}, nil
}

// SendEmail sends an email using Amazon SES
func (s *SESService) SendEmail(ctx context.Context, email *Email) error {
	if err := validateEmailAddress(email.To); err != nil {
		return err
	}

	// Add security headers
	addSecurityHeaders(email)

	// For now, this is a placeholder that logs the email
	// In production, you would use the AWS SES SDK
	fmt.Printf("SES Email Service - Would send email:\n")
	fmt.Printf("Region: %s\n", s.region)
	fmt.Printf("To: %s\n", email.To)
	fmt.Printf("From: %s <%s>\n", s.fromName, s.fromEmail)
	fmt.Printf("Subject: %s\n", email.Subject)
	fmt.Printf("Body: %s\n", email.Body)
	fmt.Printf("---\n")

	return nil
}

// SendWelcomeEmail sends a welcome email
func (s *SESService) SendWelcomeEmail(ctx context.Context, to, name string) error {
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
func (s *SESService) SendPasswordResetCode(ctx context.Context, to, code string, securityCtx SecurityContext) error {
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
func (s *SESService) SendEmailVerification(ctx context.Context, to, name, verificationURL string) error {
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
func (s *SESService) SendSecurityAlert(ctx context.Context, to, alertMessage string, securityCtx SecurityContext) error {
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
func (s *SESService) GetProviderName() string {
	return "Amazon SES"
}

// Template builders (simplified versions for SES)
func (s *SESService) buildWelcomeEmailTemplate(name string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Welcome to SaaSPlatform</title>
</head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <h1>Welcome to SaaSPlatform!</h1>
    <p>Hello %s,</p>
    <p>Thank you for joining SaaSPlatform! Your account has been successfully created.</p>
    <p><a href="https://app.saasplatform.com/dashboard">Go to Dashboard</a></p>
</body>
</html>
`, name)
}

func (s *SESService) buildPasswordResetTemplate(code string, securityCtx SecurityContext) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Password Reset Code</title>
</head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <h1>Password Reset Request</h1>
    <p>Use this code to reset your password:</p>
    <div style="font-size: 24px; font-weight: bold; padding: 15px; background: #f0f0f0; text-align: center;">
        %s
    </div>
    <p>This code expires in 15 minutes.</p>
    <p><small>Request from IP: %s at %s</small></p>
</body>
</html>
`, code, securityCtx.RequestIP, securityCtx.RequestTime.Format("2006-01-02 15:04:05 UTC"))
}

func (s *SESService) buildEmailVerificationTemplate(name, verificationURL string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Verify Your Email</title>
</head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <h1>Verify Your Email Address</h1>
    <p>Hello %s,</p>
    <p>Please verify your email address by clicking the link below:</p>
    <p><a href="%s" style="display: inline-block; padding: 10px 20px; background: #007bff; color: white; text-decoration: none; border-radius: 5px;">Verify Email</a></p>
    <p>If you can't click the button, copy and paste this link: %s</p>
</body>
</html>
`, name, verificationURL, verificationURL)
}

func (s *SESService) buildSecurityAlertTemplate(alertMessage string, securityCtx SecurityContext) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Security Alert</title>
</head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <h1 style="color: red;">🚨 Security Alert</h1>
    <p><strong>Alert:</strong> %s</p>
    <div style="background: #f8f9fa; padding: 15px; border-left: 4px solid #dc3545;">
        <h3>Activity Details:</h3>
        <ul>
            <li>IP Address: %s</li>
            <li>Time: %s</li>
            <li>User Agent: %s</li>
        </ul>
    </div>
    <p>If this wasn't you, please contact support immediately.</p>
</body>
</html>
`, alertMessage, securityCtx.RequestIP, securityCtx.RequestTime.Format("2006-01-02 15:04:05 UTC"), securityCtx.UserAgent)
}

// Note: To implement full SES functionality, you would need to:
// 1. Add AWS SDK dependency: go get github.com/aws/aws-sdk-go-v2/service/sesv2
// 2. Implement proper SES client initialization
// 3. Handle SES-specific features like bounce/complaint handling
// 4. Implement proper error handling for SES API responses
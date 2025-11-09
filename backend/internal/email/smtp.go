package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"github.com/danielsaas/generic-saas/internal/config"
)

// SMTPService implements EmailService for SMTP
type SMTPService struct {
	host       string
	port       string
	username   string
	password   string
	fromEmail  string
	fromName   string
	requireTLS bool
}

// NewSMTPService creates a new SMTP email service
func NewSMTPService(config *Config) (*SMTPService, error) {
	// For SMTP, we'll use environment variables or config
	// This is a basic implementation - in production you'd want more config options
	return &SMTPService{
		host:       "localhost", // Default for development
		port:       "587",
		fromEmail:  config.FromEmail,
		fromName:   config.FromName,
		requireTLS: config.RequireTLS,
	}, nil
}

// SendEmail sends an email using SMTP
func (s *SMTPService) SendEmail(ctx context.Context, email *Email) error {
	if err := validateEmailAddress(email.To); err != nil {
		return err
	}

	// Add security headers
	addSecurityHeaders(email)

	// Create the message
	message := s.buildMessage(email)

	// For development/testing, we'll just log the email
	// In production, you'd implement actual SMTP sending
	fmt.Printf("SMTP Email Service - Would send email:\n")
	fmt.Printf("To: %s\n", email.To)
	fmt.Printf("Subject: %s\n", email.Subject)
	fmt.Printf("Message:\n%s\n", message)
	fmt.Printf("---\n")

	return nil
}

// buildMessage builds the email message with proper headers
func (s *SMTPService) buildMessage(email *Email) string {
	var message strings.Builder

	// Standard headers
	message.WriteString(fmt.Sprintf("From: %s <%s>\r\n", s.fromName, s.fromEmail))
	message.WriteString(fmt.Sprintf("To: %s\r\n", email.To))
	message.WriteString(fmt.Sprintf("Subject: %s\r\n", email.Subject))
	message.WriteString("MIME-Version: 1.0\r\n")
	message.WriteString("Content-Type: text/html; charset=UTF-8\r\n")

	// Add custom headers
	for key, value := range email.Headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}

	message.WriteString("\r\n")
	message.WriteString(email.Body)

	return message.String()
}

// SendWelcomeEmail sends a welcome email
func (s *SMTPService) SendWelcomeEmail(ctx context.Context, to, name string) error {
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
func (s *SMTPService) SendPasswordResetCode(ctx context.Context, to, code string, securityCtx SecurityContext) error {
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
func (s *SMTPService) SendEmailVerification(ctx context.Context, to, name, verificationURL string) error {
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
func (s *SMTPService) SendSecurityAlert(ctx context.Context, to, alertMessage string, securityCtx SecurityContext) error {
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
func (s *SMTPService) GetProviderName() string {
	return "SMTP"
}

// Template builders (reusing SendGrid templates for consistency)
func (s *SMTPService) buildWelcomeEmailTemplate(name string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Welcome to SaaSPlatform</title>
</head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="text-align: center; margin-bottom: 30px;">
        <h1 style="color: #1f2937;">Welcome to SaaSPlatform!</h1>
    </div>

    <div style="background: #f9fafb; padding: 20px; border-radius: 8px; margin-bottom: 20px;">
        <h2 style="color: #1f2937; margin-top: 0;">Hello %s,</h2>
        <p style="color: #4b5563; line-height: 1.6;">
            Thank you for joining SaaSPlatform! Your account has been successfully created.
        </p>
    </div>

    <hr style="border: none; border-top: 1px solid #e5e7eb; margin: 30px 0;">

    <p style="color: #6b7280; font-size: 12px; text-align: center;">
        This email was sent from SaaSPlatform.
    </p>
</body>
</html>
`, name)
}

func (s *SMTPService) buildPasswordResetTemplate(code string, securityCtx SecurityContext) string {
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

    <div style="background: #fef3c7; padding: 20px; border-radius: 8px; margin-bottom: 20px;">
        <h2 style="color: #92400e; margin-top: 0;">Security Code Required</h2>
        <p style="color: #92400e; line-height: 1.6;">
            Use this code to reset your password:
        </p>

        <div style="background: white; padding: 15px; border-radius: 6px; text-align: center; margin: 20px 0;">
            <div style="font-size: 24px; font-weight: bold; color: #1f2937; letter-spacing: 3px;">
                %s
            </div>
        </div>

        <p style="color: #92400e; line-height: 1.6;">
            This code expires in 15 minutes. Request from IP: %s at %s
        </p>
    </div>
</body>
</html>
`, code, securityCtx.RequestIP, securityCtx.RequestTime.Format("2006-01-02 15:04:05 UTC"))
}

func (s *SMTPService) buildEmailVerificationTemplate(name, verificationURL string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Verify Your Email</title>
</head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <h1>Verify Your Email Address</h1>
    <p>Hello %s, please verify your email by clicking this link:</p>
    <p><a href="%s">Verify Email</a></p>
    <p>If you can't click the link, copy and paste: %s</p>
</body>
</html>
`, name, verificationURL, verificationURL)
}

func (s *SMTPService) buildSecurityAlertTemplate(alertMessage string, securityCtx SecurityContext) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Security Alert</title>
</head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <h1>Security Alert</h1>
    <p><strong>Alert:</strong> %s</p>
    <p><strong>Details:</strong></p>
    <ul>
        <li>IP: %s</li>
        <li>Time: %s</li>
        <li>User Agent: %s</li>
    </ul>
</body>
</html>
`, alertMessage, securityCtx.RequestIP, securityCtx.RequestTime.Format("2006-01-02 15:04:05 UTC"), securityCtx.UserAgent)
}

// actualSendSMTP would implement real SMTP sending in production
func (s *SMTPService) actualSendSMTP(to string, message string) error {
	// This would be the real SMTP implementation
	auth := smtp.PlainAuth("", s.username, s.password, s.host)

	// For TLS connection
	if s.requireTLS {
		tlsConfig := &tls.Config{
			ServerName: s.host,
		}

		addr := net.JoinHostPort(s.host, s.port)
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return err
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, s.host)
		if err != nil {
			return err
		}
		defer client.Quit()

		if err := client.Auth(auth); err != nil {
			return err
		}

		if err := client.Mail(s.fromEmail); err != nil {
			return err
		}

		if err := client.Rcpt(to); err != nil {
			return err
		}

		w, err := client.Data()
		if err != nil {
			return err
		}

		_, err = w.Write([]byte(message))
		if err != nil {
			return err
		}

		return w.Close()
	}

	// For non-TLS (development only)
	addr := net.JoinHostPort(s.host, s.port)
	return smtp.SendMail(addr, auth, s.fromEmail, []string{to}, []byte(message))
}
package email

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/danielsaas/generic-saas/internal/config"
)

// Email represents an email message
type Email struct {
	To      string
	From    string
	Subject string
	Body    string
	Headers map[string]string
}

// SecurityContext provides request context for security logging
type SecurityContext struct {
	RequestIP   string
	UserAgent   string
	RequestTime time.Time
}

// EmailService defines the interface for sending emails
type EmailService interface {
	// Core email sending
	SendEmail(ctx context.Context, email *Email) error

	// Authentication emails
	SendWelcomeEmail(ctx context.Context, to, name string) error
	SendPasswordResetCode(ctx context.Context, to, code string, securityCtx SecurityContext) error
	SendEmailVerification(ctx context.Context, to, name, verificationURL string) error

	// Security notifications
	SendSecurityAlert(ctx context.Context, to, alertMessage string, securityCtx SecurityContext) error

	// Provider info
	GetProviderName() string
}

// Config holds email service configuration
type Config struct {
	Provider string

	// SendGrid config
	SendGridAPIKey string

	// SES config
	SESRegion          string
	SESAccessKeyID     string
	SESSecretAccessKey string

	// Common config
	FromEmail string
	FromName  string

	// Rate limiting
	RateLimitPerHour int

	// Security
	RequireTLS bool
}

// Provider types
const (
	ProviderSendGrid = "sendgrid"
	ProviderSES      = "ses"
	ProviderSMTP     = "smtp"
)

// Errors
var (
	ErrInvalidProvider    = errors.New("invalid email provider")
	ErrInvalidEmail       = errors.New("invalid email address")
	ErrRateLimitExceeded  = errors.New("rate limit exceeded")
	ErrTokenExpired       = errors.New("token has expired")
	ErrTokenAlreadyUsed   = errors.New("token has already been used")
	ErrInvalidToken       = errors.New("invalid token")
)

// NewEmailService creates a new email service based on configuration
func NewEmailService(config *Config) (EmailService, error) {
	if config == nil {
		return nil, errors.New("email config cannot be nil")
	}

	// Validate common config
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid email config: %w", err)
	}

	switch strings.ToLower(config.Provider) {
	case ProviderSendGrid:
		return NewSendGridService(config)
	case ProviderSES:
		return NewSESService(config)
	case ProviderSMTP:
		return NewSMTPService(config)
	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidProvider, config.Provider)
	}
}

// validateConfig validates email configuration
func validateConfig(config *Config) error {
	if config.FromEmail == "" {
		return errors.New("from email is required")
	}

	if config.FromName == "" {
		return errors.New("from name is required")
	}

	if !isValidEmail(config.FromEmail) {
		return errors.New("from email is invalid")
	}

	if config.RateLimitPerHour <= 0 {
		config.RateLimitPerHour = 100 // Default rate limit
	}

	return nil
}

// isValidEmail validates email format and prevents dangerous domains
func isValidEmail(email string) bool {
	if email == "" {
		return false
	}

	// Basic format validation
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	local, domain := parts[0], parts[1]
	if local == "" || domain == "" {
		return false
	}

	// Prevent dangerous domains in production
	blockedDomains := []string{
		"localhost", "127.0.0.1", "0.0.0.0",
		"10.", "192.168.", "172.", // Private IPs
	}

	for _, blocked := range blockedDomains {
		if strings.Contains(domain, blocked) {
			return false
		}
	}

	return true
}

// generateSecureToken generates a cryptographically secure token
func generateSecureToken() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// generateNumericCode generates a secure numeric code
func generateNumericCode(length int) (string, error) {
	if length <= 0 || length > 10 {
		return "", errors.New("code length must be between 1 and 10")
	}

	max := big.NewInt(1)
	for i := 0; i < length; i++ {
		max.Mul(max, big.NewInt(10))
	}

	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("failed to generate numeric code: %w", err)
	}

	// Format with leading zeros if necessary
	format := fmt.Sprintf("%%0%dd", length)
	return fmt.Sprintf(format, n), nil
}

// hashToken creates a SHA-256 hash of a token
func hashToken(token string) [32]byte {
	return sha256.Sum256([]byte(token))
}

// validateTokenConstantTime performs constant-time token validation
func validateTokenConstantTime(provided, stored string) bool {
	providedHash := hashToken(provided)
	storedHash := hashToken(stored)
	return subtle.ConstantTimeCompare(providedHash[:], storedHash[:]) == 1
}

// addSecurityHeaders adds security headers to email
func addSecurityHeaders(email *Email) {
	if email.Headers == nil {
		email.Headers = make(map[string]string)
	}

	email.Headers["X-Priority"] = "1"                      // High priority for auth emails
	email.Headers["X-Auto-Response-Suppress"] = "All"     // Prevent auto-replies
	email.Headers["Precedence"] = "bulk"                  // Prevent out-of-office replies
	email.Headers["X-Mailer"] = config.GetAppConfig().EmailMailer // Identify our emails
}

// sanitizeEmail prevents email header injection
func sanitizeEmail(email string) error {
	if strings.Contains(email, "\n") || strings.Contains(email, "\r") {
		return errors.New("invalid email format: contains newline characters")
	}
	return nil
}

// validateEmailAddress performs comprehensive email validation
func validateEmailAddress(email string) error {
	if err := sanitizeEmail(email); err != nil {
		return err
	}

	if !isValidEmail(email) {
		return ErrInvalidEmail
	}

	return nil
}
# Email Service Package

A comprehensive, secure email service implementation for SaaS platforms with factory pattern for easy provider switching.

## Features

### 🔒 **Security First**
- Cryptographically secure token generation (256-bit entropy)
- Constant-time token validation to prevent timing attacks
- Email header injection prevention
- Rate limiting with configurable thresholds
- Secure token storage with SHA-256 hashing
- IP and User-Agent logging for security audit

### 📧 **Multiple Providers**
- **SendGrid**: Production-ready with rich templates
- **Amazon SES**: Cost-effective for high volume
- **SMTP**: Custom servers and development

### 🎯 **Email Types**
- Welcome emails for new users
- Password reset with secure 6-digit codes
- Email verification with secure links
- Security alert notifications

### 🏭 **Factory Pattern**
- Easy provider switching without code changes
- Consistent interface across all providers
- Environment-based configuration

## Quick Start

### 1. Install Dependencies

```bash
# For SendGrid (optional)
go get github.com/sendgrid/sendgrid-go

# For AWS SES (optional)
go get github.com/aws/aws-sdk-go-v2/service/sesv2
```

### 2. Database Setup

```sql
-- Run the migration
\i internal/email/migrations.sql
```

### 3. Basic Usage

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/danielsaas/generic-saas/internal/email"
)

func main() {
    // Configure email service
    config := &email.Config{
        Provider:       email.ProviderSendGrid,
        SendGridAPIKey: os.Getenv("SENDGRID_API_KEY"),
        FromEmail:      "noreply@yourapp.com",
        FromName:       "Your App",
        RateLimitPerHour: 100,
    }

    // Create service using factory
    emailService, err := email.NewEmailService(config)
    if err != nil {
        log.Fatal(err)
    }

    // Send welcome email
    err = emailService.SendWelcomeEmail(
        context.Background(),
        "user@example.com",
        "John Doe",
    )
    if err != nil {
        log.Printf("Failed to send email: %v", err)
    }
}
```

## Authentication Flows

### Password Reset Flow

```go
// Initialize token manager
tokenManager := email.NewTokenManager(db, emailService)

// Step 1: Request password reset
resetRequest := email.PasswordResetRequest{
    Email:     "user@example.com",
    RequestIP: "192.168.1.100",
    UserAgent: "Mozilla/5.0...",
}

err := tokenManager.RequestPasswordReset(resetRequest)
if err != nil {
    // Handle error (rate limiting, invalid email, etc.)
}

// Step 2: Verify user-provided code
token, err := tokenManager.VerifyPasswordResetCode("user@example.com", "123456")
if err != nil {
    // Handle error (invalid/expired code)
}

// Now allow user to set new password
// token.UserID contains the verified user ID
```

### Email Verification Flow

```go
// Step 1: Request email verification
verificationRequest := email.EmailVerificationRequest{
    UserID:    123,
    Email:     "newuser@example.com",
    Name:      "Jane Doe",
    RequestIP: "192.168.1.101",
    UserAgent: "Mozilla/5.0...",
}

err := tokenManager.RequestEmailVerification(verificationRequest)

// Step 2: Verify token from email link
token, err := tokenManager.VerifyEmailToken("token_from_url")
if err != nil {
    // Handle error
}

// Email is now verified for token.UserID
```

## Configuration

### Environment Variables

```bash
# SendGrid
SENDGRID_API_KEY=your_sendgrid_api_key

# Amazon SES
AWS_ACCESS_KEY_ID=your_aws_access_key
AWS_SECRET_ACCESS_KEY=your_aws_secret
SES_REGION=us-east-1

# App Configuration
FROM_EMAIL=noreply@yourapp.com
FROM_NAME="Your App Name"
```

### Provider Switching

```go
// Switch from SendGrid to SES with just config change
config := &email.Config{
    Provider: email.ProviderSES, // Changed this line only
    SESRegion: "us-east-1",
    SESAccessKeyID: os.Getenv("AWS_ACCESS_KEY_ID"),
    SESSecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
    FromEmail: "noreply@yourapp.com",
    FromName: "Your App",
}

emailService, err := email.NewEmailService(config)
// All methods work exactly the same!
```

## Security Features

### Token Security
- **Secure Generation**: Uses `crypto/rand` for cryptographically secure tokens
- **Hashed Storage**: Tokens are SHA-256 hashed before database storage
- **Single Use**: Tokens are marked as used after consumption
- **Time-Limited**: Configurable expiration times (15 min for password reset, 48h for verification)
- **Constant-Time Validation**: Prevents timing attack vectors

### Rate Limiting
- **Email-Based Limiting**: 3 requests per hour per email address
- **Type-Specific**: Different limits for different token types
- **IP Logging**: All requests logged with IP and User-Agent

### Email Security
- **Header Injection Prevention**: Validates email addresses for newline characters
- **Domain Validation**: Blocks internal/localhost domains in production
- **Security Headers**: Adds appropriate headers to prevent auto-replies
- **TLS Enforcement**: Optional TLS requirement for SMTP

## Templates

### Template Features
- **Mobile Responsive**: Works on all devices
- **Security Information**: Includes IP and timestamp for transparency
- **Clear CTAs**: Prominent buttons and codes
- **Professional Design**: Consistent branding and styling

### Customizing Templates
Templates are defined in each provider implementation. To customize:

1. Modify the template functions in `sendgrid.go`, `ses.go`, etc.
2. Add your own branding, colors, and messaging
3. Ensure security information is preserved

## Monitoring & Maintenance

### Token Cleanup
```go
// Run periodically (e.g., daily cron job)
err := tokenManager.CleanupExpiredTokens()
```

### Statistics
```go
stats, err := tokenManager.GetTokenStats()
// Returns active tokens by type, expired tokens count, etc.
```

### Database Function
```sql
-- Manual cleanup
SELECT cleanup_expired_email_tokens();

-- Schedule with pg_cron (if available)
SELECT cron.schedule('cleanup-email-tokens', '0 2 * * *',
    'SELECT cleanup_expired_email_tokens();');
```

## Production Considerations

### SendGrid Setup
1. Get API key from SendGrid dashboard
2. Set up domain authentication (SPF/DKIM)
3. Configure dedicated IP (for high volume)

### Amazon SES Setup
1. Move out of sandbox mode
2. Set up domain verification
3. Configure bounce/complaint handling
4. Set up dedicated IP pool (optional)

### Rate Limiting
- Adjust `RateLimitPerHour` based on your needs
- Monitor for abuse patterns
- Consider implementing CAPTCHA for high-risk scenarios

### Error Handling
```go
switch {
case errors.Is(err, email.ErrRateLimitExceeded):
    // Show user-friendly rate limit message
case errors.Is(err, email.ErrInvalidEmail):
    // Show email format error
case errors.Is(err, email.ErrTokenExpired):
    // Offer to resend token
case errors.Is(err, email.ErrTokenAlreadyUsed):
    // Show token already used message
}
```

## Testing

For development and testing, use the SMTP provider which logs emails instead of sending them:

```go
config := &email.Config{
    Provider:  email.ProviderSMTP,
    FromEmail: "test@yourapp.com",
    FromName:  "Your App (Test)",
}
```

This implementation provides enterprise-grade email functionality with security best practices built-in from day one.
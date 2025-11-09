package email

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"time"
)

// ExampleEmailServiceUsage demonstrates how to use the email service
func ExampleEmailServiceUsage() {
	// Create email service configuration
	config := &Config{
		Provider:       ProviderSendGrid, // or ProviderSES, ProviderSMTP
		SendGridAPIKey: os.Getenv("SENDGRID_API_KEY"),
		FromEmail:      "noreply@saasplatform.com",
		FromName:       "SaaSPlatform",
		RateLimitPerHour: 100,
		RequireTLS:     true,
	}

	// Create the email service using the factory
	emailService, err := NewEmailService(config)
	if err != nil {
		log.Fatalf("Failed to create email service: %v", err)
	}

	log.Printf("Using email provider: %s", emailService.GetProviderName())

	// Example: Send a welcome email
	err = emailService.SendWelcomeEmail(context.Background(), "user@example.com", "John Doe")
	if err != nil {
		log.Printf("Failed to send welcome email: %v", err)
	} else {
		log.Println("Welcome email sent successfully")
	}

	// Example: Password reset flow
	securityCtx := SecurityContext{
		RequestIP:   "192.168.1.100",
		UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		RequestTime: time.Now(),
	}

	err = emailService.SendPasswordResetCode(context.Background(), "user@example.com", "123456", securityCtx)
	if err != nil {
		log.Printf("Failed to send password reset email: %v", err)
	} else {
		log.Println("Password reset email sent successfully")
	}
}

// ExampleTokenManagerUsage demonstrates how to use the token manager
func ExampleTokenManagerUsage(db *sql.DB, emailService EmailService) {
	// Create token manager
	tokenManager := NewTokenManager(db, emailService)

	// Example: Password reset flow
	resetRequest := PasswordResetRequest{
		Email:     "user@example.com",
		RequestIP: "192.168.1.100",
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
	}

	// Step 1: Request password reset (sends email with code)
	err := tokenManager.RequestPasswordReset(resetRequest)
	if err != nil {
		log.Printf("Failed to request password reset: %v", err)
		return
	}

	log.Println("Password reset code sent to user's email")

	// Step 2: User provides the code, verify it
	// In a real application, this code would come from user input
	userProvidedCode := "123456"

	token, err := tokenManager.VerifyPasswordResetCode("user@example.com", userProvidedCode)
	if err != nil {
		log.Printf("Failed to verify reset code: %v", err)
		return
	}

	log.Printf("Password reset code verified for user ID: %d", token.UserID)
	// Now you can proceed to allow the user to set a new password

	// Example: Email verification flow
	verificationRequest := EmailVerificationRequest{
		UserID:    123,
		Email:     "newuser@example.com",
		Name:      "Jane Doe",
		RequestIP: "192.168.1.101",
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
	}

	// Step 1: Request email verification (sends email with link)
	err = tokenManager.RequestEmailVerification(verificationRequest)
	if err != nil {
		log.Printf("Failed to request email verification: %v", err)
		return
	}

	log.Println("Email verification link sent to user's email")

	// Step 2: User clicks the link, verify the token
	// In a real application, this token would come from the URL parameter
	userClickedToken := "abc123def456..."

	verificationToken, err := tokenManager.VerifyEmailToken(userClickedToken)
	if err != nil {
		log.Printf("Failed to verify email token: %v", err)
		return
	}

	log.Printf("Email verified for user ID: %d", verificationToken.UserID)

	// Cleanup expired tokens (run this periodically)
	err = tokenManager.CleanupExpiredTokens()
	if err != nil {
		log.Printf("Failed to cleanup expired tokens: %v", err)
	} else {
		log.Println("Expired tokens cleaned up successfully")
	}

	// Get token statistics for monitoring
	stats, err := tokenManager.GetTokenStats()
	if err != nil {
		log.Printf("Failed to get token stats: %v", err)
	} else {
		log.Printf("Token statistics: %+v", stats)
	}
}

// ExampleSwitchingProviders demonstrates how easy it is to switch email providers
func ExampleSwitchingProviders() {
	// Start with SendGrid
	sendgridConfig := &Config{
		Provider:         ProviderSendGrid,
		SendGridAPIKey:   os.Getenv("SENDGRID_API_KEY"),
		FromEmail:        "noreply@saasplatform.com",
		FromName:         "SaaSPlatform",
		RateLimitPerHour: 100,
	}

	emailService, err := NewEmailService(sendgridConfig)
	if err != nil {
		log.Fatalf("Failed to create SendGrid service: %v", err)
	}

	log.Printf("Using provider: %s", emailService.GetProviderName())

	// Later, switch to Amazon SES with just configuration change
	sesConfig := &Config{
		Provider:           ProviderSES,
		SESRegion:          "us-east-1",
		SESAccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		SESSecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		FromEmail:          "noreply@saasplatform.com",
		FromName:           "SaaSPlatform",
		RateLimitPerHour:   1000, // SES might have higher limits
	}

	emailService, err = NewEmailService(sesConfig)
	if err != nil {
		log.Fatalf("Failed to create SES service: %v", err)
	}

	log.Printf("Switched to provider: %s", emailService.GetProviderName())

	// All the same methods work regardless of provider
	err = emailService.SendWelcomeEmail(context.Background(), "user@example.com", "John Doe")
	if err != nil {
		log.Printf("Failed to send email: %v", err)
	}
}

// ExampleErrorHandling demonstrates proper error handling
func ExampleErrorHandling() {
	config := &Config{
		Provider:  ProviderSendGrid,
		FromEmail: "invalid-email", // Intentionally invalid
		FromName:  "SaaSPlatform",
	}

	// This will fail validation
	_, err := NewEmailService(config)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidProvider):
			log.Println("Invalid email provider specified")
		default:
			log.Printf("Email service configuration error: %v", err)
		}
		return
	}

	// Example of handling rate limiting
	tokenManager := &TokenManager{} // Assume properly initialized

	resetRequest := PasswordResetRequest{
		Email: "user@example.com",
	}

	err = tokenManager.RequestPasswordReset(resetRequest)
	if err != nil {
		switch {
		case errors.Is(err, ErrRateLimitExceeded):
			log.Println("Too many password reset requests. Please try again later.")
		case errors.Is(err, ErrInvalidEmail):
			log.Println("Invalid email address provided.")
		default:
			log.Printf("Password reset request failed: %v", err)
		}
	}
}
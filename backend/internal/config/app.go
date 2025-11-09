package config

import (
	"os"
	"sync"
)

// AppConfig holds centralized application configuration
type AppConfig struct {
	// Brand and identity
	AppName        string
	AppDisplayName string
	CompanyName    string
	SupportEmail   string

	// URLs
	AppBaseURL      string
	DashboardURL    string
	VerificationURL string
	SecurityURL     string

	// Email configuration
	EmailFromDomain string
	EmailFromName   string
	EmailMailer     string

	// Metadata
	Description string
	Version     string
}

var (
	appConfig *AppConfig
	once      sync.Once
)

// GetAppConfig returns the singleton app configuration
func GetAppConfig() *AppConfig {
	once.Do(func() {
		appConfig = loadAppConfig()
	})
	return appConfig
}

// loadAppConfig loads configuration from environment variables with sensible defaults
func loadAppConfig() *AppConfig {
	config := &AppConfig{
		// Brand and identity - can be overridden with environment variables
		AppName:        getEnvOrDefault("APP_NAME", "SaaSPlatform"),
		AppDisplayName: getEnvOrDefault("APP_DISPLAY_NAME", "SaaSPlatform"),
		CompanyName:    getEnvOrDefault("COMPANY_NAME", "SaaSPlatform Inc."),
		SupportEmail:   getEnvOrDefault("SUPPORT_EMAIL", "support@saasplatform.com"),

		// URLs - customizable for different environments
		AppBaseURL:      getEnvOrDefault("APP_BASE_URL", "https://app.saasplatform.com"),
		DashboardURL:    getEnvOrDefault("DASHBOARD_URL", "https://app.saasplatform.com/dashboard"),
		VerificationURL: getEnvOrDefault("VERIFICATION_BASE_URL", "https://app.saasplatform.com/verify"),
		SecurityURL:     getEnvOrDefault("SECURITY_URL", "https://app.saasplatform.com/settings/security"),

		// Email configuration
		EmailFromDomain: getEnvOrDefault("EMAIL_FROM_DOMAIN", "saasplatform.com"),
		EmailFromName:   getEnvOrDefault("EMAIL_FROM_NAME", "SaaSPlatform"),
		EmailMailer:     getEnvOrDefault("EMAIL_MAILER", "SaaSPlatform-Auth"),

		// Metadata
		Description: getEnvOrDefault("APP_DESCRIPTION", "A modern SaaS platform built with cutting-edge technology"),
		Version:     getEnvOrDefault("APP_VERSION", "1.0.0"),
	}

	return config
}

// getEnvOrDefault returns the environment variable value or a default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEmailFromAddress returns the complete email address for sending emails
func (c *AppConfig) GetEmailFromAddress() string {
	return "noreply@" + c.EmailFromDomain
}

// GetVerificationURL returns a complete verification URL with token
func (c *AppConfig) GetVerificationURL(token string) string {
	return c.VerificationURL + "?token=" + token
}

// GetWelcomeSubject returns the welcome email subject
func (c *AppConfig) GetWelcomeSubject() string {
	return "Welcome to " + c.AppDisplayName + "!"
}

// GetPasswordResetSubject returns the password reset email subject
func (c *AppConfig) GetPasswordResetSubject() string {
	return "Password Reset Code - " + c.AppDisplayName
}

// GetVerificationSubject returns the email verification subject
func (c *AppConfig) GetVerificationSubject() string {
	return "Verify Your Email - " + c.AppDisplayName
}

// GetSecurityAlertSubject returns the security alert email subject
func (c *AppConfig) GetSecurityAlertSubject() string {
	return "Security Alert - " + c.AppDisplayName
}
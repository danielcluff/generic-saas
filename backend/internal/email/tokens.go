package email

import (
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/danielsaas/generic-saas/internal/config"
)

// TokenType represents different types of email tokens
type TokenType string

const (
	TokenTypePasswordReset    TokenType = "password_reset"
	TokenTypeEmailVerification TokenType = "email_verification"
	TokenTypeMagicLink        TokenType = "magic_link"
)

// EmailToken represents a token stored in the database
type EmailToken struct {
	ID        int       `db:"id" json:"id"`
	Token     string    `db:"token" json:"-"`          // Hashed token
	UserID    int       `db:"user_id" json:"user_id"`
	Email     string    `db:"email" json:"email"`
	Type      TokenType `db:"type" json:"type"`
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
	Used      bool      `db:"used" json:"used"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	RequestIP string    `db:"request_ip" json:"request_ip"`
	UserAgent string    `db:"user_agent" json:"user_agent"`
}

// PasswordResetRequest represents a password reset request
type PasswordResetRequest struct {
	Email     string
	RequestIP string
	UserAgent string
}

// EmailVerificationRequest represents an email verification request
type EmailVerificationRequest struct {
	UserID    int
	Email     string
	Name      string
	RequestIP string
	UserAgent string
}

// TokenManager manages email tokens with security best practices
type TokenManager struct {
	db           *sql.DB
	emailService EmailService
}

// NewTokenManager creates a new token manager
func NewTokenManager(db *sql.DB, emailService EmailService) *TokenManager {
	return &TokenManager{
		db:           db,
		emailService: emailService,
	}
}

// RequestPasswordReset initiates a password reset flow
func (tm *TokenManager) RequestPasswordReset(req PasswordResetRequest) error {
	// Validate email
	if err := validateEmailAddress(req.Email); err != nil {
		return err
	}

	// Rate limiting check would go here
	if err := tm.checkRateLimit(req.Email, TokenTypePasswordReset); err != nil {
		return err
	}

	// Generate secure 6-digit code
	code, err := generateNumericCode(6)
	if err != nil {
		return fmt.Errorf("failed to generate reset code: %w", err)
	}

	// Hash the code for storage
	hashedCode := hashToken(code)

	// Store in database
	expiresAt := time.Now().Add(15 * time.Minute) // 15-minute expiration
	_, err = tm.db.Exec(`
		INSERT INTO email_tokens (token, user_id, email, type, expires_at, used, created_at, request_ip, user_agent)
		SELECT $1, u.id, $2, $3, $4, false, $5, $6, $7
		FROM users u
		WHERE u.email = $2
	`, string(hashedCode[:]), req.Email, TokenTypePasswordReset, expiresAt, time.Now(), req.RequestIP, req.UserAgent)

	if err != nil {
		return fmt.Errorf("failed to store password reset token: %w", err)
	}

	// Send email with code
	securityCtx := SecurityContext{
		RequestIP:   req.RequestIP,
		UserAgent:   req.UserAgent,
		RequestTime: time.Now(),
	}

	return tm.emailService.SendPasswordResetCode(nil, req.Email, code, securityCtx)
}

// VerifyPasswordResetCode verifies a password reset code
func (tm *TokenManager) VerifyPasswordResetCode(email, code string) (*EmailToken, error) {
	// Validate inputs
	if err := validateEmailAddress(email); err != nil {
		return nil, err
	}

	if code == "" {
		return nil, errors.New("code cannot be empty")
	}

	// Hash the provided code for comparison
	providedHash := hashToken(code)

	// Look up the token in the database
	var token EmailToken
	err := tm.db.QueryRow(`
		SELECT id, token, user_id, email, type, expires_at, used, created_at, request_ip, user_agent
		FROM email_tokens
		WHERE email = $1 AND type = $2 AND used = false
		ORDER BY created_at DESC
		LIMIT 1
	`, email, TokenTypePasswordReset).Scan(
		&token.ID, &token.Token, &token.UserID, &token.Email,
		&token.Type, &token.ExpiresAt, &token.Used, &token.CreatedAt,
		&token.RequestIP, &token.UserAgent,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("failed to lookup token: %w", err)
	}

	// Check if token has expired
	if time.Now().After(token.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	// Verify token using constant-time comparison
	storedHash := []byte(token.Token)
	if subtle.ConstantTimeCompare(providedHash[:], storedHash) != 1 {
		return nil, ErrInvalidToken
	}

	// Mark token as used
	_, err = tm.db.Exec(`
		UPDATE email_tokens SET used = true WHERE id = $1
	`, token.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to mark token as used: %w", err)
	}

	return &token, nil
}

// RequestEmailVerification initiates email verification flow
func (tm *TokenManager) RequestEmailVerification(req EmailVerificationRequest) error {
	// Validate inputs
	if err := validateEmailAddress(req.Email); err != nil {
		return err
	}

	if req.UserID <= 0 {
		return errors.New("user ID is required")
	}

	// Generate secure token
	token, err := generateSecureToken()
	if err != nil {
		return fmt.Errorf("failed to generate verification token: %w", err)
	}

	// Hash token for storage
	hashedToken := hashToken(token)

	// Store in database
	expiresAt := time.Now().Add(48 * time.Hour) // 48-hour expiration
	_, err = tm.db.Exec(`
		INSERT INTO email_tokens (token, user_id, email, type, expires_at, used, created_at, request_ip, user_agent)
		VALUES ($1, $2, $3, $4, $5, false, $6, $7, $8)
	`, string(hashedToken[:]), req.UserID, req.Email, TokenTypeEmailVerification, expiresAt, time.Now(), req.RequestIP, req.UserAgent)

	if err != nil {
		return fmt.Errorf("failed to store verification token: %w", err)
	}

	// Create verification URL
	verificationURL := config.GetAppConfig().GetVerificationURL(token)

	// Send verification email
	return tm.emailService.SendEmailVerification(nil, req.Email, req.Name, verificationURL)
}

// VerifyEmailToken verifies an email verification token
func (tm *TokenManager) VerifyEmailToken(token string) (*EmailToken, error) {
	if token == "" {
		return nil, errors.New("token cannot be empty")
	}

	// Hash the provided token for comparison
	providedHash := hashToken(token)

	// Look up the token in the database
	var emailToken EmailToken
	err := tm.db.QueryRow(`
		SELECT id, token, user_id, email, type, expires_at, used, created_at, request_ip, user_agent
		FROM email_tokens
		WHERE token = $1 AND type = $2 AND used = false
		LIMIT 1
	`, string(providedHash[:]), TokenTypeEmailVerification).Scan(
		&emailToken.ID, &emailToken.Token, &emailToken.UserID, &emailToken.Email,
		&emailToken.Type, &emailToken.ExpiresAt, &emailToken.Used, &emailToken.CreatedAt,
		&emailToken.RequestIP, &emailToken.UserAgent,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("failed to lookup token: %w", err)
	}

	// Check if token has expired
	if time.Now().After(emailToken.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	// Mark token as used
	_, err = tm.db.Exec(`
		UPDATE email_tokens SET used = true WHERE id = $1
	`, emailToken.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to mark token as used: %w", err)
	}

	// Mark user's email as verified
	_, err = tm.db.Exec(`
		UPDATE users SET email_verified_at = $1 WHERE id = $2 AND email_verified_at IS NULL
	`, time.Now(), emailToken.UserID)

	if err != nil {
		return nil, fmt.Errorf("failed to mark email as verified: %w", err)
	}

	return &emailToken, nil
}

// CleanupExpiredTokens removes expired tokens from the database
func (tm *TokenManager) CleanupExpiredTokens() error {
	_, err := tm.db.Exec(`
		DELETE FROM email_tokens
		WHERE expires_at < $1 OR (used = true AND created_at < $2)
	`, time.Now(), time.Now().Add(-7*24*time.Hour)) // Keep used tokens for 7 days for audit

	return err
}

// checkRateLimit checks if the email has exceeded rate limits
func (tm *TokenManager) checkRateLimit(email string, tokenType TokenType) error {
	// Count recent requests in the last hour
	var count int
	err := tm.db.QueryRow(`
		SELECT COUNT(*)
		FROM email_tokens
		WHERE email = $1 AND type = $2 AND created_at > $3
	`, email, tokenType, time.Now().Add(-time.Hour)).Scan(&count)

	if err != nil {
		return fmt.Errorf("failed to check rate limit: %w", err)
	}

	// Allow maximum 3 requests per hour
	if count >= 3 {
		return ErrRateLimitExceeded
	}

	return nil
}

// GetTokenStats returns statistics about tokens for monitoring
func (tm *TokenManager) GetTokenStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Count active tokens by type
	rows, err := tm.db.Query(`
		SELECT type, COUNT(*) as count
		FROM email_tokens
		WHERE expires_at > $1 AND used = false
		GROUP BY type
	`, time.Now())

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	activeTokens := make(map[string]int)
	for rows.Next() {
		var tokenType string
		var count int
		if err := rows.Scan(&tokenType, &count); err != nil {
			return nil, err
		}
		activeTokens[tokenType] = count
	}

	stats["active_tokens"] = activeTokens

	// Count expired tokens
	var expiredCount int
	err = tm.db.QueryRow(`
		SELECT COUNT(*) FROM email_tokens WHERE expires_at <= $1
	`, time.Now()).Scan(&expiredCount)

	if err != nil {
		return nil, err
	}

	stats["expired_tokens"] = expiredCount

	return stats, nil
}
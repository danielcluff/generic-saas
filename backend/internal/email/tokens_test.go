package email

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// MockEmailService implements EmailService for testing
type MockEmailService struct {
	sentEmails []MockEmail
	shouldFail bool
}

type MockEmail struct {
	To          string
	Type        string
	Code        string
	URL         string
	SecurityCtx SecurityContext
}

func (m *MockEmailService) SendEmail(ctx context.Context, email *Email) error {
	if m.shouldFail {
		return errors.New("mock email service failure")
	}
	return nil
}

func (m *MockEmailService) SendWelcomeEmail(ctx context.Context, to, name string) error {
	if m.shouldFail {
		return errors.New("mock email service failure")
	}
	m.sentEmails = append(m.sentEmails, MockEmail{To: to, Type: "welcome"})
	return nil
}

func (m *MockEmailService) SendPasswordResetCode(ctx context.Context, to, code string, securityCtx SecurityContext) error {
	if m.shouldFail {
		return errors.New("mock email service failure")
	}
	m.sentEmails = append(m.sentEmails, MockEmail{
		To:          to,
		Type:        "password_reset",
		Code:        code,
		SecurityCtx: securityCtx,
	})
	return nil
}

func (m *MockEmailService) SendEmailVerification(ctx context.Context, to, name, verificationURL string) error {
	if m.shouldFail {
		return errors.New("mock email service failure")
	}
	m.sentEmails = append(m.sentEmails, MockEmail{
		To:   to,
		Type: "email_verification",
		URL:  verificationURL,
	})
	return nil
}

func (m *MockEmailService) SendSecurityAlert(ctx context.Context, to, alertMessage string, securityCtx SecurityContext) error {
	if m.shouldFail {
		return errors.New("mock email service failure")
	}
	m.sentEmails = append(m.sentEmails, MockEmail{
		To:          to,
		Type:        "security_alert",
		SecurityCtx: securityCtx,
	})
	return nil
}

func (m *MockEmailService) GetProviderName() string {
	return "Mock"
}

func TestNewTokenManager(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()

	emailService := &MockEmailService{}
	tokenManager := NewTokenManager(db, emailService)

	if tokenManager == nil {
		t.Fatal("NewTokenManager() returned nil")
	}

	if tokenManager.db != db {
		t.Error("TokenManager db not set correctly")
	}

	if tokenManager.emailService != emailService {
		t.Error("TokenManager emailService not set correctly")
	}
}

func TestRequestPasswordReset(t *testing.T) {
	tests := []struct {
		name        string
		request     PasswordResetRequest
		setupMock   func(sqlmock.Sqlmock)
		emailFails  bool
		expectError bool
		errorType   error
	}{
		{
			name: "successful request",
			request: PasswordResetRequest{
				Email:     "user@example.com",
				RequestIP: "192.168.1.100",
				UserAgent: "Test Agent",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				// Mock rate limiting check
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM email_tokens").
					WithArgs("user@example.com", TokenTypePasswordReset, sqlmock.AnyArg()).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

				// Mock token insertion
				mock.ExpectExec("INSERT INTO email_tokens").
					WithArgs(sqlmock.AnyArg(), "user@example.com", TokenTypePasswordReset,
						sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectError: false,
		},
		{
			name: "invalid email",
			request: PasswordResetRequest{
				Email: "invalid-email",
			},
			setupMock:   func(mock sqlmock.Sqlmock) {},
			expectError: true,
			errorType:   ErrInvalidEmail,
		},
		{
			name: "rate limited",
			request: PasswordResetRequest{
				Email:     "user@example.com",
				RequestIP: "192.168.1.100",
				UserAgent: "Test Agent",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				// Mock rate limiting check - return 3 (at limit)
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM email_tokens").
					WithArgs("user@example.com", TokenTypePasswordReset, sqlmock.AnyArg()).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
			},
			expectError: true,
			errorType:   ErrRateLimitExceeded,
		},
		{
			name: "email service fails",
			request: PasswordResetRequest{
				Email:     "user@example.com",
				RequestIP: "192.168.1.100",
				UserAgent: "Test Agent",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM email_tokens").
					WithArgs("user@example.com", TokenTypePasswordReset, sqlmock.AnyArg()).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
				mock.ExpectExec("INSERT INTO email_tokens").
					WithArgs(sqlmock.AnyArg(), "user@example.com", TokenTypePasswordReset,
						sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			emailFails:  true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("Failed to create mock DB: %v", err)
			}
			defer db.Close()

			emailService := &MockEmailService{shouldFail: tt.emailFails}
			tokenManager := NewTokenManager(db, emailService)

			tt.setupMock(mock)

			err = tokenManager.RequestPasswordReset(tt.request)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorType != nil && !errors.Is(err, tt.errorType) {
					t.Errorf("expected error type %v, got %v", tt.errorType, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify email was sent
			if len(emailService.sentEmails) != 1 {
				t.Errorf("expected 1 email sent, got %d", len(emailService.sentEmails))
				return
			}

			sentEmail := emailService.sentEmails[0]
			if sentEmail.Type != "password_reset" {
				t.Errorf("expected password_reset email, got %s", sentEmail.Type)
			}
			if sentEmail.To != tt.request.Email {
				t.Errorf("email sent to %s, expected %s", sentEmail.To, tt.request.Email)
			}

			// Verify all expectations met
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestVerifyPasswordResetCode(t *testing.T) {
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	futureTime := time.Now().Add(24 * time.Hour) // Use current time + 24 hours to avoid expiration
	pastTime := time.Now().Add(-2 * time.Hour) // Use current time - 2 hours for expiry

	// Pre-compute hash for test token
	testCode := "123456"
	hashedToken := hashToken(testCode)

	tests := []struct {
		name        string
		email       string
		code        string
		setupMock   func(sqlmock.Sqlmock)
		expectError bool
		errorType   error
	}{
		{
			name:  "successful verification",
			email: "user@example.com",
			code:  testCode,
			setupMock: func(mock sqlmock.Sqlmock) {
				// Mock token lookup
				rows := sqlmock.NewRows([]string{"id", "token", "user_id", "email", "type", "expires_at", "used", "created_at", "request_ip", "user_agent"}).
					AddRow(1, hashedToken[:], 123, "user@example.com", TokenTypePasswordReset, futureTime, false, fixedTime, "192.168.1.1", "Test Agent")
				mock.ExpectQuery("SELECT id, token, user_id, email, type, expires_at, used, created_at, request_ip, user_agent FROM email_tokens").
					WithArgs("user@example.com", TokenTypePasswordReset).
					WillReturnRows(rows)

				// Mock token update
				mock.ExpectExec("UPDATE email_tokens SET used = true WHERE id = \\$1").
					WithArgs(1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectError: false,
		},
		{
			name:  "invalid email",
			email: "invalid",
			code:  testCode,
			setupMock: func(mock sqlmock.Sqlmock) {
				// No DB operations expected
			},
			expectError: true,
			errorType:   ErrInvalidEmail,
		},
		{
			name:  "empty code",
			email: "user@example.com",
			code:  "",
			setupMock: func(mock sqlmock.Sqlmock) {
				// No DB operations expected
			},
			expectError: true,
		},
		{
			name:  "token not found",
			email: "user@example.com",
			code:  testCode,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, token, user_id, email, type, expires_at, used, created_at, request_ip, user_agent FROM email_tokens").
					WithArgs("user@example.com", TokenTypePasswordReset).
					WillReturnError(sql.ErrNoRows)
			},
			expectError: true,
			errorType:   ErrInvalidToken,
		},
		{
			name:  "expired token",
			email: "user@example.com",
			code:  testCode,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "token", "user_id", "email", "type", "expires_at", "used", "created_at", "request_ip", "user_agent"}).
					AddRow(1, hashedToken[:], 123, "user@example.com", TokenTypePasswordReset, pastTime, false, fixedTime, "192.168.1.1", "Test Agent")
				mock.ExpectQuery("SELECT id, token, user_id, email, type, expires_at, used, created_at, request_ip, user_agent FROM email_tokens").
					WithArgs("user@example.com", TokenTypePasswordReset).
					WillReturnRows(rows)
			},
			expectError: true,
			errorType:   ErrTokenExpired,
		},
		{
			name:  "wrong code",
			email: "user@example.com",
			code:  "wrong-code",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "token", "user_id", "email", "type", "expires_at", "used", "created_at", "request_ip", "user_agent"}).
					AddRow(1, hashedToken[:], 123, "user@example.com", TokenTypePasswordReset, futureTime, false, fixedTime, "192.168.1.1", "Test Agent")
				mock.ExpectQuery("SELECT id, token, user_id, email, type, expires_at, used, created_at, request_ip, user_agent FROM email_tokens").
					WithArgs("user@example.com", TokenTypePasswordReset).
					WillReturnRows(rows)
			},
			expectError: true,
			errorType:   ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
			if err != nil {
				t.Fatalf("Failed to create mock DB: %v", err)
			}
			defer db.Close()

			emailService := &MockEmailService{}
			tokenManager := NewTokenManager(db, emailService)

			tt.setupMock(mock)

			token, err := tokenManager.VerifyPasswordResetCode(tt.email, tt.code)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorType != nil && !errors.Is(err, tt.errorType) {
					t.Errorf("expected error type %v, got %v", tt.errorType, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if token == nil {
				t.Error("expected token but got nil")
				return
			}

			if token.UserID != 123 {
				t.Errorf("expected user ID 123, got %d", token.UserID)
			}

			// Verify all expectations met
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRequestEmailVerification(t *testing.T) {
	tests := []struct {
		name        string
		request     EmailVerificationRequest
		setupMock   func(sqlmock.Sqlmock)
		emailFails  bool
		expectError bool
	}{
		{
			name: "successful request",
			request: EmailVerificationRequest{
				UserID:    123,
				Email:     "user@example.com",
				Name:      "John Doe",
				RequestIP: "192.168.1.100",
				UserAgent: "Test Agent",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO email_tokens").
					WithArgs(sqlmock.AnyArg(), 123, "user@example.com", TokenTypeEmailVerification,
						sqlmock.AnyArg(), sqlmock.AnyArg(), "192.168.1.100", "Test Agent").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectError: false,
		},
		{
			name: "invalid email",
			request: EmailVerificationRequest{
				UserID: 123,
				Email:  "invalid-email",
				Name:   "John Doe",
			},
			setupMock:   func(mock sqlmock.Sqlmock) {},
			expectError: true,
		},
		{
			name: "invalid user ID",
			request: EmailVerificationRequest{
				UserID: 0,
				Email:  "user@example.com",
				Name:   "John Doe",
			},
			setupMock:   func(mock sqlmock.Sqlmock) {},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("Failed to create mock DB: %v", err)
			}
			defer db.Close()

			emailService := &MockEmailService{shouldFail: tt.emailFails}
			tokenManager := NewTokenManager(db, emailService)

			tt.setupMock(mock)

			err = tokenManager.RequestEmailVerification(tt.request)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify email was sent
			if len(emailService.sentEmails) != 1 {
				t.Errorf("expected 1 email sent, got %d", len(emailService.sentEmails))
				return
			}

			sentEmail := emailService.sentEmails[0]
			if sentEmail.Type != "email_verification" {
				t.Errorf("expected email_verification email, got %s", sentEmail.Type)
			}
			if sentEmail.To != tt.request.Email {
				t.Errorf("email sent to %s, expected %s", sentEmail.To, tt.request.Email)
			}
			if sentEmail.URL == "" {
				t.Error("verification URL should not be empty")
			}

			// Verify all expectations met
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestVerifyEmailToken(t *testing.T) {
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	futureTime := time.Now().Add(48 * time.Hour) // Use current time + 48 hours to avoid expiration

	// Pre-compute hash for test token
	testToken := "abc123def456"
	hashedToken := hashToken(testToken)

	tests := []struct {
		name        string
		token       string
		setupMock   func(sqlmock.Sqlmock)
		expectError bool
		errorType   error
	}{
		{
			name:  "successful verification",
			token: testToken,
			setupMock: func(mock sqlmock.Sqlmock) {
				// Mock token lookup
				rows := sqlmock.NewRows([]string{"id", "token", "user_id", "email", "type", "expires_at", "used", "created_at", "request_ip", "user_agent"}).
					AddRow(1, hashedToken[:], 123, "user@example.com", TokenTypeEmailVerification, futureTime, false, fixedTime, "192.168.1.1", "Test Agent")
				mock.ExpectQuery("SELECT id, token, user_id, email, type, expires_at, used, created_at, request_ip, user_agent FROM email_tokens").
					WithArgs(string(hashedToken[:]), TokenTypeEmailVerification).
					WillReturnRows(rows)

				// Mock token update
				mock.ExpectExec("UPDATE email_tokens SET used = true WHERE id = \\$1").
					WithArgs(1).
					WillReturnResult(sqlmock.NewResult(0, 1))

				// Mock email verification update
				mock.ExpectExec("UPDATE users SET email_verified_at = \\$1 WHERE id = \\$2 AND email_verified_at IS NULL").
					WithArgs(sqlmock.AnyArg(), 123).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectError: false,
		},
		{
			name:  "empty token",
			token: "",
			setupMock: func(mock sqlmock.Sqlmock) {
				// No DB operations expected
			},
			expectError: true,
		},
		{
			name:  "token not found",
			token: testToken,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, token, user_id, email, type, expires_at, used, created_at, request_ip, user_agent FROM email_tokens").
					WithArgs(string(hashedToken[:]), TokenTypeEmailVerification).
					WillReturnError(sql.ErrNoRows)
			},
			expectError: true,
			errorType:   ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
			if err != nil {
				t.Fatalf("Failed to create mock DB: %v", err)
			}
			defer db.Close()

			emailService := &MockEmailService{}
			tokenManager := NewTokenManager(db, emailService)

			tt.setupMock(mock)

			token, err := tokenManager.VerifyEmailToken(tt.token)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorType != nil && !errors.Is(err, tt.errorType) {
					t.Errorf("expected error type %v, got %v", tt.errorType, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if token == nil {
				t.Error("expected token but got nil")
				return
			}

			// Verify all expectations met
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestCleanupExpiredTokens(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()

	emailService := &MockEmailService{}
	tokenManager := NewTokenManager(db, emailService)

	mock.ExpectExec("DELETE FROM email_tokens").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 5))

	err = tokenManager.CleanupExpiredTokens()
	if err != nil {
		t.Errorf("CleanupExpiredTokens() failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetTokenStats(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()

	emailService := &MockEmailService{}
	tokenManager := NewTokenManager(db, emailService)

	// Mock active tokens query
	rows := sqlmock.NewRows([]string{"type", "count"}).
		AddRow("password_reset", 5).
		AddRow("email_verification", 10)
	mock.ExpectQuery("SELECT type, COUNT\\(\\*\\) as count FROM email_tokens").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(rows)

	// Mock expired tokens query
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM email_tokens WHERE expires_at <= \\$1").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	stats, err := tokenManager.GetTokenStats()
	if err != nil {
		t.Errorf("GetTokenStats() failed: %v", err)
	}

	if stats == nil {
		t.Fatal("GetTokenStats() returned nil stats")
	}

	activeTokens, ok := stats["active_tokens"].(map[string]int)
	if !ok {
		t.Fatal("active_tokens not found or wrong type")
	}

	if activeTokens["password_reset"] != 5 {
		t.Errorf("expected 5 password_reset tokens, got %d", activeTokens["password_reset"])
	}

	if activeTokens["email_verification"] != 10 {
		t.Errorf("expected 10 email_verification tokens, got %d", activeTokens["email_verification"])
	}

	expiredCount, ok := stats["expired_tokens"].(int)
	if !ok {
		t.Fatal("expired_tokens not found or wrong type")
	}

	if expiredCount != 3 {
		t.Errorf("expected 3 expired tokens, got %d", expiredCount)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestCheckRateLimit(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		tokenType   TokenType
		count       int
		expectError bool
		errorType   error
	}{
		{
			name:        "under limit",
			email:       "user@example.com",
			tokenType:   TokenTypePasswordReset,
			count:       2,
			expectError: false,
		},
		{
			name:        "at limit",
			email:       "user@example.com",
			tokenType:   TokenTypePasswordReset,
			count:       3,
			expectError: true,
			errorType:   ErrRateLimitExceeded,
		},
		{
			name:        "over limit",
			email:       "user@example.com",
			tokenType:   TokenTypePasswordReset,
			count:       5,
			expectError: true,
			errorType:   ErrRateLimitExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("Failed to create mock DB: %v", err)
			}
			defer db.Close()

			emailService := &MockEmailService{}
			tokenManager := NewTokenManager(db, emailService)

			mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM email_tokens").
				WithArgs(tt.email, tt.tokenType, sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(tt.count))

			err = tokenManager.checkRateLimit(tt.email, tt.tokenType)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorType != nil && !errors.Is(err, tt.errorType) {
					t.Errorf("expected error type %v, got %v", tt.errorType, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
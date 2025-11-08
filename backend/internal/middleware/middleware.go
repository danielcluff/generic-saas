package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/danielsaas/generic-saas/internal/database"
)

type contextKey string

const (
	RequestIDKey contextKey = "requestID"
	StartTimeKey contextKey = "startTime"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.written += int64(n)
	return n, err
}

func RequestLogging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			requestID := generateRequestID()

			ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
			ctx = context.WithValue(ctx, StartTimeKey, start)
			r = r.WithContext(ctx)

			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			logger.Info("Request started",
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
				"request_id", requestID,
			)

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)

			logger.Info("Request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status_code", wrapped.statusCode,
				"duration_ms", duration.Milliseconds(),
				"bytes_written", wrapped.written,
				"request_id", requestID,
			)
		})
	}
}

func ErrorRecovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					requestID := getRequestID(r.Context())

					logger.Error("Panic recovered",
						"error", err,
						"method", r.Method,
						"path", r.URL.Path,
						"request_id", requestID,
					)

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error": "Internal server error", "request_id": "` + requestID + `"}`))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && contains(allowedOrigins, origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else if len(allowedOrigins) == 1 && allowedOrigins[0] == "*" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func getRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return "unknown"
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// RequireAuth middleware ensures the request has valid authentication
func RequireAuth(db database.Database) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeAuthError(w, "Authorization header required")
				return
			}

			// Check Bearer token format
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				writeAuthError(w, "Invalid authorization header format")
				return
			}

			token := parts[1]
			if token == "" {
				writeAuthError(w, "Token required")
				return
			}

			// Parse and validate token (simplified - in production use JWT)
			userID, err := parseToken(token, db)
			if err != nil {
				writeAuthError(w, "Invalid token")
				return
			}

			// Add user ID to context
			ctx := context.WithValue(r.Context(), "user_id", userID)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

// parseToken extracts user ID from token (simplified implementation)
func parseToken(token string, db database.Database) (int, error) {
	// Simplified token parsing - in production use JWT
	// Token format: "token_YYYYMMDDHHMMSS_X" where X is userID+48
	if !strings.HasPrefix(token, "token_") {
		return 0, &AuthError{"invalid token format"}
	}

	parts := strings.Split(token, "_")
	if len(parts) != 3 {
		return 0, &AuthError{"invalid token format"}
	}

	// Extract user ID from the last part (simplified)
	userIDStr := parts[2]
	if len(userIDStr) == 0 {
		return 0, &AuthError{"invalid user ID in token"}
	}

	// Convert back from rune to int (reverse of the generation)
	userID := int(userIDStr[0]) - 48
	if userID < 1 {
		return 0, &AuthError{"invalid user ID"}
	}

	// Verify user exists in database
	_, err := db.Users().GetUserByID(context.Background(), userID)
	if err != nil {
		return 0, &AuthError{"user not found"}
	}

	return userID, nil
}

func writeAuthError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error": "` + message + `"}`))
}

type AuthError struct {
	message string
}

func (e *AuthError) Error() string {
	return e.message
}
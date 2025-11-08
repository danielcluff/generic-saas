package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	handler := RequestLogging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	logOutput := buf.String()

	if !strings.Contains(logOutput, "Request started") {
		t.Error("Expected 'Request started' log message")
	}

	if !strings.Contains(logOutput, "Request completed") {
		t.Error("Expected 'Request completed' log message")
	}

	if !strings.Contains(logOutput, "GET") {
		t.Error("Expected method 'GET' in log")
	}

	if !strings.Contains(logOutput, "/test") {
		t.Error("Expected path '/test' in log")
	}

	if !strings.Contains(logOutput, "test-agent") {
		t.Error("Expected user agent in log")
	}
}

func TestErrorRecovery(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	handler := ErrorRecovery(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Internal server error") {
		t.Error("Expected error message in response body")
	}

	if !strings.Contains(body, "request_id") {
		t.Error("Expected request_id in response body")
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "Panic recovered") {
		t.Error("Expected panic recovery log message")
	}

	if !strings.Contains(logOutput, "test panic") {
		t.Error("Expected panic message in log")
	}
}

func TestCORS(t *testing.T) {
	tests := []struct {
		name            string
		allowedOrigins  []string
		requestOrigin   string
		expectedOrigin  string
		expectAllowed   bool
	}{
		{
			name:            "Allow all origins with wildcard",
			allowedOrigins:  []string{"*"},
			requestOrigin:   "https://example.com",
			expectedOrigin:  "*",
			expectAllowed:   true,
		},
		{
			name:            "Allow specific origin",
			allowedOrigins:  []string{"https://example.com", "https://localhost:3000"},
			requestOrigin:   "https://example.com",
			expectedOrigin:  "https://example.com",
			expectAllowed:   true,
		},
		{
			name:            "Block non-allowed origin",
			allowedOrigins:  []string{"https://example.com"},
			requestOrigin:   "https://malicious.com",
			expectedOrigin:  "",
			expectAllowed:   false,
		},
		{
			name:            "No origin header",
			allowedOrigins:  []string{"https://example.com"},
			requestOrigin:   "",
			expectedOrigin:  "",
			expectAllowed:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := CORS(tt.allowedOrigins)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.requestOrigin != "" {
				req.Header.Set("Origin", tt.requestOrigin)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			actualOrigin := rr.Header().Get("Access-Control-Allow-Origin")
			if actualOrigin != tt.expectedOrigin {
				t.Errorf("Expected Access-Control-Allow-Origin '%s', got '%s'", tt.expectedOrigin, actualOrigin)
			}

			if tt.expectAllowed && actualOrigin == "" {
				t.Error("Expected CORS to be allowed but Access-Control-Allow-Origin header is empty")
			}

			if !tt.expectAllowed && actualOrigin != "" && tt.allowedOrigins[0] != "*" {
				t.Error("Expected CORS to be blocked but Access-Control-Allow-Origin header is set")
			}

			expectedMethods := "GET, POST, PUT, DELETE, OPTIONS"
			actualMethods := rr.Header().Get("Access-Control-Allow-Methods")
			if actualMethods != expectedMethods {
				t.Errorf("Expected Access-Control-Allow-Methods '%s', got '%s'", expectedMethods, actualMethods)
			}
		})
	}
}

func TestCORSPreflight(t *testing.T) {
	handler := CORS([]string{"*"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for OPTIONS request")
	}))

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d for OPTIONS request, got %d", http.StatusOK, rr.Code)
	}

	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Expected Access-Control-Allow-Origin header for OPTIONS request")
	}
}

func TestGenerateRequestID(t *testing.T) {
	id1 := generateRequestID()
	id2 := generateRequestID()

	if id1 == id2 {
		t.Error("Request IDs should be unique")
	}

	if len(id1) < 10 {
		t.Error("Request ID should be reasonably long")
	}
}

func TestGetRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	var capturedRequestID string

	handler := RequestLogging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestID = getRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if capturedRequestID == "" {
		t.Error("Request ID should be captured from context")
	}

	if capturedRequestID == "unknown" {
		t.Error("Request ID should not be 'unknown' when properly set")
	}
}

func TestMiddlewareChain(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	})

	var handler http.Handler = finalHandler
	handler = CORS([]string{"*"})(handler)
	handler = ErrorRecovery(logger)(handler)
	handler = RequestLogging(logger)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("CORS middleware should set Access-Control-Allow-Origin")
	}

	if rr.Header().Get("Content-Type") != "application/json" {
		t.Error("Content-Type should be preserved through middleware chain")
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "Request started") {
		t.Error("Request logging should be active in middleware chain")
	}
}
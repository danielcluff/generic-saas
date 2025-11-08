package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// RootResponse represents the expected JSON response from the root endpoint
type RootResponse struct {
	Message   string                 `json:"message"`
	Status    string                 `json:"status"`
	Version   string                 `json:"version"`
	Endpoints map[string]string      `json:"endpoints"`
}

// HealthResponse represents the expected JSON response from the health endpoint
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

func TestHandleRoot(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedMessage string
	}{
		{
			name:           "GET request to root",
			method:         "GET",
			expectedStatus: http.StatusOK,
			expectedMessage: "Welcome to Generic SaaS API",
		},
		{
			name:           "POST request to root",
			method:         "POST",
			expectedStatus: http.StatusOK,
			expectedMessage: "Welcome to Generic SaaS API",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a request
			req, err := http.NewRequest(tt.method, "/", nil)
			if err != nil {
				t.Fatalf("could not create request: %v", err)
			}

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Call the handler
			handleRoot(rr, req)

			// Check status code
			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}

			// Check content type
			expectedContentType := "application/json"
			if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
				t.Errorf("handler returned wrong content type: got %v want %v",
					contentType, expectedContentType)
			}

			// Parse and validate JSON response
			var response RootResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Errorf("could not parse JSON response: %v", err)
			}

			// Check response fields
			if response.Message != tt.expectedMessage {
				t.Errorf("handler returned wrong message: got %v want %v",
					response.Message, tt.expectedMessage)
			}

			if response.Status != "running" {
				t.Errorf("handler returned wrong status: got %v want %v",
					response.Status, "running")
			}

			if response.Version != "1.0.0" {
				t.Errorf("handler returned wrong version: got %v want %v",
					response.Version, "1.0.0")
			}

			// Check endpoints structure
			expectedEndpoints := map[string]string{
				"health": "/health",
				"root":   "/",
			}

			for key, expectedValue := range expectedEndpoints {
				if actualValue, exists := response.Endpoints[key]; !exists {
					t.Errorf("endpoint %s missing from response", key)
				} else if actualValue != expectedValue {
					t.Errorf("endpoint %s has wrong value: got %v want %v",
						key, actualValue, expectedValue)
				}
			}
		})
	}
}

func TestHandleHealth(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedHealth string
	}{
		{
			name:           "GET request to health",
			method:         "GET",
			expectedStatus: http.StatusOK,
			expectedHealth: "healthy",
		},
		{
			name:           "POST request to health",
			method:         "POST",
			expectedStatus: http.StatusOK,
			expectedHealth: "healthy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a request
			req, err := http.NewRequest(tt.method, "/health", nil)
			if err != nil {
				t.Fatalf("could not create request: %v", err)
			}

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Call the handler
			handleHealth(rr, req)

			// Check status code
			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}

			// Check content type
			expectedContentType := "application/json"
			if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
				t.Errorf("handler returned wrong content type: got %v want %v",
					contentType, expectedContentType)
			}

			// Parse and validate JSON response
			var response HealthResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Errorf("could not parse JSON response: %v", err)
			}

			// Check response fields
			if response.Status != tt.expectedHealth {
				t.Errorf("handler returned wrong health status: got %v want %v",
					response.Status, tt.expectedHealth)
			}
		})
	}
}

// TestHTTPServerIntegration tests the full HTTP server using httptest.Server
func TestHTTPServerIntegration(t *testing.T) {
	// Set up HTTP handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/health", handleHealth)

	// Create test server
	server := httptest.NewServer(mux)
	defer server.Close()

	tests := []struct {
		name           string
		endpoint       string
		expectedStatus int
	}{
		{
			name:           "Integration test - root endpoint",
			endpoint:       "/",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Integration test - health endpoint",
			endpoint:       "/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Integration test - non-existent endpoint",
			endpoint:       "/nonexistent",
			expectedStatus: http.StatusOK, // ServeMux defaults to root handler
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make HTTP request to test server
			resp, err := http.Get(server.URL + tt.endpoint)
			if err != nil {
				t.Fatalf("could not make request: %v", err)
			}
			defer resp.Body.Close()

			// Check status code
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			// For successful responses, check content type
			if resp.StatusCode == http.StatusOK {
				expectedContentType := "application/json"
				if contentType := resp.Header.Get("Content-Type"); contentType != expectedContentType {
					t.Errorf("expected content type %s, got %s", expectedContentType, contentType)
				}
			}
		})
	}
}

// BenchmarkHandleRoot benchmarks the root handler
func BenchmarkHandleRoot(b *testing.B) {
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		b.Fatalf("could not create request: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handleRoot(rr, req)
	}
}

// BenchmarkHandleHealth benchmarks the health handler
func BenchmarkHandleHealth(b *testing.B) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		b.Fatalf("could not create request: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handleHealth(rr, req)
	}
}
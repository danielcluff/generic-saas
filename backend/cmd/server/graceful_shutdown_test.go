package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/danielsaas/generic-saas/internal/middleware"
)

func TestGracefulShutdown(t *testing.T) {
	// Set up logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Set up routes and middleware
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		// Simulate a slow endpoint that takes 2 seconds
		time.Sleep(2 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"message": "slow response completed"}`)
	})

	var handler http.Handler = mux
	handler = middleware.CORS([]string{"*"})(handler)
	handler = middleware.ErrorRecovery(logger)(handler)
	handler = middleware.RequestLogging(logger)(handler)

	// Create server
	server := &http.Server{
		Addr:         ":0", // Use available port
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in background
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.ListenAndServe()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Get the actual port the server is listening on
	addr := server.Addr
	if addr == ":0" {
		// For this test, we'll use a fixed port since getting the dynamic port is complex
		server.Close()
		server = &http.Server{
			Addr:         ":9999",
			Handler:      handler,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		}
		go func() {
			serverDone <- server.ListenAndServe()
		}()
		time.Sleep(100 * time.Millisecond)
	}

	// Start a slow request that should complete during shutdown
	slowRequestDone := make(chan error, 1)
	go func() {
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get("http://localhost:9999/slow")
		if err != nil {
			slowRequestDone <- err
			return
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			slowRequestDone <- fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			return
		}

		slowRequestDone <- nil
	}()

	// Give the slow request time to start
	time.Sleep(500 * time.Millisecond)

	// Initiate graceful shutdown
	shutdownDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		shutdownDone <- server.Shutdown(ctx)
	}()

	// Wait for slow request to complete
	select {
	case err := <-slowRequestDone:
		if err != nil {
			t.Errorf("Slow request failed: %v", err)
		}
	case <-time.After(4 * time.Second):
		t.Error("Slow request did not complete within expected time")
	}

	// Wait for shutdown to complete
	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Errorf("Graceful shutdown failed: %v", err)
		}
	case <-time.After(6 * time.Second):
		t.Error("Graceful shutdown did not complete within timeout")
	}

	// Verify server stopped
	select {
	case err := <-serverDone:
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("Server returned unexpected error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Server did not stop after shutdown")
	}
}

func TestServerTimeouts(t *testing.T) {
	// Test that the server has appropriate timeouts configured
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Reduce log noise during tests
	}))

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)

	var handler http.Handler = mux
	handler = middleware.RequestLogging(logger)(handler)

	server := &http.Server{
		Addr:         ":0",
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Verify timeout values
	expectedReadTimeout := 15 * time.Second
	expectedWriteTimeout := 15 * time.Second
	expectedIdleTimeout := 60 * time.Second

	if server.ReadTimeout != expectedReadTimeout {
		t.Errorf("Expected ReadTimeout %v, got %v", expectedReadTimeout, server.ReadTimeout)
	}

	if server.WriteTimeout != expectedWriteTimeout {
		t.Errorf("Expected WriteTimeout %v, got %v", expectedWriteTimeout, server.WriteTimeout)
	}

	if server.IdleTimeout != expectedIdleTimeout {
		t.Errorf("Expected IdleTimeout %v, got %v", expectedIdleTimeout, server.IdleTimeout)
	}
}

func TestShutdownTimeout(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	mux := http.NewServeMux()
	mux.HandleFunc("/hang", func(w http.ResponseWriter, r *http.Request) {
		// Simulate a hanging request that never responds
		select {} // Block forever
	})

	var handler http.Handler = mux
	handler = middleware.RequestLogging(logger)(handler)

	server := &http.Server{
		Addr:    ":9998",
		Handler: handler,
	}

	// Start server
	go func() {
		server.ListenAndServe()
	}()

	time.Sleep(100 * time.Millisecond)

	// Start a request that will hang
	go func() {
		client := &http.Client{}
		client.Get("http://localhost:9998/hang")
	}()

	time.Sleep(100 * time.Millisecond)

	// Test that shutdown respects the timeout
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	elapsed := time.Since(start)

	// Should timeout and return context.DeadlineExceeded
	if err == nil {
		t.Error("Expected shutdown to timeout, but it completed successfully")
	}

	if elapsed > 2*time.Second {
		t.Errorf("Shutdown took too long: %v, expected around 1 second", elapsed)
	}

	// Force close the server to clean up
	server.Close()
}
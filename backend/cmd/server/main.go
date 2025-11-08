package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danielsaas/generic-saas/internal/auth"
	"github.com/danielsaas/generic-saas/internal/database"
	"github.com/danielsaas/generic-saas/internal/metrics"
	"github.com/danielsaas/generic-saas/internal/middleware"
)

// handleUserProfile routes between GET and PUT for user profile
func handleUserProfile(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		metrics.HandleGetUserProfile(w, r)
	case http.MethodPut:
		metrics.HandleUpdateUserProfile(w, r)
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`{"error": "Method not allowed"}`))
	}
}

func main() {
	// Set up structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Initialize database
	dbFactory := database.NewFactory()
	var db database.Database
	var err error

	// Check if PostgreSQL DSN is provided
	if postgresURI := os.Getenv("DATABASE_URL"); postgresURI != "" {
		logger.Info("Using PostgreSQL database")
		config := database.PostgreSQLConfig(postgresURI)
		db, err = dbFactory.Create(config)
		if err != nil {
			logger.Error("Failed to connect to PostgreSQL", "error", err)
			os.Exit(1)
		}
	} else {
		logger.Info("Using in-memory database")
		db = dbFactory.CreateMemory()
	}

	// Initialize services
	authService := auth.NewService(db)
	auth.SetService(authService)

	metricsService := metrics.NewService(db)
	metrics.SetService(metricsService)

	// Set up routes
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/health", handleHealth)

	// Auth routes
	mux.HandleFunc("/auth/login", auth.HandleLogin)
	mux.HandleFunc("/auth/register", auth.HandleRegister)

	// Protected API routes
	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("/api/metrics", metrics.HandleGetMetrics)
	protectedMux.HandleFunc("/api/user/profile", handleUserProfile)
	protectedMux.HandleFunc("/api/user/password", metrics.HandleUpdateUserPassword)

	// Apply auth middleware to protected routes
	protectedHandler := middleware.RequireAuth(db)(protectedMux)
	mux.Handle("/api/", protectedHandler)

	// Apply middleware
	var handler http.Handler = mux
	handler = middleware.CORS([]string{"*"})(handler) // Allow all origins for development
	handler = middleware.ErrorRecovery(logger)(handler)
	handler = middleware.RequestLogging(logger)(handler)

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create HTTP server with timeouts
	addr := fmt.Sprintf(":%s", port)
	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Create a channel to listen for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		logger.Info("Server starting", "port", port, "addr", fmt.Sprintf("http://localhost:%s", port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	<-quit
	logger.Info("Shutdown signal received")

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	logger.Info("Shutting down server...")
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("Server exited gracefully")
}

// handleRoot handles requests to the root path
func handleRoot(w http.ResponseWriter, r *http.Request) {
	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Write response
	response := `{
	"message": "Welcome to Generic SaaS API",
	"status": "running",
	"version": "1.0.0",
	"endpoints": {
		"health": "/health",
		"root": "/"
	}
}`

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, response)
}

// handleHealth handles health check requests
func handleHealth(w http.ResponseWriter, r *http.Request) {
	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Write health response with current timestamp
	response := `{
	"status": "healthy",
	"timestamp": "` + fmt.Sprintf("%d", 1699545600) + `"
}`

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, response)
}
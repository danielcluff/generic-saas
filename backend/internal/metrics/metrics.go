package metrics

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/danielsaas/generic-saas/internal/database"
	"golang.org/x/crypto/bcrypt"
)

// MetricsResponse represents the dashboard metrics
type MetricsResponse struct {
	TotalUsers     int        `json:"total_users"`
	ActiveSessions int        `json:"active_sessions"`
	APICalls       int        `json:"api_calls"`
	StorageUsed    int64      `json:"storage_used"`
	RecentActivity []Activity `json:"recent_activity"`
}

// Activity represents a user activity
type Activity struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
}

// UserProfileResponse represents user profile information
type UserProfileResponse struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// UpdateProfileRequest represents a profile update request
type UpdateProfileRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// UpdatePasswordRequest represents a password update request
type UpdatePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// Service holds the metrics service dependencies
type Service struct {
	db database.Database
}

// NewService creates a new metrics service
func NewService(db database.Database) *Service {
	return &Service{
		db: db,
	}
}

// GetMetrics returns dashboard metrics for the authenticated user
func (s *Service) GetMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user from context (will be set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		writeErrorResponse(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Generate sample metrics (in a real app, this would query actual data)
	metrics := s.generateSampleMetrics(userID)

	writeJSONResponse(w, metrics, http.StatusOK)
}

// GetUserProfile returns the user's profile information
func (s *Service) GetUserProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user from context (will be set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		writeErrorResponse(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user from database
	user, err := s.db.Users().GetUserByID(r.Context(), userID)
	if err != nil {
		writeErrorResponse(w, "User not found", http.StatusNotFound)
		return
	}

	profile := UserProfileResponse{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
	}

	writeJSONResponse(w, profile, http.StatusOK)
}

// UpdateUserProfile updates the user's profile information
func (s *Service) UpdateUserProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user from context
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		writeErrorResponse(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	name := strings.TrimSpace(req.Name)
	email := strings.ToLower(strings.TrimSpace(req.Email))

	if name == "" {
		writeErrorResponse(w, "Name is required", http.StatusBadRequest)
		return
	}

	if email == "" {
		writeErrorResponse(w, "Email is required", http.StatusBadRequest)
		return
	}

	if !isValidEmail(email) {
		writeErrorResponse(w, "Invalid email format", http.StatusBadRequest)
		return
	}

	// Get current user
	currentUser, err := s.db.Users().GetUserByID(r.Context(), userID)
	if err != nil {
		writeErrorResponse(w, "User not found", http.StatusNotFound)
		return
	}

	// Update user data
	updatedUser := &database.User{
		ID:       userID,
		Name:     name,
		Email:    email,
		Password: currentUser.Password, // Keep existing password
	}

	// Save updated user
	user, err := s.db.Users().UpdateUser(r.Context(), updatedUser)
	if err != nil {
		if errors.Is(err, database.ErrUserAlreadyExists) {
			writeErrorResponse(w, "Email already in use", http.StatusConflict)
			return
		}
		writeErrorResponse(w, "Failed to update profile", http.StatusInternalServerError)
		return
	}

	profile := UserProfileResponse{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
	}

	writeJSONResponse(w, profile, http.StatusOK)
}

// UpdateUserPassword updates the user's password
func (s *Service) UpdateUserPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user from context
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		writeErrorResponse(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req UpdatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.CurrentPassword == "" || req.NewPassword == "" {
		writeErrorResponse(w, "Current password and new password are required", http.StatusBadRequest)
		return
	}

	if len(req.NewPassword) < 8 {
		writeErrorResponse(w, "New password must be at least 8 characters long", http.StatusBadRequest)
		return
	}

	// Get current user
	currentUser, err := s.db.Users().GetUserByID(r.Context(), userID)
	if err != nil {
		writeErrorResponse(w, "User not found", http.StatusNotFound)
		return
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(currentUser.Password), []byte(req.CurrentPassword)); err != nil {
		writeErrorResponse(w, "Current password is incorrect", http.StatusUnauthorized)
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		writeErrorResponse(w, "Failed to process new password", http.StatusInternalServerError)
		return
	}

	// Update user with new password
	updatedUser := &database.User{
		ID:       userID,
		Name:     currentUser.Name,
		Email:    currentUser.Email,
		Password: string(hashedPassword),
	}

	_, err = s.db.Users().UpdateUser(r.Context(), updatedUser)
	if err != nil {
		writeErrorResponse(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	writeJSONResponse(w, map[string]string{"message": "Password updated successfully"}, http.StatusOK)
}

// generateSampleMetrics creates sample metrics for demonstration
func (s *Service) generateSampleMetrics(userID int) MetricsResponse {
	// Generate some sample data based on user ID and current time
	now := time.Now()
	baseValue := userID * 100

	return MetricsResponse{
		TotalUsers:     baseValue + 1247,
		ActiveSessions: baseValue/10 + 23,
		APICalls:       baseValue*50 + 15640,
		StorageUsed:    int64(baseValue*1024*1024 + 524288000), // ~500MB base + variation
		RecentActivity: []Activity{
			{
				ID:          1,
				UserID:      userID,
				Description: "User logged in successfully",
				Timestamp:   now.Add(-10 * time.Minute),
			},
			{
				ID:          2,
				UserID:      userID,
				Description: "Dashboard metrics accessed",
				Timestamp:   now.Add(-25 * time.Minute),
			},
			{
				ID:          3,
				UserID:      userID,
				Description: "API key generated",
				Timestamp:   now.Add(-1 * time.Hour),
			},
			{
				ID:          4,
				UserID:      userID,
				Description: "Profile updated",
				Timestamp:   now.Add(-3 * time.Hour),
			},
			{
				ID:          5,
				UserID:      userID,
				Description: "New project created",
				Timestamp:   now.Add(-6 * time.Hour),
			},
		},
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func writeJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	response := ErrorResponse{Error: message}
	writeJSONResponse(w, response, statusCode)
}

// Global service instance
var globalMetricsService *Service

// SetService sets the global metrics service instance
func SetService(service *Service) {
	globalMetricsService = service
}

// HandleGetMetrics is a wrapper for backwards compatibility
func HandleGetMetrics(w http.ResponseWriter, r *http.Request) {
	if globalMetricsService == nil {
		writeErrorResponse(w, "Metrics service not initialized", http.StatusInternalServerError)
		return
	}
	globalMetricsService.GetMetrics(w, r)
}

// HandleGetUserProfile is a wrapper for backwards compatibility
func HandleGetUserProfile(w http.ResponseWriter, r *http.Request) {
	if globalMetricsService == nil {
		writeErrorResponse(w, "Metrics service not initialized", http.StatusInternalServerError)
		return
	}
	globalMetricsService.GetUserProfile(w, r)
}

// HandleUpdateUserProfile is a wrapper for backwards compatibility
func HandleUpdateUserProfile(w http.ResponseWriter, r *http.Request) {
	if globalMetricsService == nil {
		writeErrorResponse(w, "Metrics service not initialized", http.StatusInternalServerError)
		return
	}
	globalMetricsService.UpdateUserProfile(w, r)
}

// HandleUpdateUserPassword is a wrapper for backwards compatibility
func HandleUpdateUserPassword(w http.ResponseWriter, r *http.Request) {
	if globalMetricsService == nil {
		writeErrorResponse(w, "Metrics service not initialized", http.StatusInternalServerError)
		return
	}
	globalMetricsService.UpdateUserPassword(w, r)
}

// isValidEmail validates email format
func isValidEmail(email string) bool {
	// Simple email validation - in production use a more robust regex
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}
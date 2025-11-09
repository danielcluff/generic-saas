package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/danielsaas/generic-saas/internal/database"
	"golang.org/x/crypto/bcrypt"
)

// Use the User type from the database package
type User = database.User

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token,omitempty"`
	User  User   `json:"user"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// Service holds the auth service dependencies
type Service struct {
	db database.Database
}

// NewService creates a new auth service
func NewService(db database.Database) *Service {
	return &Service{
		db: db,
	}
}

// Login handles user login
func (s *Service) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" {
		writeErrorResponse(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	// Find user by email
	user, err := s.db.Users().GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			writeErrorResponse(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}
		writeErrorResponse(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		writeErrorResponse(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	// Generate token (simplified for now)
	token := generateToken(user.ID)

	response := AuthResponse{
		Token: token,
		User:  *user,
	}

	writeJSONResponse(w, response, http.StatusOK)
}

// Register handles user registration
func (s *Service) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if err := validateRegisterRequest(req); err != nil {
		writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeErrorResponse(w, "Failed to process password", http.StatusInternalServerError)
		return
	}

	// Create user
	user := &User{
		Name:     req.Name,
		Email:    strings.ToLower(strings.TrimSpace(req.Email)),
		Password: string(hashedPassword),
	}

	// Save user to database
	createdUser, err := s.db.Users().CreateUser(r.Context(), user)
	if err != nil {
		if errors.Is(err, database.ErrUserAlreadyExists) {
			writeErrorResponse(w, "User with this email already exists", http.StatusConflict)
			return
		}
		writeErrorResponse(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := AuthResponse{
		User: *createdUser,
	}

	writeJSONResponse(w, response, http.StatusCreated)
}

func validateRegisterRequest(req RegisterRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return &ValidationError{"Name is required"}
	}

	if len(strings.TrimSpace(req.Name)) < 2 {
		return &ValidationError{"Name must be at least 2 characters"}
	}

	if strings.TrimSpace(req.Email) == "" {
		return &ValidationError{"Email is required"}
	}

	if !isValidEmail(req.Email) {
		return &ValidationError{"Invalid email format"}
	}

	if len(req.Password) < 8 {
		return &ValidationError{"Password must be at least 8 characters"}
	}

	// Check for uppercase letter
	if !strings.ContainsAny(req.Password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return &ValidationError{"Password must contain at least one uppercase letter"}
	}

	// Check for lowercase letter
	if !strings.ContainsAny(req.Password, "abcdefghijklmnopqrstuvwxyz") {
		return &ValidationError{"Password must contain at least one lowercase letter"}
	}

	// Check for special character
	if !strings.ContainsAny(req.Password, "!@#$%^&*()_+-=[]{}|;':\",./<>?") {
		return &ValidationError{"Password must contain at least one special character"}
	}

	return nil
}

func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// Global service instance (will be initialized by main)
var globalAuthService *Service

// SetService sets the global auth service instance
func SetService(service *Service) {
	globalAuthService = service
}

// HandleLogin is a wrapper around the service Login method for backwards compatibility
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if globalAuthService == nil {
		writeErrorResponse(w, "Auth service not initialized", http.StatusInternalServerError)
		return
	}
	globalAuthService.Login(w, r)
}

// HandleRegister is a wrapper around the service Register method for backwards compatibility
func HandleRegister(w http.ResponseWriter, r *http.Request) {
	if globalAuthService == nil {
		writeErrorResponse(w, "Auth service not initialized", http.StatusInternalServerError)
		return
	}
	globalAuthService.Register(w, r)
}

func generateToken(userID int) string {
	// Simplified token generation - in production use JWT
	return "token_" + time.Now().Format("20060102150405") + "_" + string(rune(userID+48))
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

type ValidationError struct {
	message string
}

func (e *ValidationError) Error() string {
	return e.message
}
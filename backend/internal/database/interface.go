package database

import (
	"context"
	"time"
)

// User represents a user in the system
type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // Don't include in JSON responses
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserRepository defines the interface for user data operations
type UserRepository interface {
	// CreateUser creates a new user and returns the created user
	CreateUser(ctx context.Context, user *User) (*User, error)

	// GetUserByID retrieves a user by their ID
	GetUserByID(ctx context.Context, id int) (*User, error)

	// GetUserByEmail retrieves a user by their email address
	GetUserByEmail(ctx context.Context, email string) (*User, error)

	// UpdateUser updates an existing user
	UpdateUser(ctx context.Context, user *User) (*User, error)

	// DeleteUser deletes a user by their ID
	DeleteUser(ctx context.Context, id int) error

	// ListUsers retrieves all users (with optional pagination)
	ListUsers(ctx context.Context, limit, offset int) ([]*User, error)

	// Close closes any database connections
	Close() error
}

// Database represents the main database interface that can provide repositories
type Database interface {
	// Users returns the user repository
	Users() UserRepository

	// Close closes all database connections
	Close() error

	// Ping checks if the database connection is alive
	Ping(ctx context.Context) error
}

// DatabaseType represents the type of database implementation
type DatabaseType string

const (
	DatabaseTypeMemory     DatabaseType = "memory"
	DatabaseTypePostgreSQL DatabaseType = "postgresql"
	DatabaseTypeMySQL      DatabaseType = "mysql"
)

// Config holds database configuration
type Config struct {
	Type DatabaseType
	DSN  string // Data Source Name for external databases
}

// DatabaseError represents a database-specific error
type DatabaseError struct {
	Type    string
	Message string
	Err     error
}

func (e *DatabaseError) Error() string {
	if e.Err != nil {
		return e.Type + ": " + e.Message + " (" + e.Err.Error() + ")"
	}
	return e.Type + ": " + e.Message
}

func (e *DatabaseError) Unwrap() error {
	return e.Err
}

// Common error types
var (
	ErrUserNotFound      = &DatabaseError{Type: "NOT_FOUND", Message: "user not found"}
	ErrUserAlreadyExists = &DatabaseError{Type: "CONFLICT", Message: "user already exists"}
	ErrInvalidInput      = &DatabaseError{Type: "INVALID_INPUT", Message: "invalid input provided"}
	ErrDatabaseConnection = &DatabaseError{Type: "CONNECTION", Message: "database connection error"}
)
package database

import (
	"context"
	"strings"
	"sync"
	"time"
)

// MemoryDatabase implements the Database interface using in-memory storage
type MemoryDatabase struct {
	userRepo *MemoryUserRepository
}

// MemoryUserRepository implements UserRepository interface using in-memory storage
type MemoryUserRepository struct {
	mu           sync.RWMutex
	users        map[int]*User
	usersByEmail map[string]*User
	nextID       int
}

// NewMemoryDatabase creates a new in-memory database instance
func NewMemoryDatabase() *MemoryDatabase {
	return &MemoryDatabase{
		userRepo: &MemoryUserRepository{
			users:        make(map[int]*User),
			usersByEmail: make(map[string]*User),
			nextID:       1,
		},
	}
}

// Users returns the user repository
func (db *MemoryDatabase) Users() UserRepository {
	return db.userRepo
}

// Close closes the database (no-op for memory database)
func (db *MemoryDatabase) Close() error {
	return nil
}

// Ping checks if the database is available (always returns nil for memory database)
func (db *MemoryDatabase) Ping(ctx context.Context) error {
	return nil
}

// CreateUser creates a new user and returns the created user
func (r *MemoryUserRepository) CreateUser(ctx context.Context, user *User) (*User, error) {
	if user == nil {
		return nil, &DatabaseError{Type: "INVALID_INPUT", Message: "user cannot be nil"}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Normalize email
	email := strings.ToLower(strings.TrimSpace(user.Email))
	if email == "" {
		return nil, &DatabaseError{Type: "INVALID_INPUT", Message: "email is required"}
	}

	// Check if user already exists
	if _, exists := r.usersByEmail[email]; exists {
		return nil, ErrUserAlreadyExists
	}

	// Create new user
	now := time.Now()
	newUser := &User{
		ID:        r.nextID,
		Name:      strings.TrimSpace(user.Name),
		Email:     email,
		Password:  user.Password,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Store user
	r.users[newUser.ID] = newUser
	r.usersByEmail[email] = newUser
	r.nextID++

	// Return a copy to prevent external modifications
	return r.copyUser(newUser), nil
}

// GetUserByID retrieves a user by their ID
func (r *MemoryUserRepository) GetUserByID(ctx context.Context, id int) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[id]
	if !exists {
		return nil, ErrUserNotFound
	}

	return r.copyUser(user), nil
}

// GetUserByEmail retrieves a user by their email address
func (r *MemoryUserRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	user, exists := r.usersByEmail[normalizedEmail]
	if !exists {
		return nil, ErrUserNotFound
	}

	return r.copyUser(user), nil
}

// UpdateUser updates an existing user
func (r *MemoryUserRepository) UpdateUser(ctx context.Context, user *User) (*User, error) {
	if user == nil {
		return nil, &DatabaseError{Type: "INVALID_INPUT", Message: "user cannot be nil"}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if user exists
	existingUser, exists := r.users[user.ID]
	if !exists {
		return nil, ErrUserNotFound
	}

	// Normalize new email
	newEmail := strings.ToLower(strings.TrimSpace(user.Email))
	if newEmail == "" {
		return nil, &DatabaseError{Type: "INVALID_INPUT", Message: "email is required"}
	}

	// If email is changing, check for conflicts
	if existingUser.Email != newEmail {
		if _, emailExists := r.usersByEmail[newEmail]; emailExists {
			return nil, ErrUserAlreadyExists
		}

		// Remove old email mapping
		delete(r.usersByEmail, existingUser.Email)
	}

	// Update user
	updatedUser := &User{
		ID:        user.ID,
		Name:      strings.TrimSpace(user.Name),
		Email:     newEmail,
		Password:  user.Password,
		CreatedAt: existingUser.CreatedAt,
		UpdatedAt: time.Now(),
	}

	// Store updated user
	r.users[user.ID] = updatedUser
	r.usersByEmail[newEmail] = updatedUser

	return r.copyUser(updatedUser), nil
}

// DeleteUser deletes a user by their ID
func (r *MemoryUserRepository) DeleteUser(ctx context.Context, id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.users[id]
	if !exists {
		return ErrUserNotFound
	}

	// Remove from both maps
	delete(r.users, id)
	delete(r.usersByEmail, user.Email)

	return nil
}

// ListUsers retrieves all users (with optional pagination)
func (r *MemoryUserRepository) ListUsers(ctx context.Context, limit, offset int) ([]*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Convert map to slice
	allUsers := make([]*User, 0, len(r.users))
	for _, user := range r.users {
		allUsers = append(allUsers, r.copyUser(user))
	}

	// Apply pagination
	totalUsers := len(allUsers)
	if offset >= totalUsers {
		return []*User{}, nil
	}

	end := offset + limit
	if limit <= 0 || end > totalUsers {
		end = totalUsers
	}

	return allUsers[offset:end], nil
}

// Close closes any database connections (no-op for memory repository)
func (r *MemoryUserRepository) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clear all data
	r.users = make(map[int]*User)
	r.usersByEmail = make(map[string]*User)
	r.nextID = 1

	return nil
}

// copyUser creates a deep copy of a user to prevent external modifications
func (r *MemoryUserRepository) copyUser(user *User) *User {
	if user == nil {
		return nil
	}

	return &User{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Password:  user.Password,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

// GetUserCount returns the total number of users (helper method for testing)
func (r *MemoryUserRepository) GetUserCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.users)
}

// Reset clears all users (helper method for testing)
func (r *MemoryUserRepository) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.users = make(map[int]*User)
	r.usersByEmail = make(map[string]*User)
	r.nextID = 1
}
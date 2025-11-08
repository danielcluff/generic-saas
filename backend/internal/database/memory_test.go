package database

import (
	"context"
	"testing"
)

func TestMemoryUserRepository_CreateUser(t *testing.T) {
	repo := &MemoryUserRepository{
		users:        make(map[int]*User),
		usersByEmail: make(map[string]*User),
		nextID:       1,
	}

	tests := []struct {
		name      string
		user      *User
		wantError bool
		errorType string
	}{
		{
			name: "Valid user",
			user: &User{
				Name:     "John Doe",
				Email:    "john@example.com",
				Password: "hashedpassword",
			},
			wantError: false,
		},
		{
			name:      "Nil user",
			user:      nil,
			wantError: true,
			errorType: "INVALID_INPUT",
		},
		{
			name: "Empty email",
			user: &User{
				Name:     "John Doe",
				Email:    "",
				Password: "hashedpassword",
			},
			wantError: true,
			errorType: "INVALID_INPUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			createdUser, err := repo.CreateUser(ctx, tt.user)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, but got none")
					return
				}
				if dbErr, ok := err.(*DatabaseError); ok {
					if dbErr.Type != tt.errorType {
						t.Errorf("Expected error type %s, got %s", tt.errorType, dbErr.Type)
					}
				} else {
					t.Errorf("Expected DatabaseError, got %T", err)
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error, but got: %v", err)
				return
			}

			if createdUser == nil {
				t.Error("Expected created user, but got nil")
				return
			}

			if createdUser.ID != 1 {
				t.Errorf("Expected ID 1, got %d", createdUser.ID)
			}

			if createdUser.Email != "john@example.com" {
				t.Errorf("Expected email 'john@example.com', got '%s'", createdUser.Email)
			}

			// Verify user is stored
			if len(repo.users) != 1 {
				t.Errorf("Expected 1 user in storage, got %d", len(repo.users))
			}
		})
	}
}

func TestMemoryUserRepository_DuplicateEmail(t *testing.T) {
	repo := &MemoryUserRepository{
		users:        make(map[int]*User),
		usersByEmail: make(map[string]*User),
		nextID:       1,
	}

	ctx := context.Background()

	// Create first user
	user1 := &User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "hashedpassword",
	}

	_, err := repo.CreateUser(ctx, user1)
	if err != nil {
		t.Fatalf("Failed to create first user: %v", err)
	}

	// Try to create user with same email
	user2 := &User{
		Name:     "Jane Doe",
		Email:    "john@example.com",
		Password: "hashedpassword2",
	}

	_, err = repo.CreateUser(ctx, user2)
	if err == nil {
		t.Error("Expected error for duplicate email, but got none")
		return
	}

	if !isErrorType(err, ErrUserAlreadyExists) {
		t.Errorf("Expected ErrUserAlreadyExists, got %v", err)
	}
}

func TestMemoryUserRepository_GetUserByEmail(t *testing.T) {
	repo := &MemoryUserRepository{
		users:        make(map[int]*User),
		usersByEmail: make(map[string]*User),
		nextID:       1,
	}

	ctx := context.Background()

	// Create a user
	user := &User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "hashedpassword",
	}

	createdUser, err := repo.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Test getting existing user
	foundUser, err := repo.GetUserByEmail(ctx, "john@example.com")
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
		return
	}

	if foundUser.ID != createdUser.ID {
		t.Errorf("Expected ID %d, got %d", createdUser.ID, foundUser.ID)
	}

	// Test getting non-existent user
	_, err = repo.GetUserByEmail(ctx, "nonexistent@example.com")
	if err == nil {
		t.Error("Expected error for non-existent user, but got none")
		return
	}

	if !isErrorType(err, ErrUserNotFound) {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestMemoryUserRepository_UpdateUser(t *testing.T) {
	repo := &MemoryUserRepository{
		users:        make(map[int]*User),
		usersByEmail: make(map[string]*User),
		nextID:       1,
	}

	ctx := context.Background()

	// Create a user
	user := &User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "hashedpassword",
	}

	createdUser, err := repo.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Update user
	updatedUser := &User{
		ID:       createdUser.ID,
		Name:     "John Smith",
		Email:    "john.smith@example.com",
		Password: "newhashedpassword",
	}

	result, err := repo.UpdateUser(ctx, updatedUser)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
		return
	}

	if result.Name != "John Smith" {
		t.Errorf("Expected name 'John Smith', got '%s'", result.Name)
	}

	if result.Email != "john.smith@example.com" {
		t.Errorf("Expected email 'john.smith@example.com', got '%s'", result.Email)
	}

	// Verify old email is removed from index
	_, err = repo.GetUserByEmail(ctx, "john@example.com")
	if !isErrorType(err, ErrUserNotFound) {
		t.Error("Expected old email to be removed from index")
	}

	// Verify new email works
	foundUser, err := repo.GetUserByEmail(ctx, "john.smith@example.com")
	if err != nil {
		t.Errorf("Expected to find user by new email: %v", err)
	}
	if foundUser.ID != createdUser.ID {
		t.Error("Found user should have same ID")
	}
}

func TestMemoryUserRepository_DeleteUser(t *testing.T) {
	repo := &MemoryUserRepository{
		users:        make(map[int]*User),
		usersByEmail: make(map[string]*User),
		nextID:       1,
	}

	ctx := context.Background()

	// Create a user
	user := &User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "hashedpassword",
	}

	createdUser, err := repo.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Delete user
	err = repo.DeleteUser(ctx, createdUser.ID)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}

	// Verify user is deleted
	_, err = repo.GetUserByID(ctx, createdUser.ID)
	if !isErrorType(err, ErrUserNotFound) {
		t.Error("Expected user to be deleted")
	}

	// Verify email index is cleaned up
	_, err = repo.GetUserByEmail(ctx, "john@example.com")
	if !isErrorType(err, ErrUserNotFound) {
		t.Error("Expected email index to be cleaned up")
	}
}

func TestMemoryUserRepository_ListUsers(t *testing.T) {
	repo := &MemoryUserRepository{
		users:        make(map[int]*User),
		usersByEmail: make(map[string]*User),
		nextID:       1,
	}

	ctx := context.Background()

	// Create multiple users
	users := []*User{
		{Name: "User 1", Email: "user1@example.com", Password: "pass1"},
		{Name: "User 2", Email: "user2@example.com", Password: "pass2"},
		{Name: "User 3", Email: "user3@example.com", Password: "pass3"},
	}

	for _, user := range users {
		_, err := repo.CreateUser(ctx, user)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
	}

	// Test listing all users
	allUsers, err := repo.ListUsers(ctx, 0, 0)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}

	if len(allUsers) != 3 {
		t.Errorf("Expected 3 users, got %d", len(allUsers))
	}

	// Test pagination
	pageUsers, err := repo.ListUsers(ctx, 2, 1)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}

	if len(pageUsers) != 2 {
		t.Errorf("Expected 2 users in page, got %d", len(pageUsers))
	}
}

func TestMemoryDatabase(t *testing.T) {
	db := NewMemoryDatabase()

	// Test interface implementation
	if db.Users() == nil {
		t.Error("Expected Users() to return non-nil repository")
	}

	// Test Ping
	ctx := context.Background()
	if err := db.Ping(ctx); err != nil {
		t.Errorf("Expected Ping to succeed, got: %v", err)
	}

	// Test Close
	if err := db.Close(); err != nil {
		t.Errorf("Expected Close to succeed, got: %v", err)
	}
}

// Helper function to check error types
func isErrorType(err error, target *DatabaseError) bool {
	if dbErr, ok := err.(*DatabaseError); ok {
		return dbErr.Type == target.Type
	}
	return false
}
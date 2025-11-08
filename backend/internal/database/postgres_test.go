package database

import (
	"context"
	"testing"
	"time"
)

const testDSN = "postgres://saas_user:saas_password@localhost:5432/generic_saas?sslmode=disable"

func setupPostgreSQLTest(t *testing.T) *PostgreSQLDatabase {
	db, err := NewPostgreSQLDatabase(testDSN)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}

	// Clean up any existing test data
	_, err = db.db.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("Failed to clean test data: %v", err)
	}

	return db
}

func TestPostgreSQLUserRepository_CreateUser(t *testing.T) {
	db := setupPostgreSQLTest(t)
	defer db.Close()

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
		{
			name: "Empty name",
			user: &User{
				Name:     "",
				Email:    "john@example.com",
				Password: "hashedpassword",
			},
			wantError: true,
			errorType: "INVALID_INPUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			createdUser, err := db.Users().CreateUser(ctx, tt.user)

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

			if createdUser.ID == 0 {
				t.Error("Expected non-zero ID")
			}

			if createdUser.Email != "john@example.com" {
				t.Errorf("Expected email 'john@example.com', got '%s'", createdUser.Email)
			}

			if createdUser.CreatedAt.IsZero() {
				t.Error("Expected non-zero CreatedAt")
			}

			if createdUser.UpdatedAt.IsZero() {
				t.Error("Expected non-zero UpdatedAt")
			}
		})
	}
}

func TestPostgreSQLUserRepository_DuplicateEmail(t *testing.T) {
	db := setupPostgreSQLTest(t)
	defer db.Close()

	ctx := context.Background()

	// Create first user
	user1 := &User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "hashedpassword",
	}

	_, err := db.Users().CreateUser(ctx, user1)
	if err != nil {
		t.Fatalf("Failed to create first user: %v", err)
	}

	// Try to create user with same email
	user2 := &User{
		Name:     "Jane Doe",
		Email:    "john@example.com",
		Password: "hashedpassword2",
	}

	_, err = db.Users().CreateUser(ctx, user2)
	if err == nil {
		t.Error("Expected error for duplicate email, but got none")
		return
	}

	if !isErrorType(err, ErrUserAlreadyExists) {
		t.Errorf("Expected ErrUserAlreadyExists, got %v", err)
	}
}

func TestPostgreSQLUserRepository_GetUserByEmail(t *testing.T) {
	db := setupPostgreSQLTest(t)
	defer db.Close()

	ctx := context.Background()

	// Create a user
	user := &User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "hashedpassword",
	}

	createdUser, err := db.Users().CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Test getting existing user
	foundUser, err := db.Users().GetUserByEmail(ctx, "john@example.com")
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
		return
	}

	if foundUser.ID != createdUser.ID {
		t.Errorf("Expected ID %d, got %d", createdUser.ID, foundUser.ID)
	}

	// Test getting non-existent user
	_, err = db.Users().GetUserByEmail(ctx, "nonexistent@example.com")
	if err == nil {
		t.Error("Expected error for non-existent user, but got none")
		return
	}

	if !isErrorType(err, ErrUserNotFound) {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestPostgreSQLUserRepository_GetUserByID(t *testing.T) {
	db := setupPostgreSQLTest(t)
	defer db.Close()

	ctx := context.Background()

	// Create a user
	user := &User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "hashedpassword",
	}

	createdUser, err := db.Users().CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Test getting existing user
	foundUser, err := db.Users().GetUserByID(ctx, createdUser.ID)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
		return
	}

	if foundUser.Email != createdUser.Email {
		t.Errorf("Expected email %s, got %s", createdUser.Email, foundUser.Email)
	}

	// Test getting non-existent user
	_, err = db.Users().GetUserByID(ctx, 99999)
	if err == nil {
		t.Error("Expected error for non-existent user, but got none")
		return
	}

	if !isErrorType(err, ErrUserNotFound) {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestPostgreSQLUserRepository_UpdateUser(t *testing.T) {
	db := setupPostgreSQLTest(t)
	defer db.Close()

	ctx := context.Background()

	// Create a user
	user := &User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "hashedpassword",
	}

	createdUser, err := db.Users().CreateUser(ctx, user)
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

	result, err := db.Users().UpdateUser(ctx, updatedUser)
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

	// Verify old email no longer works
	_, err = db.Users().GetUserByEmail(ctx, "john@example.com")
	if !isErrorType(err, ErrUserNotFound) {
		t.Error("Expected old email to be removed")
	}

	// Verify new email works
	foundUser, err := db.Users().GetUserByEmail(ctx, "john.smith@example.com")
	if err != nil {
		t.Errorf("Expected to find user by new email: %v", err)
	}
	if foundUser.ID != createdUser.ID {
		t.Error("Found user should have same ID")
	}
}

func TestPostgreSQLUserRepository_DeleteUser(t *testing.T) {
	db := setupPostgreSQLTest(t)
	defer db.Close()

	ctx := context.Background()

	// Create a user
	user := &User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "hashedpassword",
	}

	createdUser, err := db.Users().CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Delete user
	err = db.Users().DeleteUser(ctx, createdUser.ID)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}

	// Verify user is deleted
	_, err = db.Users().GetUserByID(ctx, createdUser.ID)
	if !isErrorType(err, ErrUserNotFound) {
		t.Error("Expected user to be deleted")
	}

	// Test deleting non-existent user
	err = db.Users().DeleteUser(ctx, 99999)
	if !isErrorType(err, ErrUserNotFound) {
		t.Error("Expected ErrUserNotFound for non-existent user")
	}
}

func TestPostgreSQLUserRepository_ListUsers(t *testing.T) {
	db := setupPostgreSQLTest(t)
	defer db.Close()

	ctx := context.Background()

	// Create multiple users
	users := []*User{
		{Name: "User 1", Email: "user1@example.com", Password: "pass1"},
		{Name: "User 2", Email: "user2@example.com", Password: "pass2"},
		{Name: "User 3", Email: "user3@example.com", Password: "pass3"},
	}

	for _, user := range users {
		_, err := db.Users().CreateUser(ctx, user)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
		// Small delay to ensure different timestamps
		time.Sleep(1 * time.Millisecond)
	}

	// Test listing all users
	allUsers, err := db.Users().ListUsers(ctx, 0, 0)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}

	if len(allUsers) != 3 {
		t.Errorf("Expected 3 users, got %d", len(allUsers))
	}

	// Test pagination
	pageUsers, err := db.Users().ListUsers(ctx, 2, 1)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}

	if len(pageUsers) != 2 {
		t.Errorf("Expected 2 users in page, got %d", len(pageUsers))
	}
}

func TestPostgreSQLDatabase_Interface(t *testing.T) {
	db := setupPostgreSQLTest(t)
	defer db.Close()

	// Test interface implementation
	if db.Users() == nil {
		t.Error("Expected Users() to return non-nil repository")
	}

	// Test Ping
	ctx := context.Background()
	if err := db.Ping(ctx); err != nil {
		t.Errorf("Expected Ping to succeed, got: %v", err)
	}
}

func TestFactory_CreatePostgreSQL(t *testing.T) {
	factory := NewFactory()

	// Test valid DSN
	config := PostgreSQLConfig(testDSN)
	db, err := factory.Create(config)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer db.Close()

	if db == nil {
		t.Error("Expected non-nil database")
	}

	// Test empty DSN
	config = PostgreSQLConfig("")
	_, err = factory.Create(config)
	if err == nil {
		t.Error("Expected error for empty DSN")
	}
}
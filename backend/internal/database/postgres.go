package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// PostgreSQLDatabase implements the Database interface using PostgreSQL
type PostgreSQLDatabase struct {
	db       *sql.DB
	userRepo *PostgreSQLUserRepository
}

// PostgreSQLUserRepository implements UserRepository interface using PostgreSQL
type PostgreSQLUserRepository struct {
	db *sql.DB
}

// NewPostgreSQLDatabase creates a new PostgreSQL database instance
func NewPostgreSQLDatabase(dsn string) (*PostgreSQLDatabase, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Run migrations
	migrationRunner := NewMigrationRunner(db)
	if err := migrationRunner.RunMigrations(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &PostgreSQLDatabase{
		db: db,
		userRepo: &PostgreSQLUserRepository{
			db: db,
		},
	}, nil
}

// Users returns the user repository
func (db *PostgreSQLDatabase) Users() UserRepository {
	return db.userRepo
}

// Close closes the database connection
func (db *PostgreSQLDatabase) Close() error {
	return db.db.Close()
}

// Ping checks if the database connection is alive
func (db *PostgreSQLDatabase) Ping(ctx context.Context) error {
	return db.db.PingContext(ctx)
}

// CreateUser creates a new user and returns the created user
func (r *PostgreSQLUserRepository) CreateUser(ctx context.Context, user *User) (*User, error) {
	if user == nil {
		return nil, &DatabaseError{Type: "INVALID_INPUT", Message: "user cannot be nil"}
	}

	// Normalize email
	email := strings.ToLower(strings.TrimSpace(user.Email))
	if email == "" {
		return nil, &DatabaseError{Type: "INVALID_INPUT", Message: "email is required"}
	}

	name := strings.TrimSpace(user.Name)
	if name == "" {
		return nil, &DatabaseError{Type: "INVALID_INPUT", Message: "name is required"}
	}

	query := `
		INSERT INTO users (name, email, password, created_at, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, name, email, password, created_at, updated_at
	`

	var createdUser User
	err := r.db.QueryRowContext(ctx, query, name, email, user.Password).Scan(
		&createdUser.ID,
		&createdUser.Name,
		&createdUser.Email,
		&createdUser.Password,
		&createdUser.CreatedAt,
		&createdUser.UpdatedAt,
	)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return nil, ErrUserAlreadyExists
		}
		return nil, &DatabaseError{
			Type:    "DATABASE_ERROR",
			Message: "failed to create user",
			Err:     err,
		}
	}

	return &createdUser, nil
}

// GetUserByID retrieves a user by their ID
func (r *PostgreSQLUserRepository) GetUserByID(ctx context.Context, id int) (*User, error) {
	query := `
		SELECT id, name, email, password, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, &DatabaseError{
			Type:    "DATABASE_ERROR",
			Message: "failed to get user by ID",
			Err:     err,
		}
	}

	return &user, nil
}

// GetUserByEmail retrieves a user by their email address
func (r *PostgreSQLUserRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))

	query := `
		SELECT id, name, email, password, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user User
	err := r.db.QueryRowContext(ctx, query, normalizedEmail).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, &DatabaseError{
			Type:    "DATABASE_ERROR",
			Message: "failed to get user by email",
			Err:     err,
		}
	}

	return &user, nil
}

// UpdateUser updates an existing user
func (r *PostgreSQLUserRepository) UpdateUser(ctx context.Context, user *User) (*User, error) {
	if user == nil {
		return nil, &DatabaseError{Type: "INVALID_INPUT", Message: "user cannot be nil"}
	}

	// Normalize email
	email := strings.ToLower(strings.TrimSpace(user.Email))
	if email == "" {
		return nil, &DatabaseError{Type: "INVALID_INPUT", Message: "email is required"}
	}

	name := strings.TrimSpace(user.Name)
	if name == "" {
		return nil, &DatabaseError{Type: "INVALID_INPUT", Message: "name is required"}
	}

	query := `
		UPDATE users
		SET name = $2, email = $3, password = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING id, name, email, password, created_at, updated_at
	`

	var updatedUser User
	err := r.db.QueryRowContext(ctx, query, user.ID, name, email, user.Password).Scan(
		&updatedUser.ID,
		&updatedUser.Name,
		&updatedUser.Email,
		&updatedUser.Password,
		&updatedUser.CreatedAt,
		&updatedUser.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return nil, ErrUserAlreadyExists
		}
		return nil, &DatabaseError{
			Type:    "DATABASE_ERROR",
			Message: "failed to update user",
			Err:     err,
		}
	}

	return &updatedUser, nil
}

// DeleteUser deletes a user by their ID
func (r *PostgreSQLUserRepository) DeleteUser(ctx context.Context, id int) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return &DatabaseError{
			Type:    "DATABASE_ERROR",
			Message: "failed to delete user",
			Err:     err,
		}
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return &DatabaseError{
			Type:    "DATABASE_ERROR",
			Message: "failed to get rows affected",
			Err:     err,
		}
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// ListUsers retrieves all users (with optional pagination)
func (r *PostgreSQLUserRepository) ListUsers(ctx context.Context, limit, offset int) ([]*User, error) {
	query := `
		SELECT id, name, email, password, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
	`

	args := []interface{}{}
	if limit > 0 {
		query += " LIMIT $1"
		args = append(args, limit)
		if offset > 0 {
			query += " OFFSET $2"
			args = append(args, offset)
		}
	} else if offset > 0 {
		query += " OFFSET $1"
		args = append(args, offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, &DatabaseError{
			Type:    "DATABASE_ERROR",
			Message: "failed to list users",
			Err:     err,
		}
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.Password,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, &DatabaseError{
				Type:    "DATABASE_ERROR",
				Message: "failed to scan user row",
				Err:     err,
			}
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, &DatabaseError{
			Type:    "DATABASE_ERROR",
			Message: "error iterating user rows",
			Err:     err,
		}
	}

	return users, nil
}

// Close closes any database connections (no-op for PostgreSQL user repository)
func (r *PostgreSQLUserRepository) Close() error {
	return nil
}
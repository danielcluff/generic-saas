package database

import (
	"database/sql"
	"fmt"
)

// Migration represents a database migration
type Migration struct {
	Version int
	Name    string
	Up      string
	Down    string
}

// GetMigrations returns all available migrations
func GetMigrations() []Migration {
	return []Migration{
		{
			Version: 1,
			Name:    "create_users_table",
			Up: `
				CREATE TABLE IF NOT EXISTS users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					email VARCHAR(255) UNIQUE NOT NULL,
					password VARCHAR(255) NOT NULL,
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
				);

				-- Create index on email for faster lookups
				CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

				-- Create trigger to automatically update updated_at
				CREATE OR REPLACE FUNCTION update_updated_at_column()
				RETURNS TRIGGER AS $$
				BEGIN
					NEW.updated_at = CURRENT_TIMESTAMP;
					RETURN NEW;
				END;
				$$ language 'plpgsql';

				CREATE TRIGGER update_users_updated_at
				BEFORE UPDATE ON users
				FOR EACH ROW
				EXECUTE FUNCTION update_updated_at_column();
			`,
			Down: `
				DROP TRIGGER IF EXISTS update_users_updated_at ON users;
				DROP FUNCTION IF EXISTS update_updated_at_column();
				DROP INDEX IF EXISTS idx_users_email;
				DROP TABLE IF EXISTS users;
			`,
		},
	}
}

// MigrationRunner handles database migrations
type MigrationRunner struct {
	db *sql.DB
}

// NewMigrationRunner creates a new migration runner
func NewMigrationRunner(db *sql.DB) *MigrationRunner {
	return &MigrationRunner{db: db}
}

// Initialize creates the migrations table if it doesn't exist
func (mr *MigrationRunner) Initialize() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
	`
	_, err := mr.db.Exec(query)
	return err
}

// GetAppliedMigrations returns a list of applied migration versions
func (mr *MigrationRunner) GetAppliedMigrations() (map[int]bool, error) {
	applied := make(map[int]bool)

	rows, err := mr.db.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}

// RunMigrations applies all pending migrations
func (mr *MigrationRunner) RunMigrations() error {
	if err := mr.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize migrations table: %w", err)
	}

	applied, err := mr.GetAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	migrations := GetMigrations()
	for _, migration := range migrations {
		if applied[migration.Version] {
			continue // Already applied
		}

		// Begin transaction
		tx, err := mr.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %d: %w", migration.Version, err)
		}

		// Execute migration
		if _, err := tx.Exec(migration.Up); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %d (%s): %w", migration.Version, migration.Name, err)
		}

		// Record migration as applied
		if _, err := tx.Exec(
			"INSERT INTO schema_migrations (version, name) VALUES ($1, $2)",
			migration.Version, migration.Name,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", migration.Version, err)
		}

		fmt.Printf("Applied migration %d: %s\n", migration.Version, migration.Name)
	}

	return nil
}

// RollbackMigration rolls back a specific migration
func (mr *MigrationRunner) RollbackMigration(version int) error {
	migrations := GetMigrations()
	var targetMigration *Migration

	for _, migration := range migrations {
		if migration.Version == version {
			targetMigration = &migration
			break
		}
	}

	if targetMigration == nil {
		return fmt.Errorf("migration version %d not found", version)
	}

	// Begin transaction
	tx, err := mr.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for rollback %d: %w", version, err)
	}

	// Execute rollback
	if _, err := tx.Exec(targetMigration.Down); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to execute rollback %d (%s): %w", version, targetMigration.Name, err)
	}

	// Remove migration record
	if _, err := tx.Exec("DELETE FROM schema_migrations WHERE version = $1", version); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to remove migration record %d: %w", version, err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollback %d: %w", version, err)
	}

	fmt.Printf("Rolled back migration %d: %s\n", version, targetMigration.Name)
	return nil
}
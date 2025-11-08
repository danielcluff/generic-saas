package database

import (
	"fmt"
)

// Factory provides a way to create different database implementations
type Factory struct{}

// NewFactory creates a new database factory
func NewFactory() *Factory {
	return &Factory{}
}

// Create creates a new database instance based on the provided configuration
func (f *Factory) Create(config *Config) (Database, error) {
	if config == nil {
		return nil, &DatabaseError{
			Type:    "INVALID_CONFIG",
			Message: "database configuration cannot be nil",
		}
	}

	switch config.Type {
	case DatabaseTypeMemory:
		return f.createMemoryDatabase(config)
	case DatabaseTypePostgreSQL:
		return f.createPostgreSQLDatabase(config)
	case DatabaseTypeMySQL:
		return f.createMySQLDatabase(config)
	default:
		return nil, &DatabaseError{
			Type:    "UNSUPPORTED_TYPE",
			Message: fmt.Sprintf("unsupported database type: %s", config.Type),
		}
	}
}

// CreateMemory is a convenience method to create an in-memory database
func (f *Factory) CreateMemory() Database {
	return NewMemoryDatabase()
}

// createMemoryDatabase creates a new in-memory database instance
func (f *Factory) createMemoryDatabase(config *Config) (Database, error) {
	return NewMemoryDatabase(), nil
}

// createPostgreSQLDatabase creates a new PostgreSQL database instance
func (f *Factory) createPostgreSQLDatabase(config *Config) (Database, error) {
	if config.DSN == "" {
		return nil, &DatabaseError{
			Type:    "INVALID_CONFIG",
			Message: "PostgreSQL DSN cannot be empty",
		}
	}

	db, err := NewPostgreSQLDatabase(config.DSN)
	if err != nil {
		return nil, &DatabaseError{
			Type:    "CONNECTION_ERROR",
			Message: "failed to create PostgreSQL database",
			Err:     err,
		}
	}

	return db, nil
}

// createMySQLDatabase creates a new MySQL database instance
func (f *Factory) createMySQLDatabase(config *Config) (Database, error) {
	// TODO: Implement MySQL database
	return nil, &DatabaseError{
		Type:    "NOT_IMPLEMENTED",
		Message: "MySQL database not yet implemented",
	}
}

// DefaultConfig returns a default database configuration for development
func DefaultConfig() *Config {
	return &Config{
		Type: DatabaseTypeMemory,
		DSN:  "",
	}
}

// PostgreSQLConfig returns a PostgreSQL configuration template
func PostgreSQLConfig(dsn string) *Config {
	return &Config{
		Type: DatabaseTypePostgreSQL,
		DSN:  dsn,
	}
}

// MySQLConfig returns a MySQL configuration template
func MySQLConfig(dsn string) *Config {
	return &Config{
		Type: DatabaseTypeMySQL,
		DSN:  dsn,
	}
}
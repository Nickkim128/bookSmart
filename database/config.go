package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// Config holds database configuration
type Config struct {
	Host         string
	Port         string
	User         string
	Password     string
	DatabaseName string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
	MaxLifetime  time.Duration
}

// LoadConfigFromEnv loads database configuration from environment variables
func LoadConfigFromEnv() *Config {
	return &Config{
		Host:         getEnvOrDefault("DB_HOST", "localhost"),
		Port:         getEnvOrDefault("DB_PORT", "5432"),
		User:         getEnvOrDefault("DB_USER", "postgres"),
		Password:     getEnvOrDefault("DB_PASSWORD", ""),
		DatabaseName: getEnvOrDefault("DB_NAME", "scheduler"),
		SSLMode:      getEnvOrDefault("DB_SSL_MODE", "prefer"),
		MaxOpenConns: 25,
		MaxIdleConns: 5,
		MaxLifetime:  time.Hour,
	}
}

// LoadNeonConfig loads configuration for Neon database from environment
func LoadNeonConfig() *Config {
	// Neon typically provides a single DATABASE_URL
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL != "" {
		return &Config{
			Host:         "", // Will use DATABASE_URL directly
			Port:         "",
			User:         "",
			Password:     "",
			DatabaseName: "",
			SSLMode:      "require", // Neon requires SSL
			MaxOpenConns: 20,        // Neon has connection limits
			MaxIdleConns: 2,
			MaxLifetime:  time.Minute * 30,
		}
	}

	// Fallback to individual environment variables
	return &Config{
		Host:         getEnvOrDefault("NEON_HOST", ""),
		Port:         getEnvOrDefault("NEON_PORT", "5432"),
		User:         getEnvOrDefault("NEON_USER", ""),
		Password:     getEnvOrDefault("NEON_PASSWORD", ""),
		DatabaseName: getEnvOrDefault("NEON_DATABASE", ""),
		SSLMode:      "require",
		MaxOpenConns: 20,
		MaxIdleConns: 2,
		MaxLifetime:  time.Minute * 30,
	}
}

// ConnectionString returns the PostgreSQL connection string
func (c *Config) ConnectionString() string {
	// If we have a DATABASE_URL (common for Neon), use it directly
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		return databaseURL
	}

	// Build connection string from individual components
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DatabaseName, c.SSLMode,
	)
}

// Connect establishes a connection to the database
func Connect(config *Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", config.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.MaxLifetime)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("Failed to close database connection: %v", closeErr)
		}
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Successfully connected to database")
	return db, nil
}

// Migrate runs database migrations
func Migrate(db *sql.DB, migrationsDir string) error {
	// Create migrations table if it doesn't exist
	createMigrationsTable := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ DEFAULT NOW()
		);
	`

	if _, err := db.Exec(createMigrationsTable); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// For now, we'll implement a simple migration system
	// In production, consider using a proper migration library like golang-migrate
	migrations := []struct {
		version string
		file    string
	}{
		{"001", "001_create_tables.sql"},
		{"002", "002_create_indexes.sql"},
		{"003", "003_sample_data.sql"},
		{"004", "004_add_firebase_auth.sql"},
	}

	for _, migration := range migrations {
		// Check if migration was already applied
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = $1", migration.version).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check migration status: %w", err)
		}

		if count > 0 {
			log.Printf("Migration %s already applied, skipping", migration.version)
			continue
		}

		// Read migration file
		migrationPath := fmt.Sprintf("%s/%s", migrationsDir, migration.file)
		migrationSQL, err := os.ReadFile(migrationPath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", migrationPath, err)
		}

		// Execute migration in a transaction
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %s: %w", migration.version, err)
		}

		if _, err := tx.Exec(string(migrationSQL)); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Failed to rollback transaction: %v", rollbackErr)
			}
			return fmt.Errorf("failed to execute migration %s: %w", migration.version, err)
		}

		// Record migration as applied
		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", migration.version); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Failed to rollback transaction: %v", rollbackErr)
			}
			return fmt.Errorf("failed to record migration %s: %w", migration.version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", migration.version, err)
		}

		log.Printf("Successfully applied migration %s", migration.version)
	}

	return nil
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

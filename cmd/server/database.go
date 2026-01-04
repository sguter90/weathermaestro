package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/sguter90/weathermaestro/pkg/migrations"
)

// connectDatabase establishes a connection to the PostgreSQL database
func connectDatabase() (*sql.DB, error) {
	// Get database configuration from environment variables with defaults
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "weather_user")
	password := getEnv("DB_PASSWORD", "weather_pass")
	dbname := getEnv("DB_NAME", "weather_db")
	sslmode := getEnv("DB_SSLMODE", "disable")

	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode,
	)

	// Open database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	log.Printf("Successfully connected to database: %s@%s:%s/%s", user, host, port, dbname)

	return db, nil
}

// initDatabase runs database migrations
func initDatabase(db *sql.DB) error {
	log.Println("Running database migrations...")

	// Create migration runner
	runner, err := migrations.NewRunner(db)
	if err != nil {
		return fmt.Errorf("failed to create migration runner: %w", err)
	}

	// Run all pending migrations
	if err := runner.Run(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Database initialization completed successfully")
	return nil
}

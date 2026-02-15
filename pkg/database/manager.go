package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// DatabaseManager handles all database operations
type DatabaseManager struct {
	db            *sql.DB
	healthChecker *HealthChecker
}

// NewDatabaseManager creates a new DatabaseManager instance
func NewDatabaseManager() (*DatabaseManager, error) {
	db, err := connectDatabase()
	if err != nil {
		return nil, err
	}
	dm := &DatabaseManager{
		db:            db,
		healthChecker: NewHealthChecker(db, 30*time.Second),
	}

	// Start health checking
	dm.healthChecker.Start()

	return dm, nil
}

// GetDB returns the underlying database connection
func (dm *DatabaseManager) GetDB() *sql.DB {
	return dm.db
}

// Close closes the database connection and stops health checking
func (dm *DatabaseManager) Close() error {
	if dm.healthChecker != nil {
		dm.healthChecker.Stop()
	}
	if dm.db != nil {
		return dm.db.Close()
	}
	return nil
}

// QueryWithHealthCheck executes a query with connection health verification
func (dm *DatabaseManager) QueryWithHealthCheck(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if err := dm.healthChecker.EnsureConnection(ctx); err != nil {
		return nil, err
	}

	return dm.db.QueryContext(ctx, query, args...)
}

// QueryRowWithHealthCheck executes a query that returns a single row with health check
func (dm *DatabaseManager) QueryRowWithHealthCheck(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if err := dm.healthChecker.EnsureConnection(ctx); err != nil {
		// Return a row that will fail on scan
		return dm.db.QueryRowContext(context.Background(), "SELECT NULL WHERE FALSE")
	}

	return dm.db.QueryRowContext(ctx, query, args...)
}

// ExecWithHealthCheck executes a statement with connection health verification
func (dm *DatabaseManager) ExecWithHealthCheck(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if err := dm.healthChecker.EnsureConnection(ctx); err != nil {
		return nil, err
	}

	return dm.db.ExecContext(ctx, query, args...)
}

// IsConnectionHealthy returns the current health status
func (dm *DatabaseManager) IsConnectionHealthy() bool {
	return dm.healthChecker.IsHealthy()
}

// Init initializes the database with migrations
func (dm *DatabaseManager) Init() error {
	log.Println("Running database migrations...")

	runner, err := NewMigrationsRunner(dm.db)
	if err != nil {
		return fmt.Errorf("failed to create migration runner: %w", err)
	}

	if err := runner.Run(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("✓ Database initialization completed successfully")
	return nil
}

// connectDatabase establishes a connection to the database
func connectDatabase() (*sql.DB, error) {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "weather_user")
	password := getEnv("DB_PASSWORD", "weather_pass")
	dbName := getEnv("DB_NAME", "weather_db")
	sslmode := getEnv("DB_SSLMODE", "disable")

	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbName, sslmode,
	)

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

	return db, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

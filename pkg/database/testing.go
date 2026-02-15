package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// FOR TESTING:
//   export TEST_DATABASE_URL="postgres://user:password@localhost:5432/weathermaestro_test?sslmode=disable"
//   go test ./pkg/database/...

// setupTestDatabaseManager creates a test database manager for integration tests
func setupTestDatabaseManager(t *testing.T) *DatabaseManager {
	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		return nil
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	// Drop all tables before running tests
	if err := dropAllTables(db); err != nil {
		t.Fatalf("Failed to drop tables: %v", err)
	}

	// Run migrations
	if err := runMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	dm := &DatabaseManager{
		db:            db,
		healthChecker: NewHealthChecker(db, 30*time.Second),
	}

	// Start health checking
	dm.healthChecker.Start()

	return dm
}

// dropAllTables drops all tables in the database
func dropAllTables(db *sql.DB) error {
	ctx := context.Background()

	// Get all table names
	query := `
        SELECT tablename 
        FROM pg_tables 
        WHERE schemaname = 'public'
    `

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	// Drop each table
	for _, table := range tables {
		dropQuery := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table)
		if _, err := db.ExecContext(ctx, dropQuery); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	return nil
}

// runMigrations runs all database migrations
func runMigrations(db *sql.DB) error {
	runner, err := NewMigrationsRunner(db)
	if err != nil {
		return fmt.Errorf("failed to create migrations runner: %w", err)
	}

	runner.DisableLogging()

	if err := runner.Run(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// setupTestDB creates a test database connection (for simpler tests)
func setupTestDB(t *testing.T) *sql.DB {
	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		return nil
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	return db
}

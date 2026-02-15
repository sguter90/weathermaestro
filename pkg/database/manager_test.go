package database

import (
	"context"
	"testing"
	"time"
)

func TestNewDatabaseManager(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	if dm.db == nil {
		t.Error("Expected database connection to be initialized")
	}

	if dm.healthChecker == nil {
		t.Error("Expected health checker to be initialized")
	}

	if !dm.IsConnectionHealthy() {
		t.Error("Expected database connection to be healthy")
	}
}

func TestGetDB(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	db := dm.GetDB()
	if db == nil {
		t.Error("Expected GetDB to return non-nil database connection")
	}

	// Verify we can use the returned connection
	err := db.Ping()
	if err != nil {
		t.Errorf("Expected to ping database successfully: %v", err)
	}
}

func TestClose(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}

	err := dm.Close()
	if err != nil {
		t.Errorf("Expected Close to succeed: %v", err)
	}

	// Verify connection is closed
	err = dm.db.Ping()
	if err == nil {
		t.Error("Expected database connection to be closed")
	}
}

func TestQueryWithHealthCheck(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()

	// Test a simple query
	rows, err := dm.QueryWithHealthCheck(ctx, "SELECT 1 as num")
	if err != nil {
		t.Fatalf("Expected query to succeed: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Error("Expected at least one row")
	}

	var num int
	err = rows.Scan(&num)
	if err != nil {
		t.Errorf("Expected to scan result: %v", err)
	}

	if num != 1 {
		t.Errorf("Expected num=1, got %d", num)
	}
}

func TestQueryRowWithHealthCheck(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()

	// Test a simple query
	var num int
	err := dm.QueryRowWithHealthCheck(ctx, "SELECT 1 as num").Scan(&num)
	if err != nil {
		t.Fatalf("Expected query to succeed: %v", err)
	}

	if num != 1 {
		t.Errorf("Expected num=1, got %d", num)
	}
}

func TestExecWithHealthCheck(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()

	// Create a temporary table
	_, err := dm.ExecWithHealthCheck(ctx, `
        CREATE TEMPORARY TABLE test_exec (
            id SERIAL PRIMARY KEY,
            name TEXT NOT NULL
        )
    `)
	if err != nil {
		t.Fatalf("Expected table creation to succeed: %v", err)
	}

	// Insert data
	result, err := dm.ExecWithHealthCheck(ctx, "INSERT INTO test_exec (name) VALUES ($1)", "test_name")
	if err != nil {
		t.Fatalf("Expected insert to succeed: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		t.Errorf("Expected to get rows affected: %v", err)
	}

	if rowsAffected != 1 {
		t.Errorf("Expected 1 row affected, got %d", rowsAffected)
	}
}

func TestIsConnectionHealthy(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	if !dm.IsConnectionHealthy() {
		t.Error("Expected connection to be healthy")
	}
}

func TestInit(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	// Note: setupTestDatabaseManager already runs migrations,
	// but calling Init again should be idempotent
	err := dm.Init()
	if err != nil {
		t.Errorf("Expected Init to succeed: %v", err)
	}

	// Verify migrations table exists
	var count int
	err = dm.QueryRowWithHealthCheck(context.Background(),
		"SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if err != nil {
		t.Errorf("Expected to query migrations table: %v", err)
	}

	if count == 0 {
		t.Error("Expected at least one migration to be applied")
	}
}

func TestQueryWithHealthCheck_ContextCancellation(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := dm.QueryWithHealthCheck(ctx, "SELECT 1")
	if err == nil {
		t.Error("Expected error due to cancelled context")
	}
}

func TestQueryWithHealthCheck_ContextTimeout(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	time.Sleep(10 * time.Millisecond) // Ensure timeout has passed

	_, err := dm.QueryWithHealthCheck(ctx, "SELECT pg_sleep(1)")
	if err == nil {
		t.Error("Expected error due to context timeout")
	}
}

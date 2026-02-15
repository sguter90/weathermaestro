package database

import (
	"strings"
	"testing"
)

func TestNewMigrationsRunner(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer db.Close()

	runner, err := NewMigrationsRunner(db)
	if err != nil {
		t.Fatalf("Expected NewMigrationsRunner to succeed: %v", err)
	}

	if runner.db == nil {
		t.Error("Expected database connection to be set")
	}

	if runner.logger == nil {
		t.Error("Expected logger to be initialized")
	}

	if len(runner.migrations) == 0 {
		t.Error("Expected migrations to be loaded")
	}
}

func TestLoadMigrations(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer db.Close()

	runner, err := NewMigrationsRunner(db)
	if err != nil {
		t.Fatalf("Expected NewMigrationsRunner to succeed: %v", err)
	}

	if len(runner.migrations) == 0 {
		t.Error("Expected at least one migration to be loaded")
	}

	// Verify migrations are sorted by version
	for i := 1; i < len(runner.migrations); i++ {
		if runner.migrations[i-1].Version >= runner.migrations[i].Version {
			t.Errorf("Expected migrations to be sorted by version, but %d >= %d",
				runner.migrations[i-1].Version, runner.migrations[i].Version)
		}
	}

	// Verify each migration has required fields
	for _, migration := range runner.migrations {
		if migration.Version == 0 {
			t.Error("Expected migration version to be non-zero")
		}
		if migration.Name == "" {
			t.Error("Expected migration name to be non-empty")
		}
		if migration.SQL == "" {
			t.Error("Expected migration SQL to be non-empty")
		}
	}
}

func TestEnableDisableLogging(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer db.Close()

	runner, err := NewMigrationsRunner(db)
	if err != nil {
		t.Fatalf("Expected NewMigrationsRunner to succeed: %v", err)
	}

	// Test DisableLogging
	runner.DisableLogging()
	// Logger output should be set to io.Discard, but we can't easily verify this
	// Just ensure it doesn't panic

	// Test EnableLogging
	runner.EnableLogging()
	// Logger output should be restored, but we can't easily verify this
	// Just ensure it doesn't panic
}

func TestCreateMigrationsTable(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer db.Close()

	runner, err := NewMigrationsRunner(db)
	if err != nil {
		t.Fatalf("Expected NewMigrationsRunner to succeed: %v", err)
	}

	err = runner.createMigrationsTable()
	if err != nil {
		t.Fatalf("Expected createMigrationsTable to succeed: %v", err)
	}

	// Verify table exists
	var exists bool
	err = db.QueryRow(`
        SELECT EXISTS (
            SELECT FROM information_schema.tables 
            WHERE table_schema = 'public' 
            AND table_name = 'schema_migrations'
        )
    `).Scan(&exists)
	if err != nil {
		t.Fatalf("Expected to query table existence: %v", err)
	}

	if !exists {
		t.Error("Expected schema_migrations table to exist")
	}

	// Calling it again should be idempotent
	err = runner.createMigrationsTable()
	if err != nil {
		t.Error("Expected createMigrationsTable to be idempotent")
	}
}

func TestGetAppliedMigrations(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer db.Close()

	dropAllTables(db)

	runner, err := NewMigrationsRunner(db)
	if err != nil {
		t.Fatalf("Expected NewMigrationsRunner to succeed: %v", err)
	}

	err = runner.createMigrationsTable()
	if err != nil {
		t.Fatalf("Expected createMigrationsTable to succeed: %v", err)
	}

	// Initially should be empty
	applied, err := runner.getAppliedMigrations()
	if err != nil {
		t.Fatalf("Expected getAppliedMigrations to succeed: %v", err)
	}

	if len(applied) != 0 {
		t.Errorf("Expected no applied migrations, got %d", len(applied))
	}

	// Insert a test migration
	_, err = db.Exec("INSERT INTO schema_migrations (version, name) VALUES ($1, $2)", 1, "test_migration")
	if err != nil {
		t.Fatalf("Expected to insert test migration: %v", err)
	}

	// Now should have one
	applied, err = runner.getAppliedMigrations()
	if err != nil {
		t.Fatalf("Expected getAppliedMigrations to succeed: %v", err)
	}

	if len(applied) != 1 {
		t.Errorf("Expected 1 applied migration, got %d", len(applied))
	}

	if !applied[1] {
		t.Error("Expected migration version 1 to be marked as applied")
	}
}

func TestRun(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer db.Close()

	// Drop all tables first
	if err := dropAllTables(db); err != nil {
		t.Fatalf("Failed to drop tables: %v", err)
	}

	runner, err := NewMigrationsRunner(db)
	if err != nil {
		t.Fatalf("Expected NewMigrationsRunner to succeed: %v", err)
	}

	runner.DisableLogging()

	err = runner.Run()
	if err != nil {
		t.Fatalf("Expected Run to succeed: %v", err)
	}

	// Verify migrations were applied
	applied, err := runner.getAppliedMigrations()
	if err != nil {
		t.Fatalf("Expected getAppliedMigrations to succeed: %v", err)
	}

	if len(applied) == 0 {
		t.Error("Expected at least one migration to be applied")
	}

	// Verify all migrations were applied
	for _, migration := range runner.migrations {
		if !applied[migration.Version] {
			t.Errorf("Expected migration %d to be applied", migration.Version)
		}
	}
}

func TestRun_Idempotent(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer db.Close()

	// Drop all tables first
	if err := dropAllTables(db); err != nil {
		t.Fatalf("Failed to drop tables: %v", err)
	}

	runner, err := NewMigrationsRunner(db)
	if err != nil {
		t.Fatalf("Expected NewMigrationsRunner to succeed: %v", err)
	}

	runner.DisableLogging()

	// Run migrations first time
	err = runner.Run()
	if err != nil {
		t.Fatalf("Expected first Run to succeed: %v", err)
	}

	// Run migrations second time - should be idempotent
	err = runner.Run()
	if err != nil {
		t.Fatalf("Expected second Run to succeed: %v", err)
	}

	// Verify count didn't change
	applied, err := runner.getAppliedMigrations()
	if err != nil {
		t.Fatalf("Expected getAppliedMigrations to succeed: %v", err)
	}

	if len(applied) != len(runner.migrations) {
		t.Errorf("Expected %d migrations, got %d", len(runner.migrations), len(applied))
	}
}

func TestRun_TransactionRollback(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer db.Close()

	// Drop all tables first
	if err := dropAllTables(db); err != nil {
		t.Fatalf("Failed to drop tables: %v", err)
	}

	runner, err := NewMigrationsRunner(db)
	if err != nil {
		t.Fatalf("Expected NewMigrationsRunner to succeed: %v", err)
	}

	runner.DisableLogging()

	// Add a migration with invalid SQL
	runner.migrations = append(runner.migrations, Migration{
		Version: 99999,
		Name:    "invalid_migration",
		SQL:     "THIS IS INVALID SQL;",
	})

	err = runner.Run()
	if err == nil {
		t.Error("Expected Run to fail with invalid SQL")
	}

	if !strings.Contains(err.Error(), "failed to apply migration") {
		t.Errorf("Expected error message to contain 'failed to apply migration', got: %v", err)
	}

	// Verify the invalid migration was not recorded
	applied, err := runner.getAppliedMigrations()
	if err != nil {
		t.Fatalf("Expected getAppliedMigrations to succeed: %v", err)
	}

	if applied[99999] {
		t.Error("Expected invalid migration to not be recorded")
	}
}

func TestMigrationStructure(t *testing.T) {
	migration := Migration{
		Version: 1,
		Name:    "test_migration",
		SQL:     "CREATE TABLE test (id INT);",
	}

	if migration.Version != 1 {
		t.Errorf("Expected Version=1, got %d", migration.Version)
	}

	if migration.Name != "test_migration" {
		t.Errorf("Expected Name='test_migration', got %s", migration.Name)
	}

	if migration.SQL != "CREATE TABLE test (id INT);" {
		t.Errorf("Expected SQL to match, got %s", migration.SQL)
	}
}

func TestMigrationFilesEmbedded(t *testing.T) {
	// Verify that migration files are embedded
	entries, err := migrationFiles.ReadDir("sql")
	if err != nil {
		t.Fatalf("Expected to read embedded migration directory: %v", err)
	}

	if len(entries) == 0 {
		t.Error("Expected at least one migration file to be embedded")
	}

	// Verify we can read a migration file
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			content, err := migrationFiles.ReadFile("sql/" + entry.Name())
			if err != nil {
				t.Errorf("Expected to read migration file %s: %v", entry.Name(), err)
			}
			if len(content) == 0 {
				t.Errorf("Expected migration file %s to have content", entry.Name())
			}
			break
		}
	}
}

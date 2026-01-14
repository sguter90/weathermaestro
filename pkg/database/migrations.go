package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"sort"
	"strings"
)

//go:embed sql/*.sql
var migrationFiles embed.FS

// Migration represents a single database migration
type Migration struct {
	Version int
	Name    string
	SQL     string
}

// MigrationsRunner handles database migrations
type MigrationsRunner struct {
	db         *sql.DB
	migrations []Migration
}

// NewMigrationsRunner creates a new migration runner
func NewMigrationsRunner(db *sql.DB) (*MigrationsRunner, error) {
	runner := &MigrationsRunner{
		db:         db,
		migrations: []Migration{},
	}

	if err := runner.loadMigrations(); err != nil {
		return nil, fmt.Errorf("failed to load migrations: %w", err)
	}

	return runner, nil
}

// loadMigrations loads all .up.sql migration files from the embedded filesystem
func (r *MigrationsRunner) loadMigrations() error {
	// Read from the sql subdirectory
	entries, err := migrationFiles.ReadDir("sql")
	if err != nil {
		return fmt.Errorf("failed to read migration directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()

		// Only process .up.sql files
		if !strings.HasSuffix(filename, ".up.sql") {
			continue
		}

		// ParseWeatherData filename: 000001_name.up.sql
		parts := strings.Split(filename, "_")
		if len(parts) < 2 {
			continue
		}

		var version int
		if _, err := fmt.Sscanf(parts[0], "%d", &version); err != nil {
			log.Printf("Warning: skipping invalid migration file: %s", filename)
			continue
		}

		// Extract name from filename
		nameParts := parts[1:]
		name := strings.Join(nameParts, "_")
		name = strings.TrimSuffix(name, ".up.sql")

		// Read migration content from sql subdirectory
		content, err := migrationFiles.ReadFile("sql/" + filename)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		r.migrations = append(r.migrations, Migration{
			Version: version,
			Name:    name,
			SQL:     string(content),
		})
	}

	// Sort migrations by version
	sort.Slice(r.migrations, func(i, j int) bool {
		return r.migrations[i].Version < r.migrations[j].Version
	})

	return nil
}

// createMigrationsTable creates the schema_migrations table if it doesn't exist
func (r *MigrationsRunner) createMigrationsTable() error {
	query := `
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version INTEGER PRIMARY KEY,
            name VARCHAR(255) NOT NULL,
            applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
        )
    `
	_, err := r.db.Exec(query)
	return err
}

// getAppliedMigrations returns a set of applied migration versions
func (r *MigrationsRunner) getAppliedMigrations() (map[int]bool, error) {
	applied := make(map[int]bool)

	rows, err := r.db.Query("SELECT version FROM schema_migrations ORDER BY version")
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

	return applied, nil
}

// Run executes all pending migrations
func (r *MigrationsRunner) Run() error {
	if err := r.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	applied, err := r.getAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	pendingCount := 0
	for _, migration := range r.migrations {
		if !applied[migration.Version] {
			pendingCount++
		}
	}

	if pendingCount == 0 {
		log.Println("No pending migrations")
		return nil
	}

	log.Printf("Found %d pending migration(s)", pendingCount)

	for _, migration := range r.migrations {
		if applied[migration.Version] {
			continue
		}

		log.Printf("Applying migration %d: %s", migration.Version, migration.Name)

		// Start transaction
		tx, err := r.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}

		// Execute migration
		if _, err := tx.Exec(migration.SQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to apply migration %d (%s): %w", migration.Version, migration.Name, err)
		}

		// Record migration
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

		log.Printf("✓ Successfully applied migration %d: %s", migration.Version, migration.Name)
	}

	log.Println("All migrations completed successfully")
	return nil
}

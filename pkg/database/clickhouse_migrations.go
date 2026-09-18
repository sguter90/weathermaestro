package database

import (
	"context"
	"embed"
	"fmt"
	"io"
	"log"
	"sort"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

//go:embed sql_clickhouse/*.up.sql
var clickHouseMigrationFiles embed.FS

// ClickHouseMigration represents a single ClickHouse migration.
type ClickHouseMigration struct {
	Version int
	Name    string
	SQL     string
}

// ClickHouseMigrationsRunner applies versioned schema changes to ClickHouse.
// ClickHouse does not support multi-statement DDL transactions the way Postgres
// does, so each migration file may contain multiple statements separated by `;`
// which are executed sequentially. If a statement fails the version is not
// recorded and the runner returns an error; re-running will retry from the
// failed migration onwards.
type ClickHouseMigrationsRunner struct {
	conn       driver.Conn
	migrations []ClickHouseMigration
	logger     *log.Logger
}

// NewClickHouseMigrationsRunner creates a new ClickHouse migration runner.
func NewClickHouseMigrationsRunner(conn driver.Conn) (*ClickHouseMigrationsRunner, error) {
	runner := &ClickHouseMigrationsRunner{
		conn:       conn,
		migrations: []ClickHouseMigration{},
		logger:     log.Default(),
	}

	if err := runner.loadMigrations(); err != nil {
		return nil, fmt.Errorf("failed to load clickhouse migrations: %w", err)
	}

	return runner, nil
}

// EnableLogging enables migration logging.
func (r *ClickHouseMigrationsRunner) EnableLogging() {
	r.logger.SetOutput(log.Writer())
}

// DisableLogging silences migration logging.
func (r *ClickHouseMigrationsRunner) DisableLogging() {
	r.logger.SetOutput(io.Discard)
}

func (r *ClickHouseMigrationsRunner) loadMigrations() error {
	entries, err := clickHouseMigrationFiles.ReadDir("sql_clickhouse")
	if err != nil {
		return fmt.Errorf("failed to read clickhouse migration directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		if !strings.HasSuffix(filename, ".up.sql") {
			continue
		}

		parts := strings.Split(filename, "_")
		if len(parts) < 2 {
			continue
		}

		var version int
		if _, err := fmt.Sscanf(parts[0], "%d", &version); err != nil {
			r.logger.Printf("Warning: skipping invalid clickhouse migration file: %s", filename)
			continue
		}

		name := strings.TrimSuffix(strings.Join(parts[1:], "_"), ".up.sql")

		content, err := clickHouseMigrationFiles.ReadFile("sql_clickhouse/" + filename)
		if err != nil {
			return fmt.Errorf("failed to read clickhouse migration file %s: %w", filename, err)
		}

		r.migrations = append(r.migrations, ClickHouseMigration{
			Version: version,
			Name:    name,
			SQL:     string(content),
		})
	}

	sort.Slice(r.migrations, func(i, j int) bool {
		return r.migrations[i].Version < r.migrations[j].Version
	})

	return nil
}

func (r *ClickHouseMigrationsRunner) createMigrationsTable(ctx context.Context) error {
	const ddl = `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    UInt32,
			name       String,
			applied_at DateTime DEFAULT now()
		) ENGINE = TinyLog
	`
	return r.conn.Exec(ctx, ddl)
}

func (r *ClickHouseMigrationsRunner) getAppliedMigrations(ctx context.Context) (map[int]bool, error) {
	applied := make(map[int]bool)

	rows, err := r.conn.Query(ctx, "SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var version uint32
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[int(version)] = true
	}

	return applied, rows.Err()
}

// Run executes all pending ClickHouse migrations in ascending version order.
func (r *ClickHouseMigrationsRunner) Run(ctx context.Context) error {
	if err := r.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create clickhouse migrations table: %w", err)
	}

	applied, err := r.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied clickhouse migrations: %w", err)
	}

	pendingCount := 0
	for _, migration := range r.migrations {
		if !applied[migration.Version] {
			pendingCount++
		}
	}

	if pendingCount == 0 {
		r.logger.Println("No pending ClickHouse migrations")
		return nil
	}

	r.logger.Printf("Found %d pending ClickHouse migration(s)", pendingCount)

	for _, migration := range r.migrations {
		if applied[migration.Version] {
			continue
		}

		r.logger.Printf("Applying ClickHouse migration %d: %s", migration.Version, migration.Name)

		for _, stmt := range splitSQLStatements(migration.SQL) {
			if err := r.conn.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("failed to apply clickhouse migration %d (%s): %w", migration.Version, migration.Name, err)
			}
		}

		if err := r.conn.Exec(ctx,
			"INSERT INTO schema_migrations (version, name) VALUES (?, ?)",
			uint32(migration.Version), migration.Name,
		); err != nil {
			return fmt.Errorf("failed to record clickhouse migration %d: %w", migration.Version, err)
		}

		r.logger.Printf("✓ Successfully applied ClickHouse migration %d: %s", migration.Version, migration.Name)
	}

	r.logger.Println("All ClickHouse migrations completed successfully")
	return nil
}

// splitSQLStatements splits a migration file into individual statements on `;`
// boundaries. Empty segments are dropped. Assumes migrations do not contain
// semicolons inside string literals — a fair assumption for schema DDL.
func splitSQLStatements(sql string) []string {
	parts := strings.Split(sql, ";")
	statements := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		statements = append(statements, trimmed)
	}
	return statements
}

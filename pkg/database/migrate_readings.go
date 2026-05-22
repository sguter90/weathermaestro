package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

const readingsMigrationBatchSize = 50000

// migrateSensorReadingsFromPostgres performs a one-time copy of sensor_readings
// rows from Postgres to ClickHouse and truncates the Postgres table on success.
//
// Idempotency: the trigger is "Postgres sensor_readings has rows". Once the
// migration succeeds the table is emptied, so subsequent calls are no-ops.
//
// Crash safety: before copying, the ClickHouse table is truncated. If the
// process is interrupted mid-copy, the next start finds Postgres still
// populated and restarts from a clean slate in ClickHouse. The worst case is
// repeated work, not data loss or duplicates.
func (dm *DatabaseManager) migrateSensorReadingsFromPostgres() error {
	ctx := context.Background()

	exists, err := postgresTableExists(ctx, dm.db, "sensor_readings")
	if err != nil {
		return fmt.Errorf("failed to check sensor_readings existence: %w", err)
	}
	if !exists {
		return nil
	}

	var total int64
	if err := dm.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sensor_readings").Scan(&total); err != nil {
		return fmt.Errorf("failed to count sensor_readings: %w", err)
	}
	if total == 0 {
		return nil
	}

	log.Printf("Migrating %d sensor_readings from Postgres to ClickHouse...", total)
	start := time.Now()

	// Clean slate in CH so the migration is safely restartable.
	if err := dm.ch.Conn().Exec(ctx, "TRUNCATE TABLE sensor_readings"); err != nil {
		return fmt.Errorf("failed to truncate clickhouse sensor_readings: %w", err)
	}

	rows, err := dm.db.QueryContext(ctx,
		"SELECT id, sensor_id, value, date_utc FROM sensor_readings ORDER BY date_utc ASC")
	if err != nil {
		return fmt.Errorf("failed to query sensor_readings: %w", err)
	}
	defer rows.Close()

	const insertStmt = "INSERT INTO sensor_readings (id, sensor_id, value, date_utc)"
	batch, err := dm.ch.Conn().PrepareBatch(ctx, insertStmt)
	if err != nil {
		return fmt.Errorf("failed to prepare batch: %w", err)
	}

	var migrated int64
	for rows.Next() {
		var (
			id       uuid.UUID
			sensorID uuid.UUID
			value    float64
			dateUTC  time.Time
		)
		if err := rows.Scan(&id, &sensorID, &value, &dateUTC); err != nil {
			return fmt.Errorf("failed to scan row %d: %w", migrated, err)
		}
		if err := batch.Append(id, sensorID, value, dateUTC.UTC()); err != nil {
			return fmt.Errorf("failed to append row %d: %w", migrated, err)
		}
		migrated++

		if migrated%readingsMigrationBatchSize == 0 {
			if err := batch.Send(); err != nil {
				return fmt.Errorf("failed to send batch ending at row %d: %w", migrated, err)
			}
			batch, err = dm.ch.Conn().PrepareBatch(ctx, insertStmt)
			if err != nil {
				return fmt.Errorf("failed to prepare next batch: %w", err)
			}
			log.Printf("  ...migrated %d/%d (%.0f%%)", migrated, total, float64(migrated)/float64(total)*100)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("row iteration failed: %w", err)
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("failed to send final batch: %w", err)
	}

	if migrated != total {
		return fmt.Errorf("migrated count %d != expected %d", migrated, total)
	}

	if _, err := dm.db.ExecContext(ctx, "TRUNCATE TABLE sensor_readings"); err != nil {
		return fmt.Errorf("failed to truncate postgres sensor_readings: %w", err)
	}

	log.Printf("✓ Migrated %d rows in %s. Postgres sensor_readings truncated.", migrated, time.Since(start))
	return nil
}

// postgresTableExists returns true if a table with the given name exists in the
// public schema. After Phase 5 drops the sensor_readings table this will return
// false and the migration becomes a permanent no-op.
func postgresTableExists(ctx context.Context, db *sql.DB, name string) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = $1
		)`, name).Scan(&exists)
	return exists, err
}

package database

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ClickHouseManager handles the ClickHouse connection used for sensor readings.
type ClickHouseManager struct {
	conn driver.Conn
}

// NewClickHouseManager establishes a connection to ClickHouse and ensures the schema exists.
func NewClickHouseManager() (*ClickHouseManager, error) {
	conn, err := connectClickHouse()
	if err != nil {
		return nil, err
	}

	cm := &ClickHouseManager{conn: conn}

	if err := cm.ensureSchema(context.Background()); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to ensure clickhouse schema: %w", err)
	}

	return cm, nil
}

// Conn exposes the underlying ClickHouse connection for use by other database methods.
func (cm *ClickHouseManager) Conn() driver.Conn {
	return cm.conn
}

// Ping verifies the ClickHouse connection is alive.
func (cm *ClickHouseManager) Ping(ctx context.Context) error {
	return cm.conn.Ping(ctx)
}

// Close terminates the ClickHouse connection.
func (cm *ClickHouseManager) Close() error {
	if cm.conn == nil {
		return nil
	}
	return cm.conn.Close()
}

// ensureSchema creates the sensor_readings table if it does not already exist.
func (cm *ClickHouseManager) ensureSchema(ctx context.Context) error {
	const ddl = `
		CREATE TABLE IF NOT EXISTS sensor_readings (
			id         UUID DEFAULT generateUUIDv4(),
			sensor_id  UUID,
			value      Float64,
			date_utc   DateTime64(3, 'UTC'),
			created_at DateTime DEFAULT now()
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(date_utc)
		ORDER BY (sensor_id, date_utc)
	`
	return cm.conn.Exec(ctx, ddl)
}

func connectClickHouse() (driver.Conn, error) {
	host := getEnv("CH_HOST", "localhost")
	port := getEnv("CH_PORT", "9000")
	user := getEnv("CH_USER", "weather")
	password := getEnv("CH_PASSWORD", "weather")
	database := getEnv("CH_DATABASE", "weather")

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%s", host, port)},
		Auth: clickhouse.Auth{
			Database: database,
			Username: user,
			Password: password,
		},
		Settings: clickhouse.Settings{
			// Server-side buffering of small inserts. Avoids the MergeTree
			// small-parts problem when many sensors push one row at a time.
			"async_insert":          1,
			"wait_for_async_insert": 0,
		},
		DialTimeout:     5 * time.Second,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open clickhouse connection: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := conn.Ping(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to ping clickhouse: %w", err)
	}

	return conn, nil
}

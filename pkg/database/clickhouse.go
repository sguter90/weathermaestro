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

// NewClickHouseManager establishes a connection to ClickHouse and applies any pending schema migrations.
func NewClickHouseManager() (*ClickHouseManager, error) {
	conn, err := connectClickHouse()
	if err != nil {
		return nil, err
	}

	cm := &ClickHouseManager{conn: conn}

	runner, err := NewClickHouseMigrationsRunner(conn)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to create clickhouse migrations runner: %w", err)
	}
	if err := runner.Run(context.Background()); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to run clickhouse migrations: %w", err)
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

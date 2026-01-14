package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"
)

// HealthChecker monitors and maintains database connection health
type HealthChecker struct {
	db            *sql.DB
	checkInterval time.Duration
	stopChan      chan struct{}
	ticker        *time.Ticker
	mu            sync.RWMutex
	isHealthy     bool
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(db *sql.DB, checkInterval time.Duration) *HealthChecker {
	return &HealthChecker{
		db:            db,
		checkInterval: checkInterval,
		stopChan:      make(chan struct{}),
		isHealthy:     true,
	}
}

// Start begins monitoring the database connection
func (chc *HealthChecker) Start() {
	chc.ticker = time.NewTicker(chc.checkInterval)

	go func() {
		for {
			select {
			case <-chc.stopChan:
				chc.ticker.Stop()
				return
			case <-chc.ticker.C:
				chc.checkConnection()
			}
		}
	}()
}

// Stop stops monitoring the database connection
func (chc *HealthChecker) Stop() {
	close(chc.stopChan)
}

// checkConnection performs a health check on the database connection
func (chc *HealthChecker) checkConnection() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := chc.db.PingContext(ctx)

	chc.mu.Lock()
	defer chc.mu.Unlock()

	if err != nil {
		log.Printf("❌ Database connection health check failed: %v", err)
		chc.isHealthy = false

		// Attempt to reconnect
		if err := chc.reconnect(); err != nil {
			log.Printf("❌ Failed to reconnect to database: %v", err)
		}
	} else {
		if !chc.isHealthy {
			log.Println("✓ Database connection restored")
		}
		chc.isHealthy = true
	}
}

// reconnect attempts to re-establish the database connection
func (chc *HealthChecker) reconnect() error {
	// Close existing connection
	if chc.db != nil {
		chc.db.Close()
	}

	// Attempt to create new connection
	newDB, err := connectDatabase()
	if err != nil {
		return err
	}

	chc.db = newDB
	log.Println("✓ Database connection re-established")
	return nil
}

// IsHealthy returns the current health status of the connection
func (chc *HealthChecker) IsHealthy() bool {
	chc.mu.RLock()
	defer chc.mu.RUnlock()
	return chc.isHealthy
}

// EnsureConnection ensures the connection is healthy before executing a query
func (chc *HealthChecker) EnsureConnection(ctx context.Context) error {
	chc.mu.RLock()
	isHealthy := chc.isHealthy
	chc.mu.RUnlock()

	if !isHealthy {
		return fmt.Errorf("database connection is not healthy")
	}

	// Perform a quick ping to verify
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := chc.db.PingContext(pingCtx); err != nil {
		chc.mu.Lock()
		chc.isHealthy = false
		chc.mu.Unlock()
		return fmt.Errorf("database connection check failed: %w", err)
	}

	return nil
}

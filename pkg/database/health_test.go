package database

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func TestNewHealthChecker(t *testing.T) {
	db := &sql.DB{}
	interval := 5 * time.Second

	hc := NewHealthChecker(db, interval)

	if hc == nil {
		t.Fatal("Expected HealthChecker instance, got nil")
	}

	if hc.db != db {
		t.Error("Expected db to be set correctly")
	}

	if hc.checkInterval != interval {
		t.Errorf("Expected checkInterval=%v, got %v", interval, hc.checkInterval)
	}

	if !hc.isHealthy {
		t.Error("Expected initial health status to be true")
	}

	if hc.stopChan == nil {
		t.Error("Expected stopChan to be initialized")
	}
}

func TestIsHealthy(t *testing.T) {
	db := &sql.DB{}
	hc := NewHealthChecker(db, 5*time.Second)

	if !hc.IsHealthy() {
		t.Error("Expected initial health status to be true")
	}

	hc.mu.Lock()
	hc.isHealthy = false
	hc.mu.Unlock()

	if hc.IsHealthy() {
		t.Error("Expected health status to be false after manual change")
	}

	hc.mu.Lock()
	hc.isHealthy = true
	hc.mu.Unlock()

	if !hc.IsHealthy() {
		t.Error("Expected health status to be true after manual change")
	}
}

func TestStartStop(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer db.Close()

	hc := NewHealthChecker(db, 100*time.Millisecond)

	hc.Start()

	if hc.ticker == nil {
		t.Error("Expected ticker to be initialized after Start()")
	}

	time.Sleep(150 * time.Millisecond)

	hc.Stop()

	// Verify stop channel is closed
	select {
	case <-hc.stopChan:
		// Channel is closed as expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected stopChan to be closed after Stop()")
	}
}

func TestStop_WithoutStart(t *testing.T) {
	db := &sql.DB{}
	hc := NewHealthChecker(db, 5*time.Second)

	// Should not panic when stopping without starting
	hc.Stop()

	select {
	case <-hc.stopChan:
		// Channel is closed as expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected stopChan to be closed after Stop()")
	}
}

func TestEnsureConnection_Unhealthy(t *testing.T) {
	db := &sql.DB{}
	hc := NewHealthChecker(db, 5*time.Second)

	hc.mu.Lock()
	hc.isHealthy = false
	hc.mu.Unlock()

	ctx := context.Background()
	err := hc.EnsureConnection(ctx)

	if err == nil {
		t.Error("Expected error when connection is unhealthy")
	}

	if err.Error() != "database connection is not healthy" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestEnsureConnection_Healthy(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer db.Close()

	hc := NewHealthChecker(db, 5*time.Second)

	ctx := context.Background()
	err := hc.EnsureConnection(ctx)

	if err != nil {
		t.Errorf("Expected no error for healthy connection, got: %v", err)
	}
}

func TestEnsureConnection_ContextCanceled(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer db.Close()

	hc := NewHealthChecker(db, 5*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := hc.EnsureConnection(ctx)

	if err == nil {
		t.Error("Expected error when context is canceled")
	}
}

func TestHealthChecker_ConcurrentAccess(t *testing.T) {
	db := &sql.DB{}
	hc := NewHealthChecker(db, 5*time.Second)

	done := make(chan bool)

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = hc.IsHealthy()
			}
			done <- true
		}()
	}

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(val bool) {
			for j := 0; j < 100; j++ {
				hc.mu.Lock()
				hc.isHealthy = val
				hc.mu.Unlock()
			}
			done <- true
		}(i%2 == 0)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}
}

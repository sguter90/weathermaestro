package database

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sguter90/weathermaestro/pkg/models"
)

func TestCreateDashboard(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	config := json.RawMessage(`{"layout": "grid", "widgets": []}`)

	dashboard := &models.Dashboard{
		Name:        "Test Dashboard",
		Description: "Test Description",
		Config:      config,
		IsDefault:   false,
	}

	err := dm.CreateDashboard(ctx, dashboard)
	if err != nil {
		t.Fatalf("Failed to create dashboard: %v", err)
	}

	if dashboard.ID == uuid.Nil {
		t.Error("Expected dashboard ID to be set")
	}

	if dashboard.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	if dashboard.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}
}

func TestGetDashboards(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()

	dashboards, err := dm.GetDashboards(ctx)
	if err != nil {
		t.Fatalf("Failed to get dashboards: %v", err)
	}

	if dashboards == nil {
		t.Error("Expected dashboards slice to be initialized")
	}
}

func TestGetDashboard(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	config := json.RawMessage(`{"layout": "grid"}`)

	// Create a dashboard first
	dashboard := &models.Dashboard{
		Name:        "Test Dashboard",
		Description: "Test Description",
		Config:      config,
		IsDefault:   false,
	}

	err := dm.CreateDashboard(ctx, dashboard)
	if err != nil {
		t.Fatalf("Failed to create dashboard: %v", err)
	}

	// Retrieve the dashboard
	retrieved, err := dm.GetDashboard(ctx, dashboard.ID)
	if err != nil {
		t.Fatalf("Failed to get dashboard: %v", err)
	}

	if retrieved.ID != dashboard.ID {
		t.Errorf("Expected ID=%v, got %v", dashboard.ID, retrieved.ID)
	}

	if retrieved.Name != dashboard.Name {
		t.Errorf("Expected Name=%s, got %s", dashboard.Name, retrieved.Name)
	}
}

func TestGetDashboard_NotFound(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	nonExistentID := uuid.New()

	_, err := dm.GetDashboard(ctx, nonExistentID)
	if err == nil {
		t.Error("Expected error for non-existent dashboard")
	}
}

func TestUpdateDashboard(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	config := json.RawMessage(`{"layout": "grid"}`)

	// Create a dashboard first
	dashboard := &models.Dashboard{
		Name:        "Original Name",
		Description: "Original Description",
		Config:      config,
		IsDefault:   false,
	}

	err := dm.CreateDashboard(ctx, dashboard)
	if err != nil {
		t.Fatalf("Failed to create dashboard: %v", err)
	}

	originalUpdatedAt := dashboard.UpdatedAt
	time.Sleep(10 * time.Millisecond)

	// Update the dashboard
	dashboard.Name = "Updated Name"
	dashboard.Description = "Updated Description"
	dashboard.IsDefault = true

	err = dm.UpdateDashboard(ctx, dashboard)
	if err != nil {
		t.Fatalf("Failed to update dashboard: %v", err)
	}

	if !dashboard.UpdatedAt.After(originalUpdatedAt) {
		t.Error("Expected UpdatedAt to be updated")
	}

	// Verify the update
	retrieved, err := dm.GetDashboard(ctx, dashboard.ID)
	if err != nil {
		t.Fatalf("Failed to get updated dashboard: %v", err)
	}

	if retrieved.Name != "Updated Name" {
		t.Errorf("Expected Name='Updated Name', got %s", retrieved.Name)
	}

	if retrieved.Description != "Updated Description" {
		t.Errorf("Expected Description='Updated Description', got %s", retrieved.Description)
	}

	if !retrieved.IsDefault {
		t.Error("Expected IsDefault to be true")
	}
}

func TestUpdateDashboard_NotFound(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	config := json.RawMessage(`{"layout": "grid"}`)

	dashboard := &models.Dashboard{
		ID:          uuid.New(),
		Name:        "Non-existent",
		Description: "Test",
		Config:      config,
		IsDefault:   false,
	}

	err := dm.UpdateDashboard(ctx, dashboard)
	if err == nil {
		t.Error("Expected error when updating non-existent dashboard")
	}
}

func TestDeleteDashboard(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	config := json.RawMessage(`{"layout": "grid"}`)

	// Create a dashboard first
	dashboard := &models.Dashboard{
		Name:        "To Delete",
		Description: "Test",
		Config:      config,
		IsDefault:   false,
	}

	err := dm.CreateDashboard(ctx, dashboard)
	if err != nil {
		t.Fatalf("Failed to create dashboard: %v", err)
	}

	// Delete the dashboard
	err = dm.DeleteDashboard(ctx, dashboard.ID)
	if err != nil {
		t.Fatalf("Failed to delete dashboard: %v", err)
	}

	// Verify deletion
	_, err = dm.GetDashboard(ctx, dashboard.ID)
	if err == nil {
		t.Error("Expected error when getting deleted dashboard")
	}
}

func TestDeleteDashboard_NotFound(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	nonExistentID := uuid.New()

	err := dm.DeleteDashboard(ctx, nonExistentID)
	if err == nil {
		t.Error("Expected error when deleting non-existent dashboard")
	}
}

func TestGetDefaultDashboard(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	config := json.RawMessage(`{"layout": "grid"}`)

	// Create a default dashboard
	dashboard := &models.Dashboard{
		Name:        "Default Dashboard",
		Description: "Test",
		Config:      config,
		IsDefault:   true,
	}

	err := dm.CreateDashboard(ctx, dashboard)
	if err != nil {
		t.Fatalf("Failed to create dashboard: %v", err)
	}

	// Retrieve default dashboard
	defaultDashboard, err := dm.GetDefaultDashboard(ctx)
	if err != nil {
		t.Fatalf("Failed to get default dashboard: %v", err)
	}

	if defaultDashboard == nil {
		t.Fatal("Expected default dashboard to be found")
	}

	if !defaultDashboard.IsDefault {
		t.Error("Expected IsDefault to be true")
	}

	if defaultDashboard.ID != dashboard.ID {
		t.Errorf("Expected ID=%v, got %v", dashboard.ID, defaultDashboard.ID)
	}
}

func TestGetDefaultDashboard_None(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()

	// Assuming no default dashboard exists
	defaultDashboard, err := dm.GetDefaultDashboard(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if defaultDashboard != nil {
		t.Error("Expected no default dashboard when none exists")
	}
}

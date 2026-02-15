package database

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sguter90/weathermaestro/pkg/models"
)

func TestSaveStation(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := &models.StationData{
		ID:          uuid.New(),
		PassKey:     "test-pass-key-" + uuid.New().String(),
		StationType: "netatmo",
		Model:       "Weather Station",
		Freq:        "868",
		Mode:        "pull",
		ServiceName: "netatmo_service",
		Config: map[string]interface{}{
			"device_id": "70:ee:50:xx:xx:xx",
			"interval":  300,
		},
	}

	err := dm.SaveStation(station)
	if err != nil {
		t.Fatalf("Failed to save station: %v", err)
	}

	// Verify station was saved
	if station.ID == uuid.Nil {
		t.Error("Expected station ID to be set")
	}

	// Retrieve and verify
	retrieved, err := dm.LoadStation(station.ID)
	if err != nil {
		t.Fatalf("Failed to load station: %v", err)
	}

	if retrieved.PassKey != station.PassKey {
		t.Errorf("Expected PassKey=%s, got %s", station.PassKey, retrieved.PassKey)
	}

	if retrieved.StationType != station.StationType {
		t.Errorf("Expected StationType=%s, got %s", station.StationType, retrieved.StationType)
	}

	if retrieved.Config["device_id"] != "70:ee:50:xx:xx:xx" {
		t.Errorf("Expected device_id in config, got %v", retrieved.Config)
	}
}

func TestSaveStation_UpdateOnConflict(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	passKey := "test-pass-key-" + uuid.New().String()

	// Create initial station
	station1 := &models.StationData{
		ID:          uuid.New(),
		PassKey:     passKey,
		StationType: "netatmo",
		Model:       "Model V1",
		Freq:        "868",
		Mode:        "pull",
		ServiceName: "service1",
		Config:      map[string]interface{}{"version": 1},
	}

	err := dm.SaveStation(station1)
	if err != nil {
		t.Fatalf("Failed to save initial station: %v", err)
	}

	initialID := station1.ID

	// Save station with same pass_key but different data
	station2 := &models.StationData{
		ID:          uuid.New(), // Different ID
		PassKey:     passKey,    // Same pass_key
		StationType: "ecowitt",
		Model:       "Model V2",
		Freq:        "915",
		Mode:        "push",
		ServiceName: "service2",
		Config:      map[string]interface{}{"version": 2},
	}

	err = dm.SaveStation(station2)
	if err != nil {
		t.Fatalf("Failed to save updated station: %v", err)
	}

	// Should keep the original ID due to ON CONFLICT
	if station2.ID != initialID {
		t.Errorf("Expected ID to remain %s, got %s", initialID, station2.ID)
	}

	// Verify updated values
	retrieved, err := dm.LoadStation(initialID)
	if err != nil {
		t.Fatalf("Failed to load station: %v", err)
	}

	if retrieved.StationType != "ecowitt" {
		t.Errorf("Expected StationType=ecowitt, got %s", retrieved.StationType)
	}

	if retrieved.Model != "Model V2" {
		t.Errorf("Expected Model='Model V2', got %s", retrieved.Model)
	}

	if retrieved.Config["version"].(float64) != 2 {
		t.Errorf("Expected config version=2, got %v", retrieved.Config["version"])
	}
}

func TestLoadStations(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	// Create multiple stations
	stations := []*models.StationData{
		{
			ID:          uuid.New(),
			PassKey:     "pass-key-1-" + uuid.New().String(),
			StationType: "netatmo",
			Model:       "Station 1",
			Freq:        "868",
			Mode:        "pull",
			ServiceName: "service1",
			Config:      map[string]interface{}{"id": 1},
		},
		{
			ID:          uuid.New(),
			PassKey:     "pass-key-2-" + uuid.New().String(),
			StationType: "ecowitt",
			Model:       "Station 2",
			Freq:        "915",
			Mode:        "push",
			ServiceName: "service2",
			Config:      map[string]interface{}{"id": 2},
		},
	}

	for _, station := range stations {
		err := dm.SaveStation(station)
		if err != nil {
			t.Fatalf("Failed to save station: %v", err)
		}
	}

	// Load all stations
	loaded, err := dm.LoadStations()
	if err != nil {
		t.Fatalf("Failed to load stations: %v", err)
	}

	if len(loaded) < 2 {
		t.Errorf("Expected at least 2 stations, got %d", len(loaded))
	}

	// Verify our stations are in the list
	foundCount := 0
	for _, loaded := range loaded {
		for _, original := range stations {
			if loaded.ID == original.ID {
				foundCount++
				if loaded.PassKey != original.PassKey {
					t.Errorf("PassKey mismatch for station %s", original.ID)
				}
			}
		}
	}

	if foundCount != 2 {
		t.Errorf("Expected to find 2 stations, found %d", foundCount)
	}
}

func TestLoadStation(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := &models.StationData{
		ID:          uuid.New(),
		PassKey:     "test-pass-key-" + uuid.New().String(),
		StationType: "netatmo",
		Model:       "Test Model",
		Freq:        "868",
		Mode:        "pull",
		ServiceName: "test_service",
		Config: map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		},
	}

	err := dm.SaveStation(station)
	if err != nil {
		t.Fatalf("Failed to save station: %v", err)
	}

	// Load the station
	loaded, err := dm.LoadStation(station.ID)
	if err != nil {
		t.Fatalf("Failed to load station: %v", err)
	}

	if loaded.ID != station.ID {
		t.Errorf("Expected ID=%s, got %s", station.ID, loaded.ID)
	}

	if loaded.PassKey != station.PassKey {
		t.Errorf("Expected PassKey=%s, got %s", station.PassKey, loaded.PassKey)
	}

	if loaded.Config["key1"] != "value1" {
		t.Errorf("Expected config key1=value1, got %v", loaded.Config["key1"])
	}

	if loaded.Config["key2"].(float64) != 123 {
		t.Errorf("Expected config key2=123, got %v", loaded.Config["key2"])
	}
}

func TestLoadStation_NotFound(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	nonExistentID := uuid.New()

	_, err := dm.LoadStation(nonExistentID)
	if err == nil {
		t.Error("Expected error when loading non-existent station")
	}
}

func TestEnsureStation(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	data := &models.StationData{
		PassKey:     "ensure-test-" + uuid.New().String(),
		StationType: "netatmo",
		Model:       "Test Model",
		Mode:        "pull",
		ServiceName: "test_service",
	}

	// First call should create
	id1, err := dm.EnsureStation(data)
	if err != nil {
		t.Fatalf("Failed to ensure station: %v", err)
	}

	if id1 == uuid.Nil {
		t.Error("Expected valid station ID")
	}

	// Second call with same pass_key should return same ID
	data.Model = "Updated Model"
	id2, err := dm.EnsureStation(data)
	if err != nil {
		t.Fatalf("Failed to ensure station second time: %v", err)
	}

	if id1 != id2 {
		t.Errorf("Expected same ID on second ensure, got %s and %s", id1, id2)
	}

	// Verify model was updated
	loaded, err := dm.LoadStation(id1)
	if err != nil {
		t.Fatalf("Failed to load station: %v", err)
	}

	if loaded.Model != "Updated Model" {
		t.Errorf("Expected model to be updated to 'Updated Model', got %s", loaded.Model)
	}
}
func TestGetStationList(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	// Create station with sensors and readings
	station := setupTestStation(t, dm)
	sensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")

	// Add some readings
	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		err := dm.StoreSensorReading(sensor.ID, 20.0+float64(i), now.Add(time.Duration(i)*time.Minute))
		if err != nil {
			t.Fatalf("Failed to store reading: %v", err)
		}
	}

	// Get station list
	stations, err := dm.GetStationList()
	if err != nil {
		t.Fatalf("Failed to get station list: %v", err)
	}

	// Find our station
	var found *models.StationDetail
	for i := range stations {
		if stations[i].ID == station.ID {
			found = &stations[i]
			break
		}
	}

	if found == nil {
		t.Fatal("Station not found in list")
	}

	if found.TotalReadings != 5 {
		t.Errorf("Expected 5 readings, got %d", found.TotalReadings)
	}

	if found.FirstReading.IsZero() {
		t.Error("Expected FirstReading to be set")
	}

	if found.LastReading.IsZero() {
		t.Error("Expected LastReading to be set")
	}

	if found.PassKey != station.PassKey {
		t.Errorf("Expected PassKey=%s, got %s", station.PassKey, found.PassKey)
	}
}

func TestGetStationList_NoReadings(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	// Create station without readings
	station := setupTestStation(t, dm)

	// Get station list
	stations, err := dm.GetStationList()
	if err != nil {
		t.Fatalf("Failed to get station list: %v", err)
	}

	// Find our station
	var found *models.StationDetail
	for i := range stations {
		if stations[i].ID == station.ID {
			found = &stations[i]
			break
		}
	}

	if found == nil {
		t.Fatal("Station not found in list")
	}

	if found.TotalReadings != 0 {
		t.Errorf("Expected 0 readings, got %d", found.TotalReadings)
	}

	if !found.FirstReading.IsZero() {
		t.Error("Expected FirstReading to be zero for station without readings")
	}

	if !found.LastReading.IsZero() {
		t.Error("Expected LastReading to be zero for station without readings")
	}
}

func TestGetStation(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	// Create station with sensors and readings
	station := setupTestStation(t, dm)
	sensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")

	// Add readings
	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		err := dm.StoreSensorReading(sensor.ID, 20.0+float64(i), now.Add(time.Duration(i)*time.Minute))
		if err != nil {
			t.Fatalf("Failed to store reading: %v", err)
		}
	}

	// Get specific station
	detail, err := dm.GetStation(station.ID)
	if err != nil {
		t.Fatalf("Failed to get station: %v", err)
	}

	if detail.ID != station.ID {
		t.Errorf("Expected ID=%s, got %s", station.ID, detail.ID)
	}

	if detail.TotalReadings != 3 {
		t.Errorf("Expected 3 readings, got %d", detail.TotalReadings)
	}

	if detail.FirstReading.IsZero() {
		t.Error("Expected FirstReading to be set")
	}

	if detail.LastReading.IsZero() {
		t.Error("Expected LastReading to be set")
	}
}

func TestGetStation_NotFound(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	nonExistentID := uuid.New()

	_, err := dm.GetStation(nonExistentID)
	if err == nil {
		t.Error("Expected error when getting non-existent station")
	}
}

func TestGetStationConfig(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := &models.StationData{
		ID:          uuid.New(),
		PassKey:     "test-pass-key-" + uuid.New().String(),
		StationType: "netatmo",
		Model:       "Test Model",
		Freq:        "868",
		Mode:        "pull",
		ServiceName: "test_service",
		Config: map[string]interface{}{
			"device_id":     "70:ee:50:xx:xx:xx",
			"refresh_token": "test_token",
			"interval":      300,
			"nested_config": map[string]interface{}{
				"key1": "value1",
				"key2": 123,
			},
		},
	}

	err := dm.SaveStation(station)
	if err != nil {
		t.Fatalf("Failed to save station: %v", err)
	}

	// Get config
	config, err := dm.GetStationConfig(station.ID)
	if err != nil {
		t.Fatalf("Failed to get station config: %v", err)
	}

	if config["device_id"] != "70:ee:50:xx:xx:xx" {
		t.Errorf("Expected device_id in config, got %v", config["device_id"])
	}

	if config["refresh_token"] != "test_token" {
		t.Errorf("Expected refresh_token in config, got %v", config["refresh_token"])
	}

	if config["interval"].(float64) != 300 {
		t.Errorf("Expected interval=300, got %v", config["interval"])
	}

	// Check nested config
	nestedConfig := config["nested_config"].(map[string]interface{})
	if nestedConfig["key1"] != "value1" {
		t.Errorf("Expected nested key1=value1, got %v", nestedConfig["key1"])
	}
}

func TestGetStationConfig_NotFound(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	nonExistentID := uuid.New()

	_, err := dm.GetStationConfig(nonExistentID)
	if err == nil {
		t.Error("Expected error when getting config for non-existent station")
	}
}

func TestSetStationConfig(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)

	// Set new config
	newConfig := map[string]interface{}{
		"device_id":     "new_device_id",
		"access_token":  "new_token",
		"refresh_token": "new_refresh",
		"interval":      600,
	}

	err := dm.SetStationConfig(station.ID, newConfig)
	if err != nil {
		t.Fatalf("Failed to set station config: %v", err)
	}

	// Verify config was updated
	retrieved, err := dm.GetStationConfig(station.ID)
	if err != nil {
		t.Fatalf("Failed to get station config: %v", err)
	}

	if retrieved["device_id"] != "new_device_id" {
		t.Errorf("Expected device_id=new_device_id, got %v", retrieved["device_id"])
	}

	if retrieved["access_token"] != "new_token" {
		t.Errorf("Expected access_token=new_token, got %v", retrieved["access_token"])
	}

	if retrieved["interval"].(float64) != 600 {
		t.Errorf("Expected interval=600, got %v", retrieved["interval"])
	}
}

func TestSetStationConfig_PartialUpdate(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	// Create station with initial config
	station := &models.StationData{
		ID:          uuid.New(),
		PassKey:     "test-pass-key-" + uuid.New().String(),
		StationType: "netatmo",
		Model:       "Test Model",
		Freq:        "868",
		Mode:        "pull",
		ServiceName: "test_service",
		Config: map[string]interface{}{
			"device_id": "original_device",
			"key1":      "value1",
			"key2":      "value2",
		},
	}

	err := dm.SaveStation(station)
	if err != nil {
		t.Fatalf("Failed to save station: %v", err)
	}

	// Update only some fields
	partialConfig := map[string]interface{}{
		"device_id": "updated_device",
		"key1":      "updated_value1",
		"key3":      "new_value3",
	}

	err = dm.SetStationConfig(station.ID, partialConfig)
	if err != nil {
		t.Fatalf("Failed to set station config: %v", err)
	}

	// Verify all fields
	retrieved, err := dm.GetStationConfig(station.ID)
	if err != nil {
		t.Fatalf("Failed to get station config: %v", err)
	}

	if retrieved["device_id"] != "updated_device" {
		t.Errorf("Expected device_id=updated_device, got %v", retrieved["device_id"])
	}

	if retrieved["key1"] != "updated_value1" {
		t.Errorf("Expected key1=updated_value1, got %v", retrieved["key1"])
	}

	if retrieved["key3"] != "new_value3" {
		t.Errorf("Expected key3=new_value3, got %v", retrieved["key3"])
	}

	// Note: key2 is replaced (not merged) since we're replacing the entire config
	if _, exists := retrieved["key2"]; exists {
		t.Error("Expected key2 to be removed after config replacement")
	}
}

func TestGetStationIDByConfigValue(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := &models.StationData{
		ID:          uuid.New(),
		PassKey:     "test-pass-key-" + uuid.New().String(),
		StationType: "netatmo",
		Model:       "Test Model",
		Freq:        "868",
		Mode:        "pull",
		ServiceName: "test_service",
		Config: map[string]interface{}{
			"device_id":     "70:ee:50:aa:bb:cc",
			"refresh_token": "unique_token_123",
			"location":      "home",
		},
	}

	err := dm.SaveStation(station)
	if err != nil {
		t.Fatalf("Failed to save station: %v", err)
	}

	// Verify the station was saved with correct config
	savedStation, err := dm.LoadStation(station.ID)
	if err != nil {
		t.Fatalf("Failed to load saved station: %v", err)
	}

	t.Logf("Saved station config: %+v", savedStation.Config)

	// Verify config was saved correctly
	if savedStation.Config["device_id"] != "70:ee:50:aa:bb:cc" {
		t.Fatalf("Config not saved correctly. Expected device_id='70:ee:50:aa:bb:cc', got %v", savedStation.Config["device_id"])
	}

	// Test finding by device_id
	foundID, err := dm.GetStationIDByConfigValue("device_id", "70:ee:50:aa:bb:cc")
	if err != nil {
		t.Fatalf("Failed to get station ID by device_id: %v", err)
	}

	if foundID != station.ID {
		t.Errorf("Expected station ID=%s, got %s", station.ID, foundID)
	}

	// Test finding by refresh_token
	foundID, err = dm.GetStationIDByConfigValue("refresh_token", "unique_token_123")
	if err != nil {
		t.Fatalf("Failed to get station ID by refresh_token: %v", err)
	}

	if foundID != station.ID {
		t.Errorf("Expected station ID=%s, got %s", station.ID, foundID)
	}

	// Test finding by location
	foundID, err = dm.GetStationIDByConfigValue("location", "home")
	if err != nil {
		t.Fatalf("Failed to get station ID by location: %v", err)
	}

	if foundID != station.ID {
		t.Errorf("Expected station ID=%s, got %s", station.ID, foundID)
	}
}

func TestGetStationIDByConfigValue_NotFound(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	_, err := dm.GetStationIDByConfigValue("device_id", "non_existent_device")
	if err == nil {
		t.Error("Expected error when searching for non-existent config value")
	}
}

func TestGetStationIDByConfigValue_MultipleStations(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	// Create multiple stations with different device_ids
	station1 := &models.StationData{
		ID:          uuid.New(),
		PassKey:     "pass-key-1-" + uuid.New().String(),
		StationType: "netatmo",
		Model:       "Station 1",
		Freq:        "868",
		Mode:        "pull",
		ServiceName: "service1",
		Config: map[string]interface{}{
			"device_id": "device_001",
		},
	}

	station2 := &models.StationData{
		ID:          uuid.New(),
		PassKey:     "pass-key-2-" + uuid.New().String(),
		StationType: "netatmo",
		Model:       "Station 2",
		Freq:        "868",
		Mode:        "pull",
		ServiceName: "service2",
		Config: map[string]interface{}{
			"device_id": "device_002",
		},
	}

	err := dm.SaveStation(station1)
	if err != nil {
		t.Fatalf("Failed to save station1: %v", err)
	}

	err = dm.SaveStation(station2)
	if err != nil {
		t.Fatalf("Failed to save station2: %v", err)
	}

	// Find station1
	foundID, err := dm.GetStationIDByConfigValue("device_id", "device_001")
	if err != nil {
		t.Fatalf("Failed to find station1: %v", err)
	}

	if foundID != station1.ID {
		t.Errorf("Expected station1 ID=%s, got %s", station1.ID, foundID)
	}

	// Find station2
	foundID, err = dm.GetStationIDByConfigValue("device_id", "device_002")
	if err != nil {
		t.Fatalf("Failed to find station2: %v", err)
	}

	if foundID != station2.ID {
		t.Errorf("Expected station2 ID=%s, got %s", station2.ID, foundID)
	}
}

func TestGetStationsData(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	// Create test stations
	station1 := &models.StationData{
		ID:          uuid.New(),
		PassKey:     "pass-key-1-" + uuid.New().String(),
		StationType: "netatmo",
		Model:       "Station 1",
		Freq:        "868",
		Mode:        "pull",
		ServiceName: "netatmo_service",
		Config: map[string]interface{}{
			"device_id": "device_001",
		},
	}

	station2 := &models.StationData{
		ID:          uuid.New(),
		PassKey:     "pass-key-2-" + uuid.New().String(),
		StationType: "ecowitt",
		Model:       "Station 2",
		Freq:        "915",
		Mode:        "push",
		ServiceName: "ecowitt_service",
		Config: map[string]interface{}{
			"api_key": "key_002",
		},
	}

	err := dm.SaveStation(station1)
	if err != nil {
		t.Fatalf("Failed to save station1: %v", err)
	}

	err = dm.SaveStation(station2)
	if err != nil {
		t.Fatalf("Failed to save station2: %v", err)
	}

	// Get stations data
	stations, err := dm.GetStationsData()
	if err != nil {
		t.Fatalf("Failed to get stations data: %v", err)
	}

	if len(stations) < 2 {
		t.Errorf("Expected at least 2 stations, got %d", len(stations))
	}

	// Find our stations
	var found1, found2 *models.StationData
	for i := range stations {
		if stations[i].ID == station1.ID {
			found1 = &stations[i]
		}
		if stations[i].ID == station2.ID {
			found2 = &stations[i]
		}
	}

	if found1 == nil {
		t.Error("Station1 not found in results")
	} else {
		if found1.PassKey != station1.PassKey {
			t.Errorf("Station1 PassKey mismatch: expected %s, got %s", station1.PassKey, found1.PassKey)
		}
		if found1.StationType != "netatmo" {
			t.Errorf("Station1 type mismatch: expected netatmo, got %s", found1.StationType)
		}
		if found1.ServiceName != "netatmo_service" {
			t.Errorf("Station1 service mismatch: expected netatmo_service, got %s", found1.ServiceName)
		}
	}

	if found2 == nil {
		t.Error("Station2 not found in results")
	} else {
		if found2.PassKey != station2.PassKey {
			t.Errorf("Station2 PassKey mismatch: expected %s, got %s", station2.PassKey, found2.PassKey)
		}
		if found2.StationType != "ecowitt" {
			t.Errorf("Station2 type mismatch: expected ecowitt, got %s", found2.StationType)
		}
	}
}

func TestGetStationsData_Empty(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	// Clear all stations first
	stations, err := dm.GetStationsData()
	if err != nil {
		t.Fatalf("Failed to get stations: %v", err)
	}

	for _, station := range stations {
		err := dm.DeleteStation(station.ID)
		if err != nil {
			t.Logf("Warning: Failed to delete station %s: %v", station.ID, err)
		}
	}

	// Now get stations data - should be empty or minimal
	stations, err = dm.GetStationsData()
	if err != nil {
		t.Fatalf("Failed to get stations data: %v", err)
	}

	// Should return empty slice, not error
	if stations == nil {
		t.Error("Expected empty slice, got nil")
	}
}

func TestDeleteStation(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	// Create station with sensors and readings
	station := setupTestStation(t, dm)
	sensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")

	// Add readings
	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		err := dm.StoreSensorReading(sensor.ID, 20.0+float64(i), now.Add(time.Duration(i)*time.Minute))
		if err != nil {
			t.Fatalf("Failed to store reading: %v", err)
		}
	}

	// Verify station exists
	_, err := dm.LoadStation(station.ID)
	if err != nil {
		t.Fatalf("Station should exist before deletion: %v", err)
	}

	// Delete station
	err = dm.DeleteStation(station.ID)
	if err != nil {
		t.Fatalf("Failed to delete station: %v", err)
	}

	// Verify station is deleted
	_, err = dm.LoadStation(station.ID)
	if err == nil {
		t.Error("Expected error when loading deleted station")
	}

	// Verify sensors are deleted (cascade)
	params := models.SensorQueryParams{
		StationID:     &station.ID,
		IncludeLatest: false,
	}
	sensors, err := dm.GetSensors(params)
	if err != nil {
		t.Fatalf("Failed to query sensors: %v", err)
	}

	if len(sensors) != 0 {
		t.Errorf("Expected 0 sensors after station deletion, got %d", len(sensors))
	}
}

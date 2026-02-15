package database

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sguter90/weathermaestro/pkg/models"
)

func TestCreateSensor(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)

	batteryLevel := 95
	signalStrength := 85
	sensor := &models.Sensor{
		StationID:      station.ID,
		SensorType:     models.SensorTypeTemperature,
		Location:       "indoor",
		Name:           "Living Room Temperature",
		Model:          "DHT22",
		BatteryLevel:   &batteryLevel,
		SignalStrength: &signalStrength,
		Enabled:        true,
	}

	err := dm.CreateSensor(sensor)
	if err != nil {
		t.Fatalf("Failed to create sensor: %v", err)
	}

	// Verify sensor was created with ID and timestamps
	if sensor.ID == uuid.Nil {
		t.Error("Expected sensor ID to be set")
	}

	if sensor.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	if sensor.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}

	// Verify sensor can be retrieved
	retrieved, err := dm.GetSensor(sensor.ID, false)
	if err != nil {
		t.Fatalf("Failed to retrieve sensor: %v", err)
	}

	if retrieved.Sensor.StationID != station.ID {
		t.Errorf("Expected station_id=%s, got %s", station.ID, retrieved.Sensor.StationID)
	}

	if retrieved.Sensor.SensorType != models.SensorTypeTemperature {
		t.Errorf("Expected sensor_type=%s, got %s", models.SensorTypeTemperature, retrieved.Sensor.SensorType)
	}

	if retrieved.Sensor.Location != "indoor" {
		t.Errorf("Expected location=indoor, got %s", retrieved.Sensor.Location)
	}

	if retrieved.Sensor.Name != "Living Room Temperature" {
		t.Errorf("Expected name='Living Room Temperature', got %s", retrieved.Sensor.Name)
	}
}

func TestGetSensor_WithoutLatestReading(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	sensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")

	// Get sensor without latest reading
	retrieved, err := dm.GetSensor(sensor.ID, false)
	if err != nil {
		t.Fatalf("Failed to get sensor: %v", err)
	}

	if retrieved.Sensor.ID != sensor.ID {
		t.Errorf("Expected sensor ID=%s, got %s", sensor.ID, retrieved.Sensor.ID)
	}

	if retrieved.LatestReading != nil {
		t.Error("Expected LatestReading to be nil when includeLatest=false")
	}
}

func TestGetSensor_WithLatestReading(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	sensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")

	// Store a reading
	now := time.Now().UTC()
	expectedValue := 22.5
	err := dm.StoreSensorReading(sensor.ID, expectedValue, now)
	if err != nil {
		t.Fatalf("Failed to store reading: %v", err)
	}

	// Get sensor with latest reading
	retrieved, err := dm.GetSensor(sensor.ID, true)
	if err != nil {
		t.Fatalf("Failed to get sensor: %v", err)
	}

	if retrieved.LatestReading == nil {
		t.Fatal("Expected LatestReading to be set when includeLatest=true")
	}

	if retrieved.LatestReading.SensorID != sensor.ID {
		t.Errorf("Expected reading sensor_id=%s, got %s", sensor.ID, retrieved.LatestReading.SensorID)
	}

	if retrieved.LatestReading.Value != expectedValue {
		t.Errorf("Expected reading value=%f, got %f", expectedValue, retrieved.LatestReading.Value)
	}

	// Verify it's the latest reading
	timeDiff := retrieved.LatestReading.DateUTC.Sub(now).Abs()
	if timeDiff > time.Second {
		t.Errorf("Expected reading timestamp close to %v, got %v (diff: %v)",
			now, retrieved.LatestReading.DateUTC, timeDiff)
	}
}

func TestGetSensor_NonExistent(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	nonExistentID := uuid.New()

	_, err := dm.GetSensor(nonExistentID, false)
	if err == nil {
		t.Error("Expected error when getting non-existent sensor")
	}
}

func TestGetSensors_FilterByStationID(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station1 := setupTestStation(t, dm)
	station2 := setupTestStation(t, dm)

	// Create sensors for both stations
	sensor1 := setupTestSensor(t, dm, station1.ID, models.SensorTypeTemperature, "indoor")
	sensor2 := setupTestSensor(t, dm, station2.ID, models.SensorTypeTemperature, "indoor")

	// Query for station1 sensors only
	params := models.SensorQueryParams{
		StationID:     &station1.ID,
		IncludeLatest: false,
	}

	sensors, err := dm.GetSensors(params)
	if err != nil {
		t.Fatalf("Failed to get sensors: %v", err)
	}

	if len(sensors) != 1 {
		t.Errorf("Expected 1 sensor for station1, got %d", len(sensors))
	}

	if len(sensors) > 0 && sensors[0].Sensor.ID != sensor1.ID {
		t.Errorf("Expected sensor1 (ID=%s), got %s", sensor1.ID, sensors[0].Sensor.ID)
	}

	// Verify sensor2 is not in results
	for _, s := range sensors {
		if s.Sensor.ID == sensor2.ID {
			t.Error("sensor2 should not be in results for station1")
		}
	}
}

func TestGetSensors_FilterBySensorType(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)

	// Create sensors of different types
	tempSensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")
	humSensor := setupTestSensor(t, dm, station.ID, models.SensorTypeHumidity, "indoor")
	pressSensor := setupTestSensor(t, dm, station.ID, models.SensorTypePressure, "indoor")

	// Query for temperature sensors only
	params := models.SensorQueryParams{
		StationID:     &station.ID,
		SensorType:    models.SensorTypeTemperature,
		IncludeLatest: false,
	}

	sensors, err := dm.GetSensors(params)
	if err != nil {
		t.Fatalf("Failed to get sensors: %v", err)
	}

	if len(sensors) != 1 {
		t.Errorf("Expected 1 temperature sensor, got %d", len(sensors))
	}

	if len(sensors) > 0 && sensors[0].Sensor.ID != tempSensor.ID {
		t.Errorf("Expected temperature sensor (ID=%s), got %s", tempSensor.ID, sensors[0].Sensor.ID)
	}

	// Verify other sensor types are not in results
	for _, s := range sensors {
		if s.Sensor.ID == humSensor.ID || s.Sensor.ID == pressSensor.ID {
			t.Error("Non-temperature sensors should not be in results")
		}
	}
}

func TestGetSensors_FilterByLocation(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)

	// Create sensors in different locations
	indoorSensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")
	outdoorSensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "outdoor")
	garageSensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "garage")

	// Query for indoor sensors only
	params := models.SensorQueryParams{
		StationID:     &station.ID,
		Location:      "indoor",
		IncludeLatest: false,
	}

	sensors, err := dm.GetSensors(params)
	if err != nil {
		t.Fatalf("Failed to get sensors: %v", err)
	}

	if len(sensors) != 1 {
		t.Errorf("Expected 1 indoor sensor, got %d", len(sensors))
	}

	if len(sensors) > 0 && sensors[0].Sensor.ID != indoorSensor.ID {
		t.Errorf("Expected indoor sensor (ID=%s), got %s", indoorSensor.ID, sensors[0].Sensor.ID)
	}

	// Verify other locations are not in results
	for _, s := range sensors {
		if s.Sensor.ID == outdoorSensor.ID || s.Sensor.ID == garageSensor.ID {
			t.Error("Non-indoor sensors should not be in results")
		}
	}
}

func TestGetSensors_FilterByEnabled(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)

	// Create enabled and disabled sensors
	batteryLevel := 100
	signalStrength := 100
	enabledSensor := &models.Sensor{
		StationID:      station.ID,
		SensorType:     models.SensorTypeTemperature,
		Location:       "indoor",
		Name:           "Enabled Sensor",
		Enabled:        true,
		BatteryLevel:   &batteryLevel,
		SignalStrength: &signalStrength,
	}
	err := dm.CreateSensor(enabledSensor)
	if err != nil {
		t.Fatalf("Failed to create enabled sensor: %v", err)
	}

	disabledSensor := &models.Sensor{
		StationID:      station.ID,
		SensorType:     models.SensorTypeTemperature,
		Location:       "outdoor",
		Name:           "Disabled Sensor",
		Enabled:        false,
		BatteryLevel:   &batteryLevel,
		SignalStrength: &signalStrength,
	}
	err = dm.CreateSensor(disabledSensor)
	if err != nil {
		t.Fatalf("Failed to create disabled sensor: %v", err)
	}

	// Query for enabled sensors only
	enabledFilter := true
	params := models.SensorQueryParams{
		StationID:     &station.ID,
		Enabled:       &enabledFilter,
		IncludeLatest: false,
	}

	sensors, err := dm.GetSensors(params)
	if err != nil {
		t.Fatalf("Failed to get sensors: %v", err)
	}

	if len(sensors) != 1 {
		t.Errorf("Expected 1 enabled sensor, got %d", len(sensors))
	}

	if len(sensors) > 0 {
		if sensors[0].Sensor.ID != enabledSensor.ID {
			t.Errorf("Expected enabled sensor (ID=%s), got %s", enabledSensor.ID, sensors[0].Sensor.ID)
		}
		if !sensors[0].Sensor.Enabled {
			t.Error("Expected sensor to be enabled")
		}
	}

	// Query for disabled sensors only
	disabledFilter := false
	params.Enabled = &disabledFilter

	sensors, err = dm.GetSensors(params)
	if err != nil {
		t.Fatalf("Failed to get sensors: %v", err)
	}

	if len(sensors) != 1 {
		t.Errorf("Expected 1 disabled sensor, got %d", len(sensors))
	}

	if len(sensors) > 0 {
		if sensors[0].Sensor.ID != disabledSensor.ID {
			t.Errorf("Expected disabled sensor (ID=%s), got %s", disabledSensor.ID, sensors[0].Sensor.ID)
		}
		if sensors[0].Sensor.Enabled {
			t.Error("Expected sensor to be disabled")
		}
	}
}

func TestGetSensors_MultipleFilters(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)

	// Create various sensors
	setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")
	targetSensor := setupTestSensor(t, dm, station.ID, models.SensorTypeHumidity, "indoor")
	setupTestSensor(t, dm, station.ID, models.SensorTypeHumidity, "outdoor")
	setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "outdoor")

	// Query with multiple filters
	params := models.SensorQueryParams{
		StationID:     &station.ID,
		SensorType:    models.SensorTypeHumidity,
		Location:      "indoor",
		IncludeLatest: false,
	}

	sensors, err := dm.GetSensors(params)
	if err != nil {
		t.Fatalf("Failed to get sensors: %v", err)
	}

	if len(sensors) != 1 {
		t.Errorf("Expected 1 sensor matching all filters, got %d", len(sensors))
	}

	if len(sensors) > 0 {
		if sensors[0].Sensor.ID != targetSensor.ID {
			t.Errorf("Expected target sensor (ID=%s), got %s", targetSensor.ID, sensors[0].Sensor.ID)
		}
		if sensors[0].Sensor.SensorType != models.SensorTypeHumidity {
			t.Errorf("Expected sensor_type=%s, got %s", models.SensorTypeHumidity, sensors[0].Sensor.SensorType)
		}
		if sensors[0].Sensor.Location != "indoor" {
			t.Errorf("Expected location=indoor, got %s", sensors[0].Sensor.Location)
		}
	}
}

func TestGetSensors_WithLatestReadings(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	sensor1 := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")
	sensor2 := setupTestSensor(t, dm, station.ID, models.SensorTypeHumidity, "indoor")

	// Store readings for sensor1 only
	now := time.Now().UTC()
	err := dm.StoreSensorReading(sensor1.ID, 22.5, now)
	if err != nil {
		t.Fatalf("Failed to store reading: %v", err)
	}

	// Query with include_latest=true
	params := models.SensorQueryParams{
		StationID:     &station.ID,
		IncludeLatest: true,
	}

	sensors, err := dm.GetSensors(params)
	if err != nil {
		t.Fatalf("Failed to get sensors: %v", err)
	}

	if len(sensors) != 2 {
		t.Errorf("Expected 2 sensors, got %d", len(sensors))
	}

	// Find sensor1 and sensor2 in results
	var sensor1Result, sensor2Result *models.SensorWithLatestReading
	for i := range sensors {
		if sensors[i].Sensor.ID == sensor1.ID {
			sensor1Result = &sensors[i]
		} else if sensors[i].Sensor.ID == sensor2.ID {
			sensor2Result = &sensors[i]
		}
	}

	// Verify sensor1 has latest reading
	if sensor1Result == nil {
		t.Fatal("sensor1 not found in results")
	}
	if sensor1Result.LatestReading == nil {
		t.Error("Expected sensor1 to have latest reading")
	} else {
		if sensor1Result.LatestReading.Value != 22.5 {
			t.Errorf("Expected reading value=22.5, got %f", sensor1Result.LatestReading.Value)
		}
	}

	// Verify sensor2 has no latest reading
	if sensor2Result == nil {
		t.Fatal("sensor2 not found in results")
	}
	if sensor2Result.LatestReading != nil {
		t.Error("Expected sensor2 to have no latest reading")
	}
}

func TestEnsureSensorsByRemoteId_CreateNew(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)

	// Create sensors map with remote IDs
	batteryLevel := 95
	signalStrength := 85
	sensors := map[string]models.Sensor{
		"remote_temp_1": {
			SensorType:     models.SensorTypeTemperature,
			Location:       "indoor",
			Name:           "Remote Temperature 1",
			Model:          "DHT22",
			BatteryLevel:   &batteryLevel,
			SignalStrength: &signalStrength,
			Enabled:        true,
		},
		"remote_hum_1": {
			SensorType:     models.SensorTypeHumidity,
			Location:       "indoor",
			Name:           "Remote Humidity 1",
			Model:          "DHT22",
			BatteryLevel:   &batteryLevel,
			SignalStrength: &signalStrength,
			Enabled:        true,
		},
	}

	// Ensure sensors (should create new ones)
	result, err := dm.EnsureSensorsByRemoteId(station.ID, sensors)
	if err != nil {
		t.Fatalf("Failed to ensure sensors: %v", err)
	}

	// Verify both sensors were created with IDs
	if result["remote_temp_1"].ID == uuid.Nil {
		t.Error("Expected remote_temp_1 to have ID set")
	}
	if result["remote_hum_1"].ID == uuid.Nil {
		t.Error("Expected remote_hum_1 to have ID set")
	}

	// Verify sensors can be retrieved
	params := models.SensorQueryParams{
		StationID:     &station.ID,
		IncludeLatest: false,
	}
	retrievedSensors, err := dm.GetSensors(params)
	if err != nil {
		t.Fatalf("Failed to get sensors: %v", err)
	}

	if len(retrievedSensors) != 2 {
		t.Errorf("Expected 2 sensors, got %d", len(retrievedSensors))
	}
}
func TestEnsureSensorsByRemoteId_UpdateExisting(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)

	// Create initial sensor with remote_id
	initialBattery := 100
	initialSignal := 90
	initialSensor := &models.Sensor{
		StationID:      station.ID,
		SensorType:     models.SensorTypeTemperature,
		Location:       "indoor",
		Name:           "Initial Name",
		Model:          "DHT22",
		BatteryLevel:   &initialBattery,
		SignalStrength: &initialSignal,
		Enabled:        true,
		RemoteID:       "remote_temp_1",
	}
	err := dm.CreateSensor(initialSensor)
	if err != nil {
		t.Fatalf("Failed to create initial sensor: %v", err)
	}

	originalID := initialSensor.ID

	// Update sensor via EnsureSensorsByRemoteId
	updatedBattery := 75
	updatedSignal := 80
	sensors := map[string]models.Sensor{
		"remote_temp_1": {
			StationID:      station.ID,
			SensorType:     models.SensorTypeTemperature,
			Location:       "outdoor",      // Changed
			Name:           "Updated Name", // Changed
			Model:          "DHT22",
			BatteryLevel:   &updatedBattery, // Changed
			SignalStrength: &updatedSignal,  // Changed
			Enabled:        true,
			RemoteID:       "remote_temp_1",
		},
	}

	result, err := dm.EnsureSensorsByRemoteId(station.ID, sensors)
	if err != nil {
		t.Fatalf("Failed to ensure sensors: %v", err)
	}

	// Verify ID remained the same
	if result["remote_temp_1"].ID != originalID {
		t.Errorf("Expected sensor ID to remain %s, got %s", originalID, result["remote_temp_1"].ID)
	}

	// Verify sensor was updated
	retrieved, err := dm.GetSensor(originalID, false)
	if err != nil {
		t.Fatalf("Failed to retrieve updated sensor: %v", err)
	}

	if retrieved.Sensor.Location != "outdoor" {
		t.Errorf("Expected location=outdoor, got %s", retrieved.Sensor.Location)
	}

	if retrieved.Sensor.Name != "Updated Name" {
		t.Errorf("Expected name='Updated Name', got %s", retrieved.Sensor.Name)
	}

	if retrieved.Sensor.BatteryLevel == nil {
		t.Error("Expected BatteryLevel to be set")
	} else if *retrieved.Sensor.BatteryLevel != 75 {
		t.Errorf("Expected BatteryLevel=75, got %d", *retrieved.Sensor.BatteryLevel)
	}

	if retrieved.Sensor.SignalStrength == nil {
		t.Error("Expected SignalStrength to be set")
	} else if *retrieved.Sensor.SignalStrength != 80 {
		t.Errorf("Expected SignalStrength=80, got %d", *retrieved.Sensor.SignalStrength)
	}
}

func TestEnsureSensorsByRemoteId_MixedCreateAndUpdate(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)

	// Create one existing sensor
	existingBattery := 100
	existingSignal := 90
	existingSensor := &models.Sensor{
		StationID:      station.ID,
		SensorType:     models.SensorTypeTemperature,
		Location:       "indoor",
		Name:           "Existing Sensor",
		Model:          "DHT22",
		BatteryLevel:   &existingBattery,
		SignalStrength: &existingSignal,
		Enabled:        true,
		RemoteID:       "remote_temp_1",
	}
	err := dm.CreateSensor(existingSensor)
	if err != nil {
		t.Fatalf("Failed to create existing sensor: %v", err)
	}

	existingID := existingSensor.ID

	// Ensure sensors: one existing (update), one new (create)
	updatedBattery := 85
	updatedSignal := 75
	newBattery := 95
	newSignal := 88
	sensors := map[string]models.Sensor{
		"remote_temp_1": {
			SensorType:     models.SensorTypeTemperature,
			Location:       "outdoor",        // Changed
			Name:           "Updated Sensor", // Changed
			Model:          "DHT22",
			BatteryLevel:   &updatedBattery, // Changed
			SignalStrength: &updatedSignal,  // Changed
			Enabled:        true,
		},
		"remote_hum_1": {
			SensorType:     models.SensorTypeHumidity,
			Location:       "indoor",
			Name:           "New Humidity Sensor",
			Model:          "DHT22",
			BatteryLevel:   &newBattery,
			SignalStrength: &newSignal,
			Enabled:        true,
		},
	}

	result, err := dm.EnsureSensorsByRemoteId(station.ID, sensors)
	if err != nil {
		t.Fatalf("Failed to ensure sensors: %v", err)
	}

	// Verify existing sensor was updated (ID unchanged)
	if result["remote_temp_1"].ID != existingID {
		t.Errorf("Expected existing sensor ID to remain %s, got %s", existingID, result["remote_temp_1"].ID)
	}

	// Verify new sensor was created (has new ID)
	if result["remote_hum_1"].ID == uuid.Nil {
		t.Error("Expected new sensor to have ID set")
	}

	// Verify both sensors exist in database
	params := models.SensorQueryParams{
		StationID:     &station.ID,
		IncludeLatest: false,
	}
	retrievedSensors, err := dm.GetSensors(params)
	if err != nil {
		t.Fatalf("Failed to get sensors: %v", err)
	}

	if len(retrievedSensors) != 2 {
		t.Errorf("Expected 2 sensors, got %d", len(retrievedSensors))
	}

	// Verify updated sensor properties
	updatedSensor, err := dm.GetSensor(existingID, false)
	if err != nil {
		t.Fatalf("Failed to retrieve updated sensor: %v", err)
	}

	if updatedSensor.Sensor.Location != "outdoor" {
		t.Errorf("Expected location=outdoor, got %s", updatedSensor.Sensor.Location)
	}

	if updatedSensor.Sensor.Name != "Updated Sensor" {
		t.Errorf("Expected name='Updated Sensor', got %s", updatedSensor.Sensor.Name)
	}

	if updatedSensor.Sensor.BatteryLevel == nil {
		t.Error("Expected BatteryLevel to be set")
	} else if *updatedSensor.Sensor.BatteryLevel != 85 {
		t.Errorf("Expected BatteryLevel=85, got %d", *updatedSensor.Sensor.BatteryLevel)
	}
}

func TestEnsureSensorsByRemoteId_WithNullBatteryAndSignal(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)

	// Create sensors with nil battery and signal (wired sensors)
	sensors := map[string]models.Sensor{
		"remote_temp_wired": {
			SensorType:     models.SensorTypeTemperature,
			Location:       "indoor",
			Name:           "Wired Temperature Sensor",
			Model:          "DS18B20",
			BatteryLevel:   nil, // Wired sensor
			SignalStrength: nil, // No wireless signal
			Enabled:        true,
		},
		"remote_hum_wireless": {
			SensorType:     models.SensorTypeHumidity,
			Location:       "outdoor",
			Name:           "Wireless Humidity Sensor",
			Model:          "DHT22",
			BatteryLevel:   func() *int { v := 90; return &v }(),
			SignalStrength: func() *int { v := 85; return &v }(),
			Enabled:        true,
		},
	}

	result, err := dm.EnsureSensorsByRemoteId(station.ID, sensors)
	if err != nil {
		t.Fatalf("Failed to ensure sensors: %v", err)
	}

	// Verify wired sensor has nil battery and signal
	wiredSensor, err := dm.GetSensor(result["remote_temp_wired"].ID, false)
	if err != nil {
		t.Fatalf("Failed to retrieve wired sensor: %v", err)
	}

	if wiredSensor.Sensor.BatteryLevel != nil {
		t.Errorf("Expected wired sensor BatteryLevel to be nil, got %d", *wiredSensor.Sensor.BatteryLevel)
	}

	if wiredSensor.Sensor.SignalStrength != nil {
		t.Errorf("Expected wired sensor SignalStrength to be nil, got %d", *wiredSensor.Sensor.SignalStrength)
	}

	// Verify wireless sensor has battery and signal values
	wirelessSensor, err := dm.GetSensor(result["remote_hum_wireless"].ID, false)
	if err != nil {
		t.Fatalf("Failed to retrieve wireless sensor: %v", err)
	}

	if wirelessSensor.Sensor.BatteryLevel == nil {
		t.Error("Expected wireless sensor BatteryLevel to be set")
	} else if *wirelessSensor.Sensor.BatteryLevel != 90 {
		t.Errorf("Expected BatteryLevel=90, got %d", *wirelessSensor.Sensor.BatteryLevel)
	}

	if wirelessSensor.Sensor.SignalStrength == nil {
		t.Error("Expected wireless sensor SignalStrength to be set")
	} else if *wirelessSensor.Sensor.SignalStrength != 85 {
		t.Errorf("Expected SignalStrength=85, got %d", *wirelessSensor.Sensor.SignalStrength)
	}
}

func TestEnsureSensorsByRemoteId_UpdateToNullBatteryAndSignal(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)

	// Create initial sensor with battery and signal
	initialBattery := 100
	initialSignal := 90
	initialSensor := &models.Sensor{
		StationID:      station.ID,
		SensorType:     models.SensorTypeTemperature,
		Location:       "indoor",
		Name:           "Wireless Sensor",
		Model:          "DHT22",
		BatteryLevel:   &initialBattery,
		SignalStrength: &initialSignal,
		Enabled:        true,
		RemoteID:       "remote_temp_1",
	}
	err := dm.CreateSensor(initialSensor)
	if err != nil {
		t.Fatalf("Failed to create initial sensor: %v", err)
	}

	originalID := initialSensor.ID

	// Update sensor to wired (nil battery and signal)
	sensors := map[string]models.Sensor{
		"remote_temp_1": {
			SensorType:     models.SensorTypeTemperature,
			Location:       "indoor",
			Name:           "Now Wired Sensor",
			Model:          "DS18B20",
			BatteryLevel:   nil, // Changed to wired
			SignalStrength: nil, // Changed to wired
			Enabled:        true,
		},
	}

	result, err := dm.EnsureSensorsByRemoteId(station.ID, sensors)
	if err != nil {
		t.Fatalf("Failed to ensure sensors: %v", err)
	}

	// Verify ID remained the same
	if result["remote_temp_1"].ID != originalID {
		t.Errorf("Expected sensor ID to remain %s, got %s", originalID, result["remote_temp_1"].ID)
	}

	// Verify sensor was updated to wired (nil values)
	retrieved, err := dm.GetSensor(originalID, false)
	if err != nil {
		t.Fatalf("Failed to retrieve updated sensor: %v", err)
	}

	if retrieved.Sensor.BatteryLevel != nil {
		t.Errorf("Expected BatteryLevel to be nil, got %d", *retrieved.Sensor.BatteryLevel)
	}

	if retrieved.Sensor.SignalStrength != nil {
		t.Errorf("Expected SignalStrength to be nil, got %d", *retrieved.Sensor.SignalStrength)
	}

	if retrieved.Sensor.Name != "Now Wired Sensor" {
		t.Errorf("Expected name='Now Wired Sensor', got %s", retrieved.Sensor.Name)
	}
}

func TestEnsureSensorsByRemoteId_EmptyMap(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)

	// Ensure with empty map (should not error)
	sensors := map[string]models.Sensor{}

	result, err := dm.EnsureSensorsByRemoteId(station.ID, sensors)
	if err != nil {
		t.Fatalf("Failed to ensure sensors with empty map: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result map, got %d sensors", len(result))
	}
}

func TestEnsureSensorsByRemoteId_DifferentStations(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station1 := setupTestStation(t, dm)
	station2 := setupTestStation(t, dm)

	// Create sensor with same remote_id on station1
	battery := 100
	signal := 90
	sensor1 := &models.Sensor{
		StationID:      station1.ID,
		SensorType:     models.SensorTypeTemperature,
		Location:       "indoor",
		Name:           "Station 1 Sensor",
		Model:          "DHT22",
		BatteryLevel:   &battery,
		SignalStrength: &signal,
		Enabled:        true,
		RemoteID:       "remote_temp_shared",
	}
	err := dm.CreateSensor(sensor1)
	if err != nil {
		t.Fatalf("Failed to create sensor on station1: %v", err)
	}

	// Ensure sensor with same remote_id on station2 (should create new sensor)
	sensors := map[string]models.Sensor{
		"remote_temp_shared": {
			SensorType:     models.SensorTypeTemperature,
			Location:       "outdoor",
			Name:           "Station 2 Sensor",
			Model:          "DHT22",
			BatteryLevel:   &battery,
			SignalStrength: &signal,
			Enabled:        true,
		},
	}

	result, err := dm.EnsureSensorsByRemoteId(station2.ID, sensors)
	if err != nil {
		t.Fatalf("Failed to ensure sensors on station2: %v", err)
	}

	// Verify new sensor was created (different ID)
	if result["remote_temp_shared"].ID == sensor1.ID {
		t.Error("Expected different sensor ID for station2, got same as station1")
	}

	// Verify both sensors exist with same remote_id but different stations
	sensor1Retrieved, err := dm.GetSensor(sensor1.ID, false)
	if err != nil {
		t.Fatalf("Failed to retrieve station1 sensor: %v", err)
	}

	sensor2Retrieved, err := dm.GetSensor(result["remote_temp_shared"].ID, false)
	if err != nil {
		t.Fatalf("Failed to retrieve station2 sensor: %v", err)
	}

	if sensor1Retrieved.Sensor.StationID != station1.ID {
		t.Errorf("Expected station1 sensor to belong to station1, got %s", sensor1Retrieved.Sensor.StationID)
	}

	if sensor2Retrieved.Sensor.StationID != station2.ID {
		t.Errorf("Expected station2 sensor to belong to station2, got %s", sensor2Retrieved.Sensor.StationID)
	}

	if sensor1Retrieved.Sensor.RemoteID != sensor2Retrieved.Sensor.RemoteID {
		t.Error("Expected both sensors to have same remote_id")
	}
}

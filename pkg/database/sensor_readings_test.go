package database

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sguter90/weathermaestro/pkg/models"
)

// setupTestStation creates a test station and returns it
func setupTestStation(t *testing.T, dm *DatabaseManager) *models.StationData {
	t.Helper()

	station := &models.StationData{
		ID:          uuid.New(),
		PassKey:     "test-pass-key-" + uuid.New().String(),
		StationType: "test",
		Model:       "test-model",
		Freq:        "868",
		Mode:        "push",
		ServiceName: "test_service",
		Config:      map[string]interface{}{},
	}

	err := dm.SaveStation(station)
	if err != nil {
		t.Fatalf("Failed to create test station: %v", err)
	}

	return station
}

// setupTestSensor creates a test sensor for a given station
func setupTestSensor(t *testing.T, dm *DatabaseManager, stationID uuid.UUID, sensorType string, location string) *models.Sensor {
	t.Helper()

	sensor := &models.Sensor{
		StationID:  stationID,
		SensorType: sensorType,
		Location:   location,
		Name:       string(sensorType) + " Sensor",
		Enabled:    true,
	}

	err := dm.CreateSensor(sensor)
	if err != nil {
		t.Fatalf("Failed to create test sensor: %v", err)
	}

	return sensor
}

// storeTestReadings stores a series of readings for a sensor
func storeTestReadings(t *testing.T, dm *DatabaseManager, sensorID uuid.UUID, startTime time.Time, count int, valueFunc func(int) float64) {
	t.Helper()

	for i := 0; i < count; i++ {
		value := valueFunc(i)
		timestamp := startTime.Add(time.Duration(i) * time.Minute)

		err := dm.StoreSensorReading(sensorID, value, timestamp)
		if err != nil {
			t.Fatalf("Failed to store reading %d: %v", i, err)
		}
	}
}

func TestStoreSensorReading(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	sensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")

	// Store a reading
	now := time.Now().UTC()
	value := 23.5

	err := dm.StoreSensorReading(sensor.ID, value, now)
	if err != nil {
		t.Fatalf("Failed to store sensor reading: %v", err)
	}

	// Verify the reading was stored
	readings, err := dm.GetSensorReadings(sensor.ID, now.Add(-1*time.Hour), now.Add(1*time.Hour), 10)
	if err != nil {
		t.Fatalf("Failed to get sensor readings: %v", err)
	}

	if len(readings) != 1 {
		t.Errorf("Expected 1 reading, got %d", len(readings))
	}

	if len(readings) > 0 {
		if readings[0].Value != value {
			t.Errorf("Expected value=%f, got %f", value, readings[0].Value)
		}
		if readings[0].SensorID != sensor.ID {
			t.Errorf("Expected sensor_id=%s, got %s", sensor.ID, readings[0].SensorID)
		}
	}
}

func TestGetSensorReadings(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	sensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")

	// Store multiple readings
	now := time.Now().UTC()
	storeTestReadings(t, dm, sensor.ID, now, 5, func(i int) float64 {
		return float64(20 + i)
	})

	// Get readings
	readings, err := dm.GetSensorReadings(sensor.ID, now.Add(-1*time.Hour), now.Add(1*time.Hour), 10)
	if err != nil {
		t.Fatalf("Failed to get sensor readings: %v", err)
	}

	if len(readings) != 5 {
		t.Errorf("Expected 5 readings, got %d", len(readings))
	}

	// Verify readings are ordered by date_utc DESC
	for i := 1; i < len(readings); i++ {
		if readings[i-1].DateUTC.Before(readings[i].DateUTC) {
			t.Error("Expected readings to be ordered by date_utc DESC")
		}
	}
}

func TestGetSensorReadings_WithLimit(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	sensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")

	// Store 10 readings
	now := time.Now().UTC()
	storeTestReadings(t, dm, sensor.ID, now, 10, func(i int) float64 {
		return float64(20 + i)
	})

	// Get readings with limit of 5
	readings, err := dm.GetSensorReadings(sensor.ID, now.Add(-1*time.Hour), now.Add(1*time.Hour), 5)
	if err != nil {
		t.Fatalf("Failed to get sensor readings: %v", err)
	}

	if len(readings) != 5 {
		t.Errorf("Expected 5 readings (limit), got %d", len(readings))
	}
}

func TestGetReadings(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	sensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")

	// Store readings
	now := time.Now().UTC()
	storeTestReadings(t, dm, sensor.ID, now, 15, func(i int) float64 {
		return float64(20 + i)
	})

	// Test basic query
	params := models.ReadingQueryParams{
		StationID: &station.ID,
		Page:      1,
		Limit:     10,
		Order:     "desc",
	}

	response, err := dm.GetReadings(params)
	if err != nil {
		t.Fatalf("Failed to get readings: %v", err)
	}

	if response.Total != 15 {
		t.Errorf("Expected total=15, got %d", response.Total)
	}

	if len(response.Data.([]models.SensorReading)) != 10 {
		t.Errorf("Expected 10 readings on page 1, got %d", len(response.Data.([]models.SensorReading)))
	}

	if response.TotalPages != 2 {
		t.Errorf("Expected 2 total pages, got %d", response.TotalPages)
	}

	if !response.HasMore {
		t.Error("Expected HasMore to be true")
	}

	if response.IsAggregated {
		t.Error("Expected IsAggregated to be false")
	}
}

func TestGetReadings_Pagination(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	sensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")

	// Store 25 readings
	now := time.Now().UTC()
	storeTestReadings(t, dm, sensor.ID, now, 25, func(i int) float64 {
		return float64(20 + i)
	})

	// Test page 2
	params := models.ReadingQueryParams{
		StationID: &station.ID,
		Page:      2,
		Limit:     10,
		Order:     "desc",
	}

	response, err := dm.GetReadings(params)
	if err != nil {
		t.Fatalf("Failed to get readings: %v", err)
	}

	if len(response.Data.([]models.SensorReading)) != 10 {
		t.Errorf("Expected 10 readings on page 2, got %d", len(response.Data.([]models.SensorReading)))
	}

	if !response.HasMore {
		t.Error("Expected HasMore to be true on page 2")
	}

	// Test page 3 (last page)
	params.Page = 3
	response, err = dm.GetReadings(params)
	if err != nil {
		t.Fatalf("Failed to get readings: %v", err)
	}

	if len(response.Data.([]models.SensorReading)) != 5 {
		t.Errorf("Expected 5 readings on page 3, got %d", len(response.Data.([]models.SensorReading)))
	}

	if response.HasMore {
		t.Error("Expected HasMore to be false on last page")
	}
}

func TestGetReadings_FilterBySensorType(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	tempSensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")
	humSensor := setupTestSensor(t, dm, station.ID, models.SensorTypeHumidity, "indoor")

	// Store readings for both sensors
	now := time.Now().UTC()
	storeTestReadings(t, dm, tempSensor.ID, now, 5, func(i int) float64 {
		return float64(20 + i)
	})
	storeTestReadings(t, dm, humSensor.ID, now, 5, func(i int) float64 {
		return float64(50 + i)
	})

	// Query only temperature readings
	params := models.ReadingQueryParams{
		StationID:  &station.ID,
		SensorType: models.SensorTypeTemperature,
		Page:       1,
		Limit:      10,
		Order:      "desc",
	}

	response, err := dm.GetReadings(params)
	if err != nil {
		t.Fatalf("Failed to get readings: %v", err)
	}

	if response.Total != 5 {
		t.Errorf("Expected 5 temperature readings, got %d", response.Total)
	}

	readings := response.Data.([]models.SensorReading)
	for _, reading := range readings {
		if reading.SensorID != tempSensor.ID {
			t.Errorf("Expected only temperature sensor readings, got sensor_id=%s", reading.SensorID)
		}
	}
}

func TestGetReadings_FilterByLocation(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	indoorSensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")
	outdoorSensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "outdoor")

	// Store readings for both sensors
	now := time.Now().UTC()
	storeTestReadings(t, dm, indoorSensor.ID, now, 5, func(i int) float64 {
		return float64(20 + i)
	})
	storeTestReadings(t, dm, outdoorSensor.ID, now, 5, func(i int) float64 {
		return float64(10 + i)
	})

	// Query only indoor readings
	params := models.ReadingQueryParams{
		StationID: &station.ID,
		Location:  "indoor",
		Page:      1,
		Limit:     10,
		Order:     "desc",
	}

	response, err := dm.GetReadings(params)
	if err != nil {
		t.Fatalf("Failed to get readings: %v", err)
	}

	if response.Total != 5 {
		t.Errorf("Expected 5 indoor readings, got %d", response.Total)
	}

	readings := response.Data.([]models.SensorReading)
	for _, reading := range readings {
		if reading.SensorID != indoorSensor.ID {
			t.Errorf("Expected only indoor sensor readings, got sensor_id=%s", reading.SensorID)
		}
	}
}

func TestGetReadings_FilterByTimeRange(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	sensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")

	// Store readings over 2 hours
	now := time.Now().UTC()
	storeTestReadings(t, dm, sensor.ID, now, 120, func(i int) float64 {
		return float64(20 + i)
	})

	// Query only readings from the first hour
	startTime := now.Format(time.RFC3339)
	endTime := now.Add(1 * time.Hour).Format(time.RFC3339)

	params := models.ReadingQueryParams{
		StationID: &station.ID,
		StartTime: startTime,
		EndTime:   endTime,
		Page:      1,
		Limit:     100,
		Order:     "desc",
	}

	response, err := dm.GetReadings(params)
	if err != nil {
		t.Fatalf("Failed to get readings: %v", err)
	}

	// Should have approximately 60 readings (1 per minute for 1 hour)
	if response.Total < 59 || response.Total > 61 {
		t.Errorf("Expected approximately 60 readings in first hour, got %d", response.Total)
	}
}

func TestGetReadings_MultipleSensorIDs(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	sensor1 := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")
	sensor2 := setupTestSensor(t, dm, station.ID, models.SensorTypeHumidity, "indoor")
	sensor3 := setupTestSensor(t, dm, station.ID, models.SensorTypePressure, "indoor")

	// Store readings for all sensors
	now := time.Now().UTC()
	storeTestReadings(t, dm, sensor1.ID, now, 5, func(i int) float64 {
		return float64(20 + i)
	})
	storeTestReadings(t, dm, sensor2.ID, now, 5, func(i int) float64 {
		return float64(50 + i)
	})
	storeTestReadings(t, dm, sensor3.ID, now, 5, func(i int) float64 {
		return float64(1000 + i)
	})

	// Query only sensor1 and sensor2
	params := models.ReadingQueryParams{
		StationID: &station.ID,
		SensorIDs: []uuid.UUID{sensor1.ID, sensor2.ID},
		Page:      1,
		Limit:     20,
		Order:     "desc",
	}

	response, err := dm.GetReadings(params)
	if err != nil {
		t.Fatalf("Failed to get readings: %v", err)
	}

	if response.Total != 10 {
		t.Errorf("Expected 10 readings from 2 sensors, got %d", response.Total)
	}

	readings := response.Data.([]models.SensorReading)
	for _, reading := range readings {
		if reading.SensorID != sensor1.ID && reading.SensorID != sensor2.ID {
			t.Errorf("Expected only sensor1 or sensor2 readings, got sensor_id=%s", reading.SensorID)
		}
	}
}

func TestGetAggregatedReadings(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	sensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")

	// Store readings every minute for 2 hours
	now := time.Now().UTC().Truncate(time.Hour)
	storeTestReadings(t, dm, sensor.ID, now, 120, func(i int) float64 {
		return float64(20 + (i % 10)) // Values oscillate between 20-29
	})

	// Test 15-minute aggregation with average
	params := models.ReadingQueryParams{
		StationID:     &station.ID,
		Aggregate:     "15m",
		AggregateFunc: "avg",
		Page:          1,
		Limit:         10,
		Order:         "asc",
	}

	response, err := dm.GetAggregatedReadings(params)
	if err != nil {
		t.Fatalf("Failed to get aggregated readings: %v", err)
	}

	if !response.IsAggregated {
		t.Error("Expected IsAggregated to be true")
	}

	// Should have 8 buckets (120 minutes / 15 minutes)
	if response.Total != 8 {
		t.Errorf("Expected 8 aggregated buckets, got %d", response.Total)
	}

	readings := response.Data.([]models.AggregatedReading)
	if len(readings) != 8 {
		t.Errorf("Expected 8 aggregated readings, got %d", len(readings))
	}

	// Verify each bucket has the expected count
	for i, reading := range readings {
		if reading.Count != 15 {
			t.Errorf("Bucket %d: expected count=15, got %d", i, reading.Count)
		}
		if reading.SensorID != sensor.ID {
			t.Errorf("Bucket %d: expected sensor_id=%s, got %s", i, sensor.ID, reading.SensorID)
		}
	}
}

func TestGetAggregatedReadings_DifferentFunctions(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	sensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")

	// Store readings with known values
	now := time.Now().UTC().Truncate(time.Hour)
	values := []float64{10.0, 20.0, 30.0, 40.0, 50.0}
	for i, val := range values {
		timestamp := now.Add(time.Duration(i) * time.Minute)
		err := dm.StoreSensorReading(sensor.ID, val, timestamp)
		if err != nil {
			t.Fatalf("Failed to store reading: %v", err)
		}
	}

	testCases := []struct {
		funcName      string
		expectedValue float64
	}{
		{"min", 10.0},
		{"max", 50.0},
		{"avg", 30.0},
		{"sum", 150.0},
	}

	for _, tc := range testCases {
		t.Run(tc.funcName, func(t *testing.T) {
			params := models.ReadingQueryParams{
				StationID:     &station.ID,
				Aggregate:     "1h",
				AggregateFunc: tc.funcName,
				Page:          1,
				Limit:         10,
				Order:         "asc",
			}

			response, err := dm.GetAggregatedReadings(params)
			if err != nil {
				t.Fatalf("Failed to get aggregated readings with %s: %v", tc.funcName, err)
			}

			readings := response.Data.([]models.AggregatedReading)
			if len(readings) == 0 {
				t.Fatalf("Expected at least one aggregated reading")
			}

			// Check the aggregated value
			actualValue := readings[0].Value
			if actualValue != tc.expectedValue {
				t.Errorf("Expected %s=%f, got %f", tc.funcName, tc.expectedValue, actualValue)
			}
		})
	}
}

func TestGetAggregatedReadings_Pagination(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	sensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")

	// Store readings every minute for 5 hours
	now := time.Now().UTC().Truncate(time.Hour)
	storeTestReadings(t, dm, sensor.ID, now, 300, func(i int) float64 {
		return float64(20 + i)
	})

	// Test 1-hour aggregation with pagination
	params := models.ReadingQueryParams{
		StationID:     &station.ID,
		Aggregate:     "1h",
		AggregateFunc: "avg",
		Page:          1,
		Limit:         3,
		Order:         "asc",
	}

	response, err := dm.GetAggregatedReadings(params)
	if err != nil {
		t.Fatalf("Failed to get aggregated readings: %v", err)
	}

	// Should have 5 total buckets (300 minutes / 60 minutes)
	if response.Total != 5 {
		t.Errorf("Expected 5 total buckets, got %d", response.Total)
	}

	if response.TotalPages != 2 {
		t.Errorf("Expected 2 total pages, got %d", response.TotalPages)
	}

	readings := response.Data.([]models.AggregatedReading)
	if len(readings) != 3 {
		t.Errorf("Expected 3 readings on page 1, got %d", len(readings))
	}

	if !response.HasMore {
		t.Error("Expected HasMore to be true on page 1")
	}

	// Test page 2
	params.Page = 2
	response, err = dm.GetAggregatedReadings(params)
	if err != nil {
		t.Fatalf("Failed to get page 2: %v", err)
	}

	readings = response.Data.([]models.AggregatedReading)
	if len(readings) != 2 {
		t.Errorf("Expected 2 readings on page 2, got %d", len(readings))
	}

	if response.HasMore {
		t.Error("Expected HasMore to be false on last page")
	}
}

func TestGetAggregatedReadings_DifferentIntervals(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	sensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")

	// Store readings every minute for 2 hours
	now := time.Now().UTC().Truncate(time.Hour)
	storeTestReadings(t, dm, sensor.ID, now, 120, func(i int) float64 {
		return float64(20 + i)
	})

	testCases := []struct {
		interval      string
		expectedCount int
	}{
		{"5m", 24}, // 120 minutes / 5 minutes
		{"15m", 8}, // 120 minutes / 15 minutes
		{"1h", 2},  // 120 minutes / 60 minutes
	}

	for _, tc := range testCases {
		t.Run(tc.interval, func(t *testing.T) {
			params := models.ReadingQueryParams{
				StationID:     &station.ID,
				Aggregate:     tc.interval,
				AggregateFunc: "avg",
				Page:          1,
				Limit:         100,
				Order:         "asc",
			}

			response, err := dm.GetAggregatedReadings(params)
			if err != nil {
				t.Fatalf("Failed to get aggregated readings with interval %s: %v", tc.interval, err)
			}

			if response.Total != tc.expectedCount {
				t.Errorf("Expected %d buckets for interval %s, got %d", tc.expectedCount, tc.interval, response.Total)
			}
		})
	}
}

func TestGetAggregatedReadings_MultipleSensors(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	sensor1 := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")
	sensor2 := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "outdoor")

	// Store readings for both sensors
	now := time.Now().UTC().Truncate(time.Hour)
	storeTestReadings(t, dm, sensor1.ID, now, 60, func(i int) float64 {
		return float64(20 + i)
	})
	storeTestReadings(t, dm, sensor2.ID, now, 60, func(i int) float64 {
		return float64(10 + i)
	})

	// Query aggregated data for both sensors
	params := models.ReadingQueryParams{
		StationID:     &station.ID,
		SensorIDs:     []uuid.UUID{sensor1.ID, sensor2.ID},
		Aggregate:     "1h",
		AggregateFunc: "avg",
		Page:          1,
		Limit:         10,
		Order:         "asc",
	}

	response, err := dm.GetAggregatedReadings(params)
	if err != nil {
		t.Fatalf("Failed to get aggregated readings: %v", err)
	}

	// Should have 2 buckets (one per sensor for the 1-hour period)
	if response.Total != 2 {
		t.Errorf("Expected 2 aggregated buckets (one per sensor), got %d", response.Total)
	}

	readings := response.Data.([]models.AggregatedReading)
	sensorIDs := make(map[uuid.UUID]bool)
	for _, reading := range readings {
		sensorIDs[reading.SensorID] = true
	}

	if len(sensorIDs) != 2 {
		t.Errorf("Expected readings from 2 different sensors, got %d", len(sensorIDs))
	}
}

func TestSensorQueryParams_Defaults(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	sensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")

	// Store a reading
	now := time.Now().UTC()
	err := dm.StoreSensorReading(sensor.ID, 20.5, now)
	if err != nil {
		t.Fatalf("Failed to store reading: %v", err)
	}

	// Query with minimal params (should use defaults)
	params := models.SensorQueryParams{
		StationID: &station.ID,
	}

	sensors, err := dm.GetSensors(params)
	if err != nil {
		t.Fatalf("Failed to get sensors: %v", err)
	}

	if len(sensors) != 1 {
		t.Errorf("Expected 1 sensor, got %d", len(sensors))
	}
}

func TestGetReadings_EmptyResult(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	sensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")

	// Don't store any readings

	params := models.ReadingQueryParams{
		StationID: &station.ID,
		SensorIDs: []uuid.UUID{sensor.ID},
		Page:      1,
		Limit:     10,
		Order:     "desc",
	}

	response, err := dm.GetReadings(params)
	if err != nil {
		t.Fatalf("Failed to get readings: %v", err)
	}

	if response.Total != 0 {
		t.Errorf("Expected total=0, got %d", response.Total)
	}

	readings := response.Data.([]models.SensorReading)
	if len(readings) != 0 {
		t.Errorf("Expected 0 readings, got %d", len(readings))
	}

	if response.TotalPages != 1 {
		t.Errorf("Expected 1 total page (even with no data), got %d", response.TotalPages)
	}

	if response.HasMore {
		t.Error("Expected HasMore to be false")
	}
}

func TestGetAggregatedReadings_EmptyResult(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	sensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")

	// Don't store any readings

	params := models.ReadingQueryParams{
		StationID:     &station.ID,
		SensorIDs:     []uuid.UUID{sensor.ID},
		Aggregate:     "1h",
		AggregateFunc: "avg",
		Page:          1,
		Limit:         10,
		Order:         "desc",
	}

	response, err := dm.GetAggregatedReadings(params)
	if err != nil {
		t.Fatalf("Failed to get aggregated readings: %v", err)
	}

	if response.Total != 0 {
		t.Errorf("Expected total=0, got %d", response.Total)
	}

	readings := response.Data.([]models.AggregatedReading)
	if len(readings) != 0 {
		t.Errorf("Expected 0 aggregated readings, got %d", len(readings))
	}

	if !response.IsAggregated {
		t.Error("Expected IsAggregated to be true")
	}
}

func TestGetReadings_OrderAscending(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	sensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")

	// Store readings with increasing timestamps
	now := time.Now().UTC()
	storeTestReadings(t, dm, sensor.ID, now, 5, func(i int) float64 {
		return float64(20 + i)
	})

	// Query with ascending order
	params := models.ReadingQueryParams{
		StationID: &station.ID,
		Page:      1,
		Limit:     10,
		Order:     "asc",
	}

	response, err := dm.GetReadings(params)
	if err != nil {
		t.Fatalf("Failed to get readings: %v", err)
	}

	readings := response.Data.([]models.SensorReading)
	if len(readings) != 5 {
		t.Errorf("Expected 5 readings, got %d", len(readings))
	}

	// Verify ascending order
	for i := 0; i < len(readings)-1; i++ {
		if readings[i].DateUTC.After(readings[i+1].DateUTC) {
			t.Errorf("Readings not in ascending order at index %d", i)
		}
	}

	// Verify values are in ascending order (20, 21, 22, 23, 24)
	for i, reading := range readings {
		expectedValue := float64(20 + i)
		if reading.Value != expectedValue {
			t.Errorf("Expected value=%f at index %d, got %f", expectedValue, i, reading.Value)
		}
	}
}

func TestGetReadings_OrderDescending(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	sensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")

	// Store readings with increasing timestamps
	now := time.Now().UTC()
	storeTestReadings(t, dm, sensor.ID, now, 5, func(i int) float64 {
		return float64(20 + i)
	})

	// Query with descending order (default)
	params := models.ReadingQueryParams{
		StationID: &station.ID,
		Page:      1,
		Limit:     10,
		Order:     "desc",
	}

	response, err := dm.GetReadings(params)
	if err != nil {
		t.Fatalf("Failed to get readings: %v", err)
	}

	readings := response.Data.([]models.SensorReading)
	if len(readings) != 5 {
		t.Errorf("Expected 5 readings, got %d", len(readings))
	}

	// Verify descending order
	for i := 0; i < len(readings)-1; i++ {
		if readings[i].DateUTC.Before(readings[i+1].DateUTC) {
			t.Errorf("Readings not in descending order at index %d", i)
		}
	}

	// Verify values are in descending order (24, 23, 22, 21, 20)
	for i, reading := range readings {
		expectedValue := float64(24 - i)
		if reading.Value != expectedValue {
			t.Errorf("Expected value=%f at index %d, got %f", expectedValue, i, reading.Value)
		}
	}
}

func TestGetReadings_TimeRangeFilter(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	sensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")

	// Store readings over 10 minutes
	now := time.Now().UTC().Truncate(time.Second)
	storeTestReadings(t, dm, sensor.ID, now, 10, func(i int) float64 {
		return float64(20 + i)
	})

	// Query for readings between minute 3 and minute 7
	startTime := now.Add(3 * time.Minute)
	endTime := now.Add(7 * time.Minute)

	params := models.ReadingQueryParams{
		StationID: &station.ID,
		StartTime: startTime.Format(time.RFC3339),
		EndTime:   endTime.Format(time.RFC3339),
		Page:      1,
		Limit:     10,
		Order:     "asc",
	}

	response, err := dm.GetReadings(params)
	if err != nil {
		t.Fatalf("Failed to get readings: %v", err)
	}

	readings := response.Data.([]models.SensorReading)

	// Should have 5 readings (minutes 3, 4, 5, 6, 7)
	if len(readings) != 5 {
		t.Errorf("Expected 5 readings in time range, got %d", len(readings))
	}

	// Verify all readings are within the time range
	for _, reading := range readings {
		if reading.DateUTC.Before(startTime) || reading.DateUTC.After(endTime) {
			t.Errorf("Reading timestamp %v is outside time range [%v, %v]",
				reading.DateUTC, startTime, endTime)
		}
	}
}

func TestGetReadings_MultipleSensorsFilter(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	sensor1 := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")
	sensor2 := setupTestSensor(t, dm, station.ID, models.SensorTypeHumidity, "indoor")
	sensor3 := setupTestSensor(t, dm, station.ID, models.SensorTypePressure, "indoor")

	// Store readings for all sensors
	now := time.Now().UTC()
	storeTestReadings(t, dm, sensor1.ID, now, 5, func(i int) float64 {
		return float64(20 + i)
	})
	storeTestReadings(t, dm, sensor2.ID, now, 5, func(i int) float64 {
		return float64(50 + i)
	})
	storeTestReadings(t, dm, sensor3.ID, now, 5, func(i int) float64 {
		return float64(1000 + i)
	})

	// Query for sensor1 and sensor2 only
	params := models.ReadingQueryParams{
		StationID: &station.ID,
		SensorIDs: []uuid.UUID{sensor1.ID, sensor2.ID},
		Page:      1,
		Limit:     20,
		Order:     "asc",
	}

	response, err := dm.GetReadings(params)
	if err != nil {
		t.Fatalf("Failed to get readings: %v", err)
	}

	readings := response.Data.([]models.SensorReading)

	// Should have 10 readings (5 from each sensor)
	if len(readings) != 10 {
		t.Errorf("Expected 10 readings, got %d", len(readings))
	}

	// Verify only sensor1 and sensor2 readings are returned
	for _, reading := range readings {
		if reading.SensorID != sensor1.ID && reading.SensorID != sensor2.ID {
			t.Errorf("Unexpected sensor_id=%s in results", reading.SensorID)
		}
	}

	// Count readings per sensor
	sensor1Count := 0
	sensor2Count := 0
	for _, reading := range readings {
		if reading.SensorID == sensor1.ID {
			sensor1Count++
		} else if reading.SensorID == sensor2.ID {
			sensor2Count++
		}
	}

	if sensor1Count != 5 {
		t.Errorf("Expected 5 readings from sensor1, got %d", sensor1Count)
	}
	if sensor2Count != 5 {
		t.Errorf("Expected 5 readings from sensor2, got %d", sensor2Count)
	}
}

func TestGetReadings_SensorTypeFilter(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	tempSensor1 := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")
	tempSensor2 := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "outdoor")
	humSensor := setupTestSensor(t, dm, station.ID, models.SensorTypeHumidity, "indoor")

	// Store readings for all sensors
	now := time.Now().UTC()
	storeTestReadings(t, dm, tempSensor1.ID, now, 5, func(i int) float64 {
		return float64(20 + i)
	})
	storeTestReadings(t, dm, tempSensor2.ID, now, 5, func(i int) float64 {
		return float64(10 + i)
	})
	storeTestReadings(t, dm, humSensor.ID, now, 5, func(i int) float64 {
		return float64(50 + i)
	})

	// Query for temperature sensors only
	params := models.ReadingQueryParams{
		StationID:  &station.ID,
		SensorType: models.SensorTypeTemperature,
		Page:       1,
		Limit:      20,
		Order:      "asc",
	}

	response, err := dm.GetReadings(params)
	if err != nil {
		t.Fatalf("Failed to get readings: %v", err)
	}

	readings := response.Data.([]models.SensorReading)

	// Should have 10 readings (5 from each temperature sensor)
	if len(readings) != 10 {
		t.Errorf("Expected 10 temperature readings, got %d", len(readings))
	}

	// Verify only temperature sensor readings are returned
	for _, reading := range readings {
		if reading.SensorID != tempSensor1.ID && reading.SensorID != tempSensor2.ID {
			t.Errorf("Unexpected sensor_id=%s in temperature results", reading.SensorID)
		}
	}
}

func TestGetReadings_CombinedFilters(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	indoorTemp := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")
	outdoorTemp := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "outdoor")
	indoorHum := setupTestSensor(t, dm, station.ID, models.SensorTypeHumidity, "indoor")

	// Store readings for all sensors
	now := time.Now().UTC().Truncate(time.Second)
	storeTestReadings(t, dm, indoorTemp.ID, now, 10, func(i int) float64 {
		return float64(20 + i)
	})
	storeTestReadings(t, dm, outdoorTemp.ID, now, 10, func(i int) float64 {
		return float64(10 + i)
	})
	storeTestReadings(t, dm, indoorHum.ID, now, 10, func(i int) float64 {
		return float64(50 + i)
	})

	// Query for indoor temperature sensors with time range
	startTime := now.Add(3 * time.Minute)
	endTime := now.Add(7 * time.Minute)

	params := models.ReadingQueryParams{
		StationID:  &station.ID,
		SensorType: models.SensorTypeTemperature,
		Location:   "indoor",
		StartTime:  startTime.Format(time.RFC3339),
		EndTime:    endTime.Format(time.RFC3339),
		Page:       1,
		Limit:      20,
		Order:      "asc",
	}

	response, err := dm.GetReadings(params)
	if err != nil {
		t.Fatalf("Failed to get readings: %v", err)
	}

	readings := response.Data.([]models.SensorReading)

	// Should have 5 readings (minutes 3-7 from indoor temperature sensor)
	if len(readings) != 5 {
		t.Errorf("Expected 5 readings, got %d", len(readings))
	}

	// Verify all readings match the filters
	for _, reading := range readings {
		if reading.SensorID != indoorTemp.ID {
			t.Errorf("Expected only indoor temperature sensor readings, got sensor_id=%s", reading.SensorID)
		}
		if reading.DateUTC.Before(startTime) || reading.DateUTC.After(endTime) {
			t.Errorf("Reading timestamp %v is outside time range [%v, %v]",
				reading.DateUTC, startTime, endTime)
		}
	}
}

func TestGetAggregatedReadings_GroupBySensor(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	sensor1 := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")
	sensor2 := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "outdoor")

	// Store readings for both sensors
	now := time.Now().UTC().Truncate(time.Hour)
	storeTestReadings(t, dm, sensor1.ID, now, 60, func(i int) float64 {
		return float64(20 + i)
	})
	storeTestReadings(t, dm, sensor2.ID, now, 60, func(i int) float64 {
		return float64(10 + i)
	})

	// Query with group_by=sensor
	params := models.ReadingQueryParams{
		StationID:     &station.ID,
		Aggregate:     "1h",
		AggregateFunc: "avg",
		GroupBy:       "sensor",
		Page:          1,
		Limit:         10,
		Order:         "asc",
	}

	response, err := dm.GetAggregatedReadings(params)
	if err != nil {
		t.Fatalf("Failed to get aggregated readings: %v", err)
	}

	readings := response.Data.([]models.AggregatedReading)

	// Should have 2 buckets (one per sensor)
	if len(readings) != 2 {
		t.Errorf("Expected 2 aggregated readings (one per sensor), got %d", len(readings))
	}

	// Verify we have readings from both sensors
	sensorIDs := make(map[uuid.UUID]bool)
	for _, reading := range readings {
		sensorIDs[reading.SensorID] = true
	}

	if !sensorIDs[sensor1.ID] || !sensorIDs[sensor2.ID] {
		t.Error("Expected readings from both sensors")
	}
}

func TestGetAggregatedReadings_GroupByLocation(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	indoorSensor1 := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")
	indoorSensor2 := setupTestSensor(t, dm, station.ID, models.SensorTypeHumidity, "indoor")
	outdoorSensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "outdoor")

	// Store readings for all sensors
	now := time.Now().UTC().Truncate(time.Hour)
	storeTestReadings(t, dm, indoorSensor1.ID, now, 60, func(i int) float64 {
		return float64(20 + i)
	})
	storeTestReadings(t, dm, indoorSensor2.ID, now, 60, func(i int) float64 {
		return float64(50 + i)
	})
	storeTestReadings(t, dm, outdoorSensor.ID, now, 60, func(i int) float64 {
		return float64(10 + i)
	})

	// Query with group_by=location
	params := models.ReadingQueryParams{
		StationID:     &station.ID,
		Aggregate:     "1h",
		AggregateFunc: "avg",
		GroupBy:       "location",
		Page:          1,
		Limit:         10,
		Order:         "asc",
	}

	response, err := dm.GetAggregatedReadings(params)
	if err != nil {
		t.Fatalf("Failed to get aggregated readings: %v", err)
	}

	readings := response.Data.([]models.AggregatedReading)

	// Should have 2 buckets (indoor sensor + outdoor sensor)
	if len(readings) != 2 {
		t.Errorf("Expected 2 aggregated readings, got %d", len(readings))
	}
}

func TestGetAggregatedReadings_GroupBySensorType(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	tempSensor1 := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")
	tempSensor2 := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "outdoor")
	humSensor := setupTestSensor(t, dm, station.ID, models.SensorTypeHumidity, "indoor")

	// Store readings for all sensors
	now := time.Now().UTC().Truncate(time.Hour)
	storeTestReadings(t, dm, tempSensor1.ID, now, 60, func(i int) float64 {
		return float64(20 + i)
	})
	storeTestReadings(t, dm, tempSensor2.ID, now, 60, func(i int) float64 {
		return float64(10 + i)
	})
	storeTestReadings(t, dm, humSensor.ID, now, 60, func(i int) float64 {
		return float64(50 + i)
	})

	// Query with group_by=sensor_type
	params := models.ReadingQueryParams{
		StationID:     &station.ID,
		Aggregate:     "1h",
		AggregateFunc: "avg",
		GroupBy:       "sensor_type",
		Page:          1,
		Limit:         10,
		Order:         "asc",
	}

	response, err := dm.GetAggregatedReadings(params)
	if err != nil {
		t.Fatalf("Failed to get aggregated readings: %v", err)
	}

	readings := response.Data.([]models.AggregatedReading)

	// Should have 2 buckets (temperature sensor + humidity sensor)
	if len(readings) != 2 {
		t.Errorf("Expected 2 aggregated readings, got %d", len(readings))
	}
}

func TestGetAggregatedReadings_DifferentAggregateFunctions(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	sensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")

	// Store readings with known values
	now := time.Now().UTC().Truncate(time.Hour)
	values := []float64{10.0, 20.0, 30.0, 40.0, 50.0}
	for i, val := range values {
		err := dm.StoreSensorReading(sensor.ID, val, now.Add(time.Duration(i)*time.Minute))
		if err != nil {
			t.Fatalf("Failed to store reading: %v", err)
		}
	}

	testCases := []struct {
		function      string
		expectedValue float64
	}{
		{"avg", 30.0},  // (10+20+30+40+50)/5
		{"min", 10.0},  // minimum value
		{"max", 50.0},  // maximum value
		{"sum", 150.0}, // 10+20+30+40+50
		{"count", 5.0}, // number of readings
	}

	for _, tc := range testCases {
		t.Run(tc.function, func(t *testing.T) {
			params := models.ReadingQueryParams{
				StationID:     &station.ID,
				Aggregate:     "1h",
				AggregateFunc: tc.function,
				Page:          1,
				Limit:         10,
				Order:         "asc",
			}

			response, err := dm.GetAggregatedReadings(params)
			if err != nil {
				t.Fatalf("Failed to get aggregated readings with function %s: %v", tc.function, err)
			}

			readings := response.Data.([]models.AggregatedReading)
			if len(readings) != 1 {
				t.Errorf("Expected 1 aggregated reading, got %d", len(readings))
			}

			if len(readings) > 0 {
				if readings[0].Value != tc.expectedValue {
					t.Errorf("Expected %s value=%f, got %f", tc.function, tc.expectedValue, readings[0].Value)
				}
			}
		})
	}
}

func TestGetAggregatedReadings_MinMaxValues(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	sensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")

	// Store readings with known values
	now := time.Now().UTC().Truncate(time.Hour)
	values := []float64{15.0, 25.0, 35.0, 45.0, 55.0}
	for i, val := range values {
		err := dm.StoreSensorReading(sensor.ID, val, now.Add(time.Duration(i)*time.Minute))
		if err != nil {
			t.Fatalf("Failed to store reading: %v", err)
		}
	}

	params := models.ReadingQueryParams{
		StationID:     &station.ID,
		Aggregate:     "1h",
		AggregateFunc: "avg",
		Page:          1,
		Limit:         10,
		Order:         "asc",
	}

	response, err := dm.GetAggregatedReadings(params)
	if err != nil {
		t.Fatalf("Failed to get aggregated readings: %v", err)
	}

	readings := response.Data.([]models.AggregatedReading)
	if len(readings) != 1 {
		t.Fatalf("Expected 1 aggregated reading, got %d", len(readings))
	}

	reading := readings[0]

	// Verify min and max values
	if reading.MinValue != 15.0 {
		t.Errorf("Expected min_value=15.0, got %f", reading.MinValue)
	}

	if reading.MaxValue != 55.0 {
		t.Errorf("Expected max_value=55.0, got %f", reading.MaxValue)
	}

	// Verify count
	if reading.Count != 5 {
		t.Errorf("Expected count=5, got %d", reading.Count)
	}
}

func TestGetAggregatedReadings_TimeRangeFilter(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	station := setupTestStation(t, dm)
	sensor := setupTestSensor(t, dm, station.ID, models.SensorTypeTemperature, "indoor")

	// Store readings over 3 hours
	now := time.Now().UTC().Truncate(time.Hour)
	for i := 0; i < 180; i++ { // 3 hours * 60 minutes
		err := dm.StoreSensorReading(sensor.ID, float64(20+i), now.Add(time.Duration(i)*time.Minute))
		if err != nil {
			t.Fatalf("Failed to store reading: %v", err)
		}
	}

	// Query for middle hour only
	startTime := now.Add(1 * time.Hour)
	endTime := now.Add(2 * time.Hour).Add(-1 * time.Second)

	params := models.ReadingQueryParams{
		StationID:     &station.ID,
		StartTime:     startTime.Format(time.RFC3339),
		EndTime:       endTime.Format(time.RFC3339),
		Aggregate:     "1h",
		AggregateFunc: "avg",
		Page:          1,
		Limit:         10,
		Order:         "asc",
	}

	response, err := dm.GetAggregatedReadings(params)
	if err != nil {
		t.Fatalf("Failed to get aggregated readings: %v", err)
	}

	readings := response.Data.([]models.AggregatedReading)

	// Should have 1 bucket (the middle hour)
	if len(readings) != 1 {
		t.Errorf("Expected 1 aggregated reading for middle hour, got %d", len(readings))
	}

	if len(readings) > 0 {
		// Verify the bucket is within the time range
		if readings[0].DateUTC.Before(startTime) || readings[0].DateUTC.After(endTime) {
			t.Errorf("Aggregated reading timestamp %v is outside time range [%v, %v]",
				readings[0].DateUTC, startTime, endTime)
		}
	}
}

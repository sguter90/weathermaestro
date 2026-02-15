package ecowitt

import (
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sguter90/weathermaestro/pkg/models"
)

func TestPusher_GetEndpoint(t *testing.T) {
	pusher := &Pusher{}
	endpoint := pusher.GetEndpoint()

	expected := "/data/report"
	if endpoint != expected {
		t.Errorf("Expected endpoint %s, got %s", expected, endpoint)
	}
}

func TestPusher_GetStationType(t *testing.T) {
	pusher := &Pusher{}
	stationType := pusher.GetStationType()

	expected := "Ecowitt"
	if stationType != expected {
		t.Errorf("Expected station type %s, got %s", expected, stationType)
	}
}

func TestPusher_ParseStation(t *testing.T) {
	pusher := &Pusher{}

	testCases := []struct {
		name     string
		params   url.Values
		expected models.StationData
	}{
		{
			name: "Complete station data",
			params: url.Values{
				"PASSKEY":     []string{"ABC123"},
				"stationtype": []string{"EasyWeatherV1.6.4"},
				"model":       []string{"GW1000B_V1.6.8"},
				"freq":        []string{"868M"},
			},
			expected: models.StationData{
				PassKey:     "ABC123",
				StationType: "EasyWeatherV1.6.4",
				Model:       "GW1000B_V1.6.8",
				Freq:        "868M",
				Mode:        "push",
			},
		},
		{
			name: "Minimal station data",
			params: url.Values{
				"PASSKEY": []string{"XYZ789"},
			},
			expected: models.StationData{
				PassKey: "XYZ789",
				Mode:    "push",
			},
		},
		{
			name:   "Empty params",
			params: url.Values{},
			expected: models.StationData{
				Mode: "push",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := pusher.ParseStation(tc.params)

			if result.PassKey != tc.expected.PassKey {
				t.Errorf("Expected PassKey %s, got %s", tc.expected.PassKey, result.PassKey)
			}
			if result.StationType != tc.expected.StationType {
				t.Errorf("Expected StationType %s, got %s", tc.expected.StationType, result.StationType)
			}
			if result.Model != tc.expected.Model {
				t.Errorf("Expected Model %s, got %s", tc.expected.Model, result.Model)
			}
			if result.Freq != tc.expected.Freq {
				t.Errorf("Expected Freq %s, got %s", tc.expected.Freq, result.Freq)
			}
			if result.Mode != tc.expected.Mode {
				t.Errorf("Expected Mode %s, got %s", tc.expected.Mode, result.Mode)
			}
		})
	}
}

func TestPusher_ParseSensors(t *testing.T) {
	pusher := &Pusher{}

	testCases := []struct {
		name          string
		params        url.Values
		expectedCount int
		expectedTypes []string
	}{
		{
			name: "Multiple sensors",
			params: url.Values{
				"tempf":        []string{"72.5"},
				"humidity":     []string{"65"},
				"baromrelin":   []string{"29.92"},
				"windspeedmph": []string{"5.2"},
			},
			expectedCount: 4,
			expectedTypes: []string{
				models.SensorTypeTemperature,
				models.SensorTypeHumidity,
				models.SensorTypePressureRelative,
				models.SensorTypeWindSpeed,
			},
		},
		{
			name: "Single sensor",
			params: url.Values{
				"tempf": []string{"68.0"},
			},
			expectedCount: 1,
			expectedTypes: []string{models.SensorTypeTemperature},
		},
		{
			name:          "No matching sensors",
			params:        url.Values{},
			expectedCount: 0,
			expectedTypes: []string{},
		},
		{
			name: "Unknown sensor ignored",
			params: url.Values{
				"tempf":       []string{"72.5"},
				"unknown_key": []string{"123"},
			},
			expectedCount: 1,
			expectedTypes: []string{models.SensorTypeTemperature},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := pusher.ParseSensors(tc.params)

			if len(result) != tc.expectedCount {
				t.Errorf("Expected %d sensors, got %d", tc.expectedCount, len(result))
			}

			for _, expectedType := range tc.expectedTypes {
				found := false
				for _, sensor := range result {
					if sensor.SensorType == expectedType {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected sensor type %s not found", expectedType)
				}
			}
		})
	}
}

func TestPusher_ParseWeatherData_Temperature(t *testing.T) {
	pusher := &Pusher{}

	sensorID := uuid.New()
	sensors := map[string]models.Sensor{
		"tempf": {
			ID:         sensorID,
			RemoteID:   "tempf",
			SensorType: models.SensorTypeTemperature,
		},
	}

	params := url.Values{
		"tempf":   []string{"72.5"}, // 72.5°F = 22.5°C
		"dateutc": []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 reading, got %d", len(result))
	}

	reading := result[sensorID]
	expectedTemp := 22.5
	tolerance := 0.1

	if reading.SensorID != sensorID {
		t.Errorf("Expected sensor ID %s, got %s", sensorID, reading.SensorID)
	}

	if reading.Value < expectedTemp-tolerance || reading.Value > expectedTemp+tolerance {
		t.Errorf("Expected temperature ~%.1f°C, got %.1f°C", expectedTemp, reading.Value)
	}
}

func TestPusher_ParseWeatherData_Humidity(t *testing.T) {
	pusher := &Pusher{}

	sensorID := uuid.New()
	sensors := map[string]models.Sensor{
		"humidity": {
			ID:         sensorID,
			RemoteID:   "humidity",
			SensorType: models.SensorTypeHumidity,
		},
	}

	params := url.Values{
		"humidity": []string{"65"},
		"dateutc":  []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 reading, got %d", len(result))
	}

	reading := result[sensorID]
	if reading.Value != 65.0 {
		t.Errorf("Expected humidity 65%%, got %.1f%%", reading.Value)
	}
}

func TestPusher_ParseWeatherData_Pressure(t *testing.T) {
	pusher := &Pusher{}

	sensorID := uuid.New()
	sensors := map[string]models.Sensor{
		"baromrelin": {
			ID:         sensorID,
			RemoteID:   "baromrelin",
			SensorType: models.SensorTypePressureRelative,
		},
	}

	params := url.Values{
		"baromrelin": []string{"29.92"}, // 29.92 inHg ≈ 1013.25 hPa
		"dateutc":    []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 reading, got %d", len(result))
	}

	reading := result[sensorID]
	expectedPressure := 1013.25
	tolerance := 1.0

	if reading.Value < expectedPressure-tolerance || reading.Value > expectedPressure+tolerance {
		t.Errorf("Expected pressure ~%.2f hPa, got %.2f hPa", expectedPressure, reading.Value)
	}
}

func TestPusher_ParseWeatherData_WindSpeed(t *testing.T) {
	pusher := &Pusher{}

	sensorID := uuid.New()
	sensors := map[string]models.Sensor{
		"windspeedmph": {
			ID:         sensorID,
			RemoteID:   "windspeedmph",
			SensorType: models.SensorTypeWindSpeed,
		},
	}

	params := url.Values{
		"windspeedmph": []string{"10.0"}, // 10 mph ≈ 4.47 m/s
		"dateutc":      []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 reading, got %d", len(result))
	}

	reading := result[sensorID]
	expectedSpeed := 4.47
	tolerance := 0.1

	if reading.Value < expectedSpeed-tolerance || reading.Value > expectedSpeed+tolerance {
		t.Errorf("Expected wind speed ~%.2f m/s, got %.2f m/s", expectedSpeed, reading.Value)
	}
}

func TestPusher_ParseWeatherData_WindDirection(t *testing.T) {
	pusher := &Pusher{}

	sensorID := uuid.New()
	sensors := map[string]models.Sensor{
		"winddir": {
			ID:         sensorID,
			RemoteID:   "winddir",
			SensorType: models.SensorTypeWindDirection,
		},
	}

	params := url.Values{
		"winddir": []string{"180"},
		"dateutc": []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 reading, got %d", len(result))
	}

	reading := result[sensorID]
	if reading.Value != 180.0 {
		t.Errorf("Expected wind direction 180°, got %.1f°", reading.Value)
	}
}

func TestPusher_ParseWeatherData_Rainfall(t *testing.T) {
	pusher := &Pusher{}

	sensorID := uuid.New()
	sensors := map[string]models.Sensor{
		"dailyrainin": {
			ID:         sensorID,
			RemoteID:   "dailyrainin",
			SensorType: models.SensorTypeRainfallDaily,
		},
	}

	params := url.Values{
		"dailyrainin": []string{"0.5"}, // 0.5 inches = 12.7 mm
		"dateutc":     []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 reading, got %d", len(result))
	}

	reading := result[sensorID]
	expectedRain := 12.7
	tolerance := 0.1

	if reading.Value < expectedRain-tolerance || reading.Value > expectedRain+tolerance {
		t.Errorf("Expected rainfall ~%.1f mm, got %.1f mm", expectedRain, reading.Value)
	}
}

func TestPusher_ParseWeatherData_SolarRadiation(t *testing.T) {
	pusher := &Pusher{}

	sensorID := uuid.New()
	sensors := map[string]models.Sensor{
		"solarradiation": {
			ID:         sensorID,
			RemoteID:   "solarradiation",
			SensorType: models.SensorTypeSolarRadiation,
		},
	}

	params := url.Values{
		"solarradiation": []string{"450.5"},
		"dateutc":        []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 reading, got %d", len(result))
	}

	reading := result[sensorID]
	if reading.Value != 450.5 {
		t.Errorf("Expected solar radiation 450.5 W/m², got %.1f W/m²", reading.Value)
	}
}

func TestPusher_ParseWeatherData_UVIndex(t *testing.T) {
	pusher := &Pusher{}

	sensorID := uuid.New()
	sensors := map[string]models.Sensor{
		"uv": {
			ID:         sensorID,
			RemoteID:   "uv",
			SensorType: models.SensorTypeUVIndex,
		},
	}

	params := url.Values{
		"uv":      []string{"7"},
		"dateutc": []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 reading, got %d", len(result))
	}

	reading := result[sensorID]
	if reading.Value != 7.0 {
		t.Errorf("Expected UV index 7, got %.1f", reading.Value)
	}
}

func TestPusher_ParseWeatherData_MultipleSensors(t *testing.T) {
	pusher := &Pusher{}

	tempID := uuid.New()
	humidityID := uuid.New()
	pressureID := uuid.New()

	sensors := map[string]models.Sensor{
		"tempf": {
			ID:         tempID,
			RemoteID:   "tempf",
			SensorType: models.SensorTypeTemperature,
		},
		"humidity": {
			ID:         humidityID,
			RemoteID:   "humidity",
			SensorType: models.SensorTypeHumidity,
		},
		"baromrelin": {
			ID:         pressureID,
			RemoteID:   "baromrelin",
			SensorType: models.SensorTypePressureRelative,
		},
	}

	params := url.Values{
		"tempf":      []string{"68.0"},
		"humidity":   []string{"50"},
		"baromrelin": []string{"30.00"},
		"dateutc":    []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("Expected 3 readings, got %d", len(result))
	}

	// Verify all sensors have readings
	if _, ok := result[tempID]; !ok {
		t.Error("Expected temperature reading")
	}
	if _, ok := result[humidityID]; !ok {
		t.Error("Expected humidity reading")
	}
	if _, ok := result[pressureID]; !ok {
		t.Error("Expected pressure reading")
	}
}

func TestPusher_ParseWeatherData_MissingSensorValue(t *testing.T) {
	pusher := &Pusher{}

	tempID := uuid.New()
	humidityID := uuid.New()

	sensors := map[string]models.Sensor{
		"tempf": {
			ID:         tempID,
			RemoteID:   "tempf",
			SensorType: models.SensorTypeTemperature,
		},
		"humidity": {
			ID:         humidityID,
			RemoteID:   "humidity",
			SensorType: models.SensorTypeHumidity,
		},
	}

	// Only provide temperature, not humidity
	params := url.Values{
		"tempf":   []string{"68.0"},
		"dateutc": []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 reading, got %d", len(result))
	}

	// Only temperature should be present
	if _, ok := result[tempID]; !ok {
		t.Error("Expected temperature reading")
	}
	if _, ok := result[humidityID]; ok {
		t.Error("Did not expect humidity reading")
	}
}

func TestPusher_ParseWeatherData_InvalidValues(t *testing.T) {
	pusher := &Pusher{}

	tempID := uuid.New()
	humidityID := uuid.New()

	sensors := map[string]models.Sensor{
		"tempf": {
			ID:         tempID,
			RemoteID:   "tempf",
			SensorType: models.SensorTypeTemperature,
		},
		"humidity": {
			ID:         humidityID,
			RemoteID:   "humidity",
			SensorType: models.SensorTypeHumidity,
		},
	}

	params := url.Values{
		"tempf":    []string{"invalid"},
		"humidity": []string{"not_a_number"},
		"dateutc":  []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should return empty result since values are invalid
	if len(result) != 0 {
		t.Errorf("Expected 0 readings for invalid values, got %d", len(result))
	}
}

func TestPusher_ParseWeatherData_DateParsing(t *testing.T) {
	pusher := &Pusher{}

	sensorID := uuid.New()
	sensors := map[string]models.Sensor{
		"tempf": {
			ID:         sensorID,
			RemoteID:   "tempf",
			SensorType: models.SensorTypeTemperature,
		},
	}

	testCases := []struct {
		name        string
		dateStr     string
		shouldParse bool
	}{
		{
			name:        "Standard format with space",
			dateStr:     "2024-01-15 12:00:00",
			shouldParse: true,
		},
		{
			name:        "Invalid format",
			dateStr:     "invalid-date",
			shouldParse: false,
		},
		{
			name:        "Empty date",
			dateStr:     "",
			shouldParse: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := url.Values{
				"tempf": []string{"68.0"},
			}
			if tc.dateStr != "" {
				params.Set("dateutc", tc.dateStr)
			}

			result, err := pusher.ParseWeatherData(params, sensors)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if len(result) != 1 {
				t.Fatalf("Expected 1 reading, got %d", len(result))
			}

			reading := result[sensorID]
			if tc.shouldParse {
				// Check if date was parsed correctly
				expectedTime, _ := time.Parse("2006-01-02 15:04:05", tc.dateStr)
				if !reading.DateUTC.Equal(expectedTime) {
					t.Errorf("Expected date %v, got %v", expectedTime, reading.DateUTC)
				}
			} else {
				// Should use current time if parsing fails
				if reading.DateUTC.IsZero() {
					t.Error("Expected non-zero date when parsing fails")
				}
			}
		})
	}
}

func TestPusher_ParseWeatherData_EmptyParams(t *testing.T) {
	pusher := &Pusher{}

	sensors := map[string]models.Sensor{
		"tempf": {
			ID:         uuid.New(),
			RemoteID:   "tempf",
			SensorType: models.SensorTypeTemperature,
		},
	}

	params := url.Values{}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected 0 readings for empty params, got %d", len(result))
	}
}

func TestPusher_ParseWeatherData_EmptySensors(t *testing.T) {
	pusher := &Pusher{}

	sensors := map[string]models.Sensor{}

	params := url.Values{
		"tempf":   []string{"68.0"},
		"dateutc": []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected 0 readings for empty sensors, got %d", len(result))
	}
}

func TestPusher_ParseWeatherData_AllRainfallTypes(t *testing.T) {
	pusher := &Pusher{}

	rainfallTypes := []struct {
		remoteID   string
		sensorType string
	}{
		{"rainratein", models.SensorTypeRainfallRate},
		{"eventrainin", models.SensorTypeRainfallEvent},
		{"hourlyrainin", models.SensorTypeRainfallHourly},
		{"dailyrainin", models.SensorTypeRainfallDaily},
		{"weeklyrainin", models.SensorTypeRainfallWeekly},
		{"monthlyrainin", models.SensorTypeRainfallMonthly},
		{"yearlyrainin", models.SensorTypeRainfallYearly},
		{"totalrainin", models.SensorTypeRainfallTotal},
	}

	sensors := make(map[string]models.Sensor)
	params := url.Values{
		"dateutc": []string{"2024-01-15 12:00:00"},
	}

	for _, rt := range rainfallTypes {
		sensorID := uuid.New()
		sensors[rt.remoteID] = models.Sensor{
			ID:         sensorID,
			RemoteID:   rt.remoteID,
			SensorType: rt.sensorType,
		}
		params.Set(rt.remoteID, "1.0") // 1 inch = 25.4 mm
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != len(rainfallTypes) {
		t.Fatalf("Expected %d readings, got %d", len(rainfallTypes), len(result))
	}

	// Verify all rainfall readings are converted correctly
	expectedValue := 25.4
	tolerance := 0.1
	for _, sensor := range sensors {
		reading, ok := result[sensor.ID]
		if !ok {
			t.Errorf("Missing reading for sensor %s", sensor.RemoteID)
			continue
		}
		if reading.Value < expectedValue-tolerance || reading.Value > expectedValue+tolerance {
			t.Errorf("Sensor %s: expected ~%.1f mm, got %.1f mm",
				sensor.RemoteID, expectedValue, reading.Value)
		}
	}
}

func TestPusher_ParseWeatherData_AllWindTypes(t *testing.T) {
	pusher := &Pusher{}

	windSensors := []struct {
		remoteID   string
		sensorType string
		value      string
		expected   float64
	}{
		{"windspeedmph", models.SensorTypeWindSpeed, "10.0", 4.47},
		{"windgustmph", models.SensorTypeWindGust, "15.0", 6.71},
		{"maxdailygust", models.SensorTypeWindGustMaxDaily, "20.0", 8.94},
		{"winddir", models.SensorTypeWindDirection, "180", 180.0},
	}

	sensors := make(map[string]models.Sensor)
	params := url.Values{
		"dateutc": []string{"2024-01-15 12:00:00"},
	}

	for _, ws := range windSensors {
		sensorID := uuid.New()
		sensors[ws.remoteID] = models.Sensor{
			ID:         sensorID,
			RemoteID:   ws.remoteID,
			SensorType: ws.sensorType,
		}
		params.Set(ws.remoteID, ws.value)
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != len(windSensors) {
		t.Fatalf("Expected %d readings, got %d", len(windSensors), len(result))
	}

	// Verify all wind readings
	tolerance := 0.1
	for i, ws := range windSensors {
		sensor := sensors[ws.remoteID]
		reading, ok := result[sensor.ID]
		if !ok {
			t.Errorf("Missing reading for sensor %s", ws.remoteID)
			continue
		}
		if reading.Value < ws.expected-tolerance || reading.Value > ws.expected+tolerance {
			t.Errorf("Sensor %s: expected ~%.2f, got %.2f",
				ws.remoteID, windSensors[i].expected, reading.Value)
		}
	}
}

func TestPusher_ParseWeatherData_BatteryAndSignal(t *testing.T) {
	pusher := &Pusher{}

	batteryID := uuid.New()
	signalID := uuid.New()

	sensors := map[string]models.Sensor{
		"wh65batt": {
			ID:         batteryID,
			RemoteID:   "wh65batt",
			SensorType: models.SensorTypeBattery,
		},
		"rssi": {
			ID:         signalID,
			RemoteID:   "rssi",
			SensorType: models.SensorTypeSignalStrength,
		},
	}

	params := url.Values{
		"wh65batt": []string{"5"},
		"rssi":     []string{"-45"},
		"dateutc":  []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("Expected 2 readings, got %d", len(result))
	}

	batteryReading := result[batteryID]
	if batteryReading.Value != 5.0 {
		t.Errorf("Expected battery 5, got %.1f", batteryReading.Value)
	}

	signalReading := result[signalID]
	if signalReading.Value != -45.0 {
		t.Errorf("Expected signal strength -45 dBm, got %.1f dBm", signalReading.Value)
	}
}

func TestPusher_ParseWeatherData_VPD(t *testing.T) {
	pusher := &Pusher{}

	sensorID := uuid.New()
	sensors := map[string]models.Sensor{
		"vpd": {
			ID:         sensorID,
			RemoteID:   "vpd",
			SensorType: models.SensorTypeVPD,
		},
	}

	params := url.Values{
		"vpd":     []string{"1.25"},
		"dateutc": []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 reading, got %d", len(result))
	}

	reading := result[sensorID]
	if reading.Value != 1.25 {
		t.Errorf("Expected VPD 1.25 kPa, got %.2f kPa", reading.Value)
	}
}

func TestPusher_ParseWeatherData_OutdoorSensors(t *testing.T) {
	pusher := &Pusher{}

	tempID := uuid.New()
	humidityID := uuid.New()

	sensors := map[string]models.Sensor{
		"tempf": {
			ID:         tempID,
			RemoteID:   "tempf",
			SensorType: models.SensorTypeTemperatureOutdoor,
		},
		"humidity": {
			ID:         humidityID,
			RemoteID:   "humidity",
			SensorType: models.SensorTypeHumidityOutdoor,
		},
	}

	params := url.Values{
		"tempf":    []string{"50.0"}, // 50°F = 10°C
		"humidity": []string{"80"},
		"dateutc":  []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("Expected 2 readings, got %d", len(result))
	}

	tempReading := result[tempID]
	expectedTemp := 10.0
	tolerance := 0.1
	if tempReading.Value < expectedTemp-tolerance || tempReading.Value > expectedTemp+tolerance {
		t.Errorf("Expected outdoor temperature ~%.1f°C, got %.1f°C", expectedTemp, tempReading.Value)
	}

	humidityReading := result[humidityID]
	if humidityReading.Value != 80.0 {
		t.Errorf("Expected outdoor humidity 80%%, got %.1f%%", humidityReading.Value)
	}
}

func TestPusher_ParseWeatherData_UnknownSensorType(t *testing.T) {
	pusher := &Pusher{}

	sensorID := uuid.New()
	sensors := map[string]models.Sensor{
		"unknown_sensor": {
			ID:         sensorID,
			RemoteID:   "unknown_sensor",
			SensorType: "unknown_type",
		},
	}

	params := url.Values{
		"unknown_sensor": []string{"123.45"},
		"dateutc":        []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 reading, got %d", len(result))
	}

	reading := result[sensorID]
	if reading.Value != 123.45 {
		t.Errorf("Expected value 123.45, got %.2f", reading.Value)
	}
}

func TestPusher_ParseWeatherData_PressureTypes(t *testing.T) {
	pusher := &Pusher{}

	relativeID := uuid.New()
	absoluteID := uuid.New()

	sensors := map[string]models.Sensor{
		"baromrelin": {
			ID:         relativeID,
			RemoteID:   "baromrelin",
			SensorType: models.SensorTypePressureRelative,
		},
		"baromabsin": {
			ID:         absoluteID,
			RemoteID:   "baromabsin",
			SensorType: models.SensorTypePressureAbsolute,
		},
	}

	params := url.Values{
		"baromrelin": []string{"29.92"}, // 1013.25 hPa
		"baromabsin": []string{"30.00"}, // 1015.95 hPa
		"dateutc":    []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("Expected 2 readings, got %d", len(result))
	}

	relativeReading := result[relativeID]
	expectedRelative := 1013.25
	tolerance := 1.0
	if relativeReading.Value < expectedRelative-tolerance || relativeReading.Value > expectedRelative+tolerance {
		t.Errorf("Expected relative pressure ~%.2f hPa, got %.2f hPa", expectedRelative, relativeReading.Value)
	}

	absoluteReading := result[absoluteID]
	expectedAbsolute := 1015.95
	if absoluteReading.Value < expectedAbsolute-tolerance || absoluteReading.Value > expectedAbsolute+tolerance {
		t.Errorf("Expected absolute pressure ~%.2f hPa, got %.2f hPa", expectedAbsolute, absoluteReading.Value)
	}
}

func TestPusher_ParseWeatherData_ZeroValues(t *testing.T) {
	pusher := &Pusher{}

	tempID := uuid.New()
	humidityID := uuid.New()
	windID := uuid.New()

	sensors := map[string]models.Sensor{
		"tempf": {
			ID:         tempID,
			RemoteID:   "tempf",
			SensorType: models.SensorTypeTemperature,
		},
		"humidity": {
			ID:         humidityID,
			RemoteID:   "humidity",
			SensorType: models.SensorTypeHumidity,
		},
		"windspeedmph": {
			ID:         windID,
			RemoteID:   "windspeedmph",
			SensorType: models.SensorTypeWindSpeed,
		},
	}

	params := url.Values{
		"tempf":        []string{"32.0"}, // 0°C
		"humidity":     []string{"0"},
		"windspeedmph": []string{"0.0"},
		"dateutc":      []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("Expected 3 readings, got %d", len(result))
	}

	// Verify zero values are handled correctly
	tempReading := result[tempID]
	expectedTemp := 0.0
	tolerance := 0.1
	if tempReading.Value < expectedTemp-tolerance || tempReading.Value > expectedTemp+tolerance {
		t.Errorf("Expected temperature ~%.1f°C, got %.1f°C", expectedTemp, tempReading.Value)
	}

	humidityReading := result[humidityID]
	if humidityReading.Value != 0.0 {
		t.Errorf("Expected humidity 0%%, got %.1f%%", humidityReading.Value)
	}

	windReading := result[windID]
	if windReading.Value != 0.0 {
		t.Errorf("Expected wind speed 0 m/s, got %.1f m/s", windReading.Value)
	}
}

func TestPusher_ParseWeatherData_NegativeValues(t *testing.T) {
	pusher := &Pusher{}

	tempID := uuid.New()
	signalID := uuid.New()

	sensors := map[string]models.Sensor{
		"tempf": {
			ID:         tempID,
			RemoteID:   "tempf",
			SensorType: models.SensorTypeTemperature,
		},
		"rssi": {
			ID:         signalID,
			RemoteID:   "rssi",
			SensorType: models.SensorTypeSignalStrength,
		},
	}

	params := url.Values{
		"tempf":   []string{"-4.0"}, // -20°C
		"rssi":    []string{"-75"},
		"dateutc": []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("Expected 2 readings, got %d", len(result))
	}

	tempReading := result[tempID]
	expectedTemp := -20.0
	tolerance := 0.1
	if tempReading.Value < expectedTemp-tolerance || tempReading.Value > expectedTemp+tolerance {
		t.Errorf("Expected temperature ~%.1f°C, got %.1f°C", expectedTemp, tempReading.Value)
	}

	signalReading := result[signalID]
	if signalReading.Value != -75.0 {
		t.Errorf("Expected signal strength -75 dBm, got %.1f dBm", signalReading.Value)
	}
}

func TestPusher_ParseWeatherData_LargeValues(t *testing.T) {
	pusher := &Pusher{}

	solarID := uuid.New()
	pressureID := uuid.New()

	sensors := map[string]models.Sensor{
		"solarradiation": {
			ID:         solarID,
			RemoteID:   "solarradiation",
			SensorType: models.SensorTypeSolarRadiation,
		},
		"baromrelin": {
			ID:         pressureID,
			RemoteID:   "baromrelin",
			SensorType: models.SensorTypePressureRelative,
		},
	}

	params := url.Values{
		"solarradiation": []string{"1200.5"},
		"baromrelin":     []string{"31.50"}, // High pressure
		"dateutc":        []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("Expected 2 readings, got %d", len(result))
	}

	solarReading := result[solarID]
	if solarReading.Value != 1200.5 {
		t.Errorf("Expected solar radiation 1200.5 W/m², got %.1f W/m²", solarReading.Value)
	}

	pressureReading := result[pressureID]
	expectedPressure := 1066.53
	tolerance := 1.0
	if pressureReading.Value < expectedPressure-tolerance || pressureReading.Value > expectedPressure+tolerance {
		t.Errorf("Expected pressure ~%.2f hPa, got %.2f hPa", expectedPressure, pressureReading.Value)
	}
}

func TestPusher_ParseWeatherData_DecimalPrecision(t *testing.T) {
	pusher := &Pusher{}

	tempID := uuid.New()
	rainID := uuid.New()

	sensors := map[string]models.Sensor{
		"tempf": {
			ID:         tempID,
			RemoteID:   "tempf",
			SensorType: models.SensorTypeTemperature,
		},
		"dailyrainin": {
			ID:         rainID,
			RemoteID:   "dailyrainin",
			SensorType: models.SensorTypeRainfallDaily,
		},
	}

	params := url.Values{
		"tempf":       []string{"72.567"},
		"dailyrainin": []string{"0.123"},
		"dateutc":     []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("Expected 2 readings, got %d", len(result))
	}

	// Verify precision is maintained
	tempReading := result[tempID]
	expectedTemp := 22.537 // (72.567 - 32) * 5/9
	tolerance := 0.01
	if tempReading.Value < expectedTemp-tolerance || tempReading.Value > expectedTemp+tolerance {
		t.Errorf("Expected temperature ~%.3f°C, got %.3f°C", expectedTemp, tempReading.Value)
	}

	rainReading := result[rainID]
	expectedRain := 3.1242 // 0.123 * 25.4
	if rainReading.Value < expectedRain-tolerance || rainReading.Value > expectedRain+tolerance {
		t.Errorf("Expected rainfall ~%.4f mm, got %.4f mm", expectedRain, rainReading.Value)
	}
}

func TestPusher_ParseWeatherData_SameTimestamp(t *testing.T) {
	pusher := &Pusher{}

	tempID := uuid.New()
	humidityID := uuid.New()
	pressureID := uuid.New()

	sensors := map[string]models.Sensor{
		"tempf": {
			ID:         tempID,
			RemoteID:   "tempf",
			SensorType: models.SensorTypeTemperature,
		},
		"humidity": {
			ID:         humidityID,
			RemoteID:   "humidity",
			SensorType: models.SensorTypeHumidity,
		},
		"baromrelin": {
			ID:         pressureID,
			RemoteID:   "baromrelin",
			SensorType: models.SensorTypePressureRelative,
		},
	}

	params := url.Values{
		"tempf":      []string{"68.0"},
		"humidity":   []string{"50"},
		"baromrelin": []string{"30.00"},
		"dateutc":    []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("Expected 3 readings, got %d", len(result))
	}

	// Verify all readings have the same timestamp
	expectedTime, _ := time.Parse("2006-01-02 15:04:05", "2024-01-15 12:00:00")
	for _, reading := range result {
		if !reading.DateUTC.Equal(expectedTime) {
			t.Errorf("Expected timestamp %v, got %v", expectedTime, reading.DateUTC)
		}
	}
}

func TestPusher_ParseWeatherData_ExtraUnknownParams(t *testing.T) {
	pusher := &Pusher{}

	tempID := uuid.New()
	sensors := map[string]models.Sensor{
		"tempf": {
			ID:         tempID,
			RemoteID:   "tempf",
			SensorType: models.SensorTypeTemperature,
		},
	}

	params := url.Values{
		"tempf":          []string{"68.0"},
		"unknown_param1": []string{"value1"},
		"unknown_param2": []string{"value2"},
		"dateutc":        []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should only parse known sensors
	if len(result) != 1 {
		t.Fatalf("Expected 1 reading, got %d", len(result))
	}

	if _, ok := result[tempID]; !ok {
		t.Error("Expected temperature reading")
	}
}

func TestPusher_ParseWeatherData_MultipleValuesForSameSensor(t *testing.T) {
	pusher := &Pusher{}

	tempID := uuid.New()
	sensors := map[string]models.Sensor{
		"tempf": {
			ID:         tempID,
			RemoteID:   "tempf",
			SensorType: models.SensorTypeTemperature,
		},
	}

	// URL params can have multiple values for the same key
	params := url.Values{
		"tempf":   []string{"68.0", "70.0"}, // Multiple values
		"dateutc": []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 reading, got %d", len(result))
	}

	// Should use the first value
	reading := result[tempID]
	expectedTemp := 20.0 // (68 - 32) * 5/9
	tolerance := 0.1
	if reading.Value < expectedTemp-tolerance || reading.Value > expectedTemp+tolerance {
		t.Errorf("Expected temperature ~%.1f°C, got %.1f°C", expectedTemp, reading.Value)
	}
}

func TestPusher_ParseWeatherData_ScientificNotation(t *testing.T) {
	pusher := &Pusher{}

	solarID := uuid.New()
	sensors := map[string]models.Sensor{
		"solarradiation": {
			ID:         solarID,
			RemoteID:   "solarradiation",
			SensorType: models.SensorTypeSolarRadiation,
		},
	}

	params := url.Values{
		"solarradiation": []string{"1.2e3"}, // 1200
		"dateutc":        []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 reading, got %d", len(result))
	}

	reading := result[solarID]
	if reading.Value != 1200.0 {
		t.Errorf("Expected solar radiation 1200.0 W/m², got %.1f W/m²", reading.Value)
	}
}

func TestPusher_ParseWeatherData_AllSensorTypesPresent(t *testing.T) {
	pusher := &Pusher{}

	// Create sensors for all supported types
	sensorTypes := []struct {
		remoteID   string
		sensorType string
		value      string
	}{
		{"tempf", models.SensorTypeTemperature, "68.0"},
		{"temp1f", models.SensorTypeTemperatureOutdoor, "50.0"},
		{"humidity", models.SensorTypeHumidity, "50"},
		{"humidity1", models.SensorTypeHumidityOutdoor, "80"},
		{"baromrelin", models.SensorTypePressureRelative, "30.00"},
		{"baromabsin", models.SensorTypePressureAbsolute, "29.92"},
		{"windspeedmph", models.SensorTypeWindSpeed, "10.0"},
		{"windgustmph", models.SensorTypeWindGust, "15.0"},
		{"maxdailygust", models.SensorTypeWindGustMaxDaily, "20.0"},
		{"winddir", models.SensorTypeWindDirection, "180"},
		{"rainratein", models.SensorTypeRainfallRate, "0.1"},
		{"eventrainin", models.SensorTypeRainfallEvent, "0.2"},
		{"hourlyrainin", models.SensorTypeRainfallHourly, "0.3"},
		{"dailyrainin", models.SensorTypeRainfallDaily, "0.5"},
		{"weeklyrainin", models.SensorTypeRainfallWeekly, "1.0"},
		{"monthlyrainin", models.SensorTypeRainfallMonthly, "2.0"},
		{"yearlyrainin", models.SensorTypeRainfallYearly, "10.0"},
		{"totalrainin", models.SensorTypeRainfallTotal, "50.0"},
		{"solarradiation", models.SensorTypeSolarRadiation, "450.5"},
		{"uv", models.SensorTypeUVIndex, "7"},
		{"vpd", models.SensorTypeVPD, "1.25"},
		{"wh65batt", models.SensorTypeBattery, "5"},
		{"rssi", models.SensorTypeSignalStrength, "-45"},
	}

	sensors := make(map[string]models.Sensor)
	params := url.Values{
		"dateutc": []string{"2024-01-15 12:00:00"},
	}

	for _, st := range sensorTypes {
		sensorID := uuid.New()
		sensors[st.remoteID] = models.Sensor{
			ID:         sensorID,
			RemoteID:   st.remoteID,
			SensorType: st.sensorType,
		}
		params.Set(st.remoteID, st.value)
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != len(sensorTypes) {
		t.Fatalf("Expected %d readings, got %d", len(sensorTypes), len(result))
	}

	// Verify all sensor types were parsed
	for _, st := range sensorTypes {
		sensor := sensors[st.remoteID]
		if _, ok := result[sensor.ID]; !ok {
			t.Errorf("Missing reading for sensor type %s (remoteID: %s)", st.sensorType, st.remoteID)
		}
	}
}

func TestPusher_ParseWeatherData_PartialSensorData(t *testing.T) {
	pusher := &Pusher{}

	tempID := uuid.New()
	humidityID := uuid.New()
	pressureID := uuid.New()

	sensors := map[string]models.Sensor{
		"tempf": {
			ID:         tempID,
			RemoteID:   "tempf",
			SensorType: models.SensorTypeTemperature,
		},
		"humidity": {
			ID:         humidityID,
			RemoteID:   "humidity",
			SensorType: models.SensorTypeHumidity,
		},
		"baromrelin": {
			ID:         pressureID,
			RemoteID:   "baromrelin",
			SensorType: models.SensorTypePressureRelative,
		},
	}

	// Only provide data for temperature and humidity, not pressure
	params := url.Values{
		"tempf":    []string{"68.0"},
		"humidity": []string{"50"},
		"dateutc":  []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should only return readings for sensors with data
	if len(result) != 2 {
		t.Fatalf("Expected 2 readings, got %d", len(result))
	}

	if _, ok := result[tempID]; !ok {
		t.Error("Expected temperature reading")
	}

	if _, ok := result[humidityID]; !ok {
		t.Error("Expected humidity reading")
	}

	if _, ok := result[pressureID]; ok {
		t.Error("Did not expect pressure reading without data")
	}
}

func TestPusher_ParseWeatherData_MixedValidInvalidValues(t *testing.T) {
	pusher := &Pusher{}

	tempID := uuid.New()
	humidityID := uuid.New()
	pressureID := uuid.New()

	sensors := map[string]models.Sensor{
		"tempf": {
			ID:         tempID,
			RemoteID:   "tempf",
			SensorType: models.SensorTypeTemperature,
		},
		"humidity": {
			ID:         humidityID,
			RemoteID:   "humidity",
			SensorType: models.SensorTypeHumidity,
		},
		"baromrelin": {
			ID:         pressureID,
			RemoteID:   "baromrelin",
			SensorType: models.SensorTypePressureRelative,
		},
	}

	params := url.Values{
		"tempf":      []string{"68.0"},    // Valid
		"humidity":   []string{"invalid"}, // Invalid
		"baromrelin": []string{"30.00"},   // Valid
		"dateutc":    []string{"2024-01-15 12:00:00"},
	}

	result, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should only return valid readings
	if len(result) != 2 {
		t.Fatalf("Expected 2 readings, got %d", len(result))
	}

	if _, ok := result[tempID]; !ok {
		t.Error("Expected temperature reading")
	}

	if _, ok := result[humidityID]; ok {
		t.Error("Did not expect humidity reading with invalid value")
	}

	if _, ok := result[pressureID]; !ok {
		t.Error("Expected pressure reading")
	}
}

func TestPusher_ParseWeatherData_BoundaryTemperatures(t *testing.T) {
	pusher := &Pusher{}

	testCases := []struct {
		name       string
		fahrenheit string
		expectedC  float64
	}{
		{
			name:       "Freezing point",
			fahrenheit: "32.0",
			expectedC:  0.0,
		},
		{
			name:       "Boiling point",
			fahrenheit: "212.0",
			expectedC:  100.0,
		},
		{
			name:       "Absolute zero",
			fahrenheit: "-459.67",
			expectedC:  -273.15,
		},
		{
			name:       "Room temperature",
			fahrenheit: "68.0",
			expectedC:  20.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sensorID := uuid.New()
			sensors := map[string]models.Sensor{
				"tempf": {
					ID:         sensorID,
					RemoteID:   "tempf",
					SensorType: models.SensorTypeTemperature,
				},
			}

			params := url.Values{
				"tempf":   []string{tc.fahrenheit},
				"dateutc": []string{"2024-01-15 12:00:00"},
			}

			result, err := pusher.ParseWeatherData(params, sensors)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if len(result) != 1 {
				t.Fatalf("Expected 1 reading, got %d", len(result))
			}

			reading := result[sensorID]
			tolerance := 0.1
			if reading.Value < tc.expectedC-tolerance || reading.Value > tc.expectedC+tolerance {
				t.Errorf("Expected temperature ~%.2f°C, got %.2f°C", tc.expectedC, reading.Value)
			}
		})
	}
}

func TestPusher_ParseWeatherData_ExtremePressureValues(t *testing.T) {
	pusher := &Pusher{}

	testCases := []struct {
		name        string
		inHg        string
		expectedHPa float64
	}{
		{
			name:        "Very low pressure (hurricane)",
			inHg:        "26.00",
			expectedHPa: 880.46,
		},
		{
			name:        "Standard pressure",
			inHg:        "29.92",
			expectedHPa: 1013.25,
		},
		{
			name:        "Very high pressure",
			inHg:        "32.00",
			expectedHPa: 1083.64,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sensorID := uuid.New()
			sensors := map[string]models.Sensor{
				"baromrelin": {
					ID:         sensorID,
					RemoteID:   "baromrelin",
					SensorType: models.SensorTypePressureRelative,
				},
			}

			params := url.Values{
				"baromrelin": []string{tc.inHg},
				"dateutc":    []string{"2024-01-15 12:00:00"},
			}

			result, err := pusher.ParseWeatherData(params, sensors)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if len(result) != 1 {
				t.Fatalf("Expected 1 reading, got %d", len(result))
			}

			reading := result[sensorID]
			tolerance := 1.0
			if reading.Value < tc.expectedHPa-tolerance || reading.Value > tc.expectedHPa+tolerance {
				t.Errorf("Expected pressure ~%.2f hPa, got %.2f hPa", tc.expectedHPa, reading.Value)
			}
		})
	}
}

func TestPusher_ParseWeatherData_WindDirectionBoundaries(t *testing.T) {
	pusher := &Pusher{}

	testCases := []struct {
		name      string
		direction string
		expected  float64
	}{
		{
			name:      "North",
			direction: "0",
			expected:  0.0,
		},
		{
			name:      "East",
			direction: "90",
			expected:  90.0,
		},
		{
			name:      "South",
			direction: "180",
			expected:  180.0,
		},
		{
			name:      "West",
			direction: "270",
			expected:  270.0,
		},
		{
			name:      "Almost full circle",
			direction: "359",
			expected:  359.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sensorID := uuid.New()
			sensors := map[string]models.Sensor{
				"winddir": {
					ID:         sensorID,
					RemoteID:   "winddir",
					SensorType: models.SensorTypeWindDirection,
				},
			}

			params := url.Values{
				"winddir": []string{tc.direction},
				"dateutc": []string{"2024-01-15 12:00:00"},
			}

			result, err := pusher.ParseWeatherData(params, sensors)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if len(result) != 1 {
				t.Fatalf("Expected 1 reading, got %d", len(result))
			}

			reading := result[sensorID]
			if reading.Value != tc.expected {
				t.Errorf("Expected wind direction %.0f°, got %.0f°", tc.expected, reading.Value)
			}
		})
	}
}

func TestPusher_ParseWeatherData_UVIndexRange(t *testing.T) {
	pusher := &Pusher{}

	testCases := []struct {
		name     string
		uvValue  string
		expected float64
	}{
		{
			name:     "No UV",
			uvValue:  "0",
			expected: 0.0,
		},
		{
			name:     "Low UV",
			uvValue:  "2",
			expected: 2.0,
		},
		{
			name:     "Moderate UV",
			uvValue:  "5",
			expected: 5.0,
		},
		{
			name:     "High UV",
			uvValue:  "8",
			expected: 8.0,
		},
		{
			name:     "Very high UV",
			uvValue:  "11",
			expected: 11.0,
		},
		{
			name:     "Extreme UV",
			uvValue:  "15",
			expected: 15.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sensorID := uuid.New()
			sensors := map[string]models.Sensor{
				"uv": {
					ID:         sensorID,
					RemoteID:   "uv",
					SensorType: models.SensorTypeUVIndex,
				},
			}

			params := url.Values{
				"uv":      []string{tc.uvValue},
				"dateutc": []string{"2024-01-15 12:00:00"},
			}

			result, err := pusher.ParseWeatherData(params, sensors)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if len(result) != 1 {
				t.Fatalf("Expected 1 reading, got %d", len(result))
			}

			reading := result[sensorID]
			if reading.Value != tc.expected {
				t.Errorf("Expected UV index %.0f, got %.0f", tc.expected, reading.Value)
			}
		})
	}
}

func TestPusher_ParseWeatherData_RainfallAccumulation(t *testing.T) {
	pusher := &Pusher{}

	testCases := []struct {
		name       string
		remoteID   string
		sensorType string
		inches     string
		expectedMM float64
	}{
		{
			name:       "Hourly rainfall",
			remoteID:   "hourlyrainin",
			sensorType: models.SensorTypeRainfallHourly,
			inches:     "0.1",
			expectedMM: 2.54,
		},
		{
			name:       "Daily rainfall",
			remoteID:   "dailyrainin",
			sensorType: models.SensorTypeRainfallDaily,
			inches:     "0.5",
			expectedMM: 12.7,
		},
		{
			name:       "Weekly rainfall",
			remoteID:   "weeklyrainin",
			sensorType: models.SensorTypeRainfallWeekly,
			inches:     "1.0",
			expectedMM: 25.4,
		},
		{
			name:       "Monthly rainfall",
			remoteID:   "monthlyrainin",
			sensorType: models.SensorTypeRainfallMonthly,
			inches:     "2.5",
			expectedMM: 63.5,
		},
		{
			name:       "Yearly rainfall",
			remoteID:   "yearlyrainin",
			sensorType: models.SensorTypeRainfallYearly,
			inches:     "10.0",
			expectedMM: 254.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sensorID := uuid.New()
			sensors := map[string]models.Sensor{
				tc.remoteID: {
					ID:         sensorID,
					RemoteID:   tc.remoteID,
					SensorType: tc.sensorType,
				},
			}

			params := url.Values{
				tc.remoteID: []string{tc.inches},
				"dateutc":   []string{"2024-01-15 12:00:00"},
			}

			result, err := pusher.ParseWeatherData(params, sensors)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if len(result) != 1 {
				t.Fatalf("Expected 1 reading, got %d", len(result))
			}

			reading := result[sensorID]
			tolerance := 0.1
			if reading.Value < tc.expectedMM-tolerance || reading.Value > tc.expectedMM+tolerance {
				t.Errorf("Expected rainfall ~%.1f mm, got %.1f mm", tc.expectedMM, reading.Value)
			}
		})
	}
}

func TestPusher_ParseWeatherData_WindSpeedConversions(t *testing.T) {
	pusher := &Pusher{}

	testCases := []struct {
		name       string
		remoteID   string
		sensorType string
		mph        string
		expectedMS float64
	}{
		{
			name:       "Current wind speed",
			remoteID:   "windspeedmph",
			sensorType: models.SensorTypeWindSpeed,
			mph:        "10.0",
			expectedMS: 4.4704,
		},
		{
			name:       "Wind gust",
			remoteID:   "windgustmph",
			sensorType: models.SensorTypeWindGust,
			mph:        "15.0",
			expectedMS: 6.7056,
		},
		{
			name:       "Max daily gust",
			remoteID:   "maxdailygust",
			sensorType: models.SensorTypeWindGustMaxDaily,
			mph:        "25.0",
			expectedMS: 11.176,
		},
		{
			name:       "Light breeze",
			remoteID:   "windspeedmph",
			sensorType: models.SensorTypeWindSpeed,
			mph:        "5.0",
			expectedMS: 2.2352,
		},
		{
			name:       "Strong wind",
			remoteID:   "windgustmph",
			sensorType: models.SensorTypeWindGust,
			mph:        "50.0",
			expectedMS: 22.352,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sensorID := uuid.New()
			sensors := map[string]models.Sensor{
				tc.remoteID: {
					ID:         sensorID,
					RemoteID:   tc.remoteID,
					SensorType: tc.sensorType,
				},
			}

			params := url.Values{
				tc.remoteID: []string{tc.mph},
				"dateutc":   []string{"2024-01-15 12:00:00"},
			}

			result, err := pusher.ParseWeatherData(params, sensors)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if len(result) != 1 {
				t.Fatalf("Expected 1 reading, got %d", len(result))
			}

			reading := result[sensorID]
			tolerance := 0.01
			if reading.Value < tc.expectedMS-tolerance || reading.Value > tc.expectedMS+tolerance {
				t.Errorf("Expected wind speed ~%.4f m/s, got %.4f m/s", tc.expectedMS, reading.Value)
			}
		})
	}
}

func TestPusher_ParseWeatherData_BatteryLevels(t *testing.T) {
	pusher := &Pusher{}

	testCases := []struct {
		name         string
		batteryValue string
		expected     float64
	}{
		{
			name:         "Empty battery",
			batteryValue: "0",
			expected:     0.0,
		},
		{
			name:         "Low battery",
			batteryValue: "1",
			expected:     1.0,
		},
		{
			name:         "Medium battery",
			batteryValue: "3",
			expected:     3.0,
		},
		{
			name:         "Full battery",
			batteryValue: "5",
			expected:     5.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sensorID := uuid.New()
			sensors := map[string]models.Sensor{
				"wh65batt": {
					ID:         sensorID,
					RemoteID:   "wh65batt",
					SensorType: models.SensorTypeBattery,
				},
			}

			params := url.Values{
				"wh65batt": []string{tc.batteryValue},
				"dateutc":  []string{"2024-01-15 12:00:00"},
			}

			result, err := pusher.ParseWeatherData(params, sensors)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if len(result) != 1 {
				t.Fatalf("Expected 1 reading, got %d", len(result))
			}

			reading := result[sensorID]
			if reading.Value != tc.expected {
				t.Errorf("Expected battery level %.0f, got %.0f", tc.expected, reading.Value)
			}
		})
	}
}

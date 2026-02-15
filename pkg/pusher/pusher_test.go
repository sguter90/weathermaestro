package pusher

import (
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/sguter90/weathermaestro/pkg/models"
)

// MockPusher implements the Pusher interface for testing
type MockPusher struct {
	endpoint    string
	stationType string
}

func (m *MockPusher) GetEndpoint() string {
	return m.endpoint
}

func (m *MockPusher) GetStationType() string {
	return m.stationType
}

func (m *MockPusher) ParseStation(params url.Values) *models.StationData {
	return &models.StationData{
		StationType: m.stationType,
		PassKey:     params.Get("PASSKEY"),
	}
}

func (m *MockPusher) ParseSensors(params url.Values) map[string]models.Sensor {
	sensors := make(map[string]models.Sensor)
	if params.Get("tempf") != "" {
		sensors["tempf"] = models.Sensor{
			RemoteID:   "tempf",
			SensorType: models.SensorTypeTemperature,
		}
	}
	return sensors
}

func (m *MockPusher) ParseWeatherData(params url.Values, sensors map[string]models.Sensor) (map[uuid.UUID]models.SensorReading, error) {
	return make(map[uuid.UUID]models.SensorReading), nil
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	if registry == nil {
		t.Fatal("Expected registry to be created, got nil")
	}

	if registry.pushers == nil {
		t.Fatal("Expected pushers map to be initialized, got nil")
	}

	if len(registry.pushers) != 0 {
		t.Errorf("Expected empty registry, got %d pushers", len(registry.pushers))
	}
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	pusher1 := &MockPusher{
		endpoint:    "/api/v1/ecowitt",
		stationType: "ecowitt",
	}

	registry.Register(pusher1)

	if len(registry.pushers) != 1 {
		t.Errorf("Expected 1 pusher, got %d", len(registry.pushers))
	}

	if _, ok := registry.pushers["ecowitt"]; !ok {
		t.Error("Expected pusher to be registered with key 'ecowitt'")
	}
}

func TestRegistry_Register_MultiplePushers(t *testing.T) {
	registry := NewRegistry()

	pusher1 := &MockPusher{
		endpoint:    "/api/v1/ecowitt",
		stationType: "ecowitt",
	}

	pusher2 := &MockPusher{
		endpoint:    "/api/v1/weathercloud",
		stationType: "weathercloud",
	}

	pusher3 := &MockPusher{
		endpoint:    "/api/v1/wunderground",
		stationType: "wunderground",
	}

	registry.Register(pusher1)
	registry.Register(pusher2)
	registry.Register(pusher3)

	if len(registry.pushers) != 3 {
		t.Errorf("Expected 3 pushers, got %d", len(registry.pushers))
	}

	expectedTypes := []string{"ecowitt", "weathercloud", "wunderground"}
	for _, expectedType := range expectedTypes {
		if _, ok := registry.pushers[expectedType]; !ok {
			t.Errorf("Expected pusher type '%s' to be registered", expectedType)
		}
	}
}

func TestRegistry_Register_OverwriteExisting(t *testing.T) {
	registry := NewRegistry()

	pusher1 := &MockPusher{
		endpoint:    "/api/v1/ecowitt",
		stationType: "ecowitt",
	}

	pusher2 := &MockPusher{
		endpoint:    "/api/v2/ecowitt",
		stationType: "ecowitt",
	}

	registry.Register(pusher1)
	registry.Register(pusher2)

	if len(registry.pushers) != 1 {
		t.Errorf("Expected 1 pusher, got %d", len(registry.pushers))
	}

	pusher, ok := registry.pushers["ecowitt"]
	if !ok {
		t.Fatal("Expected pusher to be registered")
	}

	if pusher.GetEndpoint() != "/api/v2/ecowitt" {
		t.Errorf("Expected endpoint '/api/v2/ecowitt', got '%s'", pusher.GetEndpoint())
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()

	pusher1 := &MockPusher{
		endpoint:    "/api/v1/ecowitt",
		stationType: "ecowitt",
	}

	registry.Register(pusher1)

	pusher, ok := registry.Get("ecowitt")
	if !ok {
		t.Fatal("Expected to find pusher")
	}

	if pusher.GetStationType() != "ecowitt" {
		t.Errorf("Expected station type 'ecowitt', got '%s'", pusher.GetStationType())
	}

	if pusher.GetEndpoint() != "/api/v1/ecowitt" {
		t.Errorf("Expected endpoint '/api/v1/ecowitt', got '%s'", pusher.GetEndpoint())
	}
}

func TestRegistry_Get_NotFound(t *testing.T) {
	registry := NewRegistry()

	pusher1 := &MockPusher{
		endpoint:    "/api/v1/ecowitt",
		stationType: "ecowitt",
	}

	registry.Register(pusher1)

	_, ok := registry.Get("nonexistent")
	if ok {
		t.Error("Expected not to find pusher, but found one")
	}
}

func TestRegistry_Get_EmptyRegistry(t *testing.T) {
	registry := NewRegistry()

	_, ok := registry.Get("ecowitt")
	if ok {
		t.Error("Expected not to find pusher in empty registry")
	}
}

func TestRegistry_All(t *testing.T) {
	registry := NewRegistry()

	pusher1 := &MockPusher{
		endpoint:    "/api/v1/ecowitt",
		stationType: "ecowitt",
	}

	pusher2 := &MockPusher{
		endpoint:    "/api/v1/weathercloud",
		stationType: "weathercloud",
	}

	registry.Register(pusher1)
	registry.Register(pusher2)

	pushers := registry.All()

	if len(pushers) != 2 {
		t.Errorf("Expected 2 pushers, got %d", len(pushers))
	}

	// Verify all pushers are present
	foundTypes := make(map[string]bool)
	for _, p := range pushers {
		foundTypes[p.GetStationType()] = true
	}

	if !foundTypes["ecowitt"] {
		t.Error("Expected to find 'ecowitt' pusher")
	}

	if !foundTypes["weathercloud"] {
		t.Error("Expected to find 'weathercloud' pusher")
	}
}

func TestRegistry_All_EmptyRegistry(t *testing.T) {
	registry := NewRegistry()

	pushers := registry.All()

	if pushers == nil {
		t.Fatal("Expected non-nil slice, got nil")
	}

	if len(pushers) != 0 {
		t.Errorf("Expected empty slice, got %d pushers", len(pushers))
	}
}

func TestRegistry_All_SinglePusher(t *testing.T) {
	registry := NewRegistry()

	pusher1 := &MockPusher{
		endpoint:    "/api/v1/ecowitt",
		stationType: "ecowitt",
	}

	registry.Register(pusher1)

	pushers := registry.All()

	if len(pushers) != 1 {
		t.Errorf("Expected 1 pusher, got %d", len(pushers))
	}

	if pushers[0].GetStationType() != "ecowitt" {
		t.Errorf("Expected station type 'ecowitt', got '%s'", pushers[0].GetStationType())
	}
}

func TestRegistry_All_OrderIndependent(t *testing.T) {
	registry := NewRegistry()

	pusher1 := &MockPusher{
		endpoint:    "/api/v1/ecowitt",
		stationType: "ecowitt",
	}

	pusher2 := &MockPusher{
		endpoint:    "/api/v1/weathercloud",
		stationType: "weathercloud",
	}

	pusher3 := &MockPusher{
		endpoint:    "/api/v1/wunderground",
		stationType: "wunderground",
	}

	registry.Register(pusher1)
	registry.Register(pusher2)
	registry.Register(pusher3)

	pushers := registry.All()

	if len(pushers) != 3 {
		t.Errorf("Expected 3 pushers, got %d", len(pushers))
	}

	// Verify all types are present (order doesn't matter)
	foundTypes := make(map[string]bool)
	for _, p := range pushers {
		foundTypes[p.GetStationType()] = true
	}

	expectedTypes := []string{"ecowitt", "weathercloud", "wunderground"}
	for _, expectedType := range expectedTypes {
		if !foundTypes[expectedType] {
			t.Errorf("Expected to find pusher type '%s'", expectedType)
		}
	}
}

func TestRegistry_RegisterAndRetrieve(t *testing.T) {
	registry := NewRegistry()

	testCases := []struct {
		name        string
		endpoint    string
		stationType string
	}{
		{
			name:        "Ecowitt pusher",
			endpoint:    "/api/v1/ecowitt",
			stationType: "ecowitt",
		},
		{
			name:        "WeatherCloud pusher",
			endpoint:    "/api/v1/weathercloud",
			stationType: "weathercloud",
		},
		{
			name:        "Weather Underground pusher",
			endpoint:    "/api/v1/wunderground",
			stationType: "wunderground",
		},
	}

	// Register all pushers
	for _, tc := range testCases {
		t.Run("Register_"+tc.name, func(t *testing.T) {
			pusher := &MockPusher{
				endpoint:    tc.endpoint,
				stationType: tc.stationType,
			}
			registry.Register(pusher)
		})
	}

	// Retrieve and verify all pushers
	for _, tc := range testCases {
		t.Run("Retrieve_"+tc.name, func(t *testing.T) {
			pusher, ok := registry.Get(tc.stationType)
			if !ok {
				t.Fatalf("Expected to find pusher for type '%s'", tc.stationType)
			}

			if pusher.GetStationType() != tc.stationType {
				t.Errorf("Expected station type '%s', got '%s'", tc.stationType, pusher.GetStationType())
			}

			if pusher.GetEndpoint() != tc.endpoint {
				t.Errorf("Expected endpoint '%s', got '%s'", tc.endpoint, pusher.GetEndpoint())
			}
		})
	}
}
func TestRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewRegistry()

	done := make(chan bool)

	// Concurrent registration
	for i := 0; i < 10; i++ {
		go func(id int) {
			pusher := &MockPusher{
				endpoint:    "/api/v1/pusher" + string(rune('0'+id)),
				stationType: "pusher" + string(rune('0'+id)),
			}
			registry.Register(pusher)
			done <- true
		}(i)
	}

	// Wait for all registrations
	for i := 0; i < 10; i++ {
		<-done
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func(id int) {
			stationType := "pusher" + string(rune('0'+id))
			_, ok := registry.Get(stationType)
			if !ok {
				t.Errorf("Expected to find pusher %s", stationType)
			}
			done <- true
		}(i)
	}

	// Wait for all reads
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all pushers are registered
	all := registry.All()
	if len(all) != 10 {
		t.Errorf("Expected 10 pushers after concurrent operations, got %d", len(all))
	}
}

func TestRegistry_ConcurrentRegisterSameType(t *testing.T) {
	registry := NewRegistry()

	done := make(chan bool)

	// Try to register same type concurrently
	for i := 0; i < 5; i++ {
		go func(id int) {
			pusher := &MockPusher{
				endpoint:    "/api/v" + string(rune('0'+id)) + "/ecowitt",
				stationType: "ecowitt",
			}
			registry.Register(pusher)
			done <- true
		}(i)
	}

	// Wait for all registrations
	for i := 0; i < 5; i++ {
		<-done
	}

	// Should only have one pusher (last one wins)
	if len(registry.pushers) != 1 {
		t.Errorf("Expected 1 pusher, got %d", len(registry.pushers))
	}

	pusher, ok := registry.Get("ecowitt")
	if !ok {
		t.Fatal("Expected to find pusher")
	}

	if pusher.GetStationType() != "ecowitt" {
		t.Errorf("Expected station type 'ecowitt', got '%s'", pusher.GetStationType())
	}
}

func TestMockPusher_GetEndpoint(t *testing.T) {
	pusher := &MockPusher{
		endpoint:    "/api/v1/test",
		stationType: "test",
	}

	endpoint := pusher.GetEndpoint()
	if endpoint != "/api/v1/test" {
		t.Errorf("Expected endpoint '/api/v1/test', got '%s'", endpoint)
	}
}

func TestMockPusher_GetStationType(t *testing.T) {
	pusher := &MockPusher{
		endpoint:    "/api/v1/test",
		stationType: "test",
	}

	stationType := pusher.GetStationType()
	if stationType != "test" {
		t.Errorf("Expected station type 'test', got '%s'", stationType)
	}
}

func TestMockPusher_ParseStation(t *testing.T) {
	pusher := &MockPusher{
		endpoint:    "/api/v1/test",
		stationType: "test",
	}

	params := url.Values{
		"PASSKEY": []string{"test-passkey-123"},
	}

	stationData := pusher.ParseStation(params)

	if stationData == nil {
		t.Fatal("Expected station data, got nil")
	}

	if stationData.StationType != "test" {
		t.Errorf("Expected station type 'test', got '%s'", stationData.StationType)
	}

	if stationData.PassKey != "test-passkey-123" {
		t.Errorf("Expected remote ID 'test-passkey-123', got '%s'", stationData.PassKey)
	}
}

func TestMockPusher_ParseStation_EmptyPasskey(t *testing.T) {
	pusher := &MockPusher{
		endpoint:    "/api/v1/test",
		stationType: "test",
	}

	params := url.Values{}

	stationData := pusher.ParseStation(params)

	if stationData == nil {
		t.Fatal("Expected station data, got nil")
	}

	if stationData.StationType != "test" {
		t.Errorf("Expected station type 'test', got '%s'", stationData.StationType)
	}

	if stationData.PassKey != "" {
		t.Errorf("Expected empty remote ID, got '%s'", stationData.PassKey)
	}
}

func TestMockPusher_ParseSensors(t *testing.T) {
	pusher := &MockPusher{
		endpoint:    "/api/v1/test",
		stationType: "test",
	}

	params := url.Values{
		"tempf": []string{"72.5"},
	}

	sensors := pusher.ParseSensors(params)

	if len(sensors) != 1 {
		t.Errorf("Expected 1 sensor, got %d", len(sensors))
	}

	sensor, ok := sensors["tempf"]
	if !ok {
		t.Fatal("Expected to find 'tempf' sensor")
	}

	if sensor.RemoteID != "tempf" {
		t.Errorf("Expected remote ID 'tempf', got '%s'", sensor.RemoteID)
	}

	if sensor.SensorType != models.SensorTypeTemperature {
		t.Errorf("Expected sensor type '%s', got '%s'", models.SensorTypeTemperature, sensor.SensorType)
	}
}

func TestMockPusher_ParseSensors_NoTemperature(t *testing.T) {
	pusher := &MockPusher{
		endpoint:    "/api/v1/test",
		stationType: "test",
	}

	params := url.Values{
		"humidity": []string{"65"},
	}

	sensors := pusher.ParseSensors(params)

	if len(sensors) != 0 {
		t.Errorf("Expected 0 sensors, got %d", len(sensors))
	}
}

func TestMockPusher_ParseWeatherData(t *testing.T) {
	pusher := &MockPusher{
		endpoint:    "/api/v1/test",
		stationType: "test",
	}

	sensorID := uuid.New()
	sensors := map[string]models.Sensor{
		"tempf": {
			ID:         sensorID,
			RemoteID:   "tempf",
			SensorType: models.SensorTypeTemperature,
		},
	}

	params := url.Values{
		"tempf": []string{"72.5"},
	}

	readings, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if readings == nil {
		t.Fatal("Expected readings map, got nil")
	}

	if len(readings) != 0 {
		t.Errorf("Expected empty readings map, got %d readings", len(readings))
	}
}

func TestRegistry_MultipleOperations(t *testing.T) {
	registry := NewRegistry()

	// Register multiple pushers
	pushers := []*MockPusher{
		{endpoint: "/api/v1/ecowitt", stationType: "ecowitt"},
		{endpoint: "/api/v1/weathercloud", stationType: "weathercloud"},
		{endpoint: "/api/v1/wunderground", stationType: "wunderground"},
	}

	for _, p := range pushers {
		registry.Register(p)
	}

	// Verify registration
	if len(registry.pushers) != 3 {
		t.Errorf("Expected 3 pushers, got %d", len(registry.pushers))
	}

	// Get each pusher
	for _, p := range pushers {
		retrieved, ok := registry.Get(p.stationType)
		if !ok {
			t.Errorf("Expected to find pusher '%s'", p.stationType)
			continue
		}

		if retrieved.GetStationType() != p.stationType {
			t.Errorf("Expected station type '%s', got '%s'", p.stationType, retrieved.GetStationType())
		}

		if retrieved.GetEndpoint() != p.endpoint {
			t.Errorf("Expected endpoint '%s', got '%s'", p.endpoint, retrieved.GetEndpoint())
		}
	}

	// Get all pushers
	all := registry.All()
	if len(all) != 3 {
		t.Errorf("Expected 3 pushers from All(), got %d", len(all))
	}

	// Verify all types are present
	foundTypes := make(map[string]bool)
	for _, p := range all {
		foundTypes[p.GetStationType()] = true
	}

	for _, p := range pushers {
		if !foundTypes[p.stationType] {
			t.Errorf("Expected to find pusher type '%s' in All() result", p.stationType)
		}
	}
}

func TestRegistry_ReplaceAndVerify(t *testing.T) {
	registry := NewRegistry()

	// Register initial pusher
	pusher1 := &MockPusher{
		endpoint:    "/api/v1/ecowitt",
		stationType: "ecowitt",
	}
	registry.Register(pusher1)

	// Verify initial registration
	retrieved1, ok := registry.Get("ecowitt")
	if !ok {
		t.Fatal("Expected to find initial pusher")
	}

	if retrieved1.GetEndpoint() != "/api/v1/ecowitt" {
		t.Errorf("Expected endpoint '/api/v1/ecowitt', got '%s'", retrieved1.GetEndpoint())
	}

	// Replace with new pusher
	pusher2 := &MockPusher{
		endpoint:    "/api/v2/ecowitt",
		stationType: "ecowitt",
	}
	registry.Register(pusher2)

	// Verify replacement
	retrieved2, ok := registry.Get("ecowitt")
	if !ok {
		t.Fatal("Expected to find replaced pusher")
	}

	if retrieved2.GetEndpoint() != "/api/v2/ecowitt" {
		t.Errorf("Expected endpoint '/api/v2/ecowitt', got '%s'", retrieved2.GetEndpoint())
	}

	// Verify only one pusher exists
	if len(registry.pushers) != 1 {
		t.Errorf("Expected 1 pusher after replacement, got %d", len(registry.pushers))
	}
}

func TestRegistry_EmptyStationType(t *testing.T) {
	registry := NewRegistry()

	pusher := &MockPusher{
		endpoint:    "/api/v1/test",
		stationType: "",
	}

	registry.Register(pusher)

	// Should be able to register with empty station type
	if len(registry.pushers) != 1 {
		t.Errorf("Expected 1 pusher, got %d", len(registry.pushers))
	}

	// Should be able to retrieve with empty key
	retrieved, ok := registry.Get("")
	if !ok {
		t.Error("Expected to find pusher with empty station type")
	}

	if retrieved.GetStationType() != "" {
		t.Errorf("Expected empty station type, got '%s'", retrieved.GetStationType())
	}

	if retrieved.GetEndpoint() != "/api/v1/test" {
		t.Errorf("Expected endpoint '/api/v1/test', got '%s'", retrieved.GetEndpoint())
	}
}

func TestRegistry_NilPusher(t *testing.T) {
	registry := NewRegistry()

	// This should not panic
	registry.Register(nil)

	// Registry should still be empty or contain nil
	if len(registry.pushers) > 1 {
		t.Errorf("Expected at most 1 entry, got %d", len(registry.pushers))
	}
}

func TestMockPusher_ParseStation_MultipleParams(t *testing.T) {
	pusher := &MockPusher{
		endpoint:    "/api/v1/test",
		stationType: "test",
	}

	params := url.Values{
		"PASSKEY": []string{"test-passkey-123"},
		"model":   []string{"test-model"},
		"freq":    []string{"868"},
	}

	stationData := pusher.ParseStation(params)

	if stationData == nil {
		t.Fatal("Expected station data, got nil")
	}

	if stationData.PassKey != "test-passkey-123" {
		t.Errorf("Expected pass key 'test-passkey-123', got '%s'", stationData.PassKey)
	}

	// MockPusher only extracts PASSKEY, other fields should be default
	if stationData.StationType != "test" {
		t.Errorf("Expected station type 'test', got '%s'", stationData.StationType)
	}
}

func TestMockPusher_ParseSensors_MultipleSensors(t *testing.T) {
	pusher := &MockPusher{
		endpoint:    "/api/v1/test",
		stationType: "test",
	}

	params := url.Values{
		"tempf":    []string{"72.5"},
		"humidity": []string{"65"},
		"pressure": []string{"29.92"},
	}

	sensors := pusher.ParseSensors(params)

	// MockPusher only recognizes tempf
	if len(sensors) != 1 {
		t.Errorf("Expected 1 sensor, got %d", len(sensors))
	}

	sensor, ok := sensors["tempf"]
	if !ok {
		t.Fatal("Expected to find 'tempf' sensor")
	}

	if sensor.RemoteID != "tempf" {
		t.Errorf("Expected remote ID 'tempf', got '%s'", sensor.RemoteID)
	}
}

func TestMockPusher_ParseWeatherData_WithError(t *testing.T) {
	pusher := &MockPusher{
		endpoint:    "/api/v1/test",
		stationType: "test",
	}

	sensorID := uuid.New()
	sensors := map[string]models.Sensor{
		"tempf": {
			ID:         sensorID,
			RemoteID:   "tempf",
			SensorType: models.SensorTypeTemperature,
		},
	}

	params := url.Values{}

	readings, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if readings == nil {
		t.Fatal("Expected readings map, got nil")
	}

	// MockPusher always returns empty map
	if len(readings) != 0 {
		t.Errorf("Expected empty readings map, got %d readings", len(readings))
	}
}

func TestRegistry_GetAfterClear(t *testing.T) {
	registry := NewRegistry()

	pusher := &MockPusher{
		endpoint:    "/api/v1/ecowitt",
		stationType: "ecowitt",
	}

	registry.Register(pusher)

	// Verify registration
	_, ok := registry.Get("ecowitt")
	if !ok {
		t.Fatal("Expected to find pusher")
	}

	// Clear registry by creating new map
	registry.pushers = make(map[string]Pusher)

	// Should not find pusher anymore
	_, ok = registry.Get("ecowitt")
	if ok {
		t.Error("Expected not to find pusher after clearing registry")
	}
}

func TestRegistry_AllAfterModification(t *testing.T) {
	registry := NewRegistry()

	pusher1 := &MockPusher{
		endpoint:    "/api/v1/ecowitt",
		stationType: "ecowitt",
	}

	registry.Register(pusher1)

	// Get all pushers
	all1 := registry.All()
	if len(all1) != 1 {
		t.Errorf("Expected 1 pusher, got %d", len(all1))
	}

	// Add another pusher
	pusher2 := &MockPusher{
		endpoint:    "/api/v1/weathercloud",
		stationType: "weathercloud",
	}

	registry.Register(pusher2)

	// Get all pushers again
	all2 := registry.All()
	if len(all2) != 2 {
		t.Errorf("Expected 2 pushers, got %d", len(all2))
	}

	// Verify first result wasn't modified
	if len(all1) != 1 {
		t.Errorf("Expected first result to remain 1 pusher, got %d", len(all1))
	}
}

func TestMockPusher_InterfaceCompliance(t *testing.T) {
	var _ Pusher = (*MockPusher)(nil)
}

func TestRegistry_RegisterMultipleTimes(t *testing.T) {
	registry := NewRegistry()

	pusher := &MockPusher{
		endpoint:    "/api/v1/ecowitt",
		stationType: "ecowitt",
	}

	// Register same pusher multiple times
	for i := 0; i < 5; i++ {
		registry.Register(pusher)
	}

	// Should only have one entry
	if len(registry.pushers) != 1 {
		t.Errorf("Expected 1 pusher, got %d", len(registry.pushers))
	}

	retrieved, ok := registry.Get("ecowitt")
	if !ok {
		t.Fatal("Expected to find pusher")
	}

	if retrieved.GetEndpoint() != "/api/v1/ecowitt" {
		t.Errorf("Expected endpoint '/api/v1/ecowitt', got '%s'", retrieved.GetEndpoint())
	}
}

func TestRegistry_GetWithSpecialCharacters(t *testing.T) {
	registry := NewRegistry()

	testCases := []struct {
		name        string
		stationType string
		endpoint    string
	}{
		{
			name:        "Hyphenated type",
			stationType: "weather-cloud",
			endpoint:    "/api/v1/weather-cloud",
		},
		{
			name:        "Underscored type",
			stationType: "weather_underground",
			endpoint:    "/api/v1/weather_underground",
		},
		{
			name:        "Dotted type",
			stationType: "weather.station",
			endpoint:    "/api/v1/weather.station",
		},
		{
			name:        "Mixed special chars",
			stationType: "weather-station_v2.0",
			endpoint:    "/api/v1/weather-station_v2.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pusher := &MockPusher{
				endpoint:    tc.endpoint,
				stationType: tc.stationType,
			}

			registry.Register(pusher)

			retrieved, ok := registry.Get(tc.stationType)
			if !ok {
				t.Fatalf("Expected to find pusher for type '%s'", tc.stationType)
			}

			if retrieved.GetStationType() != tc.stationType {
				t.Errorf("Expected station type '%s', got '%s'", tc.stationType, retrieved.GetStationType())
			}

			if retrieved.GetEndpoint() != tc.endpoint {
				t.Errorf("Expected endpoint '%s', got '%s'", tc.endpoint, retrieved.GetEndpoint())
			}
		})
	}
}

func TestMockPusher_ParseStation_CaseInsensitiveKey(t *testing.T) {
	pusher := &MockPusher{
		endpoint:    "/api/v1/test",
		stationType: "test",
	}

	testCases := []struct {
		name     string
		params   url.Values
		expected string
	}{
		{
			name: "Uppercase PASSKEY",
			params: url.Values{
				"PASSKEY": []string{"test-123"},
			},
			expected: "test-123",
		},
		{
			name: "Lowercase passkey",
			params: url.Values{
				"passkey": []string{"test-456"},
			},
			expected: "",
		},
		{
			name: "Mixed case PassKey",
			params: url.Values{
				"PassKey": []string{"test-789"},
			},
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stationData := pusher.ParseStation(tc.params)

			if stationData == nil {
				t.Fatal("Expected station data, got nil")
			}

			if stationData.PassKey != tc.expected {
				t.Errorf("Expected pass key '%s', got '%s'", tc.expected, stationData.PassKey)
			}
		})
	}
}

func TestMockPusher_ParseSensors_EmptyParams(t *testing.T) {
	pusher := &MockPusher{
		endpoint:    "/api/v1/test",
		stationType: "test",
	}

	params := url.Values{}

	sensors := pusher.ParseSensors(params)

	if sensors == nil {
		t.Fatal("Expected sensors map, got nil")
	}

	if len(sensors) != 0 {
		t.Errorf("Expected empty sensors map, got %d sensors", len(sensors))
	}
}

func TestMockPusher_ParseWeatherData_EmptySensors(t *testing.T) {
	pusher := &MockPusher{
		endpoint:    "/api/v1/test",
		stationType: "test",
	}

	sensors := make(map[string]models.Sensor)

	params := url.Values{
		"tempf": []string{"72.5"},
	}

	readings, err := pusher.ParseWeatherData(params, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if readings == nil {
		t.Fatal("Expected readings map, got nil")
	}

	if len(readings) != 0 {
		t.Errorf("Expected empty readings map, got %d readings", len(readings))
	}
}

func TestMockPusher_ParseWeatherData_NilSensors(t *testing.T) {
	pusher := &MockPusher{
		endpoint:    "/api/v1/test",
		stationType: "test",
	}

	params := url.Values{
		"tempf": []string{"72.5"},
	}

	readings, err := pusher.ParseWeatherData(params, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if readings == nil {
		t.Fatal("Expected readings map, got nil")
	}

	if len(readings) != 0 {
		t.Errorf("Expected empty readings map, got %d readings", len(readings))
	}
}

func TestRegistry_RegisterNilDoesNotPanic(t *testing.T) {
	registry := NewRegistry()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Register(nil) caused panic: %v", r)
		}
	}()

	registry.Register(nil)
}

func TestRegistry_GetFromNilMap(t *testing.T) {
	registry := &Registry{
		pushers: nil,
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Get() on nil map caused panic: %v", r)
		}
	}()

	_, ok := registry.Get("test")
	if ok {
		t.Error("Expected not to find pusher in nil map")
	}
}

func TestRegistry_AllFromNilMap(t *testing.T) {
	registry := &Registry{
		pushers: nil,
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("All() on nil map caused panic: %v", r)
		}
	}()

	all := registry.All()
	if all == nil {
		t.Error("Expected non-nil slice from All(), got nil")
	}

	if len(all) != 0 {
		t.Errorf("Expected empty slice from All(), got %d pushers", len(all))
	}
}

func TestMockPusher_ParseStation_NilParams(t *testing.T) {
	pusher := &MockPusher{
		endpoint:    "/api/v1/test",
		stationType: "test",
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ParseStation(nil) caused panic: %v", r)
		}
	}()

	stationData := pusher.ParseStation(nil)

	if stationData == nil {
		t.Fatal("Expected station data, got nil")
	}

	if stationData.StationType != "test" {
		t.Errorf("Expected station type 'test', got '%s'", stationData.StationType)
	}

	if stationData.PassKey != "" {
		t.Errorf("Expected empty pass key, got '%s'", stationData.PassKey)
	}
}

func TestMockPusher_ParseSensors_NilParams(t *testing.T) {
	pusher := &MockPusher{
		endpoint:    "/api/v1/test",
		stationType: "test",
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ParseSensors(nil) caused panic: %v", r)
		}
	}()

	sensors := pusher.ParseSensors(nil)

	if sensors == nil {
		t.Fatal("Expected sensors map, got nil")
	}

	if len(sensors) != 0 {
		t.Errorf("Expected empty sensors map, got %d sensors", len(sensors))
	}
}

func TestMockPusher_ParseWeatherData_NilParams(t *testing.T) {
	pusher := &MockPusher{
		endpoint:    "/api/v1/test",
		stationType: "test",
	}

	sensorID := uuid.New()
	sensors := map[string]models.Sensor{
		"tempf": {
			ID:         sensorID,
			RemoteID:   "tempf",
			SensorType: models.SensorTypeTemperature,
		},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ParseWeatherData(nil, sensors) caused panic: %v", r)
		}
	}()

	readings, err := pusher.ParseWeatherData(nil, sensors)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if readings == nil {
		t.Fatal("Expected readings map, got nil")
	}

	if len(readings) != 0 {
		t.Errorf("Expected empty readings map, got %d readings", len(readings))
	}
}

func TestRegistry_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	registry := NewRegistry()

	done := make(chan bool)
	operations := 1000

	// Concurrent registrations
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < operations; j++ {
				pusher := &MockPusher{
					endpoint:    "/api/v1/pusher" + string(rune('0'+(id%10))),
					stationType: "pusher" + string(rune('0'+(id%10))),
				}
				registry.Register(pusher)
			}
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < operations; j++ {
				stationType := "pusher" + string(rune('0'+(id%10)))
				registry.Get(stationType)
			}
			done <- true
		}(i)
	}

	// Concurrent All() calls
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < operations; j++ {
				registry.All()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 30; i++ {
		<-done
	}

	// Verify final state is consistent
	all := registry.All()
	if len(all) > 10 {
		t.Errorf("Expected at most 10 pushers, got %d", len(all))
	}

	// Verify each pusher can be retrieved
	for _, p := range all {
		retrieved, ok := registry.Get(p.GetStationType())
		if !ok {
			t.Errorf("Expected to find pusher '%s'", p.GetStationType())
		}

		if retrieved.GetStationType() != p.GetStationType() {
			t.Errorf("Expected station type '%s', got '%s'", p.GetStationType(), retrieved.GetStationType())
		}
	}
}

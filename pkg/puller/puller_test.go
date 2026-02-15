package puller

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sguter90/weathermaestro/pkg/models"
)

// MockPuller implements the Puller interface for testing
type MockPuller struct {
	providerType      string
	pullFunc          func(ctx context.Context, config map[string]interface{}) (map[string]models.SensorReading, *models.StationData, error)
	validateFunc      func(config map[string]interface{}) error
	pullCallCount     int
	validateCallCount int
}

func (m *MockPuller) GetProviderType() string {
	return m.providerType
}

func (m *MockPuller) Pull(ctx context.Context, config map[string]interface{}) (map[string]models.SensorReading, *models.StationData, error) {
	m.pullCallCount++
	if m.pullFunc != nil {
		return m.pullFunc(ctx, config)
	}
	return nil, nil, nil
}

func (m *MockPuller) ValidateConfig(config map[string]interface{}) error {
	m.validateCallCount++
	if m.validateFunc != nil {
		return m.validateFunc(config)
	}
	return nil
}

func TestNewPullerRegistry(t *testing.T) {
	registry := NewPullerRegistry()

	if registry == nil {
		t.Fatal("Expected registry to be created")
	}

	if registry.pullers == nil {
		t.Error("Expected pullers map to be initialized")
	}

	if len(registry.pullers) != 0 {
		t.Errorf("Expected empty registry, got %d pullers", len(registry.pullers))
	}
}

func TestPullerRegistry_Register(t *testing.T) {
	registry := NewPullerRegistry()

	puller1 := &MockPuller{providerType: "provider1"}
	puller2 := &MockPuller{providerType: "provider2"}

	registry.Register(puller1)
	registry.Register(puller2)

	if len(registry.pullers) != 2 {
		t.Errorf("Expected 2 pullers, got %d", len(registry.pullers))
	}

	if registry.pullers["provider1"] != puller1 {
		t.Error("Expected provider1 to be registered")
	}

	if registry.pullers["provider2"] != puller2 {
		t.Error("Expected provider2 to be registered")
	}
}

func TestPullerRegistry_Register_Overwrite(t *testing.T) {
	registry := NewPullerRegistry()

	puller1 := &MockPuller{providerType: "provider1"}
	puller2 := &MockPuller{providerType: "provider1"} // Same provider type

	registry.Register(puller1)
	registry.Register(puller2)

	if len(registry.pullers) != 1 {
		t.Errorf("Expected 1 puller, got %d", len(registry.pullers))
	}

	// Should have the second puller (overwritten)
	if registry.pullers["provider1"] != puller2 {
		t.Error("Expected second puller to overwrite first")
	}
}

func TestPullerRegistry_Get(t *testing.T) {
	registry := NewPullerRegistry()

	puller := &MockPuller{providerType: "testprovider"}
	registry.Register(puller)

	// Test getting existing puller
	retrieved, ok := registry.Get("testprovider")
	if !ok {
		t.Error("Expected to find registered puller")
	}
	if retrieved != puller {
		t.Error("Expected to get the same puller instance")
	}

	// Test getting non-existent puller
	_, ok = registry.Get("nonexistent")
	if ok {
		t.Error("Expected not to find non-existent puller")
	}
}

func TestPullerRegistry_All(t *testing.T) {
	registry := NewPullerRegistry()

	puller1 := &MockPuller{providerType: "provider1"}
	puller2 := &MockPuller{providerType: "provider2"}
	puller3 := &MockPuller{providerType: "provider3"}

	registry.Register(puller1)
	registry.Register(puller2)
	registry.Register(puller3)

	all := registry.All()

	if len(all) != 3 {
		t.Errorf("Expected 3 pullers, got %d", len(all))
	}

	// Verify all pullers are present (order doesn't matter)
	found := make(map[string]bool)
	for _, p := range all {
		found[p.GetProviderType()] = true
	}

	if !found["provider1"] || !found["provider2"] || !found["provider3"] {
		t.Error("Expected all registered pullers to be returned")
	}
}

func TestPullerRegistry_All_Empty(t *testing.T) {
	registry := NewPullerRegistry()

	all := registry.All()

	if len(all) != 0 {
		t.Errorf("Expected empty slice, got %d pullers", len(all))
	}
}

func TestMockPuller_GetProviderType(t *testing.T) {
	puller := &MockPuller{providerType: "testprovider"}

	if puller.GetProviderType() != "testprovider" {
		t.Errorf("Expected provider type 'testprovider', got '%s'", puller.GetProviderType())
	}
}

func TestMockPuller_Pull_Success(t *testing.T) {
	expectedReadings := map[string]models.SensorReading{
		"sensor1": {
			ID:       uuid.New(),
			SensorID: uuid.New(),
			Value:    25.5,
			DateUTC:  time.Now().UTC(),
		},
	}

	expectedStation := &models.StationData{
		ID:    uuid.New(),
		Model: "Test Station",
	}

	puller := &MockPuller{
		providerType: "testprovider",
		pullFunc: func(ctx context.Context, config map[string]interface{}) (map[string]models.SensorReading, *models.StationData, error) {
			return expectedReadings, expectedStation, nil
		},
	}

	ctx := context.Background()
	config := map[string]interface{}{"api_key": "test"}

	readings, station, err := puller.Pull(ctx, config)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if readings == nil {
		t.Error("Expected readings to be returned")
	}

	if station == nil {
		t.Error("Expected station data to be returned")
	}

	if puller.pullCallCount != 1 {
		t.Errorf("Expected Pull to be called once, got %d", puller.pullCallCount)
	}
}

func TestMockPuller_Pull_Error(t *testing.T) {
	expectedError := errors.New("pull failed")

	puller := &MockPuller{
		providerType: "testprovider",
		pullFunc: func(ctx context.Context, config map[string]interface{}) (map[string]models.SensorReading, *models.StationData, error) {
			return nil, nil, expectedError
		},
	}

	ctx := context.Background()
	config := map[string]interface{}{"api_key": "test"}

	readings, station, err := puller.Pull(ctx, config)

	if err == nil {
		t.Error("Expected error to be returned")
	}

	if err != expectedError {
		t.Errorf("Expected error '%v', got '%v'", expectedError, err)
	}

	if readings != nil {
		t.Error("Expected nil readings on error")
	}

	if station != nil {
		t.Error("Expected nil station data on error")
	}
}

func TestMockPuller_Pull_ContextCancellation(t *testing.T) {
	puller := &MockPuller{
		providerType: "testprovider",
		pullFunc: func(ctx context.Context, config map[string]interface{}) (map[string]models.SensorReading, *models.StationData, error) {
			select {
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return nil, nil, nil
			}
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	config := map[string]interface{}{"api_key": "test"}

	_, _, err := puller.Pull(ctx, config)

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

func TestMockPuller_ValidateConfig_Success(t *testing.T) {
	puller := &MockPuller{
		providerType: "testprovider",
		validateFunc: func(config map[string]interface{}) error {
			if _, ok := config["api_key"]; !ok {
				return errors.New("api_key required")
			}
			return nil
		},
	}

	config := map[string]interface{}{"api_key": "test"}

	err := puller.ValidateConfig(config)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if puller.validateCallCount != 1 {
		t.Errorf("Expected ValidateConfig to be called once, got %d", puller.validateCallCount)
	}
}

func TestMockPuller_ValidateConfig_Error(t *testing.T) {
	puller := &MockPuller{
		providerType: "testprovider",
		validateFunc: func(config map[string]interface{}) error {
			if _, ok := config["api_key"]; !ok {
				return errors.New("api_key required")
			}
			return nil
		},
	}

	config := map[string]interface{}{} // Missing api_key

	err := puller.ValidateConfig(config)

	if err == nil {
		t.Error("Expected validation error")
	}

	if err.Error() != "api_key required" {
		t.Errorf("Expected 'api_key required' error, got '%v'", err)
	}
}

func TestMockPuller_ValidateConfig_NilConfig(t *testing.T) {
	puller := &MockPuller{
		providerType: "testprovider",
		validateFunc: func(config map[string]interface{}) error {
			if config == nil {
				return errors.New("config cannot be nil")
			}
			return nil
		},
	}

	err := puller.ValidateConfig(nil)

	if err == nil {
		t.Error("Expected validation error for nil config")
	}
}

func TestPullerRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewPullerRegistry()

	done := make(chan bool)

	// Concurrent registration
	for i := 0; i < 10; i++ {
		go func(id int) {
			puller := &MockPuller{
				providerType: "provider" + string(rune('0'+id)),
			}
			registry.Register(puller)
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
			providerType := "provider" + string(rune('0'+id))
			_, ok := registry.Get(providerType)
			if !ok {
				t.Errorf("Expected to find provider %s", providerType)
			}
			done <- true
		}(i)
	}

	// Wait for all reads
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all pullers are registered
	all := registry.All()
	if len(all) != 10 {
		t.Errorf("Expected 10 pullers after concurrent operations, got %d", len(all))
	}
}

func TestMockPuller_CallCounts(t *testing.T) {
	puller := &MockPuller{
		providerType: "testprovider",
	}

	ctx := context.Background()
	config := map[string]interface{}{"api_key": "test"}

	// Call Pull multiple times
	for i := 0; i < 3; i++ {
		puller.Pull(ctx, config)
	}

	if puller.pullCallCount != 3 {
		t.Errorf("Expected Pull to be called 3 times, got %d", puller.pullCallCount)
	}

	// Call ValidateConfig multiple times
	for i := 0; i < 5; i++ {
		puller.ValidateConfig(config)
	}

	if puller.validateCallCount != 5 {
		t.Errorf("Expected ValidateConfig to be called 5 times, got %d", puller.validateCallCount)
	}
}

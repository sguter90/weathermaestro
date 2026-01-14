package puller

import (
	"context"

	"github.com/google/uuid"
	"github.com/sguter90/weathermaestro/pkg/models"
)

// Puller defines the interface for weather data providers that push data via HTTP
type Puller interface {
	// GetProviderType returns the provider type identifier (e.g., "openweathermap", "weatherapi")
	GetProviderType() string

	// Pull fetches weather data from the external provider
	// ctx: context for cancellation and timeouts
	// config: provider-specific configuration (API keys, station IDs, etc.)
	Pull(ctx context.Context, config map[string]string) (map[uuid.UUID]models.SensorReading, *models.StationData, error)

	// ValidateConfig checks if the provided configuration is valid for this provider
	ValidateConfig(config map[string]string) error
}

// PullerRegistry holds all registered data pullers
type PullerRegistry struct {
	pullers map[string]Puller
}

// NewPullerRegistry creates a new puller registry
func NewPullerRegistry() *PullerRegistry {
	return &PullerRegistry{
		pullers: make(map[string]Puller),
	}
}

// Register adds a puller to the registry
func (r *PullerRegistry) Register(p Puller) {
	r.pullers[p.GetProviderType()] = p
}

// Get retrieves a puller by provider type
func (r *PullerRegistry) Get(providerType string) (Puller, bool) {
	p, ok := r.pullers[providerType]
	return p, ok
}

// All returns all registered pullers
func (r *PullerRegistry) All() []Puller {
	pullers := make([]Puller, 0, len(r.pullers))
	for _, p := range r.pullers {
		pullers = append(pullers, p)
	}
	return pullers
}

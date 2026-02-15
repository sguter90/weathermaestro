package pusher

import (
	"net/url"
	"sync"

	"github.com/google/uuid"
	"github.com/sguter90/weathermaestro/pkg/models"
)

// Pusher defines the interface for all weather station pushers
type Pusher interface {
	// GetEndpoint returns the HTTP endpoint path for this pusher
	GetEndpoint() string

	// ParseStation converts URL parameters to StationData
	ParseStation(params url.Values) *models.StationData

	// ParseSensors converts URL parameters to SensorMap indexed by remoted ID
	ParseSensors(params url.Values) map[string]models.Sensor

	// ParseWeatherData converts URL parameters to WeatherData
	ParseWeatherData(params url.Values, sensors map[string]models.Sensor) (map[uuid.UUID]models.SensorReading, error)

	// GetStationType returns the station type identifier
	GetStationType() string
}

// Registry holds all registered pushers
type Registry struct {
	mu      sync.RWMutex
	pushers map[string]Pusher
}

// NewRegistry creates a new pusher registry
func NewRegistry() *Registry {
	return &Registry{
		pushers: make(map[string]Pusher),
	}
}

// Register adds a pusher to the registry
func (r *Registry) Register(p Pusher) {
	if p == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.pushers[p.GetStationType()] = p
}

// Get retrieves a pusher by station type
func (r *Registry) Get(stationType string) (Pusher, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.pushers[stationType]
	return p, ok
}

// All returns all registered pushers
func (r *Registry) All() []Pusher {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pushers := make([]Pusher, 0, len(r.pushers))
	for _, p := range r.pushers {
		pushers = append(pushers, p)
	}
	return pushers
}

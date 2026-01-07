package pusher

import (
	"github.com/sguter90/weathermaestro/pkg/models"
	"net/url"
)

// Pusher defines the interface for all weather station pushers
type Pusher interface {
	// GetEndpoint returns the HTTP endpoint path for this pusher
	GetEndpoint() string

	// Parse converts URL parameters to WeatherData
	Parse(params url.Values) (*models.WeatherData, *models.StationData, error)

	// GetStationType returns the station type identifier
	GetStationType() string
}

// Registry holds all registered pushers
type Registry struct {
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
	r.pushers[p.GetStationType()] = p
}

// Get retrieves a pusher by station type
func (r *Registry) Get(stationType string) (Pusher, bool) {
	p, ok := r.pushers[stationType]
	return p, ok
}

// All returns all registered pushers
func (r *Registry) All() []Pusher {
	pushers := make([]Pusher, 0, len(r.pushers))
	for _, p := range r.pushers {
		pushers = append(pushers, p)
	}
	return pushers
}

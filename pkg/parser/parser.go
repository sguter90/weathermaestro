package parser

import (
	"github.com/sguter90/weathermaestro/pkg/models"
	"net/url"
)

// Parser defines the interface for all weather station parsers
type Parser interface {
	// GetEndpoint returns the HTTP endpoint path for this parser
	GetEndpoint() string

	// Parse converts URL parameters to WeatherData
	Parse(params url.Values) (*models.WeatherData, *models.StationData, error)

	// GetStationType returns the station type identifier
	GetStationType() string
}

// Registry holds all registered parsers
type Registry struct {
	parsers map[string]Parser
}

// NewRegistry creates a new parser registry
func NewRegistry() *Registry {
	return &Registry{
		parsers: make(map[string]Parser),
	}
}

// Register adds a parser to the registry
func (r *Registry) Register(p Parser) {
	r.parsers[p.GetStationType()] = p
}

// Get retrieves a parser by station type
func (r *Registry) Get(stationType string) (Parser, bool) {
	p, ok := r.parsers[stationType]
	return p, ok
}

// All returns all registered parsers
func (r *Registry) All() []Parser {
	parsers := make([]Parser, 0, len(r.parsers))
	for _, p := range r.parsers {
		parsers = append(parsers, p)
	}
	return parsers
}

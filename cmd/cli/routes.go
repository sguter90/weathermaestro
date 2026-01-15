package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sguter90/weathermaestro/pkg/database"
)

// RouteManager handles all API routes
type RouteManager struct {
	dbManager       *database.DatabaseManager
	registryManager *RegistryManager
	Router          *mux.Router
}

// NewRouteManager creates a new RouteManager instance
func NewRouteManager(dbManager *database.DatabaseManager, registryManager *RegistryManager) *RouteManager {
	return &RouteManager{
		dbManager:       dbManager,
		registryManager: registryManager,
		Router:          mux.NewRouter(),
	}
}

// Setup configures all API routes
func (rm *RouteManager) Setup() {
	r := rm.Router
	r.Use(rm.corsMiddleware)
	r.Use(rm.contextMiddleware)

	// Health check
	r.HandleFunc("/health", rm.healthHandler).Methods("GET")

	// Dynamic pusher endpoints
	rm.setupPusherEndpoints(r)

	// API v1 routes
	api := r.PathPrefix("/api/v1").Subrouter()
	rm.setupAPIRoutes(api)

	// OAuth callbacks
	rm.setupOAuthRoutes(r)
}

// setupPusherEndpoints registers dynamic pusher endpoints
func (rm *RouteManager) setupPusherEndpoints(r *mux.Router) {
	for _, p := range rm.registryManager.PusherRegistry.All() {
		endpoint := p.GetEndpoint()
		log.Printf("✓ Registering endpoint: %s for station type: %s", endpoint, p.GetStationType())
		r.HandleFunc(endpoint, rm.weatherUpdateHandler(p)).Methods("GET", "POST")
	}
}

// setupAPIRoutes configures all API v1 routes
func (rm *RouteManager) setupAPIRoutes(api *mux.Router) {
	// Stations
	api.HandleFunc("/stations", rm.getStationsHandler).Methods("GET")
	api.HandleFunc("/stations/{id}", rm.getStationHandler).Methods("GET")

	// Sensors
	api.HandleFunc("/stations/{id}/sensors", rm.getSensorsHandler).Methods("GET")
	api.HandleFunc("/sensors/{id}", rm.getSensorHandler).Methods("GET")

	// Readings
	api.HandleFunc("/readings", rm.getReadingsHandler).Methods("GET")
}

// setupOAuthRoutes configures OAuth callback routes
func (rm *RouteManager) setupOAuthRoutes(r *mux.Router) {
	r.HandleFunc("/netatmo/callback/{stationID}", rm.netatmoCallbackHandler).Methods("GET")
}

// contextMiddleware adds database context to requests
func (rm *RouteManager) contextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), "db", rm.dbManager.GetDB())
		ctx = context.WithValue(ctx, "dbManager", rm.dbManager)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

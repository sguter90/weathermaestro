package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sguter90/weathermaestro/pkg/pusher"
)

// setupRoutes configures all API routes
func setupRoutes(r *mux.Router, db *sql.DB, registry *pusher.Registry) {
	r.Use(corsMiddleware)

	// Health check
	r.HandleFunc("/health", healthHandler).Methods("GET")

	// API v1 routes
	api := r.PathPrefix("/api/v1").Subrouter()

	// Dynamic pusher endpoints
	for _, p := range registry.All() {
		endpoint := p.GetEndpoint()
		log.Printf("Registering endpoint: %s for station type: %s", endpoint, p.GetStationType())
		r.HandleFunc(endpoint, weatherUpdateHandler(db, p)).Methods("GET", "POST")
	}

	// Stations
	api.HandleFunc("/stations", getStationsHandler(db)).Methods("GET")
	api.HandleFunc("/stations/{id}", getStationHandler(db)).Methods("GET")
	api.HandleFunc("/stations/{id}/weather/current", getCurrentWeatherHandler(db)).Methods("GET")
	api.HandleFunc("/stations/{id}/weather/history", getWeatherHistoryHandler(db)).Methods("GET")
}

// healthHandler returns server health status
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "weathermaestro",
		"version": "1.0.0",
	})
}

package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sguter90/weathermaestro/pkg/database"
	"github.com/sguter90/weathermaestro/pkg/puller/netatmo"
	"github.com/sguter90/weathermaestro/pkg/pusher"
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

	// Add database to request context
	rm.Router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), "db", rm.dbManager.GetDB())
			ctx = context.WithValue(ctx, "dbManager", rm.dbManager)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	// Health check
	r.HandleFunc("/health", rm.healthHandler).Methods("GET")

	// API v1 routes
	api := r.PathPrefix("/api/v1").Subrouter()

	// Dynamic pusher endpoints
	for _, p := range rm.registryManager.PusherRegistry.All() {
		endpoint := p.GetEndpoint()
		log.Printf("✓ Registering endpoint: %s for station type: %s", endpoint, p.GetStationType())
		r.HandleFunc(endpoint, rm.weatherUpdateHandler(p)).Methods("GET", "POST")
	}

	// Stations
	api.HandleFunc("/stations", rm.getStationsHandler).Methods("GET")
	api.HandleFunc("/stations/{id}", rm.getStationHandler).Methods("GET")
	api.HandleFunc("/stations/{id}/weather/current", rm.getCurrentWeatherHandler).Methods("GET")
	api.HandleFunc("/stations/{id}/weather/history", rm.getWeatherHistoryHandler).Methods("GET")

	// Netatmo OAuth callback
	r.HandleFunc("/netatmo/callback/{stationID}", rm.netatmoCallbackHandler).Methods("GET")
}

// corsMiddleware handles CORS headers
func (rm *RouteManager) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get allowed origins from environment variable
		allowedOriginsEnv := getEnv("SERVER_ALLOWED_ORIGINS", "")
		var allowedOrigins []string

		if allowedOriginsEnv != "" {
			// Split comma-separated origins
			allowedOrigins = strings.Split(allowedOriginsEnv, ",")
			// Trim whitespace from each origin
			for i, origin := range allowedOrigins {
				allowedOrigins[i] = strings.TrimSpace(origin)
			}
		} else {
			// Default fallback if not configured
			allowedOrigins = []string{
				"http://localhost:5173",
				"http://localhost:3000",
			}
		}

		// Check if origin is allowed
		origin := r.Header.Get("Origin")
		if origin != "" {
			isAllowed := false
			for _, allowed := range allowedOrigins {
				if origin == allowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					isAllowed = true
					break
				}
			}
			if !isAllowed {
				log.Printf("Origin '%s' is not within allowed origins: %s", origin, strings.Join(allowedOrigins, ", "))
			}
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "3600")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// healthHandler returns server health status
func (rm *RouteManager) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// weatherUpdateHandler handles incoming weather data from stations
func (rm *RouteManager) weatherUpdateHandler(p pusher.Pusher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// ParseWeatherData query parameters
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		stationData := p.ParseStation(r.Form)

		// Ensure station exists
		stationID, err := rm.dbManager.EnsureStation(stationData)
		if err != nil {
			log.Printf("❌ Failed to ensure station: %v", err)
			http.Error(w, "Failed to ensure station", http.StatusInternalServerError)
			return
		}

		sensors := p.ParseSensors(r.Form)
		// Ensure sensors exist
		sensors, err = rm.dbManager.EnsureSensorsByRemoteId(stationID, sensors)
		if err != nil {
			log.Printf("❌ F Failed to ensure sensors: %v", err)
			http.Error(w, "Failed to ensure sensors", http.StatusInternalServerError)
			return
		}
		if len(sensors) == 0 {
			log.Printf("❌ No sensors found for station ID: %s", stationID.String())
			http.Error(w, "No sensors found for station ID", http.StatusBadRequest)
			return
		}

		// ParseWeatherData weather data using pusher
		readings, err := p.ParseWeatherData(r.Form, sensors)
		if err != nil {
			log.Printf("❌ Failed to parse weather data: %v", err)
			http.Error(w, "Failed to parse weather data", http.StatusBadRequest)
			return
		}

		// Store weather data
		for _, reading := range readings {
			if err := rm.dbManager.StoreSensorReading(reading.SensorID, reading.Value, reading.DateUTC); err != nil {
				log.Printf("❌ Failed to store reading: %v", err)
				http.Error(w, "Failed to store readings", http.StatusInternalServerError)
				return
			}
		}

		log.Printf("✓ Pushed %d Weather readings for station ID: %s", len(readings), stationID.String())

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"status":     "success",
			"message":    "Weather data stored successfully",
			"station_id": stationID.String(),
		})
	}
}

// getStationsHandler returns all registered weather stations
func (rm *RouteManager) getStationsHandler(w http.ResponseWriter, r *http.Request) {
	stations, err := rm.dbManager.GetStationList()
	if err != nil {
		log.Printf("❌ Failed to query stations: %v", err)
		http.Error(w, "Failed to query stations", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stations)
}

// getStationHandler returns details for a specific station
func (rm *RouteManager) getStationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	stationIDStr := vars["id"]
	stationID, err := uuid.Parse(stationIDStr)
	if err != nil {
		http.Error(w, "Invalid station_id format", http.StatusBadRequest)
		return
	}

	station, err := rm.dbManager.GetStation(stationID)
	if err != nil {
		log.Printf("❌ Failed to query station: %v", err)
		http.Error(w, "Station not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(station)
}

// getCurrentWeatherHandler returns the most recent weather data for a specific station
func (rm *RouteManager) getCurrentWeatherHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	stationIDStr := vars["id"]
	stationID, err := uuid.Parse(stationIDStr)
	if err != nil {
		http.Error(w, "Invalid station_id format", http.StatusBadRequest)
		return
	}

	weatherData, err := rm.dbManager.GetCurrentWeather(stationID)
	if err != nil {
		log.Printf("❌ Failed to query weather data: %v", err)
		http.Error(w, "No weather data available for this station", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(weatherData)
}

// getWeatherHistoryHandler returns historical weather data
func (rm *RouteManager) getWeatherHistoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	stationIDStr := vars["id"]
	stationID, err := uuid.Parse(stationIDStr)
	if err != nil {
		http.Error(w, "Invalid station_id format", http.StatusBadRequest)
		return
	}

	// ParseWeatherData query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 100 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	startTime := r.URL.Query().Get("start")
	endTime := r.URL.Query().Get("end")

	weatherDataList, err := rm.dbManager.GetWeatherHistory(stationID, startTime, endTime, limit)
	if err != nil {
		log.Printf("❌ Failed to query weather history: %v", err)
		http.Error(w, "Failed to retrieve weather history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(weatherDataList)
}

// netatmoCallbackHandler handles the OAuth2 callback from Netatmo
func (rm *RouteManager) netatmoCallbackHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	stationIDStr := vars["stationID"]
	stationID, err := uuid.Parse(stationIDStr)
	if err != nil {
		http.Error(w, "Invalid station_id format", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	if state == "" {
		http.Error(w, "Missing state parameter", http.StatusBadRequest)
		return
	}

	// Get current config
	config, err := rm.dbManager.GetStationConfig(stationID)
	if err != nil {
		http.Error(w, "Config error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Exchange code for access token
	clientID, ok := config["client_id"].(string)
	if !ok {
		http.Error(w, "Invalid client_id in config", http.StatusInternalServerError)
		return
	}

	clientSecret, ok := config["client_secret"].(string)
	if !ok {
		http.Error(w, "Invalid client_secret in config", http.StatusInternalServerError)
		return
	}

	redirectURI, ok := config["redirect_uri"].(string)
	if !ok {
		http.Error(w, "Invalid redirect_uri in config", http.StatusInternalServerError)
		return
	}

	dbState, ok := config["state"].(string)
	if !ok {
		http.Error(w, "Invalid state in config", http.StatusInternalServerError)
		return
	}

	client := netatmo.NewClient(clientID, clientSecret, redirectURI)
	client.SetState(dbState)
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := client.GetAccessTokenFromCode(ctx, code, state); err != nil {
		log.Printf("Failed to get access token: %v", err)
		http.Error(w, "Failed to get access token", http.StatusInternalServerError)
		return
	}

	// Update config with access token
	config["access_token"] = client.GetAccessToken()
	config["refresh_token"] = client.GetAccessToken()
	config["token_expiry"] = client.GetTokenExpiry().Format(time.RFC3339)

	err = rm.dbManager.SetStationConfig(stationID, config)
	if err != nil {
		log.Printf("Failed to update station config: %v", err)
		http.Error(w, "Failed to save access token", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully saved access token for station %s", stationID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Authorization successful! You can close this window.",
	})
}

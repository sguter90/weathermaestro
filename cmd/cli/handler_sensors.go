package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sguter90/weathermaestro/pkg/models"
)

// getSensorsHandler returns sensors with flexible filtering
// Query params:
//   - sensor_type: filter by sensor type (temperature, humidity, etc.)
//   - location: filter by location (indoor, outdoor)
//   - enabled: filter by enabled status (true/false)
//   - include_latest: include latest reading for each sensor (true/false)
func (rm *RouteManager) getSensorsHandler(w http.ResponseWriter, r *http.Request) {
	params := parseSensorQueryParams(r)
	vars := mux.Vars(r)
	stationId, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid sensor_id format", http.StatusBadRequest)
		return
	}

	params.StationID = &stationId

	sensors, err := rm.dbManager.GetSensors(params)
	if err != nil {
		log.Printf("❌ Failed to query sensors: %v", err)
		http.Error(w, "Failed to query sensors", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sensors)
}

// getSensorHandler returns a single sensor by ID
// Query params:
//   - include_latest: include latest reading (true/false)
func (rm *RouteManager) getSensorHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sensorID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid sensor_id format", http.StatusBadRequest)
		return
	}

	includeLatest := r.URL.Query().Get("include_latest") == "true"

	sensor, err := rm.dbManager.GetSensor(sensorID, includeLatest)
	if err != nil {
		log.Printf("❌ Failed to query sensor: %v", err)
		http.Error(w, "Sensor not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sensor)
}

// parseSensorQueryParams extracts and parses query parameters from the request
func parseSensorQueryParams(r *http.Request) models.SensorQueryParams {
	params := models.SensorQueryParams{
		SensorType:    r.URL.Query().Get("sensor_type"),
		Location:      r.URL.Query().Get("location"),
		IncludeLatest: r.URL.Query().Get("include_latest") == "true",
	}

	// Parse station_id
	if stationIDStr := r.URL.Query().Get("station_id"); stationIDStr != "" {
		if id, err := uuid.Parse(stationIDStr); err == nil {
			params.StationID = &id
		}
	}

	// Parse enabled
	if enabledStr := r.URL.Query().Get("enabled"); enabledStr != "" {
		enabled := enabledStr == "true"
		params.Enabled = &enabled
	}

	return params
}

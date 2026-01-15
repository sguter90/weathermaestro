package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

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

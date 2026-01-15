package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/sguter90/weathermaestro/pkg/pusher"
)

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

		log.Printf("✓ Pushed %d Weather readings for station: %s", len(readings), stationData.StationType)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"status":     "success",
			"message":    "Weather data stored successfully",
			"station_id": stationID.String(),
		})
	}
}

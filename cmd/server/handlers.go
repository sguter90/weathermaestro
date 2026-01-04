package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sguter90/weathermaestro/pkg/models"
	"github.com/sguter90/weathermaestro/pkg/parser"
)

// CORS Middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get allowed origins from environment variable
		allowedOriginsEnv := os.Getenv("SERVER_ALLOWED_ORIGINS")
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
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				break
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

// weatherUpdateHandler handles incoming weather data from stations
func weatherUpdateHandler(db *sql.DB, p parser.Parser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse query parameters
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form weatherData", http.StatusBadRequest)
			return
		}

		// Parse weather weatherData using the appropriate parser
		weatherData, stationData, err := p.Parse(r.Form)
		if err != nil {
			log.Printf("Failed to parse weather weatherData: %v", err)
			http.Error(w, "Failed to parse weather weatherData", http.StatusBadRequest)
			return
		}

		// Ensure station exists (create if not)
		stationID, err := ensureStation(db, stationData)
		if err != nil {
			log.Printf("Failed to ensure station exists: %v", err)
			http.Error(w, "Failed to process station", http.StatusInternalServerError)
			return
		}

		// Store weather weatherData with station_id
		if err := storeWeatherData(db, weatherData, stationID); err != nil {
			log.Printf("Failed to store weather weatherData: %v", err)
			http.Error(w, "Failed to store weather weatherData", http.StatusInternalServerError)
			return
		}

		// Return success response with station ID
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"status":     "success",
			"message":    "Weather weatherData stored successfully",
			"station_id": stationID.String(),
		})
	}
}

// ensureStation checks if a station exists and creates it if not
func ensureStation(db *sql.DB, data *models.StationData) (uuid.UUID, error) {
	var stationID uuid.UUID

	// Try to find existing station by passKey
	err := db.QueryRow(`
		SELECT id FROM stations 
		WHERE pass_key = $1
	`, data.PassKey).Scan(&stationID)

	if err == nil {
		// Station exists, update last_seen
		_, err = db.Exec(`
			UPDATE stations 
			SET updated_at = CURRENT_TIMESTAMP 
			WHERE id = $1
		`, stationID)
		return stationID, err
	}

	if err != sql.ErrNoRows {
		return uuid.Nil, err
	}

	// Station doesn't exist, create it
	stationID = uuid.New()
	_, err = db.Exec(`
		INSERT INTO stations (id, pass_key, station_type, model, freq)
		VALUES ($1, $2, $3, $4, $5)
	`, stationID, data.PassKey, data.StationType, data.Model, data.Freq)

	if err != nil {
		return uuid.Nil, err
	}

	log.Printf("Created new station: %s (ID: %s)", data.PassKey, stationID)
	return stationID, nil
}

// getStationCurrentWeatherHandler returns the most recent weather data for a specific station
func getCurrentWeatherHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		stationID := vars["id"]

		query := `
			SELECT 
				wd.date_utc,
				wd.runtime, wd.heap,
				wd.temp_in_c, wd.temp_in_f, wd.humidity_in,
				wd.temp_out_c, wd.temp_out_f, wd.humidity_out,
				wd.barom_rel_hpa, wd.barom_abs_hpa, wd.barom_rel_in, wd.barom_abs_in,
				wd.wind_dir, wd.wind_speed_ms, wd.wind_gust_ms, wd.max_daily_gust_ms,
				wd.wind_speed_kmh, wd.wind_gust_kmh, wd.max_daily_gust_kmh,
				wd.wind_speed_mph, wd.wind_gust_mph, wd.max_daily_gust_mph,
				wd.solar_radiation, wd.uv,
				wd.rain_rate_mm_h, wd.event_rain_mm, wd.hourly_rain_mm, wd.daily_rain_mm,
				wd.weekly_rain_mm, wd.monthly_rain_mm, wd.yearly_rain_mm, wd.total_rain_mm,
				wd.rain_rate_in, wd.event_rain_in, wd.hourly_rain_in, wd.daily_rain_in,
				wd.weekly_rain_in, wd.monthly_rain_in, wd.yearly_rain_in, wd.total_rain_in,
				wd.vpd, wd.wh65_batt
			FROM weather_data wd
			JOIN stations s ON wd.station_id = s.id
			WHERE s.id = $1
			ORDER BY wd.date_utc DESC
			LIMIT 1
		`

		var data models.WeatherData
		err := db.QueryRow(query, stationID).Scan(
			&data.DateUTC,
			&data.Runtime, &data.Heap,
			&data.TempInC, &data.TempInF, &data.HumidityIn,
			&data.TempOutC, &data.TempOutF, &data.HumidityOut,
			&data.BaromRelHPa, &data.BaromAbsHPa, &data.BaromRelIn, &data.BaromAbsIn,
			&data.WindDir, &data.WindSpeedMS, &data.WindGustMS, &data.MaxDailyGustMS,
			&data.WindSpeedKmH, &data.WindGustKmH, &data.MaxDailyGustKmH,
			&data.WindSpeedMPH, &data.WindGustMPH, &data.MaxDailyGustMPH,
			&data.SolarRadiation, &data.UV,
			&data.RainRateMmH, &data.EventRainMm, &data.HourlyRainMm, &data.DailyRainMm,
			&data.WeeklyRainMm, &data.MonthlyRainMm, &data.YearlyRainMm, &data.TotalRainMm,
			&data.RainRateIn, &data.EventRainIn, &data.HourlyRainIn, &data.DailyRainIn,
			&data.WeeklyRainIn, &data.MonthlyRainIn, &data.YearlyRainIn, &data.TotalRainIn,
			&data.VPD, &data.WH65Batt,
		)

		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "No weather data available for this station", http.StatusNotFound)
				return
			}
			log.Printf("Failed to query weather data: %v", err)
			http.Error(w, "Failed to retrieve weather data", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	}
}

// getWeatherHistoryHandler returns historical weather data
func getWeatherHistoryHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		stationIDStr := vars["id"]

		// Parse query parameters
		limitStr := r.URL.Query().Get("limit")
		limit := 100 // default
		if limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
				limit = l
			}
		}

		startTime := r.URL.Query().Get("start")
		endTime := r.URL.Query().Get("end")

		// Build query
		query := `
			SELECT 
				wd.date_utc,
				wd.runtime, wd.heap,
				wd.temp_in_c, wd.temp_in_f, wd.humidity_in,
				wd.temp_out_c, wd.temp_out_f, wd.humidity_out,
				wd.barom_rel_hpa, wd.barom_abs_hpa, wd.barom_rel_in, wd.barom_abs_in,
				wd.wind_dir, wd.wind_speed_ms, wd.wind_gust_ms, wd.max_daily_gust_ms,
				wd.wind_speed_kmh, wd.wind_gust_kmh, wd.max_daily_gust_kmh,
				wd.wind_speed_mph, wd.wind_gust_mph, wd.max_daily_gust_mph,
				wd.solar_radiation, wd.uv,
				wd.rain_rate_mm_h, wd.event_rain_mm, wd.hourly_rain_mm, wd.daily_rain_mm,
				wd.weekly_rain_mm, wd.monthly_rain_mm, wd.yearly_rain_mm, wd.total_rain_mm,
				wd.rain_rate_in, wd.event_rain_in, wd.hourly_rain_in, wd.daily_rain_in,
				wd.weekly_rain_in, wd.monthly_rain_in, wd.yearly_rain_in, wd.total_rain_in,
				wd.vpd, wd.wh65_batt
			FROM weather_data wd
			JOIN stations s ON wd.station_id = s.id
		`

		args := []interface{}{}
		argCount := 1

		if startTime != "" || endTime != "" || stationIDStr != "" {
			query += " WHERE "
			if stationIDStr != "" {
				stationID, err := uuid.Parse(stationIDStr)
				if err != nil {
					http.Error(w, "Invalid station_id format", http.StatusBadRequest)
					return
				}
				query += "wd.station_id = $" + strconv.Itoa(argCount)
				args = append(args, stationID)
				argCount++
			}
			if startTime != "" {
				if stationIDStr != "" {
					query += " AND "
				}
				query += "wd.date_utc >= $" + strconv.Itoa(argCount)
				args = append(args, startTime)
				argCount++
			}
			if endTime != "" {
				if stationIDStr != "" || startTime != "" {
					query += " AND "
				}
				query += "wd.date_utc <= $" + strconv.Itoa(argCount)
				args = append(args, endTime)
				argCount++
			}
		}

		query += " ORDER BY wd.date_utc DESC LIMIT $" + strconv.Itoa(argCount)
		args = append(args, limit)

		rows, err := db.Query(query, args...)
		if err != nil {
			log.Printf("Failed to query weather history: %v", err)
			http.Error(w, "Failed to retrieve weather history", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var weatherDataList []models.WeatherData
		for rows.Next() {
			var data models.WeatherData
			err := rows.Scan(
				&data.DateUTC,
				&data.Runtime, &data.Heap,
				&data.TempInC, &data.TempInF, &data.HumidityIn,
				&data.TempOutC, &data.TempOutF, &data.HumidityOut,
				&data.BaromRelHPa, &data.BaromAbsHPa, &data.BaromRelIn, &data.BaromAbsIn,
				&data.WindDir, &data.WindSpeedMS, &data.WindGustMS, &data.MaxDailyGustMS,
				&data.WindSpeedKmH, &data.WindGustKmH, &data.MaxDailyGustKmH,
				&data.WindSpeedMPH, &data.WindGustMPH, &data.MaxDailyGustMPH,
				&data.SolarRadiation, &data.UV,
				&data.RainRateMmH, &data.EventRainMm, &data.HourlyRainMm, &data.DailyRainMm,
				&data.WeeklyRainMm, &data.MonthlyRainMm, &data.YearlyRainMm, &data.TotalRainMm,
				&data.RainRateIn, &data.EventRainIn, &data.HourlyRainIn, &data.DailyRainIn,
				&data.WeeklyRainIn, &data.MonthlyRainIn, &data.YearlyRainIn, &data.TotalRainIn,
				&data.VPD, &data.WH65Batt,
			)
			if err != nil {
				log.Printf("Failed to scan weather history row: %v", err)
				http.Error(w, "Failed to process weather history data", http.StatusInternalServerError)
				return
			}
			weatherDataList = append(weatherDataList, data)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(weatherDataList)
	}
}

// getStationsHandler returns all registered weather stations
func getStationsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := `
			SELECT DISTINCT 
				s.id, s.pass_key, 
				s.station_type, 
				s.model,
				MAX(wd.date_utc) as last_update
			FROM stations s
			LEFT JOIN weather_data wd ON s.id = wd.station_id
			GROUP BY s.id, s.pass_key, s.station_type, s.model
			ORDER BY last_update DESC
		`

		rows, err := db.Query(query)
		if err != nil {
			log.Printf("Failed to query stations: %v", err)
			http.Error(w, "Failed to retrieve stations", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type Station struct {
			ID          uuid.UUID `json:"id"`
			PassKey     string    `json:"pass_key"`
			StationType string    `json:"station_type"`
			Model       string    `json:"model"`
			LastUpdate  time.Time `json:"last_update"`
		}

		var stations []Station
		for rows.Next() {
			var s Station
			if err := rows.Scan(&s.ID, &s.PassKey, &s.StationType, &s.Model, &s.LastUpdate); err != nil {
				log.Printf("Failed to scan station row: %v", err)
				continue
			}
			stations = append(stations, s)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stations)
	}
}

// getStationHandler returns details for a specific station
func getStationHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		stationId := vars["id"]

		query := `
			SELECT 
				s.id, 
				s.pass_key, 
				s.station_type, 
				s.model,
				COUNT(wd.id) as total_readings,
				MIN(wd.date_utc) as first_reading,
				MAX(wd.date_utc) as last_reading
			FROM stations s
			LEFT JOIN weather_data wd ON s.id = wd.station_id
			WHERE s.id = $1
			GROUP BY s.id
		`

		type StationDetail struct {
			ID            uuid.UUID `json:"id"`
			PassKey       string    `json:"pass_key"`
			StationType   string    `json:"station_type"`
			Model         string    `json:"model"`
			TotalReadings int       `json:"total_readings"`
			FirstReading  time.Time `json:"first_reading"`
			LastReading   time.Time `json:"last_reading"`
		}

		var station StationDetail
		err := db.QueryRow(query, stationId).Scan(
			&station.ID,
			&station.PassKey,
			&station.StationType,
			&station.Model,
			&station.TotalReadings,
			&station.FirstReading,
			&station.LastReading,
		)

		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Station not found", http.StatusNotFound)
				return
			}
			log.Printf("Failed to query station: %v", err)
			http.Error(w, "Failed to retrieve station", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(station)
	}
}

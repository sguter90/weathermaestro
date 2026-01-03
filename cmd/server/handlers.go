package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/sguter90/weathermaestro/pkg/models"
	"github.com/sguter90/weathermaestro/pkg/parser"
)

// weatherUpdateHandler handles incoming weather data from stations
func weatherUpdateHandler(db *sql.DB, p parser.Parser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse query parameters
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form data", http.StatusBadRequest)
			return
		}

		// Parse weather data using the appropriate parser
		data, err := p.Parse(r.Form)
		if err != nil {
			log.Printf("Failed to parse weather data: %v", err)
			http.Error(w, "Failed to parse weather data", http.StatusBadRequest)
			return
		}

		// Store in database
		if err := storeWeatherData(db, data); err != nil {
			log.Printf("Failed to store weather data: %v", err)
			http.Error(w, "Failed to store weather data", http.StatusInternalServerError)
			return
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": "Weather data stored successfully",
		})
	}
}

// storeWeatherData saves weather data to the database
func storeWeatherData(db *sql.DB, data *models.WeatherData) error {
	query := `
        INSERT INTO weather_data (
            pass_key, station_type, model, freq, date_utc, interval,
            runtime, heap,
            temp_in_c, temp_in_f, humidity_in,
            temp_out_c, temp_out_f, humidity_out,
            barom_rel_hpa, barom_abs_hpa, barom_rel_in, barom_abs_in,
            wind_dir, wind_speed_ms, wind_gust_ms, max_daily_gust_ms,
            wind_speed_kmh, wind_gust_kmh, max_daily_gust_kmh,
            wind_speed_mph, wind_gust_mph, max_daily_gust_mph,
            solar_radiation, uv,
            rain_rate_mm_h, event_rain_mm, hourly_rain_mm, daily_rain_mm,
            weekly_rain_mm, monthly_rain_mm, yearly_rain_mm, total_rain_mm,
            rain_rate_in, event_rain_in, hourly_rain_in, daily_rain_in,
            weekly_rain_in, monthly_rain_in, yearly_rain_in, total_rain_in,
            vpd, wh65_batt
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
            $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
            $21, $22, $23, $24, $25, $26, $27, $28, $29, $30,
            $31, $32, $33, $34, $35, $36, $37, $38, $39, $40,
            $41, $42, $43, $44, $45, $46
        )
    `

	_, err := db.Exec(query,
		data.PassKey, data.StationType, data.Model, data.Freq, data.DateUTC, data.Interval,
		data.Runtime, data.Heap,
		data.TempInC, data.TempInF, data.HumidityIn,
		data.TempOutC, data.TempOutF, data.HumidityOut,
		data.BaromRelHPa, data.BaromAbsHPa, data.BaromRelIn, data.BaromAbsIn,
		data.WindDir, data.WindSpeedMS, data.WindGustMS, data.MaxDailyGustMS,
		data.WindSpeedKmH, data.WindGustKmH, data.MaxDailyGustKmH,
		data.WindSpeedMPH, data.WindGustMPH, data.MaxDailyGustMPH,
		data.SolarRadiation, data.UV,
		data.RainRateMmH, data.EventRainMm, data.HourlyRainMm, data.DailyRainMm,
		data.WeeklyRainMm, data.MonthlyRainMm, data.YearlyRainMm, data.TotalRainMm,
		data.RainRateIn, data.EventRainIn, data.HourlyRainIn, data.DailyRainIn,
		data.WeeklyRainIn, data.MonthlyRainIn, data.YearlyRainIn, data.TotalRainIn,
		data.VPD, data.WH65Batt,
	)

	return err
}

// getCurrentWeatherHandler returns the most recent weather data
func getCurrentWeatherHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := `
            SELECT 
                pass_key, station_type, model, freq, date_utc, interval,
                runtime, heap,
                temp_in_c, temp_in_f, humidity_in,
                temp_out_c, temp_out_f, humidity_out,
                barom_rel_hpa, barom_abs_hpa, barom_rel_in, barom_abs_in,
                wind_dir, wind_speed_ms, wind_gust_ms, max_daily_gust_ms,
                wind_speed_kmh, wind_gust_kmh, max_daily_gust_kmh,
                wind_speed_mph, wind_gust_mph, max_daily_gust_mph,
                solar_radiation, uv,
                rain_rate_mm_h, event_rain_mm, hourly_rain_mm, daily_rain_mm,
                weekly_rain_mm, monthly_rain_mm, yearly_rain_mm, total_rain_mm,
                rain_rate_in, event_rain_in, hourly_rain_in, daily_rain_in,
                weekly_rain_in, monthly_rain_in, yearly_rain_in, total_rain_in,
                vpd, wh65_batt
            FROM weather_data
            ORDER BY date_utc DESC
            LIMIT 1
        `

		var data models.WeatherData
		err := db.QueryRow(query).Scan(
			&data.PassKey, &data.StationType, &data.Model, &data.Freq, &data.DateUTC, &data.Interval,
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
				http.Error(w, "No weather data available", http.StatusNotFound)
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
                pass_key, station_type, model, freq, date_utc, interval,
                runtime, heap,
                temp_in_c, temp_in_f, humidity_in,
                temp_out_c, temp_out_f, humidity_out,
                barom_rel_hpa, barom_abs_hpa, barom_rel_in, barom_abs_in,
                wind_dir, wind_speed_ms, wind_gust_ms, max_daily_gust_ms,
                wind_speed_kmh, wind_gust_kmh, max_daily_gust_kmh,
                wind_speed_mph, wind_gust_mph, max_daily_gust_mph,
                solar_radiation, uv,
                rain_rate_mm_h, event_rain_mm, hourly_rain_mm, daily_rain_mm,
                weekly_rain_mm, monthly_rain_mm, yearly_rain_mm, total_rain_mm,
                rain_rate_in, event_rain_in, hourly_rain_in, daily_rain_in,
                weekly_rain_in, monthly_rain_in, yearly_rain_in, total_rain_in,
                vpd, wh65_batt
            FROM weather_data
        `

		args := []interface{}{}
		argCount := 1

		if startTime != "" || endTime != "" {
			query += " WHERE "
			if startTime != "" {
				query += "date_utc >= $" + strconv.Itoa(argCount)
				args = append(args, startTime)
				argCount++
			}
			if endTime != "" {
				if startTime != "" {
					query += " AND "
				}
				query += "date_utc <= $" + strconv.Itoa(argCount)
				args = append(args, endTime)
				argCount++
			}
		}

		query += " ORDER BY date_utc DESC LIMIT $" + strconv.Itoa(argCount)
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
				&data.PassKey, &data.StationType, &data.Model, &data.Freq, &data.DateUTC, &data.Interval,
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
                pass_key, 
                station_type, 
                model,
                MAX(date_utc) as last_update
            FROM weather_data
            GROUP BY pass_key, station_type, model
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
			PassKey     string    `json:"pass_key"`
			StationType string    `json:"station_type"`
			Model       string    `json:"model"`
			LastUpdate  time.Time `json:"last_update"`
		}

		var stations []Station
		for rows.Next() {
			var s Station
			if err := rows.Scan(&s.PassKey, &s.StationType, &s.Model, &s.LastUpdate); err != nil {
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
		passKey := vars["id"]

		query := `
            SELECT 
                pass_key, 
                station_type, 
                model,
                COUNT(*) as total_readings,
                MIN(date_utc) as first_reading,
                MAX(date_utc) as last_reading
            FROM weather_data
            WHERE pass_key = $1
            GROUP BY pass_key, station_type, model
        `

		type StationDetail struct {
			PassKey       string    `json:"pass_key"`
			StationType   string    `json:"station_type"`
			Model         string    `json:"model"`
			TotalReadings int       `json:"total_readings"`
			FirstReading  time.Time `json:"first_reading"`
			LastReading   time.Time `json:"last_reading"`
		}

		var station StationDetail
		err := db.QueryRow(query, passKey).Scan(
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

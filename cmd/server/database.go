package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/sguter90/weathermaestro/pkg/migrations"
	"github.com/sguter90/weathermaestro/pkg/models"
)

// connectDatabase establishes a connection to the PostgreSQL database
func connectDatabase() (*sql.DB, error) {
	// Get database configuration from environment variables with defaults
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "weather_user")
	password := getEnv("DB_PASSWORD", "weather_pass")
	dbname := getEnv("DB_NAME", "weather_db")
	sslmode := getEnv("DB_SSLMODE", "disable")

	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode,
	)

	// Open database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	log.Printf("Successfully connected to database: %s@%s:%s/%s", user, host, port, dbname)

	return db, nil
}

// initDatabase runs database migrations
func initDatabase(db *sql.DB) error {
	log.Println("Running database migrations...")

	// Create migration runner
	runner, err := migrations.NewRunner(db)
	if err != nil {
		return fmt.Errorf("failed to create migration runner: %w", err)
	}

	// Run all pending migrations
	if err := runner.Run(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Database initialization completed successfully")
	return nil
}

// storeWeatherData saves weather data to the database
func storeWeatherData(db *sql.DB, data *models.WeatherData, stationID uuid.UUID) error {
	query := `
		INSERT INTO weather_data (
			station_id, date_utc,
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
			$41, $42, $43, $44
		)
	`

	_, err := db.Exec(query,
		stationID, data.DateUTC,
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

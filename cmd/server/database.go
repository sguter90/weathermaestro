package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
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

// initDatabase runs initial database setup (migrations)
func initDatabase(db *sql.DB) error {
	// Create weather_data table if it doesn't exist
	createTableQuery := `
        CREATE TABLE IF NOT EXISTS weather_data (
            id SERIAL PRIMARY KEY,
            pass_key VARCHAR(255) NOT NULL,
            station_type VARCHAR(50),
            model VARCHAR(100),
            freq VARCHAR(20),
            date_utc TIMESTAMPTZ NOT NULL,
            interval INTEGER,
            runtime INTEGER,
            heap INTEGER,
            
            -- Indoor measurements
            temp_in_c DOUBLE PRECISION,
            temp_in_f DOUBLE PRECISION,
            humidity_in INTEGER,
            
            -- Outdoor measurements
            temp_out_c DOUBLE PRECISION,
            temp_out_f DOUBLE PRECISION,
            humidity_out INTEGER,
            
            -- Barometric pressure
            barom_rel_hpa DOUBLE PRECISION,
            barom_abs_hpa DOUBLE PRECISION,
            barom_rel_in DOUBLE PRECISION,
            barom_abs_in DOUBLE PRECISION,
            
            -- Wind measurements
            wind_dir INTEGER,
            wind_speed_ms DOUBLE PRECISION,
            wind_gust_ms DOUBLE PRECISION,
            max_daily_gust_ms DOUBLE PRECISION,
            wind_speed_kmh DOUBLE PRECISION,
            wind_gust_kmh DOUBLE PRECISION,
            max_daily_gust_kmh DOUBLE PRECISION,
            wind_speed_mph DOUBLE PRECISION,
            wind_gust_mph DOUBLE PRECISION,
            max_daily_gust_mph DOUBLE PRECISION,
            
            -- Solar and UV
            solar_radiation DOUBLE PRECISION,
            uv INTEGER,
            
            -- Rain measurements (metric)
            rain_rate_mm_h DOUBLE PRECISION,
            event_rain_mm DOUBLE PRECISION,
            hourly_rain_mm DOUBLE PRECISION,
            daily_rain_mm DOUBLE PRECISION,
            weekly_rain_mm DOUBLE PRECISION,
            monthly_rain_mm DOUBLE PRECISION,
            yearly_rain_mm DOUBLE PRECISION,
            total_rain_mm DOUBLE PRECISION,
            
            -- Rain measurements (imperial)
            rain_rate_in DOUBLE PRECISION,
            event_rain_in DOUBLE PRECISION,
            hourly_rain_in DOUBLE PRECISION,
            daily_rain_in DOUBLE PRECISION,
            weekly_rain_in DOUBLE PRECISION,
            monthly_rain_in DOUBLE PRECISION,
            yearly_rain_in DOUBLE PRECISION,
            total_rain_in DOUBLE PRECISION,
            
            -- Additional measurements
            vpd DOUBLE PRECISION,
            wh65_batt INTEGER,
            
            created_at TIMESTAMPTZ DEFAULT NOW()
        );

        -- Create indexes for better query performance
        CREATE INDEX IF NOT EXISTS idx_weather_data_date_utc ON weather_data(date_utc DESC);
        CREATE INDEX IF NOT EXISTS idx_weather_data_pass_key ON weather_data(pass_key);
        CREATE INDEX IF NOT EXISTS idx_weather_data_station_type ON weather_data(station_type);
    `

	if _, err := db.Exec(createTableQuery); err != nil {
		return fmt.Errorf("failed to create weather_data table: %w", err)
	}

	log.Println("Database tables initialized successfully")
	return nil
}

package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/sguter90/weathermaestro/pkg/models"
)

// DatabaseManager handles all database operations
type DatabaseManager struct {
	db            *sql.DB
	healthChecker *HealthChecker
}

// NewDatabaseManager creates a new DatabaseManager instance
func NewDatabaseManager() (*DatabaseManager, error) {
	db, err := connectDatabase()
	if err != nil {
		return nil, err
	}
	dm := &DatabaseManager{
		db:            db,
		healthChecker: NewHealthChecker(db, 30*time.Second),
	}

	// Start health checking
	dm.healthChecker.Start()

	return dm, nil
}

// GetDB returns the underlying database connection
func (dm *DatabaseManager) GetDB() *sql.DB {
	return dm.db
}

// Close closes the database connection and stops health checking
func (dm *DatabaseManager) Close() error {
	if dm.healthChecker != nil {
		dm.healthChecker.Stop()
	}
	if dm.db != nil {
		return dm.db.Close()
	}
	return nil
}

// QueryWithHealthCheck executes a query with connection health verification
func (dm *DatabaseManager) QueryWithHealthCheck(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if err := dm.healthChecker.EnsureConnection(ctx); err != nil {
		return nil, err
	}

	return dm.db.QueryContext(ctx, query, args...)
}

// QueryRowWithHealthCheck executes a query that returns a single row with health check
func (dm *DatabaseManager) QueryRowWithHealthCheck(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if err := dm.healthChecker.EnsureConnection(ctx); err != nil {
		// Return a row that will fail on scan
		return dm.db.QueryRowContext(context.Background(), "SELECT NULL WHERE FALSE")
	}

	return dm.db.QueryRowContext(ctx, query, args...)
}

// ExecWithHealthCheck executes a statement with connection health verification
func (dm *DatabaseManager) ExecWithHealthCheck(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if err := dm.healthChecker.EnsureConnection(ctx); err != nil {
		return nil, err
	}

	return dm.db.ExecContext(ctx, query, args...)
}

// IsConnectionHealthy returns the current health status
func (dm *DatabaseManager) IsConnectionHealthy() bool {
	return dm.healthChecker.IsHealthy()
}

// Init initializes the database with migrations
func (dm *DatabaseManager) Init() error {
	log.Println("Running database migrations...")

	runner, err := NewMigrationsRunner(dm.db)
	if err != nil {
		return fmt.Errorf("failed to create migration runner: %w", err)
	}

	if err := runner.Run(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("✓ Database initialization completed successfully")
	return nil
}

// LoadStations loads all stations from the database
func (dm *DatabaseManager) LoadStations() ([]models.StationData, error) {
	query := `
        SELECT id, pass_key, station_type, model, freq, mode, service_name, config, updated_at
        FROM stations
        ORDER BY created_at DESC
    `

	rows, err := dm.QueryWithHealthCheck(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stations []models.StationData
	for rows.Next() {
		var station models.StationData
		var configJSON []byte

		err := rows.Scan(
			&station.ID,
			&station.PassKey,
			&station.StationType,
			&station.Model,
			&station.Freq,
			&station.Mode,
			&station.ServiceName,
			&configJSON,
			&station.UpdatedAt,
		)
		if err != nil {
			log.Printf("Failed to scan station: %v", err)
			continue
		}

		// ParseWeatherData config JSON
		station.Config = make(map[string]interface{})
		if err := json.Unmarshal(configJSON, &station.Config); err != nil {
			log.Printf("Failed to parse config for station %s: %v", station.PassKey, err)
		}

		stations = append(stations, station)
	}

	return stations, rows.Err()
}

// StoreWeatherData stores weather data for a station
func (dm *DatabaseManager) StoreWeatherData(readings map[uuid.UUID]models.SensorReading) error {
	for sensorID, reading := range readings {
		err := dm.StoreSensorReading(sensorID, reading.Value, reading.DateUTC)
		if err != nil {
			log.Printf("Failed to store sensor reading for sensor %s: %v", sensorID, err)
			return err
		}
	}

	return nil
}

// EnsureStation checks if a station exists and creates it if not
func (dm *DatabaseManager) EnsureStation(data *models.StationData) (uuid.UUID, error) {
	query := `
        INSERT INTO stations (pass_key, station_type, model, mode, service_name)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (pass_key) DO UPDATE
        SET station_type = $2, model = $3, updated_at = CURRENT_TIMESTAMP
        RETURNING id
    `

	var stationIDString string
	err := dm.QueryRowWithHealthCheck(context.Background(), query,
		data.PassKey,
		data.StationType,
		data.Model,
		data.Mode,
		data.ServiceName,
	).Scan(&stationIDString)

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to ensure station: %w", err)
	}

	stationID, err := uuid.Parse(stationIDString)

	return stationID, err
}
func (dm *DatabaseManager) EnsureSensorsByRemoteId(stationID uuid.UUID, sensors map[string]models.Sensor) (map[string]models.Sensor, error) {
	for remoteID, sensor := range sensors {
		var existingSensorID string
		checkQuery := `
            SELECT id FROM sensors 
            WHERE station_id = $1 AND remote_id = $2
        `

		err := dm.QueryRowWithHealthCheck(context.Background(), checkQuery, stationID, remoteID).Scan(&existingSensorID)

		if errors.Is(err, sql.ErrNoRows) {
			// Sensor doesn't exist, create it
			insertQuery := `
                INSERT INTO sensors (
                    station_id, sensor_type, location, name, model, 
                    battery_level, signal_strength, enabled, remote_id
                )
                VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
                RETURNING id
            `

			var newSensorID uuid.UUID
			err = dm.QueryRowWithHealthCheck(context.Background(), insertQuery,
				stationID,
				sensor.SensorType,
				sensor.Location,
				sensor.Name,
				sensor.Model,
				sensor.BatteryLevel,
				sensor.SignalStrength,
				sensor.Enabled,
				remoteID,
			).Scan(&newSensorID)

			if err != nil {
				log.Printf("Failed to create sensor with remote_id %s: %v", remoteID, err)
				return sensors, fmt.Errorf("failed to create sensor: %w", err)
			}

			// Update the sensor in the map with the new ID
			sensor.ID = newSensorID
			sensors[remoteID] = sensor

			log.Printf("Created new sensor with remote_id %s (ID: %s)", remoteID, newSensorID)
			continue // Skip to next sensor since we just created it
		} else if err != nil {
			log.Printf("Failed to check sensor existence for remote_id %s: %v", remoteID, err)
			return sensors, fmt.Errorf("failed to check sensor: %w", err)
		}

		// Sensor exists, parse the ID and update the sensor in the map
		parsedID, err := uuid.Parse(existingSensorID)
		if err != nil {
			log.Printf("Failed to parse sensor ID for remote_id %s: %v", remoteID, err)
			return sensors, fmt.Errorf("failed to parse sensor ID: %w", err)
		}

		sensor.ID = parsedID
		sensors[remoteID] = sensor

		// Sensor exists, update it
		updateQuery := `
            UPDATE sensors 
            SET sensor_type = $1, location = $2, name = $3, model = $4,
                battery_level = $5, signal_strength = $6, enabled = $7
            WHERE id = $8
        `

		_, err = dm.ExecWithHealthCheck(context.Background(), updateQuery,
			sensor.SensorType,
			sensor.Location,
			sensor.Name,
			sensor.Model,
			sensor.BatteryLevel,
			sensor.SignalStrength,
			sensor.Enabled,
			parsedID,
		)

		if err != nil {
			log.Printf("Failed to update sensor with remote_id %s: %v", remoteID, err)
			return sensors, fmt.Errorf("failed to update sensor: %w", err)
		}
	}

	return sensors, nil
}

// GetStationList retrieves a list of all stations
func (dm *DatabaseManager) GetStationList() ([]models.StationListItem, error) {
	query := `
            SELECT DISTINCT 
                s.id, 
                s.pass_key, 
                s.station_type, 
                s.model,
                MAX(wd.date_utc) as last_update
            FROM stations s
            LEFT JOIN weather_data wd ON s.id = wd.station_id
            GROUP BY s.id, s.pass_key, s.station_type, s.model
            ORDER BY last_update DESC
        `

	var stations []models.StationListItem

	rows, err := dm.QueryWithHealthCheck(context.Background(), query)
	if err != nil {
		return stations, err
	}
	defer rows.Close()

	for rows.Next() {
		var s models.StationListItem
		err := rows.Scan(
			&s.ID,
			&s.PassKey,
			&s.StationType,
			&s.Model,
			&s.LastUpdate,
		)
		if err != nil {
			log.Printf("❌ Failed to scan s: %v", err)
			continue
		}

		stations = append(stations, s)
	}

	return stations, nil
}

// GetStation retrieves detailed information about a specific station
func (dm *DatabaseManager) GetStation(stationID uuid.UUID) (models.StationDetail, error) {
	query := `
            SELECT 
                s.id, 
                s.pass_key, 
                s.station_type, 
                s.model,
                COUNT(sr.id) as total_readings,
                MIN(sr.date_utc) as first_reading,
                MAX(sr.date_utc) as last_reading
            FROM stations s
            LEFT JOIN sensors sens ON s.id = sens.station_id
            LEFT JOIN sensor_readings sr ON sens.id = sr.sensor_id
            WHERE s.id = $1
            GROUP BY s.id, s.pass_key, s.station_type, s.model
        `

	var station models.StationDetail
	err := dm.QueryRowWithHealthCheck(context.Background(), query, stationID).Scan(
		&station.ID,
		&station.PassKey,
		&station.StationType,
		&station.Model,
		&station.TotalReadings,
		&station.FirstReading,
		&station.LastReading,
	)

	return station, err
}

// GetCurrentWeather retrieves the latest weather data for a station
func (dm *DatabaseManager) GetCurrentWeather(stationID uuid.UUID) (models.WeatherData, error) {
	query := `
            SELECT 
                s.id,
                st.name,
                st.category,
                st.unit,
                sr.value,
                sr.date_utc
            FROM sensors s
            JOIN sensor_types st ON s.sensor_type_id = st.id
            LEFT JOIN LATERAL (
                SELECT value, date_utc
                FROM sensor_readings
                WHERE sensor_id = s.id
                ORDER BY date_utc DESC
                LIMIT 1
            ) sr ON TRUE
            WHERE s.station_id = $1
            ORDER BY st.category, s.location
        `

	rows, err := dm.QueryWithHealthCheck(context.Background(), query, stationID)
	if err != nil {
		return models.WeatherData{}, err
	}
	defer rows.Close()

	weatherData := models.WeatherData{
		DateUTC: time.Now().UTC(),
	}

	for rows.Next() {
		var sensorID uuid.UUID
		var sensorName string
		var category string
		var unit string
		var value *float64
		var dateUTC *time.Time

		err := rows.Scan(&sensorID, &sensorName, &category, &unit, &value, &dateUTC)
		if err != nil {
			log.Printf("Failed to scan sensor reading: %v", err)
			continue
		}

		if value == nil {
			continue
		}

		if dateUTC != nil {
			weatherData.DateUTC = *dateUTC
		}

		mapSensorToWeatherData(&weatherData, sensorName, *value)
	}

	return weatherData, rows.Err()
}

// GetWeatherHistory retrieves weather data history for a station
func (dm *DatabaseManager) GetWeatherHistory(stationID uuid.UUID, startTime string, endTime string, limit int) ([]models.WeatherData, error) {
	query := `
            SELECT 
                s.id,
                st.name,
                st.category,
                st.unit,
                sr.value,
                sr.date_utc
            FROM sensors s
            JOIN sensor_types st ON s.sensor_type_id = st.id
            LEFT JOIN sensor_readings sr ON s.id = sr.sensor_id
            WHERE s.station_id = $1
        `

	args := []interface{}{stationID}
	argCount := 2

	if startTime != "" {
		query += " AND sr.date_utc >= $" + strconv.Itoa(argCount)
		args = append(args, startTime)
		argCount++
	}
	if endTime != "" {
		query += " AND sr.date_utc <= $" + strconv.Itoa(argCount)
		args = append(args, endTime)
		argCount++
	}

	query += " ORDER BY sr.date_utc DESC LIMIT $" + strconv.Itoa(argCount)
	args = append(args, limit)

	rows, err := dm.QueryWithHealthCheck(context.Background(), query, args...)
	if err != nil {
		log.Printf("Failed to query weather history: %v", err)
		return nil, errors.New("Failed to query weather history")
	}
	defer rows.Close()

	// Group readings by date_utc to build WeatherData objects
	weatherDataMap := make(map[time.Time]*models.WeatherData)

	for rows.Next() {
		var sensorID uuid.UUID
		var sensorName string
		var category string
		var unit string
		var value *float64
		var dateUTC *time.Time

		err := rows.Scan(&sensorID, &sensorName, &category, &unit, &value, &dateUTC)
		if err != nil {
			log.Printf("Failed to scan sensor reading: %v", err)
			continue
		}

		if value == nil || dateUTC == nil {
			continue
		}

		// Get or create WeatherData for this timestamp
		if _, exists := weatherDataMap[*dateUTC]; !exists {
			weatherDataMap[*dateUTC] = &models.WeatherData{
				DateUTC: *dateUTC,
			}
		}

		wd := weatherDataMap[*dateUTC]

		mapSensorToWeatherData(wd, sensorName, *value)
	}

	// Convert map to sorted slice
	var weatherDataList []models.WeatherData
	for _, wd := range weatherDataMap {
		weatherDataList = append(weatherDataList, *wd)
	}

	// Sort by date descending
	sort.Slice(weatherDataList, func(i, j int) bool {
		return weatherDataList[i].DateUTC.After(weatherDataList[j].DateUTC)
	})

	return weatherDataList, rows.Err()
}

// GetStationConfig retrieves the configuration for a specific station
func (dm *DatabaseManager) GetStationConfig(id uuid.UUID) (map[string]interface{}, error) {
	var config map[string]interface{}

	query := `SELECT config FROM stations WHERE id = $1`
	var configJSON string
	err := dm.QueryRowWithHealthCheck(context.Background(), query, id.String()).Scan(&configJSON)
	if err != nil {
		err = errors.New("Station not found: " + err.Error())
		return config, err
	}

	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		err = errors.New("Failed to parse station config: " + err.Error())
		return config, err
	}

	return config, nil
}

// SetStationConfig updates the configuration for a specific station
func (dm *DatabaseManager) SetStationConfig(id uuid.UUID, config map[string]interface{}) error {
	updatedConfigJSON, err := json.Marshal(config)
	if err != nil {
		log.Printf("Failed to marshal config: %v", err)
		return errors.New("failed to encode config to JSON")
	}

	updateQuery := `UPDATE stations SET config = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	_, err = dm.ExecWithHealthCheck(context.Background(), updateQuery, updatedConfigJSON, id)
	if err != nil {
		log.Printf("Failed to update station config: %v", err)
		return errors.New("failed to save access token")
	}

	return nil
}

// SaveStation saves a station to the database
func (dm *DatabaseManager) SaveStation(station *models.StationData) error {
	configJSON, err := json.Marshal(station.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	query := `
        INSERT INTO stations (id, pass_key, station_type, model, freq, mode, service_name, config)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        ON CONFLICT (pass_key) DO UPDATE
        SET station_type = $3, model = $4, freq = $5, mode = $6, service_name = $7, config = $8, updated_at = CURRENT_TIMESTAMP
        RETURNING id
    `

	err = dm.QueryRowWithHealthCheck(context.Background(), query,
		station.ID,
		station.PassKey,
		station.StationType,
		station.Model,
		station.Freq,
		station.Mode,
		station.ServiceName,
		configJSON,
	).Scan(&station.ID)

	return err
}

// GetStationIDByConfigValue retrieves a station ID by a config key-value pair
func (dm *DatabaseManager) GetStationIDByConfigValue(key string, value string) (uuid.UUID, error) {
	query := `SELECT id FROM stations WHERE config->>'$1' = $2`

	var stationID uuid.UUID
	err := dm.QueryRowWithHealthCheck(context.Background(), query, value, key).Scan(&stationID)
	if err != nil {
		return stationID, fmt.Errorf("failed to query station ID: %w", err)
	}

	return stationID, nil
}

// GetStationsData retrieves detailed information about a specific station for CLI output
func (dm *DatabaseManager) GetStationsData() ([]models.StationData, error) {
	var stations []models.StationData

	query := `
        SELECT id, pass_key, station_type, model, mode, service_name, freq, updated_at
        FROM stations
        ORDER BY created_at DESC
    `

	rows, err := dm.QueryWithHealthCheck(context.Background(), query)
	if err != nil {
		return stations, fmt.Errorf("failed to query stations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var station models.StationData
		err := rows.Scan(
			&station.ID,
			&station.PassKey,
			&station.StationType,
			&station.Model,
			&station.Mode,
			&station.ServiceName,
			&station.Freq,
			&station.UpdatedAt,
		)
		if err != nil {
			return stations, errors.New("failed to scan station: " + err.Error())
		}

		stations = append(stations, station)
	}

	return stations, rows.Err()
}

// DeleteStation deletes a station and its associated weather data
func (dm *DatabaseManager) DeleteStation(stationID uuid.UUID) error {
	// Delete weather data first (foreign key constraint)
	deleteWeatherQuery := `DELETE FROM weather_data WHERE station_id = $1`
	_, err := dm.ExecWithHealthCheck(context.Background(), deleteWeatherQuery, stationID.String())
	if err != nil {
		return fmt.Errorf("failed to delete weather data: %w", err)
	}

	// Delete station
	deleteStationQuery := `DELETE FROM stations WHERE id = $1`
	_, err = dm.ExecWithHealthCheck(context.Background(), deleteStationQuery, stationID.String())
	if err != nil {
		return fmt.Errorf("failed to delete station: %w", err)
	}

	return nil
}

// connectDatabase establishes a connection to the database
func connectDatabase() (*sql.DB, error) {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "weather_user")
	password := getEnv("DB_PASSWORD", "weather_pass")
	dbName := getEnv("DB_NAME", "weather_db")
	sslmode := getEnv("DB_SSLMODE", "disable")

	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbName, sslmode,
	)

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

	return db, nil
}

// mapSensorToWeatherData maps a sensor reading to the appropriate WeatherData field
func mapSensorToWeatherData(wd *models.WeatherData, sensorName string, value float64) {
	switch sensorName {
	case models.SensorTypeTemperature:
		wd.TempInC = value
		wd.TempInF = (value * 9 / 5) + 32
	case models.SensorTypeHumidity:
		wd.HumidityIn = int(value)
	case models.SensorTypePressureRelative:
		wd.BaromRelHPa = value
		wd.BaromRelIn = value / 33.8639
	case models.SensorTypePressureAbsolute:
		wd.BaromAbsHPa = value
		wd.BaromAbsIn = value / 33.8639
	case models.SensorTypeTemperatureOutdoor:
		wd.TempOutC = value
		wd.TempOutF = (value * 9 / 5) + 32
	case models.SensorTypeHumidityOutdoor:
		wd.HumidityOut = int(value)
	case models.SensorTypeWindSpeed:
		wd.WindSpeedMS = value
		wd.WindSpeedKmH = value * 3.6
		wd.WindSpeedMPH = value * 2.237
	case models.SensorTypeWindDirection:
		wd.WindDir = int(value)
	case models.SensorTypeWindGust:
		wd.WindGustMS = value
		wd.WindGustKmH = value * 3.6
		wd.WindGustMPH = value * 2.237
	case models.SensorTypeWindGustMaxDaily:
		wd.MaxDailyGustMS = value
		wd.MaxDailyGustKmH = value * 3.6
		wd.MaxDailyGustMPH = value * 2.237
	case models.SensorTypeRainfallRate:
		wd.RainRateMmH = value
		wd.RainRateIn = value / 25.4
	case models.SensorTypeRainfallEvent:
		wd.EventRainMm = value
		wd.EventRainIn = value / 25.4
	case models.SensorTypeRainfallHourly:
		wd.HourlyRainMm = value
		wd.HourlyRainIn = value / 25.4
	case models.SensorTypeRainfallDaily:
		wd.DailyRainMm = value
		wd.DailyRainIn = value / 25.4
	case models.SensorTypeRainfallWeekly:
		wd.WeeklyRainMm = value
		wd.WeeklyRainIn = value / 25.4
	case models.SensorTypeRainfallMonthly:
		wd.MonthlyRainMm = value
		wd.MonthlyRainIn = value / 25.4
	case models.SensorTypeRainfallYearly:
		wd.YearlyRainMm = value
		wd.YearlyRainIn = value / 25.4
	case models.SensorTypeRainfallTotal:
		wd.TotalRainMm = value
		wd.TotalRainIn = value / 25.4
	case models.SensorTypeSolarRadiation:
		wd.SolarRadiation = value
	case models.SensorTypeUVIndex:
		wd.UV = int(value)
	case models.SensorTypeVPD:
		wd.VPD = value
	case models.SensorTypeBattery:
		wd.WH65Batt = int(value)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

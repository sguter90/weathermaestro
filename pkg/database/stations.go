package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/sguter90/weathermaestro/pkg/models"
)

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
		var freq sql.NullString

		err := rows.Scan(
			&station.ID,
			&station.PassKey,
			&station.StationType,
			&station.Model,
			&freq,
			&station.Mode,
			&station.ServiceName,
			&configJSON,
			&station.UpdatedAt,
		)
		if err != nil {
			log.Printf("Failed to scan station: %v", err)
			continue
		}

		if freq.Valid {
			station.Freq = freq.String
		} else {
			station.Freq = ""
		}

		// Parse config JSON
		station.Config = make(map[string]interface{})
		if err := json.Unmarshal(configJSON, &station.Config); err != nil {
			log.Printf("Failed to parse config for station %s: %v", station.PassKey, err)
		}

		stations = append(stations, station)
	}

	return stations, rows.Err()
}

// LoadStation loads specific station from the database
func (dm *DatabaseManager) LoadStation(stationID uuid.UUID) (models.StationData, error) {
	query := `
		SELECT id, pass_key, station_type, model, freq, mode, service_name, config, updated_at
        FROM stations
        WHERE id = $1
    `

	var station models.StationData
	var configJSON []byte
	var freq sql.NullString
	err := dm.QueryRowWithHealthCheck(context.Background(), query, stationID).Scan(
		&station.ID,
		&station.PassKey,
		&station.StationType,
		&station.Model,
		&freq,
		&station.Mode,
		&station.ServiceName,
		&configJSON,
		&station.UpdatedAt,
	)

	if err != nil {
		return station, fmt.Errorf("failed to scan station %s", err.Error())
	}

	if freq.Valid {
		station.Freq = freq.String
	} else {
		station.Freq = ""
	}

	// ParseWeatherData config JSON
	station.Config = make(map[string]interface{})
	if err := json.Unmarshal(configJSON, &station.Config); err != nil {
		log.Printf("Failed to parse config for station %s: %v", station.PassKey, err)
	}

	return station, err
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

// GetStationList retrieves a list of all stations
func (dm *DatabaseManager) GetStationList() ([]models.StationDetail, error) {
	query := `
            SELECT DISTINCT 
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
            GROUP BY s.id
        `

	var stations []models.StationDetail

	rows, err := dm.QueryWithHealthCheck(context.Background(), query)
	if err != nil {
		return stations, err
	}
	defer rows.Close()

	for rows.Next() {
		var s models.StationDetail
		var firstReading, lastReading sql.NullTime
		err := rows.Scan(
			&s.ID,
			&s.PassKey,
			&s.StationType,
			&s.Model,
			&s.TotalReadings,
			&firstReading,
			&lastReading,
		)
		if err != nil {
			log.Printf("❌ Failed to scan s: %v", err)
			continue
		}

		// Convert sql.NullTime to *time.Time
		if firstReading.Valid {
			s.FirstReading = firstReading.Time
		}
		if lastReading.Valid {
			s.LastReading = lastReading.Time
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
            GROUP BY s.id
        `

	var station models.StationDetail
	var firstReading, lastReading sql.NullTime

	err := dm.QueryRowWithHealthCheck(context.Background(), query, stationID).Scan(
		&station.ID,
		&station.PassKey,
		&station.StationType,
		&station.Model,
		&station.TotalReadings,
		&firstReading,
		&lastReading,
	)

	// Convert sql.NullTime to *time.Time
	if firstReading.Valid {
		station.FirstReading = firstReading.Time
	}
	if lastReading.Valid {
		station.LastReading = lastReading.Time
	}

	return station, err
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
	query := `SELECT id FROM stations WHERE config->>$1 = $2`

	var stationID uuid.UUID
	err := dm.QueryRowWithHealthCheck(context.Background(), query, key, value).Scan(&stationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return stationID, fmt.Errorf("no station found with config %s=%s", key, value)
		}
		return stationID, fmt.Errorf("failed to query station ID: %w", err)
	}

	return stationID, nil
}

// GetStationsData retrieves detailed information about a specific station for CLI output
func (dm *DatabaseManager) GetStationsData() ([]models.StationData, error) {
	stations := []models.StationData{}

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
		var freq sql.NullString

		err := rows.Scan(
			&station.ID,
			&station.PassKey,
			&station.StationType,
			&station.Model,
			&station.Mode,
			&station.ServiceName,
			&freq,
			&station.UpdatedAt,
		)
		if err != nil {
			return stations, errors.New("failed to scan station: " + err.Error())
		}

		if freq.Valid {
			station.Freq = freq.String
		} else {
			station.Freq = ""
		}

		stations = append(stations, station)
	}

	return stations, rows.Err()
}

// DeleteStation deletes a station and its associated weather data
func (dm *DatabaseManager) DeleteStation(stationID uuid.UUID) error {
	// Delete station
	deleteStationQuery := `DELETE FROM stations WHERE id = $1`
	_, err := dm.ExecWithHealthCheck(context.Background(), deleteStationQuery, stationID.String())
	if err != nil {
		return fmt.Errorf("failed to delete station: %w", err)
	}

	return nil
}

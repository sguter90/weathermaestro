package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/sguter90/weathermaestro/pkg/models"
)

// CreateSensor creates a new sensor for a station
func (dm *DatabaseManager) CreateSensor(sensor *models.Sensor) error {
	query := `
        INSERT INTO sensors (station_id, sensor_type, location, name, model, battery_level, signal_strength, enabled, remote_id)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id, created_at, updated_at
    `

	var remoteID sql.NullString
	if sensor.RemoteID != "" {
		remoteID = sql.NullString{String: sensor.RemoteID, Valid: true}
	}

	err := dm.QueryRowWithHealthCheck(context.Background(), query,
		sensor.StationID,
		sensor.SensorType,
		sensor.Location,
		sensor.Name,
		sensor.Model,
		sensor.BatteryLevel,
		sensor.SignalStrength,
		sensor.Enabled,
		remoteID,
	).Scan(&sensor.ID, &sensor.CreatedAt, &sensor.UpdatedAt)

	return err
}

// GetSensor retrieves a single sensor by ID
func (dm *DatabaseManager) GetSensor(sensorID uuid.UUID, includeLatest bool) (*models.SensorWithLatestReading, error) {
	var query string

	if includeLatest {
		query = `
            SELECT 
                s.id, s.station_id, s.sensor_type, s.location, s.name, s.model,
                s.battery_level, s.signal_strength, s.enabled, s.created_at, s.updated_at,
                sr.id, sr.sensor_id, sr.value, sr.date_utc
            FROM sensors s
            LEFT JOIN LATERAL (
                SELECT id, sensor_id, value, date_utc
                FROM sensor_readings
                WHERE sensor_id = s.id
                ORDER BY date_utc DESC
                LIMIT 1
            ) sr ON TRUE
            WHERE s.id = $1
        `
	} else {
		query = `
            SELECT 
                s.id, s.station_id, s.sensor_type, s.location, s.name, s.model,
                s.battery_level, s.signal_strength, s.enabled, s.created_at, s.updated_at,
                NULL, NULL, NULL, NULL
            FROM sensors s
            WHERE s.id = $1
        `
	}

	var swr models.SensorWithLatestReading
	var readingID *uuid.UUID
	var readingSensorID *uuid.UUID
	var readingValue *float64
	var readingDateUTC *time.Time

	err := dm.QueryRowWithHealthCheck(context.Background(), query, sensorID).Scan(
		&swr.Sensor.ID, &swr.Sensor.StationID, &swr.Sensor.SensorType,
		&swr.Sensor.Location, &swr.Sensor.Name, &swr.Sensor.Model,
		&swr.Sensor.BatteryLevel, &swr.Sensor.SignalStrength, &swr.Sensor.Enabled,
		&swr.Sensor.CreatedAt, &swr.Sensor.UpdatedAt,
		&readingID, &readingSensorID, &readingValue, &readingDateUTC,
	)
	if err != nil {
		return nil, err
	}

	// Construct latest reading if it exists
	if readingID != nil {
		swr.LatestReading = &models.SensorReading{
			ID:       *readingID,
			SensorID: *readingSensorID,
			Value:    *readingValue,
			DateUTC:  *readingDateUTC,
		}
	}

	return &swr, nil
}

// GetSensors retrieves sensors with flexible filtering
func (dm *DatabaseManager) GetSensors(params models.SensorQueryParams) ([]models.SensorWithLatestReading, error) {
	var query string
	args := []interface{}{}
	argCount := 1

	if params.IncludeLatest {
		query = `
            SELECT 
                s.id, s.station_id, s.sensor_type, s.location, s.name, s.model,
                s.battery_level, s.signal_strength, s.enabled, s.created_at, s.updated_at,
                sr.id, sr.sensor_id, sr.value, sr.date_utc
            FROM sensors s
            LEFT JOIN LATERAL (
                SELECT id, sensor_id, value, date_utc
                FROM sensor_readings
                WHERE sensor_id = s.id
                ORDER BY date_utc DESC
                LIMIT 1
            ) sr ON TRUE
            WHERE 1=1
        `
	} else {
		query = `
            SELECT 
                s.id, s.station_id, s.sensor_type, s.location, s.name, s.model,
                s.battery_level, s.signal_strength, s.enabled, s.created_at, s.updated_at,
                NULL, NULL, NULL, NULL
            FROM sensors s
            WHERE 1=1
        `
	}

	// Build WHERE clause dynamically
	if params.StationID != nil {
		query += " AND s.station_id = $" + string(rune(argCount+'0'))
		args = append(args, *params.StationID)
		argCount++
	}

	if params.SensorType != "" {
		query += " AND s.sensor_type = $" + string(rune(argCount+'0'))
		args = append(args, params.SensorType)
		argCount++
	}

	if params.Location != "" {
		query += " AND s.location = $" + string(rune(argCount+'0'))
		args = append(args, params.Location)
		argCount++
	}

	if params.Enabled != nil {
		query += " AND s.enabled = $" + string(rune(argCount+'0'))
		args = append(args, *params.Enabled)
		argCount++
	}

	query += " ORDER BY s.location, s.sensor_type, s.created_at"

	rows, err := dm.QueryWithHealthCheck(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sensors []models.SensorWithLatestReading
	for rows.Next() {
		var swr models.SensorWithLatestReading
		var readingID *uuid.UUID
		var readingSensorID *uuid.UUID
		var readingValue *float64
		var readingDateUTC *time.Time

		err := rows.Scan(
			&swr.Sensor.ID, &swr.Sensor.StationID, &swr.Sensor.SensorType,
			&swr.Sensor.Location, &swr.Sensor.Name, &swr.Sensor.Model,
			&swr.Sensor.BatteryLevel, &swr.Sensor.SignalStrength, &swr.Sensor.Enabled,
			&swr.Sensor.CreatedAt, &swr.Sensor.UpdatedAt,
			&readingID, &readingSensorID, &readingValue, &readingDateUTC,
		)
		if err != nil {
			log.Printf("Failed to scan sensor: %v", err)
			continue
		}

		// Construct latest reading if it exists
		if readingID != nil {
			swr.LatestReading = &models.SensorReading{
				ID:       *readingID,
				SensorID: *readingSensorID,
				Value:    *readingValue,
				DateUTC:  *readingDateUTC,
			}
		}

		sensors = append(sensors, swr)
	}

	return sensors, rows.Err()
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

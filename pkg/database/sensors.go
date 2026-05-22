package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sguter90/weathermaestro/pkg/models"
)

// latestReadingsForSensors fetches the most recent reading per sensor from ClickHouse
// for the given sensor IDs. Sensors with no readings are absent from the result map.
func (dm *DatabaseManager) latestReadingsForSensors(ctx context.Context, sensorIDs []uuid.UUID) (map[uuid.UUID]*models.SensorReading, error) {
	result := map[uuid.UUID]*models.SensorReading{}
	if len(sensorIDs) == 0 {
		return result, nil
	}

	const query = `
		SELECT
			sensor_id,
			argMax(id, date_utc)    AS latest_id,
			argMax(value, date_utc) AS latest_value,
			max(date_utc)           AS latest_date
		FROM sensor_readings
		WHERE sensor_id IN ?
		GROUP BY sensor_id
	`
	rows, err := dm.ch.Conn().Query(ctx, query, sensorIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest readings: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			sensorID    uuid.UUID
			latestID    uuid.UUID
			latestValue float64
			latestDate  time.Time
		)
		if err := rows.Scan(&sensorID, &latestID, &latestValue, &latestDate); err != nil {
			log.Printf("Failed to scan latest reading: %v", err)
			continue
		}
		result[sensorID] = &models.SensorReading{
			ID:       latestID,
			SensorID: sensorID,
			Value:    latestValue,
			DateUTC:  latestDate,
		}
	}
	return result, rows.Err()
}

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

// GetSensor retrieves a single sensor by ID. When includeLatest is true the most recent
// reading for the sensor is fetched from ClickHouse and attached.
func (dm *DatabaseManager) GetSensor(sensorID uuid.UUID, includeLatest bool) (*models.SensorWithLatestReading, error) {
	const query = `
		SELECT id, station_id, sensor_type, location, name, model,
		       battery_level, signal_strength, enabled, created_at, updated_at
		FROM sensors
		WHERE id = $1
	`

	var swr models.SensorWithLatestReading
	err := dm.QueryRowWithHealthCheck(context.Background(), query, sensorID).Scan(
		&swr.Sensor.ID, &swr.Sensor.StationID, &swr.Sensor.SensorType,
		&swr.Sensor.Location, &swr.Sensor.Name, &swr.Sensor.Model,
		&swr.Sensor.BatteryLevel, &swr.Sensor.SignalStrength, &swr.Sensor.Enabled,
		&swr.Sensor.CreatedAt, &swr.Sensor.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if includeLatest {
		latest, err := dm.latestReadingsForSensors(context.Background(), []uuid.UUID{sensorID})
		if err != nil {
			return nil, err
		}
		if r, ok := latest[sensorID]; ok {
			swr.LatestReading = r
		}
	}

	return &swr, nil
}

// GetSensors retrieves sensors with flexible filtering. When IncludeLatest is true
// the most recent reading per sensor is fetched in a single batch query against ClickHouse.
func (dm *DatabaseManager) GetSensors(params models.SensorQueryParams) ([]models.SensorWithLatestReading, error) {
	conditions := []string{}
	args := []interface{}{}
	idx := 1

	if params.StationID != nil {
		conditions = append(conditions, fmt.Sprintf("station_id = $%d", idx))
		args = append(args, *params.StationID)
		idx++
	}
	if params.SensorType != "" {
		conditions = append(conditions, fmt.Sprintf("sensor_type = $%d", idx))
		args = append(args, params.SensorType)
		idx++
	}
	if params.Location != "" {
		conditions = append(conditions, fmt.Sprintf("location = $%d", idx))
		args = append(args, params.Location)
		idx++
	}
	if params.Enabled != nil {
		conditions = append(conditions, fmt.Sprintf("enabled = $%d", idx))
		args = append(args, *params.Enabled)
		idx++
	}

	query := `
		SELECT id, station_id, sensor_type, location, name, model,
		       battery_level, signal_strength, enabled, created_at, updated_at
		FROM sensors`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY location, sensor_type, created_at"

	rows, err := dm.QueryWithHealthCheck(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sensors []models.SensorWithLatestReading
	var sensorIDs []uuid.UUID
	for rows.Next() {
		var swr models.SensorWithLatestReading
		err := rows.Scan(
			&swr.Sensor.ID, &swr.Sensor.StationID, &swr.Sensor.SensorType,
			&swr.Sensor.Location, &swr.Sensor.Name, &swr.Sensor.Model,
			&swr.Sensor.BatteryLevel, &swr.Sensor.SignalStrength, &swr.Sensor.Enabled,
			&swr.Sensor.CreatedAt, &swr.Sensor.UpdatedAt,
		)
		if err != nil {
			log.Printf("Failed to scan sensor: %v", err)
			continue
		}
		sensors = append(sensors, swr)
		sensorIDs = append(sensorIDs, swr.Sensor.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if params.IncludeLatest && len(sensorIDs) > 0 {
		latest, err := dm.latestReadingsForSensors(context.Background(), sensorIDs)
		if err != nil {
			return nil, err
		}
		for i := range sensors {
			if r, ok := latest[sensors[i].Sensor.ID]; ok {
				sensors[i].LatestReading = r
			}
		}
	}

	return sensors, nil
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

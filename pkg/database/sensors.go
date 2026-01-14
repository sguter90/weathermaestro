package database

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/sguter90/weathermaestro/pkg/models"
)

// CreateSensor creates a new sensor for a station
func (dm *DatabaseManager) CreateSensor(sensor *models.Sensor) error {
	query := `
        INSERT INTO sensors (station_id, sensor_type_id, location, name, model, battery_level, signal_strength, enabled)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        RETURNING id, created_at, updated_at
    `

	err := dm.QueryRowWithHealthCheck(context.Background(), query,
		sensor.StationID,
		sensor.SensorType,
		sensor.Location,
		sensor.Name,
		sensor.Model,
		sensor.BatteryLevel,
		sensor.SignalStrength,
		sensor.Enabled,
	).Scan(&sensor.ID, &sensor.CreatedAt, &sensor.UpdatedAt)

	return err
}

// GetSensorsByStation retrieves all sensors for a station
func (dm *DatabaseManager) GetSensorsByStation(stationID uuid.UUID) ([]models.SensorWithLatestReading, error) {
	query := `
        SELECT 
            s.id, s.station_id, s.sensor_type, s.location, s.name, s.model,
            s.battery_level, s.signal_strength, s.enabled, s.created_at, s.updated_at,
            sr.id, sr.sensor_id, sr.value, sr.date_utc, sr.created_at
        FROM sensors s
        LEFT JOIN LATERAL (
            SELECT id, sensor_id, value, date_utc, created_at
            FROM sensor_readings
            WHERE sensor_id = s.id
            ORDER BY date_utc DESC
            LIMIT 1
        ) sr ON TRUE
        WHERE s.station_id = $1
        ORDER BY s.location, st.category, s.created_at
    `

	rows, err := dm.QueryWithHealthCheck(context.Background(), query, stationID)
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
		var readingCreatedAt *time.Time

		err := rows.Scan(
			&swr.Sensor.ID, &swr.Sensor.StationID, &swr.Sensor.SensorType,
			&swr.Sensor.Location, &swr.Sensor.Name, &swr.Sensor.Model,
			&swr.Sensor.BatteryLevel, &swr.Sensor.SignalStrength, &swr.Sensor.Enabled,
			&swr.Sensor.CreatedAt, &swr.Sensor.UpdatedAt,
			&readingID, &readingSensorID, &readingValue, &readingDateUTC, &readingCreatedAt,
		)
		if err != nil {
			log.Printf("Failed to scan sensor: %v", err)
			continue
		}

		// Construct latest reading if it exists
		if readingID != nil {
			swr.LatestReading = &models.SensorReading{
				ID:        *readingID,
				SensorID:  *readingSensorID,
				Value:     *readingValue,
				DateUTC:   *readingDateUTC,
				CreatedAt: *readingCreatedAt,
			}
		}

		sensors = append(sensors, swr)
	}

	return sensors, rows.Err()
}

// StoreSensorReading stores a single sensor reading
func (dm *DatabaseManager) StoreSensorReading(sensorID uuid.UUID, value float64, dateUTC time.Time) error {
	query := `
        INSERT INTO sensor_readings (sensor_id, value, date_utc)
        VALUES ($1, $2, $3)
    `

	_, err := dm.ExecWithHealthCheck(context.Background(), query, sensorID, value, dateUTC)
	return err
}

// GetSensorReadings retrieves readings for a sensor within a time range
func (dm *DatabaseManager) GetSensorReadings(sensorID uuid.UUID, startTime, endTime time.Time, limit int) ([]models.SensorReading, error) {
	query := `
        SELECT id, sensor_id, value, date_utc, created_at
        FROM sensor_readings
        WHERE sensor_id = $1 AND date_utc >= $2 AND date_utc <= $3
        ORDER BY date_utc DESC
        LIMIT $4
    `

	rows, err := dm.QueryWithHealthCheck(context.Background(), query, sensorID, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var readings []models.SensorReading
	for rows.Next() {
		var reading models.SensorReading
		err := rows.Scan(&reading.ID, &reading.SensorID, &reading.Value, &reading.DateUTC, &reading.CreatedAt)
		if err != nil {
			log.Printf("Failed to scan reading: %v", err)
			continue
		}
		readings = append(readings, reading)
	}

	return readings, rows.Err()
}

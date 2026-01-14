package models

import (
	"time"

	"github.com/google/uuid"
)

// Sensor represents a physical sensor on a weather station
type Sensor struct {
	ID             uuid.UUID `json:"id"`
	StationID      uuid.UUID `json:"station_id"`
	SensorType     string    `json:"sensor_type"`
	Location       string    `json:"location"`
	Name           string    `json:"name,omitempty"`
	Model          string    `json:"model,omitempty"`
	BatteryLevel   *int      `json:"battery_level,omitempty"`
	SignalStrength *int      `json:"signal_strength,omitempty"`
	Enabled        bool      `json:"enabled"`
	RemoteID       string    `json:"remote_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// SensorReading represents a single measurement from a sensor
type SensorReading struct {
	ID        uuid.UUID `json:"id"`
	SensorID  uuid.UUID `json:"sensor_id"`
	Value     float64   `json:"value"`
	DateUTC   time.Time `json:"date_utc"`
	CreatedAt time.Time `json:"created_at"`
}

// SensorWithLatestReading combines sensor info with its latest reading
type SensorWithLatestReading struct {
	Sensor        Sensor         `json:"sensor"`
	LatestReading *SensorReading `json:"latest_reading,omitempty"`
	SensorType    string         `json:"sensor_type"`
}

// ParsedSensorData represents parsed sensor data from a weather station
type ParsedSensorData struct {
	Station StationData                 `json:"station"`
	Sensors map[uuid.UUID]SensorReading `json:"sensors"` // grouped by location
}

// SensorReadingWithMetadata includes sensor metadata with the reading
type SensorReadingWithMetadata struct {
	Reading  SensorReading `json:"reading"`
	Location string        `json:"location"`
	Type     string        `json:"type"`
	Unit     string        `json:"unit"`
	Category string        `json:"category"`
}

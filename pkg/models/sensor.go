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

// SensorQueryParams holds query parameters for sensor queries
type SensorQueryParams struct {
	StationID     *uuid.UUID
	SensorType    string
	Location      string
	Enabled       *bool
	IncludeLatest bool
}

// SensorWithLatestReading combines sensor info with its latest reading
type SensorWithLatestReading struct {
	Sensor        Sensor         `json:"sensor"`
	LatestReading *SensorReading `json:"latest_reading,omitempty"`
}

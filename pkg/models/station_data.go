package models

import "github.com/google/uuid"
import "time"

// StationData represents a weather station configuration
type StationData struct {
	ID          uuid.UUID              `json:"id"`
	PassKey     string                 `json:"pass_key"`
	StationType string                 `json:"station_type"`
	Model       string                 `json:"model"`
	Freq        string                 `json:"freq"`
	Interval    int                    `json:"interval"`
	Mode        string                 `json:"mode"`         // "push" or "pull"
	ServiceName string                 `json:"service_name"` // "ecowitt", "netatmo", etc.
	Config      map[string]interface{} `json:"config"`
	LastUpdate  *time.Time             `json:"last_update"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type StationListItem struct {
	ID          uuid.UUID  `json:"id"`
	PassKey     string     `json:"pass_key"`
	StationType string     `json:"station_type"`
	Model       string     `json:"model"`
	LastUpdate  *time.Time `json:"last_update"`
}

type StationDetail struct {
	ID            uuid.UUID `json:"id"`
	PassKey       string    `json:"pass_key"`
	StationType   string    `json:"station_type"`
	Model         string    `json:"model"`
	TotalReadings int       `json:"total_readings"`
	FirstReading  time.Time `json:"first_reading"`
	LastReading   time.Time `json:"last_reading"`
}

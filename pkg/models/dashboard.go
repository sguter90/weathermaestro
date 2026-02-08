package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Dashboard struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Config      json.RawMessage `json:"config"`
	IsDefault   bool            `json:"is_default"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type DashboardConfig struct {
	Layout  string         `json:"layout"`
	Widgets []WidgetConfig `json:"widgets"`
}

type WidgetConfig struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Position   WidgetPosition         `json:"position"`
	DataSource WidgetDataSource       `json:"data_source"`
	Settings   map[string]interface{} `json:"settings"`
}

type WidgetPosition struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type WidgetDataSource struct {
	StationID  *uuid.UUID `json:"station_id,omitempty"`
	SensorID   *uuid.UUID `json:"sensor_id,omitempty"`
	SensorType string     `json:"sensor_type,omitempty"`
	TimeRange  string     `json:"time_range"`
}

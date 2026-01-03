package api

import (
	"encoding/json"
	"fmt"
)

// Station represents a weather station
type Station struct {
	ID          int    `json:"id"`
	PassKey     string `json:"pass_key"`
	StationType string `json:"station_type"`
	Model       string `json:"model"`
	Freq        string `json:"freq"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// GetStations retrieves all registered weather stations
func (c *Client) GetStations() ([]Station, error) {
	resp, err := c.doRequest("GET", "/api/v1/stations", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var stations []Station
	if err := json.NewDecoder(resp.Body).Decode(&stations); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return stations, nil
}

// GetStation retrieves a specific weather station by ID
func (c *Client) GetStation(id int) (*Station, error) {
	path := fmt.Sprintf("/api/v1/stations/%d", id)

	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var station Station
	if err := json.NewDecoder(resp.Body).Decode(&station); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &station, nil
}

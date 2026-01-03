package api

import (
	"encoding/json"
	"fmt"
)

// HealthStatus represents the API health status
type HealthStatus struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
}

// Health checks if the API is healthy
func (c *Client) Health() (*HealthStatus, error) {
	resp, err := c.doRequest("GET", "/api/v1/health", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var health HealthStatus
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &health, nil
}

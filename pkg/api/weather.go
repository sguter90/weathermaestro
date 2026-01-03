package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/sguter90/weathermaestro/pkg/models"
)

// GetCurrentWeather retrieves the current weather data
func (c *Client) GetCurrentWeather() (*models.WeatherData, error) {
	resp, err := c.doRequest("GET", "/api/v1/weather/current", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data models.WeatherData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &data, nil
}

// HistoryOptions contains options for querying historical weather data
type HistoryOptions struct {
	Start    time.Time
	End      time.Time
	Interval string // "1m", "5m", "15m", "1h", "1d"
	Limit    int
}

// GetWeatherHistory retrieves historical weather data
func (c *Client) GetWeatherHistory(opts HistoryOptions) ([]models.WeatherData, error) {
	params := url.Values{}

	if !opts.Start.IsZero() {
		params.Set("start", opts.Start.Format(time.RFC3339))
	}
	if !opts.End.IsZero() {
		params.Set("end", opts.End.Format(time.RFC3339))
	}
	if opts.Interval != "" {
		params.Set("interval", opts.Interval)
	}
	if opts.Limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", opts.Limit))
	}

	path := "/api/v1/weather/history"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data []models.WeatherData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return data, nil
}

// WeatherStatistics represents aggregated weather statistics
type WeatherStatistics struct {
	Period    string    `json:"period"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`

	TempAvg float64 `json:"temp_avg"`
	TempMin float64 `json:"temp_min"`
	TempMax float64 `json:"temp_max"`

	HumidityAvg float64 `json:"humidity_avg"`

	RainTotal float64 `json:"rain_total"`

	WindSpeedAvg float64 `json:"wind_speed_avg"`
	WindSpeedMax float64 `json:"wind_speed_max"`
}

// GetStatistics retrieves aggregated weather statistics
func (c *Client) GetStatistics(start, end time.Time, period string) (*WeatherStatistics, error) {
	params := url.Values{}
	params.Set("start", start.Format(time.RFC3339))
	params.Set("end", end.Format(time.RFC3339))
	params.Set("period", period) // "hour", "day", "week", "month"

	path := "/api/v1/weather/statistics?" + params.Encode()

	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var stats WeatherStatistics
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &stats, nil
}

// ChartData represents data optimized for chart rendering
type ChartData struct {
	Labels []string  `json:"labels"`
	Temps  []float64 `json:"temps"`
	Rain   []float64 `json:"rain"`
	Wind   []float64 `json:"wind"`
}

// GetChartData retrieves data optimized for charts
func (c *Client) GetChartData(start, end time.Time, dataPoints int) (*ChartData, error) {
	params := url.Values{}
	params.Set("start", start.Format(time.RFC3339))
	params.Set("end", end.Format(time.RFC3339))
	params.Set("points", fmt.Sprintf("%d", dataPoints))

	path := "/api/v1/weather/charts?" + params.Encode()

	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data ChartData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &data, nil
}

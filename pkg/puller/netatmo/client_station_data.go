package netatmo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type StationDataDeviceDashboard struct {
	TimeUTC          int64   `json:"time_utc"`
	Temperature      float64 `json:"Temperature"`
	CO2              int     `json:"CO2"`
	Humidity         int     `json:"Humidity"`
	Noise            int     `json:"Noise"`
	Pressure         float64 `json:"Pressure"`
	AbsolutePressure float64 `json:"AbsolutePressure"`
	MinTemp          float64 `json:"min_temp"`
	MaxTemp          float64 `json:"max_temp"`
	DateMinTemp      int64   `json:"date_min_temp"`
	DateMaxTemp      int64   `json:"date_max_temp"`
	TempTrend        string  `json:"temp_trend"`
	PressureTrend    string  `json:"pressure_trend"`
}

type StationDataModuleDashboard struct {
	TimeUTC          int64   `json:"time_utc"`
	Temperature      float64 `json:"Temperature"`
	Humidity         int     `json:"Humidity"`
	Pressure         float64 `json:"Pressure"`
	AbsolutePressure float64 `json:"AbsolutePressure"`
	CO2              int     `json:"CO2"`
	Noise            int     `json:"Noise"`
	MinTemp          float64 `json:"min_temp"`
	MaxTemp          float64 `json:"max_temp"`
	DateMinTemp      int64   `json:"date_min_temp"`
	DateMaxTemp      int64   `json:"date_max_temp"`
	TempTrend        string  `json:"temp_trend"`
	Rain             float64 `json:"Rain"`
	SumRain24        float64 `json:"sum_rain_24"`
	SumRain1         float64 `json:"sum_rain_1"`
	WindStrength     int     `json:"WindStrength"`
	WindAngle        int     `json:"WindAngle"`
	GustStrength     int     `json:"GustStrength"`
	GustAngle        int     `json:"GustAngle"`
	MaxWindStr       int     `json:"max_wind_str"`
	MaxWindAngle     int     `json:"max_wind_angle"`
	DateMaxWindStr   int64   `json:"date_max_wind_str"`
	UV               int     `json:"UV"`
}

type StationDataModule struct {
	ID             string                     `json:"_id"`
	Type           string                     `json:"type"`
	ModuleName     string                     `json:"module_name"`
	DataType       []string                   `json:"data_type"`
	LastSetup      int64                      `json:"last_setup"`
	Reachable      bool                       `json:"reachable"`
	Firmware       int                        `json:"firmware"`
	LastMessage    int64                      `json:"last_message"`
	LastSeen       int64                      `json:"last_seen"`
	RFStatus       int                        `json:"rf_status"`
	BatteryVP      int                        `json:"battery_vp"`
	BatteryPercent int                        `json:"battery_percent"`
	DashboardData  StationDataModuleDashboard `json:"dashboard_data"`
}

type StationDataDevice struct {
	ID              string   `json:"_id"`
	DateSetup       int64    `json:"date_setup"`
	LastSetup       int64    `json:"last_setup"`
	Type            string   `json:"type"`
	LastStatusStore int64    `json:"last_status_store"`
	ModuleName      string   `json:"module_name"`
	Firmware        int      `json:"firmware"`
	LastUpgrade     int64    `json:"last_upgrade"`
	WifiStatus      int      `json:"wifi_status"`
	Reachable       bool     `json:"reachable"`
	CO2Calibrating  bool     `json:"co2_calibrating"`
	StationName     string   `json:"station_name"`
	DataType        []string `json:"data_type"`
	ReadOnly        bool     `json:"read_only"`
	HomeID          string   `json:"home_id"`
	HomeName        string   `json:"home_name"`
	Place           struct {
		Timezone string    `json:"timezone"`
		Country  string    `json:"country"`
		Altitude int       `json:"altitude"`
		Location []float64 `json:"location"`
	} `json:"place"`
	DashboardData StationDataDeviceDashboard `json:"dashboard_data"`
	Modules       []StationDataModule        `json:"modules"`
}

type StationDataUser struct {
	Mail           string `json:"mail"`
	Administrative struct {
		RegLocale    string `json:"reg_locale"`
		Lang         string `json:"lang"`
		Country      string `json:"country"`
		Unit         int    `json:"unit"`
		WindUnit     int    `json:"windunit"`
		PressureUnit int    `json:"pressureunit"`
		FeelLikeAlgo int    `json:"feel_like_algo"`
	} `json:"administrative"`
}

// StationDataResponse represents the Netatmo API response structure
type StationDataResponse struct {
	Body struct {
		Devices []StationDataDevice `json:"devices"`
		User    StationDataUser     `json:"user"`
	} `json:"body"`
	Status     string  `json:"status"`
	TimeExec   float64 `json:"time_exec"`
	TimeServer int64   `json:"time_server"`
}

// GetStationsData retrieves weather station data from Netatmo API
func (c *Client) GetStationsData(ctx context.Context, deviceID string) (*StationDataResponse, error) {
	if err := c.ensureValidToken(ctx); err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.netatmo.com/api/getstationsdata", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := url.Values{}
	q.Add("access_token", c.accessToken)
	if deviceID != "" {
		q.Add("device_id", deviceID)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data from Netatmo: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("netatmo API returned status %d: %s", resp.StatusCode, string(body))
	}

	var netatmoResp StationDataResponse
	if err := json.Unmarshal(body, &netatmoResp); err != nil {
		return nil, fmt.Errorf("failed to parse Netatmo response: %w\nResponse:\n%s", err, string(body))
	}

	return &netatmoResp, nil
}

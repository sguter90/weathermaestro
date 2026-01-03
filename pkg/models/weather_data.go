package models

import "time"

// WeatherData represents weather data with all units for API output
type WeatherData struct {
	// Station Info
	PassKey     string    `json:"pass_key"`
	StationType string    `json:"station_type"`
	Model       string    `json:"model"`
	Freq        string    `json:"freq"`
	DateUTC     time.Time `json:"date_utc"`
	Interval    int       `json:"interval"`

	// System Info
	Runtime int `json:"runtime"`
	Heap    int `json:"heap"`

	// Indoor - Metric
	TempInC    float64 `json:"temp_in_c"`
	HumidityIn int     `json:"humidity_in"`

	// Indoor - Imperial
	TempInF float64 `json:"temp_in_f"`

	// Outdoor - Metric
	TempOutC    float64 `json:"temp_out_c"`
	HumidityOut int     `json:"humidity_out"`

	// Outdoor - Imperial
	TempOutF float64 `json:"temp_out_f"`

	// Barometric Pressure - Metric
	BaromRelHPa float64 `json:"barom_rel_hpa"`
	BaromAbsHPa float64 `json:"barom_abs_hpa"`

	// Barometric Pressure - Imperial
	BaromRelIn float64 `json:"barom_rel_in"`
	BaromAbsIn float64 `json:"barom_abs_in"`

	// Wind - Metric
	WindDir         int     `json:"wind_dir"`
	WindSpeedMS     float64 `json:"wind_speed_ms"`
	WindGustMS      float64 `json:"wind_gust_ms"`
	MaxDailyGustMS  float64 `json:"max_daily_gust_ms"`
	WindSpeedKmH    float64 `json:"wind_speed_kmh"`
	WindGustKmH     float64 `json:"wind_gust_kmh"`
	MaxDailyGustKmH float64 `json:"max_daily_gust_kmh"`

	// Wind - Imperial
	WindSpeedMPH    float64 `json:"wind_speed_mph"`
	WindGustMPH     float64 `json:"wind_gust_mph"`
	MaxDailyGustMPH float64 `json:"max_daily_gust_mph"`

	// Solar & UV
	SolarRadiation float64 `json:"solar_radiation"`
	UV             int     `json:"uv"`

	// Rain - Metric
	RainRateMmH   float64 `json:"rain_rate_mm_h"`
	EventRainMm   float64 `json:"event_rain_mm"`
	HourlyRainMm  float64 `json:"hourly_rain_mm"`
	DailyRainMm   float64 `json:"daily_rain_mm"`
	WeeklyRainMm  float64 `json:"weekly_rain_mm"`
	MonthlyRainMm float64 `json:"monthly_rain_mm"`
	YearlyRainMm  float64 `json:"yearly_rain_mm"`
	TotalRainMm   float64 `json:"total_rain_mm"`

	// Rain - Imperial
	RainRateIn    float64 `json:"rain_rate_in"`
	EventRainIn   float64 `json:"event_rain_in"`
	HourlyRainIn  float64 `json:"hourly_rain_in"`
	DailyRainIn   float64 `json:"daily_rain_in"`
	WeeklyRainIn  float64 `json:"weekly_rain_in"`
	MonthlyRainIn float64 `json:"monthly_rain_in"`
	YearlyRainIn  float64 `json:"yearly_rain_in"`
	TotalRainIn   float64 `json:"total_rain_in"`

	// Additional Sensors
	VPD      float64 `json:"vpd"`
	WH65Batt int     `json:"wh65_batt"`
}

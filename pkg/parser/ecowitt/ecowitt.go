package ecowitt

import (
	"net/url"
	"strconv"
	"time"

	"github.com/sguter90/weathermaestro/pkg/models"
)

// Parser implements the Ecowitt weather station parser
type Parser struct{}

// GetEndpoint returns the endpoint path for Ecowitt stations
func (p *Parser) GetEndpoint() string {
	return "/data/report"
}

// GetStationType returns the station type identifier
func (p *Parser) GetStationType() string {
	return "Ecowitt"
}

// Parse converts Ecowitt format to WeatherData
func (p *Parser) Parse(params url.Values) (*models.WeatherData, *models.StationData, error) {
	stationData := &models.StationData{
		PassKey:     params.Get("PASSKEY"),
		StationType: params.Get("stationtype"),
		Model:       params.Get("model"),
		Freq:        params.Get("freq"),
	}
	weatherData := &models.WeatherData{}

	// Parse date
	if dateStr := params.Get("dateutc"); dateStr != "" {
		formats := []string{
			"2006-01-02 15:04:05",
			"2006-01-02+15:04:05",
		}

		for _, format := range formats {
			if t, err := time.Parse(format, dateStr); err == nil {
				weatherData.DateUTC = t
				break
			}
		}
	}

	// Helper function to parse integers
	parseInt := func(key string) int {
		if val := params.Get(key); val != "" {
			if i, err := strconv.Atoi(val); err == nil {
				return i
			}
		}
		return 0
	}

	// Helper function to parse floats
	parseFloat := func(key string) float64 {
		if val := params.Get(key); val != "" {
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				return f
			}
		}
		return 0.0
	}

	// Parse system info
	weatherData.Runtime = parseInt("runtime")
	weatherData.Heap = parseInt("heap")
	stationData.Interval = parseInt("interval")

	// Parse indoor temperature (Ecowitt sends in Fahrenheit)
	tempInF := parseFloat("tempinf")
	weatherData.TempInF = tempInF
	weatherData.TempInC = (tempInF - 32) * 5 / 9
	weatherData.HumidityIn = parseInt("humidityin")

	// Parse outdoor temperature (Ecowitt sends in Fahrenheit)
	tempOutF := parseFloat("tempf")
	weatherData.TempOutF = tempOutF
	weatherData.TempOutC = (tempOutF - 32) * 5 / 9
	weatherData.HumidityOut = parseInt("humidity")

	// Parse barometric pressure (Ecowitt sends in inHg)
	baromRelIn := parseFloat("baromrelin")
	baromAbsIn := parseFloat("baromabsin")
	weatherData.BaromRelIn = baromRelIn
	weatherData.BaromAbsIn = baromAbsIn
	weatherData.BaromRelHPa = baromRelIn * 33.8639
	weatherData.BaromAbsHPa = baromAbsIn * 33.8639

	// Parse wind (Ecowitt sends in mph)
	weatherData.WindDir = parseInt("winddir")
	windSpeedMPH := parseFloat("windspeedmph")
	windGustMPH := parseFloat("windgustmph")
	maxDailyGustMPH := parseFloat("maxdailygust")

	weatherData.WindSpeedMPH = windSpeedMPH
	weatherData.WindGustMPH = windGustMPH
	weatherData.MaxDailyGustMPH = maxDailyGustMPH

	weatherData.WindSpeedMS = windSpeedMPH * 0.44704
	weatherData.WindGustMS = windGustMPH * 0.44704
	weatherData.MaxDailyGustMS = maxDailyGustMPH * 0.44704

	weatherData.WindSpeedKmH = windSpeedMPH * 1.60934
	weatherData.WindGustKmH = windGustMPH * 1.60934
	weatherData.MaxDailyGustKmH = maxDailyGustMPH * 1.60934

	// Parse solar & UV
	weatherData.SolarRadiation = parseFloat("solarradiation")
	weatherData.UV = parseInt("uv")

	// Parse rain (Ecowitt sends in inches)
	rainRateIn := parseFloat("rainratein")
	eventRainIn := parseFloat("eventrainin")
	hourlyRainIn := parseFloat("hourlyrainin")
	dailyRainIn := parseFloat("dailyrainin")
	weeklyRainIn := parseFloat("weeklyrainin")
	monthlyRainIn := parseFloat("monthlyrainin")
	yearlyRainIn := parseFloat("yearlyrainin")
	totalRainIn := parseFloat("totalrainin")

	weatherData.RainRateIn = rainRateIn
	weatherData.EventRainIn = eventRainIn
	weatherData.HourlyRainIn = hourlyRainIn
	weatherData.DailyRainIn = dailyRainIn
	weatherData.WeeklyRainIn = weeklyRainIn
	weatherData.MonthlyRainIn = monthlyRainIn
	weatherData.YearlyRainIn = yearlyRainIn
	weatherData.TotalRainIn = totalRainIn

	weatherData.RainRateMmH = rainRateIn * 25.4
	weatherData.EventRainMm = eventRainIn * 25.4
	weatherData.HourlyRainMm = hourlyRainIn * 25.4
	weatherData.DailyRainMm = dailyRainIn * 25.4
	weatherData.WeeklyRainMm = weeklyRainIn * 25.4
	weatherData.MonthlyRainMm = monthlyRainIn * 25.4
	weatherData.YearlyRainMm = yearlyRainIn * 25.4
	weatherData.TotalRainMm = totalRainIn * 25.4

	// Parse additional sensors
	weatherData.VPD = parseFloat("vpd")
	weatherData.WH65Batt = parseInt("wh65batt")

	return weatherData, stationData, nil
}

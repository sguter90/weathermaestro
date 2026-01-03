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
	return "/api/v1/weather/update/ecowitt"
}

// GetStationType returns the station type identifier
func (p *Parser) GetStationType() string {
	return "Ecowitt"
}

// Parse converts Ecowitt format to WeatherData
func (p *Parser) Parse(params url.Values) (*models.WeatherData, error) {
	// KORREKTUR: models.WeatherData statt WeatherData
	data := &models.WeatherData{
		PassKey:     params.Get("PASSKEY"),
		StationType: params.Get("stationtype"),
		Model:       params.Get("model"),
		Freq:        params.Get("freq"),
	}

	// Parse date
	if dateStr := params.Get("dateutc"); dateStr != "" {
		formats := []string{
			"2006-01-02 15:04:05",
			"2006-01-02+15:04:05",
		}

		for _, format := range formats {
			if t, err := time.Parse(format, dateStr); err == nil {
				data.DateUTC = t
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
	data.Runtime = parseInt("runtime")
	data.Heap = parseInt("heap")
	data.Interval = parseInt("interval")

	// Parse indoor temperature (Ecowitt sends in Fahrenheit)
	tempInF := parseFloat("tempinf")
	data.TempInF = tempInF
	data.TempInC = (tempInF - 32) * 5 / 9
	data.HumidityIn = parseInt("humidityin")

	// Parse outdoor temperature (Ecowitt sends in Fahrenheit)
	tempOutF := parseFloat("tempf")
	data.TempOutF = tempOutF
	data.TempOutC = (tempOutF - 32) * 5 / 9
	data.HumidityOut = parseInt("humidity")

	// Parse barometric pressure (Ecowitt sends in inHg)
	baromRelIn := parseFloat("baromrelin")
	baromAbsIn := parseFloat("baromabsin")
	data.BaromRelIn = baromRelIn
	data.BaromAbsIn = baromAbsIn
	data.BaromRelHPa = baromRelIn * 33.8639
	data.BaromAbsHPa = baromAbsIn * 33.8639

	// Parse wind (Ecowitt sends in mph)
	data.WindDir = parseInt("winddir")
	windSpeedMPH := parseFloat("windspeedmph")
	windGustMPH := parseFloat("windgustmph")
	maxDailyGustMPH := parseFloat("maxdailygust")

	data.WindSpeedMPH = windSpeedMPH
	data.WindGustMPH = windGustMPH
	data.MaxDailyGustMPH = maxDailyGustMPH

	data.WindSpeedMS = windSpeedMPH * 0.44704
	data.WindGustMS = windGustMPH * 0.44704
	data.MaxDailyGustMS = maxDailyGustMPH * 0.44704

	data.WindSpeedKmH = windSpeedMPH * 1.60934
	data.WindGustKmH = windGustMPH * 1.60934
	data.MaxDailyGustKmH = maxDailyGustMPH * 1.60934

	// Parse solar & UV
	data.SolarRadiation = parseFloat("solarradiation")
	data.UV = parseInt("uv")

	// Parse rain (Ecowitt sends in inches)
	rainRateIn := parseFloat("rainratein")
	eventRainIn := parseFloat("eventrainin")
	hourlyRainIn := parseFloat("hourlyrainin")
	dailyRainIn := parseFloat("dailyrainin")
	weeklyRainIn := parseFloat("weeklyrainin")
	monthlyRainIn := parseFloat("monthlyrainin")
	yearlyRainIn := parseFloat("yearlyrainin")
	totalRainIn := parseFloat("totalrainin")

	data.RainRateIn = rainRateIn
	data.EventRainIn = eventRainIn
	data.HourlyRainIn = hourlyRainIn
	data.DailyRainIn = dailyRainIn
	data.WeeklyRainIn = weeklyRainIn
	data.MonthlyRainIn = monthlyRainIn
	data.YearlyRainIn = yearlyRainIn
	data.TotalRainIn = totalRainIn

	data.RainRateMmH = rainRateIn * 25.4
	data.EventRainMm = eventRainIn * 25.4
	data.HourlyRainMm = hourlyRainIn * 25.4
	data.DailyRainMm = dailyRainIn * 25.4
	data.WeeklyRainMm = weeklyRainIn * 25.4
	data.MonthlyRainMm = monthlyRainIn * 25.4
	data.YearlyRainMm = yearlyRainIn * 25.4
	data.TotalRainMm = totalRainIn * 25.4

	// Parse additional sensors
	data.VPD = parseFloat("vpd")
	data.WH65Batt = parseInt("wh65batt")

	return data, nil
}

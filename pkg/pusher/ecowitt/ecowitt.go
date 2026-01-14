package ecowitt

import (
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/sguter90/weathermaestro/pkg/models"
)

// Pusher implements the Ecowitt weather station pusher
type Pusher struct{}

// GetEndpoint returns the endpoint path for Ecowitt stations
func (p *Pusher) GetEndpoint() string {
	return "/data/report"
}

// GetStationType returns the station type identifier
func (p *Pusher) GetStationType() string {
	return "Ecowitt"
}

func (p *Pusher) ParseStation(params url.Values) *models.StationData {
	return &models.StationData{
		PassKey:     params.Get("PASSKEY"),
		StationType: params.Get("stationtype"),
		Model:       params.Get("model"),
		Freq:        params.Get("freq"),
		Mode:        "push",
	}
}

func (p *Pusher) ParseSensors(params url.Values) map[string]models.Sensor {
	supportedSensors := GetSupportedEcowittSensors()

	result := make(map[string]models.Sensor)
	for _, sensor := range supportedSensors {
		if val := params.Get(sensor.Name); val != "" {
			result[sensor.RemoteID] = sensor
		}
	}

	return result
}

// ParseWeatherData Parse parses Ecowitt data with multiple sensors and returns structured sensor data
func (p *Pusher) ParseWeatherData(params url.Values, sensors map[string]models.Sensor) (map[uuid.UUID]models.SensorReading, error) {
	result := make(map[uuid.UUID]models.SensorReading)

	// Parse date once
	var dateUTC time.Time
	if dateStr := params.Get("dateutc"); dateStr != "" {
		formats := []string{
			"2006-01-02 15:04:05",
			"2006-01-02+15:04:05",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, dateStr); err == nil {
				dateUTC = t
				break
			}
		}
	}
	if dateUTC.IsZero() {
		dateUTC = time.Now().UTC()
	}

	// Helper functions
	parseFloat := func(key string) (float64, bool) {
		if val := params.Get(key); val != "" {
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				return f, true
			}
		}
		return 0.0, false
	}

	parseInt := func(key string) (int, bool) {
		if val := params.Get(key); val != "" {
			if i, err := strconv.Atoi(val); err == nil {
				return i, true
			}
		}
		return 0, false
	}

	// Iterate over all sensors and parse their values
	for remoteID, sensor := range sensors {
		var value float64
		var hasValue bool

		// Get raw value from params
		rawValue := params.Get(remoteID)
		if rawValue == "" {
			continue
		}

		// Parse and convert based on sensor type
		switch sensor.SensorType {
		// Temperature sensors (Fahrenheit to Celsius)
		case models.SensorTypeTemperature, models.SensorTypeTemperatureOutdoor:
			if f, ok := parseFloat(remoteID); ok {
				value = (f - 32) * 5 / 9
				hasValue = true
			}

		// Humidity sensors (percentage)
		case models.SensorTypeHumidity, models.SensorTypeHumidityOutdoor:
			if i, ok := parseInt(remoteID); ok {
				value = float64(i)
				hasValue = true
			}

		// Pressure sensors (inHg to hPa)
		case models.SensorTypePressureRelative, models.SensorTypePressureAbsolute:
			if f, ok := parseFloat(remoteID); ok {
				value = f * 33.8639
				hasValue = true
			}

		// Wind speed sensors (mph to m/s)
		case models.SensorTypeWindSpeed, models.SensorTypeWindGust, models.SensorTypeWindGustMaxDaily:
			if f, ok := parseFloat(remoteID); ok {
				value = f * 0.44704
				hasValue = true
			}

		// Wind direction (degrees)
		case models.SensorTypeWindDirection:
			if i, ok := parseInt(remoteID); ok {
				value = float64(i)
				hasValue = true
			}

		// Rain sensors (inches to mm)
		case models.SensorTypeRainfallRate,
			models.SensorTypeRainfallEvent,
			models.SensorTypeRainfallHourly,
			models.SensorTypeRainfallDaily,
			models.SensorTypeRainfallWeekly,
			models.SensorTypeRainfallMonthly,
			models.SensorTypeRainfallYearly,
			models.SensorTypeRainfallTotal:
			if f, ok := parseFloat(remoteID); ok {
				value = f * 25.4
				hasValue = true
			}

		// Solar radiation (W/m²)
		case models.SensorTypeSolarRadiation:
			if f, ok := parseFloat(remoteID); ok {
				value = f
				hasValue = true
			}

		// UV Index
		case models.SensorTypeUVIndex:
			if i, ok := parseInt(remoteID); ok {
				value = float64(i)
				hasValue = true
			}

		// VPD (kPa)
		case models.SensorTypeVPD:
			if f, ok := parseFloat(remoteID); ok {
				value = f
				hasValue = true
			}

		// Battery (percentage)
		case models.SensorTypeBattery:
			if i, ok := parseInt(remoteID); ok {
				value = float64(i)
				hasValue = true
			}

		// Signal strength (dBm)
		case models.SensorTypeSignalStrength:
			if i, ok := parseInt(remoteID); ok {
				value = float64(i)
				hasValue = true
			}

		default:
			// For unknown sensor types, try to parse as float
			if f, ok := parseFloat(remoteID); ok {
				value = f
				hasValue = true
			}
		}

		// Add reading to result if we have a valid value
		if hasValue {
			result[sensor.ID] = models.SensorReading{
				SensorID: sensor.ID,
				Value:    value,
				DateUTC:  dateUTC,
			}
		}
	}

	return result, nil
}

package netatmo

import (
	"sort"
	"strings"

	"github.com/sguter90/weathermaestro/pkg/models"
)

// SupportedSensor describes a sensor this provider knows about. NetatmoType is
// the type identifier as expected by the Netatmo getmeasure endpoint, or empty
// for sensors whose value is not exposed via getmeasure (e.g. AbsolutePressure).
type SupportedSensor struct {
	Sensor      models.Sensor
	NetatmoType string
}

// GetSupportedSensors returns the catalog of sensors this provider can produce.
// Keys are "<ModuleType>-<SensorType>" — the same shape used as the sensor's
// remote-ID suffix.
func GetSupportedSensors() map[string]SupportedSensor {
	return map[string]SupportedSensor{
		// Indoor Module (Main Device, NAMain)
		"NAMain-" + models.SensorTypeTemperature: {
			Sensor:      models.Sensor{Name: "Temperature (Indoor)", SensorType: models.SensorTypeTemperature, Location: "Indoor", Enabled: true},
			NetatmoType: "temperature",
		},
		"NAMain-" + models.SensorTypeHumidity: {
			Sensor:      models.Sensor{Name: "Humidity (Indoor)", SensorType: models.SensorTypeHumidity, Location: "Indoor", Enabled: true},
			NetatmoType: "humidity",
		},
		"NAMain-" + models.SensorTypePressure: {
			Sensor:      models.Sensor{Name: "Pressure", SensorType: models.SensorTypePressure, Location: "Indoor", Enabled: true},
			NetatmoType: "pressure",
		},
		"NAMain-" + models.SensorTypeCO2: {
			Sensor:      models.Sensor{Name: "CO2", SensorType: models.SensorTypeCO2, Location: "Indoor", Enabled: true},
			NetatmoType: "co2",
		},
		"NAMain-" + models.SensorTypeNoise: {
			Sensor:      models.Sensor{Name: "Noise", SensorType: models.SensorTypeNoise, Location: "Indoor", Enabled: true},
			NetatmoType: "noise",
		},

		// Outdoor Module (NAModule1)
		"NAModule1-" + models.SensorTypeTemperatureOutdoor: {
			Sensor:      models.Sensor{Name: "Temperature (Outdoor)", SensorType: models.SensorTypeTemperatureOutdoor, Location: "Outdoor", Enabled: true},
			NetatmoType: "temperature",
		},
		"NAModule1-" + models.SensorTypeHumidityOutdoor: {
			Sensor:      models.Sensor{Name: "Humidity (Outdoor)", SensorType: models.SensorTypeHumidityOutdoor, Location: "Outdoor", Enabled: true},
			NetatmoType: "humidity",
		},

		// Wind Gauge (NAModule2)
		"NAModule2-" + models.SensorTypeWindDirection: {
			Sensor:      models.Sensor{Name: "Wind Direction", SensorType: models.SensorTypeWindDirection, Location: "Outdoor", Enabled: true},
			NetatmoType: "windangle",
		},
		"NAModule2-" + models.SensorTypeWindSpeed: {
			Sensor:      models.Sensor{Name: "Wind Speed", SensorType: models.SensorTypeWindSpeed, Location: "Outdoor", Enabled: true},
			NetatmoType: "windstrength",
		},
		"NAModule2-" + models.SensorTypeWindGust: {
			Sensor:      models.Sensor{Name: "Wind Gust", SensorType: models.SensorTypeWindGust, Location: "Outdoor", Enabled: true},
			NetatmoType: "guststrength",
		},
		"NAModule2-" + models.SensorTypeWindGustAngle: {
			Sensor:      models.Sensor{Name: "Wind Gust Angle", SensorType: models.SensorTypeWindGustAngle, Location: "Outdoor", Enabled: true},
			NetatmoType: "gustangle",
		},

		// Rain Gauge (NAModule3)
		"NAModule3-" + models.SensorTypeRainfallRate: {
			Sensor:      models.Sensor{Name: "Rain Rate", SensorType: models.SensorTypeRainfallRate, Location: "Outdoor", Enabled: true},
			NetatmoType: "rain",
		},
		"NAModule3-" + models.SensorTypeRainfallDaily: {
			Sensor:      models.Sensor{Name: "Rain (24h)", SensorType: models.SensorTypeRainfallDaily, Location: "Outdoor", Enabled: true},
			NetatmoType: "sum_rain",
		},

		// Additional Indoor Module (NAModule4)
		"NAModule4-" + models.SensorTypeTemperature: {
			Sensor:      models.Sensor{Name: "Temperature (Additional Indoor)", SensorType: models.SensorTypeTemperature, Location: "Indoor", Enabled: true},
			NetatmoType: "temperature",
		},
		"NAModule4-" + models.SensorTypeHumidity: {
			Sensor:      models.Sensor{Name: "Humidity (Additional Indoor)", SensorType: models.SensorTypeHumidity, Location: "Indoor", Enabled: true},
			NetatmoType: "humidity",
		},
		"NAModule4-" + models.SensorTypePressure: {
			Sensor:      models.Sensor{Name: "Pressure (Additional Indoor)", SensorType: models.SensorTypePressure, Location: "Indoor", Enabled: true},
			NetatmoType: "pressure",
		},
		"NAModule4-" + models.SensorTypeCO2: {
			Sensor:      models.Sensor{Name: "CO2 (Additional Indoor)", SensorType: models.SensorTypeCO2, Location: "Indoor", Enabled: true},
			NetatmoType: "co2",
		},
		"NAModule4-" + models.SensorTypeNoise: {
			Sensor:      models.Sensor{Name: "Noise (Additional Indoor)", SensorType: models.SensorTypeNoise, Location: "Indoor", Enabled: true},
			NetatmoType: "noise",
		},
	}
}

// netatmoMeasureMapping pairs a Netatmo getmeasure data type with the sensor
// type that receives the value. Position in a slice = position in the API response.
type netatmoMeasureMapping struct {
	NetatmoType string
	SensorType  string
}

// getMeasureMappingsFor returns the (NetatmoType, SensorType) pairs for sensors
// belonging to the given module type (e.g. "NAMain", "NAModule1"). Sensors
// without a NetatmoType (not exposed by getmeasure) are filtered out. The order
// is deterministic so the API response can be mapped positionally.
func getMeasureMappingsFor(moduleType string) []netatmoMeasureMapping {
	prefix := moduleType + "-"
	var mappings []netatmoMeasureMapping
	for key, s := range GetSupportedSensors() {
		if !strings.HasPrefix(key, prefix) || s.NetatmoType == "" {
			continue
		}
		mappings = append(mappings, netatmoMeasureMapping{
			NetatmoType: s.NetatmoType,
			SensorType:  s.Sensor.SensorType,
		})
	}
	sort.Slice(mappings, func(i, j int) bool {
		return mappings[i].NetatmoType < mappings[j].NetatmoType
	})
	return mappings
}

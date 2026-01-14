package ecowitt

import (
	"github.com/sguter90/weathermaestro/pkg/models"
)

func GetSupportedEcowittSensors() []models.Sensor {
	return []models.Sensor{
		// Indoor
		{
			Name:       "Temperature",
			SensorType: models.SensorTypeTemperature,
			Location:   "Indoor",
			Enabled:    true,
			RemoteID:   "tempinf",
		},
		{
			Name:       "Humidity",
			SensorType: models.SensorTypeHumidity,
			Location:   "Indoor",
			Enabled:    true,
			RemoteID:   "humidityin",
		},
		{
			Name:       "Barometric Pressure (Relative)",
			SensorType: models.SensorTypePressureRelative,
			Location:   "Indoor",
			Enabled:    true,
			RemoteID:   "baromrelin",
		},
		{
			Name:       "Barometric Pressure (baromabsin)",
			SensorType: models.SensorTypePressureRelative,
			Location:   "Indoor",
			Enabled:    true,
			RemoteID:   "baromrelin",
		},
		// Outdoor
		{
			Name:       "Temperature",
			SensorType: models.SensorTypeTemperature,
			Location:   "Outdoor",
			Enabled:    true,
			RemoteID:   "tempf",
		},
		{
			Name:       "Humidity",
			SensorType: models.SensorTypeHumidity,
			Location:   "Outdoor",
			Enabled:    true,
			RemoteID:   "humidity",
		},
		{
			Name:       "Wind Direction",
			SensorType: models.SensorTypeWindDirection,
			Location:   "Outdoor",
			Enabled:    true,
			RemoteID:   "winddir",
		},
		{
			Name:       "Wind Speed",
			SensorType: models.SensorTypeWindSpeed,
			Location:   "Outdoor",
			Enabled:    true,
			RemoteID:   "windspeedmph",
		},
		{
			Name:       "Wind Gust",
			SensorType: models.SensorTypeWindGust,
			Location:   "Outdoor",
			Enabled:    true,
			RemoteID:   "windgustmph",
		},
		{
			Name:       "Wind Gust (Max Daily)",
			SensorType: models.SensorTypeWindGustMaxDaily,
			Location:   "Outdoor",
			Enabled:    true,
			RemoteID:   "maxdailygust",
		},
		{
			Name:       "Solar Radiation",
			SensorType: models.SensorTypeSolarRadiation,
			Location:   "Outdoor",
			Enabled:    true,
			RemoteID:   "solarradiation",
		},
		{
			Name:       "UV Index",
			SensorType: models.SensorTypeUVIndex,
			Location:   "Outdoor",
			Enabled:    true,
			RemoteID:   "uv",
		},
		{
			Name:       "Rain Rate",
			SensorType: models.SensorTypeRainfallRate,
			Location:   "Outdoor",
			Enabled:    true,
			RemoteID:   "rainratein",
		},
		{
			Name:       "Rain (Event)",
			SensorType: models.SensorTypeRainfallEvent,
			Location:   "Outdoor",
			Enabled:    true,
			RemoteID:   "eventrainin",
		},
		{
			Name:       "Rain (Hourly)",
			SensorType: models.SensorTypeRainfallHourly,
			Location:   "Outdoor",
			Enabled:    true,
			RemoteID:   "hourlyrainin",
		},
		{
			Name:       "Rain (Daily)",
			SensorType: models.SensorTypeRainfallDaily,
			Location:   "Outdoor",
			Enabled:    true,
			RemoteID:   "dailyrainin",
		},
		{
			Name:       "Rain (Weekly)",
			SensorType: models.SensorTypeRainfallWeekly,
			Location:   "Outdoor",
			Enabled:    true,
			RemoteID:   "weeklyrainin",
		},
		{
			Name:       "Rain (Monthly)",
			SensorType: models.SensorTypeRainfallMonthly,
			Location:   "Outdoor",
			Enabled:    true,
			RemoteID:   "monthlyrainin",
		},
		{
			Name:       "Rain (Yearly)",
			SensorType: models.SensorTypeRainfallYearly,
			Location:   "Outdoor",
			Enabled:    true,
			RemoteID:   "yearlyrainin",
		},
		{
			Name:       "Rain (Total)",
			SensorType: models.SensorTypeRainfallTotal,
			Location:   "Outdoor",
			Enabled:    true,
			RemoteID:   "totalrainin",
		},
		{
			Name:       "Vapour Pressure Deficit",
			SensorType: models.SensorTypeVPD,
			Location:   "Outdoor",
			Enabled:    true,
			RemoteID:   "vpd",
		},
		{
			Name:       "Vapour Pressure Deficit",
			SensorType: models.SensorTypeVPD,
			Location:   "Outdoor",
			Enabled:    true,
			RemoteID:   "vpd",
		},
		{
			Name:       "Battery (Outdoor Device)",
			SensorType: models.SensorTypeBattery,
			Location:   "Outdoor",
			Enabled:    true,
			RemoteID:   "wh65batt",
		},
	}
}

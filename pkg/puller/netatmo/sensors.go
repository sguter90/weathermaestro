package netatmo

import "github.com/sguter90/weathermaestro/pkg/models"

func GetSupportedSensors() map[string]models.Sensor {
	sensors := make(map[string]models.Sensor)
	// Indoor Module (Main Device)
	sensors["NAMain-"+models.SensorTypeTemperature] = models.Sensor{
		Name:       "Temperature (Indoor)",
		SensorType: models.SensorTypeTemperature,
		Location:   "Indoor",
		Enabled:    true,
	}
	sensors["NAMain-"+models.SensorTypeHumidity] = models.Sensor{
		Name:       "Humidity (Indoor)",
		SensorType: models.SensorTypeHumidity,
		Location:   "Indoor",
		Enabled:    true,
	}
	sensors["NAMain-"+models.SensorTypePressure] = models.Sensor{
		Name:       "Pressure",
		SensorType: models.SensorTypePressure,
		Location:   "Indoor",
		Enabled:    true,
	}
	sensors["NAMain-"+models.SensorTypePressureAbsolute] = models.Sensor{
		Name:       "Pressure (Absolute)",
		SensorType: models.SensorTypePressureAbsolute,
		Location:   "Indoor",
		Enabled:    true,
	}
	sensors["NAMain-"+models.SensorTypeCO2] = models.Sensor{
		Name:       "CO2",
		SensorType: models.SensorTypeCO2,
		Location:   "Indoor",
		Enabled:    true,
	}
	sensors["NAMain-"+models.SensorTypeNoise] = models.Sensor{
		Name:       "Noise",
		SensorType: models.SensorTypeNoise,
		Location:   "Indoor",
		Enabled:    true,
	}

	// Outdoor Module (NAModule1)
	sensors["NAModule1-"+models.SensorTypeTemperatureOutdoor] = models.Sensor{
		Name:       "Temperature (Outdoor)",
		SensorType: models.SensorTypeTemperatureOutdoor,
		Location:   "Outdoor",
		Enabled:    true,
	}
	sensors["NAModule1-"+models.SensorTypeHumidityOutdoor] = models.Sensor{
		Name:       "Humidity (Outdoor)",
		SensorType: models.SensorTypeHumidityOutdoor,
		Location:   "Outdoor",
		Enabled:    true,
	}

	// Wind Gauge (NAModule2)
	sensors["NAModule2-"+models.SensorTypeWindDirection] = models.Sensor{
		Name:       "Wind Direction",
		SensorType: models.SensorTypeWindDirection,
		Location:   "Outdoor",
		Enabled:    true,
	}
	sensors["NAModule2-"+models.SensorTypeWindSpeed] = models.Sensor{
		Name:       "Wind Speed",
		SensorType: models.SensorTypeWindSpeed,
		Location:   "Outdoor",
		Enabled:    true,
	}
	sensors["NAModule2-"+models.SensorTypeWindGust] = models.Sensor{
		Name:       "Wind Gust",
		SensorType: models.SensorTypeWindGust,
		Location:   "Outdoor",
		Enabled:    true,
	}
	sensors["NAModule2-"+models.SensorTypeWindGustAngle] = models.Sensor{
		Name:       "Wind Gust Angle",
		SensorType: models.SensorTypeWindGustAngle,
		Location:   "Outdoor",
		Enabled:    true,
	}
	sensors["NAModule2-"+models.SensorTypeWindSpeedMaxDaily] = models.Sensor{
		Name:       "Max Wind Speed (Daily)",
		SensorType: models.SensorTypeWindSpeedMaxDaily,
		Location:   "Outdoor",
		Enabled:    true,
	}

	// Rain Gauge (NAModule3)
	sensors["NAModule3-"+models.SensorTypeRainfallRate] = models.Sensor{
		Name:       "Rain Rate",
		SensorType: models.SensorTypeRainfallRate,
		Location:   "Outdoor",
		Enabled:    true,
	}
	sensors["NAModule3-"+models.SensorTypeRainfallDaily] = models.Sensor{
		Name:       "Rain (24h)",
		SensorType: models.SensorTypeRainfallDaily,
		Location:   "Outdoor",
		Enabled:    true,
	}
	sensors["NAModule3-"+models.SensorTypeRainfallHourly] = models.Sensor{
		Name:       "Rain (1h)",
		SensorType: models.SensorTypeRainfallHourly,
		Location:   "Outdoor",
		Enabled:    true,
	}

	// Additional Indoor Module (NAModule4)
	sensors["NAModule4-"+models.SensorTypeTemperature] = models.Sensor{
		Name:       "Temperature (Additional Indoor)",
		SensorType: models.SensorTypeTemperature,
		Location:   "Indoor",
		Enabled:    true,
	}
	sensors["NAModule4-"+models.SensorTypeHumidity] = models.Sensor{
		Name:       "Humidity (Additional Indoor)",
		SensorType: models.SensorTypeHumidity,
		Location:   "Indoor",
		Enabled:    true,
	}
	sensors["NAModule4-"+models.SensorTypePressure] = models.Sensor{
		Name:       "Pressure (Additional Indoor)",
		SensorType: models.SensorTypePressure,
		Location:   "Indoor",
		Enabled:    true,
	}
	sensors["NAModule4-"+models.SensorTypeCO2] = models.Sensor{
		Name:       "CO2 (Additional Indoor)",
		SensorType: models.SensorTypeCO2,
		Location:   "Indoor",
		Enabled:    true,
	}
	sensors["NAModule4-"+models.SensorTypeCO2] = models.Sensor{
		Name:       "CO2 (Additional Indoor)",
		SensorType: models.SensorTypeCO2,
		Location:   "Indoor",
		Enabled:    true,
	}
	sensors["NAModule4-"+models.SensorTypeNoise] = models.Sensor{
		Name:       "Noise (Additional Indoor)",
		SensorType: models.SensorTypeNoise,
		Location:   "Indoor",
		Enabled:    true,
	}

	return sensors
}
